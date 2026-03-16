package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// DB Wrapper
type DB struct {
	conn *sql.DB
}

// InitDB creates an SQLite database for the daemon
func InitDB(dsn string) (*DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := createSchema(db); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &DB{conn: db}, nil
}

func createSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS folders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		parent_id INTEGER,
		is_virtual BOOLEAN DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		remote_id TEXT, -- e.g. IMAP UID or AMP hash
		folder_id INTEGER,
		protocol TEXT, -- 'imap', 'pop3', 'amp', 'web3'
		sender TEXT,
		subject TEXT,
		body_plain TEXT,
		body_html TEXT,
		raw_headers TEXT,
		received_at DATETIME,
		flags TEXT, -- JSON array of tags/flags
		FOREIGN KEY(folder_id) REFERENCES folders(id)
	);

	-- Default Folders
	INSERT OR IGNORE INTO folders (name) VALUES ('Inbox'), ('Sent'), ('Trash'), ('Spam');
	`
	_, err := db.Exec(schema)
	return err
}

func (d *DB) InsertMessage(folderID int, sender, subject, plain, html, protocol string) (int64, error) {
	res, err := d.conn.Exec(`INSERT INTO messages 
		(folder_id, sender, subject, body_plain, body_html, protocol, received_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`, 
		folderID, sender, subject, plain, html, protocol, time.Now())
	
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) GetFolderByName(name string) (int, error) {
	var id int
	err := d.conn.QueryRow("SELECT id FROM folders WHERE name = ?", name).Scan(&id)
	return id, err
}

func (d *DB) Close() error {
	return d.conn.Close()
}
