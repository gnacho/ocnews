package api

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/gnacho/ocnews/backend/internal/store"
)

// listAutoRead: reglas de auto-marcado del usuario (GET /auto-read).
func (s *Server) listAutoRead(w http.ResponseWriter, r *http.Request) {
	rules, err := s.store.ListAutoRead(user(r).ID)
	if err != nil {
		s.logError(w, r, "listar reglas de auto-marcado", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": rules})
}

// addAutoRead: crea una regla de auto-marcado (POST /auto-read).
func (s *Server) addAutoRead(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FeedID       int64  `json:"feedId"`
		TitlePattern string `json:"titlePattern"`
	}
	if err := decodeBody(r, &body); err != nil {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_body")
		return
	}
	body.TitlePattern = strings.TrimSpace(body.TitlePattern)
	if body.TitlePattern == "" {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_rule")
		return
	}
	if _, err := regexp.Compile(body.TitlePattern); err != nil {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_rule")
		return
	}
	u := user(r)
	rule, err := s.store.AddAutoRead(u.ID, body.FeedID, body.TitlePattern)
	if errors.Is(err, store.ErrNotFound) {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	if err != nil {
		s.logError(w, r, "crear regla de auto-marcado", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"rule": rule})
}

// deleteAutoRead: borra una regla de auto-marcado (DELETE /auto-read/{ruleId}).
func (s *Server) deleteAutoRead(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "ruleId")
	if !ok {
		errorStatus(w, r, http.StatusNotFound, "rule_not_found")
		return
	}
	if err := s.store.DeleteAutoRead(user(r).ID, id); errors.Is(err, store.ErrNotFound) {
		errorStatus(w, r, http.StatusNotFound, "rule_not_found")
		return
	} else if err != nil {
		s.logError(w, r, "borrar regla de auto-marcado", err)
		return
	}
	writeEmpty(w)
}
