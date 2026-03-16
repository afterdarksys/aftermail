package gui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ryan/meowmail/pkg/tlsconn"
)

// StartGUI initializes and shows the Fyne application.
func StartGUI() {
	a := app.New()
	w := a.NewWindow("Meowmail Debugger")

	w.Resize(fyne.NewSize(800, 600))

	// Add migration wizard menu item
	migrationMenu := fyne.NewMenuItem("Migration Wizard", func() {
		ShowMigrationWizard(w)
	})

	accountsMenu := fyne.NewMenuItem("Manage Accounts", func() {
		// TODO: Show accounts management dialog
	})

	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("About Meowmail", func() {
			// Show about dialog
		}),
		fyne.NewMenuItem("Documentation", func() {
			// Open docs
		}),
	)

	toolsMenu := fyne.NewMenu("Tools",
		migrationMenu,
		accountsMenu,
	)

	mainMenu := fyne.NewMainMenu(toolsMenu, helpMenu)
	w.SetMainMenu(mainMenu)

	tabs := container.NewAppTabs(
		container.NewTabItem("📥 Inbox & Folders", buildFoldersTab()),
		container.NewTabItem("✏️ Composer", buildComposerTab()),
		container.NewTabItem("📋 Rules Studio", buildRulesTab()),
		container.NewTabItem("🔌 Raw Session", buildSessionTab()),
		container.NewTabItem("🔬 Protocol Inspector", buildProtocolTab()),
		container.NewTabItem("🔒 Security Checks", buildSecurityTab()),
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
	var isConnected bool
	
	connectBtn := widget.NewButton("Connect", nil)
connectBtn.OnTapped = func() {
		if isConnected {
			if currentSession != nil {
				currentSession.Close()
				currentSession = nil
			}
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
		
		currentSession = session
		isConnected = true
		connectBtn.SetText("Disconnect")
		sessionLog.SetText(sessionLog.Text + "[System] Connected!\n")

		// Start a goroutine to read from the session
		go func() {
			for {
				if currentSession == nil {
					break
				}
				line, err := currentSession.ReadLine()
				if err != nil {
					// Connection closed or error
					sessionLog.SetText(sessionLog.Text + fmt.Sprintf("\n[System] Connection lost: %v\n", err))
					isConnected = false
					connectBtn.SetText("Connect")
					currentSession = nil
					break
				}
				sessionLog.SetText(sessionLog.Text + "< " + line)
			}
		}()
	}
	
	commandEntry := widget.NewEntry()
	commandEntry.SetPlaceHolder("Enter raw command (e.g. EHLO example.com)")
	
	sendBtn := widget.NewButton("Send", func() {
		if !isConnected || currentSession == nil {
			sessionLog.SetText(sessionLog.Text + "\n[System] Error: Not connected.\n")
			return
		}
		
		cmd := commandEntry.Text
		if cmd == "" {
			return
		}

		err := currentSession.WriteLine(cmd)
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
