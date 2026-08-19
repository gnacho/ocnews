package scheduler

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gnacho/ocnews/backend/internal/auth"
	"github.com/gnacho/ocnews/backend/internal/cred"
	"github.com/gnacho/ocnews/backend/internal/feed"
	"github.com/gnacho/ocnews/backend/internal/favicon"
	"github.com/gnacho/ocnews/backend/internal/refresher"
	"github.com/gnacho/ocnews/backend/internal/store"
)

const rssBody = `<?xml version="1.0"?><rss version="2.0"><channel>
<title>Feed Test</title><link>https://x.example</link><description>d</description>ITEMSPLACEHOLDER
</channel></rss>`

func itemXML(guid string) string {
	return `<item><title>` + guid + `</title><link>https://x.example/` + guid +
		`</link><guid isPermaLink="false">` + guid + `</guid><description>cuerpo ` + guid + `</description></item>`
}

// feedServer sirve un RSS cuyos items crecen al pulsar el botón add.
func feedServer(t *testing.T) (*httptest.Server, *int32, *int32) {
	t.Helper()
	var hits, extra int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		items := itemXML("base-1") + itemXML("base-2")
		for i := int32(0); i < atomic.LoadInt32(&extra); i++ {
			items += itemXML("extra-" + string(rune('a'+i)))
		}
		w.Write([]byte(strings.Replace(rssBody, "ITEMSPLACEHOLDER", items, 1)))
	}))
	t.Cleanup(ts.Close)
	return ts, &hits, &extra
}

// TestSchedulerPicksUpNewItems: feed suscrito con 2 items; el scheduler
// (tick 100ms) descubre un 3º añadido al feed sin reiniciar el backend.
func TestSchedulerPicksUpNewItems(t *testing.T) {
	ts, hits, extra := feedServer(t)
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	hash, _ := auth.HashPassword("x")
	uid, _ := st.CreateUser("u", hash, "u", "user")

	// suscripción con fetch inicial vía API de store (2 items)
	f, items, err := feed.NewHTTPFetcherAllowLocal(5*time.Second).Fetch(context.Background(), ts.URL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	created, err := st.CreateFeed(uid, ts.URL, nil, f.Title, f.Link, "", items)
	if err != nil {
		t.Fatal(err)
	}
	// CreateFeed agenda a +600s (ya fetcheado en la suscripción): forzar
	// vencimiento para que el ciclo del scheduler lo recoja YA.
	st.SetNextUpdate(created.ID, time.Now().Unix()-1)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	creds, err := cred.Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	refresh := refresher.New(st, feed.NewHTTPFetcherAllowLocal(5*time.Second), creds, log, 200*time.Millisecond, time.Hour, nil)
	fc, _ := favicon.NewCache(filepath.Join(dir, "fav"), log)
	s := New(st, refresh, fc, log, 100*time.Millisecond, 2, 0, nil, "")

	ctx, cancel := context.WithCancel(context.Background())
	var done = make(chan struct{})
	go func() { s.Run(ctx); close(done) }()
	defer func() {
		cancel()
		<-done
	}()

	// el scheduler refresca en el primer ciclo; esperamos ≥2 hits
	waitFor(t, 5*time.Second, func() bool { return atomic.LoadInt32(hits) >= 2 })

	// aparece un item nuevo en el feed origen
	atomic.AddInt32(extra, 1)

	waitFor(t, 8*time.Second, func() bool {
		newest, _ := st.NewestItemID(uid)
		return newest >= 3
	})
	n, _ := st.CountItems(store.ItemFilter{UserID: uid, Type: 3, BatchSize: -1})
	if n != 3 {
		t.Fatalf("items tras descubrimiento: %d (esperaba 3)", n)
	}
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("condición no alcanzada en " + timeout.String())
}
