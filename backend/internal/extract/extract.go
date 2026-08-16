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
// Devuelve ("", error) si no se puede; el HTML va sanitizado.
func (e *Extractor) Article(ctx context.Context, rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("url inválida")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*")
	resp, err := e.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("descargar artículo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("artículo: HTTP %d", resp.StatusCode)
	}

	// charset: readability maneja la detección internamente con FromReader
	// si pasamos el content-type correcto
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
