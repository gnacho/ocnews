package store

import (
	"database/sql"
	"fmt"
	"strings"
)

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

	cols := "i.id, i.guid, i.guid_hash, i.url, i.title, i.author, i.pub_date, i.body, " +
		"i.enclosure_mime, i.enclosure_link, i.media_thumbnail, i.media_description, " +
		"i.feed_id, i.unread, i.starred, i.last_modified, i.fingerprint"
	sel := "SELECT " + cols + " FROM items i"
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
	var unread, starred int
	err := row.Scan(&it.ID, &it.GUID, &it.GUIDHash, &it.URL, &it.Title, &it.Author, &it.PubDate,
		&it.Body, &it.EnclosureMime, &it.EnclosureLink, &it.MediaThumbnail, &it.MediaDescription,
		&it.FeedID, &unread, &starred, &it.LastModified, &it.Fingerprint)
	if err != nil {
		return nil, err
	}
	it.Unread = unread != 0
	it.Starred = starred != 0
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
