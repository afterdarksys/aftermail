package gui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ryan/meowmail/pkg/accounts"
	ampProto "github.com/ryan/meowmail/pkg/proto"
	"google.golang.org/protobuf/proto"
)

// AMFMessageViewer displays AfterSMTP messages with AMF (ADS Mail Format) support
type AMFMessageViewer struct {
	widget.BaseWidget
	message *accounts.Message
	content *fyne.Container
}

// NewAMFMessageViewer creates a new AMF message viewer
func NewAMFMessageViewer(msg *accounts.Message) *AMFMessageViewer {
	v := &AMFMessageViewer{
		message: msg,
	}
	v.ExtendBaseWidget(v)
	v.buildContent()
	return v
}

// buildContent constructs the viewer UI
func (v *AMFMessageViewer) buildContent() {
	// Check if this is an AMF message
	isAMF := v.message.Protocol == "amp" || len(v.message.AMFPayload) > 0

	// Header section
	headerText := fmt.Sprintf("From: %s\n", v.message.Sender)
	if v.message.SenderDID != "" {
		headerText += fmt.Sprintf("DID: %s\n", v.message.SenderDID)
	}
	if len(v.message.Recipients) > 0 {
		headerText += fmt.Sprintf("To: %s\n", v.message.Recipients[0])
	}
	headerText += fmt.Sprintf("Subject: %s\n", v.message.Subject)
	headerText += fmt.Sprintf("Date: %s\n", v.message.ReceivedAt.Format(time.RFC1123))

	if isAMF {
		headerText += fmt.Sprintf("Protocol: AfterSMTP AMF (Next-Gen)\n")
		if v.message.Verified {
			headerText += "Signature: ✅ VERIFIED (Blockchain-backed)\n"
		} else if len(v.message.Signature) > 0 {
			headerText += "Signature: ⚠️ UNVERIFIED\n"
		}
	} else {
		headerText += fmt.Sprintf("Protocol: %s (Legacy MIME)\n", v.message.Protocol)
	}

	headerLabel := widget.NewLabel(headerText)
	headerLabel.Wrapping = fyne.TextWrapWord

	// Body section
	var bodyContent *widget.RichText
	if isAMF && len(v.message.AMFPayload) > 0 {
		// Parse AMF payload
		var amfPayload ampProto.AMFPayload
		if err := proto.Unmarshal(v.message.AMFPayload, &amfPayload); err == nil {
			bodyContent = v.renderAMFBody(&amfPayload)
		} else {
			bodyContent = widget.NewRichTextFromMarkdown("**Error:** Failed to parse AMF payload")
		}
	} else {
		// Traditional MIME rendering
		bodyContent = v.renderMIMEBody()
	}

	// Attachments section
	var attachmentsWidget *fyne.Container
	if len(v.message.Attachments) > 0 {
		attachmentsWidget = v.renderAttachments(isAMF)
	}

	// Build layout
	sections := []fyne.CanvasObject{
		widget.NewSeparator(),
		headerLabel,
		widget.NewSeparator(),
		container.NewScroll(bodyContent),
	}

	if attachmentsWidget != nil {
		sections = append(sections, widget.NewSeparator())
		sections = append(sections, attachmentsWidget)
	}

	v.content = container.NewVBox(sections...)
}

// renderAMFBody renders an AMF payload with enhanced formatting
func (v *AMFMessageViewer) renderAMFBody(payload *ampProto.AMFPayload) *widget.RichText {
	// Prefer HTML body if available, otherwise fall back to plain text
	var content string
	if payload.HtmlBody != "" {
		// In a production app, render HTML properly
		// For now, show as-is with a note
		content = "**[HTML Content]**\n\n" + payload.HtmlBody
	} else {
		content = payload.TextBody
	}

	// Show extended headers if present
	if len(payload.ExtendedHeaders) > 0 {
		content += "\n\n---\n**Extended Headers:**\n"
		for key, val := range payload.ExtendedHeaders {
			content += fmt.Sprintf("- %s: %s\n", key, val)
		}
	}

	return widget.NewRichTextFromMarkdown(content)
}

// renderMIMEBody renders traditional MIME message body
func (v *AMFMessageViewer) renderMIMEBody() *widget.RichText {
	// Prefer HTML body if available
	if v.message.BodyHTML != "" {
		return widget.NewRichTextFromMarkdown("**[HTML Content]**\n\n" + v.message.BodyHTML)
	}
	return widget.NewRichTextFromMarkdown(v.message.BodyPlain)
}

// renderAttachments renders the attachments section
func (v *AMFMessageViewer) renderAttachments(isAMF bool) *fyne.Container {
	label := widget.NewLabelWithStyle(
		fmt.Sprintf("📎 Attachments (%d)", len(v.message.Attachments)),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	attachmentList := widget.NewList(
		func() int { return len(v.message.Attachments) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("filename.ext"),
				widget.NewLabel("1.2 MB"),
				widget.NewButton("Save", nil),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			att := v.message.Attachments[id]
			box := obj.(*fyne.Container)

			nameLabel := box.Objects[0].(*widget.Label)
			sizeLabel := box.Objects[1].(*widget.Label)
			saveBtn := box.Objects[2].(*widget.Button)

			nameLabel.SetText(att.Filename)
			sizeLabel.SetText(formatFileSize(att.Size))

			if isAMF && att.Hash != "" {
				nameLabel.SetText(fmt.Sprintf("%s (✅ Hash: %s...)", att.Filename, att.Hash[:8]))
			}

			saveBtn.OnTapped = func() {
				// TODO: Implement save attachment functionality
				fmt.Printf("Saving attachment: %s\n", att.Filename)
			}
		},
	)

	return container.NewBorder(
		label,
		nil, nil, nil,
		attachmentList,
	)
}

// formatFileSize formats bytes into human-readable format
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// CreateRenderer implements the widget.Widget interface
func (v *AMFMessageViewer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(v.content)
}

// SetMessage updates the message being displayed
func (v *AMFMessageViewer) SetMessage(msg *accounts.Message) {
	v.message = msg
	v.buildContent()
	v.Refresh()
}
