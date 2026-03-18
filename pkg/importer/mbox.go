package importer

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/afterdarksys/aftermail/pkg/accounts"
	"github.com/afterdarksys/aftermail/pkg/storage"
)

// ParseMbox streams through a legacy UNIX .mbox file and injects elements into SQLite
func ParseMbox(db *storage.DB, mboxPath string, targetAccountID int64) error {
	log.Printf("[Importer] Spinning up Mbox parser for %s...", mboxPath)

	file, err := os.Open(mboxPath)
	if err != nil {
		return fmt.Errorf("could not open mbox: %w", err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	var messageBuffer bytes.Buffer
	msgCount := 0
	errorCount := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}

		// "From " (with a space) indicates a boundary in .mbox standard
		// Skip the mbox "From " envelope line itself
		if strings.HasPrefix(line, "From ") && messageBuffer.Len() == 0 {
			// This is the mbox envelope marker, skip it
			if err == io.EOF {
				break
			}
			continue
		}

		if strings.HasPrefix(line, "From ") && messageBuffer.Len() > 0 {
			// Flush current buffer if we have an existing message
			if err := injectMessage(db, messageBuffer.Bytes(), targetAccountID); err != nil {
				log.Printf("[Importer] Error parsing message %d: %v", msgCount+1, err)
				errorCount++
			} else {
				msgCount++
			}
			messageBuffer.Reset()
			// Skip this "From " line as it's the next message's envelope
			if err == io.EOF {
				break
			}
			continue
		}

		messageBuffer.WriteString(line)

		if err == io.EOF {
			// Flush final pending message
			if messageBuffer.Len() > 0 {
				if err := injectMessage(db, messageBuffer.Bytes(), targetAccountID); err != nil {
					log.Printf("[Importer] Error parsing final message: %v", err)
					errorCount++
				} else {
					msgCount++
				}
			}
			break
		}
	}

	log.Printf("[Importer] Successfully processed %d messages (%d errors).", msgCount, errorCount)
	return nil
}

func injectMessage(db *storage.DB, raw []byte, accountID int64) error {
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
		recipients = append(recipients, strings.Split(to, ",")...)
	}
	if cc != "" {
		recipients = append(recipients, strings.Split(cc, ",")...)
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

	// Create message struct
	message := &accounts.Message{
		AccountID:  accountID,
		RemoteID:   messageID, // Use Message-ID as remote ID
		FolderID:   1,         // Default to inbox folder
		Protocol:   "mbox",
		Sender:     from,
		Recipients: recipients,
		Subject:    subject,
		BodyPlain:  bodyPlain,
		BodyHTML:   "",
		RawHeaders: headersBuf.String(),
		ReceivedAt: receivedAt,
		Flags:      []string{},
		Verified:   false,
	}

	// Save to database
	_, err = db.SaveMessage(message)
	return err
}
