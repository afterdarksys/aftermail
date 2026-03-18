package importer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/afterdarksys/aftermail/pkg/accounts"
	"github.com/afterdarksys/aftermail/pkg/storage"
	_ "modernc.org/sqlite"
)

// ImportAfterMailDB imports messages from another AfterMail SQLite database
// This is useful for migrating from an old AfterMail installation
func ImportAfterMailDB(targetDB *storage.DB, sourceDBPath string, targetAccountID int64) error {
	log.Printf("[Importer] Importing from AfterMail database: %s", sourceDBPath)

	// Open the source database
	sourceDB, err := sql.Open("sqlite", sourceDBPath)
	if err != nil {
		return fmt.Errorf("failed to open source database: %w", err)
	}
	defer sourceDB.Close()

	// Query all messages from the source database
	rows, err := sourceDB.Query(`
		SELECT
			remote_id, folder_id, protocol,
			sender, recipients, subject, body_plain, body_html, raw_headers,
			amf_payload, received_at, flags,
			sender_did, signature, verified, stake_amount, ipfs_cid
		FROM messages
	`)
	if err != nil {
		return fmt.Errorf("failed to query source messages: %w", err)
	}
	defer rows.Close()

	msgCount := 0
	errorCount := 0

	for rows.Next() {
		var msg accounts.Message
		var recipientsJSON, flagsJSON, signaturesJSON string

		err := rows.Scan(
			&msg.RemoteID, &msg.FolderID, &msg.Protocol,
			&msg.Sender, &recipientsJSON, &msg.Subject, &msg.BodyPlain, &msg.BodyHTML, &msg.RawHeaders,
			&msg.AMFPayload, &msg.ReceivedAt, &flagsJSON,
			&msg.SenderDID, &signaturesJSON, &msg.Verified, &msg.StakeAmount, &msg.IPFSCID,
		)
		if err != nil {
			log.Printf("[Importer] Error scanning message: %v", err)
			errorCount++
			continue
		}

		// Deserialize JSON fields
		if err := json.Unmarshal([]byte(recipientsJSON), &msg.Recipients); err != nil {
			log.Printf("[Importer] Error parsing recipients: %v", err)
			errorCount++
			continue
		}
		if err := json.Unmarshal([]byte(flagsJSON), &msg.Flags); err != nil {
			log.Printf("[Importer] Error parsing flags: %v", err)
			msg.Flags = []string{} // Default to empty flags
		}
		if signaturesJSON != "" && signaturesJSON != "null" {
			if err := json.Unmarshal([]byte(signaturesJSON), &msg.Signatures); err != nil {
				log.Printf("[Importer] Error parsing signatures: %v", err)
				msg.Signatures = [][]byte{} // Default to empty
			}
		}

		// Override account ID to target account
		msg.AccountID = targetAccountID
		msg.ID = 0 // Let the database assign a new ID

		// Import attachments for this message
		// We'll need to get the original message ID first
		var originalMsgID int64
		err = sourceDB.QueryRow(`
			SELECT id FROM messages
			WHERE remote_id = ?
			LIMIT 1
		`, msg.RemoteID).Scan(&originalMsgID)

		if err == nil {
			attachments, err := getAttachments(sourceDB, originalMsgID)
			if err == nil && len(attachments) > 0 {
				msg.Attachments = attachments
			}
		}

		// Save to target database
		if _, err := targetDB.SaveMessage(&msg); err != nil {
			log.Printf("[Importer] Error saving message: %v", err)
			errorCount++
			continue
		}

		msgCount++
		if msgCount%100 == 0 {
			log.Printf("[Importer] Progress: %d messages imported...", msgCount)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	log.Printf("[Importer] Successfully imported %d messages (%d errors).", msgCount, errorCount)
	return nil
}

func getAttachments(db *sql.DB, messageID int64) ([]accounts.Attachment, error) {
	rows, err := db.Query(`
		SELECT filename, content_type, size, data, hash
		FROM attachments
		WHERE message_id = ?
	`, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []accounts.Attachment
	for rows.Next() {
		var att accounts.Attachment
		err := rows.Scan(&att.Filename, &att.ContentType, &att.Size, &att.Data, &att.Hash)
		if err != nil {
			log.Printf("Error scanning attachment: %v", err)
			continue
		}
		attachments = append(attachments, att)
	}

	return attachments, rows.Err()
}
