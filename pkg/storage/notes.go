package storage

import (
	"time"
)

// Note represents a markdown-capable note in the local database
type Note struct {
	ID        int64
	Title     string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AddNote inserts a new note into the database
func (db *DB) AddNote(n *Note) (int64, error) {
	query := `INSERT INTO notes (title, content, created_at, updated_at) VALUES (?, ?, ?, ?)`
	now := time.Now()
	res, err := db.conn.Exec(query, n.Title, n.Content, now, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateNote modifies an existing note
func (db *DB) UpdateNote(id int64, title, content string) error {
	query := `UPDATE notes SET title = ?, content = ?, updated_at = ? WHERE id = ?`
	_, err := db.conn.Exec(query, title, content, time.Now(), id)
	return err
}

// DeleteNote removes a note permanently
func (db *DB) DeleteNote(id int64) error {
	query := `DELETE FROM notes WHERE id = ?`
	_, err := db.conn.Exec(query, id)
	return err
}

// ListNotes retrieves all notes ordered by last updated
func (db *DB) ListNotes() ([]Note, error) {
	query := `SELECT id, title, content, created_at, updated_at FROM notes ORDER BY updated_at DESC`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, nil
}
