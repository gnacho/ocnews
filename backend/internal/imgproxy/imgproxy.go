// Package imgproxy: proxy de imágenes firmado. Los hosts OpenCloud sirven
// CSP img-src 'self' ... (sin dominios externos), así que el lector no puede
// cargar imágenes de los feeds directamente. Este proxy descarga la imagen y
// la sirve desde el propio dominio. El tag <img> del navegador NO puede
// llevar cabeceras de auth → la ruta es pública pero la URL va firmada con
// HMAC-SHA256 (secret persistente por instancia): solo se proxifican URLs
// que el propio servidor firmó al servir los items. Mitiga SSRF y abuso.
package imgproxy

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func newRand() io.Reader { return rand.Reader }

const (
	maxImageBytes = 8 << 20 // 8 MB
	fetchTimeout  = 15 * time.Second
	userAgent     = "ocnews/0.5 (+https://github.com/gnacho/ocnews)"
)

type Proxy struct {
	secret []byte
	dir    string // caché en disco
	client *http.Client
	log    *slog.Logger
}

// New carga (o genera) el secret en <dataDir>/imgsecret y prepara la caché.
func New(dataDir string, log *slog.Logger) (*Proxy, error) {
	secretPath := filepath.Join(dataDir, "imgsecret")
	secret, err := os.ReadFile(secretPath)
	if err != nil || len(secret) < 32 {
		secret = make([]byte, 32)
		if _, err := io.ReadFull(newRand(), secret); err != nil {
			return nil, fmt.Errorf("generar imgsecret: %w", err)
		}
		if err := os.WriteFile(secretPath, secret, 0o600); err != nil {
			return nil, fmt.Errorf("persistir imgsecret: %w", err)
		}
	}
	dir := filepath.Join(dataDir, "imgcache")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &Proxy{
		secret: secret,
		dir:    dir,
		client: &http.Client{Timeout: fetchTimeout},
		log:    log,
	}, nil
}

// Sign devuelve la firma HMAC (hex, 32 chars) de una URL de imagen.
func (p *Proxy) Sign(url string) string {
	mac := hmac.New(sha256.New, p.secret)
	mac.Write([]byte(url))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

func (p *Proxy) valid(url, sig string) bool {
	return hmac.Equal([]byte(p.Sign(url)), []byte(sig))
}

func cachePath(url string) string {
	sum := sha256.Sum256([]byte(url))
	return hex.EncodeToString(sum[:])
}

// Serve responde con la imagen (cache 24h). Firma inválida → 403.
func (p *Proxy) Serve(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	u, sig := q.Get("u"), q.Get("t")
	if u == "" || !p.valid(u, sig) {
		http.Error(w, "firma inválida", http.StatusForbidden)
		return
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		http.Error(w, "esquema no permitido", http.StatusForbidden)
		return
	}

	// caché en disco (nombre sin extensión; el content-type real se recuerda
	// en el fichero sidecar .ct)
	base := filepath.Join(p.dir, cachePath(u))
	if ct, err := os.ReadFile(base + ".ct"); err == nil {
		if img, err := os.ReadFile(base); err == nil {
			p.respond(w, string(ct), img)
			return
		}
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, u, nil)
	if err != nil {
		http.Error(w, "url inválida", http.StatusForbidden)
		return
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Debug("imgproxy fetch falló", "url", u, "err", err)
		http.Error(w, "no disponible", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		http.Error(w, "no disponible", http.StatusBadGateway)
		return
	}
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if !strings.HasPrefix(ct, "image/") {
		http.Error(w, "no es una imagen", http.StatusUnsupportedMediaType)
		return
	}
	img, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes))
	if err != nil || len(img) == 0 {
		http.Error(w, "no disponible", http.StatusBadGateway)
		return
	}
	_ = os.WriteFile(base, img, 0o600)
	_ = os.WriteFile(base+".ct", []byte(ct), 0o600)
	p.respond(w, ct, img)
}

func (p *Proxy) respond(w http.ResponseWriter, ct string, img []byte) {
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	w.Write(img)
}
