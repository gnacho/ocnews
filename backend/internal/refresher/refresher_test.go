package refresher

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gnacho/ocnews/backend/internal/auth"
	"github.com/gnacho/ocnews/backend/internal/store"
)

type fakeFetcher struct {
	items  int
	fail   bool
	fetchN int
}

func (f *fakeFetcher) Fetch(_ context.Context, _ string) (*store.Feed, []store.NewItem, error) {
	f.fetchN++
	if f.fail {
		return nil, nil, errFake
	}
	fd := &store.Feed{Title: "T", Link: "https://s.example"}
	items := make([]store.NewItem, 0, f.items)
	for i := 0; i < f.items; i++ {
		guid := strings.Repeat(string(rune('a'+i)), 3) + string(rune(f.fetchN+'0'))
		items = append(items, store.NewItem{GUID: guid, GUIDHash: guid, Title: "n" + guid})
	}
	return fd, items, nil
}

type errFetch struct{}

func (errFetch) Error() string { return "fake fetch error" }

var errFake error = errFetch{}

func newEnv(t *testing.T) (*store.Store, int64, *fakeFetcher, *Refresher) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	hash, err := auth.HashPassword("x")
	if err != nil {
		t.Fatal(err)
	}
	uid, err := st.CreateUser("u", hash, "u", "user")
	if err != nil {
		t.Fatal(err)
	}
	ff := &fakeFetcher{items: 3}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	r := New(st, ff, log, time.Minute, 10*time.Minute)
	return st, uid, ff, r
}

func gapUntil(nextUpdate int64) time.Duration {
	return time.Until(time.Unix(nextUpdate, 0))
}

func TestRefreshNewItemsResetsStreak(t *testing.T) {
	st, uid, _, r := newEnv(t)
	f, err := st.CreateFeed(uid, "https://a.example/f", nil, "T", "https://a.example", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	res := r.Refresh(context.Background(), f)
	if res.Err != nil || res.Inserted == 0 {
		t.Fatalf("res: %+v", res)
	}
	got, err := st.GetFeed(uid, f.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.NoNewStreak != 0 {
		t.Errorf("racha debe resetear: %d", got.NoNewStreak)
	}
	// con novedades → gap base (1m ±20%): entre 48s y 72s
	if g := gapUntil(got.NextUpdateTime); g < 48*time.Second || g > 72*time.Second {
		t.Errorf("gap con novedades: %v", g)
	}
}

func TestRefreshNoNewItemsDoubles(t *testing.T) {
	st, uid, ff, r := newEnv(t)
	f, err := st.CreateFeed(uid, "https://b.example/f", nil, "T", "https://b.example", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Refresh(context.Background(), f) // trae items, agenda base
	ff.items = 0                       // siguiente fetch sin novedades

	got, _ := st.GetFeed(uid, f.ID)
	res := r.Refresh(context.Background(), got)
	if res.Inserted != 0 {
		t.Fatalf("esperaba 0 nuevos: %+v", res)
	}
	after, _ := st.GetFeed(uid, f.ID)
	if after.NoNewStreak != 1 {
		t.Errorf("racha debe ser 1: %d", after.NoNewStreak)
	}
	// sin novedades (racha 1) → 2m ±20%: entre 96s y 144s
	if g := gapUntil(after.NextUpdateTime); g < 96*time.Second || g > 144*time.Second {
		t.Errorf("gap sin novedades: %v", g)
	}
}

func TestRefreshErrorBackoff(t *testing.T) {
	st, uid, ff, r := newEnv(t)
	f, err := st.CreateFeed(uid, "https://c.example/f", nil, "T", "https://c.example", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	ff.fail = true
	res := r.Refresh(context.Background(), f)
	if res.Err == nil {
		t.Fatal("esperaba error")
	}
	got, _ := st.GetFeed(uid, f.ID)
	if got.UpdateErrorCount != 1 || got.LastUpdateError == nil {
		t.Fatalf("error no registrado: %+v", got)
	}
	// 1er error → 2m ±20% (interval 1m << 1)
	if g := gapUntil(got.NextUpdateTime); g < 96*time.Second || g > 144*time.Second {
		t.Errorf("gap backoff 1er error: %v", g)
	}
}

func TestRefreshSanitizesBody(t *testing.T) {
	st, uid, _, _ := newEnv(t)
	dirty := &dirtyFetcher{}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	r2 := New(st, dirty, log, time.Minute, time.Minute)
	f, err := st.CreateFeed(uid, "https://d.example/f", nil, "T", "https://d.example", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	r2.Refresh(context.Background(), f)
	items, err := st.ListItems(store.ItemFilter{UserID: uid, Type: 3, BatchSize: -1})
	if err != nil || len(items) == 0 {
		t.Fatalf("items: %v %d", err, len(items))
	}
	body := items[0].Body
	if strings.Contains(body, "<script") || strings.Contains(body, "onerror") {
		t.Errorf("body sin sanitizar: %s", body)
	}
	if !strings.Contains(body, "hola") {
		t.Errorf("contenido legible perdido: %s", body)
	}
}

type dirtyFetcher struct{}

func (d *dirtyFetcher) Fetch(_ context.Context, _ string) (*store.Feed, []store.NewItem, error) {
	fd := &store.Feed{Title: "T", Link: "https://s.example"}
	items := []store.NewItem{{
		GUID: "g1", GUIDHash: "g1", Title: "t",
		Body: `<p>hola</p><script>alert(1)</script><img src="https://x/i.png" onerror="evil()">`,
	}}
	return fd, items, nil
}
