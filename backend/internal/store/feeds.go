package store

import (
	"database/sql"
	"errors"
)

const feedCols = `f.id, f.url, f.title, f.favicon, f.added, f.next_update, f.folder_id,
	COALESCE(c.unread, 0), f.ordering, f.link, f.pinned, f.update_error_count,
	NULLIF(f.last_update_error, '')`

const feedSelect = `SELECT ` + feedCols + `
	FROM feeds f
	LEFT JOIN (SELECT feed_id, COUNT(*) AS unread FROM items WHERE unread = 1 GROUP BY feed_id) c
		ON c.feed_id = f.id`

func scanFeed(row interface{ Scan(...any) error }) (*Feed, error) {
	f := &Feed{}
	var lastErr *string
	var pinned int
	err := row.Scan(&f.ID, &f.URL, &f.Title, &f.FaviconLink, &f.Added, &f.NextUpdateTime,
		&f.FolderID, &f.UnreadCount, &f.Ordering, &f.Link, &pinned,
		&f.UpdateErrorCount, &lastErr)
	if err != nil {
		return nil, err
	}
	f.LastUpdateError = lastErr
	f.Pinned = pinned != 0
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
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`INSERT INTO feeds (user_id, folder_id, url, link, title, favicon, added, next_update)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, folderID, url, link, title, favicon, now(), now()+600)
	if err != nil {
		return nil, ErrConflict
	}
	feedID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	var newest int64
	for _, it := range items {
		r, err := tx.Exec(itemInsertSQL(), feedID, userID, it.GUID, it.GUIDHash, it.URL, it.Title,
			it.Author, it.PubDate, it.Body, it.EnclosureMime, it.EnclosureLink,
			it.MediaThumbnail, it.MediaDescription, it.Fingerprint, now())
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
// (INSERT OR IGNORE por guid_hash) y devuelve el feed actualizado.
// Los items desaparecidos del feed NO se borran (igual que News).
func (s *Store) ReplaceFeedItems(feedID, userID int64, title, link string, items []NewItem) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE feeds SET title = ?, link = ?, next_update = ?, update_error_count = 0, last_update_error = '' WHERE id = ?`,
		title, link, now()+600, feedID); err != nil {
		return err
	}
	for _, it := range items {
		if _, err := tx.Exec(itemInsertSQL(), feedID, userID, it.GUID, it.GUIDHash, it.URL, it.Title,
			it.Author, it.PubDate, it.Body, it.EnclosureMime, it.EnclosureLink,
			it.MediaThumbnail, it.MediaDescription, it.Fingerprint, now()); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) RecordFeedError(feedID int64, msg string) {
	s.db.Exec(`UPDATE feeds SET update_error_count = update_error_count + 1, last_update_error = ? WHERE id = ?`, msg, feedID)
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
		 enclosure_mime, enclosure_link, media_thumbnail, media_description, fingerprint, last_modified)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
}
