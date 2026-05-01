package storage

import (
	"database/sql"
	"time"
)

// Commitment mirrors pkg/commitments.Commitment for DB storage.
// We duplicate the struct here to keep pkg/storage free of circular deps.
type Commitment struct {
	ID          int64
	MessageID   string
	ThreadID    string
	Sender      string
	Recipient   string
	Subject     string
	Kind        string
	Text        string
	DueDate     time.Time
	HasDueDate  bool
	Status      string
	Confidence  float64
	ExtractedAt time.Time
	ResolvedAt  time.Time
	Notes       string
}

// ensureCommitmentsTable creates the commitments table if it doesn't exist.
// Called lazily so existing DBs get the table on first use.
func (db *DB) ensureCommitmentsTable() error {
	_, err := db.conn.Exec(`
	CREATE TABLE IF NOT EXISTS commitments (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		message_id   TEXT NOT NULL,
		thread_id    TEXT,
		sender       TEXT NOT NULL,
		recipient    TEXT,
		subject      TEXT,
		kind         TEXT NOT NULL,
		text         TEXT NOT NULL,
		due_date     DATETIME,
		has_due_date BOOLEAN DEFAULT 0,
		status       TEXT DEFAULT 'open',
		confidence   REAL DEFAULT 1.0,
		extracted_at DATETIME NOT NULL,
		resolved_at  DATETIME,
		notes        TEXT DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_commitments_message ON commitments(message_id);
	CREATE INDEX IF NOT EXISTS idx_commitments_status  ON commitments(status);
	CREATE INDEX IF NOT EXISTS idx_commitments_due     ON commitments(due_date);
	CREATE INDEX IF NOT EXISTS idx_commitments_sender  ON commitments(sender);
	`)
	return err
}

// SaveCommitment inserts a new commitment record.
func (db *DB) SaveCommitment(c *Commitment) (int64, error) {
	if err := db.ensureCommitmentsTable(); err != nil {
		return 0, err
	}

	var dueDate interface{}
	if c.HasDueDate && !c.DueDate.IsZero() {
		dueDate = c.DueDate
	}

	res, err := db.conn.Exec(`
		INSERT INTO commitments
		  (message_id, thread_id, sender, recipient, subject, kind, text,
		   due_date, has_due_date, status, confidence, extracted_at, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.MessageID, c.ThreadID, c.Sender, c.Recipient, c.Subject,
		c.Kind, c.Text, dueDate, c.HasDueDate, c.Status,
		c.Confidence, c.ExtractedAt, c.Notes,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	c.ID = id
	return id, nil
}

// ResolveCommitment marks a commitment as resolved.
func (db *DB) ResolveCommitment(id int64) error {
	if err := db.ensureCommitmentsTable(); err != nil {
		return err
	}
	_, err := db.conn.Exec(
		`UPDATE commitments SET status = 'resolved', resolved_at = ? WHERE id = ?`,
		time.Now(), id,
	)
	return err
}

// SnoozeCommitment marks a commitment as snoozed.
func (db *DB) SnoozeCommitment(id int64) error {
	if err := db.ensureCommitmentsTable(); err != nil {
		return err
	}
	_, err := db.conn.Exec(
		`UPDATE commitments SET status = 'snoozed' WHERE id = ?`, id,
	)
	return err
}

// UpdateCommitmentNotes sets the user's notes on a commitment.
func (db *DB) UpdateCommitmentNotes(id int64, notes string) error {
	if err := db.ensureCommitmentsTable(); err != nil {
		return err
	}
	_, err := db.conn.Exec(
		`UPDATE commitments SET notes = ? WHERE id = ?`, notes, id,
	)
	return err
}

// ListCommitments returns commitments filtered by status and optionally by sender.
// Pass status="" to return all. Pass sender="" to return all senders.
func (db *DB) ListCommitments(status, sender string) ([]Commitment, error) {
	if err := db.ensureCommitmentsTable(); err != nil {
		return nil, err
	}

	query := `SELECT id, message_id, thread_id, sender, recipient, subject, kind,
	                 text, due_date, has_due_date, status, confidence,
	                 extracted_at, resolved_at, notes
	          FROM commitments WHERE 1=1`
	args := []interface{}{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if sender != "" {
		query += " AND sender = ?"
		args = append(args, sender)
	}
	query += " ORDER BY CASE WHEN due_date IS NULL THEN 1 ELSE 0 END, due_date ASC, extracted_at DESC"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCommitments(rows)
}

// CommitmentsForMessage returns all commitments extracted from a specific message.
func (db *DB) CommitmentsForMessage(messageID string) ([]Commitment, error) {
	if err := db.ensureCommitmentsTable(); err != nil {
		return nil, err
	}

	rows, err := db.conn.Query(`
		SELECT id, message_id, thread_id, sender, recipient, subject, kind,
		       text, due_date, has_due_date, status, confidence,
		       extracted_at, resolved_at, notes
		FROM commitments WHERE message_id = ? ORDER BY extracted_at ASC`, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCommitments(rows)
}

// OverdueCommitments returns open commitments whose due date has passed.
func (db *DB) OverdueCommitments() ([]Commitment, error) {
	if err := db.ensureCommitmentsTable(); err != nil {
		return nil, err
	}

	rows, err := db.conn.Query(`
		SELECT id, message_id, thread_id, sender, recipient, subject, kind,
		       text, due_date, has_due_date, status, confidence,
		       extracted_at, resolved_at, notes
		FROM commitments
		WHERE status = 'open' AND has_due_date = 1 AND due_date < ?
		ORDER BY due_date ASC`, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCommitments(rows)
}

func scanCommitments(rows *sql.Rows) ([]Commitment, error) {
	var result []Commitment
	for rows.Next() {
		var c Commitment
		var dueDate, resolvedAt sql.NullTime
		err := rows.Scan(
			&c.ID, &c.MessageID, &c.ThreadID, &c.Sender, &c.Recipient,
			&c.Subject, &c.Kind, &c.Text, &dueDate, &c.HasDueDate,
			&c.Status, &c.Confidence, &c.ExtractedAt, &resolvedAt, &c.Notes,
		)
		if err != nil {
			return nil, err
		}
		if dueDate.Valid {
			c.DueDate = dueDate.Time
		}
		if resolvedAt.Valid {
			c.ResolvedAt = resolvedAt.Time
		}
		result = append(result, c)
	}
	return result, rows.Err()
}
