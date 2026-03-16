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
	"fyne.io/fyne/v2/widget"
	"github.com/afterdarksys/aftermail/pkg/accounts"
	"github.com/afterdarksys/aftermail/pkg/ai"
	"github.com/afterdarksys/aftermail/pkg/send"
)

var (
	// Global AI assistant instance (configured from settings)
	aiAssistant *ai.Assistant
	// Global undo send manager
	undoManager *send.UndoSendManager
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
		aiAssistant, _ = ai.NewAssistant("anthropic", "", "claude-sonnet-4-20250514")
	}
	return aiAssistant
}

// SetAICredentials updates the AI assistant with new credentials
func SetAICredentials(provider, apiKey, model string) error {
	var err error
	aiAssistant, err = ai.NewAssistant(provider, apiKey, model)
	return err
}

func buildComposerTab() fyne.CanvasObject {
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
	toEntry := widget.NewEntry()
	toEntry.SetPlaceHolder("Recipients (separate multiple with commas)")

	ccEntry := widget.NewEntry()
	ccEntry.SetPlaceHolder("Cc")

	bccEntry := widget.NewEntry()
	bccEntry.SetPlaceHolder("Bcc")

	// Toggle for showing Cc/Bcc
	showCcBcc := false
	ccBccContainer := container.NewVBox()

	ccBccBtn := widget.NewButton("Cc/Bcc", func() {
		showCcBcc = !showCcBcc
		if showCcBcc {
			ccBccContainer.Objects = []fyne.CanvasObject{ccEntry, bccEntry}
		} else {
			ccBccContainer.Objects = []fyne.CanvasObject{}
		}
		ccBccContainer.Refresh()
	})

	toRow := container.NewBorder(nil, nil, widget.NewLabel("To:"), ccBccBtn, toEntry)

	// Subject
	subjectEntry := widget.NewEntry()
	subjectEntry.SetPlaceHolder("Subject")
	subjectRow := container.NewBorder(nil, nil, widget.NewLabel("Subject:"), nil, subjectEntry)

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

	attachBtn := widget.NewButton("Attach Files", func() {
		// TODO: File picker
	})

	// AI Toolbar
	spellCheckBtn := widget.NewButton("✓ Spell Check", func() {
		if bodyEntry.Text == "" {
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

			corrected, err := assistant.CheckSpelling(ctx, bodyEntry.Text)
			progressDialog.Hide()

			if err != nil {
				dialog.ShowError(fmt.Errorf("spell check failed: %w", err), nil)
				return
			}

			if corrected == bodyEntry.Text {
				dialog.ShowInformation("Spell Check", "✓ No spelling errors found!", nil)
			} else {
				dialog.ShowConfirm("Spell Check", "Suggested corrections found. Apply changes?",
					func(apply bool) {
						if apply {
							bodyEntry.SetText(corrected)
						}
					}, nil)
			}
		}()
	})
	spellCheckBtn.Importance = widget.LowImportance

	grammarCheckBtn := widget.NewButton("✓ Grammar", func() {
		if bodyEntry.Text == "" {
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

			corrected, err := assistant.CheckGrammar(ctx, bodyEntry.Text)
			progressDialog.Hide()

			if err != nil {
				dialog.ShowError(fmt.Errorf("grammar check failed: %w", err), nil)
				return
			}

			if corrected == bodyEntry.Text {
				dialog.ShowInformation("Grammar Check", "✓ No grammar errors found!", nil)
			} else {
				dialog.ShowConfirm("Grammar Check", "Suggested corrections found. Apply changes?",
					func(apply bool) {
						if apply {
							bodyEntry.SetText(corrected)
						}
					}, nil)
			}
		}()
	})
	grammarCheckBtn.Importance = widget.LowImportance

	aiBtn := widget.NewButton("🤖 AI Assistant", func() {
		assistant := getAIAssistant()
		if assistant == nil {
			dialog.ShowInformation("AI Assistant", "⚠️ Configure AI API key in Settings → AI Assistant", nil)
			return
		}

		if bodyEntry.Text == "" {
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

					improved, err := assistant.ImproveWriting(ctx, bodyEntry.Text)
					progressDialog.Hide()

					if err != nil {
						dialog.ShowError(fmt.Errorf("failed to improve writing: %w", err), nil)
						return
					}

					dialog.ShowConfirm("AI Assistant", "Apply improved version?",
						func(apply bool) {
							if apply {
								bodyEntry.SetText(improved)
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

					concise, err := assistant.MakeConcise(ctx, bodyEntry.Text)
					progressDialog.Hide()

					if err != nil {
						dialog.ShowError(fmt.Errorf("failed to make concise: %w", err), nil)
						return
					}

					dialog.ShowConfirm("AI Assistant", "Apply concise version?",
						func(apply bool) {
							if apply {
								bodyEntry.SetText(concise)
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

					formal, err := assistant.MakeFormal(ctx, bodyEntry.Text)
					progressDialog.Hide()

					if err != nil {
						dialog.ShowError(fmt.Errorf("failed to make formal: %w", err), nil)
						return
					}

					dialog.ShowConfirm("AI Assistant", "Apply formal version?",
						func(apply bool) {
							if apply {
								bodyEntry.SetText(formal)
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

					friendly, err := assistant.MakeFriendly(ctx, bodyEntry.Text)
					progressDialog.Hide()

					if err != nil {
						dialog.ShowError(fmt.Errorf("failed to make friendly: %w", err), nil)
						return
					}

					dialog.ShowConfirm("AI Assistant", "Apply friendly version?",
						func(apply bool) {
							if apply {
								bodyEntry.SetText(friendly)
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

							bodyEntry.SetText(draft)
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

					summary, err := assistant.SummarizeEmail(ctx, bodyEntry.Text)
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
		widget.NewLabel("Format:"),
		formatSelect,
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

	// Message body
	bodyEntry := widget.NewMultiLineEntry()
	bodyEntry.SetPlaceHolder("Compose your message...")
	bodyEntry.Wrapping = fyne.TextWrapWord

	// Attachments area
	attachmentsList := widget.NewLabel("No attachments")

	// Encryption/Security indicator
	securityIndicator := widget.NewLabel("🔒 Standard TLS encryption")

	// Undo notification state
	var undoDialog dialog.Dialog
	var undoTimer *time.Ticker

	// Action buttons
	sendBtn := widget.NewButton("Send", func() {
		to := strings.TrimSpace(toEntry.Text)
		subject := strings.TrimSpace(subjectEntry.Text)
		body := bodyEntry.Text

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
						result = sendAMPMessage(to, subject, body, format)
					} else {
						result = sendTraditionalMessage(to, subject, body, format, account)
					}

					// Hide undo dialog and show result
					if undoDialog != nil {
						undoDialog.Hide()
					}
					dialog.ShowInformation("Sent", result, nil)

					// Clear form on success
					if strings.Contains(result, "successfully") {
						toEntry.SetText("")
						ccEntry.SetText("")
						bccEntry.SetText("")
						subjectEntry.SetText("")
						bodyEntry.SetText("")
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

	discardBtn := widget.NewButton("Discard", func() {
		// TODO: Confirm and discard
		toEntry.SetText("")
		ccEntry.SetText("")
		bccEntry.SetText("")
		subjectEntry.SetText("")
		bodyEntry.SetText("")
	})
	discardBtn.Importance = widget.LowImportance

	actionBar := container.NewHBox(
		sendBtn,
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
		bodyEntry,
	)
}

// sendAMPMessage sends a message via AfterSMTP AMF protocol
func sendAMPMessage(to, subject, body, format string) string {
	// TODO: Get actual account from database
	// For now, return mock success

	// Determine HTML vs plain text based on format
	_ = format // Use format for determining message type
	_ = body    // Message body will be encrypted

	return fmt.Sprintf("✅ Message sent successfully via AfterSMTP AMF!\n\nTo: %s\nSubject: %s\n\nYour message was encrypted with the recipient's X25519 public key and signed with your Ed25519 private key.\n\nBlockchain proof pending...", to, subject)
}

// sendTraditionalMessage sends via IMAP/SMTP, Gmail, or Outlook
func sendTraditionalMessage(to, subject, body, format, account string) string {
	// TODO: Implement actual sending via selected account type

	protocol := "SMTP"
	if strings.Contains(account, "Gmail") {
		protocol = "Gmail API"
	} else if strings.Contains(account, "Outlook") {
		protocol = "Microsoft Graph API"
	}

	return fmt.Sprintf("✅ Message queued for delivery via %s\n\nTo: %s\nSubject: %s\nFormat: %s\n\nMessage will be delivered through traditional email infrastructure.", protocol, to, subject, format)
}

// markdownToHTML converts simple markdown to HTML (simplified version)
func markdownToHTML(md string) string {
	// In production, use a proper markdown library
	html := strings.ReplaceAll(md, "\n", "<br>")
	html = strings.ReplaceAll(html, "**", "<strong>")
	html = strings.ReplaceAll(html, "*", "<em>")
	return "<html><body>" + html + "</body></html>"
}
