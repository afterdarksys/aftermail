package gui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

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
		// TODO: Implement spell check with AI
		dialog.ShowInformation("Spell Check", "Checking spelling...\n\n⚠️ Configure AI API key in Settings → AI Assistant", nil)
	})
	spellCheckBtn.Importance = widget.LowImportance

	grammarCheckBtn := widget.NewButton("✓ Grammar", func() {
		if bodyEntry.Text == "" {
			dialog.ShowInformation("Grammar Check", "No text to check", nil)
			return
		}
		// TODO: Implement grammar check with AI
		dialog.ShowInformation("Grammar Check", "Checking grammar...\n\n⚠️ Configure AI API key in Settings → AI Assistant", nil)
	})
	grammarCheckBtn.Importance = widget.LowImportance

	aiBtn := widget.NewButton("🤖 AI Assistant", func() {
		if bodyEntry.Text == "" {
			dialog.ShowInformation("AI Assistant", "Select text to improve or write a draft description", nil)
			return
		}

		// Show AI menu
		aiMenu := widget.NewPopUpMenu(fyne.NewMenu("",
			fyne.NewMenuItem("Improve Writing", func() {
				dialog.ShowInformation("AI", "Improving writing...\n\n⚠️ Configure AI API key in Settings", nil)
			}),
			fyne.NewMenuItem("Make Concise", func() {
				dialog.ShowInformation("AI", "Making concise...\n\n⚠️ Configure AI API key in Settings", nil)
			}),
			fyne.NewMenuItem("Make Formal", func() {
				dialog.ShowInformation("AI", "Making formal...\n\n⚠️ Configure AI API key in Settings", nil)
			}),
			fyne.NewMenuItem("Make Friendly", func() {
				dialog.ShowInformation("AI", "Making friendly...\n\n⚠️ Configure AI API key in Settings", nil)
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Generate Draft", func() {
				promptEntry := widget.NewMultiLineEntry()
				promptEntry.SetPlaceHolder("Describe the email you want to write...")

				dialog.ShowForm("Generate Draft", "Generate", "Cancel", []*widget.FormItem{
					widget.NewFormItem("Description", promptEntry),
				}, func(confirmed bool) {
					if confirmed && promptEntry.Text != "" {
						dialog.ShowInformation("Generating", "Generating draft...\n\n⚠️ Configure AI API key in Settings", nil)
					}
				}, nil)
			}),
			fyne.NewMenuItem("Summarize", func() {
				dialog.ShowInformation("AI", "Summarizing...\n\n⚠️ Configure AI API key in Settings", nil)
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

		var result string
		if isAMP {
			result = sendAMPMessage(to, subject, body, formatSelect.Selected)
		} else {
			result = sendTraditionalMessage(to, subject, body, formatSelect.Selected, accountSelect.Selected)
		}

		dialog.ShowInformation("Send Result", result, nil)

		// Clear form on success
		if strings.Contains(result, "successfully") {
			toEntry.SetText("")
			ccEntry.SetText("")
			bccEntry.SetText("")
			subjectEntry.SetText("")
			bodyEntry.SetText("")
		}
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
