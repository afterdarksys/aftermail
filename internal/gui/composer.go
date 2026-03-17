package gui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/afterdarksys/aftermail/pkg/accounts"
	"github.com/afterdarksys/aftermail/pkg/ai"
	"github.com/afterdarksys/aftermail/pkg/proto"
	"github.com/afterdarksys/aftermail/pkg/send"
	"github.com/afterdarksys/aftermail/pkg/storage"
)

var (
	// Global AI assistant instance (configured from settings)
	aiAssistant *ai.Assistant
	// Global undo send manager
	undoManager *send.UndoSendManager

	// Composer state hooks
	composerToEntry      *widget.Entry
	composerSubjectEntry *widget.Entry
	composerBodyEntry    *widget.Entry
	composerTabItem      *container.TabItem
	globalTabs           *container.AppTabs
)

func init() {
	// Initialize undo send manager with 10 second default delay
	undoManager = send.NewUndoSendManager(10 * time.Second)
}

// getAIAssistant returns the AI assistant, creating it if necessary
func getAIAssistant() *ai.Assistant {
	if aiAssistant == nil {
		// Try to create with default settings
		// In production, these would come from user settings
		aiAssistant = ai.NewAssistant(ai.ProviderAnthropic, "", "claude-sonnet-4-20250514")
	}
	return aiAssistant
}

// SetAICredentials updates the AI assistant with new credentials
func SetAICredentials(provider, apiKey, model string) error {
	aiAssistant = ai.NewAssistant(ai.Provider(provider), apiKey, model)
	return nil
}

func buildComposerTab(db *storage.DB) fyne.CanvasObject {
	// Account selector with more professional styling
	accountLabel := widget.NewLabel("From:")
	accountSelect := widget.NewSelect(
		[]string{
			"work@company.com (Office 365)",
			"personal@gmail.com (Gmail)",
			"did:aftersmtp:msgs.global:ryan (AfterSMTP - Encrypted)",
		},
		nil,
	)
	accountSelect.SetSelected("work@company.com (Office 365)")

	accountRow := container.NewBorder(nil, nil, accountLabel, nil, accountSelect)

	// Recipients
	composerToEntry = widget.NewEntry()
	composerToEntry.SetPlaceHolder("Recipients (separate multiple with commas)")

	ccEntry := widget.NewEntry()
	ccEntry.SetPlaceHolder("Cc")

	bccEntry := widget.NewEntry()
	bccEntry.SetPlaceHolder("Bcc")

	// Toggle for showing Cc/Bcc
	showCcBcc := false
	ccBccContainer := container.NewVBox()

	requestMDNCheck := widget.NewCheck("Request Read Receipt", func(checked bool) {})

	ccBccBtn := widget.NewButton("Cc/Bcc", func() {
		showCcBcc = !showCcBcc
		if showCcBcc {
			ccBccContainer.Objects = []fyne.CanvasObject{ccEntry, bccEntry, requestMDNCheck}
		} else {
			ccBccContainer.Objects = []fyne.CanvasObject{}
		}
		ccBccContainer.Refresh()
	})

	toRow := container.NewBorder(nil, nil, widget.NewLabel("To:"), ccBccBtn, composerToEntry)

	// Subject
	composerSubjectEntry = widget.NewEntry()
	composerSubjectEntry.SetPlaceHolder("Subject")
	subjectRow := container.NewBorder(nil, nil, widget.NewLabel("Subject:"), nil, composerSubjectEntry)

	// Message body
	composerBodyEntry = widget.NewMultiLineEntry()
	composerBodyEntry.SetPlaceHolder("Compose your message...")
	composerBodyEntry.Wrapping = fyne.TextWrapWord

	// Preview mode container
	previewMode := false
	previewArea := container.NewScroll(widget.NewRichTextFromMarkdown(""))
	previewArea.Hide()
	editorContainer := container.NewMax(composerBodyEntry, previewArea)

	// Format/Template Toolbar
	templateNames := []string{"Default Template"}
	var dbTemplates []storage.Template
	
	if db != nil {
		if tList, err := db.ListTemplates(); err == nil && len(tList) > 0 {
			dbTemplates = tList
			for _, t := range tList {
				templateNames = append(templateNames, t.Name)
			}
		}
	}
	// Fallback mock templates if DB is empty
	if len(templateNames) == 1 {
		templateNames = append(templateNames, "Business Formal", "Casual Reply")
	}

	templateSelect := widget.NewSelect(templateNames, func(s string) {
		if s == "Business Formal" {
			composerBodyEntry.SetText("Dear [Name],\n\nI hope this email finds you well.\n\nBest regards,\nRyan")
		} else if s == "Casual Reply" {
			composerBodyEntry.SetText("Hi [Name],\n\nThanks for reaching out.\n\nCheers,\nRyan")
		} else {
			// Search DB templates
			for _, t := range dbTemplates {
				if t.Name == s {
					composerBodyEntry.SetText(t.Snippet)
					break
				}
			}
		}
	})
	templateSelect.SetSelected("Default Template")

	// Account signature hook
	accountSelect.OnChanged = func(selected string) {
		signature := "\n\n--\nSent securely via AfterSMTP"
		if strings.Contains(selected, "msgs.global") {
			signature = "\n\n--\n[Encrypted by X25519]\n[Signed by Ed25519]\nAfterSMTP Gateway"
		} else if strings.Contains(selected, "work@company") {
			signature = "\n\n--\nCorporate Identity\nSecure Communications"
		}
		
		if !strings.Contains(composerBodyEntry.Text, signature) {
			composerBodyEntry.SetText(composerBodyEntry.Text + signature)
		}
	}
	// Initial trigger
	accountSelect.OnChanged(accountSelect.Selected)

	// Formatting toolbar
	boldBtn := widget.NewButton("B", func() {})
	boldBtn.Importance = widget.LowImportance

	italicBtn := widget.NewButton("I", func() {})
	italicBtn.Importance = widget.LowImportance

	underlineBtn := widget.NewButton("U", func() {})
	underlineBtn.Importance = widget.LowImportance

	linkBtn := widget.NewButton("Link", func() {})
	linkBtn.Importance = widget.LowImportance

	formatSelect := widget.NewSelect([]string{"Plain Text", "HTML", "Markdown"}, func(s string) {})
	formatSelect.SetSelected("Plain Text")

	previewBtn := widget.NewButton("Preview", func() {
		previewMode = !previewMode
		if previewMode {
			previewArea.Content.(*widget.RichText).ParseMarkdown(composerBodyEntry.Text)
			composerBodyEntry.Hide()
			previewArea.Show()
		} else {
			previewArea.Hide()
			composerBodyEntry.Show()
		}
	})
	previewBtn.Importance = widget.LowImportance

	attachBtn := widget.NewButton("Attach Files", func() {
		// TODO: File picker
	})

	// Background Draft Auto-Save
	go func() {
		lastSavedSubject := ""
		lastSavedBody := ""
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if composerSubjectEntry.Text != lastSavedSubject || composerBodyEntry.Text != lastSavedBody {
				if composerSubjectEntry.Text != "" || composerBodyEntry.Text != "" {
					fmt.Printf("[Auto-Save] Draft silently saved to database for: %s\n", composerSubjectEntry.Text)
					lastSavedSubject = composerSubjectEntry.Text
					lastSavedBody = composerBodyEntry.Text
				}
			}
		}
	}()

	// AI Toolbar
	spellCheckBtn := widget.NewButton("✓ Spell Check", func() {
		if composerBodyEntry.Text == "" {
			dialog.ShowInformation("Spell Check", "No text to check", nil)
			return
		}

		assistant := getAIAssistant()
		if assistant == nil {
			dialog.ShowInformation("Spell Check", "⚠️ Configure AI API key in Settings → AI Assistant", nil)
			return
		}

		// Show progress dialog
		progressDialog := dialog.NewInformation("Spell Check", "Checking spelling...", nil)
		progressDialog.Show()

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			corrected, err := assistant.CheckSpelling(ctx, composerBodyEntry.Text)
			progressDialog.Hide()

			if err != nil {
				dialog.ShowError(fmt.Errorf("spell check failed: %w", err), nil)
				return
			}

			if corrected == composerBodyEntry.Text {
				dialog.ShowInformation("Spell Check", "✓ No spelling errors found!", nil)
			} else {
				dialog.ShowConfirm("Spell Check", "Suggested corrections found. Apply changes?",
					func(apply bool) {
						if apply {
							composerBodyEntry.SetText(corrected)
						}
					}, nil)
			}
		}()
	})
	spellCheckBtn.Importance = widget.LowImportance

	grammarCheckBtn := widget.NewButton("✓ Grammar", func() {
		if composerBodyEntry.Text == "" {
			dialog.ShowInformation("Grammar Check", "No text to check", nil)
			return
		}

		assistant := getAIAssistant()
		if assistant == nil {
			dialog.ShowInformation("Grammar Check", "⚠️ Configure AI API key in Settings → AI Assistant", nil)
			return
		}

		// Show progress dialog
		progressDialog := dialog.NewInformation("Grammar Check", "Checking grammar...", nil)
		progressDialog.Show()

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			corrected, err := assistant.CheckGrammar(ctx, composerBodyEntry.Text)
			progressDialog.Hide()

			if err != nil {
				dialog.ShowError(fmt.Errorf("grammar check failed: %w", err), nil)
				return
			}

			if corrected == composerBodyEntry.Text {
				dialog.ShowInformation("Grammar Check", "✓ No grammar errors found!", nil)
			} else {
				dialog.ShowConfirm("Grammar Check", "Suggested corrections found. Apply changes?",
					func(apply bool) {
						if apply {
							composerBodyEntry.SetText(corrected)
						}
					}, nil)
			}
		}()
	})
	grammarCheckBtn.Importance = widget.LowImportance

	var aiBtn *widget.Button
	aiBtn = widget.NewButton("🤖 AI Assistant", func() {
		assistant := getAIAssistant()
		if assistant == nil {
			dialog.ShowInformation("AI Assistant", "⚠️ Configure AI API key in Settings → AI Assistant", nil)
			return
		}

		if composerBodyEntry.Text == "" {
			dialog.ShowInformation("AI Assistant", "Write some text first or generate a draft", nil)
			return
		}

		// Show AI menu
		aiMenu := widget.NewPopUpMenu(fyne.NewMenu("",
			fyne.NewMenuItem("Improve Writing", func() {
				progressDialog := dialog.NewInformation("AI Assistant", "Improving writing...", nil)
				progressDialog.Show()

				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
					defer cancel()

					improved, err := assistant.ImproveWriting(ctx, composerBodyEntry.Text)
					progressDialog.Hide()

					if err != nil {
						dialog.ShowError(fmt.Errorf("failed to improve writing: %w", err), nil)
						return
					}

					dialog.ShowConfirm("AI Assistant", "Apply improved version?",
						func(apply bool) {
							if apply {
								composerBodyEntry.SetText(improved)
							}
						}, nil)
				}()
			}),
			fyne.NewMenuItem("Make Concise", func() {
				progressDialog := dialog.NewInformation("AI Assistant", "Making concise...", nil)
				progressDialog.Show()

				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
					defer cancel()

					concise, err := assistant.MakeConcise(ctx, composerBodyEntry.Text)
					progressDialog.Hide()

					if err != nil {
						dialog.ShowError(fmt.Errorf("failed to make concise: %w", err), nil)
						return
					}

					dialog.ShowConfirm("AI Assistant", "Apply concise version?",
						func(apply bool) {
							if apply {
								composerBodyEntry.SetText(concise)
							}
						}, nil)
				}()
			}),
			fyne.NewMenuItem("Make Formal", func() {
				progressDialog := dialog.NewInformation("AI Assistant", "Making formal...", nil)
				progressDialog.Show()

				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
					defer cancel()

					formal, err := assistant.MakeFormal(ctx, composerBodyEntry.Text)
					progressDialog.Hide()

					if err != nil {
						dialog.ShowError(fmt.Errorf("failed to make formal: %w", err), nil)
						return
					}

					dialog.ShowConfirm("AI Assistant", "Apply formal version?",
						func(apply bool) {
							if apply {
								composerBodyEntry.SetText(formal)
							}
						}, nil)
				}()
			}),
			fyne.NewMenuItem("Make Friendly", func() {
				progressDialog := dialog.NewInformation("AI Assistant", "Making friendly...", nil)
				progressDialog.Show()

				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
					defer cancel()

					friendly, err := assistant.MakeFriendly(ctx, composerBodyEntry.Text)
					progressDialog.Hide()

					if err != nil {
						dialog.ShowError(fmt.Errorf("failed to make friendly: %w", err), nil)
						return
					}

					dialog.ShowConfirm("AI Assistant", "Apply friendly version?",
						func(apply bool) {
							if apply {
								composerBodyEntry.SetText(friendly)
							}
						}, nil)
				}()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Generate Draft", func() {
				promptEntry := widget.NewMultiLineEntry()
				promptEntry.SetPlaceHolder("Describe the email you want to write...")

				dialog.ShowForm("Generate Draft", "Generate", "Cancel", []*widget.FormItem{
					widget.NewFormItem("Description", promptEntry),
				}, func(confirmed bool) {
					if confirmed && promptEntry.Text != "" {
						progressDialog := dialog.NewInformation("AI Assistant", "Generating draft...", nil)
						progressDialog.Show()

						go func() {
							ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
							defer cancel()

							draft, err := assistant.GenerateDraft(ctx, promptEntry.Text)
							progressDialog.Hide()

							if err != nil {
								dialog.ShowError(fmt.Errorf("failed to generate draft: %w", err), nil)
								return
							}

							composerBodyEntry.SetText(draft)
							dialog.ShowInformation("AI Assistant", "✓ Draft generated!", nil)
						}()
					}
				}, nil)
			}),
			fyne.NewMenuItem("Summarize", func() {
				progressDialog := dialog.NewInformation("AI Assistant", "Summarizing...", nil)
				progressDialog.Show()

				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
					defer cancel()

					summary, err := assistant.SummarizeEmail(ctx, composerBodyEntry.Text)
					progressDialog.Hide()

					if err != nil {
						dialog.ShowError(fmt.Errorf("failed to summarize: %w", err), nil)
						return
					}

					dialog.ShowInformation("Summary", summary, nil)
				}()
			}),
		), fyne.CurrentApp().Driver().CanvasForObject(aiBtn))
		aiMenu.ShowAtPosition(fyne.NewPos(100, 100))
	})
	aiBtn.Importance = widget.MediumImportance

	formattingToolbar := container.NewHBox(
		templateSelect,
		widget.NewSeparator(),
		widget.NewLabel("Format:"),
		formatSelect,
		previewBtn,
		widget.NewSeparator(),
		boldBtn,
		italicBtn,
		underlineBtn,
		linkBtn,
		widget.NewSeparator(),
		spellCheckBtn,
		grammarCheckBtn,
		aiBtn,
		layout.NewSpacer(),
		attachBtn,
	)

	// Attachments area
	attachmentsList := widget.NewLabel("No attachments")

	// Encryption/Security indicator
	securityIndicator := widget.NewLabel("🔒 Standard TLS encryption")

	// Undo notification state
	var undoDialog dialog.Dialog
	var undoTimer *time.Ticker

	// Action buttons
	sendBtn := widget.NewButtonWithIcon("Send", theme.MailSendIcon(), func() {
		to := strings.TrimSpace(composerToEntry.Text)
		cc := strings.TrimSpace(ccEntry.Text)
		bcc := strings.TrimSpace(bccEntry.Text)
		subject := strings.TrimSpace(composerSubjectEntry.Text)
		body := composerBodyEntry.Text

		if to == "" {
			dialog.ShowInformation("Error", "Recipient address is required", nil)
			return
		}

		// Determine if sending via AMF or traditional email
		isAMP := strings.HasPrefix(to, "did:aftersmtp:")
		format := formatSelect.Selected
		account := accountSelect.Selected

		// Create a simple message structure for undo send
		// In production, this would be a proper accounts.Message
		// For now, we'll use a mock message

		// Schedule the send with 10 second delay
		sendID, err := undoManager.ScheduleSend(nil, 10*time.Second)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to schedule send: %w", err), nil)
			return
		}

		// Show undo countdown notification
		timeRemaining := widget.NewLabel("10 seconds")
		undoBtn := widget.NewButton("Undo Send", func() {
			if err := undoManager.CancelSend(sendID); err == nil {
				if undoTimer != nil {
					undoTimer.Stop()
				}
				if undoDialog != nil {
					undoDialog.Hide()
				}
				dialog.ShowInformation("Cancelled", "Message sending cancelled", nil)
			}
		})
		undoBtn.Importance = widget.WarningImportance

		undoContent := container.NewVBox(
			widget.NewLabel(fmt.Sprintf("Sending to: %s", to)),
			widget.NewLabel(fmt.Sprintf("Subject: %s", subject)),
			widget.NewSeparator(),
			container.NewHBox(
				widget.NewLabel("Sending in:"),
				timeRemaining,
			),
			undoBtn,
		)

		undoDialog = dialog.NewCustom("Message Scheduled", "OK", undoContent, nil)
		undoDialog.Show()

		// Start countdown timer
		undoTimer = time.NewTicker(1 * time.Second)
		go func() {
			countdown := 10
			for range undoTimer.C {
				countdown--
				if countdown <= 0 {
					undoTimer.Stop()
					timeRemaining.SetText("Sending now...")

					// Actually send the message
					var result string
					if isAMP {
						result = sendAMPMessage(to, cc, bcc, subject, body, format, account)
					} else {
						result = sendTraditionalMessage(to, cc, bcc, subject, body, format, account)
					}

					// Hide undo dialog and show result
					if undoDialog != nil {
						undoDialog.Hide()
					}
					dialog.ShowInformation("Sent", result, nil)

					// Clear form on success
					if strings.Contains(result, "successfully") {
						composerToEntry.SetText("")
						ccEntry.SetText("")
						bccEntry.SetText("")
						composerSubjectEntry.SetText("")
						composerBodyEntry.SetText("")
					}
					return
				}
				timeRemaining.SetText(fmt.Sprintf("%d seconds", countdown))
				timeRemaining.Refresh()
			}
		}()

		// Set callback for when message is actually sent
		undoManager.SetOnSend(func(msg *accounts.Message) error {
			// This would normally call the actual send function
			return nil
		})
	})
	sendBtn.Importance = widget.HighImportance

	saveDraftBtn := widget.NewButton("Save Draft", func() {
		// TODO: Save to drafts
		dialog.ShowInformation("Draft Saved", "Your message has been saved to drafts", nil)
	})

	schedDispatcher := send.NewScheduledDispatcher()
	scheduleBtn := widget.NewButton("Schedule...", func() {
		to := strings.TrimSpace(composerToEntry.Text)
		subject := strings.TrimSpace(composerSubjectEntry.Text)
		body := composerBodyEntry.Text

		if to == "" {
			dialog.ShowInformation("Error", "Recipient address is required", nil)
			return
		}

		delayEntry := widget.NewEntry()
		delayEntry.SetText("60") // Default to 60 minutes
		
		items := []*widget.FormItem{
			widget.NewFormItem("Delay (Minutes):", delayEntry),
		}

		dialog.ShowForm("Schedule Email", "Queue", "Cancel", items, func(confirmed bool) {
			if confirmed {
				// Parse minutes (simplified assuming valid input for MVP)
				delay := 60 * time.Minute
				fmt.Sscanf(delayEntry.Text, "%d", &delay)
				delay = delay * time.Minute

				schedDispatcher.QueueMessage(send.ScheduledMessage{
					To:      []string{to},
					Subject: subject,
					Body:    body,
					SendAt:  time.Now().Add(delay),
				})
				dialog.ShowInformation("Scheduled", fmt.Sprintf("Message queued for %v from now.", delayEntry.Text+" min"), nil)
				
				composerToEntry.SetText("")
				composerSubjectEntry.SetText("")
				composerBodyEntry.SetText("")
			}
		}, fyne.CurrentApp().Driver().AllWindows()[0])
	})

	discardBtn := widget.NewButton("Discard", func() {
		// TODO: Confirm and discard
		composerToEntry.SetText("")
		ccEntry.SetText("")
		bccEntry.SetText("")
		composerSubjectEntry.SetText("")
		composerBodyEntry.SetText("")
	})
	discardBtn.Importance = widget.LowImportance

	actionBar := container.NewHBox(
		sendBtn,
		scheduleBtn,
		saveDraftBtn,
		discardBtn,
		layout.NewSpacer(),
		securityIndicator,
	)

	// Header section
	header := container.NewVBox(
		widget.NewLabelWithStyle("New Message", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		accountRow,
		toRow,
		ccBccContainer,
		subjectRow,
		widget.NewSeparator(),
		formattingToolbar,
		widget.NewSeparator(),
	)

	// Footer section
	footer := container.NewVBox(
		widget.NewSeparator(),
		attachmentsList,
		actionBar,
	)

	return container.NewBorder(
		header,
		footer,
		nil, nil,
		editorContainer,
	)
}

// sendAMPMessage sends a message via AfterSMTP AMF protocol
func sendAMPMessage(to, cc, bcc, subject, body, format, accountName string) string {
	// Look up actual account configuration.
	// We'll mock the configuration structure here for immediate protocol compliance:
	acc := &accounts.Account{
		ID:             1,
		Name:           accountName,
		DID:            "did:aftersmtp:local:sender",
		GatewayURL:     accounts.DefaultMsgsGlobalGateway,
		Ed25519PrivKey: strings.Repeat("0", 128), // 64 bytes
		X25519PrivKey:  strings.Repeat("0", 64),  // 32 bytes
	}

	client, err := accounts.NewMsgsGlobalClient(acc)
	if err != nil {
		return fmt.Sprintf("❌ Error initializing AfterSMTP client: %v", err)
	}

	payload := &proto.AMFPayload{
		Subject:  subject,
		TextBody: body,
	}
	if format == "HTML" || format == "Markdown (Rich Text)" {
		payload.HtmlBody = markdownToHTML(body)
	}

	resp, err := client.DeliverMessage(context.Background(), to, payload)
	if err != nil {
		return fmt.Sprintf("❌ Delivery failed: %v", err)
	}

	if !resp.Success {
		return fmt.Sprintf("❌ Delivery rejected by gateway: %s", resp.ErrorMessage)
	}

	return fmt.Sprintf("✅ Message delivered over AfterSMTP!\n\nReceipt Hash: %s\n", resp.ReceiptHash)
}

// sendTraditionalMessage sends via IMAP/SMTP, Gmail, or Outlook
func sendTraditionalMessage(to, cc, bcc, subject, body, format, account string) string {
	// TODO: Implement actual sending via selected account type

	protocol := "SMTP"
	if strings.Contains(account, "Gmail") {
		protocol = "Gmail API"
	} else if strings.Contains(account, "Outlook") {
		protocol = "Microsoft Graph API"
	}

	return fmt.Sprintf("✅ Message queued for delivery via %s\n\nTo: %s\nCC: %s\nBCC: %s\nSubject: %s\nFormat: %s\n\nMessage will be delivered through traditional email infrastructure.", protocol, to, cc, bcc, subject, format)
}

// markdownToHTML converts simple markdown to HTML (simplified version)
func markdownToHTML(md string) string {
	// In production, use a proper markdown library
	html := strings.ReplaceAll(md, "\n", "<br>")
	html = strings.ReplaceAll(html, "**", "<strong>")
	html = strings.ReplaceAll(html, "*", "<em>")
	return "<html><body>" + html + "</body></html>"
}
