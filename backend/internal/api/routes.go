package api

import "net/http"

func (s *Server) routes(mux *http.ServeMux) {
	// Misc
	mux.HandleFunc("GET /version", s.handleVersion)
	mux.HandleFunc("GET /status", s.handleStatus)
	mux.HandleFunc("GET /user", s.handleUser)

	// Folders
	mux.HandleFunc("GET /folders", s.listFolders)
	mux.HandleFunc("POST /folders", s.createFolder)
	mux.HandleFunc("PUT /folders/{folderId}", s.renameFolder)
	mux.HandleFunc("DELETE /folders/{folderId}", s.deleteFolder)
	mux.HandleFunc("POST /folders/{folderId}/read", s.markFolderRead)

	// Feeds
	mux.HandleFunc("GET /feeds", s.listFeeds)
	mux.HandleFunc("POST /feeds", s.createFeed)
	mux.HandleFunc("DELETE /feeds/{feedId}", s.deleteFeed)
	mux.HandleFunc("POST /feeds/{feedId}/move", s.moveFeed)
	mux.HandleFunc("POST /feeds/{feedId}/rename", s.renameFeed)
	mux.HandleFunc("POST /feeds/{feedId}/read", s.markFeedRead)
	mux.HandleFunc("GET /feeds/update", s.updateFeed) // updater API, solo admin

	// Items
	mux.HandleFunc("GET /items", s.listItems)
	mux.HandleFunc("GET /items/updated", s.updatedItems)
	mux.HandleFunc("POST /items/read", s.markAllRead)
	mux.HandleFunc("POST /items/{itemId}/read", s.markItem(false, "unread"))
	mux.HandleFunc("POST /items/{itemId}/unread", s.markItem(true, "unread"))
	mux.HandleFunc("POST /items/{itemId}/star", s.markItem(true, "starred"))
	mux.HandleFunc("POST /items/{itemId}/unstar", s.markItem(false, "starred"))

	// Marcado múltiple: la spec oficial se contradice (definiciones: POST con
	// "itemIds"; sección How To Sync: PUT con "items") → registramos ambos
	// métodos y aceptamos ambas claves.
	for _, m := range []string{http.MethodPost, http.MethodPut} {
		mux.HandleFunc(m+" /items/read/multiple", s.markMultiple(false, "unread"))
		mux.HandleFunc(m+" /items/unread/multiple", s.markMultiple(true, "unread"))
		mux.HandleFunc(m+" /items/star/multiple", s.markMultiple(true, "starred"))
		mux.HandleFunc(m+" /items/unstar/multiple", s.markMultiple(false, "starred"))
	}
}

func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"version": reportedVersion})
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"version": reportedVersion,
		"warnings": map[string]bool{
			"improperlyConfiguredCron": false,
			"incorrectDbCharset":       false,
		},
	})
}

func (s *Server) handleUser(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"userId":             u.Username,
		"displayName":        u.DisplayName,
		"lastLoginTimestamp": u.LastLoginAt,
		"language":           u.Language, // extensión propia; la spec permite atributos nuevos
		"avatar":             nil,
	})
}
