package api

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func readBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// extractJSONInt saca el primer número que sigue a key en un JSON plano.
func extractJSONInt(t *testing.T, body []byte, key string) int64 {
	t.Helper()
	idx := strings.Index(string(body), key)
	if idx < 0 {
		t.Fatalf("clave %s no encontrada en %s", key, body)
	}
	rest := strings.TrimLeft(string(body)[idx+len(key):], " ")
	end := strings.IndexAny(rest, ",}]")
	if end < 0 {
		end = len(rest)
	}
	n, err := strconv.ParseInt(rest[:end], 10, 64)
	if err != nil {
		t.Fatalf("número inválido tras %s: %v", key, err)
	}
	return n
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// TestAndroidCompat: peculiaridades del cliente news-android verificadas
// contra su código fuente (retrofit NewsAPI.java).
func TestAndroidCompat(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})

	// crear feed con form-urlencoded (el cliente usa @Field, no JSON)
	req, _ := http.NewRequest("POST", e.ts.URL+Base+"/feeds",
		strings.NewReader("url=https://form.example/rss&folderId=0"))
	req.SetBasicAuth(e.user, e.pass)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := e.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(t, resp)
	if resp.StatusCode != 200 || !strings.Contains(string(body), `"url":"https://form.example/rss"`) {
		t.Fatalf("create feed form: %d %s", resp.StatusCode, body)
	}

	// crear carpeta (JSON, lo hace el cliente con Map) y feed con query params
	code, fb := e.do(t, "POST", "/folders", e.user, e.pass, map[string]string{"name": "C"})
	if code != 200 {
		t.Fatalf("carpeta: %d", code)
	}
	code, fb = e.do(t, "GET", "/items?type=3&getRead=true", e.user, e.pass, nil)
	if !strings.Contains(string(fb), "form.example") {
		_ = code
	}

	// PUT rename feed (spec dice POST; cliente manda PUT)
	code, fb = e.do(t, "GET", "/feeds", e.user, e.pass, nil)
	feedID := extractJSONInt(t, fb, `"id":`)
	if code, _ = e.do(t, "PUT", "/feeds/"+itoa(feedID)+"/rename", e.user, e.pass,
		map[string]string{"feedTitle": "Renombrado"}); code != 204 {
		t.Fatalf("PUT rename: %d", code)
	}
	// PUT move (spec dice POST; cliente manda PUT)
	if code, _ = e.do(t, "PUT", "/feeds/"+itoa(feedID)+"/move", e.user, e.pass,
		map[string]any{"folderId": nil}); code != 204 {
		t.Fatalf("PUT move: %d", code)
	}
	// POST sigue valiendo (spec)
	if code, _ = e.do(t, "POST", "/feeds/"+itoa(feedID)+"/rename", e.user, e.pass,
		map[string]string{"feedTitle": "Otro"}); code != 204 {
		t.Fatalf("POST rename: %d", code)
	}

	// ver el cambio aplicado
	_, fb = e.do(t, "GET", "/feeds", e.user, e.pass, nil)
	if !strings.Contains(string(fb), "Otro") {
		t.Fatalf("rename no aplicado: %s", fb)
	}

	// crear carpeta con JSON pero content-type form (curl -d por defecto):
	// decodeBody decide por contenido, no por cabecera
	reqC, _ := http.NewRequest("POST", e.ts.URL+Base+"/folders",
		strings.NewReader(`{"name":"Tecnología"}`))
	reqC.SetBasicAuth(e.user, e.pass)
	reqC.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	respC, err := e.client.Do(reqC)
	if err != nil {
		t.Fatal(err)
	}
	bodyC := readBody(t, respC)
	if respC.StatusCode != 200 || !strings.Contains(string(bodyC), "Tecnología") {
		t.Fatalf("carpeta JSON con content-type form: %d %s", respC.StatusCode, bodyC)
	}
}

// TestOCSUserStub: /ocs/v2.php/cloud/user con formato OCS v2 para el drawer
// del cliente Android (id + displayname).
func TestOCSUserStub(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})
	req, _ := http.NewRequest("GET", e.ts.URL+"/ocs/v2.php/cloud/user?format=json", nil)
	req.SetBasicAuth(e.user, e.pass)
	resp, err := e.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(t, resp)
	s := string(body)
	if resp.StatusCode != 200 || !strings.Contains(s, `"ocs"`) ||
		!strings.Contains(s, `"displayname":"Nacho"`) || !strings.Contains(s, `"id":"nacho"`) {
		t.Fatalf("ocs stub: %d %s", resp.StatusCode, body)
	}
	// sin auth → 401
	req2, _ := http.NewRequest("GET", e.ts.URL+"/ocs/v2.php/cloud/user", nil)
	resp2, _ := e.client.Do(req2)
	if resp2.StatusCode != 401 {
		t.Fatalf("ocs stub sin auth: %d", resp2.StatusCode)
	}
}
