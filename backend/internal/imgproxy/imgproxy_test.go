package imgproxy

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func newProxy(t *testing.T) *Proxy {
	t.Helper()
	p, err := NewAllowLocal(t.TempDir(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSignStable(t *testing.T) {
	p := newProxy(t)
	a, b := p.Sign("https://x.example/i.png"), p.Sign("https://x.example/i.png")
	if a != b || len(a) != 32 {
		t.Fatalf("firma inestable: %q vs %q", a, b)
	}
	if a == p.Sign("https://x.example/otra.png") {
		t.Fatal("urls distintas firman igual")
	}
	// secret distinto → firma distinta
	p2 := newProxy(t)
	if p.Sign("https://x.example/i.png") == p2.Sign("https://x.example/i.png") {
		t.Fatal("secrets distintos deben firmar distinto")
	}
}

func TestServeValidatesSignature(t *testing.T) {
	p := newProxy(t)
	w := httptest.NewRecorder()
	p.Serve(w, httptest.NewRequest("GET", "/img?u=https://x/i.png&t=deadbeef", nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("firma mala: %d", w.Code)
	}
	// esquema no permitido aunque la firma sea válida
	sig := p.Sign("file:///etc/passwd")
	w = httptest.NewRecorder()
	p.Serve(w, httptest.NewRequest("GET", "/img?u="+url.QueryEscape("file:///etc/passwd")+"&t="+sig, nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("esquema file: %d", w.Code)
	}
	// sin firma
	w = httptest.NewRecorder()
	p.Serve(w, httptest.NewRequest("GET", "/img?u=https://x/i.png", nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("sin firma: %d", w.Code)
	}
}

func TestServeProxiesAndCaches(t *testing.T) {
	p := newProxy(t)
	png := "\x89PNG\r\n\x1a\n" + "fakeimagebytes"
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			hits++ // el probe HEAD no cuenta
		}
		w.Header().Set("Content-Type", "image/png")
		io.WriteString(w, png)
	}))
	defer ts.Close()

	u := ts.URL + "/img.png"
	sig := p.Sign(u)

	// primera petición: proxy + caché disco
	w := httptest.NewRecorder()
	p.Serve(w, httptest.NewRequest("GET", "/img?u="+url.QueryEscape(u)+"&t="+sig, nil))
	if w.Code != 200 || w.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("primera: %d %s", w.Code, w.Header().Get("Content-Type"))
	}
	if w.Body.String() != png {
		t.Fatal("contenido mal")
	}
	if hits != 1 {
		t.Fatalf("hits: %d", hits)
	}

	// segunda: desde caché, sin nuevo hit al origen
	w = httptest.NewRecorder()
	p.Serve(w, httptest.NewRequest("GET", "/img?u="+url.QueryEscape(u)+"&t="+sig, nil))
	if w.Code != 200 || hits != 1 {
		t.Fatalf("cache: %d hits=%d", w.Code, hits)
	}

	// el secret persistió (dataDir/imgsecret existe)
	p2, err := New(filepath.Dir(mustFindSecret(t, p)), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if p2.Sign(u) != sig {
		t.Fatal("el secret debe persistir entre reinicios")
	}
}

func mustFindSecret(t *testing.T, p *Proxy) string {
	t.Helper()
	// localizar imgsecret creado en el TempDir del proxy: escaneamos su dir de caché
	glob := filepath.Join(p.dir, "*")
	_ = glob
	// el secret está junto a imgcache: subir un nivel desde imgcache
	return filepath.Join(filepath.Dir(p.dir), "imgsecret")
}

func TestServeRejectsNonImage(t *testing.T) {
	p := newProxy(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, "<html>")
	}))
	defer ts.Close()
	u := ts.URL + "/x"
	w := httptest.NewRecorder()
	p.Serve(w, httptest.NewRequest("GET", "/img?u="+url.QueryEscape(u)+"&t="+p.Sign(u), nil))
	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("html: %d", w.Code)
	}
	// y no deja basura en caché
	if _, err := os.Stat(filepath.Join(p.dir, cachePath(u))); !os.IsNotExist(err) {
		t.Fatal("no debe cachear no-imágenes")
	}
}
