package api

import (
	"fmt"
	"testing"
)

// TestFeedRulesAPI: guardar/leer/borrar reglas de un feed + validación 422.
func TestFeedRulesAPI(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})
	_, body := e.do(t, "POST", "/feeds", e.user, e.pass, map[string]string{"url": "https://a.example/f"})
	var feedResp struct {
		Feeds []struct{ ID int64 }
	}
	decode(t, body, &feedResp)
	if len(feedResp.Feeds) != 1 {
		t.Fatalf("no se creó el feed: %s", body)
	}
	feedID := feedResp.Feeds[0].ID

	// regla inválida → 422
	code, _ := e.do(t, "POST", fmt.Sprintf("/feeds/%d/rules", feedID), e.user, e.pass,
		map[string]string{"block": "EntryTitle=("})
	if code != 422 {
		t.Fatalf("regex inválida esperaba 422, tengo %d", code)
	}

	// guardar regla válida
	code, body = e.do(t, "POST", fmt.Sprintf("/feeds/%d/rules", feedID), e.user, e.pass,
		map[string]string{"block": "EntryTitle=(?i)oferta\nEntryDate=future", "keep": "EntryAuthor=(?i)luis"})
	if code != 200 {
		t.Fatalf("guardar reglas: %d %s", code, body)
	}
	var got struct {
		Rules struct {
			Block string `json:"block"`
			Keep  string `json:"keep"`
		} `json:"rules"`
	}
	decode(t, body, &got)
	if got.Rules.Block == "" || got.Rules.Keep == "" {
		t.Errorf("reglas guardadas vacías: %s", body)
	}

	// leer
	code, body = e.do(t, "GET", fmt.Sprintf("/feeds/%d/rules", feedID), e.user, e.pass, nil)
	if code != 200 {
		t.Fatalf("leer reglas: %d", code)
	}

	// borrar
	code, _ = e.do(t, "DELETE", fmt.Sprintf("/feeds/%d/rules", feedID), e.user, e.pass, nil)
	if code != 204 {
		t.Fatalf("borrar reglas: %d", code)
	}
	code, body = e.do(t, "GET", fmt.Sprintf("/feeds/%d/rules", feedID), e.user, e.pass, nil)
	decode(t, body, &got)
	if got.Rules.Block != "" || got.Rules.Keep != "" {
		t.Errorf("reglas no borradas: %s", body)
	}
}

// TestGlobalRulesAPI: PUT/GET /api/me/rules valida y persiste.
func TestGlobalRulesAPI(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})
	code, _ := e.do(t, "PUT", "/api/me/rules", e.user, e.pass,
		map[string]string{"block": "EntryTitle=("})
	if code != 422 {
		t.Fatalf("regla global inválida esperaba 422, tengo %d", code)
	}
	code, body := e.do(t, "PUT", "/api/me/rules", e.user, e.pass,
		map[string]string{"keep": "EntryContent=(?i)terraform"})
	if code != 200 {
		t.Fatalf("guardar reglas globales: %d %s", code, body)
	}
	var got struct {
		Rules struct{ Keep string }
	}
	decode(t, body, &got)
	if got.Rules.Keep == "" {
		t.Errorf("keep global no guardada: %s", body)
	}
	code, body = e.do(t, "GET", "/api/me/rules", e.user, e.pass, nil)
	if code != 200 {
		t.Fatalf("leer reglas globales: %d", code)
	}
	decode(t, body, &got)
	if got.Rules.Keep == "" {
		t.Errorf("keep global no leída: %s", body)
	}
}
