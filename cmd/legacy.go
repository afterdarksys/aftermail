package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/afterdarksys/aftermail/pkg/importer"
	"github.com/afterdarksys/aftermail/pkg/storage"
)

var (
	migratePath    string
	migrateType    string
	targetAccount  int64
	autoScan       bool
	scanPaths      []string
)

var legacyCmd = &cobra.Command{
	Use:   "legacy",
	Short: "Import legacy mailboxes (Pine, Mutt, mbox, Maildir)",
	Long: `Import messages from legacy mail clients and formats.

Supported formats:
  - mbox:      Traditional UNIX mbox format (Pine, Thunderbird)
  - maildir:   Maildir format (Mutt, modern clients)
  - aftermail: AfterMail native SQLite database

Examples:
  # Import a specific mbox file
  aftermail legacy --migrate=/var/mail/ryan --type=mbox --account=1

  # Import a Maildir folder
  aftermail legacy --migrate=~/Maildir --type=maildir --account=1

  # Import from another AfterMail database
  aftermail legacy --migrate=old.db --type=aftermail --account=1

  # Auto-detect and scan common Pine/Mutt locations
  aftermail legacy --auto-scan --account=1

  # Scan specific directories
  aftermail legacy --scan-path=~/mail --scan-path=/var/mail --account=1
`,
	RunE: runLegacy,
}

func init() {
	rootCmd.AddCommand(legacyCmd)

	legacyCmd.Flags().StringVar(&migratePath, "migrate", "", "Path to mailbox file or directory")
	legacyCmd.Flags().StringVar(&migrateType, "type", "", "Mailbox type: mbox, maildir, aftermail (auto-detected if not specified)")
	legacyCmd.Flags().Int64Var(&targetAccount, "account", 1, "Target account ID in AfterMail database")
	legacyCmd.Flags().BoolVar(&autoScan, "auto-scan", false, "Auto-scan common Pine/Mutt mailbox locations")
	legacyCmd.Flags().StringSliceVar(&scanPaths, "scan-path", []string{}, "Additional paths to scan for mailboxes")
}

func runLegacy(cmd *cobra.Command, args []string) error {
	// Initialize database
	dbPath := os.Getenv("AFTERMAIL_DB")
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		dbPath = filepath.Join(home, ".aftermail", "aftermail.db")
	}

	// Ensure directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0700); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := storage.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	log.Printf("[Legacy] Using database: %s", dbPath)
	log.Printf("[Legacy] Target account ID: %d", targetAccount)

	// Ensure target account exists or create a default one
	if err := ensureAccount(db, targetAccount); err != nil {
		return fmt.Errorf("failed to ensure account exists: %w", err)
	}

	// Handle auto-scan or scan-path mode
	if autoScan || len(scanPaths) > 0 {
		return scanForMailboxes(db, targetAccount)
	}

	// Handle direct migration
	if migratePath == "" {
		return fmt.Errorf("either --migrate or --auto-scan must be specified")
	}

	// Auto-detect type if not specified
	if migrateType == "" {
		migrateType = detectMailboxType(migratePath)
		log.Printf("[Legacy] Auto-detected mailbox type: %s", migrateType)
	}

	// Perform migration
	return performMigration(db, migratePath, migrateType, targetAccount)
}

func detectMailboxType(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "unknown"
	}

	// If it's a directory, check for Maildir structure
	if info.IsDir() {
		if hasMaildirStructure(path) {
			return "maildir"
		}
		return "directory"
	}

	// If it's a file, check extension and content
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".db" || ext == ".sqlite" {
		return "aftermail"
	}

	// Try to detect mbox by reading first line
	file, err := os.Open(path)
	if err != nil {
		return "unknown"
	}
	defer file.Close()

	buf := make([]byte, 5)
	n, err := file.Read(buf)
	if err == nil && n == 5 && string(buf) == "From " {
		return "mbox"
	}

	return "unknown"
}

func hasMaildirStructure(path string) bool {
	// Check for cur, new, and tmp subdirectories
	requiredDirs := []string{"cur", "new", "tmp"}
	for _, dir := range requiredDirs {
		dirPath := filepath.Join(path, dir)
		if info, err := os.Stat(dirPath); err != nil || !info.IsDir() {
			return false
		}
	}
	return true
}

func performMigration(db *storage.DB, path, mailboxType string, accountID int64) error {
	log.Printf("[Legacy] Importing %s mailbox from %s", mailboxType, path)

	switch mailboxType {
	case "mbox":
		return importer.ParseMbox(db, path, accountID)
	case "maildir":
		return importer.ParseMaildir(db, path, accountID)
	case "aftermail":
		return importer.ImportAfterMailDB(db, path, accountID)
	default:
		return fmt.Errorf("unsupported mailbox type: %s", mailboxType)
	}
}

func scanForMailboxes(db *storage.DB, accountID int64) error {
	log.Printf("[Legacy] Scanning for legacy mailboxes...")

	// Build list of paths to scan
	paths := scanPaths

	// Add common locations if auto-scan is enabled
	if autoScan {
		home, err := os.UserHomeDir()
		if err == nil {
			commonPaths := []string{
				filepath.Join(home, "mail"),          // Pine default
				filepath.Join(home, "Mail"),          // Common alternative
				filepath.Join(home, ".mail"),         // Hidden mail
				filepath.Join(home, "Maildir"),       // Maildir default
				filepath.Join(home, ".maildir"),      // Hidden Maildir
				"/var/mail",                          // System mail spool
				"/var/spool/mail",                    // Alternative spool
			}
			paths = append(paths, commonPaths...)
		}
	}

	found := 0
	imported := 0

	for _, scanPath := range paths {
		if _, err := os.Stat(scanPath); os.IsNotExist(err) {
			continue
		}

		log.Printf("[Legacy] Scanning: %s", scanPath)

		// Walk the directory tree
		err := filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}

			// Detect mailbox type
			mailboxType := detectMailboxType(path)
			if mailboxType == "unknown" || mailboxType == "directory" {
				return nil
			}

			found++
			log.Printf("[Legacy] Found %s mailbox: %s", mailboxType, path)

			// Ask for confirmation
			fmt.Printf("\nImport %s mailbox from %s? [y/N]: ", mailboxType, path)
			var response string
			fmt.Scanln(&response)
			response = strings.ToLower(strings.TrimSpace(response))

			if response == "y" || response == "yes" {
				if err := performMigration(db, path, mailboxType, accountID); err != nil {
					log.Printf("[Legacy] Error importing %s: %v", path, err)
				} else {
					imported++
					log.Printf("[Legacy] Successfully imported: %s", path)
				}
			}

			return nil
		})

		if err != nil {
			log.Printf("[Legacy] Error scanning %s: %v", scanPath, err)
		}
	}

	log.Printf("[Legacy] Scan complete. Found %d mailboxes, imported %d", found, imported)
	return nil
}

func ensureAccount(db *storage.DB, accountID int64) error {
	// Check if account exists
	exists, err := db.AccountExists(accountID)
	if err != nil {
		return err
	}

	if !exists {
		log.Printf("[Legacy] Creating default account with ID %d for legacy import", accountID)
		return db.CreateLegacyAccount(accountID, "Legacy Import", "legacy@localhost")
	}

	return nil
}
