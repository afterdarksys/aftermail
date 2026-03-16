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
	w := a.NewWindow("Meowmail - Professional Email Client")

	w.Resize(fyne.NewSize(1400, 900))

	// Menu bar
	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("New Message", func() {
			// TODO: Open new message window
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Settings", func() {
			// TODO: Open settings dialog
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
	)

	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("Documentation", func() {
			// Open docs
		}),
		fyne.NewMenuItem("About Meowmail", func() {
			// Show about dialog
		}),
	)

	mainMenu := fyne.NewMainMenu(fileMenu, accountsMenu, toolsMenu, helpMenu)
	w.SetMainMenu(mainMenu)

	// Main content area with tabs
	tabs := container.NewAppTabs(
		container.NewTabItem("Mail", buildMailView(w)),
		container.NewTabItem("Composer", buildComposerTab()),
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
