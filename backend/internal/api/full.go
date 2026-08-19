package api

import (
	"net/http"
)

// itemFull: GET /items/{itemId}/full — cuerpo completo extraído del artículo
// original (readability), cacheado en BD. Los feeds traen resúmenes; este
// endpoint devuelve el texto íntegro sanitizado, con imágenes vía proxy.
func (s *Server) itemFull(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("itemId"))
	if err != nil || id <= 0 {
		errorStatus(w, r, http.StatusNotFound, "item_not_found")
		return
	}
	u := user(r)

	// cache: el contenido de un artículo publicado no cambia
	if body, err := s.store.GetItemFull(id); err == nil && body != "" {
		writeJSON(w, http.StatusOK, map[string]string{"body": s.rewriteImages(body)})
		return
	}

	itemURL, err := s.store.GetItemURL(u.ID, id)
	if err != nil {
		errorStatus(w, r, http.StatusNotFound, "item_not_found")
		return
	}
	// selector CSS de extracción por feed (#39): si el feed lo define, se usa
	// antes que readability (fallback si el selector falla).
	selector := ""
	if feedID, err := s.store.GetItemFeedID(u.ID, id); err == nil {
		if sel, err := s.store.GetFeedScraperSelector(feedID); err == nil {
			selector = sel
		}
	}
	body, err := s.extract.Article(r.Context(), itemURL, selector)
	if err != nil {
		// no es error del servidor: el sitio no permite extracción
		s.log.Warn("extracción completa falló", "url", itemURL, "err", err)
		errorStatus(w, r, http.StatusUnprocessableEntity, "full_unavailable")
		return
	}
	_ = s.store.SaveItemFull(id, body)
	writeJSON(w, http.StatusOK, map[string]string{"body": s.rewriteImages(body)})
}
