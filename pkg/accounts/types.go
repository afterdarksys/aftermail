package accounts

import "time"

// AccountType represents the type of email account
type AccountType string

const (
	TypeIMAP     AccountType = "imap"
	TypePOP3     AccountType = "pop3"
	TypeGmail    AccountType = "gmail"
	TypeOutlook  AccountType = "outlook"
	TypeMsgsGlobal AccountType = "msgs.global"
	TypeAfterSMTP AccountType = "aftersmtp"
	TypeMailblocks AccountType = "mailblocks"
)

// Account represents a unified email account configuration
type Account struct {
	ID          int64       `json:"id"`
	Name        string      `json:"name"`
	Type        AccountType `json:"type"`
	Email       string      `json:"email"`

	// Traditional IMAP/POP3/SMTP
	ImapHost    string `json:"imap_host,omitempty"`
	ImapPort    int    `json:"imap_port,omitempty"`
	ImapUseTLS  bool   `json:"imap_use_tls,omitempty"`
	SmtpHost    string `json:"smtp_host,omitempty"`
	SmtpPort    int    `json:"smtp_port,omitempty"`
	SmtpUseTLS  bool   `json:"smtp_use_tls,omitempty"`
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"` // Should be encrypted at rest

	// OAuth2 (Gmail, Outlook)
	OAuthProvider    string    `json:"oauth_provider,omitempty"`
	OAuthClientID    string    `json:"oauth_client_id,omitempty"`
	OAuthClientSecret string   `json:"oauth_client_secret,omitempty"`
	OAuthAccessToken string    `json:"oauth_access_token,omitempty"`
	OAuthRefreshToken string   `json:"oauth_refresh_token,omitempty"`
	OAuthExpiry      time.Time `json:"oauth_expiry,omitempty"`

	// AfterSMTP/Mailblocks DID-based
	DID              string `json:"did,omitempty"` // e.g., did:aftersmtp:msgs.global:ryan
	Ed25519PrivKey   string `json:"ed25519_priv_key,omitempty"` // Signing key (encrypted at rest)
	X25519PrivKey    string `json:"x25519_priv_key,omitempty"`  // Encryption key (encrypted at rest)
	GatewayURL       string `json:"gateway_url,omitempty"`      // e.g., tls://amp.msgs.global:4433

	// Web3/Mailblocks specific
	WalletAddress    string `json:"wallet_address,omitempty"`   // Ethereum address
	IPFSEndpoint     string `json:"ipfs_endpoint,omitempty"`

	Enabled          bool      `json:"enabled"`
	CreatedAt        time.Time `json:"created_at"`
	LastSyncedAt     time.Time `json:"last_synced_at,omitempty"`
}

// Message represents a unified message structure supporting both MIME and AMF
type Message struct {
	ID           int64       `json:"id"`
	AccountID    int64       `json:"account_id"`
	RemoteID     string      `json:"remote_id"` // IMAP UID, AMP hash, etc.
	FolderID     int64       `json:"folder_id"`
	Protocol     string      `json:"protocol"` // 'imap', 'amp', 'web3', etc.

	// MIME fields
	Sender       string      `json:"sender"`
	Recipients   []string    `json:"recipients"`
	Subject      string      `json:"subject"`
	BodyPlain    string      `json:"body_plain"`
	BodyHTML     string      `json:"body_html"`
	RawHeaders   string      `json:"raw_headers,omitempty"`

	// AMF (AfterSMTP Mail Format) fields
	AMFPayload   []byte      `json:"amf_payload,omitempty"` // Serialized AMFPayload protobuf

	// Metadata
	ReceivedAt   time.Time   `json:"received_at"`
	Flags        []string    `json:"flags"`
	Attachments  []Attachment `json:"attachments,omitempty"`

	// Crypto verification (for AMP)
	SenderDID    string      `json:"sender_did,omitempty"`
	Signature    []byte      `json:"signature,omitempty"`
	Verified     bool        `json:"verified"`

	// Web3/Mailblocks
	StakeAmount  float64     `json:"stake_amount,omitempty"`
	IPFSCID      string      `json:"ipfs_cid,omitempty"`
}

// Attachment represents a file attachment
type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Data        []byte `json:"data,omitempty"` // May be stored separately for large files
	Hash        string `json:"hash,omitempty"` // SHA-256 for AMF attachments
}
