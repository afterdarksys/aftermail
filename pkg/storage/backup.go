package storage

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// CreateBackup duplicates the current active SQLite store into a timestamped file.
func (db *DB) CreateBackup(destDir string) (string, error) {
	// SQLite locking via WAL handles most live reads fine,
	// but an explicit VACUUM INTO would be safer in production for hot backups.

	// Ensure destination exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("failed creating backup directory: %w", err)
	}

	backupName := fmt.Sprintf("aftermail_backup_%s.sqlite", time.Now().Format("20060102_150405"))
	destPath := filepath.Join(destDir, backupName)

	log.Printf("[Backup] Initiating hot copy to %s...", destPath)

	// In SQLite with `modernc.org/sqlite` binding, we can execute a VACUUM INTO command
	query := fmt.Sprintf("VACUUM INTO '%s'", destPath)
	_, err := db.conn.Exec(query)
	if err != nil {
		return "", fmt.Errorf("VACUUM INTO failed: %w", err)
	}

	return destPath, nil
}

// RestoreBackup overwrites the local database with a targeted backup snapshot
func RestoreBackup(sourceFile, destFile string) error {
	log.Printf("[Restore] Replacing %s with snapshot %s...", destFile, sourceFile)

	src, err := os.Open(sourceFile)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}
