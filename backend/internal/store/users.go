package store

import (
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")

func (s *Store) CreateUser(username, passwordHash, displayName, role string) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO users (username, password_hash, display_name, role, language, created_at) VALUES (?, ?, ?, ?, 'auto', ?)`,
		username, passwordHash, displayName, role, now())
	if err != nil {
		return 0, fmt.Errorf("crear usuario: %w", err)
	}
	return res.LastInsertId()
}

// SetUserLanguage fija el idioma preferido del usuario (auto|es|en).
func (s *Store) SetUserLanguage(userID int64, language string) error {
	_, err := s.db.Exec(`UPDATE users SET language = ? WHERE id = ?`, language, userID)
	return err
}

// BootstrapUser crea el primer usuario admin si la tabla está vacía.
// Devuelve true si se creó.
func (s *Store) BootstrapUser(username, passwordHash string) (bool, error) {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}
	_, err := s.CreateUser(username, passwordHash, username, "admin")
	return err == nil, err
}

func (s *Store) GetUserByUsername(username string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		`SELECT id, username, password_hash, display_name, role, language, created_at, last_login_at
		 FROM users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Role, &u.Language, &u.CreatedAt, &u.LastLoginAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

// BootstrapCount devuelve el número de usuarios (para validar arranque sin credenciales).
func (s *Store) BootstrapCount(count *int) error {
	return s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(count)
}

func (s *Store) TouchLogin(userID int64) {
	s.db.Exec(`UPDATE users SET last_login_at = ? WHERE id = ?`, now(), userID)
}
