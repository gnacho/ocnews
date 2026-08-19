package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
)

// Share: un artículo expuesto con URL pública (token aleatorio).
type Share struct {
	Token  string `json:"token"`
	ItemID int64  `json:"itemId"`
	URL    string `json:"url"`
}

func newShareToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateShare habilita (o devuelve la existente) la URL pública del item.
// Idempotente por (user, item).
func (s *Store) CreateShare(userID, itemID int64) (string, error) {
	var tok string
	err := s.db.QueryRow(`SELECT token FROM shared_items WHERE user_id = ? AND item_id = ?`, userID, itemID).Scan(&tok)
	if err == nil {
		return tok, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	for attempt := 0; attempt < 3; attempt++ {
		tok, err = newShareToken()
		if err != nil {
			return "", err
		}
		if _, err = s.db.Exec(
			`INSERT INTO shared_items (user_id, item_id, token) VALUES (?, ?, ?)`,
			userID, itemID, tok); err == nil {
			return tok, nil
		}
		// colisión de token: reintentar con otro aleatorio
	}
	return "", fmt.Errorf("no se pudo crear el token de compartición")
}

// DeleteShare deshabilita la URL pública del item.
func (s *Store) DeleteShare(userID, itemID int64) error {
	_, err := s.db.Exec(`DELETE FROM shared_items WHERE user_id = ? AND item_id = ?`, userID, itemID)
	return err
}

// ItemByShareToken resuelve el item compartido y su feed desde el token
// público. ErrNotFound si el token no existe.
func (s *Store) ItemByShareToken(token string) (*Item, *FeedRow, error) {
	q := `SELECT ` + itemsCols + `
		FROM shared_items sh
		JOIN items i ON i.id = sh.item_id
		JOIN feeds x ON x.id = i.feed_id
		WHERE sh.token = ?`
	it, err := scanItem(s.db.QueryRow(q, token))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, err
	}
	fr := &FeedRow{ID: it.FeedID, Title: it.Title}
	if err := s.db.QueryRow(`SELECT title FROM feeds WHERE id = ?`, it.FeedID).Scan(&fr.Title); err != nil {
		return nil, nil, err
	}
	return it, fr, nil
}
