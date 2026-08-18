package store

import (
	"database/sql"
	"errors"
	"fmt"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
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

// CreateUserWithOCID crea un shadow user con identidad OpenCloud canónica.
func (s *Store) CreateUserWithOCID(username, ocID, passwordHash, displayName, role string) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO users (username, password_hash, display_name, role, language, oc_id, created_at) VALUES (?, ?, ?, ?, 'auto', ?, ?)`,
		username, passwordHash, displayName, role, ocID, now())
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrConflict
		}
		return 0, fmt.Errorf("crear usuario: %w", err)
	}
	return res.LastInsertId()
}

// isUniqueViolation detecta un error de constraint UNIQUE (código 2067) de
// SQLite, p.ej. un shadow user con un oc_id ya existente por una carrera entre
// dos requests del mismo usuario OpenCloud.
func isUniqueViolation(err error) bool {
	var se *sqlite.Error
	if errors.As(err, &se) {
		return se.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE
	}
	return false
}

// SetUserOCID vincula el oc_id a un usuario existente (p.ej. shadow Basic
// creado antes del primer login Bearer).
func (s *Store) SetUserOCID(userID int64, ocID string) error {
	_, err := s.db.Exec(`UPDATE users SET oc_id = ? WHERE id = ?`, ocID, userID)
	return err
}

func (s *Store) GetUserByOCID(ocID string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		`SELECT id, username, password_hash, display_name, role, language, oc_id, created_at, last_login_at
		 FROM users WHERE oc_id = ?`, ocID,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Role, &u.Language, &u.OCID, &u.CreatedAt, &u.LastLoginAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
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
		`SELECT id, username, password_hash, display_name, role, language, oc_id, created_at, last_login_at
		 FROM users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Role, &u.Language, &u.OCID, &u.CreatedAt, &u.LastLoginAt)
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

// TouchLogin actualiza last_login_at con throttle: solo si la última marca
// es anterior a 60s (evita una escritura por petición en cada GET de la API).
func (s *Store) TouchLogin(userID int64) {
	s.db.Exec(`UPDATE users SET last_login_at = ? WHERE id = ? AND last_login_at < ? - 60`,
		now(), userID, now())
}

// CountUsers devuelve el número de usuarios totales.
func (s *Store) CountUsers() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

// CountAdmins devuelve el número de usuarios con rol admin.
func (s *Store) CountAdmins() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'admin'`).Scan(&n)
	return n, err
}

// ListUsers devuelve todos los usuarios (sin hashes) para admin.
func (s *Store) ListUsers() ([]User, error) {
	rows, err := s.db.Query(
		`SELECT id, username, '', display_name, role, language, oc_id, created_at, last_login_at FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		u := User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Role, &u.Language, &u.OCID, &u.CreatedAt, &u.LastLoginAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) GetUserByID(id int64) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		`SELECT id, username, password_hash, display_name, role, language, oc_id, created_at, last_login_at
		 FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Role, &u.Language, &u.OCID, &u.CreatedAt, &u.LastLoginAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

// UpdateProfile actualiza display_name y language del propio usuario.
func (s *Store) UpdateProfile(userID int64, displayName, language string) error {
	_, err := s.db.Exec(`UPDATE users SET display_name = ?, language = ? WHERE id = ?`,
		displayName, language, userID)
	return err
}

// UpdatePassword sustituye el hash de contraseña tras validar la actual.
func (s *Store) UpdatePassword(userID int64, passwordHash string) error {
	_, err := s.db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, userID)
	return err
}

// UpdateUserFields (admin): role y language de cualquier usuario.
func (s *Store) UpdateUserFields(userID int64, role, language *string) error {
	if role != nil {
		if _, err := s.db.Exec(`UPDATE users SET role = ? WHERE id = ?`, *role, userID); err != nil {
			return err
		}
	}
	if language != nil {
		if _, err := s.db.Exec(`UPDATE users SET language = ? WHERE id = ?`, *language, userID); err != nil {
			return err
		}
	}
	return nil
}

// DeleteUser borra un usuario; los FK ON DELETE CASCADE limpian sus datos.
// Guard: no borrar el último admin (devuelve ErrConflict).
func (s *Store) DeleteUser(userID int64) error {
	var target User
	err := s.db.QueryRow(`SELECT role FROM users WHERE id = ?`, userID).Scan(&target.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if target.Role == "admin" {
		var admins int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'admin'`).Scan(&admins); err != nil {
			return err
		}
		if admins <= 1 {
			return ErrConflict
		}
	}
	_, err = s.db.Exec(`DELETE FROM users WHERE id = ?`, userID)
	return err
}
