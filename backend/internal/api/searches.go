package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gnacho/ocnews/backend/internal/store"
)

// listSearches: búsquedas guardadas del usuario (GET /searches).
func (s *Server) listSearches(w http.ResponseWriter, r *http.Request) {
	searches, err := s.store.ListSavedSearches(user(r).ID)
	if err != nil {
		s.logError(w, r, "listar búsquedas guardadas", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"searches": searches})
}

// createSearch: guarda una búsqueda (POST /searches).
func (s *Server) createSearch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name  string `json:"name"`
		Query string `json:"query"`
	}
	if err := decodeBody(r, &body); err != nil {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_body")
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	body.Query = strings.TrimSpace(body.Query)
	if body.Name == "" || body.Query == "" {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_search")
		return
	}
	ss, err := s.store.CreateSavedSearch(user(r).ID, body.Name, body.Query)
	if err != nil {
		s.logError(w, r, "crear búsqueda guardada", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"search": ss})
}

// deleteSearch: borra una búsqueda guardada (DELETE /searches/{searchId}).
func (s *Server) deleteSearch(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "searchId")
	if !ok {
		errorStatus(w, r, http.StatusNotFound, "search_not_found")
		return
	}
	if err := s.store.DeleteSavedSearch(user(r).ID, id); errors.Is(err, store.ErrNotFound) {
		errorStatus(w, r, http.StatusNotFound, "search_not_found")
		return
	} else if err != nil {
		s.logError(w, r, "borrar búsqueda guardada", err)
		return
	}
	writeEmpty(w)
}

// searchSavedItems: ejecuta la búsqueda guardada (GET /searches/{searchId}/items).
func (s *Server) searchSavedItems(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "searchId")
	if !ok {
		errorStatus(w, r, http.StatusNotFound, "search_not_found")
		return
	}
	u := user(r)
	ss, err := s.store.GetSavedSearch(u.ID, id)
	if errors.Is(err, store.ErrNotFound) {
		errorStatus(w, r, http.StatusNotFound, "search_not_found")
		return
	}
	if err != nil {
		s.logError(w, r, "leer búsqueda guardada", err)
		return
	}
	f, valid := itemFilterFromRequest(r, u, false)
	if !valid {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_type")
		return
	}
	if f.BatchSize < 0 || f.BatchSize > 200 {
		f.BatchSize = 100
	}
	items, err := s.store.SearchItems(f, ss.Query, int(f.BatchSize))
	if err != nil {
		s.logError(w, r, "buscar items de búsqueda guardada", err)
		return
	}
	s.rewriteAll(items)
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
