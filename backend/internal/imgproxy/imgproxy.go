// Package imgproxy: proxy de imágenes firmado. Los hosts OpenCloud sirven
// CSP img-src 'self' ... (sin dominios externos), así que el lector no puede
// cargar imágenes de los feeds directamente. Este proxy descarga la imagen y
// la sirve desde el propio dominio. El tag <img> del navegador NO puede
// llevar cabeceras de auth → la ruta es pública pero la URL va firmada con
// HMAC-SHA256 (secret persistente por instancia): solo se proxifican URLs
// que el propio servidor firmó al servir los items. Mitiga SSRF y abuso.
package imgproxy

import (
	"context"
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

	"github.com/gnacho/ocnews/backend/internal/netguard"
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
		client: netguard.Client(fetchTimeout),
		log:    log,
	}, nil
}

// NewAllowLocal es como New pero el transporte permite loopback. SOLO tests.
func NewAllowLocal(dataDir string, log *slog.Logger) (*Proxy, error) {
	p, err := New(dataDir, log)
	if err != nil {
		return nil, err
	}
	p.client = netguard.ClientAllowLocal(fetchTimeout)
	return p, nil
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

// Serve responde con la imagen (cache 24h) o hace streaming de audio/vídeo
// (con soporte Range para seek; sin caché en disco). Firma inválida → 403.
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

	// HEAD al origen: por content-type decidimos imagen (caché) o media (stream)
	ct, ok := p.probeContentType(r.Context(), u)
	if !ok {
		http.Error(w, "no disponible", http.StatusBadGateway)
		return
	}
	switch {
	case strings.HasPrefix(ct, "image/"):
		p.serveImage(w, r, u, ct)
	case strings.HasPrefix(ct, "audio/"), strings.HasPrefix(ct, "video/"):
		p.serveMedia(w, r, u, ct)
	default:
		http.Error(w, "tipo no permitido", http.StatusUnsupportedMediaType)
	}
}

// probeContentType descubre el content-type del origen. HEAD primero; si el
// origen no soporta HEAD (405/501), GET con Range 0-0 y cerrar enseguida.
func (p *Proxy) probeContentType(ctx context.Context, u string) (string, bool) {
	for _, probe := range []struct {
		method string
		rng    string
	}{
		{http.MethodHead, ""},
		{http.MethodGet, "bytes=0-0"},
	} {
		req, err := http.NewRequestWithContext(ctx, probe.method, u, nil)
		if err != nil {
			return "", false
		}
		req.Header.Set("User-Agent", userAgent)
		if probe.rng != "" {
			req.Header.Set("Range", probe.rng)
		}
		resp, err := p.client.Do(req)
		if err != nil {
			p.log.Debug("imgproxy probe falló", "url", u, "err", err)
			return "", false
		}
		ct := strings.ToLower(resp.Header.Get("Content-Type"))
		code := resp.StatusCode
		io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<10))
		resp.Body.Close()
		if code == http.StatusOK || code == http.StatusPartialContent {
			if ct != "" && ct != "application/octet-stream" {
				return ct, true
			}
			// octet-stream o vacío con 200: puede ser media sin mime correcto;
			// lo dejamos pasar y el serveMedia/serveImage vuelve a evaluar
			if code == http.StatusOK && ct == "" {
				return "application/octet-stream", true
			}
		}
	}
	return "", false
}

// serveImage: imagen con caché en disco (comportamiento original).
func (p *Proxy) serveImage(w http.ResponseWriter, r *http.Request, u, ct string) {
	base := filepath.Join(p.dir, cachePath(u))
	if cachedCT, err := os.ReadFile(base + ".ct"); err == nil {
		if img, err := os.ReadFile(base); err == nil {
			p.respond(w, string(cachedCT), img)
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
		http.Error(w, "no disponible", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		http.Error(w, "no disponible", http.StatusBadGateway)
		return
	}
	if ct2 := strings.ToLower(resp.Header.Get("Content-Type")); !strings.HasPrefix(ct2, "image/") {
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

// serveMedia: streaming de audio/vídeo con Range passthrough (el <video> del
// navegador necesita 206 para seek). Sin buffer completo ni caché en disco.
func (p *Proxy) serveMedia(w http.ResponseWriter, r *http.Request, u, ct string) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, u, nil)
	if err != nil {
		http.Error(w, "url inválida", http.StatusForbidden)
		return
	}
	req.Header.Set("User-Agent", userAgent)
	if rng := r.Header.Get("Range"); rng != "" {
		req.Header.Set("Range", rng)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Debug("imgproxy media falló", "url", u, "err", err)
		http.Error(w, "no disponible", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		http.Error(w, "no disponible", http.StatusBadGateway)
		return
	}

	h := w.Header()
	h.Set("Content-Type", ct)
	h.Set("Accept-Ranges", "bytes")
	if cr := resp.Header.Get("Content-Range"); cr != "" {
		h.Set("Content-Range", cr)
	}
	h.Set("Cache-Control", "public, max-age=86400")
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		h.Set("Content-Length", cl)
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (p *Proxy) respond(w http.ResponseWriter, ct string, img []byte) {
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	w.Write(img)
}
