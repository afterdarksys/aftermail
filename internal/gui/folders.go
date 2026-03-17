package gui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
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
	Body     string
	Unread   bool
	Starred  bool
	Account  string
	To       string
	Attachments string
	Category string
	CategoryConfidence float64
	IsVIP    bool
	IsMuted  bool
}

// mailState holds the application state
type mailState struct {
	messages        []Message
	selectedMessage *Message
	selectedFolder  string
	window          fyne.Window
	messageViewer   *fyne.Container
	messageList     *widget.List
	statusBar       *fyne.Container
}

// buildMailView creates the professional three-pane email interface
func buildMailView(w fyne.Window) fyne.CanvasObject {
	// Initialize state with mock messages
	state := &mailState{
		messages: getMockMessages(),
		window:   w,
		selectedFolder: "inbox",
	}

	// Build components with state
	toolbar := buildToolbar(state)
	searchBar := buildSearchBar(state)
	folderPane := buildFolderPane(state)
	messageListPane := buildMessageListPane(state)
	messageViewPane := buildMessageViewPane(state)
	statusBar := buildStatusBar(state)

	// Store references for updates
	state.statusBar = statusBar.(*fyne.Container)

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

// getMockMessages returns sample messages for testing
func getMockMessages() []Message {
	return []Message{
		{
			ID:      "1",
			From:    "Sarah Johnson <sarah.johnson@company.com>",
			Subject: "Q4 Budget Review Meeting",
			Preview: "Hi team, I'd like to schedule our quarterly budget review for next Tuesday at 2pm...",
			Date:    "10:32 AM",
			Unread:  true,
			Starred: true,
			Account: "work@company.com",
			To:      "me, team@company.com",
			Body: `Hi team,

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
Financial Director`,
			Attachments: "Q4_Report.pdf (2.3 MB), Budget_Template.xlsx (156 KB)",
			Category: "Work",
			CategoryConfidence: 0.95,
			IsVIP: true,
		},
		{
			ID:      "2",
			From:    "GitHub <noreply@github.com>",
			Subject: "Your weekly report",
			Preview: "Here's a summary of your activity on GitHub this week. You pushed 12 commits to...",
			Date:    "9:15 AM",
			Unread:  true,
			Starred: false,
			Account: "personal@gmail.com",
			To:      "you@example.com",
			Body: `Here's a summary of your activity on GitHub this week:

- 12 commits pushed to meowmail repository
- 3 pull requests reviewed
- 5 issues closed

Keep up the great work!`,
			Attachments: "",
			Category: "Newsletters",
			CategoryConfidence: 0.82,
			IsMuted: true,
		},
		{
			ID:      "3",
			From:    "Alice <alice@msgs.global>",
			Subject: "Re: AfterSMTP Protocol Discussion",
			Preview: "Thanks for the detailed explanation. The E2E encryption implementation looks solid...",
			Date:    "Yesterday",
			Unread:  false,
			Starred: false,
			Account: "AfterSMTP",
			To:      "ryan@msgs.global",
			Body: `Thanks for the detailed explanation. The E2E encryption implementation looks solid.

I've reviewed the Protobuf schema and the key exchange mechanism. Everything looks good to me.

One question: How are you handling key rotation for long-lived conversations?

Best,
Alice`,
			Attachments: "",
			Category: "Work",
			CategoryConfidence: 0.88,
		},
		{
			ID:      "4",
			From:    "Marketing Team <marketing@company.com>",
			Subject: "New campaign launch - Action required",
			Preview: "The new product campaign is ready to launch. Please review the attached materials...",
			Date:    "Yesterday",
			Unread:  false,
			Starred: false,
			Account: "work@company.com",
			To:      "team@company.com",
			Body: `The new product campaign is ready to launch. Please review the attached materials and provide your feedback by EOD Friday.

We need everyone's approval before we can proceed.

Thanks!`,
			Attachments: "Campaign_Brief.pdf (1.2 MB)",
			Category: "Work",
			CategoryConfidence: 0.91,
		},
		{
			ID:      "5",
			From:    "LinkedIn <messages-noreply@linkedin.com>",
			Subject: "You appeared in 47 searches this week",
			Preview: "Your profile is gaining traction! People are finding you through these keywords...",
			Date:    "Dec 14",
			Unread:  false,
			Starred: false,
			Account: "personal@gmail.com",
			To:      "you@example.com",
			Body: `Your profile is gaining traction! People are finding you through these keywords:

- Email architecture
- Distributed systems
- Blockchain protocols

Consider updating your profile to highlight these skills.`,
			Attachments: "",
			Category: "Social",
			CategoryConfidence: 0.94,
		},
	}
}

// buildToolbar creates the main toolbar with action buttons
func buildToolbar(state *mailState) fyne.CanvasObject {
	newMailBtn := widget.NewButton("New", func() {
		dialog.ShowInformation("New Message", "Opening composer...\n(Switch to Composer tab)", state.window)
	})
	newMailBtn.Importance = widget.HighImportance

	replyBtn := widget.NewButton("Reply", func() {
		if state.selectedMessage != nil {
			dialog.ShowInformation("Reply", fmt.Sprintf("Replying to: %s\n\nSubject: Re: %s", state.selectedMessage.From, state.selectedMessage.Subject), state.window)
		} else {
			dialog.ShowInformation("No Selection", "Please select a message first", state.window)
		}
	})

	replyAllBtn := widget.NewButton("Reply All", func() {
		if state.selectedMessage != nil {
			dialog.ShowInformation("Reply All", fmt.Sprintf("Replying to all recipients of:\n%s", state.selectedMessage.Subject), state.window)
		} else {
			dialog.ShowInformation("No Selection", "Please select a message first", state.window)
		}
	})

	forwardBtn := widget.NewButton("Forward", func() {
		if state.selectedMessage != nil {
			dialog.ShowInformation("Forward", fmt.Sprintf("Forwarding message:\n%s", state.selectedMessage.Subject), state.window)
		} else {
			dialog.ShowInformation("No Selection", "Please select a message first", state.window)
		}
	})

	deleteBtn := widget.NewButton("Delete", func() {
		if state.selectedMessage != nil {
			dialog.ShowConfirm("Delete Message",
				fmt.Sprintf("Are you sure you want to delete:\n%s", state.selectedMessage.Subject),
				func(confirmed bool) {
					if confirmed {
						// Remove from list
						for i, msg := range state.messages {
							if msg.ID == state.selectedMessage.ID {
								state.messages = append(state.messages[:i], state.messages[i+1:]...)
								break
							}
						}
						state.selectedMessage = nil
						state.messageList.Refresh()
						updateMessageViewer(state, nil)
						dialog.ShowInformation("Deleted", "Message moved to trash", state.window)
					}
				}, state.window)
		} else {
			dialog.ShowInformation("No Selection", "Please select a message first", state.window)
		}
	})
	deleteBtn.Importance = widget.DangerImportance

	archiveBtn := widget.NewButton("Archive", func() {
		if state.selectedMessage != nil {
			dialog.ShowInformation("Archive", fmt.Sprintf("Archived:\n%s", state.selectedMessage.Subject), state.window)
		} else {
			dialog.ShowInformation("No Selection", "Please select a message first", state.window)
		}
	})

	markReadBtn := widget.NewButton("Mark Read", func() {
		if state.selectedMessage != nil {
			state.selectedMessage.Unread = false
			state.messageList.Refresh()
			dialog.ShowInformation("Marked as Read", state.selectedMessage.Subject, state.window)
		} else {
			dialog.ShowInformation("No Selection", "Please select a message first", state.window)
		}
	})

	syncBtn := widget.NewButton("Sync", func() {
		dialog.ShowInformation("Syncing", "Connecting to mail servers...\n\n✓ work@company.com\n✓ personal@gmail.com\n✓ AfterSMTP gateway\n\nAll accounts synchronized!", state.window)
	})

	exportBtn := widget.NewButton("Export", func() {
		if state.selectedMessage != nil {
			exportOptions := widget.NewSelect([]string{"To .eml file", "To Compressed Maildir (.tar.gz)"}, func(s string) {
				if s == "To .eml file" {
					exportEml(state)
				} else if s == "To Compressed Maildir (.tar.gz)" {
					exportMaildir(state)
				}
			})
			dialog.ShowCustom("Export Selected Email", "Cancel", container.NewVBox(
				widget.NewLabel("Choose export format:"),
				exportOptions,
			), state.window)
		} else {
			dialog.ShowInformation("No Selection", "Please select a message first to export", state.window)
		}
	})

	spacer := layout.NewSpacer()

	settingsBtn := widget.NewButton("Settings", func() {
		showSettingsDialog(state)
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
		exportBtn,
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
func buildSearchBar(state *mailState) fyne.CanvasObject {
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search mail (subject, sender, body...)")
	searchEntry.OnChanged = func(query string) {
		if query == "" {
			// Reset to all messages
			state.messages = getMockMessages()
			state.messageList.Refresh()
			return
		}
		
		query = strings.ToLower(query)
		filtered := []Message{}
		for _, msg := range getMockMessages() {
			if strings.Contains(strings.ToLower(msg.Subject), query) || 
			   strings.Contains(strings.ToLower(msg.From), query) || 
			   strings.Contains(strings.ToLower(msg.Body), query) {
				filtered = append(filtered, msg)
			}
		}
		state.messages = filtered
		state.messageList.Refresh()
	}

	filterBtn := widget.NewButton("Filter", func() {
		dialog.ShowInformation("Filters", "Filter options:\n• Unread only\n• Starred\n• Has attachments\n• Date range\n• Account", state.window)
	})

	searchBar := container.NewBorder(
		nil, nil,
		nil, filterBtn,
		searchEntry,
	)

	return container.NewPadded(searchBar)
}

// buildFolderPane creates the left folder navigation
func buildFolderPane(state *mailState) fyne.CanvasObject {
	// Account selector
	accountSelect := widget.NewSelect(
		[]string{"All Accounts", "work@company.com", "personal@gmail.com", "did:aftersmtp:msgs.global:ryan"},
		func(s string) {
			dialog.ShowInformation("Account Filter", fmt.Sprintf("Filtering messages for: %s", s), state.window)
			
			if s == "All Accounts" {
				state.messages = getMockMessages()
			} else {
				filtered := []Message{}
				for _, msg := range getMockMessages() {
					if msg.Account == s {
						filtered = append(filtered, msg)
					}
				}
				state.messages = filtered
			}

			if state.messageList != nil {
				state.messageList.Refresh()
			}
		},
	)
	accountSelect.SetSelected("All Accounts")

	// Folder tree
	folderTree := widget.NewTree(
		func(id widget.TreeNodeID) []widget.TreeNodeID {
			switch id {
			case "":
				return []widget.TreeNodeID{"smart", "favorites", "folders", "accounts"}
			case "smart":
				return []widget.TreeNodeID{"smart-today", "smart-vips", "smart-muted", "smart-attachments"}
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
				"smart":            "⚙️ Smart Mailboxes",
				"smart-today":      "📅 Today",
				"smart-vips":       "👑 VIPs",
				"smart-muted":      "🔕 Muted Threads",
				"smart-attachments": "📎 Has Attachments",
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
	folderTree.OpenBranch("smart")
	folderTree.OpenBranch("favorites")
	folderTree.OpenBranch("folders")

	folderTree.OnSelected = func(id widget.TreeNodeID) {
		filtered := []Message{}
		for _, msg := range getMockMessages() {
			include := false
			switch id {
			case "smart-today":
				include = strings.Contains(msg.Date, "AM") || strings.Contains(msg.Date, "PM")
			case "smart-vips":
				include = msg.IsVIP
			case "smart-muted":
				include = msg.IsMuted
			case "smart-attachments":
				include = msg.Attachments != ""
			case "fav-unread", "unread":
				include = msg.Unread
			case "fav-starred", "starred":
				include = msg.Starred
			default:
				// Fallback to show all mock messages for standard folders just to show the UI works
				include = true
			}
			if include {
				filtered = append(filtered, msg)
			}
		}
		state.messages = filtered
		if state.messageList != nil {
			state.messageList.Refresh()
		}
	}

	folderHeader := widget.NewLabelWithStyle("Navigation", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	return container.NewBorder(
		container.NewVBox(folderHeader, accountSelect, widget.NewSeparator()),
		nil, nil, nil,
		folderTree,
	)
}

// buildMessageListPane creates the middle message list
func buildMessageListPane(state *mailState) fyne.CanvasObject {
	messageList := widget.NewList(
		func() int {
			return len(state.messages)
		},
		func() fyne.CanvasObject {
			// Create a complex message item template
			unreadIndicator := canvas.NewCircle(theme.PrimaryColor())
			unreadIndicator.Resize(fyne.NewSize(8, 8))

			starIcon := widget.NewLabel("⭐")
			vipIcon := widget.NewLabel("👑")
			fromLabel := widget.NewLabelWithStyle("Sender Name", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			dateLabel := widget.NewLabel("Date")
			subjectLabel := widget.NewLabelWithStyle("Subject", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			previewLabel := widget.NewLabel("Preview text...")
			mutedIcon := widget.NewLabel("🔕")
			accountBadge := widget.NewLabel("Account")
			categoryBadge := widget.NewLabel("Category")

			topRow := container.NewHBox(
				unreadIndicator,
				starIcon,
				vipIcon,
				fromLabel,
				layout.NewSpacer(),
				mutedIcon,
				categoryBadge,
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
			if id >= len(state.messages) {
				return
			}

			msg := state.messages[id]
			vbox := o.(*fyne.Container)
			topRow := vbox.Objects[0].(*fyne.Container)

			unreadDot := topRow.Objects[0].(*canvas.Circle)
			starIcon := topRow.Objects[1].(*widget.Label)
			vipIcon := topRow.Objects[2].(*widget.Label)
			fromLabel := topRow.Objects[3].(*widget.Label)
			mutedIcon := topRow.Objects[5].(*widget.Label)
			categoryBadge := topRow.Objects[6].(*widget.Label)
			accountBadge := topRow.Objects[7].(*widget.Label)
			dateLabel := topRow.Objects[8].(*widget.Label)

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

			if msg.IsVIP {
				vipIcon.SetText("👑")
				vipIcon.Show()
			} else {
				vipIcon.Hide()
			}

			if msg.IsMuted {
				mutedIcon.SetText("🔕")
				mutedIcon.Show()
			} else {
				mutedIcon.Hide()
			}

			// Update category badge with color coding
			categoryIcons := map[string]string{
				"Work":        "💼",
				"Personal":    "🏠",
				"Finance":     "💰",
				"Shopping":    "🛒",
				"Social":      "👥",
				"Newsletters": "📰",
				"Promotions":  "🏷️",
				"Spam":        "🚫",
			}

			if icon, ok := categoryIcons[msg.Category]; ok {
				categoryBadge.SetText(fmt.Sprintf("%s %s", icon, msg.Category))
			} else {
				categoryBadge.SetText(msg.Category)
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

	// Add selection handler
	messageList.OnSelected = func(id widget.ListItemID) {
		if id >= 0 && id < len(state.messages) {
			state.selectedMessage = &state.messages[id]
			updateMessageViewer(state, state.selectedMessage)
		}
	}

	// Store reference in state
	state.messageList = messageList

	listHeader := container.NewHBox(
		widget.NewLabelWithStyle("Inbox", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		widget.NewLabel("42 unread"),
	)

	// Category filter buttons
	categoryFilterBar := container.NewHBox(
		widget.NewLabel("Categories:"),
		widget.NewButton("All", func() {
			state.selectedFolder = "all"
			state.messageList.Refresh()
		}),
		widget.NewButton("💼 Work", func() {
			state.selectedFolder = "Work"
			state.messageList.Refresh()
		}),
		widget.NewButton("💰 Finance", func() {
			state.selectedFolder = "Finance"
			state.messageList.Refresh()
		}),
		widget.NewButton("👥 Social", func() {
			state.selectedFolder = "Social"
			state.messageList.Refresh()
		}),
		widget.NewButton("📰 News", func() {
			state.selectedFolder = "Newsletters"
			state.messageList.Refresh()
		}),
	)

	sortBar := container.NewHBox(
		widget.NewButton("All", func() {}),
		widget.NewButton("Unread", func() {}),
		widget.NewButton("Starred", func() {}),
		layout.NewSpacer(),
		widget.NewLabel("Sort by:"),
		widget.NewSelect([]string{"Date", "Sender", "Subject", "Category"}, nil),
	)

	return container.NewBorder(
		container.NewVBox(listHeader, categoryFilterBar, sortBar, widget.NewSeparator()),
		nil, nil, nil,
		messageList,
	)
}

// buildMessageViewPane creates the right message preview/reading pane
func buildMessageViewPane(state *mailState) fyne.CanvasObject {
	// Create placeholder content
	placeholder := container.NewCenter(
		widget.NewLabel("Select a message to view"),
	)

	// Store reference for updates
	state.messageViewer = placeholder

	return placeholder
}

// updateMessageViewer updates the message viewer with the selected message
func updateMessageViewer(state *mailState, msg *Message) {
	if msg == nil {
		// Show placeholder
		state.messageViewer.Objects = []fyne.CanvasObject{
			container.NewCenter(widget.NewLabel("Select a message to view")),
		}
		state.messageViewer.Refresh()
		return
	}

	// Message header
	subjectLabel := widget.NewLabelWithStyle(msg.Subject, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	subjectLabel.Wrapping = fyne.TextWrapWord

	fromLabel := widget.NewLabel(fmt.Sprintf("From: %s", msg.From))
	toLabel := widget.NewLabel(fmt.Sprintf("To: %s", msg.To))
	dateLabel := widget.NewLabel(fmt.Sprintf("Date: %s", msg.Date))

	accountBadge := widget.NewLabel(fmt.Sprintf("📧 %s", msg.Account))
	securityBadge := widget.NewLabel("🔒 TLS Encrypted")
	if msg.Account == "AfterSMTP" {
		securityBadge.SetText("🔐 E2E Encrypted (AfterSMTP)")
	}

	headerInfo := container.NewVBox(
		subjectLabel,
		widget.NewSeparator(),
		fromLabel,
		toLabel,
		dateLabel,
		container.NewHBox(accountBadge, securityBadge),
		widget.NewSeparator(),
	)

	// Action buttons (these will use state.selectedMessage)
	replyBtn := widget.NewButton("Reply", func() {
		dialog.ShowInformation("Reply", fmt.Sprintf("Replying to: %s", msg.From), state.window)
	})
	replyAllBtn := widget.NewButton("Reply All", func() {
		dialog.ShowInformation("Reply All", "Replying to all recipients", state.window)
	})
	forwardBtn := widget.NewButton("Forward", func() {
		dialog.ShowInformation("Forward", fmt.Sprintf("Forwarding: %s", msg.Subject), state.window)
	})
	muteBtn := widget.NewButton("Mute", func() {
		dialog.ShowInformation("Mute Thread", "Thread muted", state.window)
	})
	archiveBtn := widget.NewButton("Archive", func() {
		dialog.ShowInformation("Archive", "Message archived", state.window)
	})
	deleteBtn := widget.NewButton("Delete", func() {
		dialog.ShowConfirm("Delete", "Move this message to trash?", func(confirmed bool) {
			if confirmed {
				// Remove from list
				for i, m := range state.messages {
					if m.ID == msg.ID {
						state.messages = append(state.messages[:i], state.messages[i+1:]...)
						break
					}
				}
				state.messageList.Refresh()
				updateMessageViewer(state, nil)
			}
		}, state.window)
	})
	deleteBtn.Importance = widget.DangerImportance

	actions := container.NewHBox(
		replyBtn,
		replyAllBtn,
		forwardBtn,
		layout.NewSpacer(),
		muteBtn,
		archiveBtn,
		deleteBtn,
	)

	// Message body
	messageBody := widget.NewMultiLineEntry()
	messageBody.Disable()
	messageBody.Wrapping = fyne.TextWrapWord
	messageBody.SetText(msg.Body)

	// Attachments
	var attachmentsList *widget.Label
	if msg.Attachments != "" {
		attachmentsList = widget.NewLabel(fmt.Sprintf("📎 Attachments: %s", msg.Attachments))
	} else {
		attachmentsList = widget.NewLabel("No attachments")
	}

	messageContent := container.NewBorder(
		container.NewVBox(headerInfo, actions, widget.NewSeparator()),
		container.NewVBox(widget.NewSeparator(), attachmentsList),
		nil, nil,
		messageBody,
	)

	state.messageViewer.Objects = []fyne.CanvasObject{messageContent}
	state.messageViewer.Refresh()
}

// buildStatusBar creates the bottom status bar
func buildStatusBar(state *mailState) fyne.CanvasObject {
	connectionStatus := widget.NewLabel("🟢 Connected")
	syncStatus := widget.NewLabel("Last sync: 2 minutes ago")

	// Count unread messages
	unreadCount := 0
	for _, msg := range state.messages {
		if msg.Unread {
			unreadCount++
		}
	}
	accountInfo := widget.NewLabel(fmt.Sprintf("3 accounts • %d unread", unreadCount))

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

// showSettingsDialog displays a comprehensive settings dialog
func showSettingsDialog(state *mailState) {
	// Create tabs for different settings categories
	generalSettings := buildGeneralSettings(state)
	accountsSettings := buildAccountsSettings(state)
	composerSettings := buildComposerSettings(state)
	notificationSettings := buildNotificationSettings(state)
	advancedSettings := buildAdvancedSettings(state)

	tabs := container.NewAppTabs(
		container.NewTabItem("General", generalSettings),
		container.NewTabItem("Accounts", accountsSettings),
		container.NewTabItem("Composer", composerSettings),
		container.NewTabItem("Notifications", notificationSettings),
		container.NewTabItem("Advanced", advancedSettings),
	)

	// Create dialog
	settingsDialog := dialog.NewCustom("Settings", "Close", tabs, state.window)
	settingsDialog.Resize(fyne.NewSize(700, 500))
	settingsDialog.Show()
}

// buildGeneralSettings creates general settings panel
func buildGeneralSettings(state *mailState) fyne.CanvasObject {
	themeSelect := widget.NewSelect([]string{"System", "Light", "Dark"}, func(s string) {
		dialog.ShowInformation("Theme", fmt.Sprintf("Theme changed to: %s", s), state.window)
	})
	themeSelect.SetSelected("System")

	languageSelect := widget.NewSelect([]string{"English", "Spanish", "French", "German", "Chinese"}, nil)
	languageSelect.SetSelected("English")

	spellCheckEnable := widget.NewCheck("Enable spell check", func(checked bool) {})
	spellCheckEnable.SetChecked(true)

	grammarCheckEnable := widget.NewCheck("Enable grammar check", func(checked bool) {})
	grammarCheckEnable.SetChecked(true)

	autoSaveDrafts := widget.NewCheck("Auto-save drafts", func(checked bool) {})
	autoSaveDrafts.SetChecked(true)

	startupCheck := widget.NewCheck("Launch on startup", func(checked bool) {})

	return container.NewVBox(
		widget.NewLabelWithStyle("Appearance", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Theme", themeSelect),
			widget.NewFormItem("Language", languageSelect),
		),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Editor", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		spellCheckEnable,
		grammarCheckEnable,
		autoSaveDrafts,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("System", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		startupCheck,
	)
}

// buildAccountsSettings creates account settings panel
func buildAccountsSettings(state *mailState) fyne.CanvasObject {
	accountsList := widget.NewList(
		func() int { return 3 },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Account"),
				layout.NewSpacer(),
				widget.NewButton("Edit", func() {}),
			)
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			accounts := []string{
				"📧 work@company.com (IMAP)",
				"📧 personal@gmail.com (Gmail)",
				"🔐 did:aftersmtp:msgs.global:ryan (AfterSMTP)",
			}
			c := o.(*fyne.Container)
			c.Objects[0].(*widget.Label).SetText(accounts[id])
			c.Objects[2].(*widget.Button).OnTapped = func() {
				dialog.ShowInformation("Edit Account", accounts[id], state.window)
			}
		},
	)

	addBtn := widget.NewButton("Add Account", func() {
		dialog.ShowInformation("Add Account", "Account types:\n• IMAP/SMTP\n• Gmail (OAuth2)\n• Outlook (Graph API)\n• AfterSMTP (Encrypted)", state.window)
	})
	addBtn.Importance = widget.HighImportance

	return container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Configured Accounts"),
			addBtn,
		),
		nil, nil, nil,
		accountsList,
	)
}

// buildComposerSettings creates composer settings panel
func buildComposerSettings(state *mailState) fyne.CanvasObject {
	defaultFormatSelect := widget.NewSelect([]string{"Plain Text", "HTML", "Markdown"}, nil)
	defaultFormatSelect.SetSelected("Plain Text")

	fontSelect := widget.NewSelect([]string{"System Default", "Arial", "Helvetica", "Times New Roman", "Courier"}, nil)
	fontSelect.SetSelected("System Default")

	fontSizeSelect := widget.NewSelect([]string{"10", "11", "12", "14", "16", "18"}, nil)
	fontSizeSelect.SetSelected("12")

	signatureEntry := widget.NewMultiLineEntry()
	signatureEntry.SetPlaceHolder("Enter your email signature...")
	signatureEntry.Wrapping = fyne.TextWrapWord

	enableSignature := widget.NewCheck("Include signature in new messages", func(checked bool) {})
	enableSignature.SetChecked(true)

	requestReadReceipt := widget.NewCheck("Request read receipts by default", func(checked bool) {})

	return container.NewVBox(
		widget.NewLabelWithStyle("Default Format", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Message Format", defaultFormatSelect),
			widget.NewFormItem("Font", fontSelect),
			widget.NewFormItem("Font Size", fontSizeSelect),
		),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Signature", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		enableSignature,
		signatureEntry,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Options", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		requestReadReceipt,
	)
}

// buildNotificationSettings creates notification settings panel
func buildNotificationSettings(state *mailState) fyne.CanvasObject {
	desktopNotifications := widget.NewCheck("Show desktop notifications", func(checked bool) {})
	desktopNotifications.SetChecked(true)

	soundNotifications := widget.NewCheck("Play sound on new mail", func(checked bool) {})
	soundNotifications.SetChecked(true)

	badgeCount := widget.NewCheck("Show unread count badge", func(checked bool) {})
	badgeCount.SetChecked(true)

	notifyAllAccounts := widget.NewCheck("Notify for all accounts", func(checked bool) {})
	notifyAllAccounts.SetChecked(true)

	notifyImportantOnly := widget.NewCheck("Only notify for important messages", func(checked bool) {})

	return container.NewVBox(
		widget.NewLabelWithStyle("Notification Preferences", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		desktopNotifications,
		soundNotifications,
		badgeCount,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Filters", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		notifyAllAccounts,
		notifyImportantOnly,
	)
}

// buildAdvancedSettings creates advanced settings panel
func buildAdvancedSettings(state *mailState) fyne.CanvasObject {
	syncIntervalSelect := widget.NewSelect([]string{"1 minute", "5 minutes", "15 minutes", "30 minutes", "Manual"}, nil)
	syncIntervalSelect.SetSelected("5 minutes")

	cacheEnabledCheck := widget.NewCheck("Enable message caching", func(checked bool) {})
	cacheEnabledCheck.SetChecked(true)

	debugLoggingCheck := widget.NewCheck("Enable debug logging", func(checked bool) {})

	enableScriptingCheck := widget.NewCheck("Enable mail scripting API (Starlark)", func(checked bool) {})
	enableScriptingCheck.SetChecked(true)

	scriptPathEntry := widget.NewEntry()
	scriptPathEntry.SetPlaceHolder("~/.meowmail/scripts")
	scriptPathEntry.SetText("~/.meowmail/scripts")

	helpScriptBtn := widget.NewButton("Script Documentation", func() {
		showScriptHelpDialog(state)
	})

	clearCacheBtn := widget.NewButton("Clear Cache", func() {
		dialog.ShowConfirm("Clear Cache", "Delete all cached messages and attachments?", func(confirmed bool) {
			if confirmed {
				dialog.ShowInformation("Cache Cleared", "All cached data has been deleted", state.window)
			}
		}, state.window)
	})

	exportDataBtn := widget.NewButton("Export Data", func() {
		dialog.ShowInformation("Export", "Export all messages to:\n• MBOX format\n• EML files\n• JSON archive", state.window)
	})

	return container.NewVBox(
		widget.NewLabelWithStyle("Synchronization", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Sync Interval", syncIntervalSelect),
		),
		cacheEnabledCheck,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Scripting", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		enableScriptingCheck,
		widget.NewForm(
			widget.NewFormItem("Script Directory", scriptPathEntry),
		),
		helpScriptBtn,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Debugging", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		debugLoggingCheck,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Data Management", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(clearCacheBtn, exportDataBtn),
	)
}

// showScriptHelpDialog shows scripting API documentation
func showScriptHelpDialog(state *mailState) {
	helpText := `ADS Mail Scripting API (Starlark)

Place .star scripts in ~/.meowmail/scripts to automate email tasks.

Available Functions:
• get_messages(folder) - Get messages from a folder
• send_message(to, subject, body) - Send an email
• move_message(msg_id, folder) - Move a message
• delete_message(msg_id) - Delete a message
• mark_read(msg_id) - Mark as read
• mark_unread(msg_id) - Mark as unread
• add_label(msg_id, label) - Add a label
• search(query) - Search messages

Example Script:
def auto_archive_old():
  msgs = get_messages("inbox")
  for msg in msgs:
    if msg.age_days > 30:
      move_message(msg.id, "archive")

Triggers:
• on_receive - Run when new mail arrives
• on_send - Run before sending
• on_startup - Run when ADS Mail starts

Documentation: https://meowmail.dev/scripting`

	helpContent := widget.NewMultiLineEntry()
	helpContent.SetText(helpText)
	helpContent.Disable()
	helpContent.Wrapping = fyne.TextWrapWord

	dialog.NewCustom("Scripting API Reference", "Close", helpContent, state.window).Show()
}

// Legacy function for backwards compatibility
func buildFoldersTab() fyne.CanvasObject {
	// Fetch from local HTTP loopback API running on :4460
	messagePreview := widget.NewMultiLineEntry()
	messagePreview.Disable()
	messagePreview.SetText("Click 'Sync with aftermaild' to connect to the background service.")

	refreshBtn := widget.NewButton("Sync with aftermaild", func() {
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get("http://127.0.0.1:4460/api/folders")
		if err == nil {
			defer resp.Body.Close()
			var data map[string][]string
			if json.NewDecoder(resp.Body).Decode(&data) == nil {
				messagePreview.SetText("Successfully communicated with aftermaild background service.")
			}
		} else {
			messagePreview.SetText(fmt.Sprintf("Failed to reach aftermaild HTTP loopback: %v\nMake sure aftermaild is running.", err))
		}
	})

	return container.NewVBox(
		widget.NewLabel("Use the 'Mail' tab for the full email experience"),
		refreshBtn,
		messagePreview,
	)
}

func exportEml(state *mailState) {
	msg := state.selectedMessage
	if msg == nil {
		return
	}
	
	dialog.ShowFileSave(func(uc fyne.URIWriteCloser, err error) {
		if err != nil || uc == nil {
			return
		}
		defer uc.Close()

		emlContent := fmt.Sprintf("From: %s\r\nTo: %s\r\nDate: %s\r\nSubject: %s\r\n\r\n%s",
			msg.From, msg.To, msg.Date, msg.Subject, msg.Body)
			
		uc.Write([]byte(emlContent))
		dialog.ShowInformation("Export Successful", "Saved email as "+uc.URI().Name(), state.window)

	}, state.window)
}

func exportMaildir(state *mailState) {
	msg := state.selectedMessage
	if msg == nil {
		return
	}
	
	dialog.ShowFileSave(func(uc fyne.URIWriteCloser, err error) {
		if err != nil || uc == nil {
			return
		}
		defer uc.Close()

		// For demonstration, we simply write a stub message indicating compression
		mockTarGz := fmt.Sprintf("MOCK_TAR_GZ_HEADER\n\n[maildir/new/%s]\nFrom: %s\nSubject: %s\n\n%s", 
			msg.ID, msg.From, msg.Subject, msg.Body)

		uc.Write([]byte(mockTarGz))
		dialog.ShowInformation("Export Successful", "Saved Compressed Maildir to "+uc.URI().Name(), state.window)

	}, state.window)
}
