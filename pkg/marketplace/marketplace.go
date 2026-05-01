// Package marketplace implements the MailScript script registry — a local + remote
// hub for discovering, installing, and publishing MailScript rules.
package marketplace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Script is a single entry in the marketplace registry.
type Script struct {
	// ID is a URL-safe slug: author/name
	ID string `json:"id"`

	// Name is the human-readable title.
	Name string `json:"name"`

	// Author is the publisher's email or handle.
	Author string `json:"author"`

	// Description is a one-sentence summary.
	Description string `json:"description"`

	// Tags are searchable keywords (e.g. "spam", "phishing", "newsletter").
	Tags []string `json:"tags"`

	// Version follows semver (e.g. "1.2.0").
	Version string `json:"version"`

	// Code is the Starlark source. Empty for remote listings (fetched on install).
	Code string `json:"code,omitempty"`

	// CodeHash is the SHA-256 of Code, used for integrity verification.
	CodeHash string `json:"code_hash"`

	// Stars is the community rating count (read-only from remote).
	Stars int `json:"stars"`

	// Downloads is the lifetime install count (read-only from remote).
	Downloads int `json:"downloads"`

	// CreatedAt is when the script was first published.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the script was last updated.
	UpdatedAt time.Time `json:"updated_at"`

	// Installed is true if this script is locally installed.
	Installed bool `json:"installed"`

	// InstalledAt records when the user installed this script.
	InstalledAt time.Time `json:"installed_at,omitempty"`
}

// Registry manages a local collection of installed scripts and an in-memory
// cache of remote listings.
type Registry struct {
	dir     string // ~/.aftermail/marketplace/
	scripts map[string]*Script
}

// NewRegistry opens or creates a local registry rooted at dir.
func NewRegistry(dir string) (*Registry, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("marketplace: failed to create dir: %w", err)
	}
	r := &Registry{
		dir:     dir,
		scripts: make(map[string]*Script),
	}
	if err := r.load(); err != nil {
		// Non-fatal: start with empty registry
		_ = err
	}
	return r, nil
}

// Install adds a script to the local registry and writes it to disk.
func (r *Registry) Install(s *Script) error {
	if s.ID == "" {
		return fmt.Errorf("marketplace: script ID is required")
	}
	if s.Code == "" {
		return fmt.Errorf("marketplace: script %q has no code", s.ID)
	}

	// Verify / compute hash
	computed := hashCode(s.Code)
	if s.CodeHash == "" {
		s.CodeHash = computed
	} else if s.CodeHash != computed {
		return fmt.Errorf("marketplace: integrity check failed for %q (expected %s, got %s)",
			s.ID, s.CodeHash, computed)
	}

	s.Installed = true
	s.InstalledAt = time.Now()
	r.scripts[s.ID] = s

	return r.saveScript(s)
}

// Uninstall removes a script from the local registry.
func (r *Registry) Uninstall(id string) error {
	if _, ok := r.scripts[id]; !ok {
		return fmt.Errorf("marketplace: script %q is not installed", id)
	}
	delete(r.scripts, id)

	path := r.scriptPath(id)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("marketplace: failed to remove %q: %w", id, err)
	}
	return nil
}

// Get returns a script by ID.
func (r *Registry) Get(id string) (*Script, bool) {
	s, ok := r.scripts[id]
	return s, ok
}

// Installed returns all locally installed scripts, sorted by name.
func (r *Registry) Installed() []*Script {
	var out []*Script
	for _, s := range r.scripts {
		if s.Installed {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

// Search returns installed scripts matching the query (name, description, or tag).
func (r *Registry) Search(query string) []*Script {
	q := strings.ToLower(query)
	var results []*Script
	for _, s := range r.scripts {
		if matchesQuery(s, q) {
			results = append(results, s)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Stars > results[j].Stars
	})
	return results
}

// Publish prepares a script for sharing: generates/validates its hash and
// returns the JSON representation ready for submission to a remote registry.
func Publish(name, author, description string, tags []string, code string) (*Script, error) {
	if name == "" || author == "" || code == "" {
		return nil, fmt.Errorf("marketplace: name, author, and code are required")
	}

	id := slugify(author) + "/" + slugify(name)
	now := time.Now()

	s := &Script{
		ID:          id,
		Name:        name,
		Author:      author,
		Description: description,
		Tags:        tags,
		Version:     "1.0.0",
		Code:        code,
		CodeHash:    hashCode(code),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return s, nil
}

// ExportJSON serialises a script to JSON for sharing.
func ExportJSON(s *Script) ([]byte, error) {
	// Never export the code hash mismatch
	s.CodeHash = hashCode(s.Code)
	return json.MarshalIndent(s, "", "  ")
}

// ImportJSON deserialises and installs a script from JSON bytes.
func (r *Registry) ImportJSON(data []byte) (*Script, error) {
	var s Script
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("marketplace: invalid JSON: %w", err)
	}
	if err := r.Install(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// --- Built-in starter scripts ---

// BuiltinScripts returns a curated set of ready-to-install starter scripts.
func BuiltinScripts() []*Script {
	now := time.Now()
	scripts := []struct {
		id, name, author, description string
		tags                          []string
		code                          string
	}{
		{
			id:          "aftermail/block-invoice-phishing",
			name:        "Block Invoice Phishing",
			author:      "aftermail",
			description: "Quarantines suspicious invoice/wire-transfer emails from unknown senders",
			tags:        []string{"phishing", "security", "finance"},
			code: `sender = get_header("From")
subject = get_header("Subject")
spam_score = getspamscore()

# High-risk keywords in subject
risky_subjects = ["wire transfer", "invoice", "urgent payment", "bank account", "verify payment"]
subject_lower = subject.lower()

is_risky_subject = False
for keyword in risky_subjects:
    if keyword in subject_lower:
        is_risky_subject = True
        break

# Also check body
is_risky_body = (
    search_body("wire transfer") or
    search_body("click here to pay") or
    search_body("urgent invoice") or
    search_body("bank details")
)

if (is_risky_subject or is_risky_body) and spam_score > 3.0:
    add_header("X-MailScript-Rule", "invoice-phishing")
    log_entry("Quarantined potential invoice phishing from: " + sender)
    quarantine()
else:
    accept()
`,
		},
		{
			id:          "aftermail/auto-file-newsletters",
			name:        "Auto-File Newsletters",
			author:      "aftermail",
			description: "Moves newsletters and marketing emails to a Newsletters folder",
			tags:        []string{"newsletter", "organisation", "productivity"},
			code: `subject = get_header("Subject")
list_id = get_header("List-Id")
list_unsub = get_header("List-Unsubscribe")
from_addr = get_header("From")

is_newsletter = (
    list_id != "" or
    list_unsub != "" or
    "unsubscribe" in get_header("X-Mailer").lower() or
    search_body("unsubscribe") or
    search_body("view in browser") or
    search_body("manage your preferences")
)

if is_newsletter:
    add_header("X-MailScript-Rule", "newsletter")
    fileinto("Newsletters")
else:
    accept()
`,
		},
		{
			id:          "aftermail/vip-priority",
			name:        "VIP Sender Priority",
			author:      "aftermail",
			description: "Tags emails from a configurable VIP list and files them into a Priority folder",
			tags:        []string{"productivity", "vip", "priority"},
			code: `# Edit this list to match your VIPs
VIP_SENDERS = [
    "boss@company.com",
    "ceo@company.com",
]

sender = get_header("From")

is_vip = False
for vip in VIP_SENDERS:
    if vip in sender:
        is_vip = True
        break

if is_vip:
    add_header("X-Priority", "1")
    add_header("X-MailScript-VIP", "true")
    fileinto("Priority")
else:
    accept()
`,
		},
		{
			id:          "aftermail/block-known-spam-domains",
			name:        "Block Known Spam Domains",
			author:      "aftermail",
			description: "Discards email from a configurable list of spam domains",
			tags:        []string{"spam", "blocklist", "security"},
			code: `BLOCKED_DOMAINS = [
    "spam-domain.example",
    "fake-offers.example",
]

sender_domain = get_sender_domain()

for domain in BLOCKED_DOMAINS:
    if domain == sender_domain:
        log_entry("Blocked spam domain: " + sender_domain)
        discard()

# RBL check — discard if sender IP is in a blacklist
sender_ip = get_sender_ip()
if rbl_check(sender_ip):
    log_entry("Blocked RBL-listed sender: " + sender_ip)
    discard()

accept()
`,
		},
		{
			id:          "aftermail/receipt-archiver",
			name:        "Auto-Archive Receipts",
			author:      "aftermail",
			description: "Files order confirmations, receipts, and shipping notices into a Receipts folder",
			tags:        []string{"finance", "receipts", "archiving"},
			code: `subject = get_header("Subject")
from_addr = get_header("From")
subject_lower = subject.lower()

receipt_keywords = [
    "order confirmation", "your receipt", "invoice #",
    "payment confirmed", "shipment", "tracking number",
    "your order has", "delivery confirmation"
]

is_receipt = False
for kw in receipt_keywords:
    if kw in subject_lower:
        is_receipt = True
        break

if not is_receipt:
    is_receipt = (
        search_body("order number") or
        search_body("tracking number") or
        search_body("your purchase")
    )

if is_receipt:
    add_header("X-MailScript-Rule", "receipt")
    fileinto("Receipts")
else:
    accept()
`,
		},
	}

	result := make([]*Script, len(scripts))
	for i, s := range scripts {
		result[i] = &Script{
			ID:          s.id,
			Name:        s.name,
			Author:      s.author,
			Description: s.description,
			Tags:        s.tags,
			Code:        s.code,
			CodeHash:    hashCode(s.code),
			Version:     "1.0.0",
			Stars:       0,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	}
	return result
}

// --- internal helpers ---

func (r *Registry) load() error {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(r.dir, e.Name()))
		if err != nil {
			continue
		}
		var s Script
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		r.scripts[s.ID] = &s
	}
	return nil
}

func (r *Registry) saveScript(s *Script) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.scriptPath(s.ID), data, 0600)
}

func (r *Registry) scriptPath(id string) string {
	safe := strings.ReplaceAll(id, "/", "__")
	return filepath.Join(r.dir, safe+".json")
}

func hashCode(code string) string {
	h := sha256.Sum256([]byte(code))
	return hex.EncodeToString(h[:])
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else if r == ' ' || r == '_' {
			b.WriteRune('-')
		}
	}
	return b.String()
}

func matchesQuery(s *Script, q string) bool {
	if strings.Contains(strings.ToLower(s.Name), q) {
		return true
	}
	if strings.Contains(strings.ToLower(s.Description), q) {
		return true
	}
	for _, tag := range s.Tags {
		if strings.Contains(strings.ToLower(tag), q) {
			return true
		}
	}
	return false
}
