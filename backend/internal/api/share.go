package api

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"time"

	"github.com/gnacho/ocnews/backend/internal/store"
)

// createShare: habilita la URL pública de un artículo (POST /items/{id}/share).
func (s *Server) createShare(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("itemId"))
	if err != nil || id <= 0 {
		errorStatus(w, r, http.StatusNotFound, "item_not_found")
		return
	}
	u := user(r)
	exists, err := s.store.ItemExists(u.ID, id)
	if err != nil {
		s.logError(w, r, "comprobar item", err)
		return
	}
	if !exists {
		errorStatus(w, r, http.StatusNotFound, "item_not_found")
		return
	}
	token, err := s.store.CreateShare(u.ID, id)
	if err != nil {
		s.logError(w, r, "crear compartición", err)
		return
	}
	url := Base + "/share/" + token
	writeJSON(w, http.StatusOK, map[string]any{"share": store.Share{Token: token, ItemID: id, URL: url}})
}

// deleteShare: deshabilita la URL pública de un artículo (DELETE /items/{id}/share).
func (s *Server) deleteShare(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("itemId"))
	if err != nil || id <= 0 {
		errorStatus(w, r, http.StatusNotFound, "item_not_found")
		return
	}
	if err := s.store.DeleteShare(user(r).ID, id); err != nil {
		s.logError(w, r, "borrar compartición", err)
		return
	}
	writeEmpty(w)
}

// shareRouter: ruta PÚBLICA (sin auth) de la vista de artículo compartido.
// La seguridad la aporta el token aleatorio; la vista es de solo lectura.
func (s *Server) shareRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /share/{token}", s.sharePage)
	return mux
}

// sharePage: HTML mínimo de solo lectura para una URL pública compartida.
func (s *Server) sharePage(w http.ResponseWriter, r *http.Request) {
	it, feed, err := s.store.ItemByShareToken(r.PathValue("token"))
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logError(w, r, "leer item compartido", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	body := s.rewriteImages(it.Body)
	media := ""
	if it.EnclosureLink != nil && it.EnclosureMime != nil {
		if len(*it.EnclosureMime) >= 6 && (*it.EnclosureMime)[:6] == "audio/" {
			media = `<audio src="` + html.EscapeString(*it.EnclosureLink) + `" controls preload="metadata" style="width:100%;margin:0 0 12px"></audio>`
		} else if len(*it.EnclosureMime) >= 6 && (*it.EnclosureMime)[:6] == "video/" {
			media = `<video src="` + html.EscapeString(*it.EnclosureLink) + `" controls preload="metadata" style="width:100%;border-radius:8px;margin:0 0 12px;max-height:60vh;background:#000"></video>`
		}
	}
	date := ""
	if it.PubDate > 0 {
		date = time.Unix(it.PubDate, 0).UTC().Format("2006-01-02 15:04")
	}
	author := html.EscapeString(it.Author)
	if author != "" {
		author = " · " + author
	}
	page := fmt.Sprintf(`<!DOCTYPE html>
<html lang="es"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="robots" content="noindex"><title>%s</title>
<style>body{font-family:system-ui,-apple-system,'Segoe UI',Roboto,sans-serif;margin:0;background:#f5f5f5;color:#1a1a1a}
main{max-width:760px;margin:0 auto;padding:32px 20px;background:#fff;min-height:100vh;box-sizing:border-box}
h1{font-size:24px;line-height:1.3;margin:0 0 8px}.meta{font-size:13px;opacity:.6;margin:0 0 20px}
img{max-width:100%%;height:auto;border-radius:8px;margin:.4em 0 1em}
a{color:#2563eb;text-decoration:underline}.news-body p{margin:0 0 .9em;line-height:1.65}
.news-body h1,.news-body h2,.news-body h3,.news-body h4{margin:1.2em 0 .5em;line-height:1.3}
.news-body ul,.news-body ol{margin:0 0 1em;padding-left:1.4em}.news-body blockquote{margin:0 0 1em;padding:.4em 1em;border-left:3px solid #d5d5d5;opacity:.9}
.news-body pre{background:#f0f0f0;border-radius:8px;padding:.8em 1em;overflow-x:auto}.news-body table{border-collapse:collapse;margin-bottom:1em}
.news-body th,.news-body td{border:1px solid #d5d5d5;padding:.4em .7em}
footer{margin-top:32px;font-size:12px;opacity:.5;border-top:1px solid #e5e5e5;padding-top:12px}</style></head>
<body><main><h1><a href="%s" rel="noopener noreferrer" style="color:inherit;text-decoration:none">%s</a></h1>
<p class="meta">%s%s%s</p>%s<div class="news-body">%s</div>
<footer>Shared from ocnews · <a href="%s" rel="noopener noreferrer">Open original</a></footer></main></body></html>`,
		html.EscapeString(it.Title),
		html.EscapeString(it.URL), html.EscapeString(it.Title),
		html.EscapeString(feed.Title), author, date, media, body, html.EscapeString(it.URL))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(page))
}
