// Package favicon: obtiene, cachea y sirve el favicon de cada feed.
// Caché en DATA_DIR/favicons/{url_hash}.ext; el endpoint de la API v1.3
// GET /favicon/{feedUrlHash} lo consulta (hash = md5 de la URL del feed).
package favicon

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gnacho/ocnews/backend/internal/netguard"
)

const (
	maxFaviconBytes = 512 << 10 // 512 KB
	fetchTimeout    = 10 * time.Second
	userAgent       = "ocnews/0.5 (+https://github.com/gnacho/ocnews)"
)

type Cache struct {
	dir    string
	client *http.Client
	log    *slog.Logger
}

func NewCache(dir string, log *slog.Logger) (*Cache, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("crear caché favicons: %w", err)
	}
	return &Cache{dir: dir, client: netguard.Client(fetchTimeout), log: log}, nil
}

// Hash devuelve el md5 hex de una URL de feed (identificador del endpoint).
func Hash(feedURL string) string {
	sum := md5.Sum([]byte(feedURL))
	return fmt.Sprintf("%x", sum)
}

// Fetch descarga el favicon del sitio del feed (best-effort: los errores
// solo se loguean). Prueba /favicon.ico del origen del enlace del sitio.
// La caché se indexa por el hash de la URL DEL FEED (la que usa el endpoint).
func (c *Cache) Fetch(ctx context.Context, feedURL, siteLink string) {
	icoURL, ok := faviconURL(siteLink)
	if !ok {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, icoURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Debug("favicon no descargado", "url", icoURL, "err", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFaviconBytes))
	if err != nil || len(body) == 0 {
		return
	}
	ctype := strings.ToLower(resp.Header.Get("Content-Type"))
	if !strings.HasPrefix(ctype, "image/") {
		return // HTML de error u otro contenido: no vale como icono
	}
	_ = os.WriteFile(c.pathFor(Hash(feedURL)+ext(ctype)), body, 0o600)
}

// Has dice si ya hay un favicon cacheado para el hash dado.
func (c *Cache) Has(hash string) bool {
	for _, ext := range []string{".ico", ".png", ".svg", ".jpg", ".jpeg", ".gif", ".webp"} {
		if _, err := os.Stat(c.pathFor(hash + ext)); err == nil {
			return true
		}
	}
	return false
}

// Serve responde con el favicon cacheado para el hash dado, o 404.
func (c *Cache) Serve(w http.ResponseWriter, hash string) {
	for _, ext := range []string{".ico", ".png", ".svg", ".jpg", ".jpeg", ".gif", ".webp"} {
		p := c.pathFor(hash + ext)
		if b, err := os.ReadFile(p); err == nil {
			w.Header().Set("Content-Type", contentType(ext))
			w.Header().Set("Cache-Control", "public, max-age=86400")
			w.WriteHeader(http.StatusOK)
			w.Write(b)
			return
		}
	}
	http.NotFound(w, nil)
}

func (c *Cache) pathFor(name string) string {
	return filepath.Join(c.dir, name)
}

// faviconURL deriva https://host/favicon.ico del enlace del sitio.
func faviconURL(siteLink string) (string, bool) {
	if siteLink == "" {
		return "", false
	}
	u, err := url.Parse(siteLink)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", false
	}
	return u.Scheme + "://" + u.Host + "/favicon.ico", true
}

func ext(ctype string) string {
	switch {
	case strings.Contains(ctype, "svg"):
		return ".svg"
	case strings.Contains(ctype, "png"):
		return ".png"
	case strings.Contains(ctype, "jpeg") || strings.Contains(ctype, "jpg"):
		return ".jpg"
	case strings.Contains(ctype, "gif"):
		return ".gif"
	case strings.Contains(ctype, "webp"):
		return ".webp"
	default:
		return ".ico"
	}
}

func contentType(extn string) string {
	switch extn {
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/x-icon"
	}
}
