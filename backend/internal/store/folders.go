package store

import (
	"database/sql"
	"errors"
)

func (s *Store) ListFolders(userID int64) ([]Folder, error) {
	rows, err := s.db.Query(`SELECT id, name FROM folders WHERE user_id = ? ORDER BY name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Folder{}
	for rows.Next() {
		f := Folder{}
		if err := rows.Scan(&f.ID, &f.Name); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// CreateFolder devuelve ErrConflict si ya existe una carpeta con ese nombre.
func (s *Store) CreateFolder(userID int64, name string) (*Folder, error) {
	if name == "" {
		return nil, ErrConflict
	}
	res, err := s.db.Exec(`INSERT INTO folders (user_id, name) VALUES (?, ?)`, userID, name)
	if err != nil {
		return nil, ErrConflict
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Folder{ID: id, Name: name}, nil
}

func (s *Store) DeleteFolder(userID, folderID int64) error {
	res, err := s.db.Exec(`DELETE FROM folders WHERE user_id = ? AND id = ?`, userID, folderID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// RenameFolder devuelve ErrConflict si el nuevo nombre ya existe.
func (s *Store) RenameFolder(userID, folderID int64, name string) error {
	if name == "" {
		return ErrConflict
	}
	res, err := s.db.Exec(`UPDATE folders SET name = ? WHERE user_id = ? AND id = ?`, name, userID, folderID)
	if err != nil {
		return ErrConflict
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) FolderExists(userID, folderID int64) (bool, error) {
	var one int
	err := s.db.QueryRow(`SELECT 1 FROM folders WHERE user_id = ? AND id = ?`, userID, folderID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}
