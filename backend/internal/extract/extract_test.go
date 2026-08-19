package extract

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestArticleWithSelector(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`<html><body><div id="articleBody"><h1>Título</h1><p>Lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam quis nostrud exercitation ullamco laboris.</p><p>Segundo párrafo con más contenido de relleno para superar el umbral mínimo de caracteres de la extracción y asegurar que la validación pasa sin problemas.</p><p>Tercer párrafo adicional para dar holgura a la longitud total del texto plano que se extrae del selector.</p></div></body></html>`))
	}))
	defer ts.Close()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	_ = log
	e := NewAllowLocal(5 * time.Second)
	body, err := e.Article(context.Background(), ts.URL, "div#articleBody")
	if err != nil {
		t.Fatalf("Article con selector: %v", err)
	}
	if !strings.Contains(body, "Título") {
		t.Errorf("body sin el contenido esperado: %s", body)
	}
}

func TestArticleSelectorFallbackToReadability(t *testing.T) {
	// página sin el selector: debe caer a readability (que con poco texto
	// devuelve error o contenido; aquí solo verificamos que no panic)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`<html><body><main><p>Contenido corto.</p></main></body></html>`))
	}))
	defer ts.Close()

	e := NewAllowLocal(5 * time.Second)
	_, _ = e.Article(context.Background(), ts.URL, "div#no-existe")
}

func TestArticleSelectorInvalid(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`<html><body><p>x</p></body></html>`))
	}))
	defer ts.Close()
	e := NewAllowLocal(5 * time.Second)
	// selector sin match → fallback a readability → contenido demasiado corto
	_, err := e.Article(context.Background(), ts.URL, "div.que.no.existe")
	if err == nil {
		t.Error("esperaba error (extracción demasiado corta)")
	}
}
