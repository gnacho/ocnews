// Package refresher: lógica compartida de refresco de un feed entre el
// endpoint updater de la API y el scheduler periódico. Sanitiza los cuerpos
// al ingestar y calcula el próximo intervalo (adaptativo por novedad,
// backoff exponencial por error, con jitter).
package refresher

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/gnacho/ocnews/backend/internal/feed"
	"github.com/gnacho/ocnews/backend/internal/store"
)

type Refresher struct {
	store    *store.Store
	fetcher  feed.Fetcher
	log      *slog.Logger
	interval time.Duration // intervalo base entre refrescos
	maxGap   time.Duration // tope del intervalo adaptativo
}

func New(st *store.Store, f feed.Fetcher, log *slog.Logger, interval, maxGap time.Duration) *Refresher {
	return &Refresher{store: st, fetcher: f, log: log, interval: interval, maxGap: maxGap}
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
	parsed, items, err := r.fetcher.Fetch(ctx, f.URL)
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
	}
	return Result{FeedID: f.ID, Inserted: inserted}
}

// scheduleNext fija next_update:
//   - fallo: backoff exponencial según update_error_count (interval*2^err, tope 24h)
//   - sin novedades: intervalo que dobla con la racha RESULTANTE (interval*2^(streak+1), tope maxGap)
//   - con novedades: intervalo base
//
// Todo con jitter ±20% para no sincronizar feeds entre sí.
func (r *Refresher) scheduleNext(f *store.Feed, res Result) {
	var gap time.Duration
	switch {
	case res.Err != nil:
		errCount := f.UpdateErrorCount + 1 // RecordFeedError ya lo incrementó
		gap = r.interval << min(errCount, 5)
		if gap > 24*time.Hour {
			gap = 24 * time.Hour
		}
	case res.Inserted == 0:
		streak := f.NoNewStreak + 1 // racha tras este refresh sin novedades
		gap = r.interval << min(streak, 3)
		if gap > r.maxGap {
			gap = r.maxGap
		}
	default:
		gap = r.interval
	}
	gap = jitter(gap, 0.2)
	r.store.SetNextUpdate(f.ID, time.Now().Add(gap).Unix())
}

func jitter(d time.Duration, frac float64) time.Duration {
	if d <= 0 {
		return d
	}
	delta := float64(d) * frac
	return time.Duration(float64(d) + (rand.Float64()*2-1)*delta)
}
