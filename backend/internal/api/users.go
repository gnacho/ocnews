package api

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"

	"golang.org/x/crypto/bcrypt"

	"github.com/gnacho/ocnews/backend/internal/auth"
	"github.com/gnacho/ocnews/backend/internal/store"
)

// userAPI: endpoints PROPIOS de la app (no de la spec v1.3) montados bajo
// /api/ para la PWA y la extensión: perfil propio y gestión de usuarios.
func (s *Server) userAPI() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/me", s.me)
	mux.HandleFunc("PUT /api/me", s.updateMe)
	mux.HandleFunc("PUT /api/me/password", s.changeMyPassword)
	mux.HandleFunc("GET /api/me/settings", s.mySettings)
	mux.HandleFunc("PUT /api/me/settings", s.updateMySettings)
	mux.HandleFunc("GET /api/users", s.adminOnly(s.listUsers))
	mux.HandleFunc("POST /api/users", s.adminOnly(s.createUser))
	mux.HandleFunc("PUT /api/users/{id}", s.adminOnly(s.updateUser))
	mux.HandleFunc("DELETE /api/users/{id}", s.adminOnly(s.deleteUser))
	return mux
}

// refreshUserFeeds: refresco manual e inmediato de TODOS los feeds del
// usuario autenticado (la ruta /feeds/update de la spec es solo admin).
func (s *Server) refreshUserFeeds(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	feeds, err := s.store.ListFeeds(u.ID)
	if err != nil {
		s.logError(w, r, "listar feeds para refresh", err)
		return
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	inserted := int64(0)
	for i := range feeds {
		f := feeds[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if res := s.refresher.Refresh(r.Context(), &f); res.Err == nil {
				atomic.AddInt64(&inserted, res.Inserted)
			}
		}()
	}
	wg.Wait()
	s.log.Info("refresco manual", "user", u.Username, "feeds", len(feeds), "nuevos", inserted)
	writeJSON(w, http.StatusOK, map[string]any{"refreshed": len(feeds), "newItems": inserted})
}

// ocsUser: stub del endpoint OCS de Nextcloud que consume news-android
// (display name del drawer). Formato OCS v2 con "ocs"→"data"→{id,displayname}.
func (s *Server) ocsUser(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"ocs": map[string]any{
			"meta": map[string]any{"status": "ok", "statuscode": 200, "message": "OK"},
			"data": map[string]any{
				"id":           u.Username,
				"displayname":  u.DisplayName,
				"display-name": u.DisplayName,
			},
		},
	})
}

var validUsername = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{2,31}$`)

type meResponse struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	Language    string `json:"language"`
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	writeJSON(w, http.StatusOK, meResponse{u.Username, u.DisplayName, u.Role, u.Language})
}

func (s *Server) updateMe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DisplayName *string `json:"displayName"`
		Language    *string `json:"language"`
	}
	if err := decodeBody(r, &body); err != nil {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_body")
		return
	}
	u := user(r)
	name, lang := u.DisplayName, u.Language
	if body.DisplayName != nil {
		name = *body.DisplayName
	}
	if body.Language != nil {
		if !validLanguage(*body.Language) {
			errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_language")
			return
		}
		lang = *body.Language
	}
	if err := s.store.UpdateProfile(u.ID, name, lang); err != nil {
		s.logError(w, r, "actualizar perfil", err)
		return
	}
	writeJSON(w, http.StatusOK, meResponse{u.Username, name, u.Role, lang})
}

func (s *Server) changeMyPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Current string `json:"currentPassword"`
		New     string `json:"newPassword"`
	}
	if err := decodeBody(r, &body); err != nil {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_body")
		return
	}
	u := user(r)
	// la contraseña actual debe validar (bcrypt contra el hash almacenado)
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(body.Current)) != nil {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_password")
		return
	}
	if !validPassword(body.New) {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_password")
		return
	}
	hash, err := auth.HashPassword(body.New)
	if err != nil {
		s.logError(w, r, "hashear contraseña", err)
		return
	}
	if err := s.store.UpdatePassword(u.ID, hash); err != nil {
		s.logError(w, r, "cambiar contraseña", err)
		return
	}
	writeEmpty(w)
}

// userSettings: preferencias de la app persistidas por usuario. La fuente de
// verdad del idioma sigue siendo users.language; el resto va en user_settings.
type userSettings struct {
	Theme            string `json:"theme"`            // system|light|dark
	ReaderMaxWidth   string `json:"readerMaxWidth"`   // narrow|normal|wide
	FeedIntervalMin  string `json:"feedIntervalMin"`  // minutos; vacío = default global
}

func (s *Server) mySettings(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	theme, _ := s.store.GetUserSetting(u.ID, "theme")
	width, _ := s.store.GetUserSetting(u.ID, "reader_max_width")
	interval, _ := s.store.GetUserSetting(u.ID, "feed_interval_min")
	if theme == "" {
		theme = "system"
	}
	if width == "" {
		width = "normal"
	}
	writeJSON(w, http.StatusOK, userSettings{theme, width, interval})
}

func (s *Server) updateMySettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Theme           *string `json:"theme"`
		ReaderMaxWidth  *string `json:"readerMaxWidth"`
		FeedIntervalMin *string `json:"feedIntervalMin"`
	}
	if err := decodeBody(r, &body); err != nil {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_body")
		return
	}
	u := user(r)
	if body.Theme != nil {
		if !validTheme(*body.Theme) {
			errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_theme")
			return
		}
		v := *body.Theme
		if v == "system" {
			v = ""
		}
		if err := s.store.SetUserSetting(u.ID, "theme", v); err != nil {
			s.logError(w, r, "guardar tema", err)
			return
		}
	}
	if body.ReaderMaxWidth != nil {
		if !validWidth(*body.ReaderMaxWidth) {
			errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_reader_width")
			return
		}
		v := *body.ReaderMaxWidth
		if v == "normal" {
			v = ""
		}
		if err := s.store.SetUserSetting(u.ID, "reader_max_width", v); err != nil {
			s.logError(w, r, "guardar ancho lector", err)
			return
		}
	}
	if body.FeedIntervalMin != nil {
		v := *body.FeedIntervalMin
		if v != "" {
			if !validInterval(v) {
				errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_feed_interval")
				return
			}
		}
		if err := s.store.SetUserSetting(u.ID, "feed_interval_min", v); err != nil {
			s.logError(w, r, "guardar intervalo", err)
			return
		}
	}
	s.mySettings(w, r)
}

func validTheme(t string) bool   { return t == "system" || t == "light" || t == "dark" }
func validWidth(w string) bool   { return w == "narrow" || w == "normal" || w == "wide" }
func validInterval(v string) bool {
	n, err := strconv.Atoi(v)
	return err == nil && n >= 5 && n <= 1440
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers()
	if err != nil {
		s.logError(w, r, "listar usuarios", err)
		return
	}
	out := make([]meResponse, 0, len(users))
	for _, u := range users {
		out = append(out, meResponse{u.Username, u.DisplayName, u.Role, u.Language})
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": out})
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username    string  `json:"username"`
		Password    string  `json:"password"`
		DisplayName string  `json:"displayName"`
		Role        string  `json:"role"`
		Language    *string `json:"language"`
	}
	if err := decodeBody(r, &body); err != nil {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_body")
		return
	}
	if !validUsername.MatchString(body.Username) {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_username")
		return
	}
	if !validPassword(body.Password) {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_password")
		return
	}
	role := "user"
	if body.Role != "" {
		if !validRole(body.Role) {
			errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_role")
			return
		}
		role = body.Role
	}
	language := "auto"
	if body.Language != nil {
		if !validLanguage(*body.Language) {
			errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_language")
			return
		}
		language = *body.Language
	}
	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		s.logError(w, r, "hashear contraseña", err)
		return
	}
	id, err := s.store.CreateUser(body.Username, hash, body.DisplayName, role)
	if err != nil {
		errorStatus(w, r, http.StatusConflict, "user_exists")
		return
	}
	if language != "auto" {
		s.store.SetUserLanguage(id, language)
	}
	created, err := s.store.GetUserByID(id)
	if err != nil {
		s.logError(w, r, "leer usuario creado", err)
		return
	}
	writeJSON(w, http.StatusCreated, meResponse{created.Username, created.DisplayName, created.Role, created.Language})
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		errorStatus(w, r, http.StatusNotFound, "user_not_found")
		return
	}
	target, err := s.store.GetUserByID(id)
	if errors.Is(err, store.ErrNotFound) {
		errorStatus(w, r, http.StatusNotFound, "user_not_found")
		return
	}
	if err != nil {
		s.logError(w, r, "leer usuario", err)
		return
	}
	var body struct {
		Role     *string `json:"role"`
		Language *string `json:"language"`
		Password string  `json:"password"`
	}
	if err := decodeBody(r, &body); err != nil {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_body")
		return
	}
	if body.Role != nil && !validRole(*body.Role) {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_role")
		return
	}
	if body.Language != nil && !validLanguage(*body.Language) {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_language")
		return
	}
	// guard: degradar al último admin deja el sistema cerrado
	if body.Role != nil && *body.Role == "user" && target.Role == "admin" {
		if admins, err := s.store.CountAdmins(); err == nil && admins <= 1 {
			errorStatus(w, r, http.StatusConflict, "cannot_delete_last_admin")
			return
		}
	}
	if body.Password != "" {
		if !validPassword(body.Password) {
			errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_password")
			return
		}
		hash, err := auth.HashPassword(body.Password)
		if err != nil {
			s.logError(w, r, "hashear contraseña", err)
			return
		}
		if err := s.store.UpdatePassword(id, hash); err != nil {
			s.logError(w, r, "cambiar contraseña", err)
			return
		}
	}
	if err := s.store.UpdateUserFields(id, body.Role, body.Language); err != nil {
		s.logError(w, r, "actualizar usuario", err)
		return
	}
	updated, _ := s.store.GetUserByID(id)
	writeJSON(w, http.StatusOK, meResponse{updated.Username, updated.DisplayName, updated.Role, updated.Language})
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		errorStatus(w, r, http.StatusNotFound, "user_not_found")
		return
	}
	if id == user(r).ID {
		errorStatus(w, r, http.StatusConflict, "cannot_delete_self")
		return
	}
	switch err := s.store.DeleteUser(id); {
	case errors.Is(err, store.ErrNotFound):
		errorStatus(w, r, http.StatusNotFound, "user_not_found")
	case errors.Is(err, store.ErrConflict):
		errorStatus(w, r, http.StatusConflict, "cannot_delete_last_admin")
	case err != nil:
		s.logError(w, r, "borrar usuario", err)
	default:
		writeEmpty(w)
	}
}

func validLanguage(l string) bool { return l == "auto" || l == "es" || l == "en" }
func validRole(role string) bool  { return role == "user" || role == "admin" }
func validPassword(pw string) bool { return len(pw) >= 8 }
