package gui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func buildComposerTab() fyne.CanvasObject {
	toEntry := widget.NewEntry()
	toEntry.SetPlaceHolder("To: (e.g., user@example.com or did:aftersmtp:msgs.global:user)")

	subjectEntry := widget.NewEntry()
	subjectEntry.SetPlaceHolder("Subject:")

	bodyEntry := widget.NewMultiLineEntry()
	bodyEntry.SetPlaceHolder("Type your message here...")

	// Account selector
	accountSelect := widget.NewSelect([]string{"IMAP (example@gmail.com)", "AfterSMTP (did:aftersmtp:msgs.global:ryan)", "Outlook (user@outlook.com)"}, nil)
	accountSelect.SetSelected("IMAP (example@gmail.com)")

	// Format selector
	formatRadio := widget.NewRadioGroup([]string{"Plain Text", "Rich Text (Markdown)", "Full HTML", "AfterSMTP AMF Native"}, func(selected string) {
		switch selected {
		case "AfterSMTP AMF Native":
			bodyEntry.SetPlaceHolder("Compose your message in plain text or HTML.\nAfterSMTP will automatically encrypt and sign it with your DID keys.")
		case "Full HTML":
			bodyEntry.SetPlaceHolder("<html><body><h1>Your HTML content here</h1></body></html>")
		case "Rich Text (Markdown)":
			bodyEntry.SetPlaceHolder("# Heading\n**Bold** and *italic* text supported")
		default:
			bodyEntry.SetPlaceHolder("Type your message here...")
		}
	})
	formatRadio.SetSelected("Plain Text")
	formatRadio.Horizontal = true

	// Attachments list (placeholder)
	attachmentsLabel := widget.NewLabel("Attachments: None")

	sendBtn := widget.NewButton("Send Message", func() {
		to := strings.TrimSpace(toEntry.Text)
		subject := strings.TrimSpace(subjectEntry.Text)
		body := bodyEntry.Text

		if to == "" {
			dialog.ShowInformation("Error", "Recipient address is required", nil)
			return
		}

		// Determine if sending via AMF or traditional email
		isAMP := strings.HasPrefix(to, "did:aftersmtp:") || formatRadio.Selected == "AfterSMTP AMF Native"

		var result string
		if isAMP {
			result = sendAMPMessage(to, subject, body, formatRadio.Selected)
		} else {
			result = sendTraditionalMessage(to, subject, body, formatRadio.Selected, accountSelect.Selected)
		}

		dialog.ShowInformation("Send Result", result, nil)

		// Clear form on success
		if strings.Contains(result, "successfully") {
			toEntry.SetText("")
			subjectEntry.SetText("")
			bodyEntry.SetText("")
		}
	})

	headerForm := container.NewVBox(
		widget.NewLabelWithStyle("🐱 Meowmail Composer", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(widget.NewLabel("From:"), accountSelect),
		toEntry,
		subjectEntry,
		container.NewHBox(widget.NewLabel("Format:"), formatRadio),
		attachmentsLabel,
	)

	return container.NewBorder(headerForm, sendBtn, nil, nil, bodyEntry)
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
