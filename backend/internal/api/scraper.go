package api

import (
	"errors"
	"net/http"

	"github.com/andybalholm/cascadia"

	"github.com/gnacho/ocnews/backend/internal/store"
)

// getFeedScraper: selector CSS de extracción del feed (GET /feeds/{id}/scraper).
func (s *Server) getFeedScraper(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "feedId")
	if !ok {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	if _, err := s.store.GetFeed(user(r).ID, id); errors.Is(err, store.ErrNotFound) {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	sel, err := s.store.GetFeedScraperSelector(id)
	if err != nil {
		s.logError(w, r, "leer selector del feed", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"scraperSelector": sel})
}

// setFeedScraper: fija (o quita) el selector CSS (POST /feeds/{id}/scraper).
func (s *Server) setFeedScraper(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "feedId")
	if !ok {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	if _, err := s.store.GetFeed(user(r).ID, id); errors.Is(err, store.ErrNotFound) {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	var body struct {
		ScraperSelector string `json:"scraperSelector"`
	}
	if err := decodeBody(r, &body); err != nil {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_body")
		return
	}
	sel := body.ScraperSelector
	if sel != "" {
		if _, err := cascadia.Compile(sel); err != nil {
			errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_selector")
			return
		}
	}
	if err := s.store.SetFeedScraperSelector(user(r).ID, id, sel); err != nil {
		s.logError(w, r, "guardar selector del feed", err)
		return
	}
	saved, err := s.store.GetFeedScraperSelector(id)
	if err != nil {
		s.logError(w, r, "leer selector guardado", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"scraperSelector": saved})
}
