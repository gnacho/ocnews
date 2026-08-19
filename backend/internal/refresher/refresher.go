// Package refresher: lógica compartida de refresco de un feed entre el
// endpoint updater de la API y el scheduler periódico. Sanitiza los cuerpos
// al ingestar y calcula el próximo intervalo (adaptativo por novedad,
// backoff exponencial por error, con jitter).
package refresher

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/gnacho/ocnews/backend/internal/cred"
	"github.com/gnacho/ocnews/backend/internal/feed"
	"github.com/gnacho/ocnews/backend/internal/notify"
	"github.com/gnacho/ocnews/backend/internal/store"
)

type Refresher struct {
	store    *store.Store
	fetcher  feed.Fetcher
	cred     *cred.Cipher
	log      *slog.Logger
	interval time.Duration // intervalo base entre refrescos
	maxGap   time.Duration // tope del intervalo adaptativo
	notifier *notify.Notifier
}

func New(st *store.Store, f feed.Fetcher, c *cred.Cipher, log *slog.Logger, interval, maxGap time.Duration, n *notify.Notifier) *Refresher {
	return &Refresher{store: st, fetcher: f, cred: c, log: log, interval: interval, maxGap: maxGap, notifier: n}
}

// Result resume un refresco para el scheduler.
type Result struct {
	FeedID   int64
	Inserted int64
	Err      error
}

// Refresh re-descarga el feed, persiste items nuevos (sanitizados) y
// agenda el next_update según el resultado. Un fallo de fetch NO es error
// del método: queda registrado en el feed (update_error_count) y su backoff
// pasa a formar parte del next_update.
func (r *Refresher) Refresh(ctx context.Context, f *store.Feed) Result {
	res := r.refresh(ctx, f)
	r.scheduleNext(f, res)
	return res
}

func (r *Refresher) refresh(ctx context.Context, f *store.Feed) Result {
	authPass, err := r.cred.Decrypt(f.AuthPassEnc)
	if err != nil {
		r.store.RecordFeedError(f.ID, "credenciales ilegibles (¿clave rotada?)")
		r.log.Warn("descifrar credenciales falló", "url", f.URL, "err", err)
		return Result{FeedID: f.ID, Err: err}
	}
	parsed, items, err := r.fetcher.Fetch(ctx, f.URL, f.AuthUser, authPass)
	if err != nil {
		r.store.RecordFeedError(f.ID, err.Error())
		r.log.Warn("fetch falló", "url", f.URL, "err", err)
		return Result{FeedID: f.ID, Err: err}
	}
	title, link := f.Title, f.Link
	if parsed.Title != "" {
		title = parsed.Title
	}
	if parsed.Link != "" {
		link = parsed.Link
	}
	fullContent := feed.HasFullContent(items)
	feed.SanitizeItems(items)
	inserted, err := r.store.ReplaceFeedItems(f.ID, f.UserID, title, link, items, fullContent)
	if err != nil {
		return Result{FeedID: f.ID, Err: err}
	}
	// reflejar en el struct para quien llama (favicon usa f.Link)
	f.Title, f.Link = title, link
	if inserted > 0 {
		r.log.Info("feed actualizado", "url", f.URL, "nuevos", inserted)
		r.notifyNewItems(ctx, f, inserted, items)
	}
	return Result{FeedID: f.ID, Inserted: inserted}
}

// notifyNewItems: notificación push (ntfy) cuando el feed trae artículos
// nuevos. El topic es el del usuario (user_settings ntfy_topic) o, si no, el
// global configurado en el arranque. Sin topic → no se notifica.
func (r *Refresher) notifyNewItems(ctx context.Context, f *store.Feed, inserted int64, items []store.NewItem) {
	if r.notifier == nil || inserted <= 0 {
		return
	}
	topic, err := r.store.GetUserSetting(f.UserID, "ntfy_topic")
	if err != nil || topic == "" {
		topic = r.notifier.Topic()
	}
	if topic == "" {
		return
	}
	noun := "artículo"
	if inserted > 1 {
		noun = "artículos"
	}
	title := fmt.Sprintf("%d %s nuevos en %s", inserted, noun, f.Title)
	msg := ""
	if len(items) > 0 {
		msg = items[0].Title
	}
	if err := r.notifier.Notify(ctx, topic, title, msg); err != nil {
		r.log.Warn("notificación ntfy falló", "feed", f.ID, "err", err)
	}
}

// scheduleNext fija next_update:
//   - fallo: backoff exponencial según update_error_count (interval*2^err, tope 24h)
//   - sin novedades: intervalo que dobla con la racha RESULTANTE (interval*2^(streak+1), tope maxGap)
//   - con novedades: intervalo base
//
// Todo con jitter ±20% para no sincronizar feeds entre sí.
// El intervalo base puede ser un override por usuario (user_settings[feed_interval_min]).
func (r *Refresher) scheduleNext(f *store.Feed, res Result) {
	interval := r.feedInterval(f)
	var gap time.Duration
	switch {
	case res.Err != nil:
		errCount := f.UpdateErrorCount + 1 // RecordFeedError ya lo incrementó
		gap = interval << min(errCount, 5)
		if gap > 24*time.Hour {
			gap = 24 * time.Hour
		}
	case res.Inserted == 0:
		streak := f.NoNewStreak + 1 // racha tras este refresh sin novedades
		gap = interval << min(streak, 3)
		if gap > r.maxGap {
			gap = r.maxGap
		}
	default:
		gap = interval
	}
	gap = jitter(gap, 0.2)
	r.store.SetNextUpdate(f.ID, time.Now().Add(gap).Unix())
}

// feedInterval devuelve el intervalo base del feed: el override del usuario
// (minutos, 0 si no está) o el global.
func (r *Refresher) feedInterval(f *store.Feed) time.Duration {
	if v, err := r.store.GetUserSetting(f.UserID, "feed_interval_min"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Minute
		}
	}
	return r.interval
}

func jitter(d time.Duration, frac float64) time.Duration {
	if d <= 0 {
		return d
	}
	delta := float64(d) * frac
	return time.Duration(float64(d) + (rand.Float64()*2-1)*delta)
}
