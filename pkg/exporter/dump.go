package exporter

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/afterdarksys/aftermail/pkg/storage"
)

// ExportEML extracts a raw message payload directly to a standardized .eml artifact
func ExportEML(db *storage.DB, messageID string, destPath string) error {
	log.Printf("[Exporter] Extracting Message %s to EML envelope...", messageID)

	// In a complete implementation, this would execute `SELECT raw_payload FROM messages WHERE id=?`
	dummyPayload := []byte("From: alice@example.com\r\nTo: bob@example.com\r\nSubject: Test Dump\r\n\r\nHello World!")

	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to spawn EML artifact: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(dummyPayload); err != nil {
		return err
	}

	log.Printf("[Exporter] Successfully committed %d bytes.", len(dummyPayload))
	return nil
}

// ExportPDF stubs out rendering an abstract HTML body into a PDF document
func ExportPDF(htmlBody string, destPath string) error {
	log.Printf("[Exporter] Spawning Headless PDF Render for %d characters...", len(htmlBody))

	// Under normal desktop compilation this would hook into `wkhtmltopdf` or similar bindings,
	// but for now we write a raw artifact mimicking the stream.
	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.WriteString(file, "%PDF-1.4\n%Stub PDF Header\n")
	return err
}
