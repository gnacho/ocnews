package api

import (
	"errors"
	"net/http"

	"github.com/gnacho/ocnews/backend/internal/rules"
	"github.com/gnacho/ocnews/backend/internal/store"
)

// rulesBody: reglas block/keep en texto multilínea "Campo=regex".
type rulesBody struct {
	Block string `json:"block"`
	Keep  string `json:"keep"`
}

// validateRules compila las reglas para validar; error 422 con la clave i18n
// invalid_rule si alguna línea es inválida.
func validateRules(body rulesBody) bool {
	_, err := rules.Parse(body.Block, body.Keep)
	return err == nil
}

func (s *Server) getFeedRules(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "feedId")
	if !ok {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	if _, err := s.store.GetFeed(user(r).ID, id); errors.Is(err, store.ErrNotFound) {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	rs, err := s.store.GetFeedRules(id)
	if err != nil {
		s.logError(w, r, "leer reglas del feed", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": rs})
}

// setFeedRules guarda las reglas del feed y las re-aplica a sus items.
func (s *Server) setFeedRules(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "feedId")
	if !ok {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	if _, err := s.store.GetFeed(user(r).ID, id); errors.Is(err, store.ErrNotFound) {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	var body rulesBody
	if err := decodeBody(r, &body); err != nil {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_body")
		return
	}
	if !validateRules(body) {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_rule")
		return
	}
	if err := s.store.SaveFeedRules(id, store.Rules{Block: body.Block, Keep: body.Keep}); err != nil {
		s.logError(w, r, "guardar reglas del feed", err)
		return
	}
	if _, err := s.store.ReapplyFeedFilter(id, store.FeedFilter{}); err != nil {
		s.logError(w, r, "re-aplicar reglas del feed", err)
		return
	}
	saved, err := s.store.GetFeedRules(id)
	if err != nil {
		s.logError(w, r, "leer reglas guardadas", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": saved})
}

// deleteFeedRules borra las reglas del feed y descongela sus items.
func (s *Server) deleteFeedRules(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "feedId")
	if !ok {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	if _, err := s.store.GetFeed(user(r).ID, id); errors.Is(err, store.ErrNotFound) {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	if err := s.store.DeleteFeedRules(id); err != nil {
		s.logError(w, r, "borrar reglas del feed", err)
		return
	}
	if _, err := s.store.ReapplyFeedFilter(id, store.FeedFilter{}); err != nil {
		s.logError(w, r, "descongelar items del feed", err)
		return
	}
	writeEmpty(w)
}

// myRules: reglas globales del usuario (GET /api/me/rules).
func (s *Server) myRules(w http.ResponseWriter, r *http.Request) {
	rs, err := s.store.GetGlobalRules(user(r).ID)
	if err != nil {
		s.logError(w, r, "leer reglas globales", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": rs})
}

// updateMyRules guarda las reglas globales y las re-aplica a todos los items
// del usuario (PUT /api/me/rules).
func (s *Server) updateMyRules(w http.ResponseWriter, r *http.Request) {
	var body rulesBody
	if err := decodeBody(r, &body); err != nil {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_body")
		return
	}
	if !validateRules(body) {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_rule")
		return
	}
	u := user(r)
	if err := s.store.SaveGlobalRules(u.ID, store.Rules{Block: body.Block, Keep: body.Keep}); err != nil {
		s.logError(w, r, "guardar reglas globales", err)
		return
	}
	if _, err := s.store.ReapplyGlobalRules(u.ID); err != nil {
		s.logError(w, r, "re-aplicar reglas globales", err)
		return
	}
	saved, err := s.store.GetGlobalRules(u.ID)
	if err != nil {
		s.logError(w, r, "leer reglas globales guardadas", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": saved})
}
