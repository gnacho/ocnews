package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// itemsCols: columnas del item + full_content del feed (en el orden exacto
// que espera scanItem).
const itemsCols = "i.id, i.guid, i.guid_hash, i.url, i.title, i.author, i.pub_date, i.body, " +
	"i.enclosure_mime, i.enclosure_link, i.media_thumbnail, i.media_description, " +
	"i.feed_id, i.unread, i.starred, i.last_modified, i.fingerprint, i.filtered, " +
	"COALESCE(x.full_content, 0)"

// buildItemQuery construye el WHERE según el filtro (type 0=feed, 1=folder,
// 2=starred, 3=all) + getRead/offset/updatedSince, y el ORDER/LIMIT.
func buildItemQuery(f ItemFilter, countOnly bool) (string, []any, error) {
	var where []string
	var args []any
	where = append(where, "i.user_id = ?")
	args = append(args, f.UserID)

	switch f.Type {
	case 0: // feed
		where = append(where, "i.feed_id = ?")
		args = append(args, f.ID)
	case 1: // folder
		where = append(where, "i.feed_id IN (SELECT id FROM feeds WHERE user_id = ? AND folder_id = ?)")
		args = append(args, f.UserID, f.ID)
	case 2: // starred
		where = append(where, "i.starred = 1")
	case 3: // all
	default:
		return "", nil, fmt.Errorf("type inválido: %d", f.Type)
	}

	if !f.GetRead {
		where = append(where, "i.unread = 1")
	}
	if f.OffsetID > 0 {
		where = append(where, "i.id <= ?")
		args = append(args, f.OffsetID)
	}
	if f.UpdatedSince > 0 {
		where = append(where, "i.last_modified >= ?")
		args = append(args, f.UpdatedSince)
	}
	if !f.IncludeFiltered {
		where = append(where, "i.filtered = 0")
	}

	sel := "SELECT " + itemsCols + " FROM items i LEFT JOIN feeds x ON x.id = i.feed_id"
	if countOnly {
		sel = "SELECT COUNT(*) FROM items i"
	}
	q := sel + " WHERE " + strings.Join(where, " AND ")

	if !countOnly {
		if f.OldestFirst {
			q += " ORDER BY i.id ASC"
		} else {
			q += " ORDER BY i.id DESC"
		}
		if f.BatchSize >= 0 {
			q += fmt.Sprintf(" LIMIT %d", f.BatchSize)
		}
	}
	return q, args, nil
}

func scanItem(row interface{ Scan(...any) error }) (*Item, error) {
	it := &Item{}
	var unread, starred, full, filtered int
	err := row.Scan(&it.ID, &it.GUID, &it.GUIDHash, &it.URL, &it.Title, &it.Author, &it.PubDate,
		&it.Body, &it.EnclosureMime, &it.EnclosureLink, &it.MediaThumbnail, &it.MediaDescription,
		&it.FeedID, &unread, &starred, &it.LastModified, &it.Fingerprint, &filtered, &full)
	if err != nil {
		return nil, err
	}
	it.Unread = unread != 0
	it.Starred = starred != 0
	it.Filtered = filtered != 0
	it.FeedFullContent = full != 0
	return it, nil
}

// ListItems aplica el filtro y devuelve los items.
func (s *Store) ListItems(f ItemFilter) ([]Item, error) {
	q, args, err := buildItemQuery(f, false)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Item{}
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *it)
	}
	return out, rows.Err()
}

// CountItems cuenta los items que casan con el filtro (ignora paginación).
func (s *Store) CountItems(f ItemFilter) (int64, error) {
	f.BatchSize = -1
	q, args, err := buildItemQuery(f, true)
	if err != nil {
		return 0, err
	}
	var n int64
	if err := s.db.QueryRow(q, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// SearchItems busca artículos del usuario cuyo título, cuerpo o URL contenga
// la query (case-insensitive). Reutiliza el mismo scoping de ItemFilter
// (type/id) y excluye los filtered por defecto.
func (s *Store) SearchItems(f ItemFilter, query string, limit int) ([]Item, error) {
	like := "%" + strings.ToLower(query) + "%"
	var where []string
	var args []any
	where = append(where, "i.user_id = ?")
	args = append(args, f.UserID)

	switch f.Type {
	case 0: // feed
		where = append(where, "i.feed_id = ?")
		args = append(args, f.ID)
	case 1: // folder
		where = append(where, "i.feed_id IN (SELECT id FROM feeds WHERE user_id = ? AND folder_id = ?)")
		args = append(args, f.UserID, f.ID)
	case 2: // starred
		where = append(where, "i.starred = 1")
	case 3: // all
	default:
		return nil, fmt.Errorf("type inválido: %d", f.Type)
	}
	if !f.GetRead {
		where = append(where, "i.unread = 1")
	}
	if !f.IncludeFiltered {
		where = append(where, "i.filtered = 0")
	}
	where = append(where,
		"(LOWER(i.title) LIKE ? OR LOWER(i.body) LIKE ? OR LOWER(i.url) LIKE ?)")
	args = append(args, like, like, like)

	q := "SELECT " + itemsCols + " FROM items i LEFT JOIN feeds x ON x.id = i.feed_id WHERE " +
		strings.Join(where, " AND ")
	if f.OldestFirst {
		q += " ORDER BY i.id ASC"
	} else {
		q += " ORDER BY i.id DESC"
	}
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []Item{}
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *it)
	}
	return out, rows.Err()
}

// NewestItemID devuelve MAX(id) del usuario (0 si no hay items).
func (s *Store) NewestItemID(userID int64) (int64, error) {
	var n sql.NullInt64
	if err := s.db.QueryRow(`SELECT MAX(id) FROM items WHERE user_id = ?`, userID).Scan(&n); err != nil {
		return 0, err
	}
	return n.Int64, nil
}

// StarredCount cuenta los items destacados del usuario.
func (s *Store) StarredCount(userID int64) (int64, error) {
	var n int64
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM items WHERE user_id = ? AND starred = 1`, userID).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// markItems actualiza una columna booleana (unread|starred) para una lista de
// ids del usuario, tocando last_modified (requisito de /items/updated).
// isStarStamp: para star/unstar, unread=0 implícito no aplica.
func (s *Store) markItems(userID int64, ids []int64, col string, val bool) error {
	if len(ids) == 0 {
		return nil
	}
	v := 0
	if val {
		v = 1
	}
	if col != "unread" && col != "starred" {
		return fmt.Errorf("columna inválida: %s", col)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("UPDATE items SET %s = ?, last_modified = ? WHERE user_id = ? AND id IN (", col))
	args := []any{v, now(), userID}
	for i, id := range ids {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("?")
		args = append(args, id)
	}
	sb.WriteString(")")
	_, err := s.db.Exec(sb.String(), args...)
	return err
}

// MarkItemsUnreadFlag fija unread=true/false para los ids dados.
func (s *Store) MarkItemsUnreadFlag(userID int64, ids []int64, unread bool) error {
	return s.markItems(userID, ids, "unread", unread)
}

// MarkItemsStarFlag fija starred=true/false para los ids dados.
func (s *Store) MarkItemsStarFlag(userID int64, ids []int64, starred bool) error {
	return s.markItems(userID, ids, "starred", starred)
}

// ItemExists comprueba que el item pertenece al usuario.
func (s *Store) ItemExists(userID, itemID int64) (bool, error) {
	var one int
	err := s.db.QueryRow(`SELECT 1 FROM items WHERE user_id = ? AND id = ?`, userID, itemID).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// MarkAllRead marca como leídos todos los items del usuario con id <= maxID.
// maxID = 0 → sin tope (todo). Devuelve los afectados.
func (s *Store) MarkAllRead(userID int64, maxID int64, scope string, scopeID int64) (int64, error) {
	q := `UPDATE items SET unread = 0, last_modified = ? WHERE user_id = ? AND unread = 1`
	args := []any{now(), userID}
	if maxID > 0 {
		q += ` AND id <= ?`
		args = append(args, maxID)
	}
	switch scope {
	case "all":
	case "feed":
		q += ` AND feed_id = ?`
		args = append(args, scopeID)
	case "folder":
		q += ` AND feed_id IN (SELECT id FROM feeds WHERE user_id = ? AND folder_id = ?)`
		args = append(args, userID, scopeID)
	default:
		return 0, fmt.Errorf("scope inválido: %s", scope)
	}
	res, err := s.db.Exec(q, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// PurgeOldItems borra items leídos no destacados con last_modified anterior
// al umbral (retención global), EXCEPTO los feeds que tienen override propio
// (esos se gestionan en PurgeOldItemsByFeed). Devuelve los borrados.
func (s *Store) PurgeOldItems(olderThan int64) (int64, error) {
	res, err := s.db.Exec(
		`DELETE FROM items WHERE unread = 0 AND starred = 0 AND last_modified < ? AND
		 feed_id NOT IN (SELECT id FROM feeds WHERE retention_days > 0)`, olderThan)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// PurgeOldItemsByFeed borra items leídos no destacados de un feed concreto con
// last_modified anterior al umbral (retención por feed). Devuelve los borrados.
func (s *Store) PurgeOldItemsByFeed(feedID, olderThan int64) (int64, error) {
	res, err := s.db.Exec(
		`DELETE FROM items WHERE feed_id = ? AND unread = 0 AND starred = 0 AND last_modified < ?`,
		feedID, olderThan)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// SetFeedRetentionDays fija el override de retención del feed (0 = usar la global).
func (s *Store) SetFeedRetentionDays(feedID, userID, days int64) error {
	res, err := s.db.Exec(`UPDATE feeds SET retention_days = ? WHERE id = ? AND user_id = ?`,
		days, feedID, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// FeedsWithRetentionOverride devuelve los ids de feeds con retención propia.
func (s *Store) FeedsWithRetentionOverride() ([]struct {
	ID   int64
	Days int64
}, error) {
	rows, err := s.db.Query(`SELECT id, retention_days FROM feeds WHERE retention_days > 0`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []struct {
		ID   int64
		Days int64
	}{}
	for rows.Next() {
		var r struct {
			ID   int64
			Days int64
		}
		if err := rows.Scan(&r.ID, &r.Days); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetItemFull devuelve el cuerpo completo cacheado ("" si no hay).
func (s *Store) GetItemFull(itemID int64) (string, error) {
	var body string
	err := s.db.QueryRow(`SELECT body FROM item_full WHERE item_id = ?`, itemID).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return body, nil
}

// SaveItemFull guarda/actualiza el cuerpo completo extraído.
func (s *Store) SaveItemFull(itemID int64, body string) error {
	_, err := s.db.Exec(
		`INSERT INTO item_full (item_id, body, fetched_at) VALUES (?, ?, ?)
		 ON CONFLICT(item_id) DO UPDATE SET body = excluded.body, fetched_at = excluded.fetched_at`,
		itemID, body, now())
	return err
}

// GetItemURL devuelve la URL original del item (para extraer el completo).
func (s *Store) GetItemURL(userID, itemID int64) (string, error) {
	var u string
	err := s.db.QueryRow(`SELECT url FROM items WHERE user_id = ? AND id = ?`, userID, itemID).Scan(&u)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return u, err
}

// SetItemURLForTesting reescribe la URL de un item (solo tests de extracción).
func (s *Store) SetItemURLForTesting(itemID int64, url string) error {
	_, err := s.db.Exec(`UPDATE items SET url = ? WHERE id = ?`, url, itemID)
	return err
}
