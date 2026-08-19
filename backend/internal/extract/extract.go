// Package extract: obtiene el cuerpo completo de un artículo desde su URL
// original (los feeds suelen traer solo el resumen). Usa readability y
// devuelve HTML ya sanitizado por la MISMA política que los cuerpos de feed.
package extract

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-shiori/go-readability"

	"github.com/gnacho/ocnews/backend/internal/netguard"
	"github.com/gnacho/ocnews/backend/internal/sanitize"
)

const (
	maxArticleBytes  = 5 << 20 // 5 MB de HTML
	minTextLen       = 250     // menos que esto = extracción fallida
	userAgent        = "Mozilla/5.0 (compatible; ocnews/0.5; +https://github.com/gnacho/ocnews)"
)

type Extractor struct {
	Client *http.Client
}

func New(timeout time.Duration) *Extractor {
	return &Extractor{Client: netguard.Client(timeout)}
}

// NewAllowLocal es como New pero el transporte permite loopback. SOLO tests.
func NewAllowLocal(timeout time.Duration) *Extractor {
	return &Extractor{Client: netguard.ClientAllowLocal(timeout)}
}

// Article descarga la página y extrae el contenido principal.
// Si selector no está vacío, intenta extraer ese elemento CSS primero
// (regla por feed) y cae a readability si el selector falla.
// Devuelve ("", error) si no se puede; el HTML va sanitizado.
func (e *Extractor) Article(ctx context.Context, rawURL, selector string) (string, error) {
	if selector != "" {
		if body, err := e.articleBySelector(ctx, rawURL, selector); err == nil {
			return body, nil
		}
	}
	return e.articleByReadability(ctx, rawURL)
}

// articleBySelector extrae el primer elemento que casa con el selector CSS.
func (e *Extractor) articleBySelector(ctx context.Context, rawURL, selector string) (string, error) {
	resp, err := e.fetch(ctx, rawURL)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	doc, err := goquery.NewDocumentFromReader(io.LimitReader(resp.Body, maxArticleBytes))
	if err != nil {
		return "", fmt.Errorf("parsear con selector: %w", err)
	}
	sel := doc.Find(selector).First()
	if sel.Length() == 0 {
		return "", fmt.Errorf("selector %q sin coincidencias", selector)
	}
	text := strings.TrimSpace(sel.Text())
	if len(text) < minTextLen {
		return "", fmt.Errorf("selector %q demasiado corto (%d chars)", selector, len(text))
	}
	html, err := sel.Html()
	if err != nil {
		return "", err
	}
	return sanitize.Body(html), nil
}

// articleByReadability usa go-readability (fallback por defecto).
func (e *Extractor) articleByReadability(ctx context.Context, rawURL string) (string, error) {
	resp, err := e.fetch(ctx, rawURL)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	article, err := readability.FromReader(io.LimitReader(resp.Body, maxArticleBytes), parsed)
	if err != nil {
		return "", fmt.Errorf("extraer: %w", err)
	}
	text := strings.TrimSpace(article.TextContent)
	if len(text) < minTextLen {
		return "", fmt.Errorf("extracción demasiado corta (%d chars)", len(text))
	}
	return sanitize.Body(article.Content), nil
}

// fetch descarga una URL de artículo con el User-Agent de ocnews.
func (e *Extractor) fetch(ctx context.Context, rawURL string) (*http.Response, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("url inválida")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*")
	resp, err := e.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("descargar artículo: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("artículo: HTTP %d", resp.StatusCode)
	}
	return resp, nil
}
