package store

import (
	"database/sql"
	"errors"
	"regexp"
)

// AutoReadRule: regla que marca como leídos los items nuevos cuyo título casa
// con el patrón. FeedID 0 = se aplica a todos los feeds del usuario.
type AutoReadRule struct {
	ID           int64  `json:"id"`
	FeedID       int64  `json:"feedId"`
	TitlePattern string `json:"titlePattern"`
}

// ListAutoRead devuelve las reglas de auto-marcado del usuario.
func (s *Store) ListAutoRead(userID int64) ([]AutoReadRule, error) {
	rows, err := s.db.Query(
		`SELECT id, feed_id, title_pattern FROM auto_read WHERE user_id = ? ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []AutoReadRule{}
	for rows.Next() {
		var r AutoReadRule
		if err := rows.Scan(&r.ID, &r.FeedID, &r.TitlePattern); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// AddAutoRead crea una regla de auto-marcado.
func (s *Store) AddAutoRead(userID, feedID int64, pattern string) (*AutoReadRule, error) {
	if feedID != 0 {
		var one int
		err := s.db.QueryRow(`SELECT 1 FROM feeds WHERE user_id = ? AND id = ?`, userID, feedID).Scan(&one)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, err
		}
	}
	res, err := s.db.Exec(
		`INSERT INTO auto_read (user_id, feed_id, title_pattern) VALUES (?, ?, ?)`,
		userID, feedID, pattern)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	var r AutoReadRule
	err = s.db.QueryRow(
		`SELECT id, feed_id, title_pattern FROM auto_read WHERE user_id = ? AND id = ?`,
		userID, id).Scan(&r.ID, &r.FeedID, &r.TitlePattern)
	return &r, err
}

// DeleteAutoRead borra la regla (ErrNotFound si no existe o es de otro usuario).
func (s *Store) DeleteAutoRead(userID, ruleID int64) error {
	res, err := s.db.Exec(`DELETE FROM auto_read WHERE user_id = ? AND id = ?`, userID, ruleID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

type compiledAutoRead struct {
	feedID  int64
	pattern *regexp.Regexp
}

// autoReadForFeed compila las reglas del usuario que aplican al feed dado.
// Los patrones inválidos (no deberían existir: la API valida) se ignoran.
func (s *Store) autoReadForFeed(userID, feedID int64) ([]compiledAutoRead, error) {
	rows, err := s.db.Query(
		`SELECT feed_id, title_pattern FROM auto_read WHERE user_id = ? AND (feed_id = 0 OR feed_id = ?)`,
		userID, feedID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []compiledAutoRead{}
	for rows.Next() {
		var fid int64
		var pat string
		if err := rows.Scan(&fid, &pat); err != nil {
			return nil, err
		}
		if re, err := regexp.Compile(pat); err == nil {
			out = append(out, compiledAutoRead{feedID: fid, pattern: re})
		}
	}
	return out, rows.Err()
}
