package store

import "database/sql"

// GetUserSetting devuelve el valor de un ajuste del usuario ("" si no existe).
func (s *Store) GetUserSetting(userID int64, key string) (string, error) {
	var v string
	err := s.db.QueryRow(`SELECT value FROM user_settings WHERE user_id = ? AND key = ?`, userID, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return v, nil
}

// SetUserSetting fija el valor de un ajuste del usuario (UPSERT).
func (s *Store) SetUserSetting(userID int64, key, value string) error {
	if value == "" {
		_, err := s.db.Exec(`DELETE FROM user_settings WHERE user_id = ? AND key = ?`, userID, key)
		return err
	}
	_, err := s.db.Exec(
		`INSERT INTO user_settings (user_id, key, value) VALUES (?, ?, ?)
		 ON CONFLICT(user_id, key) DO UPDATE SET value = excluded.value`,
		userID, key, value)
	return err
}
