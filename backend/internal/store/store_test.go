package store

import (
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func newTestStore(t *testing.T) (*Store, int64) {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	hash, err := bcrypt.GenerateFromPassword([]byte("x"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	uid, err := st.CreateUser("u", string(hash), "u", "user")
	if err != nil {
		t.Fatal(err)
	}
	return st, uid
}

// TestPurgeOldItems: borra leídos no destacados más viejos que el corte;
// conserva unread, starred y recientes.
func TestPurgeOldItems(t *testing.T) {
	st, uid := newTestStore(t)
	items := []NewItem{
		{GUID: "old-read", GUIDHash: "h1"},     // borrar
		{GUID: "old-unread", GUIDHash: "h2"},   // conservar (unread)
		{GUID: "old-star", GUIDHash: "h3"},     // conservar (starred)
		{GUID: "new-read", GUIDHash: "h4"},     // conservar (reciente)
	}
	if _, err := st.CreateFeed(uid, "https://r.example/f", nil, "t", "https://r.example", "", items); err != nil {
		t.Fatal(err)
	}
	got, err := st.ListItems(ItemFilter{UserID: uid, Type: 3, GetRead: true, BatchSize: -1})
	if err != nil || len(got) != 4 {
		t.Fatalf("setup: %v %d", err, len(got))
	}
	byGUID := map[string]int64{}
	for _, x := range got {
		byGUID[x.GUID] = x.ID
	}
	if err := st.MarkItemsUnreadFlag(uid, []int64{byGUID["old-read"], byGUID["new-read"]}, false); err != nil {
		t.Fatal(err)
	}
	if err := st.MarkItemsStarFlag(uid, []int64{byGUID["old-star"]}, true); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour).Unix()
	// envejecer last_modified de los tres "viejos"
	for _, guid := range []string{"old-read", "old-unread", "old-star"} {
		if _, err := st.db.Exec(`UPDATE items SET last_modified = ? WHERE id = ?`, old, byGUID[guid]); err != nil {
			t.Fatal(err)
		}
	}

	n, err := st.PurgeOldItems(time.Now().Add(-24 * time.Hour).Unix())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("purga borró %d, esperaba 1 (solo old-read)", n)
	}
	rest, _ := st.ListItems(ItemFilter{UserID: uid, Type: 3, GetRead: true, BatchSize: -1})
	if len(rest) != 3 {
		t.Fatalf("quedan %d, esperaba 3", len(rest))
	}
}

// TestListDueFeeds: solo los vencidos, ordenados por next_update.
func TestListDueFeeds(t *testing.T) {
	st, uid := newTestStore(t)
	now := time.Now().Unix()
	a, _ := st.CreateFeed(uid, "https://a.example/f", nil, "a", "", "", nil)
	b, _ := st.CreateFeed(uid, "https://b.example/f", nil, "b", "", "", nil)
	st.SetNextUpdate(a.ID, now-100)
	st.SetNextUpdate(b.ID, now+1000)

	due, err := st.ListDueFeeds(now, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 || due[0].ID != a.ID {
		t.Fatalf("due: %+v", due)
	}
}

// TestFeedByURLHash: lookup para el endpoint /favicon/{hash}.
func TestFeedByURLHash(t *testing.T) {
	st, uid := newTestStore(t)
	f, _ := st.CreateFeed(uid, "https://h.example/f", nil, "h", "", "", nil)
	if f.URLHash == "" {
		t.Fatal("url_hash vacío")
	}
	row, err := st.FeedByURLHash(f.URLHash)
	if err != nil {
		t.Fatal(err)
	}
	if row.ID != f.ID || row.UserID != uid {
		t.Fatalf("row: %+v", row)
	}
	if _, err := st.FeedByURLHash("inexistente"); err != ErrNotFound {
		t.Fatalf("esperaba ErrNotFound, tengo %v", err)
	}
}

// TestFeedFilter: guarda/lee/borra el filtro y filtra items nuevos y reaplica.
func TestFeedFilter(t *testing.T) {
	st, uid := newTestStore(t)
	f, _ := st.CreateFeed(uid, "https://fl.example/f", nil, "fl", "", "", nil)
	fid := f.ID

	// sin filtro: GetFeedFilter devuelve vacío
	ff, err := st.GetFeedFilter(fid)
	if err != nil || ff.HasFilter() {
		t.Fatalf("filtro inicial: %+v err=%v", ff, err)
	}

	// guardar filtro
	filter := FeedFilter{FeedID: fid, TitleKeywords: "sponsored,ADs", BodyKeywords: "tracking", URLKeywords: "utm_"}
	if err := st.SaveFeedFilter(filter); err != nil {
		t.Fatal(err)
	}
	got, _ := st.GetFeedFilter(fid)
	if got.TitleKeywords != "sponsored,ADs" || !got.HasFilter() {
		t.Fatalf("filtro guardado: %+v", got)
	}

	// items nuevos: el refresco marca filtered los que casan
	newItems := []NewItem{
		{GUID: "a", GUIDHash: "ha", Title: "normal", URL: "https://x.example/a", Body: "texto"},
		{GUID: "b", GUIDHash: "hb", Title: "SPONSORED post", URL: "https://x.example/b", Body: "otro"},
		{GUID: "c", GUIDHash: "hc", Title: "normal", URL: "https://x.example/c?utm_source=rss", Body: "texto"},
		{GUID: "d", GUIDHash: "hd", Title: "normal", URL: "https://x.example/d", Body: "con tracking"},
	}
	if _, err := st.ReplaceFeedItems(fid, uid, "fl", "", newItems, false); err != nil {
		t.Fatal(err)
	}
	listed, _ := st.ListItems(ItemFilter{UserID: uid, Type: 0, ID: fid, GetRead: true, BatchSize: -1})
	if len(listed) != 1 {
		t.Fatalf("sin IncludeFiltered deberían quedar 1 (solo 'a'), quedan %d", len(listed))
	}
	if listed[0].GUID != "a" {
		t.Fatalf("item no filtrado esperado: %s", listed[0].GUID)
	}
	all, _ := st.ListItems(ItemFilter{UserID: uid, Type: 0, ID: fid, GetRead: true, BatchSize: -1, IncludeFiltered: true})
	if len(all) != 4 {
		t.Fatalf("con IncludeFiltered deberían estar los 4, hay %d", len(all))
	}
	filteredCount := 0
	for _, it := range all {
		if it.Filtered {
			filteredCount++
		}
	}
	if filteredCount != 3 {
		t.Fatalf("filtered esperados 3 (b,c,d), hay %d", filteredCount)
	}

	// quitar keywords → descongela (ReapplyFeedFilter con filtro vacío)
	if err := st.SaveFeedFilter(FeedFilter{FeedID: fid}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ReapplyFeedFilter(fid, FeedFilter{FeedID: fid}); err != nil {
		t.Fatal(err)
	}
	after, _ := st.ListItems(ItemFilter{UserID: uid, Type: 0, ID: fid, GetRead: true, BatchSize: -1})
	if len(after) != 4 {
		t.Fatalf("tras descongelar deberían verse 4, hay %d", len(after))
	}
}

