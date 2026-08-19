package store

import "testing"

// TestNestedFolders: crear subcarpetas, listarlas con parentId y que el scope
// de carpeta padre incluya los feeds de sus subcarpetas (#41).
func TestNestedFolders(t *testing.T) {
	st, uid := newTestStore(t)
	parent, err := st.CreateFolder(uid, "Padre", nil)
	if err != nil {
		t.Fatal(err)
	}
	child, err := st.CreateFolder(uid, "Hijo", &parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateFeed(uid, "https://r.example/f", &child.ID, "t", "https://r.example", "",
		[]NewItem{{GUID: "a", GUIDHash: "ha", Title: "x"}}); err != nil {
		t.Fatal(err)
	}

	// listar carpetas: parentId presente
	folders, err := st.ListFolders(uid)
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 2 {
		t.Fatalf("esperaba 2 carpetas: %+v", folders)
	}
	for _, f := range folders {
		if f.ID == child.ID && (f.ParentID == nil || *f.ParentID != parent.ID) {
			t.Errorf("hijo sin parentId correcto: %+v", f)
		}
		if f.ID == parent.ID && f.ParentID != nil {
			t.Errorf("padre con parentId: %+v", f)
		}
	}

	// el scope del folder padre incluye los feeds de la subcarpeta
	got, err := st.ListItems(ItemFilter{UserID: uid, Type: 1, ID: parent.ID, GetRead: true, BatchSize: -1})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("el folder padre debería incluir 1 item de la subcarpeta, tengo %d", len(got))
	}
}

// TestDeleteParentFolderPromotesChildren: borrar un folder padre sube sus
// subcarpetas a la raíz.
func TestDeleteParentFolderPromotesChildren(t *testing.T) {
	st, uid := newTestStore(t)
	parent, err := st.CreateFolder(uid, "Padre", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateFolder(uid, "Hijo", &parent.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteFolder(uid, parent.ID); err != nil {
		t.Fatal(err)
	}
	folders, _ := st.ListFolders(uid)
	if len(folders) != 1 {
		t.Fatalf("tras borrar el padre debería quedar 1 subcarpeta, tengo %d", len(folders))
	}
	if folders[0].ParentID != nil {
		t.Errorf("la subcarpeta debería estar en la raíz: %+v", folders[0])
	}
}
