package api

import (
	"io"
	"strings"
	"testing"

	"github.com/gnacho/ocnews/backend/internal/store"
)

// TestLinkHygieneSetting: el nivel persiste en user_settings vía
// /api/me/settings, valida y el default (tracking) se guarda vacío.
func TestLinkHygieneSetting(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})

	// inválido → 422
	if code, _ := e.do(t, "PUT", "/api/me/settings", e.user, e.pass,
		map[string]string{"linkHygiene": "maximo"}); code != 422 {
		t.Fatalf("nivel inválido esperaba 422, tengo %d", code)
	}

	var got struct {
		LinkHygiene string `json:"linkHygiene"`
	}

	// default sin tocar nada: tracking
	if code, body := e.do(t, "GET", "/api/me/settings", e.user, e.pass, nil); code != 200 {
		t.Fatalf("leer settings: %d", code)
	} else {
		decode(t, body, &got)
		if got.LinkHygiene != "tracking" {
			t.Fatalf("default debería ser tracking: %s", body)
		}
	}

	// agresivo persiste
	if code, body := e.do(t, "PUT", "/api/me/settings", e.user, e.pass,
		map[string]string{"linkHygiene": "aggressive"}); code != 200 {
		t.Fatalf("guardar agresivo: %d %s", code, body)
	}
	code, body := e.do(t, "GET", "/api/me/settings", e.user, e.pass, nil)
	if code != 200 {
		t.Fatalf("leer settings: %d", code)
	}
	decode(t, body, &got)
	if got.LinkHygiene != "aggressive" {
		t.Fatalf("nivel no persistido: %s", body)
	}

	// volver a tracking se guarda como vacío (default) y se sigue leyendo
	if code, _ := e.do(t, "PUT", "/api/me/settings", e.user, e.pass,
		map[string]string{"linkHygiene": "tracking"}); code != 200 {
		t.Fatalf("reset a tracking: %d", code)
	}
	code, body = e.do(t, "GET", "/api/me/settings", e.user, e.pass, nil)
	decode(t, body, &got)
	if got.LinkHygiene != "tracking" {
		t.Fatalf("tracking no se lee como default: %s", body)
	}
}

// TestItemsServeCleanedLinks: los href del cuerpo salen limpios según el
// nivel del usuario, y el nivel agresivo además desenvuelve el intermediario.
func TestItemsServeCleanedLinks(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})
	st := e.srv.store
	uid, _ := st.GetUserByUsername(e.user)
	body := `<p><a href="https://pub.example/a?utm_source=rss&amp;id=7">uno</a></p>` +
		`<p><a href="https://go.medio.example/out?url=https%3A%2F%2Fdestino.example%2Fnota%3Futm_campaign%3Dx">dos</a></p>`
	if _, err := st.CreateFeedFull(uid.ID, "https://feed.example/rss", nil, "Feed", "https://feed.example",
		"", []store.NewItem{{GUID: "g1", GUIDHash: "h1", URL: "https://feed.example/1", Title: "Item", PubDate: 1, Body: body}}, false); err != nil {
		t.Fatal(err)
	}

	fetch := func() string {
		t.Helper()
		code, raw := e.do(t, "GET", "/items?type=3&getRead=true", e.user, e.pass, nil)
		if code != 200 {
			t.Fatalf("items: %d %s", code, raw)
		}
		var resp struct {
			Items []struct {
				Body string `json:"body"`
			} `json:"items"`
		}
		decode(t, raw, &resp)
		if len(resp.Items) != 1 {
			t.Fatalf("esperaba 1 item, tengo %d", len(resp.Items))
		}
		return resp.Items[0].Body
	}

	// default (tracking): se va el utm, el intermediario se queda
	got := fetch()
	if strings.Contains(got, "utm_source") {
		t.Errorf("tracking no eliminado: %s", got)
	}
	if !strings.Contains(got, "go.medio.example") {
		t.Errorf("tracking no debe desenvolver el intermediario: %s", got)
	}
	if !strings.Contains(got, "id=7") {
		t.Errorf("query legítima perdida: %s", got)
	}

	// agresivo: además se desenvuelve, y el destino queda sin tracking
	if code, _ := e.do(t, "PUT", "/api/me/settings", e.user, e.pass,
		map[string]string{"linkHygiene": "aggressive"}); code != 200 {
		t.Fatalf("guardar agresivo: %d", code)
	}
	got = fetch()
	if strings.Contains(got, "go.medio.example") {
		t.Errorf("intermediario no desenvuelto: %s", got)
	}
	if !strings.Contains(got, "destino.example/nota") {
		t.Errorf("destino no aplicado: %s", got)
	}
	if strings.Contains(got, "utm_campaign") {
		t.Errorf("tracking del destino no eliminado: %s", got)
	}

	// off: HTML tal cual salvo el proxy de imágenes
	if code, _ := e.do(t, "PUT", "/api/me/settings", e.user, e.pass,
		map[string]string{"linkHygiene": "off"}); code != 200 {
		t.Fatalf("guardar off: %d", code)
	}
	got = fetch()
	if !strings.Contains(got, "utm_source") || !strings.Contains(got, "go.medio.example") {
		t.Errorf("off debería dejar los enlaces intactos: %s", got)
	}
}

// TestSharePageUsesOwnerLinkHygiene: la vista pública de un artículo
// compartido se limpia con el ajuste del DUEÑO del item, no con el visitante.
func TestSharePageUsesOwnerLinkHygiene(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})
	st := e.srv.store
	uid, _ := st.GetUserByUsername(e.user)
	body := `<p><a href="https://pub.example/a?utm_source=rss">uno</a></p>`
	f, err := st.CreateFeedFull(uid.ID, "https://feed.example/rss", nil, "Feed", "https://feed.example",
		"", []store.NewItem{{GUID: "g1", GUIDHash: "h1", URL: "https://feed.example/1", Title: "Item", PubDate: 1, Body: body}}, false)
	if err != nil {
		t.Fatal(err)
	}
	items, err := st.ListItems(store.ItemFilter{UserID: uid.ID, Type: 3, GetRead: true, BatchSize: -1})
	if err != nil || len(items) == 0 {
		t.Fatalf("sin items: %v", err)
	}
	token, err := st.CreateShare(uid.ID, items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if f.ID == 0 {
		t.Fatal("feed sin id")
	}

	resp, err := e.client.Get(e.ts.URL + Base + "/share/" + token)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	page := string(raw)
	if strings.Contains(page, "utm_source") {
		t.Errorf("la vista compartida no limpió el tracking del dueño: %s", page)
	}
	if !strings.Contains(page, "pub.example/a") {
		t.Errorf("enlace perdido en la vista compartida: %s", page)
	}
}
