// Package auth: middleware de autenticación HTTP Basic con validador
// intercambiable (local bcrypt u OpenCloud Graph). El contexto de petición
// lleva el *store.User autenticado y el idioma negociado.
package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

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
	b, err := bcryptGenerate(pw)
	return string(b), err
}

// Middleware exige Basic auth válida; 401 con WWW-Authenticate si no.
// El mensaje se negocia por Accept-Language (no hay usuario todavía).
func Middleware(v Validator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := basicCredentials(r)
		var u *store.User
		if ok {
			u, ok = v.Validate(r.Context(), username, password)
		}
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

// User recupera el usuario autenticado del contexto (nil si no hay).
func User(r *http.Request) *store.User {
	u, _ := r.Context().Value(userKey).(*store.User)
	return u
}

// Lang recupera el idioma negociado de la petición (EN por defecto).
func Lang(r *http.Request) i18n.Lang {
	l, _ := r.Context().Value(langKey).(i18n.Lang)
	if l == "" {
		return i18n.EN
	}
	return l
}

func basicCredentials(r *http.Request) (string, string, bool) {
	h := r.Header.Get("Authorization")
	const prefix = "Basic "
	if !strings.HasPrefix(h, prefix) {
		return "", "", false
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(h, prefix))
	if err != nil {
		return "", "", false
	}
	user, pass, ok := strings.Cut(string(raw), ":")
	return user, pass, ok
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
