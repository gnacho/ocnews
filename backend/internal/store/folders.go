package store

import (
	"database/sql"
	"errors"
)

func (s *Store) ListFolders(userID int64) ([]Folder, error) {
	rows, err := s.db.Query(`SELECT id, name, parent_id FROM folders WHERE user_id = ? ORDER BY name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Folder{}
	for rows.Next() {
		f := Folder{}
		var parent sql.NullInt64
		if err := rows.Scan(&f.ID, &f.Name, &parent); err != nil {
			return nil, err
		}
		if parent.Valid {
			p := parent.Int64
			f.ParentID = &p
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// CreateFolder devuelve ErrConflict si ya existe una carpeta con ese nombre.
// parentID != nil crea una subcarpeta (debe pertenecer al usuario).
func (s *Store) CreateFolder(userID int64, name string, parentID *int64) (*Folder, error) {
	if name == "" {
		return nil, ErrConflict
	}
	if parentID != nil {
		exists, err := s.FolderExists(userID, *parentID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrNotFound
		}
	}
	res, err := s.db.Exec(`INSERT INTO folders (user_id, name, parent_id) VALUES (?, ?, ?)`, userID, name, parentID)
	if err != nil {
		return nil, ErrConflict
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Folder{ID: id, Name: name, ParentID: parentID}, nil
}

func (s *Store) DeleteFolder(userID, folderID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// las subcarpetas de la carpeta borrada suben a la raíz
	if _, err := tx.Exec(`UPDATE folders SET parent_id = NULL WHERE parent_id = ? AND user_id = ?`, folderID, userID); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM folders WHERE user_id = ? AND id = ?`, userID, folderID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return tx.Commit()
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
