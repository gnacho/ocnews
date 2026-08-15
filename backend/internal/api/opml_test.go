package api

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const opmlFixture = `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>subs</title></head>
  <body>
    <outline text="Tech">
      <outline type="rss" text="Site A" xmlUrl="https://a.example/rss"/>
      <outline type="rss" text="Site B" xmlUrl="https://b.example/feed.xml"/>
    </outline>
    <outline type="rss" text="Site C" xmlUrl="https://c.example/rss"/>
  </body>
</opml>`

func TestOPMLImportExportRoundtrip(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})

	// import
	req, _ := http.NewRequest("POST", e.ts.URL+Base+"/import/opml", strings.NewReader(opmlFixture))
	req.SetBasicAuth(e.user, e.pass)
	resp, err := e.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || !strings.Contains(string(body), `"imported":3`) {
		t.Fatalf("import: %d %s", resp.StatusCode, body)
	}

	// carpetas y feeds creados
	_, fb := e.do(t, "GET", "/folders", e.user, e.pass, nil)
	if !strings.Contains(string(fb), "Tech") {
		t.Fatalf("carpeta no creada: %s", fb)
	}
	_, fbf := e.do(t, "GET", "/feeds", e.user, e.pass, nil)
	for _, u := range []string{"https://a.example/rss", "https://b.example/feed.xml", "https://c.example/rss"} {
		if !strings.Contains(string(fbf), u) {
			t.Fatalf("feed %s no importado: %s", u, fbf)
		}
	}

	// import idempotente (re-import no duplica)
	req2, _ := http.NewRequest("POST", e.ts.URL+Base+"/import/opml", strings.NewReader(opmlFixture))
	req2.SetBasicAuth(e.user, e.pass)
	resp2, err := e.client.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	if !strings.Contains(string(body2), `"imported":0`) || !strings.Contains(string(body2), `"skipped":3`) {
		t.Fatalf("re-import: %s", body2)
	}

	// export roundtrip
	req3, _ := http.NewRequest("GET", e.ts.URL+Base+"/export/opml", nil)
	req3.SetBasicAuth(e.user, e.pass)
	resp3, err := e.client.Do(req3)
	if err != nil {
		t.Fatal(err)
	}
	expB, _ := io.ReadAll(resp3.Body)
	s := string(expB)
	for _, want := range []string{"<opml", `xmlUrl="https://a.example/rss"`, `xmlUrl="https://c.example/rss"`, "Tech"} {
		if !strings.Contains(s, want) {
			t.Errorf("export sin %q: %s", want, s)
		}
	}

	// opml inválido → 422
	req4, _ := http.NewRequest("POST", e.ts.URL+Base+"/import/opml", strings.NewReader("basura"))
	req4.SetBasicAuth(e.user, e.pass)
	resp4, _ := e.client.Do(req4)
	if resp4.StatusCode != 422 {
		t.Fatalf("opml inválido: %d", resp4.StatusCode)
	}
}

func TestFaviconEndpoint(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})

	// suscribir feed del sitio del fake
	_, fb := e.do(t, "POST", "/feeds", e.user, e.pass, map[string]string{"url": "https://feed.example/rss"})
	if !strings.Contains(string(fb), `"id":1`) {
		t.Fatalf("feed no creado: %s", fb)
	}
	hash := fmt.Sprintf("%x", md5.Sum([]byte("https://feed.example/rss")))

	// sin caché → 404
	code, _ := e.do(t, "GET", "/favicon/"+hash, e.user, e.pass, nil)
	if code != 404 {
		t.Fatalf("favicon sin caché: %d", code)
	}

	// poblar caché manualmente (el fetch real es best-effort del scheduler)
	png := "\x89PNG\r\n\x1a\nfakeicon"
	err := os.MkdirAll(filepath.Join(e.faviconDir), 0o700)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.faviconDir, hash+".png"), []byte(png), 0o600); err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest("GET", e.ts.URL+Base+"/favicon/"+hash, nil)
	req.SetBasicAuth(e.user, e.pass)
	resp, err := e.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("favicon: %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/png" {
		t.Errorf("content-type: %s", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != png {
		t.Errorf("contenido favicon mal")
	}

	// hash desconocido → 404 (aunque exista el fichero)
	os.WriteFile(filepath.Join(e.faviconDir, "ffff"+".png"), []byte(png), 0o600)
	code, _ = e.do(t, "GET", "/favicon/ffff", e.user, e.pass, nil)
	if code != 404 {
		t.Fatalf("hash sin feed: %d", code)
	}
}

func TestUpdaterFeedsAll(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})
	e.do(t, "POST", "/feeds", e.user, e.pass, map[string]string{"url": "https://up.example/f"})

	code, body := e.do(t, "GET", "/feeds/all", e.user, e.pass, nil)
	if code != 200 || !strings.Contains(string(body), `"userId":"nacho"`) {
		t.Fatalf("feeds/all: %d %s", code, body)
	}
	// no-admin → 401
	if code, _ = e.do(t, "GET", "/feeds/all", "otro", "otra1234", nil); code != 401 {
		t.Fatalf("feeds/all no-admin: %d", code)
	}
	// cleanup no-op de contrato
	if code, _ = e.do(t, "GET", "/cleanup/before-update", e.user, e.pass, nil); code != 204 {
		t.Fatalf("cleanup/before: %d", code)
	}
}
