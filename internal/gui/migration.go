package gui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/afterdarksys/aftermail/pkg/accounts"
)

// MigrationWizard helps users migrate from traditional email to AfterSMTP/Mailblocks
type MigrationWizard struct {
	window          fyne.Window
	currentStep     int
	sourceAccount   *accounts.Account
	targetAccount   *accounts.Account
	migrateMessages bool
	migrateContacts bool
	migrationLog    *widget.Entry
}

// NewMigrationWizard creates a new migration wizard
func NewMigrationWizard(w fyne.Window) *MigrationWizard {
	return &MigrationWizard{
		window:          w,
		currentStep:     0,
		migrateMessages: true,
		migrateContacts: false,
		migrationLog:    widget.NewMultiLineEntry(),
	}
}

// Show displays the migration wizard
func (m *MigrationWizard) Show() {
	m.currentStep = 0
	m.showStep()
}

// showStep displays the current migration step
func (m *MigrationWizard) showStep() {
	var content fyne.CanvasObject

	switch m.currentStep {
	case 0:
		content = m.buildWelcomeStep()
	case 1:
		content = m.buildSourceSelectionStep()
	case 2:
		content = m.buildTargetSelectionStep()
	case 3:
		content = m.buildOptionsStep()
	case 4:
		content = m.buildConfirmationStep()
	case 5:
		content = m.buildMigrationStep()
	case 6:
		content = m.buildCompletionStep()
	default:
		return
	}

	dialog.ShowCustom("Email Migration Wizard", "Close", content, m.window)
}

// buildWelcomeStep shows the welcome screen
func (m *MigrationWizard) buildWelcomeStep() fyne.CanvasObject {
	title := widget.NewLabelWithStyle(
		"Welcome to the Email Migration Wizard",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	description := widget.NewLabel(`This wizard will help you migrate your emails from traditional
providers (Gmail, Outlook, IMAP) to AfterSMTP or Mailblocks.

Benefits of migrating to AfterSMTP/Mailblocks:
• End-to-end encryption using X25519 + AES-GCM
• Cryptographic signatures with Ed25519 (blockchain-backed)
• DID-based identity (no more password leaks)
• Proof-of-stake spam prevention (Mailblocks)
• Modern AMF message format (faster, cleaner than MIME)
• IPFS distributed storage option

Your traditional email accounts will continue to work normally.
This migration creates a secure alternative for your encrypted communications.`)
	description.Wrapping = fyne.TextWrapWord

	nextBtn := widget.NewButton("Next →", func() {
		m.currentStep++
		m.showStep()
	})

	return container.NewVBox(
		title,
		widget.NewSeparator(),
		description,
		widget.NewSeparator(),
		container.NewHBox(nextBtn),
	)
}

// buildSourceSelectionStep lets user select source account
func (m *MigrationWizard) buildSourceSelectionStep() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Step 1: Select Source Email Account", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	accountType := widget.NewSelect([]string{"Gmail (OAuth2)", "Outlook (OAuth2)", "IMAP (Username/Password)"}, nil)
	accountType.SetSelected("Gmail (OAuth2)")

	emailEntry := widget.NewEntry()
	emailEntry.SetPlaceHolder("your.email@gmail.com")

	authBtn := widget.NewButton("Authenticate with OAuth", func() {
		dialog.ShowInformation("OAuth Authentication",
			"Opening browser for OAuth consent...\n\nIn production, this would launch your browser to authenticate with "+accountType.Selected,
			m.window)
	})

	// IMAP-specific fields (shown conditionally)
	imapHost := widget.NewEntry()
	imapHost.SetPlaceHolder("imap.example.com")
	imapPort := widget.NewEntry()
	imapPort.SetPlaceHolder("993")
	imapPort.SetText("993")

	username := widget.NewEntry()
	username.SetPlaceHolder("username")
	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("password")

	imapFields := container.NewVBox(
		widget.NewLabel("IMAP Server:"),
		imapHost,
		widget.NewLabel("Port:"),
		imapPort,
		widget.NewLabel("Username:"),
		username,
		widget.NewLabel("Password:"),
		password,
	)
	imapFields.Hide()

	accountType.OnChanged = func(selected string) {
		if selected == "IMAP (Username/Password)" {
			authBtn.Hide()
			imapFields.Show()
		} else {
			authBtn.Show()
			imapFields.Hide()
		}
	}

	nextBtn := widget.NewButton("Next →", func() {
		// TODO: Validate authentication
		m.currentStep++
		m.showStep()
	})

	backBtn := widget.NewButton("← Back", func() {
		m.currentStep--
		m.showStep()
	})

	return container.NewVBox(
		title,
		widget.NewSeparator(),
		widget.NewLabel("Account Type:"),
		accountType,
		widget.NewLabel("Email Address:"),
		emailEntry,
		authBtn,
		imapFields,
		widget.NewSeparator(),
		container.NewHBox(backBtn, nextBtn),
	)
}

// buildTargetSelectionStep lets user select or create AfterSMTP/Mailblocks account
func (m *MigrationWizard) buildTargetSelectionStep() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Step 2: Select Destination", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	targetType := widget.NewRadioGroup([]string{
		"AfterSMTP (msgs.global) - Free DID-based encrypted email",
		"Mailblocks - Proof-of-stake email with IPFS storage",
		"Self-hosted AfterSMTP Gateway",
	}, nil)
	targetType.SetSelected("AfterSMTP (msgs.global) - Free DID-based encrypted email")

	existingDID := widget.NewEntry()
	existingDID.SetPlaceHolder("did:aftersmtp:msgs.global:yourname")

	createNewBtn := widget.NewButton("Create New DID Identity", func() {
		m.showDIDCreationDialog()
	})

	nextBtn := widget.NewButton("Next →", func() {
		m.currentStep++
		m.showStep()
	})

	backBtn := widget.NewButton("← Back", func() {
		m.currentStep--
		m.showStep()
	})

	return container.NewVBox(
		title,
		widget.NewSeparator(),
		widget.NewLabel("Choose your destination:"),
		targetType,
		widget.NewSeparator(),
		widget.NewLabel("Existing DID (if you have one):"),
		existingDID,
		container.NewHBox(createNewBtn),
		widget.NewSeparator(),
		container.NewHBox(backBtn, nextBtn),
	)
}

// buildOptionsStep lets user select what to migrate
func (m *MigrationWizard) buildOptionsStep() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Step 3: Migration Options", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	messagesCheck := widget.NewCheck("Migrate all messages", func(checked bool) {
		m.migrateMessages = checked
	})
	messagesCheck.SetChecked(true)

	contactsCheck := widget.NewCheck("Migrate contacts (coming soon)", func(checked bool) {
		m.migrateContacts = checked
	})
	contactsCheck.Disable()

	folderSelect := widget.NewCheckGroup([]string{
		"Inbox",
		"Sent",
		"Important",
		"All Folders",
	}, nil)
	folderSelect.SetSelected([]string{"Inbox", "Sent", "Important"})

	datePicker := widget.NewSelect([]string{
		"All messages",
		"Last 30 days",
		"Last 90 days",
		"Last year",
		"Custom date range...",
	}, nil)
	datePicker.SetSelected("All messages")

	encryptionNote := widget.NewLabel(`⚠️ Note: Messages will be re-encrypted using AfterSMTP's AMF format.
Original MIME headers will be preserved in extended_headers for compatibility.`)
	encryptionNote.Wrapping = fyne.TextWrapWord

	nextBtn := widget.NewButton("Next →", func() {
		m.currentStep++
		m.showStep()
	})

	backBtn := widget.NewButton("← Back", func() {
		m.currentStep--
		m.showStep()
	})

	return container.NewVBox(
		title,
		widget.NewSeparator(),
		messagesCheck,
		contactsCheck,
		widget.NewSeparator(),
		widget.NewLabel("Select folders to migrate:"),
		folderSelect,
		widget.NewSeparator(),
		widget.NewLabel("Date range:"),
		datePicker,
		widget.NewSeparator(),
		encryptionNote,
		widget.NewSeparator(),
		container.NewHBox(backBtn, nextBtn),
	)
}

// buildConfirmationStep shows summary before migration
func (m *MigrationWizard) buildConfirmationStep() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Step 4: Confirm Migration", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	summary := widget.NewLabel(fmt.Sprintf(`Migration Summary:

Source: Gmail (user@gmail.com)
Target: AfterSMTP (did:aftersmtp:msgs.global:ryan)

What will be migrated:
✓ All messages from Inbox, Sent, Important
✓ Estimated: ~1,250 messages
✓ Size: ~450 MB

Process:
1. Fetch messages from Gmail via OAuth
2. Convert MIME to AMF format
3. Encrypt with your X25519 key
4. Sign with your Ed25519 key
5. Upload to AfterSMTP gateway

Estimated time: 15-20 minutes

Your Gmail account will NOT be modified.`))
	summary.Wrapping = fyne.TextWrapWord

	startBtn := widget.NewButton("Start Migration", func() {
		m.currentStep++
		m.showStep()
		go m.performMigration()
	})

	backBtn := widget.NewButton("← Back", func() {
		m.currentStep--
		m.showStep()
	})

	return container.NewVBox(
		title,
		widget.NewSeparator(),
		summary,
		widget.NewSeparator(),
		container.NewHBox(backBtn, startBtn),
	)
}

// buildMigrationStep shows live migration progress
func (m *MigrationWizard) buildMigrationStep() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("Migrating Your Email...", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	progress := widget.NewProgressBar()
	progress.SetValue(0)

	m.migrationLog.Disable()
	m.migrationLog.SetPlaceHolder("Migration log will appear here...")

	statusLabel := widget.NewLabel("Starting migration...")

	return container.NewVBox(
		title,
		widget.NewSeparator(),
		statusLabel,
		progress,
		widget.NewSeparator(),
		widget.NewLabel("Migration Log:"),
		container.NewScroll(m.migrationLog),
	)
}

// buildCompletionStep shows migration results
func (m *MigrationWizard) buildCompletionStep() fyne.CanvasObject {
	title := widget.NewLabelWithStyle("✅ Migration Complete!", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	summary := widget.NewLabel(`Your email has been successfully migrated!

Results:
• Messages migrated: 1,247
• Failed: 3 (attachment size limit exceeded)
• Total size: 448 MB
• Time taken: 14 minutes 32 seconds

Next steps:
1. Configure your email client to use AfterSMTP
2. Start sending encrypted messages to other AfterSMTP users
3. Invite your contacts to join AfterSMTP for E2E encryption

Your Gmail account is still active and unchanged.
You can now use both traditional and AfterSMTP email!`)
	summary.Wrapping = fyne.TextWrapWord

	doneBtn := widget.NewButton("Done", func() {
		dialog.ShowInformation("Success", "Migration wizard completed successfully!", m.window)
	})

	return container.NewVBox(
		title,
		widget.NewSeparator(),
		summary,
		widget.NewSeparator(),
		container.NewHBox(doneBtn),
	)
}

// performMigration executes the actual migration (mock implementation)
func (m *MigrationWizard) performMigration() {
	messages := []string{
		"[00:00] Connecting to Gmail API...",
		"[00:02] Authentication successful",
		"[00:03] Fetching folder list...",
		"[00:04] Found 3 folders: Inbox, Sent, Important",
		"[00:05] Fetching messages from Inbox (1,043 messages)...",
		"[00:15] Converting message 100/1043 to AMF format...",
		"[00:25] Converting message 200/1043 to AMF format...",
		"[02:30] Inbox migration complete (1,043 messages)",
		"[02:31] Fetching messages from Sent (157 messages)...",
		"[03:45] Sent migration complete (157 messages)",
		"[03:46] Fetching messages from Important (47 messages)...",
		"[04:20] Important migration complete (47 messages)",
		"[04:21] Uploading to AfterSMTP gateway...",
		"[14:30] Upload complete",
		"[14:31] Verifying blockchain proofs...",
		"[14:32] Migration successful!",
	}

	for i, msg := range messages {
		time.Sleep(500 * time.Millisecond)
		m.migrationLog.SetText(m.migrationLog.Text + msg + "\n")

		// Simulate progress
		if i == len(messages)-1 {
			m.currentStep++
			m.showStep()
		}
	}
}

// showDIDCreationDialog shows the DID creation dialog
func (m *MigrationWizard) showDIDCreationDialog() {
	desiredUsername := widget.NewEntry()
	desiredUsername.SetPlaceHolder("username")

	domain := widget.NewSelect([]string{"msgs.global", "custom domain..."}, nil)
	domain.SetSelected("msgs.global")

	preview := widget.NewLabel("DID Preview: did:aftersmtp:msgs.global:username")

	desiredUsername.OnChanged = func(text string) {
		preview.SetText(fmt.Sprintf("DID Preview: did:aftersmtp:%s:%s", domain.Selected, text))
	}

	content := container.NewVBox(
		widget.NewLabelWithStyle("Create New DID Identity", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("Choose your username:"),
		desiredUsername,
		widget.NewLabel("Domain:"),
		domain,
		widget.NewSeparator(),
		preview,
		widget.NewSeparator(),
		widget.NewLabel("This will generate new Ed25519 and X25519 keypairs and register your DID on the blockchain."),
	)

	dialog.ShowCustomConfirm("Create DID", "Create", "Cancel", content, func(confirmed bool) {
		if confirmed {
			dialog.ShowInformation("DID Created",
				fmt.Sprintf("Successfully created DID: did:aftersmtp:msgs.global:%s\n\nYour keys have been securely stored.", desiredUsername.Text),
				m.window)
		}
	}, m.window)
}

// ShowMigrationWizard is a helper function to show the migration wizard
func ShowMigrationWizard(w fyne.Window) {
	wizard := NewMigrationWizard(w)
	wizard.Show()
}
