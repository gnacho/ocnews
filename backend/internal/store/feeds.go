package store

import (
	"crypto/md5"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const feedCols = `f.id, f.url, f.title, f.favicon, f.added, f.next_update, f.folder_id,
	COALESCE(c.unread, 0), f.ordering, f.link, f.pinned, f.update_error_count,
	NULLIF(f.last_update_error, ''), f.no_new_streak, f.user_id, f.url_hash, f.full_content, f.retention_days, f.is_podcast`

const feedSelect = `SELECT ` + feedCols + `
	FROM feeds f
	LEFT JOIN (SELECT feed_id, COUNT(*) AS unread FROM items WHERE unread = 1 GROUP BY feed_id) c
		ON c.feed_id = f.id`

func scanFeed(row interface{ Scan(...any) error }) (*Feed, error) {
	f := &Feed{}
	var lastErr *string
	var pinned, full, retention, podcast int
	err := row.Scan(&f.ID, &f.URL, &f.Title, &f.FaviconLink, &f.Added, &f.NextUpdateTime,
		&f.FolderID, &f.UnreadCount, &f.Ordering, &f.Link, &pinned,
		&f.UpdateErrorCount, &lastErr, &f.NoNewStreak, &f.UserID, &f.URLHash, &full, &retention, &podcast)
	if err != nil {
		return nil, err
	}
	f.LastUpdateError = lastErr
	f.Pinned = pinned != 0
	f.FullContent = full != 0
	f.RetentionDays = retention
	f.IsPodcast = podcast != 0
	return f, nil
}

// ListFeeds devuelve todos los feeds del usuario con su unreadCount.
func (s *Store) ListFeeds(userID int64) ([]Feed, error) {
	rows, err := s.db.Query(feedSelect+` WHERE f.user_id = ? ORDER BY f.title`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Feed{}
	for rows.Next() {
		f, err := scanFeed(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

func (s *Store) GetFeed(userID, feedID int64) (*Feed, error) {
	f, err := scanFeed(s.db.QueryRow(feedSelect+` WHERE f.user_id = ? AND f.id = ?`, userID, feedID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return f, nil
}

// FeedExistsByURL comprueba si el usuario ya sigue esa URL.
func (s *Store) FeedExistsByURL(userID int64, url string) (bool, error) {
	var one int
	err := s.db.QueryRow(`SELECT 1 FROM feeds WHERE user_id = ? AND url = ?`, userID, url).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// CreateFeed inserta el feed (ya validado y fetcheado) y sus items.
// Devuelve ErrConflict si la URL ya existe.
func (s *Store) CreateFeed(userID int64, url string, folderID *int64, title, link, favicon string, items []NewItem) (*Feed, error) {
	return s.createFeed(userID, url, folderID, title, link, favicon, items, false)
}

// CreateFeedFull como CreateFeed fijando la detección de contenido completo.
func (s *Store) CreateFeedFull(userID int64, url string, folderID *int64, title, link, favicon string, items []NewItem, fullContent bool) (*Feed, error) {
	return s.createFeed(userID, url, folderID, title, link, favicon, items, fullContent)
}

func (s *Store) createFeed(userID int64, url string, folderID *int64, title, link, favicon string, items []NewItem, fullContent bool) (*Feed, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`INSERT INTO feeds (user_id, folder_id, url, url_hash, link, title, favicon, added, next_update, full_content, is_podcast)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, folderID, url, md5Hex(url), link, title, favicon, now(), now()+600,
		boolInt(fullContent), boolInt(hasMediaEnclosure(items)))
	if err != nil {
		return nil, ErrConflict
	}
	feedID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	// Un feed recién creado no tiene filtro (tabla feed_filter vacía para él),
	// así que no hace falta aplicar keywords aquí.
	var newest int64
	for _, it := range items {
		r, err := tx.Exec(itemInsertSQL(), feedID, userID, it.GUID, it.GUIDHash, it.URL, it.Title,
			it.Author, it.PubDate, it.Body, it.EnclosureMime, it.EnclosureLink,
			it.MediaThumbnail, it.MediaDescription, it.Fingerprint, now(), boolInt(it.Filtered))
		if err != nil {
			return nil, err
		}
		if id, _ := r.LastInsertId(); id > newest {
			newest = id
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetFeed(userID, feedID)
}

// ReplaceFeedItems: refresco completo de un feed. Inserta items nuevos
// (INSERT OR IGNORE por guid_hash) y devuelve cuántos eran nuevos;
// no_new_streak alimenta el intervalo adaptativo del scheduler.
// fullContent: si esta tanda ya trae artículos completos (sticky: MAX).
// Los items desaparecidos del feed NO se borran (igual que News).
func (s *Store) ReplaceFeedItems(feedID, userID int64, title, link string, items []NewItem, fullContent bool) (int64, error) {
	// Aplicar el filtro ANTES de abrir la tx: el Store usa una única conexión y
	// consultar el filtro dentro de la tx causaría deadlock (MaxOpenConns=1).
	if err := s.ApplyFilterToItems(feedID, items); err != nil {
		return 0, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE feeds SET title = ?, link = ?, update_error_count = 0, last_update_error = '' WHERE id = ?`,
		title, link, feedID); err != nil {
		return 0, err
	}
	var inserted int64
	for _, it := range items {
		r, err := tx.Exec(itemInsertSQL(), feedID, userID, it.GUID, it.GUIDHash, it.URL, it.Title,
			it.Author, it.PubDate, it.Body, it.EnclosureMime, it.EnclosureLink,
			it.MediaThumbnail, it.MediaDescription, it.Fingerprint, now(), boolInt(it.Filtered))
		if err != nil {
			return 0, err
		}
		if n, _ := r.RowsAffected(); n > 0 {
			inserted += n
		}
	}
	streak := 0
	if inserted == 0 {
		streak = 1 // señal: sumar 1 a la racha existente (ver UPDATE)
	}
	if _, err := tx.Exec(
		`UPDATE feeds SET no_new_streak = CASE WHEN ? = 1 THEN no_new_streak + 1 ELSE 0 END,
			full_content = MAX(full_content, ?), is_podcast = MAX(is_podcast, ?) WHERE id = ?`,
		streak, boolInt(fullContent), boolInt(hasMediaEnclosure(items)), feedID); err != nil {
		return 0, err
	}
	return inserted, tx.Commit()
}

// hasMediaEnclosure: true si algún item de la tanda trae un enclosure de
// audio o vídeo (indica que el feed es un podcast).
func hasMediaEnclosure(items []NewItem) bool {
	for _, it := range items {
		if it.EnclosureMime != nil {
			m := *it.EnclosureMime
			if strings.HasPrefix(m, "audio/") || strings.HasPrefix(m, "video/") {
				return true
			}
		}
	}
	return false
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *Store) RecordFeedError(feedID int64, msg string) {
	s.db.Exec(`UPDATE feeds SET update_error_count = update_error_count + 1, last_update_error = ? WHERE id = ?`, msg, feedID)
}

// ListDueFeeds devuelve los feeds con next_update vencido (para el scheduler).
func (s *Store) ListDueFeeds(now int64, limit int) ([]Feed, error) {
	q := feedSelect + ` WHERE f.next_update <= ? ORDER BY f.next_update`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := s.db.Query(q, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Feed{}
	for rows.Next() {
		f, err := scanFeed(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

// SetNextUpdate fija la próxima hora de refresco del feed.
func (s *Store) SetNextUpdate(feedID, nextUpdate int64) error {
	_, err := s.db.Exec(`UPDATE feeds SET next_update = ? WHERE id = ?`, nextUpdate, feedID)
	return err
}

// FeedRow: feed mínimo para el refresher (sin contadores agregados).
type FeedRow struct {
	ID     int64
	UserID int64
	URL    string
	Link   string
	Title  string
}

// FeedByURLHash busca un feed por el hash md5 de su URL (endpoint /favicon).
func (s *Store) FeedByURLHash(hash string) (*FeedRow, error) {
	row := &FeedRow{}
	err := s.db.QueryRow(`SELECT id, user_id, url, link, title FROM feeds WHERE url_hash = ? LIMIT 1`, hash).
		Scan(&row.ID, &row.UserID, &row.URL, &row.Link, &row.Title)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return row, nil
}

// CreateFeedDeferred inserta un feed sin fetch inicial (import OPML):
// título provisional = URL y next_update=0 para que el scheduler lo
// refresque en el próximo ciclo.
func (s *Store) CreateFeedDeferred(userID int64, url string, folderID *int64) (*Feed, error) {
	res, err := s.db.Exec(
		`INSERT INTO feeds (user_id, folder_id, url, url_hash, link, title, favicon, added, next_update)
		 VALUES (?, ?, ?, ?, '', ?, '', ?, 0)`,
		userID, folderID, url, md5Hex(url), url, now())
	if err != nil {
		return nil, ErrConflict
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetFeed(userID, id)
}

// UpdaterFeed: fila para la ruta admin /feeds/all de la spec.
type UpdaterFeed struct {
	ID     int64  `json:"id"`
	UserID string `json:"userId"`
}

func (s *Store) AllFeedsWithUser() ([]UpdaterFeed, error) {
	rows, err := s.db.Query(
		`SELECT f.id, u.username FROM feeds f JOIN users u ON u.id = f.user_id ORDER BY f.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UpdaterFeed{}
	for rows.Next() {
		var f UpdaterFeed
		if err := rows.Scan(&f.ID, &f.UserID); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Store) DeleteFeed(userID, feedID int64) error {
	res, err := s.db.Exec(`DELETE FROM feeds WHERE user_id = ? AND id = ?`, userID, feedID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// MoveFeed mueve el feed a otra carpeta (folderID nil = raíz).
func (s *Store) MoveFeed(userID, feedID int64, folderID *int64) error {
	if folderID != nil {
		var one int
		err := s.db.QueryRow(`SELECT 1 FROM folders WHERE user_id = ? AND id = ?`, userID, *folderID).Scan(&one)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
	}
	res, err := s.db.Exec(`UPDATE feeds SET folder_id = ? WHERE user_id = ? AND id = ?`, folderID, userID, feedID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) RenameFeed(userID, feedID int64, title string) error {
	res, err := s.db.Exec(`UPDATE feeds SET title = ? WHERE user_id = ? AND id = ?`, title, userID, feedID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func itemInsertSQL() string {
	return `INSERT OR IGNORE INTO items
		(feed_id, user_id, guid, guid_hash, url, title, author, pub_date, body,
		 enclosure_mime, enclosure_link, media_thumbnail, media_description, fingerprint, last_modified, filtered)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
}

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", sum)
}
