package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// Contact mapping to the local Address Book schema
type Contact struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	PublicKey string    `json:"public_key,omitempty"` // For Web3mail explicit verification
	GroupTag  string    `json:"group_tag,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// AddContact inserts a new contact record explicitly into SQLite
func (db *DB) AddContact(contact *Contact) (int64, error) {
	query := `INSERT INTO contacts (name, email, public_key, group_tag, notes) 
              VALUES (?, ?, ?, ?, ?)`
	
	result, err := db.conn.Exec(query,
		contact.Name,
		contact.Email,
		contact.PublicKey,
		contact.GroupTag,
		contact.Notes,
	)
	
	if err != nil {
		return 0, fmt.Errorf("failed to insert contact: %w", err)
	}

	return result.LastInsertId()
}

// UpdateContact modifies an existing address book entry
func (db *DB) UpdateContact(contact *Contact) error {
	query := `UPDATE contacts SET name=?, email=?, public_key=?, group_tag=?, notes=? WHERE id=?`
	
	_, err := db.conn.Exec(query,
		contact.Name,
		contact.Email,
		contact.PublicKey,
		contact.GroupTag,
		contact.Notes,
		contact.ID,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update contact: %w", err)
	}
	return nil
}

// DeleteContact removes a contact and associated meta completely
func (db *DB) DeleteContact(id int64) error {
	_, err := db.conn.Exec("DELETE FROM contacts WHERE id=?", id)
	return err
}

// ListContacts retrieves all stored contacts, optionally filtered by GroupTag
func (db *DB) ListContacts(groupFilter string) ([]Contact, error) {
	var rows *sql.Rows
	var err error

	if groupFilter == "" {
		rows, err = db.conn.Query("SELECT id, name, email, public_key, group_tag, notes, created_at FROM contacts ORDER BY name ASC")
	} else {
		rows, err = db.conn.Query("SELECT id, name, email, public_key, group_tag, notes, created_at FROM contacts WHERE group_tag=? ORDER BY name ASC", groupFilter)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query contacts: %w", err)
	}
	defer rows.Close()

	var contacts []Contact
	for rows.Next() {
		var c Contact
		var pubKey sql.NullString
		var group sql.NullString
		var notes sql.NullString

		if err := rows.Scan(&c.ID, &c.Name, &c.Email, &pubKey, &group, &notes, &c.CreatedAt); err != nil {
			continue // Skip corrupted rows gracefully
		}

		if pubKey.Valid { c.PublicKey = pubKey.String }
		if group.Valid { c.GroupTag = group.String }
		if notes.Valid { c.Notes = notes.String }

		contacts = append(contacts, c)
	}
	return contacts, nil
}
