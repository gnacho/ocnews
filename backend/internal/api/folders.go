package api

import (
	"errors"
	"net/http"

	"github.com/gnacho/ocnews/backend/internal/auth"
	"github.com/gnacho/ocnews/backend/internal/store"
)

func user(r *http.Request) *store.User { return auth.User(r) }

func pathID(r *http.Request, name string) (int64, bool) {
	id, err := parseID(r.PathValue(name))
	return id, err == nil && id > 0
}

func (s *Server) listFolders(w http.ResponseWriter, r *http.Request) {
	folders, err := s.store.ListFolders(user(r).ID)
	if err != nil {
		s.logError(w, r, "listar carpetas", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"folders": folders})
}

func (s *Server) createFolder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string `json:"name"`
		ParentID *int64 `json:"parentId"`
	}
	if err := decodeBody(r, &body); err != nil {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_body")
		return
	}
	if body.Name == "" {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_folder_name")
		return
	}
	f, err := s.store.CreateFolder(user(r).ID, body.Name, body.ParentID)
	if errors.Is(err, store.ErrConflict) {
		errorStatus(w, r, http.StatusConflict, "folder_exists")
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		errorStatus(w, r, http.StatusNotFound, "folder_not_found")
		return
	}
	if err != nil {
		s.logError(w, r, "crear carpeta", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"folders": []store.Folder{*f}})
}

func (s *Server) renameFolder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "folderId")
	if !ok {
		errorStatus(w, r, http.StatusNotFound, "folder_not_found")
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeBody(r, &body); err != nil || body.Name == "" {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_folder_name")
		return
	}
	switch err := s.store.RenameFolder(user(r).ID, id, body.Name); {
	case errors.Is(err, store.ErrNotFound):
		errorStatus(w, r, http.StatusNotFound, "folder_not_found")
	case errors.Is(err, store.ErrConflict):
		errorStatus(w, r, http.StatusConflict, "folder_exists")
	case err != nil:
		s.logError(w, r, "renombrar carpeta", err)
	default:
		writeEmpty(w)
	}
}

func (s *Server) deleteFolder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "folderId")
	if !ok {
		errorStatus(w, r, http.StatusNotFound, "folder_not_found")
		return
	}
	if err := s.store.DeleteFolder(user(r).ID, id); errors.Is(err, store.ErrNotFound) {
		errorStatus(w, r, http.StatusNotFound, "folder_not_found")
	} else if err != nil {
		s.logError(w, r, "borrar carpeta", err)
	} else {
		writeEmpty(w)
	}
}

// markFolderRead marca leídos los items de una carpeta hasta newestItemId.
func (s *Server) markFolderRead(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "folderId")
	if !ok {
		errorStatus(w, r, http.StatusNotFound, "folder_not_found")
		return
	}
	maxID, ok := newestItemID(r, w)
	if !ok {
		return
	}
	exists, err := s.store.FolderExists(user(r).ID, id)
	if err != nil || !exists {
		errorStatus(w, r, http.StatusNotFound, "folder_not_found")
		return
	}
	if _, err := s.store.MarkAllRead(user(r).ID, maxID, "folder", id); err != nil {
		s.logError(w, r, "marcar carpeta leída", err)
		return
	}
	writeEmpty(w)
}

func (s *Server) logError(w http.ResponseWriter, r *http.Request, what string, err error) {
	s.log.Error(what, "err", err)
	errorStatus(w, r, http.StatusInternalServerError, "internal_error")
}
