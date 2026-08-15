// Package api: servidor HTTP de ocnews con la News REST API v1.3
// (contrato: docs/development/api/api-v1-3.md del repo nextcloud/news)
// bajo /index.php/apps/news/api/v1-3/ + /healthz sin auth.
package api

import (
	"bufio"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/gnacho/ocnews/backend/internal/auth"
	"github.com/gnacho/ocnews/backend/internal/extract"
	"github.com/gnacho/ocnews/backend/internal/feed"
	"github.com/gnacho/ocnews/backend/internal/favicon"
	"github.com/gnacho/ocnews/backend/internal/i18n"
	"github.com/gnacho/ocnews/backend/internal/imgproxy"
	"github.com/gnacho/ocnews/backend/internal/refresher"
	"github.com/gnacho/ocnews/backend/internal/store"
)

// Base path exacta que usan los clientes News (Nextcloud incluido).
const Base = "/index.php/apps/news/api/v1-3"

// reportedVersion es la versión de la app News que reportamos; los clientes
// la usan para decidir features. Filtro de feeds requiere >= 28.4.0.
const reportedVersion = "28.4.0"

type Server struct {
	store     *store.Store
	validator auth.Validator
	fetcher   feed.Fetcher
	refresher *refresher.Refresher
	favicons  *favicon.Cache
	imgs      *imgproxy.Proxy
	extract   *extract.Extractor
	retention time.Duration
	log       *slog.Logger
}

func NewServer(s *store.Store, v auth.Validator, f feed.Fetcher, r *refresher.Refresher,
	fc *favicon.Cache, ip *imgproxy.Proxy, ex *extract.Extractor,
	retention time.Duration, log *slog.Logger) *Server {
	return &Server{store: s, validator: v, fetcher: f, refresher: r, favicons: fc, imgs: ip,
		extract: ex, retention: retention, log: log}
}

// Handler monta el router completo con CORS + auth en la base de la API.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	api := http.NewServeMux()
	s.routes(api)
	authed := auth.Middleware(s.validator, withCORS(api))

// Proxy de imágenes PÚBLICO (el <img> del navegador no puede mandar auth):
// la seguridad la aporta la firma HMAC de la URL. Con método explícito para
// no chocar con el patrón OPTIONS de preflight.
mux.Handle("GET "+Base+"/img", http.StripPrefix(Base, withCORS(http.HandlerFunc(s.imgs.Serve))))

	mux.Handle(Base+"/", http.StripPrefix(Base, authed))
	// API propia de la app (perfil/usuarios) bajo /api/, misma Basic auth
	mux.Handle("/api/", auth.Middleware(s.validator, withCORS(s.userAPI())))
	// Stub OCS user para clientes Nextcloud (p.ej. news-android lee el
	// display name de /ocs/v2.php/cloud/user; OpenCloud no lo sirve).
	mux.Handle("/ocs/v2.php/cloud/user", auth.Middleware(s.validator, withCORS(http.HandlerFunc(s.ocsUser))))
	// Preflight de CORS antes de auth (no lleva credenciales).
	mux.Handle("OPTIONS "+Base+"/", http.StripPrefix(Base, withCORS(preflightHandler())))
	mux.Handle("OPTIONS /api/", withCORS(preflightHandler()))
	return mux
}

func preflightHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
}

// withCORS permite que la futura PWA (otro origen) llame a la API con
// Authorization header. Sin cookies: ACAO * es suficiente.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("escribir respuesta JSON", "err", err)
	}
}

func writeEmpty(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// errorStatus escribe el envelope de error con code estable y mensaje
// traducido al idioma negociado de la petición.
func errorStatus(w http.ResponseWriter, r *http.Request, status int, key string) {
	var b struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	b.Error.Code = key
	b.Error.Message = i18n.T(auth.Lang(r), key)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(b); err != nil {
		slog.Error("escribir error JSON", "err", err)
	}
}

// decodeBody decodifica JSON tolerante: decide por el CONTENIDO (primer
// byte '{'/'['), no por el content-type — curl -d manda form-urlencoded
// por defecto aunque el body sea JSON (la spec v1.3 acepta ambos). Si no
// es JSON, deja el body intacto para que el handler haga ParseForm.
func decodeBody(r *http.Request, v any) error {
	if r.Body == nil {
		return nil
	}
	br := bufio.NewReader(io.LimitReader(r.Body, 4<<20))
	lead, err := peekNonSpace(br)
	if err != nil || (lead != '{' && lead != '[') {
		r.Body = io.NopCloser(br)
		return nil
	}
	dec := json.NewDecoder(br)
	if err := dec.Decode(v); err != nil {
		return err
	}
	return nil
}

func peekNonSpace(br *bufio.Reader) (byte, error) {
	for i := 0; i < 64; i++ {
		b, err := br.Peek(1)
		if err != nil {
			return 0, err
		}
		c := b[0]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			br.ReadByte()
			continue
		}
		return c, nil
	}
	return 0, nil
}
