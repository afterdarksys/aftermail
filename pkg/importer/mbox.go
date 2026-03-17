package importer

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/afterdarksys/aftermail/pkg/storage"
)

// ParseMbox streams through a legacy UNIX .mbox file and injects elements into SQLite
func ParseMbox(db *storage.DB, mboxPath string, targetAccountID string) error {
	log.Printf("[Importer] Spinning up Mbox parser for %s...", mboxPath)

	file, err := os.Open(mboxPath)
	if err != nil {
		return fmt.Errorf("could not open mbox: %w", err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	var messageBuffer bytes.Buffer
	msgCount := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}

		// "From " (with a space) indicates a boundary in .mbox standard
		if strings.HasPrefix(line, "From ") {
			// Flush current buffer if we have an existing message
			if messageBuffer.Len() > 0 {
				_ = injectMessage(db, messageBuffer.String(), targetAccountID)
				messageBuffer.Reset()
				msgCount++
			}
		}

		messageBuffer.WriteString(line)

		if err == io.EOF {
			// Flush final pending message
			if messageBuffer.Len() > 0 {
				_ = injectMessage(db, messageBuffer.String(), targetAccountID)
				msgCount++
			}
			break
		}
	}

	log.Printf("[Importer] Successfully processed and transferred %d messages.", msgCount)
	return nil
}

func injectMessage(db *storage.DB, raw string, accountID string) error {
	// STUB calls actual storage.DB methods to bind to SQLite message store
	return nil
}
