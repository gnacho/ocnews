package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gnacho/ocnews/backend/internal/store"
)

// filterBody: keywords separadas por coma (News 28.4.0).
type filterBody struct {
	TitleKeywords string `json:"titleKeywords"`
	BodyKeywords  string `json:"bodyKeywords"`
	URLKeywords   string `json:"urlKeywords"`
}

// sanitizeKeywords trima y normaliza (sin espacios colgantes por keyword).
func sanitizeKeywords(s string) string {
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, ",")
}

func (b filterBody) toStore(feedID int64) store.FeedFilter {
	return store.FeedFilter{
		FeedID:        feedID,
		TitleKeywords: sanitizeKeywords(b.TitleKeywords),
		BodyKeywords:  sanitizeKeywords(b.BodyKeywords),
		URLKeywords:   sanitizeKeywords(b.URLKeywords),
	}
}

// getFeedFilter devuelve el filtro del feed (GET /feeds/{id}/filter).
func (s *Server) getFeedFilter(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "feedId")
	if !ok {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	if _, err := s.store.GetFeed(user(r).ID, id); errors.Is(err, store.ErrNotFound) {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	f, err := s.store.GetFeedFilter(id)
	if err != nil {
		s.logError(w, r, "leer filtro", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"filter": f})
}

// setFeedFilter crea o actualiza el filtro (POST /feeds/{id}/filter).
// Keywords vacías → se elimina el filtro. Re-aplica sobre los items ya guardados.
func (s *Server) setFeedFilter(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "feedId")
	if !ok {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	if _, err := s.store.GetFeed(user(r).ID, id); errors.Is(err, store.ErrNotFound) {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	var body filterBody
	if err := decodeBody(r, &body); err != nil {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_body")
		return
	}
	f := body.toStore(id)
	if err := s.store.SaveFeedFilter(f); err != nil {
		s.logError(w, r, "guardar filtro", err)
		return
	}
	if _, err := s.store.ReapplyFeedFilter(id, f); err != nil {
		s.logError(w, r, "re-aplicar filtro", err)
		return
	}
	saved, _ := s.store.GetFeedFilter(id)
	writeJSON(w, http.StatusOK, map[string]any{"filter": saved})
}

// deleteFeedFilter elimina el filtro y descongela los items (DELETE /feeds/{id}/filter).
func (s *Server) deleteFeedFilter(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "feedId")
	if !ok {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	if _, err := s.store.GetFeed(user(r).ID, id); errors.Is(err, store.ErrNotFound) {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	if err := s.store.DeleteFeedFilter(id); err != nil {
		s.logError(w, r, "borrar filtro", err)
		return
	}
	// descongelar items previamente filtrados de este feed
	if _, err := s.store.ReapplyFeedFilter(id, store.FeedFilter{}); err != nil {
		s.logError(w, r, "descongelar items", err)
		return
	}
	writeEmpty(w)
}
