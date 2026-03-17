package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/afterdarksys/aftermail/pkg/accounts"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// DB Wrapper
type DB struct {
	conn    *sql.DB
	secrets accounts.SecureStorage
}

// InitDB creates an SQLite database for the daemon
func InitDB(dsn string) (*DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool for concurrent access
	// SQLite can handle multiple readers but only one writer at a time
	db.SetMaxOpenConns(10)           // Allow multiple concurrent reads
	db.SetMaxIdleConns(5)            // Keep some connections ready
	db.SetConnMaxLifetime(time.Hour) // Recycle connections periodically

	// Performance Optimizations
	pragmas := []string{
		"PRAGMA journal_mode = WAL;",        // Write-Ahead Logging for better concurrency
		"PRAGMA busy_timeout = 5000;",       // Wait up to 5s for locks instead of failing
		"PRAGMA synchronous = NORMAL;",      // Good balance of safety and speed
		"PRAGMA temp_store = MEMORY;",       // Store temp tables in memory
		"PRAGMA mmap_size = 30000000000;",   // Memory-map up to 30GB
		"PRAGMA cache_size = -64000;",       // 64MB page cache
		"PRAGMA foreign_keys = ON;",         // Enforce foreign key constraints
	}
	for _, pragma := range pragmas {
		_, err = db.Exec(pragma)
		if err != nil {
			return nil, fmt.Errorf("failed configuring SQLite %s: %w", pragma, err)
		}
	}

	// Verify WAL mode was actually enabled
	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode)
	if err != nil {
		return nil, fmt.Errorf("failed to verify journal mode: %w", err)
	}
	if journalMode != "wal" {
		log.Printf("Warning: WAL mode not enabled (got %s), falling back to default journal mode", journalMode)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := createSchema(db); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &DB{
		conn:    db,
		secrets: accounts.NewDefaultSecureStorage(),
	}, nil
}

func createSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS accounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		email TEXT NOT NULL,
		
		imap_host TEXT,
		imap_port INTEGER,
		imap_use_tls BOOLEAN,
		smtp_host TEXT,
		smtp_port INTEGER,
		smtp_use_tls BOOLEAN,
		username TEXT,
		password TEXT,

		oauth_provider TEXT,
		oauth_client_id TEXT,
		oauth_client_secret TEXT,
		oauth_access_token TEXT,
		oauth_refresh_token TEXT,
		oauth_expiry DATETIME,

		did TEXT,
		ed25519_priv_key TEXT,
		x25519_priv_key TEXT,
		gateway_url TEXT,

		wallet_address TEXT,
		ethereum_rpc_url TEXT,
		registry_address TEXT,
		ipfs_endpoint TEXT,

		enabled BOOLEAN DEFAULT 1,
		created_at DATETIME,
		last_synced_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS folders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		parent_id INTEGER,
		is_virtual BOOLEAN DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		message_id TEXT UNIQUE NOT NULL,
		thread_id TEXT,
		account_id TEXT NOT NULL,
		folder TEXT NOT NULL,
		sender TEXT NOT NULL,
		subject TEXT,
		snippet TEXT,
		date DATETIME NOT NULL,
		is_read BOOLEAN DEFAULT 0,
		is_starred BOOLEAN DEFAULT 0,
		has_attachments BOOLEAN DEFAULT 0,
		labels TEXT, -- JSON array
		raw_payload BLOB,
		read_receipt_sent BOOLEAN DEFAULT 0,
		is_phishing BOOLEAN DEFAULT 0,
		is_spam BOOLEAN DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS scheduled_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		account_id TEXT NOT NULL,
		from_addr TEXT NOT NULL,
		to_addr TEXT NOT NULL,
		payload BLOB,
		dispatch_at DATETIME NOT NULL,
		status TEXT DEFAULT 'pending'
	);

	CREATE TABLE IF NOT EXISTS templates (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		body TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS preferences (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_messages_folder ON messages(account_id, folder);
	CREATE INDEX IF NOT EXISTS idx_messages_thread ON messages(thread_id);
	CREATE INDEX IF NOT EXISTS idx_messages_date ON messages(date DESC);

	-- Global application settings
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	-- Master Address Book
	CREATE TABLE IF NOT EXISTS contacts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		public_key TEXT,
		group_tag TEXT,
		notes TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_contacts_group ON contacts(group_tag);

	-- Local Snippet Templates
	CREATE TABLE IF NOT EXISTS templates (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		snippet TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS attachments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		message_id INTEGER NOT NULL,
		filename TEXT,
		content_type TEXT,
		size INTEGER,
		data BLOB,
		hash TEXT,
		FOREIGN KEY(message_id) REFERENCES messages(id)
	);

	-- Advanced Features: Notes, Calendar, Tasks
	CREATE TABLE IF NOT EXISTS notes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		description TEXT,
		due_date DATETIME,
		completed BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS calendar_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		description TEXT,
		start_time DATETIME NOT NULL,
		end_time DATETIME NOT NULL,
		location TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS notifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type TEXT NOT NULL,
		message TEXT NOT NULL,
		is_read BOOLEAN DEFAULT 0,
		target_id INTEGER,
		data TEXT,
		created_at DATETIME
	);
	
	-- Default Folders
	INSERT OR IGNORE INTO folders (name) VALUES ('Inbox'), ('Sent'), ('Trash'), ('Spam');
	`
	_, err := db.Exec(schema)
	return err
}

// ListMessages returns the most recent messages from the inbox. 
// Used heavily by the Headless Daemon api.
func (db *DB) ListMessages() ([]map[string]interface{}, error) {
	// STUB: Actual `db.QueryContext("SELECT id, subject, sender, date FROM messages LIMIT 50")`
	// Return stub structure mapped against Fyne state schema to pass compilation.
	return []map[string]interface{}{}, nil
}

// InsertAccount adds a new account to the database
func (d *DB) InsertAccount(acc *accounts.Account) (int64, error) {
	if acc.CreatedAt.IsZero() {
		acc.CreatedAt = time.Now()
	}

	// Ensure these fields don't accidentally leak to SQLite
	blankedPassword := ""
	blankedOAuth := ""
	blankedEd25519PrivKey := ""
	blankedX25519PrivKey := ""

	res, err := d.conn.Exec(`INSERT INTO accounts (
		name, type, email,
		imap_host, imap_port, imap_use_tls, smtp_host, smtp_port, smtp_use_tls, username, password,
		oauth_provider, oauth_client_id, oauth_client_secret, oauth_access_token, oauth_refresh_token, oauth_expiry,
		did, ed25519_priv_key, x25519_priv_key, gateway_url,
		wallet_address, ethereum_rpc_url, registry_address, ipfs_endpoint,
		enabled, created_at, last_synced_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		acc.Name, string(acc.Type), acc.Email,
		acc.ImapHost, acc.ImapPort, acc.ImapUseTLS, acc.SmtpHost, acc.SmtpPort, acc.SmtpUseTLS, acc.Username, blankedPassword,
		acc.OAuthProvider, acc.OAuthClientID, acc.OAuthClientSecret, blankedOAuth, acc.OAuthRefreshToken, acc.OAuthExpiry,
		acc.DID, blankedEd25519PrivKey, blankedX25519PrivKey, acc.GatewayURL,
		acc.WalletAddress, acc.EthereumRPCURL, acc.RegistryAddress, acc.IPFSEndpoint,
		acc.Enabled, acc.CreatedAt, acc.LastSyncedAt,
	)

	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	acc.ID = id

	// Store explicit secrets safely via OS keychain AFTER getting the ID
	if err := d.secrets.StoreAccountSecret(fmt.Sprintf("%d", acc.ID), "Password", acc.Password); err != nil {
		fmt.Printf("Warning: error storing password in keychain: %v\n", err)
	}
	if err := d.secrets.StoreAccountSecret(fmt.Sprintf("%d", acc.ID), "OAuthToken", acc.OAuthAccessToken); err != nil {
		fmt.Printf("Warning: error storing oauth token in keychain: %v\n", err)
	}
	if err := d.secrets.StoreAccountSecret(fmt.Sprintf("%d", acc.ID), "Ed25519PrivKey", acc.Ed25519PrivKey); err != nil {
		fmt.Printf("Warning: error storing Ed25519 key in keychain: %v\n", err)
	}
	if err := d.secrets.StoreAccountSecret(fmt.Sprintf("%d", acc.ID), "X25519PrivKey", acc.X25519PrivKey); err != nil {
		fmt.Printf("Warning: error storing X25519 key in keychain: %v\n", err)
	}

	return id, nil
}

// GetAccount retrieves an account by ID
func (d *DB) GetAccount(id int64) (*accounts.Account, error) {
	var acc accounts.Account
	var accType string
	
	row := d.conn.QueryRow(`SELECT
		id, name, type, email,
		imap_host, imap_port, imap_use_tls, smtp_host, smtp_port, smtp_use_tls, username, password,
		oauth_provider, oauth_client_id, oauth_client_secret, oauth_access_token, oauth_refresh_token, oauth_expiry,
		did, ed25519_priv_key, x25519_priv_key, gateway_url,
		wallet_address, ethereum_rpc_url, registry_address, ipfs_endpoint,
		enabled, created_at, last_synced_at
	FROM accounts WHERE id = ?`, id)

	err := row.Scan(
		&acc.ID, &acc.Name, &accType, &acc.Email,
		&acc.ImapHost, &acc.ImapPort, &acc.ImapUseTLS, &acc.SmtpHost, &acc.SmtpPort, &acc.SmtpUseTLS, &acc.Username, &acc.Password,
		&acc.OAuthProvider, &acc.OAuthClientID, &acc.OAuthClientSecret, &acc.OAuthAccessToken, &acc.OAuthRefreshToken, &acc.OAuthExpiry,
		&acc.DID, &acc.Ed25519PrivKey, &acc.X25519PrivKey, &acc.GatewayURL,
		&acc.WalletAddress, &acc.EthereumRPCURL, &acc.RegistryAddress, &acc.IPFSEndpoint,
		&acc.Enabled, &acc.CreatedAt, &acc.LastSyncedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("account not found")
		}
		return nil, err
	}
	
	acc.Type = accounts.AccountType(accType)

	// Re-hydrate secrets
	accIDStr := fmt.Sprintf("%d", acc.ID)
	if pw, err := d.secrets.RetrieveAccountSecret(accIDStr, "Password"); err == nil && pw != "" { acc.Password = pw }
	if ox, err := d.secrets.RetrieveAccountSecret(accIDStr, "OAuthToken"); err == nil && ox != "" { acc.OAuthAccessToken = ox }
	if ed, err := d.secrets.RetrieveAccountSecret(accIDStr, "Ed25519PrivKey"); err == nil && ed != "" { acc.Ed25519PrivKey = ed }
	if xw, err := d.secrets.RetrieveAccountSecret(accIDStr, "X25519PrivKey"); err == nil && xw != "" { acc.X25519PrivKey = xw }

	return &acc, nil
}

// ListAccounts retrieves all accounts
func (d *DB) ListAccounts() ([]*accounts.Account, error) {
	rows, err := d.conn.Query(`SELECT
		id, name, type, email,
		imap_host, imap_port, imap_use_tls, smtp_host, smtp_port, smtp_use_tls, username, password,
		oauth_provider, oauth_client_id, oauth_client_secret, oauth_access_token, oauth_refresh_token, oauth_expiry,
		did, ed25519_priv_key, x25519_priv_key, gateway_url,
		wallet_address, ethereum_rpc_url, registry_address, ipfs_endpoint,
		enabled, created_at, last_synced_at
	FROM accounts ORDER BY created_at ASC`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*accounts.Account
	for rows.Next() {
		var acc accounts.Account
		var accType string

		err := rows.Scan(
			&acc.ID, &acc.Name, &accType, &acc.Email,
			&acc.ImapHost, &acc.ImapPort, &acc.ImapUseTLS, &acc.SmtpHost, &acc.SmtpPort, &acc.SmtpUseTLS, &acc.Username, &acc.Password,
			&acc.OAuthProvider, &acc.OAuthClientID, &acc.OAuthClientSecret, &acc.OAuthAccessToken, &acc.OAuthRefreshToken, &acc.OAuthExpiry,
			&acc.DID, &acc.Ed25519PrivKey, &acc.X25519PrivKey, &acc.GatewayURL,
			&acc.WalletAddress, &acc.EthereumRPCURL, &acc.RegistryAddress, &acc.IPFSEndpoint,
			&acc.Enabled, &acc.CreatedAt, &acc.LastSyncedAt,
		)
		if err != nil {
			return nil, err
		}
		acc.Type = accounts.AccountType(accType)

		// Re-hydrate secrets dynamically as we populate list
		accIDStr := fmt.Sprintf("%d", acc.ID)
		if pw, err := d.secrets.RetrieveAccountSecret(accIDStr, "Password"); err == nil && pw != "" { acc.Password = pw }
		if ox, err := d.secrets.RetrieveAccountSecret(accIDStr, "OAuthToken"); err == nil && ox != "" { acc.OAuthAccessToken = ox }
		if ed, err := d.secrets.RetrieveAccountSecret(accIDStr, "Ed25519PrivKey"); err == nil && ed != "" { acc.Ed25519PrivKey = ed }
		if xw, err := d.secrets.RetrieveAccountSecret(accIDStr, "X25519PrivKey"); err == nil && xw != "" { acc.X25519PrivKey = xw }

		result = append(result, &acc)
	}
	return result, nil
}

// UpdateAccount updates an existing account in the database
func (d *DB) UpdateAccount(acc *accounts.Account) error {
	if acc.ID == 0 {
		return fmt.Errorf("account ID is required for update")
	}

	// Update secrets in keychain first
	accIDStr := fmt.Sprintf("%d", acc.ID)
	if err := d.secrets.StoreAccountSecret(accIDStr, "Password", acc.Password); err != nil {
		log.Printf("Warning: error updating password in keychain: %v\n", err)
	}
	if err := d.secrets.StoreAccountSecret(accIDStr, "OAuthToken", acc.OAuthAccessToken); err != nil {
		log.Printf("Warning: error updating oauth token in keychain: %v\n", err)
	}
	if err := d.secrets.StoreAccountSecret(accIDStr, "Ed25519PrivKey", acc.Ed25519PrivKey); err != nil {
		log.Printf("Warning: error updating Ed25519 key in keychain: %v\n", err)
	}
	if err := d.secrets.StoreAccountSecret(accIDStr, "X25519PrivKey", acc.X25519PrivKey); err != nil {
		log.Printf("Warning: error updating X25519 key in keychain: %v\n", err)
	}

	// Don't store secrets in database
	blankedPassword := ""
	blankedOAuth := ""
	blankedEd25519PrivKey := ""
	blankedX25519PrivKey := ""

	_, err := d.conn.Exec(`UPDATE accounts SET
		name = ?, type = ?, email = ?,
		imap_host = ?, imap_port = ?, imap_use_tls = ?, smtp_host = ?, smtp_port = ?, smtp_use_tls = ?, username = ?, password = ?,
		oauth_provider = ?, oauth_client_id = ?, oauth_client_secret = ?, oauth_access_token = ?, oauth_refresh_token = ?, oauth_expiry = ?,
		did = ?, ed25519_priv_key = ?, x25519_priv_key = ?, gateway_url = ?,
		wallet_address = ?, ethereum_rpc_url = ?, registry_address = ?, ipfs_endpoint = ?,
		enabled = ?, last_synced_at = ?
	WHERE id = ?`,
		acc.Name, string(acc.Type), acc.Email,
		acc.ImapHost, acc.ImapPort, acc.ImapUseTLS, acc.SmtpHost, acc.SmtpPort, acc.SmtpUseTLS, acc.Username, blankedPassword,
		acc.OAuthProvider, acc.OAuthClientID, acc.OAuthClientSecret, blankedOAuth, acc.OAuthRefreshToken, acc.OAuthExpiry,
		acc.DID, blankedEd25519PrivKey, blankedX25519PrivKey, acc.GatewayURL,
		acc.WalletAddress, acc.EthereumRPCURL, acc.RegistryAddress, acc.IPFSEndpoint,
		acc.Enabled, acc.LastSyncedAt,
		acc.ID,
	)

	return err
}

// SaveMessage inserts a new message or updates an existing one (basic upsert logic using remote_id and account_id could be added)
// Currently implements simple insert.
func (d *DB) SaveMessage(msg *accounts.Message) (int64, error) {
	if msg.ReceivedAt.IsZero() {
		msg.ReceivedAt = time.Now()
	}

	// Serialize JSON fields
	recipientsJSON, _ := json.Marshal(msg.Recipients)
	flagsJSON, _ := json.Marshal(msg.Flags)
	signaturesJSON, _ := json.Marshal(msg.Signatures)

	tx, err := d.conn.Begin()
	if err != nil {
		return 0, err
	}
	// Defer a rollback in case anything fails
	defer func() {
		if err := tx.Rollback(); err != nil && err.Error() != "sql: transaction has already been committed or rolled back" {
			log.Printf("Error rolling back transaction: %v", err)
		}
	}()

	res, err := tx.Exec(`INSERT INTO messages (
		account_id, remote_id, folder_id, protocol,
		sender, recipients, subject, body_plain, body_html, raw_headers,
		amf_payload, received_at, flags,
		sender_did, signature, verified, stake_amount, ipfs_cid
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.AccountID, msg.RemoteID, msg.FolderID, msg.Protocol,
		msg.Sender, string(recipientsJSON), msg.Subject, msg.BodyPlain, msg.BodyHTML, msg.RawHeaders,
		msg.AMFPayload, msg.ReceivedAt, string(flagsJSON),
		msg.SenderDID, string(signaturesJSON), msg.Verified, msg.StakeAmount, msg.IPFSCID,
	)

	if err != nil {
		return 0, fmt.Errorf("insert message: %w", err)
	}

	msgID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	msg.ID = msgID

	// Insert attachments
	if len(msg.Attachments) > 0 {
		stmt, err := tx.Prepare(`INSERT INTO attachments (
			message_id, filename, content_type, size, data, hash
		) VALUES (?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return 0, err
		}
		defer stmt.Close()

		for _, att := range msg.Attachments {
			_, err = stmt.Exec(msgID, att.Filename, att.ContentType, att.Size, att.Data, att.Hash)
			if err != nil {
				return 0, fmt.Errorf("insert attachment: %w", err)
			}
		}
	}

	return msgID, tx.Commit()
}

// GetMessage retrieves a message and its attachments by ID
func (d *DB) GetMessage(id int64) (*accounts.Message, error) {
	var msg accounts.Message
	var recipientsJSON, flagsJSON, signaturesJSON string

	row := d.conn.QueryRow(`SELECT 
		id, account_id, remote_id, folder_id, protocol,
		sender, recipients, subject, body_plain, body_html, raw_headers,
		amf_payload, received_at, flags,
		sender_did, signature, verified, stake_amount, ipfs_cid
	FROM messages WHERE id = ?`, id)

	err := row.Scan(
		&msg.ID, &msg.AccountID, &msg.RemoteID, &msg.FolderID, &msg.Protocol,
		&msg.Sender, &recipientsJSON, &msg.Subject, &msg.BodyPlain, &msg.BodyHTML, &msg.RawHeaders,
		&msg.AMFPayload, &msg.ReceivedAt, &flagsJSON,
		&msg.SenderDID, &signaturesJSON, &msg.Verified, &msg.StakeAmount, &msg.IPFSCID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("message not found")
		}
		return nil, err
	}

	if err := json.Unmarshal([]byte(recipientsJSON), &msg.Recipients); err != nil {
		log.Printf("Failed to unmarshal recipients for message %d: %v", id, err)
		return nil, fmt.Errorf("corrupted recipients data: %w", err)
	}
	if err := json.Unmarshal([]byte(flagsJSON), &msg.Flags); err != nil {
		log.Printf("Failed to unmarshal flags for message %d: %v", id, err)
		return nil, fmt.Errorf("corrupted flags data: %w", err)
	}
	if err := json.Unmarshal([]byte(signaturesJSON), &msg.Signatures); err != nil {
		log.Printf("Failed to unmarshal signatures for message %d: %v", id, err)
		return nil, fmt.Errorf("corrupted signatures data: %w", err)
	}

	// Fetch attachments
	attRows, err := d.conn.Query(`SELECT filename, content_type, size, data, hash FROM attachments WHERE message_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer attRows.Close()

	for attRows.Next() {
		var att accounts.Attachment
		if err := attRows.Scan(&att.Filename, &att.ContentType, &att.Size, &att.Data, &att.Hash); err == nil {
			msg.Attachments = append(msg.Attachments, att)
		}
	}

	return &msg, nil
}

func (d *DB) GetFolderByName(name string) (int, error) {
	var id int
	err := d.conn.QueryRow("SELECT id FROM folders WHERE name = ?", name).Scan(&id)
	return id, err
}

func (d *DB) Close() error {
	return d.conn.Close()
}
