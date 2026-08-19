package store

import (
	"testing"
)

// TestFeedRulesFilter: una regla block del feed descarta items existentes al
// re-aplicarla, y se descongela al borrarla.
func TestFeedRulesFilter(t *testing.T) {
	st, uid := newTestStore(t)
	items := []NewItem{
		{GUID: "a", GUIDHash: "h1", Title: "Comprar oferta de verano"},
		{GUID: "b", GUIDHash: "h2", Title: "Noticia normal"},
		{GUID: "c", GUIDHash: "h3", Title: "Oferta flash"},
	}
	f, err := st.CreateFeed(uid, "https://r.example/f", nil, "t", "https://r.example", "", items)
	if err != nil {
		t.Fatal(err)
	}
	feedID := f.ID

	if err := st.SaveFeedRules(feedID, Rules{Block: `EntryTitle=(?i)oferta`}); err != nil {
		t.Fatal(err)
	}
	marked, err := st.ReapplyFeedFilter(feedID, FeedFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if marked != 2 {
		t.Fatalf("reapply: esperaba 2 marcados, tengo %d", marked)
	}
	got, err := st.ListItems(ItemFilter{UserID: uid, Type: 0, ID: feedID, GetRead: true, BatchSize: -1, IncludeFiltered: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range got {
		if x.GUID == "b" && x.Filtered {
			t.Error("item 'Noticia normal' no debería estar filtered")
		}
		if x.GUID != "b" && !x.Filtered {
			t.Errorf("item %q debería estar filtered", x.GUID)
		}
	}

	if err := st.DeleteFeedRules(feedID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ReapplyFeedFilter(feedID, FeedFilter{}); err != nil {
		t.Fatal(err)
	}
	got, _ = st.ListItems(ItemFilter{UserID: uid, Type: 0, ID: feedID, GetRead: true, BatchSize: -1, IncludeFiltered: true})
	for _, x := range got {
		if x.Filtered {
			t.Errorf("item %q debería estar descongelado", x.GUID)
		}
	}
}

// TestGlobalRulesFilter: una regla keep global descarta lo que no casa.
func TestGlobalRulesFilter(t *testing.T) {
	st, uid := newTestStore(t)
	items := []NewItem{
		{GUID: "a", GUIDHash: "h1", Title: "Python 3.13 release"},
		{GUID: "b", GUIDHash: "h2", Title: "Receta de paella"},
	}
	f, err := st.CreateFeed(uid, "https://r.example/f", nil, "t", "https://r.example", "", items)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SaveGlobalRules(uid, Rules{Keep: `EntryTitle=(?i)python`}); err != nil {
		t.Fatal(err)
	}
	marked, err := st.ReapplyGlobalRules(uid)
	if err != nil {
		t.Fatal(err)
	}
	if marked != 1 {
		t.Fatalf("global keep: esperaba 1 marcado, tengo %d", marked)
	}
	got, err := st.ListItems(ItemFilter{UserID: uid, Type: 0, ID: f.ID, GetRead: true, BatchSize: -1, IncludeFiltered: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range got {
		if x.GUID == "a" && x.Filtered {
			t.Error("python no debería estar filtered")
		}
		if x.GUID == "b" && !x.Filtered {
			t.Error("receta debería estar filtered (no casa con python)")
		}
	}
}

// TestApplyFilterToItemsWithRules: el ingesto marca items nuevos según reglas.
func TestApplyFilterToItemsWithRules(t *testing.T) {
	st, uid := newTestStore(t)
	f, err := st.CreateFeed(uid, "https://r.example/f", nil, "t", "https://r.example", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SaveFeedRules(f.ID, Rules{Block: `EntryTitle=spam`}); err != nil {
		t.Fatal(err)
	}
	incoming := []NewItem{
		{GUID: "x", GUIDHash: "hx", Title: "spam de entradas"},
		{GUID: "y", GUIDHash: "hy", Title: "noticia"},
	}
	if err := st.ApplyFilterToItems(f.ID, incoming); err != nil {
		t.Fatal(err)
	}
	if !incoming[0].Filtered {
		t.Error("item spam debería quedar filtered en el ingesto")
	}
	if incoming[1].Filtered {
		t.Error("item normal no debería quedar filtered")
	}
}
