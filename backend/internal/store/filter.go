package store

import (
	"database/sql"
	"strings"
)

// GetFeedFilter devuelve el filtro de keywords del feed (vacío si no existe).
func (s *Store) GetFeedFilter(feedID int64) (*FeedFilter, error) {
	f := &FeedFilter{FeedID: feedID}
	err := s.db.QueryRow(
		`SELECT title_keywords, body_keywords, url_keywords FROM feed_filter WHERE feed_id = ?`,
		feedID).Scan(&f.TitleKeywords, &f.BodyKeywords, &f.URLKeywords)
	if err == sql.ErrNoRows {
		return f, nil
	}
	if err != nil {
		return nil, err
	}
	return f, nil
}

// SaveFeedFilter inserta o actualiza el filtro del feed. Si no tiene keywords,
// lo borra (comportamiento News: filtro vacío = sin filtro).
func (s *Store) SaveFeedFilter(f FeedFilter) error {
	if !f.HasFilter() {
		_, err := s.db.Exec(`DELETE FROM feed_filter WHERE feed_id = ?`, f.FeedID)
		return err
	}
	_, err := s.db.Exec(
		`INSERT INTO feed_filter (feed_id, title_keywords, body_keywords, url_keywords)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(feed_id) DO UPDATE SET
		   title_keywords = excluded.title_keywords,
		   body_keywords  = excluded.body_keywords,
		   url_keywords   = excluded.url_keywords`,
		f.FeedID, f.TitleKeywords, f.BodyKeywords, f.URLKeywords)
	return err
}

// DeleteFeedFilter elimina el filtro del feed.
func (s *Store) DeleteFeedFilter(feedID int64) error {
	_, err := s.db.Exec(`DELETE FROM feed_filter WHERE feed_id = ?`, feedID)
	return err
}

// ReapplyFeedFilter re-evalúa el filtro sobre los items ya guardados del feed:
// fija filtered=1 para los que casen y filtered=0 para los que no. Con un
// filtro vacío, limpia todos los filtered del feed (descongelar). Devuelve
// el número de items marcados como filtered.
func (s *Store) ReapplyFeedFilter(feedID int64, f FeedFilter) (int64, error) {
	if !f.HasFilter() {
		_, err := s.db.Exec(`UPDATE items SET filtered = 0 WHERE feed_id = ?`, feedID)
		if err != nil {
			return 0, err
		}
		return 0, nil // "marcados" aquí es 0 porque estamos descongelando
	}
	rows, err := s.db.Query(
		`SELECT id, url, title, body FROM items WHERE feed_id = ?`, feedID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	type row struct {
		id int64
		ni NewItem
	}
	var pending []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.ni.URL, &r.ni.Title, &r.ni.Body); err != nil {
			return 0, err
		}
		pending = append(pending, r)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	var marked int64
	for _, r := range pending {
		val := 0
		if f.Matches(r.ni) {
			val = 1
		}
		if _, err := s.db.Exec(`UPDATE items SET filtered = ? WHERE id = ?`, val, r.id); err != nil {
			return 0, err
		}
		if val == 1 {
			marked++
		}
	}
	return marked, nil
}

// ApplyFilterToItems marca los items que casan con el filtro del feed, en
// memoria. Debe llamarse ANTES de abrir una transacción (el Store usa una
// única conexión: consultar el filtro dentro de una tx causaría deadlock).
func (s *Store) ApplyFilterToItems(feedID int64, items []NewItem) error {
	f, err := s.GetFeedFilter(feedID)
	if err != nil {
		return err
	}
	if !f.HasFilter() {
		return nil
	}
	for i := range items {
		items[i].Filtered = f.Matches(items[i])
	}
	return nil
}

// containsAnyFold: true si text contiene alguna keyword de la lista CSV (fold).
func containsAnyFold(text, csv string) bool {
	tl := strings.ToLower(text)
	for _, kw := range strings.Split(csv, ",") {
		kw = strings.TrimSpace(kw)
		if kw != "" && strings.Contains(tl, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// plain quita las etiquetas HTML del cuerpo para matchear sobre texto plano.
func plain(html string) string {
	var b strings.Builder
	inTag := false
	for _, r := range html {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return b.String()
}
