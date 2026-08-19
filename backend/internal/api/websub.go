package api

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// websubRouter: callback PÚBLICO del hub (no lo llama el navegador). La
// seguridad la aporta la comprobación del topic y la firma X-Hub-Signature.
func (s *Server) websubRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /websub/callback/{feedId}", s.websubVerify)
	mux.HandleFunc("POST /websub/callback/{feedId}", s.websubDelivery)
	return mux
}

// websubVerify: GET de verificación de la suscripción. Responde el challenge.
func (s *Server) websubVerify(w http.ResponseWriter, r *http.Request) {
	feedID, err := parseID(r.PathValue("feedId"))
	if err != nil || feedID <= 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	q := r.URL.Query()
	if q.Get("hub.mode") != "subscribe" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	sub, err := s.store.GetWebSub(feedID)
	if err != nil || sub.Topic != q.Get("hub.topic") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if ls := q.Get("hub.lease_seconds"); ls != "" {
		if n, err := strconv.ParseInt(ls, 10, 64); err == nil && n > 0 {
			_ = s.store.SaveWebSubLease(feedID, time.Now().Add(time.Duration(n)*time.Second).Unix())
		}
	}
	_ = s.store.SaveWebSubStatus(feedID, "active")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	//nolint:gosec // el challenge del protocolo WebSub se devuelve como texto
	// plano (no HTML); el valor lo genera el hub y no se renderiza en ningún navegador.
	_, _ = w.Write([]byte(q.Get("hub.challenge")))
}

// websubDelivery: POST del hub con un feed actualizado. Verifica la firma e
// ingesta; si la ingesta falla (feed sin cambios o erróneo) responde 2xx y el
// polling periódico sigue como red de seguridad.
func (s *Server) websubDelivery(w http.ResponseWriter, r *http.Request) {
	feedID, err := parseID(r.PathValue("feedId"))
	if err != nil || feedID <= 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	sub, err := s.store.GetWebSub(feedID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 20<<20))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}
	if sub.Secret != "" {
		sig := r.Header.Get("X-Hub-Signature")
		if !verifyHubSignature(sig, sub.Secret, body) {
			http.Error(w, "bad signature", http.StatusUnauthorized)
			return
		}
	}
	if res := s.refresher.Ingest(r.Context(), feedID, body); res.Err != nil {
		s.log.Warn("websub: ingesta falló", "feed", feedID, "err", res.Err)
	}
	w.WriteHeader(http.StatusNoContent)
}

// verifyHubSignature valida X-Hub-Signature ("sha1=hex" o "sha256=hex").
func verifyHubSignature(sig, secret string, body []byte) bool {
	parts := strings.SplitN(sig, "=", 2)
	if len(parts) != 2 {
		return false
	}
	provided, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}
	var mac []byte
	switch strings.ToLower(parts[0]) {
	case "sha1":
		h := hmac.New(sha1.New, []byte(secret))
		h.Write(body)
		mac = h.Sum(nil)
	case "sha256":
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		mac = h.Sum(nil)
	default:
		return false
	}
	return subtle.ConstantTimeCompare(mac, provided) == 1
}
