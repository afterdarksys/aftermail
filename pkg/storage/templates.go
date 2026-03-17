package storage

import (
	"fmt"
)

// Template represents a user-customizable SQLite string snippet
type Template struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Snippet string `json:"snippet"`
}

// AddTemplate creates a new snippet
func (db *DB) AddTemplate(t *Template) (int64, error) {
	query := `INSERT INTO templates (name, snippet) VALUES (?, ?)`
	result, err := db.conn.Exec(query, t.Name, t.Snippet)
	if err != nil {
		return 0, fmt.Errorf("failed to insert template: %w", err)
	}
	return result.LastInsertId()
}

// ListTemplates gets all available templates
func (db *DB) ListTemplates() ([]Template, error) {
	rows, err := db.conn.Query("SELECT id, name, snippet FROM templates ORDER BY name ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to query templates: %w", err)
	}
	defer rows.Close()

	var templates []Template
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.Name, &t.Snippet); err != nil {
			continue
		}
		templates = append(templates, t)
	}
	return templates, nil
}
