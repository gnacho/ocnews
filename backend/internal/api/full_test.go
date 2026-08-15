package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// articleServer sirve una página con cuerpo largo y ruido de nav/footer.
func articleServer(t *testing.T) *httptest.Server {
	t.Helper()
	long := "<p>" + strings.Repeat("Texto sustancioso del artículo. ", 40) + "</p>"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<!doctype html><html><head><title>Artículo</title></head><body>
<nav>menu menu</nav>
<article><h1>Titulo real</h1>` + long + `</article>
<footer>pie</footer></body></html>`))
	}))
	t.Cleanup(ts.Close)
	return ts
}

func TestItemFullExtractsAndCaches(t *testing.T) {
	ts := articleServer(t)
	e := newTestEnv(t, &fakeFetcher{})

	// feed + items vía API; luego apuntar la URL del item más nuevo al artículo
	e.do(t, "POST", "/feeds", e.user, e.pass, map[string]string{"url": "https://feed.example/rss"})
	var itemsResp struct {
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
	}
	_, body := e.do(t, "GET", "/items?type=3&getRead=true", e.user, e.pass, nil)
	decode(t, body, &itemsResp)
	itemID := itemsResp.Items[0].ID

	// reescribir la URL del item al artículo local (extractor real la fetchea)
	if err := e.srv.store.SetItemURLForTesting(itemID, ts.URL+"/post"); err != nil {
		t.Fatal(err)
	}

	code, body := e.do(t, "GET", "/items/"+itoa(itemID)+"/full", e.user, e.pass, nil)
	if code != 200 {
		t.Fatalf("full: %d %s", code, body)
	}
	if !strings.Contains(string(body), "Texto sustancioso") {
		t.Fatalf("el cuerpo completo no llegó: %s", body[:min(200, len(body))])
	}
	if strings.Contains(string(body), "<nav>") || strings.Contains(string(body), "<script") {
		t.Fatal("el extraído debe ir sanitizado y sin chrome de la página")
	}

	// segunda llamada: servida desde caché (URL inválida ahora lo demostraría:
	// cambiamos la URL a algo inalcanzable y sigue funcionando por caché)
	e.srv.store.SetItemURLForTesting(itemID, "http://127.0.0.1:1/dead")
	code, body = e.do(t, "GET", "/items/"+itoa(itemID)+"/full", e.user, e.pass, nil)
	if code != 200 || !strings.Contains(string(body), "Texto sustancioso") {
		t.Fatalf("caché full: %d", code)
	}
}

func TestItemFullUnextractable(t *testing.T) {
	// página mínima: extracción < minTextLen → 422 con code estable
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`<html><body><p>corto</p></body></html>`))
	}))
	t.Cleanup(ts.Close)

	e := newTestEnv(t, &fakeFetcher{})
	e.do(t, "POST", "/feeds", e.user, e.pass, map[string]string{"url": "https://feed2.example/rss"})
	var itemsResp struct {
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
	}
	_, body := e.do(t, "GET", "/items?type=3&getRead=true", e.user, e.pass, nil)
	decode(t, body, &itemsResp)
	e.srv.store.SetItemURLForTesting(itemsResp.Items[0].ID, ts.URL+"/x")

	code, body := e.do(t, "GET", "/items/"+itoa(itemsResp.Items[0].ID)+"/full", e.user, e.pass, nil)
	if code != 422 || !strings.Contains(string(body), "full_unavailable") {
		t.Fatalf("unextractable: %d %s", code, body)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
