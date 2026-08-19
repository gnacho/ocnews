package store

import (
	"database/sql"
	"errors"

	"github.com/gnacho/ocnews/backend/internal/rules"
)

// Rules: filtro block/keep en formato texto multilínea "Campo=regex".
type Rules struct {
	Block string `json:"block"`
	Keep  string `json:"keep"`
}

func (r Rules) HasRules() bool {
	return r.Block != "" || r.Keep != ""
}

// GetFeedRules devuelve las reglas del feed (vacías si no existen).
func (s *Store) GetFeedRules(feedID int64) (*Rules, error) {
	r := &Rules{}
	err := s.db.QueryRow(`SELECT block, keep FROM feed_rules WHERE feed_id = ?`, feedID).
		Scan(&r.Block, &r.Keep)
	if errors.Is(err, sql.ErrNoRows) {
		return r, nil
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

// SaveFeedRules inserta o actualiza las reglas del feed. Vacías = se borran.
func (s *Store) SaveFeedRules(feedID int64, r Rules) error {
	if !r.HasRules() {
		_, err := s.db.Exec(`DELETE FROM feed_rules WHERE feed_id = ?`, feedID)
		return err
	}
	_, err := s.db.Exec(
		`INSERT INTO feed_rules (feed_id, block, keep) VALUES (?, ?, ?)
		 ON CONFLICT(feed_id) DO UPDATE SET block = excluded.block, keep = excluded.keep`,
		feedID, r.Block, r.Keep)
	return err
}

func (s *Store) DeleteFeedRules(feedID int64) error {
	_, err := s.db.Exec(`DELETE FROM feed_rules WHERE feed_id = ?`, feedID)
	return err
}

// GetGlobalRules devuelve las reglas globales del usuario (vacías si no hay).
func (s *Store) GetGlobalRules(userID int64) (*Rules, error) {
	r := &Rules{}
	err := s.db.QueryRow(`SELECT block, keep FROM global_rules WHERE user_id = ?`, userID).
		Scan(&r.Block, &r.Keep)
	if errors.Is(err, sql.ErrNoRows) {
		return r, nil
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

// SaveGlobalRules guarda o elimina las reglas globales del usuario.
func (s *Store) SaveGlobalRules(userID int64, r Rules) error {
	if !r.HasRules() {
		_, err := s.db.Exec(`DELETE FROM global_rules WHERE user_id = ?`, userID)
		return err
	}
	_, err := s.db.Exec(
		`INSERT INTO global_rules (user_id, block, keep) VALUES (?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET block = excluded.block, keep = excluded.keep`,
		userID, r.Block, r.Keep)
	return err
}

// FeedUserID devuelve el propietario de un feed.
func (s *Store) FeedUserID(feedID int64) (int64, error) {
	var uid int64
	err := s.db.QueryRow(`SELECT user_id FROM feeds WHERE id = ?`, feedID).Scan(&uid)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	return uid, nil
}

// HasAnyFilter: true si el feed tiene keywords, reglas o el usuario reglas
// globales. Sirve para evitar re-evaluar items cuando no hay nada configurado.
func (s *Store) HasAnyFilter(userID, feedID int64) (bool, error) {
	var n int
	err := s.db.QueryRow(
		`SELECT (SELECT COUNT(*) FROM feed_filter WHERE feed_id = ? AND
		           (title_keywords <> '' OR body_keywords <> '' OR url_keywords <> ''))
		      + (SELECT COUNT(*) FROM feed_rules WHERE feed_id = ? AND (block <> '' OR keep <> ''))
		      + (SELECT COUNT(*) FROM global_rules WHERE user_id = ? AND (block <> '' OR keep <> ''))`,
		feedID, feedID, userID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// itemFilter compone keywords (News 28.4.0) + reglas regex globales y del
// feed. true = el item debe descartarse (filtered). Las reglas con regex
// inválida (imposible vía API, que valida) se ignoran por robustez.
func (s *Store) itemFilter(userID, feedID int64, kw *FeedFilter) (func(NewItem) bool, error) {
	gr, err := s.GetGlobalRules(userID)
	if err != nil {
		return nil, err
	}
	fr, err := s.GetFeedRules(feedID)
	if err != nil {
		return nil, err
	}
	gc, gerr := rules.Parse(gr.Block, gr.Keep)
	fc, ferr := rules.Parse(fr.Block, fr.Keep)
	return func(it NewItem) bool {
		if kw.HasFilter() && kw.Matches(it) {
			return true
		}
		f := rules.Fields{Title: it.Title, URL: it.URL, Author: it.Author, Content: plain(it.Body), PubDate: it.PubDate}
		if gerr == nil && gc.Ignore(f) {
			return true
		}
		if ferr == nil && fc.Ignore(f) {
			return true
		}
		return false
	}, nil
}
