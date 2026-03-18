package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/afterdarksys/aftermail/pkg/i18n"
	"github.com/afterdarksys/aftermail/pkg/plugins"
	"github.com/afterdarksys/aftermail/pkg/storage"
	"github.com/afterdarksys/aftermail/pkg/tlsconn"
)

// StartGUI initializes and shows the Fyne application.
func StartGUI() {
	a := app.New()
	
	// Initialize i18n
	if err := i18n.Init("en"); err != nil {
		fmt.Printf("[Warning] Failed to initialize i18n: %v\n", err)
	}

	// Apply custom AfterMail theme (Dark Mode by default)
	isDark := true
	a.Settings().SetTheme(NewAfterMailTheme(isDark))

	w := a.NewWindow(i18n.T("app_title", "ADS Mail - Professional Email Client"))
	w.Resize(fyne.NewSize(1400, 900))

	// Global Keyboard Shortcuts
	ctrlF := &desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: fyne.KeyModifierShortcutDefault}
	w.Canvas().AddShortcut(ctrlF, func(shortcut fyne.Shortcut) {
		// In a real implementation this focuses the Search bar
		dialog.ShowInformation("Search", "Find shortcut triggered (Ctrl+F/Cmd+F)", w)
	})
	
	ctrlN := &desktop.CustomShortcut{KeyName: fyne.KeyN, Modifier: fyne.KeyModifierShortcutDefault}
	w.Canvas().AddShortcut(ctrlN, func(shortcut fyne.Shortcut) {
		dialog.ShowInformation("New Message", "New Message shortcut triggered (Ctrl+N/Cmd+N)", w)
	})

	// Global Network Polling Hook (Offline Mode)
	isOffline := false
	stopNetworkPolling := make(chan struct{})

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Mock network condition ping (pseudo offline-check)
				// True implementations would dial 1.1.1.1:53 or default metrics endpoint.
				_, err := os.Stat("/tmp/force_offline")
				currentlyOffline := err == nil
				if currentlyOffline != isOffline {
					isOffline = currentlyOffline
					if isOffline {
						w.SetTitle(i18n.T("app_title_offline", "ADS Mail - Professional Email Client [OFFLINE MODE]"))
					} else {
						w.SetTitle(i18n.T("app_title", "ADS Mail - Professional Email Client"))
					}
				}
			case <-stopNetworkPolling:
				return
			}
		}
	}()

	// Initialize Plugin Manager
	homeDir, _ := os.UserHomeDir()
	pluginDir := filepath.Join(homeDir, ".aftermail", "plugins")
	_ = os.MkdirAll(pluginDir, 0755)

	// Initialize Local SQLite Storage
	dbPath := filepath.Join(homeDir, ".aftermail", "aftermail.db")
	db, err := storage.InitDB(dbPath)
	if err != nil {
		fmt.Printf("[Error] Failed to initialize local storage: %v\n", err)
	}

	pluginManager := plugins.NewManager(pluginDir)
	if err := pluginManager.LoadPlugins(); err != nil {
		fmt.Printf("[Warning] Plugin loader error: %v\n", err)
	}

	// Clean up resources when window closes
	w.SetOnClosed(func() {
		close(stopNetworkPolling)
		pluginManager.ShutdownAll()
		if db != nil {
			db.Close()
		}
	})

	// Menu bar
	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("New Message", func() {
			// TODO: Open new message window
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Import .mbox", func() {
			ImportMbox(w)
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Backup Database", func() {
			BackupDatabase(w)
		}),
		fyne.NewMenuItem("Restore Database", func() {
			RestoreDatabase(w)
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Settings", func() {
			openSettingsDialog(a)
		}),
		fyne.NewMenuItem("Quit", func() {
			a.Quit()
		}),
	)

	accountsMenu := fyne.NewMenu("Accounts",
		fyne.NewMenuItem("Add Account", func() {
			// TODO: Show add account dialog
		}),
		fyne.NewMenuItem("Manage Accounts", func() {
			// TODO: Show accounts management dialog
		}),
	)

	toolsMenu := fyne.NewMenu("Tools",
		fyne.NewMenuItem("Migration Wizard", func() {
			ShowMigrationWizard(w)
		}),
		fyne.NewMenuItem("Protocol Inspector", func() {
			// Switch to protocol tab
		}),
		fyne.NewMenuItem("Security Checks", func() {
			// Switch to security tab
		}),
		fyne.NewMenuItem("Rules Studio", func() {
			// Switch to rules tab
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Manage Plugins", func() {
			loaded := pluginManager.GetLoaded()
			var msg string
			if len(loaded) == 0 {
				msg = fmt.Sprintf("No plugins loaded from:\n%s", pluginDir)
			} else {
				msg = "Loaded Plugins:\n\n"
				for _, p := range loaded {
					msg += fmt.Sprintf("- %s: %s\n", p.Name(), p.Description())
				}
			}
			dialog.ShowInformation("Plugin Manager", msg, w)
		}),
		fyne.NewMenuItem("Toggle Theme", func() {
			isDark = !isDark
			a.Settings().SetTheme(NewAfterMailTheme(isDark))
		}),
	)

	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("Documentation", func() {
			// Open docs
		}),
		fyne.NewMenuItem("About ADS Mail", func() {
			content := "Made with Love and Cats 😸\n\n(c) 2026 After Dark Systems, LLC.\n\nhttps://www.aftermail.app/\nsupport@afterdarksys.com"
			dialog.ShowInformation("About Aftermail", content, w)
		}),
	)
	
	langMenu := fyne.NewMenu("Language",
		fyne.NewMenuItem("English", func() {
			i18n.SetLanguage("en")
			dialog.ShowInformation("Language Changed", "Language set to English. Restart application to apply full translations.", w)
		}),
		fyne.NewMenuItem("Español", func() {
			i18n.SetLanguage("es")
			dialog.ShowInformation("Idioma Cambiado", "Idioma establecido a Español. Reinicie la aplicación para aplicar las traducciones completas.", w)
		}),
	)

	formatMenu := fyne.NewMenu("Format",
		fyne.NewMenuItem("Lists", func() { dialog.ShowInformation("Format", "Lists feature coming soon", w) }),
		fyne.NewMenuItem("Style", func() { dialog.ShowInformation("Format", "Style feature coming soon", w) }),
		fyne.NewMenuItem("Alignment", func() { dialog.ShowInformation("Format", "Alignment feature coming soon", w) }),
		fyne.NewMenuItem("Indentation", func() { dialog.ShowInformation("Format", "Indentation feature coming soon", w) }),
	)

	messageMenu := fyne.NewMenu("Message",
		fyne.NewMenuItem("Flags (Colors)", func() { dialog.ShowInformation("Message", "Flags: Red, Orange, Yellow, Green, Blue, Purple, Gray", w) }),
		fyne.NewMenuItem("Highlight Color", func() { dialog.ShowInformation("Message", "Highlight Color feature coming soon", w) }),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Request Read Receipt", func() { dialog.ShowInformation("Message", "Read Receipt requested", w) }),
		fyne.NewMenuItem("Set Priority", func() { dialog.ShowInformation("Message", "Priority settings coming soon", w) }),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Tag as Junk", func() { dialog.ShowInformation("Message", "Message tagged as Junk", w) }),
	)

	mailboxMenu := fyne.NewMenu("Mailbox",
		fyne.NewMenuItem("SMTP Settings", func() { dialog.ShowInformation("Mailbox", "SMTP Settings coming soon", w) }),
		fyne.NewMenuItem("IMAP Settings", func() { dialog.ShowInformation("Mailbox", "IMAP Settings coming soon", w) }),
		fyne.NewMenuItem("Cloud Mail Providers", func() { dialog.ShowInformation("Mailbox", "Cloud Mail Providers settings coming soon", w) }),
	)

	mailscriptMenu := fyne.NewMenu("MailScript",
		fyne.NewMenuItem("Open Editor", func() {
			tabs := w.Content().(*container.AppTabs)
			for i, tab := range tabs.Items {
				if tab.Text == "Rules" {
					tabs.SelectIndex(i)
					break
				}
			}
		}),
		fyne.NewMenuItem("Toggle Syntax Highlighting", func() { dialog.ShowInformation("MailScript", "Syntax Highlighting toggled", w) }),
		fyne.NewMenuItem("Policy Settings", func() { ShowMailScriptPolicyDialog(w) }),
	)

	mainMenu := fyne.NewMainMenu(fileMenu, formatMenu, messageMenu, mailboxMenu, mailscriptMenu, accountsMenu, toolsMenu, helpMenu, langMenu)
	w.SetMainMenu(mainMenu)

	// Main content area with tabs
	tabs := container.NewAppTabs(
		container.NewTabItem("Mail", buildMailView(w)),
		container.NewTabItem("Composer", buildComposerTab(db)),
		container.NewTabItem("Contacts", buildContactsTab(w, db)),
		container.NewTabItem("Notes", buildNotesTab(w, db)),
		container.NewTabItem("Calendar", buildCalendarTab(w, db)),
		container.NewTabItem("Reminders", buildRemindersTab()),
		container.NewTabItem("Tasks", buildTasksTab(db)),
		container.NewTabItem("RSS", buildRSSTab(w)),
		container.NewTabItem("AfterSMTP/Web3", buildWeb3Tab(w)),
		container.NewTabItem("Solidity Editor", SolidityEditorTab(w)),
		container.NewTabItem("Rules", buildRulesTab()),
		container.NewTabItem("Protocol Inspector", buildProtocolTab()),
		container.NewTabItem("Security", buildSecurityTab()),
		container.NewTabItem("Debug Session", buildSessionTab()),
	)

	tabs.SetTabLocation(container.TabLocationTop)

	w.SetContent(tabs)
	w.ShowAndRun()
}

func buildSessionTab() fyne.CanvasObject {
	hostEntry := widget.NewEntry()
	hostEntry.SetPlaceHolder("mail.example.com:25")
	
	tlsCheck := widget.NewCheck("Use TLS", nil)
	
	sessionLog := widget.NewMultiLineEntry()
	sessionLog.Disable() // Read-only view for output
	sessionLog.SetPlaceHolder("Session output will appear here...")
	sessionLog.Wrapping = fyne.TextWrapWord
	
	var currentSession *tlsconn.Session
	var sessionMu sync.Mutex
	var isConnected bool

	connectBtn := widget.NewButton("Connect", nil)
connectBtn.OnTapped = func() {
		if isConnected {
			sessionMu.Lock()
			if currentSession != nil {
				currentSession.Close()
				currentSession = nil
			}
			sessionMu.Unlock()
			isConnected = false
			connectBtn.SetText("Connect")
			sessionLog.SetText(sessionLog.Text + "\n[System] Disconnected.\n")
			return
		}

		host := strings.TrimSpace(hostEntry.Text)
		if host == "" {
			sessionLog.SetText(sessionLog.Text + "\n[System] Error: Host cannot be empty.\n")
			return
		}

		sessionLog.SetText(sessionLog.Text + fmt.Sprintf("\n[System] Connecting to %s (TLS: %v)...\n", host, tlsCheck.Checked))

		session, err := tlsconn.Connect(host, tlsCheck.Checked, 10*time.Second)
		if err != nil {
			sessionLog.SetText(sessionLog.Text + fmt.Sprintf("[System] Connection failed: %v\n", err))
			return
		}

		sessionMu.Lock()
		currentSession = session
		sessionMu.Unlock()
		isConnected = true
		connectBtn.SetText("Disconnect")
		sessionLog.SetText(sessionLog.Text + "[System] Connected!\n")

		// Start a goroutine to read from the session
		go func() {
			for {
				sessionMu.Lock()
				session := currentSession
				sessionMu.Unlock()

				if session == nil {
					break
				}
				line, err := session.ReadLine()
				if err != nil {
					// Connection closed or error
					sessionLog.SetText(sessionLog.Text + fmt.Sprintf("\n[System] Connection lost: %v\n", err))
					isConnected = false
					connectBtn.SetText("Connect")
					sessionMu.Lock()
					currentSession = nil
					sessionMu.Unlock()
					break
				}
				sessionLog.SetText(sessionLog.Text + "< " + line)
			}
		}()
	}
	
	commandEntry := widget.NewEntry()
	commandEntry.SetPlaceHolder("Enter raw command (e.g. EHLO example.com)")
	
	sendBtn := widget.NewButton("Send", func() {
		sessionMu.Lock()
		session := currentSession
		sessionMu.Unlock()

		if !isConnected || session == nil {
			sessionLog.SetText(sessionLog.Text + "\n[System] Error: Not connected.\n")
			return
		}

		cmd := commandEntry.Text
		if cmd == "" {
			return
		}

		err := session.WriteLine(cmd)
		if err != nil {
			sessionLog.SetText(sessionLog.Text + fmt.Sprintf("\n[System] Send error: %v\n", err))
			return
		}
		
		sessionLog.SetText(sessionLog.Text + "> " + cmd + "\r\n")
		commandEntry.SetText("") // Clear input
	})
	
	controls := container.NewHBox(widget.NewLabel("Host Details:"), hostEntry, tlsCheck, connectBtn)
	inputArea := container.NewBorder(nil, nil, nil, sendBtn, commandEntry)
	
	return container.NewBorder(
		controls, // Top
		inputArea, // Bottom
		nil, nil, // Left, Right
		sessionLog, // Center
	)
}
