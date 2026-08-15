// Package auth: middleware de autenticación HTTP Basic contra la tabla
// users (bcrypt). El contexto de petición lleva el *store.User autenticado.
package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/gnacho/ocnews/backend/internal/i18n"
	"github.com/gnacho/ocnews/backend/internal/store"
)

type ctxKey int

const (
	userKey ctxKey = iota + 1
	langKey
)

// HashPassword calcula el hash bcrypt para persistir.
func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

// Authenticate valida user:pass contra la BD y devuelve el usuario.
func Authenticate(s *store.Store, username, password string) (*store.User, error) {
	u, err := s.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, store.ErrNotFound
	}
	s.TouchLogin(u.ID)
	return u, nil
}

// Middleware exige Basic auth válida; 401 con WWW-Authenticate si no.
// El mensaje se negocia por Accept-Language (no hay usuario todavía).
func Middleware(s *store.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := basicAuth(r, s)
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="ocnews"`)
			lang := i18n.Negotiate("auto", r.Header.Get("Accept-Language"))
			writeError(w, http.StatusUnauthorized, "unauthorized", i18n.T(lang, "unauthorized"))
			return
		}
		ctx := context.WithValue(r.Context(), userKey, u)
		ctx = context.WithValue(ctx, langKey, i18n.Negotiate(u.Language, r.Header.Get("Accept-Language")))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Lang recupera el idioma negociado de la petición (EN por defecto).
func Lang(r *http.Request) i18n.Lang {
	l, _ := r.Context().Value(langKey).(i18n.Lang)
	if l == "" {
		return i18n.EN
	}
	return l
}

type errBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	var b errBody
	b.Error.Code = code
	b.Error.Message = msg
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(b)
}

// User recupera el usuario autenticado del contexto (nil si no hay).
func User(r *http.Request) *store.User {
	u, _ := r.Context().Value(userKey).(*store.User)
	return u
}

func basicAuth(r *http.Request, s *store.Store) (*store.User, bool) {
	h := r.Header.Get("Authorization")
	const prefix = "Basic "
	if !strings.HasPrefix(h, prefix) {
		return nil, false
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(h, prefix))
	if err != nil {
		return nil, false
	}
	user, pass, ok := strings.Cut(string(raw), ":")
	if !ok {
		return nil, false
	}
	u, err := Authenticate(s, user, pass)
	if err != nil {
		return nil, false
	}
	return u, true
}
