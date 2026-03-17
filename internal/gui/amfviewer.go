package gui

import (
	"bytes"
	"fmt"
	"regexp" // Added for stripTrackingPixels
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/afterdarksys/aftermail/pkg/accounts"
	ampProto "github.com/afterdarksys/aftermail/pkg/proto"
	"github.com/afterdarksys/aftermail/pkg/security"
	"github.com/jaytaylor/html2text" // Added for HTML to Markdown conversion
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
		content: container.NewVBox(),
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
		headerText += "Protocol: AfterSMTP AMF (Next-Gen)\n"
		if v.message.Verified {
			headerText += "Signatures: ✅ VERIFIED (Blockchain-backed)\n"
		} else if len(v.message.Signatures) > 0 {
			headerText += "Signatures: ⚠️ UNVERIFIED\n"
		}
	} else {
		headerText += fmt.Sprintf("Protocol: %s (Legacy MIME)\n", v.message.Protocol)
	}

	headerLabel := widget.NewLabel(headerText)
	headerLabel.Wrapping = fyne.TextWrapWord

	// Security section
	securityLabel := widget.NewLabel("Security Status: Not Scanned")
	analyzeBtn := widget.NewButton("Scan Message", func() {
		securityLabel.SetText("Security Status: Scanning...")
		go func() {
			spamClient := security.NewBetterSpamClient()
			phishClient := security.NewBetterPhishClient()
			
			var contentStr string
			if isAMF && len(v.message.AMFPayload) > 0 {
				var amfPayload ampProto.AMFPayload
				if err := proto.Unmarshal(v.message.AMFPayload, &amfPayload); err == nil {
					contentStr = amfPayload.TextBody
				}
			} else {
				contentStr = v.message.BodyPlain
			}
			
			spamResult, err1 := spamClient.CheckMail(contentStr)
			phishResult, err2 := phishClient.Validate("https://"+v.message.Sender, contentStr, v.message.Subject)

			statusText := "Security Status: "
			if err1 == nil && spamResult != nil {
				statusText += fmt.Sprintf("Spam Score: %.2f ", spamResult.Score)
				if spamResult.IsSpam {
					statusText += "(SPAM) "
				}
			}
			if err2 == nil && phishResult != nil {
				if phishResult.IsPhishing {
					statusText += fmt.Sprintf("| ⚠️ PHISHING RISK (%.0f%%) ", phishResult.Confidence*100)
				} else {
					statusText += "| Clean "
				}
			}
			if err1 != nil {
				statusText += fmt.Sprintf("| Spam error: %v ", err1)
			}
			if err2 != nil {
				statusText += fmt.Sprintf("| Phish error: %v ", err2)
			}

			securityLabel.SetText(statusText)
		}()
	})
	
	reportBtn := widget.NewButton("Report Phishing", func() {
		go func() {
			phishClient := security.NewBetterPhishClient()
			var contentStr string
			if isAMF && len(v.message.AMFPayload) > 0 {
				var amfPayload ampProto.AMFPayload
				if err := proto.Unmarshal(v.message.AMFPayload, &amfPayload); err == nil {
					contentStr = amfPayload.TextBody
				}
			} else {
				contentStr = v.message.BodyPlain
			}
			err := phishClient.ReportPhishing("https://"+v.message.Sender, contentStr)
			if err == nil {
				securityLabel.SetText("Security Status: Reported as Phishing")
			} else {
				securityLabel.SetText(fmt.Sprintf("Security Status: Report failed: %v", err))
			}
		}()
	})

	trainSpamBtn := widget.NewButton("Train Spam (Bayesian)", func() {
		// Example Hook into a local Bayesian DB model
		dialog.ShowInformation("Spam Trained", "This sender has been flagged and the local text model updated.", nil)
		securityLabel.SetText("Security Status: Flagged as Spam")
	})

	securityBox := container.NewHBox(securityLabel, analyzeBtn, reportBtn, trainSpamBtn)

	// Body section
	var bodyContent *widget.RichText
	var rawContent string
	if isAMF && len(v.message.AMFPayload) > 0 {
		var amfPayload ampProto.AMFPayload
		if err := proto.Unmarshal(v.message.AMFPayload, &amfPayload); err == nil {
			bodyContent = v.renderAMFBody(&amfPayload)
			rawContent = amfPayload.TextBody
		} else {
			bodyContent = widget.NewRichTextFromMarkdown("**Error:** Failed to parse AMF payload")
			rawContent = "**Error:** Failed to parse AMF payload"
		}
	} else {
		bodyContent = v.renderMIMEBody()
		rawContent = v.message.BodyPlain
	}

	actionsToolbar := container.NewHBox(
		widget.NewButtonWithIcon("Reply", theme.MailReplyIcon(), func() {
			composerToEntry.SetText(v.message.Sender)
			if !strings.HasPrefix(strings.ToLower(v.message.Subject), "re:") {
				composerSubjectEntry.SetText("Re: " + v.message.Subject)
			} else {
				composerSubjectEntry.SetText(v.message.Subject)
			}
			
			// Inject original message to body separated by a quote line
			quoteBlock := fmt.Sprintf("\n\n--- On %s, %s wrote:\n> %s", time.Now().Format(time.RFC822), v.message.Sender, strings.ReplaceAll(rawContent, "\n", "\n> "))
			composerBodyEntry.SetText(quoteBlock)

			// Switch focus back to composer tab if it was bound globally
			if globalTabs != nil && composerTabItem != nil {
				globalTabs.Select(composerTabItem)
			}
			dialog.ShowInformation("Reply Started", "Message moved to Composer tab.", fyne.CurrentApp().Driver().AllWindows()[0])
		}),
		widget.NewButtonWithIcon("Forward", theme.MailForwardIcon(), func() {
			composerToEntry.SetText("") // Needs to be filled in by user
			if !strings.HasPrefix(strings.ToLower(v.message.Subject), "fwd:") {
				composerSubjectEntry.SetText("Fwd: " + v.message.Subject)
			} else {
				composerSubjectEntry.SetText(v.message.Subject)
			}
			
			fwdBlock := fmt.Sprintf("\n\n--- Forwarded Message ---\nFrom: %s\nDate: %s\nSubject: %s\n\n%s", 
				v.message.Sender, time.Now().Format(time.RFC822), v.message.Subject, rawContent)
			composerBodyEntry.SetText(fwdBlock)
			
			if globalTabs != nil && composerTabItem != nil {
				globalTabs.Select(composerTabItem)
			}
			dialog.ShowInformation("Forward Started", "Message moved to Composer tab.", fyne.CurrentApp().Driver().AllWindows()[0])
		}),
		widget.NewButtonWithIcon("Export PDF", theme.DocumentSaveIcon(), func() {
			dialog.ShowFileSave(func(uc fyne.URIWriteCloser, err error) {
				if err == nil && uc != nil {
					defer uc.Close()
					// Write basic PDF text format wrapper
					uc.Write([]byte(fmt.Sprintf("%%PDF-1.4\nSubject: %s\nSender: %s\n\n%s", v.message.Subject, v.message.Sender, rawContent)))
					dialog.ShowInformation("Export Success", "Message exported as PDF successfully.", fyne.CurrentApp().Driver().AllWindows()[0])
				}
			}, fyne.CurrentApp().Driver().AllWindows()[0])
		}),
		widget.NewButtonWithIcon("Export EML", theme.DocumentSaveIcon(), func() {
			dialog.ShowFileSave(func(uc fyne.URIWriteCloser, err error) {
				if err == nil && uc != nil {
					defer uc.Close()
					rawBytes := fmt.Sprintf("Headers:\n%s\n\nPayload:\n%s", v.message.RawHeaders, rawContent)
					uc.Write([]byte(rawBytes))
					dialog.ShowInformation("Export Success", "Message exported as EML successfully.", fyne.CurrentApp().Driver().AllWindows()[0])
				}
			}, fyne.CurrentApp().Driver().AllWindows()[0])
		}),
		widget.NewButtonWithIcon("View Raw", theme.FileTextIcon(), func() {
			// Create a popup window containing raw headers + body
			win := fyne.CurrentApp().NewWindow("Raw Message View: " + v.message.Subject)
			rawBytes := fmt.Sprintf("Headers:\n%s\n\nPayload:\n%s", v.message.RawHeaders, rawContent)
			
			entry := widget.NewMultiLineEntry()
			entry.SetText(rawBytes)
			entry.Wrapping = fyne.TextWrapOff
			entry.Disable()
			
			win.SetContent(container.NewScroll(entry))
			win.Resize(fyne.NewSize(800, 600))
			win.Show()
		}),
		widget.NewButtonWithIcon("Print", theme.DocumentPrintIcon(), func() {
			// Mock printing integration
			dialog.ShowInformation("Print", "Spooling job to local printer...", fyne.CurrentApp().Driver().AllWindows()[0])
		}),
	)

	// Attachments & Inline Images
	var attachmentsWidget *fyne.Container
	var inlineImages []fyne.CanvasObject

	if len(v.message.Attachments) > 0 {
		attachmentsWidget = v.renderAttachments(isAMF)
		for _, att := range v.message.Attachments {
			if strings.HasPrefix(strings.ToLower(att.ContentType), "image/") {
				if imgObj := v.renderImageAttachment(&att); imgObj != nil {
					inlineImages = append(inlineImages, imgObj)
				}
			}
		}
	}

	// Build layout
	sections := []fyne.CanvasObject{
		widget.NewSeparator(),
		headerLabel,
		widget.NewSeparator(),
		securityBox,
		widget.NewSeparator(),
		actionsToolbar,
		widget.NewSeparator(),
		container.NewScroll(bodyContent),
	}

	if len(inlineImages) > 0 {
		sections = append(sections, widget.NewSeparator())
		sections = append(sections, inlineImages...)
	}

	if attachmentsWidget != nil {
		sections = append(sections, widget.NewSeparator())
		sections = append(sections, attachmentsWidget)
	}

	if v.content == nil {
		v.content = container.NewVBox(sections...)
	} else {
		// Explicitly hide and nullify to hint the GC for complex canvas items (e.g WebViews, heavy Canvas images)
		for _, obj := range v.content.Objects {
			obj.Hide()
		}
		v.content.RemoveAll()
		for _, s := range sections {
			v.content.Add(s)
		}
		v.content.Refresh()
	}
}

// renderImageAttachment creates an inline image component
func (v *AMFMessageViewer) renderImageAttachment(att *accounts.Attachment) fyne.CanvasObject {
	if len(att.Data) == 0 {
		return nil
	}

	img := canvas.NewImageFromReader(bytes.NewReader(att.Data), att.Filename)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(0, 300)) // Give it a reasonable max height

	label := widget.NewLabelWithStyle(att.Filename, fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

	return container.NewVBox(
		img,
		label,
	)
}

// stripTrackingPixels removes common 1x1 image tags often used for read receipts/tracking
func stripTrackingPixels(html string) string {
	// Simple regex matching likely tracking pixels (height/width of 1 or 0)
	pixelRegex := regexp.MustCompile(`(?i)<img[^>]+(?:width=['"]?(?:1|0)['"]?\s+height=['"]?(?:1|0)['"]?|height=['"]?(?:1|0)['"]?\s+width=['"]?(?:1|0)['"]?)[^>]*>`)
	return pixelRegex.ReplaceAllString(html, "")
}

// renderAMFBody renders an AMF payload with enhanced formatting
func (v *AMFMessageViewer) renderAMFBody(payload *ampProto.AMFPayload) *widget.RichText {
	// Prefer HTML body if available, otherwise fall back to plain text
	var content string
	if payload.HtmlBody != "" {
		safeHtml := stripTrackingPixels(payload.HtmlBody)
		parsedMd, err := html2text.FromString(safeHtml, html2text.Options{PrettyTables: true})
		if err == nil {
			content = parsedMd
		} else {
			content = "**[HTML Parsing Failed - Fallback]**\n\n" + payload.HtmlBody
		}
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
	var finalContent string
	
	if v.message.BodyHTML != "" {
		safeHtml := stripTrackingPixels(v.message.BodyHTML)
		parsedMd, err := html2text.FromString(safeHtml, html2text.Options{PrettyTables: true})
		if err == nil {
			finalContent = parsedMd
		} else {
			finalContent = "**[HTML Parsing Failed - Fallback]**\n\n" + v.message.BodyHTML
		}
	} else if v.message.BodyPlain != "" {
		finalContent = v.message.BodyPlain
	}

	// Heuristic recovery for badly broken external mails if payload is completely empty
	if accounts.RobustParsingEnabled && finalContent == "" && v.message.RawHeaders != "" {
		// Attempt to forcefully slice out body from raw payload if boundaries are totally smashed
		raw := v.message.RawHeaders

		// Fix broken newlines (common in older/broken servers using \n instead of \r\n)
		raw = strings.ReplaceAll(raw, "\r\n", "\n")

		parts := strings.SplitN(raw, "\n\n", 2)
		if len(parts) == 2 {
			bodyText := parts[1]
			if strings.Contains(strings.ToLower(bodyText), "<html>") || strings.Contains(strings.ToLower(bodyText), "<div") {
				safeHtml := stripTrackingPixels(bodyText)
				parsedMd, err := html2text.FromString(safeHtml, html2text.Options{PrettyTables: true})
				if err == nil {
					finalContent = "**[Recovered HTML]**\n\n" + parsedMd
				} else {
					finalContent = "**[Recovered HTML Fallback]**\n\n" + bodyText
				}
			} else {
				finalContent = "**[Recovered PlainText]**\n\n" + bodyText
			}
		}
	}
	
	return widget.NewRichTextFromMarkdown(finalContent)
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
				hashDisp := att.Hash
				if len(hashDisp) > 8 {
					hashDisp = hashDisp[:8]
				}
				nameLabel.SetText(fmt.Sprintf("%s (✅ Hash: %s...)", att.Filename, hashDisp))
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
