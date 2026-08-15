package api

import (
	"encoding/xml"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gnacho/ocnews/backend/internal/store"
)

// serveFavicon: GET /favicon/{feedUrlHash} — responde desde la caché.
// Solo servimos si el hash corresponde a un feed existente.
func (s *Server) serveFavicon(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	if hash == "" || s.favicons == nil {
		http.NotFound(w, r)
		return
	}
	if _, err := s.store.FeedByURLHash(hash); err != nil {
		http.NotFound(w, r)
		return
	}
	s.favicons.Serve(w, hash)
}

// cleanupBefore: spec — borra carpetas/feeds marcados para borrar. En nuestra
// implementación los DELETE ya son inmediatos; queda como no-op de contrato.
func (s *Server) cleanupBefore(w http.ResponseWriter, _ *http.Request) {
	writeEmpty(w)
}

// cleanupAfter: spec — purga items leídos no destacados según retención.
func (s *Server) cleanupAfter(w http.ResponseWriter, r *http.Request) {
	if s.retention > 0 {
		cutoff := time.Now().Add(-s.retention).Unix()
		if n, err := s.store.PurgeOldItems(cutoff); err != nil {
			s.logError(w, r, "cleanup after-update", err)
			return
		} else if n > 0 {
			s.log.Info("cleanup after-update", "items", n)
		}
	}
	writeEmpty(w)
}

// allFeedsForUpdater: spec — feeds con userId para el updater externo.
func (s *Server) allFeedsForUpdater(w http.ResponseWriter, r *http.Request) {
	feeds, err := s.store.AllFeedsWithUser()
	if err != nil {
		s.logError(w, r, "feeds/all", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"feeds": feeds})
}

// ---- OPML ----

type opmlOutline struct {
	XMLName  xml.Name      `xml:"outline"`
	Text     string        `xml:"text,attr"`
	Title    string        `xml:"title,attr,omitempty"`
	Type     string        `xml:"type,attr,omitempty"`
	XMLURL   string        `xml:"xmlUrl,attr,omitempty"`
	Outlines []opmlOutline `xml:"outline"`
}

type opmlDoc struct {
	XMLName   xml.Name `xml:"opml"`
	Version   string   `xml:"version,attr"`
	Title     string   `xml:"head>title"`
	Outlines  []opmlOutline `xml:"body>outline"`
}

// exportOPML: GET /export/opml — suscripciones del usuario en OPML 2.0.
func (s *Server) exportOPML(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	folders, err := s.store.ListFolders(u.ID)
	if err != nil {
		s.logError(w, r, "exportar opml", err)
		return
	}
	feeds, err := s.store.ListFeeds(u.ID)
	if err != nil {
		s.logError(w, r, "exportar opml", err)
		return
	}

	byFolder := map[int64][]opmlOutline{}
	var root []opmlOutline
	for _, f := range feeds {
		o := opmlOutline{Text: f.Title, Title: f.Title, Type: "rss", XMLURL: f.URL}
		if f.FolderID != nil {
			byFolder[*f.FolderID] = append(byFolder[*f.FolderID], o)
		} else {
			root = append(root, o)
		}
	}
	for _, fo := range folders {
		children := byFolder[fo.ID]
		if children == nil {
			children = []opmlOutline{}
		}
		root = append(root, opmlOutline{Text: fo.Name, Title: fo.Name, Outlines: children})
	}

	doc := opmlDoc{Version: "2.0", Title: "ocnews suscripciones", Outlines: root}
	w.Header().Set("Content-Type", "text/x-opml; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="subscriptions.opml"`)
	w.WriteHeader(http.StatusOK)
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(doc); err != nil {
		s.log.Error("codificar opml", "err", err)
	}
}

type importedFeed struct {
	URL        string
	FolderName string
}

// parseOPML extrae feeds (y su carpeta contenedora) de un documento OPML.
// Tolerante: carpetas anidadas se aplanan (el modelo solo tiene 1 nivel).
func parseOPML(body []byte) []importedFeed {
	var doc opmlDoc
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil
	}
	var out []importedFeed
	var walk func(outlines []opmlOutline, folder string)
	walk = func(outlines []opmlOutline, folder string) {
		for _, o := range outlines {
			if o.XMLURL != "" {
				name := folder
				out = append(out, importedFeed{URL: strings.TrimSpace(o.XMLURL), FolderName: name})
				continue
			}
			f := o.Text
			if f == "" {
				f = o.Title
			}
			walk(o.Outlines, f)
		}
	}
	walk(doc.Outlines, "")
	return out
}

// importOPML: POST /import/opml — importa suscripciones (crea carpetas,
// feeds sin fetchejar; el scheduler los refresca en el próximo ciclo al
// tener next_update=0).
func (s *Server) importOPML(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_body")
		return
	}
	imports := parseOPML(body)
	if len(imports) == 0 {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_body")
		return
	}

	created, skipped := 0, 0
	for _, imp := range imports {
		var folderID *int64
		if imp.FolderName != "" {
			f, err := s.store.CreateFolder(u.ID, imp.FolderName)
			if err == store.ErrConflict {
				existing, err2 := s.store.ListFolders(u.ID)
				if err2 == nil {
					for _, fo := range existing {
						if fo.Name == imp.FolderName {
							id := fo.ID
							folderID = &id
							break
						}
					}
				}
			} else if err == nil {
				folderID = &f.ID
			}
		}
		if exists, _ := s.store.FeedExistsByURL(u.ID, imp.URL); exists {
			skipped++
			continue
		}
		// feed sin fetch inicial: el scheduler lo rescata (next_update 0)
		if _, err := s.store.CreateFeedDeferred(u.ID, imp.URL, folderID); err == nil {
			created++
		} else {
			skipped++
		}
	}
	s.log.Info("opml importado", "creados", created, "omitidos", skipped)
	writeJSON(w, http.StatusOK, map[string]int{"imported": created, "skipped": skipped})
}
