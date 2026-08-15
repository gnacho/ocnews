package api

import (
	"net/http"
	"strconv"

	"github.com/gnacho/ocnews/backend/internal/store"
)

// parseID convierte un id en string a int64 (>0).
func parseID(s string) (int64, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// boolParam lee un booleano de query (true|1).
func boolParam(r *http.Request, key string, def bool) bool {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	return v == "true" || v == "1"
}

// newestItemID lee {"newestItemId": N} del cuerpo (0 = sin tope).
func newestItemID(r *http.Request, w http.ResponseWriter) (int64, bool) {
	var body struct {
		NewestItemID int64 `json:"newestItemId"`
	}
	if err := decodeBody(r, &body); err != nil {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_body")
		return 0, false
	}
	return body.NewestItemID, true
}

// itemFilterFromRequest construye el filtro común de /items y /items/updated.
func itemFilterFromRequest(r *http.Request, u *store.User, updated bool) (store.ItemFilter, bool) {
	q := r.URL.Query()
	f := store.ItemFilter{UserID: u.ID}

	if v, err := parseID(q.Get("type")); err == nil {
		f.Type = int(v)
	} else {
		f.Type = 3
	}
	if f.Type < 0 || f.Type > 3 {
		return f, false
	}
	if f.Type != 2 && f.Type != 3 {
		if v, err := parseID(q.Get("id")); err == nil {
			f.ID = v
		}
	}
	if updated {
		if v, err := parseID(q.Get("lastModified")); err == nil {
			f.UpdatedSince = v
		}
		f.GetRead = true // updated informa cambios de estado, leídos incluidos
		f.BatchSize = -1 // la spec no define batchSize en /items/updated
	} else {
		f.GetRead = boolParam(r, "getRead", true)
		if v, err := parseID(q.Get("batchSize")); err == nil {
			f.BatchSize = v
		} else {
			f.BatchSize = -1
		}
		if v, err := parseID(q.Get("offset")); err == nil {
			f.OffsetID = v
		}
		f.OldestFirst = boolParam(r, "oldestFirst", false)
	}
	return f, true
}

func (s *Server) listItems(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	f, valid := itemFilterFromRequest(r, u, false)
	if !valid {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_type")
		return
	}
	items, err := s.store.ListItems(f)
	if err != nil {
		s.logError(w, r, "listar items", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) updatedItems(w http.ResponseWriter, r *http.Request) {
	u := user(r)
	f, valid := itemFilterFromRequest(r, u, true)
	if !valid {
		errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_type")
		return
	}
	items, err := s.store.ListItems(f)
	if err != nil {
		s.logError(w, r, "items actualizados", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// markItem devuelve un handler para marcar un item individual.
// val=true → unread/starred según col; false → lo quita.
func (s *Server) markItem(val bool, col string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		var mErr error
		if col == "unread" {
			mErr = s.store.MarkItemsUnreadFlag(u.ID, []int64{id}, val)
		} else {
			mErr = s.store.MarkItemsStarFlag(u.ID, []int64{id}, val)
		}
		if mErr != nil {
			s.logError(w, r, "marcar item", mErr)
			return
		}
		writeEmpty(w)
	}
}

// multipleBody acepta "itemIds" (definiciones de la spec) y "items"
// (sección How To Sync): los clientes reales mandan ambas formas.
type multipleBody struct {
	ItemIDs []int64 `json:"itemIds"`
	Items   []int64 `json:"items"`
}

func (b multipleBody) ids() []int64 {
	return append(b.ItemIDs, b.Items...)
}

func (s *Server) markMultiple(val bool, col string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body multipleBody
		if err := decodeBody(r, &body); err != nil {
			errorStatus(w, r, http.StatusUnprocessableEntity, "invalid_body")
			return
		}
		ids := body.ids()
		if len(ids) == 0 {
			writeEmpty(w)
			return
		}
		var err error
		if col == "unread" {
			err = s.store.MarkItemsUnreadFlag(user(r).ID, ids, val)
		} else {
			err = s.store.MarkItemsStarFlag(user(r).ID, ids, val)
		}
		if err != nil {
			s.logError(w, r, "marcar items", err)
			return
		}
		writeEmpty(w)
	}
}

// markAllRead marca todos los items del usuario hasta newestItemId.
func (s *Server) markAllRead(w http.ResponseWriter, r *http.Request) {
	maxID, ok := newestItemID(r, w)
	if !ok {
		return
	}
	if _, err := s.store.MarkAllRead(user(r).ID, maxID, "all", 0); err != nil {
		s.logError(w, r, "marcar todo leído", err)
		return
	}
	writeEmpty(w)
}
