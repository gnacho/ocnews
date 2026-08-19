package store

import "testing"

// TestClusterAggregation: items de feeds distintos con el mismo cluster_key
// se agrupan y el primary es el de mayor id (#42).
func TestClusterAggregation(t *testing.T) {
	st, uid := newTestStore(t)
	f1, err := st.CreateFeed(uid, "https://r1.example/f", nil, "t1", "https://r1.example", "",
		[]NewItem{{GUID: "a", GUIDHash: "ha", Title: "La misma noticia", Body: "<p>cuerpo</p>", ClusterKey: "clave1"}})
	if err != nil {
		t.Fatal(err)
	}
	f2, err := st.CreateFeed(uid, "https://r2.example/f", nil, "t2", "https://r2.example", "",
		[]NewItem{{GUID: "b", GUIDHash: "hb", Title: "la misma noticia", Body: "<p>cuerpo</p>", ClusterKey: "clave1"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateFeed(uid, "https://r3.example/f", nil, "t3", "https://r3.example", "",
		[]NewItem{{GUID: "c", GUIDHash: "hc", Title: "Otra noticia", Body: "<p>x</p>", ClusterKey: "clave2"}}); err != nil {
		t.Fatal(err)
	}

	got, err := st.Clusters(uid, []string{"clave1", "clave2"})
	if err != nil {
		t.Fatal(err)
	}
	c1, ok := got["clave1"]
	if !ok {
		t.Fatalf("cluster clave1 no encontrado: %+v", got)
	}
	if c1.Size != 2 {
		t.Errorf("clave1 esperaba size 2, tengo %d", c1.Size)
	}
	// primary = el item con mayor id (el de f2, insertado después)
	itemsF1, _ := st.ListItems(ItemFilter{UserID: uid, Type: 0, ID: f1.ID, GetRead: true, BatchSize: -1})
	itemsF2, _ := st.ListItems(ItemFilter{UserID: uid, Type: 0, ID: f2.ID, GetRead: true, BatchSize: -1})
	if c1.PrimaryID != itemsF2[0].ID {
		t.Errorf("primary debería ser el de f2 (%d), tengo %d", itemsF2[0].ID, c1.PrimaryID)
	}
	_ = itemsF1
	if _, ok := got["clave2"]; ok {
		t.Errorf("clave2 (1 item) no debería aparecer como cluster")
	}
}
