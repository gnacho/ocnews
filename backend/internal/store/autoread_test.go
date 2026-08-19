package store

import "testing"

// TestAutoReadMarksNewItemsRead: los items nuevos cuyo título casa con una
// regla de auto-marcado se guardan como leídos (unread=0); el resto no.
func TestAutoReadMarksNewItemsRead(t *testing.T) {
	st, uid := newTestStore(t)
	f, err := st.CreateFeed(uid, "https://r.example/f", nil, "t", "https://r.example", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddAutoRead(uid, f.ID, `(?i)urgente`); err != nil {
		t.Fatal(err)
	}
	incoming := []NewItem{
		{GUID: "a", GUIDHash: "ha", Title: "URGENTE: actualización de seguridad"},
		{GUID: "b", GUIDHash: "hb", Title: "noticia normal"},
	}
	if _, err := st.ReplaceFeedItems(f.ID, uid, "t", "https://r.example", incoming, false); err != nil {
		t.Fatal(err)
	}
	got, err := st.ListItems(ItemFilter{UserID: uid, Type: 0, ID: f.ID, GetRead: true, BatchSize: -1})
	if err != nil {
		t.Fatal(err)
	}
	byGUID := map[string]bool{}
	for _, x := range got {
		byGUID[x.GUID] = x.Unread
	}
	if byGUID["a"] {
		t.Error("item 'URGENTE' debería estar marcado como leído por auto-read")
	}
	if !byGUID["b"] {
		t.Error("item 'noticia normal' no debería estar marcado como leído")
	}
}

// TestAutoReadRuleScoped: una regla con feed_id concreto no aplica a otro feed.
func TestAutoReadRuleScoped(t *testing.T) {
	st, uid := newTestStore(t)
	f1, _ := st.CreateFeed(uid, "https://r1.example/f", nil, "t1", "https://r1.example", "", nil)
	f2, _ := st.CreateFeed(uid, "https://r2.example/f", nil, "t2", "https://r2.example", "", nil)
	if _, err := st.AddAutoRead(uid, f1.ID, `(?i)oferta`); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ReplaceFeedItems(f2.ID, uid, "t2", "https://r2.example",
		[]NewItem{{GUID: "x", GUIDHash: "hx", Title: "Oferta en r2"}}, false); err != nil {
		t.Fatal(err)
	}
	got, _ := st.ListItems(ItemFilter{UserID: uid, Type: 0, ID: f2.ID, GetRead: true, BatchSize: -1})
	if len(got) != 1 || !got[0].Unread {
		t.Error("la regla de f1 no debería marcar items de f2")
	}
}
