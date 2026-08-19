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
	mux.HandleFunc("GET /feeds/discover", s.discoverFeed)
	mux.HandleFunc("DELETE /feeds/{feedId}", s.deleteFeed)
	// move/rename: la spec v1.3 dice POST pero news-android usa PUT → ambos
	for _, m := range []string{http.MethodPost, http.MethodPut} {
		mux.HandleFunc(m+" /feeds/{feedId}/move", s.moveFeed)
		mux.HandleFunc(m+" /feeds/{feedId}/rename", s.renameFeed)
	}
	mux.HandleFunc("POST /feeds/{feedId}/read", s.markFeedRead)
	mux.HandleFunc("GET /feeds/{feedId}/filter", s.getFeedFilter)
	mux.HandleFunc("POST /feeds/{feedId}/filter", s.setFeedFilter)
	mux.HandleFunc("DELETE /feeds/{feedId}/filter", s.deleteFeedFilter)
	mux.HandleFunc("GET /feeds/{feedId}/rules", s.getFeedRules)
	mux.HandleFunc("POST /feeds/{feedId}/rules", s.setFeedRules)
	mux.HandleFunc("DELETE /feeds/{feedId}/rules", s.deleteFeedRules)
	mux.HandleFunc("GET /feeds/{feedId}/retention", s.getFeedRetention)
	mux.HandleFunc("POST /feeds/{feedId}/retention", s.setFeedRetention)
	mux.HandleFunc("POST /feeds/{feedId}/credentials", s.setFeedCredentials)
	mux.HandleFunc("GET /feeds/update", s.updateFeed) // updater API, solo admin

	// Items
	mux.HandleFunc("GET /items", s.listItems)
	mux.HandleFunc("GET /items/updated", s.updatedItems)
	mux.HandleFunc("GET /items/search", s.searchItems)
	mux.HandleFunc("GET /items/{itemId}/full", s.itemFull)
	mux.HandleFunc("POST /items/read", s.markAllRead)
	mux.HandleFunc("POST /items/{itemId}/read", s.markItem(false, "unread"))
	mux.HandleFunc("POST /items/{itemId}/unread", s.markItem(true, "unread"))
	mux.HandleFunc("POST /items/{itemId}/star", s.markItem(true, "starred"))
	mux.HandleFunc("POST /items/{itemId}/unstar", s.markItem(false, "starred"))
	mux.HandleFunc("POST /items/{itemId}/share", s.createShare)
	mux.HandleFunc("DELETE /items/{itemId}/share", s.deleteShare)

	// Marcado múltiple: la spec oficial se contradice (definiciones: POST con
	// "itemIds"; sección How To Sync: PUT con "items") → registramos ambos
	// métodos y aceptamos ambas claves.
	for _, m := range []string{http.MethodPost, http.MethodPut} {
		mux.HandleFunc(m+" /items/read/multiple", s.markMultiple(false, "unread"))
		mux.HandleFunc(m+" /items/unread/multiple", s.markMultiple(true, "unread"))
		mux.HandleFunc(m+" /items/star/multiple", s.markMultiple(true, "starred"))
		mux.HandleFunc(m+" /items/unstar/multiple", s.markMultiple(false, "starred"))
	}

	// Favicon (spec v1.3, News 27.2+): hash md5 de la URL del feed.
	// Se registra como ruta PÚBLICA en Handler() (no requiere auth: solo
	// lee del cache en disco, sin SSRF).

	// Updater API (spec): cleanup + feeds/all, solo admin.
	mux.HandleFunc("GET /cleanup/before-update", s.adminOnly(s.cleanupBefore))
	mux.HandleFunc("GET /cleanup/after-update", s.adminOnly(s.cleanupAfter))
	mux.HandleFunc("GET /feeds/all", s.adminOnly(s.allFeedsForUpdater))

	// OPML (ruta propia de la app, no de la spec v1.3): export/import de
	// suscripciones para la PWA y migración desde otros lectores.
	mux.HandleFunc("GET /export/opml", s.exportOPML)
	mux.HandleFunc("POST /import/opml", s.importOPML)

	// Refresco manual (extensión propia bajo la base v1.3; la ruta updater
	// de la spec es solo-admin). OJO: NADA bajo /api/ — ese prefijo es de
	// OpenCloud (api/v0/settings) y no se puede proxificar a ocnews.
	mux.HandleFunc("POST /refresh", s.refreshUserFeeds)

	// Búsquedas guardadas (extensión propia, feed virtual).
	mux.HandleFunc("GET /searches", s.listSearches)
	mux.HandleFunc("POST /searches", s.createSearch)
	mux.HandleFunc("DELETE /searches/{searchId}", s.deleteSearch)
	mux.HandleFunc("GET /searches/{searchId}/items", s.searchSavedItems)

	// Auto-marcado como leído al llegar (extensión propia).
	mux.HandleFunc("GET /auto-read", s.listAutoRead)
	mux.HandleFunc("POST /auto-read", s.addAutoRead)
	mux.HandleFunc("DELETE /auto-read/{ruleId}", s.deleteAutoRead)
}

// adminOnly envuelve un handler exigiendo rol admin.
func (s *Server) adminOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if user(r).Role != "admin" {
			errorStatus(w, r, http.StatusUnauthorized, "admin_required")
			return
		}
		next(w, r)
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
