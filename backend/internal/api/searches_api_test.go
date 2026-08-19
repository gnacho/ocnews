package api

import (
	"fmt"
	"testing"
)

func TestSavedSearchesAPI(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})
	_, body := e.do(t, "POST", "/feeds", e.user, e.pass, map[string]string{"url": "https://s.example/f"})
	var feedResp struct {
		Feeds []struct{ ID int64 }
	}
	decode(t, body, &feedResp)
	if len(feedResp.Feeds) != 1 {
		t.Fatalf("feed no creado: %s", body)
	}

	// sin nombre o sin query → 422
	if code, _ := e.do(t, "POST", "/searches", e.user, e.pass, map[string]string{"name": "x"}); code != 422 {
		t.Fatalf("sin query esperaba 422, tengo %d", code)
	}

	code, body := e.do(t, "POST", "/searches", e.user, e.pass,
		map[string]string{"name": "Item 1", "query": "Item 1"})
	if code != 200 {
		t.Fatalf("crear búsqueda: %d %s", code, body)
	}
	var created struct {
		Search struct {
			ID    int64  `json:"id"`
			Name  string `json:"name"`
			Query string `json:"query"`
		} `json:"search"`
	}
	decode(t, body, &created)
	if created.Search.ID == 0 || created.Search.Name != "Item 1" {
		t.Fatalf("búsqueda creada rara: %s", body)
	}

	code, body = e.do(t, "GET", "/searches", e.user, e.pass, nil)
	if code != 200 {
		t.Fatalf("listar búsquedas: %d", code)
	}
	var list struct {
		Searches []struct{ ID int64 }
	}
	decode(t, body, &list)
	if len(list.Searches) != 1 {
		t.Fatalf("esperaba 1 búsqueda: %s", body)
	}

	// items de la búsqueda guardada
	code, body = e.do(t, "GET", fmt.Sprintf("/searches/%d/items", created.Search.ID), e.user, e.pass, nil)
	if code != 200 {
		t.Fatalf("items de búsqueda guardada: %d", code)
	}
	var items struct {
		Items []struct{ Title string }
	}
	decode(t, body, &items)
	if len(items.Items) == 0 {
		t.Errorf("sin items para la búsqueda: %s", body)
	}

	// borrar
	if code, _ := e.do(t, "DELETE", fmt.Sprintf("/searches/%d", created.Search.ID), e.user, e.pass, nil); code != 204 {
		t.Fatalf("borrar búsqueda: %d", code)
	}
	if code, _ := e.do(t, "GET", fmt.Sprintf("/searches/%d/items", created.Search.ID), e.user, e.pass, nil); code != 404 {
		t.Fatalf("items tras borrar esperaba 404, tengo %d", code)
	}
}
