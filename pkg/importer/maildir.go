package importer

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/afterdarksys/aftermail/pkg/accounts"
	"github.com/afterdarksys/aftermail/pkg/storage"
)

// ParseMaildir processes a Maildir directory structure (cur, new, tmp)
// This is the format used by Mutt and many modern mail clients
func ParseMaildir(db *storage.DB, maildirPath string, targetAccountID int64) error {
	log.Printf("[Importer] Spinning up Maildir parser for %s...", maildirPath)

	// Maildir structure has three subdirectories: new, cur, tmp
	// We'll import from 'cur' (current/read) and 'new' (unread)
	dirs := []string{
		filepath.Join(maildirPath, "cur"),
		filepath.Join(maildirPath, "new"),
	}

	msgCount := 0
	errorCount := 0

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			log.Printf("[Importer] Skipping non-existent directory: %s", dir)
			continue
		}

		// Read all files in the directory
		entries, err := os.ReadDir(dir)
		if err != nil {
			log.Printf("[Importer] Error reading directory %s: %v", dir, err)
			continue
		}

		isNew := strings.HasSuffix(dir, "new")

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			filePath := filepath.Join(dir, entry.Name())
			if err := injectMaildirMessage(db, filePath, targetAccountID, isNew); err != nil {
				log.Printf("[Importer] Error importing %s: %v", entry.Name(), err)
				errorCount++
			} else {
				msgCount++
			}
		}
	}

	log.Printf("[Importer] Successfully processed %d Maildir messages (%d errors).", msgCount, errorCount)
	return nil
}

func injectMaildirMessage(db *storage.DB, filePath string, accountID int64, isNew bool) error {
	// Read the entire message file
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse the RFC 5322 message
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("failed to parse MIME: %w", err)
	}

	// Extract headers
	from := msg.Header.Get("From")
	to := msg.Header.Get("To")
	cc := msg.Header.Get("Cc")
	subject := msg.Header.Get("Subject")
	dateStr := msg.Header.Get("Date")
	messageID := msg.Header.Get("Message-ID")

	// Parse date
	receivedAt := time.Now()
	if dateStr != "" {
		if parsed, err := mail.ParseDate(dateStr); err == nil {
			receivedAt = parsed
		}
	}

	// Build recipients list
	recipients := []string{}
	if to != "" {
		for _, addr := range strings.Split(to, ",") {
			recipients = append(recipients, strings.TrimSpace(addr))
		}
	}
	if cc != "" {
		for _, addr := range strings.Split(cc, ",") {
			recipients = append(recipients, strings.TrimSpace(addr))
		}
	}

	// Read body
	bodyBytes, err := io.ReadAll(msg.Body)
	if err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}
	bodyPlain := string(bodyBytes)

	// Build raw headers string
	var headersBuf bytes.Buffer
	for k, v := range msg.Header {
		for _, val := range v {
			headersBuf.WriteString(fmt.Sprintf("%s: %s\r\n", k, val))
		}
	}

	// Determine flags based on Maildir filename and location
	flags := []string{}
	flagsMap := make(map[string]bool) // Use map to avoid duplicates

	// Parse Maildir flags from filename (format: unique:2,FLAGS)
	filename := filepath.Base(filePath)
	if strings.Contains(filename, ":2,") {
		parts := strings.Split(filename, ":2,")
		if len(parts) == 2 {
			maildirFlags := parts[1]
			// Maildir flags: P=passed, R=replied, S=seen, T=trashed, D=draft, F=flagged
			if strings.Contains(maildirFlags, "S") {
				flagsMap["\\Seen"] = true
			}
			if strings.Contains(maildirFlags, "R") {
				flagsMap["\\Answered"] = true
			}
			if strings.Contains(maildirFlags, "F") {
				flagsMap["\\Flagged"] = true
			}
			if strings.Contains(maildirFlags, "D") {
				flagsMap["\\Draft"] = true
			}
			if strings.Contains(maildirFlags, "T") {
				flagsMap["\\Deleted"] = true
			}
		}
	} else if !isNew {
		// Messages in 'cur' without explicit flags are assumed read
		flagsMap["\\Seen"] = true
	}

	// Convert map to slice
	for flag := range flagsMap {
		flags = append(flags, flag)
	}

	// Create message struct
	message := &accounts.Message{
		AccountID:  accountID,
		RemoteID:   messageID, // Use Message-ID as remote ID
		FolderID:   1,         // Default to inbox folder
		Protocol:   "maildir",
		Sender:     from,
		Recipients: recipients,
		Subject:    subject,
		BodyPlain:  bodyPlain,
		BodyHTML:   "",
		RawHeaders: headersBuf.String(),
		ReceivedAt: receivedAt,
		Flags:      flags,
		Verified:   false,
	}

	// Save to database
	_, err = db.SaveMessage(message)
	return err
}
