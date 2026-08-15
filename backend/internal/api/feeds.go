package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gnacho/ocnews/backend/internal/feed"
	"github.com/gnacho/ocnews/backend/internal/store"
)

type feedsResponse struct {
	Feeds        []store.Feed `json:"feeds"`
	StarredCount int64        `json:"starredCount"`
	NewestItemID int64        `json:"newestItemId,omitempty"`
}

func (s *Server) listFeeds(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	feeds, err := s.store.ListFeeds(u.ID)
	if err != nil {
		s.logError(w, r, "listar feeds", err)
		return
	}
	starred, _ := s.store.StarredCount(u.ID)
	newest, _ := s.store.NewestItemID(u.ID)
	writeJSON(w, http.StatusOK, feedsResponse{feeds, starred, newest})
}

// createFeed suscribe un feed nuevo: valida, fetchea (F0: en la propia
// petición con timeout) y persiste feed+items. Acepta JSON (spec) y
// form-urlencoded/query (news-android manda @Field form).
func (s *Server) createFeed(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL      string `json:"url"`
		FolderID *int64 `json:"folderId"`
	}
	if err := decodeBody(r, &body); err != nil {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_body")
		return
	}
	// fallback form-urlencoded / query (ParseForm cubre ambos)
	if body.URL == "" {
		if err := r.ParseForm(); err == nil {
			if v := r.Form.Get("url"); v != "" {
				body.URL = v
			}
			if v := r.Form.Get("folderId"); v != "" {
				var fid int64
				if _, err := fmt.Sscanf(v, "%d", &fid); err == nil && fid > 0 {
					body.FolderID = &fid
				}
			}
		}
	}
	if body.URL == "" {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_url")
		return
	}
	if body.FolderID != nil && *body.FolderID > 0 {
		exists, err := s.store.FolderExists(user(r).ID, *body.FolderID)
		if err != nil || !exists {
			errorStatus(w, r, http.StatusNotFound, "folder_not_found")
			return
		}
	}
	if exists, _ := s.store.FeedExistsByURL(user(r).ID, body.URL); exists {
		errorStatus(w, r, http.StatusConflict, "feed_exists")
		return
	}

	f, items, err := s.fetcher.Fetch(r.Context(), body.URL)
	if err != nil {
		// spec: 422 si el feed no se puede leer
		s.log.Warn("fetch de feed nuevo falló", "url", body.URL, "err", err)
		errorStatus(w, r, http.StatusUnprocessableEntity, "feed_unreadable")
		return
	}
	f.URL = body.URL // la URL de suscripción manda sobre la declarada en el XML
	if body.FolderID != nil && *body.FolderID > 0 {
		f.FolderID = body.FolderID
	}
	feed.SanitizeItems(items) // los items de la suscripción también se limpian

	created, err := s.store.CreateFeed(user(r).ID, body.URL, f.FolderID, f.Title, f.Link, f.FaviconLink, items)
	if errors.Is(err, store.ErrConflict) {
		errorStatus(w, r, http.StatusConflict, "feed_exists")
		return
	}
	if err != nil {
		s.logError(w, r, "crear feed", err)
		return
	}
	newest, _ := s.store.NewestItemID(user(r).ID)
	s.log.Info("feed suscrito", "url", body.URL, "items", len(items))
	writeJSON(w, http.StatusOK, feedsResponse{[]store.Feed{*created}, 0, newest})
}

func (s *Server) deleteFeed(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "feedId")
	if !ok {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	if err := s.store.DeleteFeed(user(r).ID, id); errors.Is(err, store.ErrNotFound) {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
	} else if err != nil {
		s.logError(w, r, "borrar feed", err)
	} else {
		writeEmpty(w)
	}
}

func (s *Server) moveFeed(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "feedId")
	if !ok {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	var body struct {
		FolderID *int64 `json:"folderId"`
	}
	if err := decodeBody(r, &body); err != nil {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_body")
		return
	}
	switch err := s.store.MoveFeed(user(r).ID, id, normFolderID(body.FolderID)); {
	case errors.Is(err, store.ErrNotFound):
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
	case err != nil:
		s.logError(w, r, "mover feed", err)
	default:
		writeEmpty(w)
	}
}

func (s *Server) renameFeed(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "feedId")
	if !ok {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	var body struct {
		FeedTitle string `json:"feedTitle"`
	}
	if err := decodeBody(r, &body); err != nil || body.FeedTitle == "" {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_title")
		return
	}
	if err := s.store.RenameFeed(user(r).ID, id, body.FeedTitle); errors.Is(err, store.ErrNotFound) {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
	} else if err != nil {
		s.logError(w, r, "renombrar feed", err)
	} else {
		writeEmpty(w)
	}
}

func (s *Server) markFeedRead(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "feedId")
	if !ok {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	maxID, ok := newestItemID(r, w)
	if !ok {
		return
	}
	if _, err := s.store.GetFeed(user(r).ID, id); errors.Is(err, store.ErrNotFound) {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	if _, err := s.store.MarkAllRead(user(r).ID, maxID, "feed", id); err != nil {
		s.logError(w, r, "marcar feed leído", err)
		return
	}
	writeEmpty(w)
}

// updateFeed: updater API oficial (GET /feeds/update?userId=&feedId=).
// Requiere rol admin; refetchea el feed indicado.
func (s *Server) updateFeed(w http.ResponseWriter, r *http.Request) {
	if user(r).Role != "admin" {
		errorStatus(w, r, http.StatusUnauthorized, "admin_required")
		return
	}
	q := r.URL.Query()
	username := q.Get("userId")
	feedID, err := parseID(q.Get("feedId"))
	if err != nil || feedID <= 0 {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	owner, err := s.store.GetUserByUsername(username)
	if err != nil {
		errorStatus(w, r, http.StatusNotFound, "user_not_found")
		return
	}
	f, err := s.store.GetFeed(owner.ID, feedID)
	if err != nil {
		errorStatus(w, r, http.StatusNotFound, "feed_not_found")
		return
	}
	if res := s.refresher.Refresh(r.Context(), f); res.Err != nil {
		s.logError(w, r, "actualizar feed", res.Err)
		return
	}
	writeEmpty(w)
}

// normFolderID normaliza folderId 0 → nil (raíz).
func normFolderID(id *int64) *int64 {
	if id != nil && *id == 0 {
		return nil
	}
	return id
}
