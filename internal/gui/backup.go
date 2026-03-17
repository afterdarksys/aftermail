package gui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"github.com/afterdarksys/aftermail/pkg/i18n"
)

// getDatabasePath resolves the standard SQLite path for AfterMail.
func getDatabasePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "aftermail.db"
	}
	// Default generic location for Fyne/CLI sharing
	return filepath.Join(homeDir, ".aftermail", "aftermail.db")
}

// BackupDatabase opens a Save File dialog to dump the SQLite database cleanly to a user-selected path.
func BackupDatabase(w fyne.Window) {
	dbPath := getDatabasePath()
	
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		dialog.ShowError(errors.New(i18n.T("err_db_not_found", "Local database not initialized yet. Nothing to backup.")), w)
		return
	}

	dialog.ShowFileSave(func(uc fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		if uc == nil {
			return // User canceled
		}
		defer uc.Close()

		source, err := os.Open(dbPath)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to open source database: %w", err), w)
			return
		}
		defer source.Close()

		_, err = io.Copy(uc, source)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to write backup: %w", err), w)
			return
		}

		dialog.ShowInformation(i18n.T("backup_success", "Backup Successful"), i18n.T("backup_success_msg", "Database was saved securely."), w)
	}, w)
}

// RestoreDatabase opens an Open File dialog to overwrite the active SQLite file, warning the user it's destructive.
func RestoreDatabase(w fyne.Window) {
	dialog.ShowConfirm(
		i18n.T("restore_warning_title", "Restore Database?"),
		i18n.T("restore_warning_msg", "WARNING: Restoring will overwrite all current messages, rules, and local account states. Are you sure?"),
		func(b bool) {
			if !b {
				return
			}
			
			dialog.ShowFileOpen(func(uc fyne.URIReadCloser, err error) {
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				if uc == nil {
					return
				}
				defer uc.Close()

				dbPath := getDatabasePath()
				dest, err := os.OpenFile(dbPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
				if err != nil {
					dialog.ShowError(fmt.Errorf("failed to rewrite database: %w", err), w)
					return
				}
				defer dest.Close()

				_, err = io.Copy(dest, uc)
				if err != nil {
					dialog.ShowError(errors.New("corruption during restore: "+err.Error()), w)
					return
				}

				dialog.ShowInformation(
					i18n.T("restore_success", "Restore Successful"), 
					i18n.T("restore_success_msg", "Database restored successfully. Please restart ADS Mail to apply changes."), 
					w,
				)
			}, w)
		}, w)
}
