package gui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Message represents an email message
type Message struct {
	ID       string
	From     string
	Subject  string
	Preview  string
	Date     string
	Unread   bool
	Starred  bool
	Account  string
}

// buildMailView creates the professional three-pane email interface
func buildMailView(w fyne.Window) fyne.CanvasObject {
	// Toolbar at the top
	toolbar := buildToolbar()

	// Search bar
	searchBar := buildSearchBar()

	// Three-pane layout
	folderPane := buildFolderPane()
	messageListPane := buildMessageListPane()
	messageViewPane := buildMessageViewPane()

	// Status bar at the bottom
	statusBar := buildStatusBar()

	// Assemble the three-pane layout
	messageArea := container.NewHSplit(messageListPane, messageViewPane)
	messageArea.SetOffset(0.4) // 40% for message list, 60% for preview

	mainArea := container.NewHSplit(folderPane, messageArea)
	mainArea.SetOffset(0.18) // 18% for folders, 82% for messages

	// Combine everything
	topSection := container.NewVBox(toolbar, searchBar)

	content := container.NewBorder(
		topSection,  // Top
		statusBar,   // Bottom
		nil, nil,    // Left, Right
		mainArea,    // Center
	)

	return content
}

// buildToolbar creates the main toolbar with action buttons
func buildToolbar() fyne.CanvasObject {
	newMailBtn := widget.NewButton("New", func() {
		// TODO: Open composer
	})
	newMailBtn.Importance = widget.HighImportance

	replyBtn := widget.NewButton("Reply", func() {
		// TODO: Reply to message
	})

	replyAllBtn := widget.NewButton("Reply All", func() {
		// TODO: Reply all
	})

	forwardBtn := widget.NewButton("Forward", func() {
		// TODO: Forward message
	})

	deleteBtn := widget.NewButton("Delete", func() {
		// TODO: Delete message
	})
	deleteBtn.Importance = widget.DangerImportance

	archiveBtn := widget.NewButton("Archive", func() {
		// TODO: Archive message
	})

	markReadBtn := widget.NewButton("Mark Read", func() {
		// TODO: Mark as read
	})

	syncBtn := widget.NewButton("Sync", func() {
		// TODO: Sync with server
	})

	spacer := layout.NewSpacer()

	settingsBtn := widget.NewButton("Settings", func() {
		// TODO: Open settings
	})

	toolbar := container.NewHBox(
		newMailBtn,
		widget.NewSeparator(),
		replyBtn,
		replyAllBtn,
		forwardBtn,
		widget.NewSeparator(),
		deleteBtn,
		archiveBtn,
		widget.NewSeparator(),
		markReadBtn,
		syncBtn,
		spacer,
		settingsBtn,
	)

	return container.NewVBox(
		toolbar,
		widget.NewSeparator(),
	)
}

// buildSearchBar creates the search interface
func buildSearchBar() fyne.CanvasObject {
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search mail (subject, sender, body...)")

	filterBtn := widget.NewButton("Filter", func() {
		// TODO: Show filter options
	})

	searchBar := container.NewBorder(
		nil, nil,
		nil, filterBtn,
		searchEntry,
	)

	return container.NewPadded(searchBar)
}

// buildFolderPane creates the left folder navigation
func buildFolderPane() fyne.CanvasObject {
	// Account selector
	accountSelect := widget.NewSelect(
		[]string{"All Accounts", "work@company.com", "personal@gmail.com", "did:aftersmtp:msgs.global:ryan"},
		func(s string) {
			// TODO: Filter messages by account
		},
	)
	accountSelect.SetSelected("All Accounts")

	// Folder tree
	folderTree := widget.NewTree(
		func(id widget.TreeNodeID) []widget.TreeNodeID {
			switch id {
			case "":
				return []widget.TreeNodeID{"favorites", "folders", "accounts"}
			case "favorites":
				return []widget.TreeNodeID{"fav-inbox", "fav-unread", "fav-starred", "fav-important"}
			case "folders":
				return []widget.TreeNodeID{"inbox", "sent", "drafts", "archive", "trash", "spam"}
			case "accounts":
				return []widget.TreeNodeID{"acc-work", "acc-personal", "acc-aftersmtp"}
			case "acc-work":
				return []widget.TreeNodeID{"work-inbox", "work-sent", "work-custom"}
			case "acc-personal":
				return []widget.TreeNodeID{"personal-inbox", "personal-sent"}
			case "acc-aftersmtp":
				return []widget.TreeNodeID{"amp-inbox", "amp-sent"}
			default:
				return []widget.TreeNodeID{}
			}
		},
		func(id widget.TreeNodeID) bool {
			return id == "" || id == "favorites" || id == "folders" || id == "accounts" ||
				   id == "acc-work" || id == "acc-personal" || id == "acc-aftersmtp"
		},
		func(branch bool) fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Folder"),
				layout.NewSpacer(),
				widget.NewLabel("0"),
			)
		},
		func(id widget.TreeNodeID, branch bool, o fyne.CanvasObject) {
			c := o.(*fyne.Container)
			label := c.Objects[0].(*widget.Label)
			badge := c.Objects[2].(*widget.Label)

			folderNames := map[string]string{
				"favorites":        "⭐ Favorites",
				"folders":          "📁 Folders",
				"accounts":         "👤 Accounts",
				"fav-inbox":        "📥 Inbox",
				"fav-unread":       "📬 Unread",
				"fav-starred":      "⭐ Starred",
				"fav-important":    "🔴 Important",
				"inbox":            "📥 Inbox",
				"sent":             "📤 Sent",
				"drafts":           "📝 Drafts",
				"archive":          "📦 Archive",
				"trash":            "🗑️ Trash",
				"spam":             "🚫 Spam",
				"acc-work":         "work@company.com",
				"acc-personal":     "personal@gmail.com",
				"acc-aftersmtp":    "AfterSMTP",
				"work-inbox":       "📥 Inbox",
				"work-sent":        "📤 Sent",
				"work-custom":      "📂 Custom",
				"personal-inbox":   "📥 Inbox",
				"personal-sent":    "📤 Sent",
				"amp-inbox":        "📥 Inbox",
				"amp-sent":         "📤 Sent",
			}

			// Mock unread counts
			counts := map[string]int{
				"fav-inbox":     42,
				"fav-unread":    128,
				"inbox":         42,
				"work-inbox":    23,
				"personal-inbox": 19,
				"amp-inbox":     5,
			}

			if name, ok := folderNames[id]; ok {
				label.SetText(name)
			} else {
				label.SetText(id)
			}

			if count, ok := counts[id]; ok && count > 0 {
				badge.SetText(fmt.Sprintf("%d", count))
				badge.TextStyle = fyne.TextStyle{Bold: true}
			} else {
				badge.SetText("")
			}
		},
	)

	// Open favorites and folders by default
	folderTree.OpenBranch("favorites")
	folderTree.OpenBranch("folders")

	folderHeader := widget.NewLabelWithStyle("Navigation", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	return container.NewBorder(
		container.NewVBox(folderHeader, accountSelect, widget.NewSeparator()),
		nil, nil, nil,
		folderTree,
	)
}

// buildMessageListPane creates the middle message list
func buildMessageListPane() fyne.CanvasObject {
	// Mock messages
	messages := []Message{
		{
			ID:      "1",
			From:    "Sarah Johnson",
			Subject: "Q4 Budget Review Meeting",
			Preview: "Hi team, I'd like to schedule our quarterly budget review for next Tuesday at 2pm...",
			Date:    "10:32 AM",
			Unread:  true,
			Starred: true,
			Account: "work@company.com",
		},
		{
			ID:      "2",
			From:    "GitHub",
			Subject: "Your weekly report",
			Preview: "Here's a summary of your activity on GitHub this week. You pushed 12 commits to...",
			Date:    "9:15 AM",
			Unread:  true,
			Starred: false,
			Account: "personal@gmail.com",
		},
		{
			ID:      "3",
			From:    "alice@msgs.global",
			Subject: "Re: AfterSMTP Protocol Discussion",
			Preview: "Thanks for the detailed explanation. The E2E encryption implementation looks solid...",
			Date:    "Yesterday",
			Unread:  false,
			Starred: false,
			Account: "AfterSMTP",
		},
		{
			ID:      "4",
			From:    "Marketing Team",
			Subject: "New campaign launch - Action required",
			Preview: "The new product campaign is ready to launch. Please review the attached materials...",
			Date:    "Yesterday",
			Unread:  false,
			Starred: false,
			Account: "work@company.com",
		},
		{
			ID:      "5",
			From:    "LinkedIn",
			Subject: "You appeared in 47 searches this week",
			Preview: "Your profile is gaining traction! People are finding you through these keywords...",
			Date:    "Dec 14",
			Unread:  false,
			Starred: false,
			Account: "personal@gmail.com",
		},
	}

	messageList := widget.NewList(
		func() int {
			return len(messages)
		},
		func() fyne.CanvasObject {
			// Create a complex message item template
			unreadIndicator := canvas.NewCircle(theme.PrimaryColor())
			unreadIndicator.Resize(fyne.NewSize(8, 8))

			starIcon := widget.NewLabel("⭐")
			fromLabel := widget.NewLabelWithStyle("Sender Name", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			dateLabel := widget.NewLabel("Date")
			subjectLabel := widget.NewLabelWithStyle("Subject", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			previewLabel := widget.NewLabel("Preview text...")
			accountBadge := widget.NewLabel("Account")

			topRow := container.NewHBox(
				unreadIndicator,
				starIcon,
				fromLabel,
				layout.NewSpacer(),
				accountBadge,
				dateLabel,
			)

			return container.NewVBox(
				topRow,
				subjectLabel,
				previewLabel,
				widget.NewSeparator(),
			)
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			if id >= len(messages) {
				return
			}

			msg := messages[id]
			vbox := o.(*fyne.Container)
			topRow := vbox.Objects[0].(*fyne.Container)

			unreadDot := topRow.Objects[0].(*canvas.Circle)
			starIcon := topRow.Objects[1].(*widget.Label)
			fromLabel := topRow.Objects[2].(*widget.Label)
			accountBadge := topRow.Objects[4].(*widget.Label)
			dateLabel := topRow.Objects[5].(*widget.Label)

			subjectLabel := vbox.Objects[1].(*widget.Label)
			previewLabel := vbox.Objects[2].(*widget.Label)

			// Update unread indicator
			if msg.Unread {
				unreadDot.Show()
				fromLabel.TextStyle = fyne.TextStyle{Bold: true}
				subjectLabel.TextStyle = fyne.TextStyle{Bold: true}
			} else {
				unreadDot.Hide()
				fromLabel.TextStyle = fyne.TextStyle{Bold: false}
				subjectLabel.TextStyle = fyne.TextStyle{Bold: false}
			}

			// Update star
			if msg.Starred {
				starIcon.SetText("⭐")
			} else {
				starIcon.SetText("☆")
			}

			fromLabel.SetText(msg.From)
			dateLabel.SetText(msg.Date)
			subjectLabel.SetText(msg.Subject)
			previewLabel.SetText(msg.Preview)
			accountBadge.SetText(fmt.Sprintf("📧 %s", msg.Account))

			fromLabel.Refresh()
			subjectLabel.Refresh()
		},
	)

	listHeader := container.NewHBox(
		widget.NewLabelWithStyle("Inbox", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		widget.NewLabel("42 unread"),
	)

	sortBar := container.NewHBox(
		widget.NewButton("All", func() {}),
		widget.NewButton("Unread", func() {}),
		widget.NewButton("Starred", func() {}),
		layout.NewSpacer(),
		widget.NewLabel("Sort by:"),
		widget.NewSelect([]string{"Date", "Sender", "Subject"}, nil),
	)

	return container.NewBorder(
		container.NewVBox(listHeader, sortBar, widget.NewSeparator()),
		nil, nil, nil,
		messageList,
	)
}

// buildMessageViewPane creates the right message preview/reading pane
func buildMessageViewPane() fyne.CanvasObject {
	// Message header
	subjectLabel := widget.NewLabelWithStyle("Q4 Budget Review Meeting", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	subjectLabel.Wrapping = fyne.TextWrapWord

	fromLabel := widget.NewLabel("From: Sarah Johnson <sarah.johnson@company.com>")
	toLabel := widget.NewLabel("To: me, team@company.com")
	dateLabel := widget.NewLabel("Date: Monday, December 16, 2024 at 10:32 AM")

	accountBadge := widget.NewLabel("📧 work@company.com (IMAP)")
	securityBadge := widget.NewLabel("🔒 TLS Encrypted")

	headerInfo := container.NewVBox(
		subjectLabel,
		widget.NewSeparator(),
		fromLabel,
		toLabel,
		dateLabel,
		container.NewHBox(accountBadge, securityBadge),
		widget.NewSeparator(),
	)

	// Action buttons
	replyBtn := widget.NewButton("Reply", func() {})
	replyAllBtn := widget.NewButton("Reply All", func() {})
	forwardBtn := widget.NewButton("Forward", func() {})
	archiveBtn := widget.NewButton("Archive", func() {})
	deleteBtn := widget.NewButton("Delete", func() {})
	deleteBtn.Importance = widget.DangerImportance

	actions := container.NewHBox(
		replyBtn,
		replyAllBtn,
		forwardBtn,
		layout.NewSpacer(),
		archiveBtn,
		deleteBtn,
	)

	// Message body
	messageBody := widget.NewMultiLineEntry()
	messageBody.Disable()
	messageBody.Wrapping = fyne.TextWrapWord
	messageBody.SetText(`Hi team,

I'd like to schedule our quarterly budget review for next Tuesday at 2pm in the main conference room.

Please review the attached financial reports before the meeting and come prepared with:
- Department spending analysis
- Q1 2025 budget proposals
- Any questions or concerns

The meeting agenda:
1. Q4 performance review (15 min)
2. Department presentations (30 min)
3. Q1 planning discussion (20 min)
4. Open questions (10 min)

Let me know if you have any conflicts with this time.

Best regards,
Sarah Johnson
Financial Director`)

	// Attachments
	attachmentsList := widget.NewLabel("📎 Attachments: Q4_Report.pdf (2.3 MB), Budget_Template.xlsx (156 KB)")

	messageContent := container.NewBorder(
		container.NewVBox(headerInfo, actions, widget.NewSeparator()),
		container.NewVBox(widget.NewSeparator(), attachmentsList),
		nil, nil,
		messageBody,
	)

	return messageContent
}

// buildStatusBar creates the bottom status bar
func buildStatusBar() fyne.CanvasObject {
	connectionStatus := widget.NewLabel("🟢 Connected")
	syncStatus := widget.NewLabel("Last sync: 2 minutes ago")
	accountInfo := widget.NewLabel("3 accounts • 47 unread")

	return container.NewBorder(
		widget.NewSeparator(),
		nil, nil, nil,
		container.NewHBox(
			connectionStatus,
			widget.NewLabel("•"),
			syncStatus,
			layout.NewSpacer(),
			accountInfo,
		),
	)
}

// Legacy function for backwards compatibility
func buildFoldersTab() fyne.CanvasObject {
	// Fetch from local HTTP loopback API running on :4460
	messagePreview := widget.NewMultiLineEntry()
	messagePreview.Disable()
	messagePreview.SetText("Click 'Sync with meowmaild' to connect to the background service.")

	refreshBtn := widget.NewButton("Sync with meowmaild", func() {
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get("http://127.0.0.1:4460/api/folders")
		if err == nil {
			defer resp.Body.Close()
			var data map[string][]string
			if json.NewDecoder(resp.Body).Decode(&data) == nil {
				messagePreview.SetText("Successfully communicated with meowmaild background service.")
			}
		} else {
			messagePreview.SetText(fmt.Sprintf("Failed to reach meowmaild HTTP loopback: %v\nMake sure meowmaild is running.", err))
		}
	})

	return container.NewVBox(
		widget.NewLabel("Use the 'Mail' tab for the full email experience"),
		refreshBtn,
		messagePreview,
	)
}
