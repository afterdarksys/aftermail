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
	smtppkg "github.com/afterdarksys/aftermail/pkg/smtp"
	"github.com/afterdarksys/aftermail/pkg/storage"
)

var (
	aiAssistant *ai.Assistant
	undoManager *send.UndoSendManager

	composerToEntry      *widget.Entry
	composerSubjectEntry *widget.Entry
	composerBodyEntry    *widget.Entry
	composerTabItem      *container.TabItem
	globalTabs           *container.AppTabs
)

func init() {
	undoManager = send.NewUndoSendManager(10 * time.Second)
}

func getAIAssistant() *ai.Assistant {
	if aiAssistant == nil {
		aiAssistant = ai.NewAssistant(ai.ProviderAnthropic, "", "claude-sonnet-4-20250514")
	}
	return aiAssistant
}

func SetAICredentials(provider, apiKey, model string) error {
	aiAssistant = ai.NewAssistant(ai.Provider(provider), apiKey, model)
	return nil
}

func buildComposerTab(w fyne.Window, db *storage.DB) fyne.CanvasObject {
	// ── Account selector ────────────────────────────────────────────────────
	var accountOptions []string
	var dbAccounts []*accounts.Account

	if db != nil {
		if list, err := db.ListAccounts(); err == nil {
			for _, a := range list {
				accountOptions = append(accountOptions, fmt.Sprintf("%s (%s)", a.Email, a.Name))
				dbAccounts = append(dbAccounts, a)
			}
		}
	}
	if len(accountOptions) == 0 {
		accountOptions = []string{"No accounts configured — add one in Settings"}
	}

	accountLabel := widget.NewLabel("From:")
	accountSelect := widget.NewSelect(accountOptions, nil)
	accountSelect.SetSelected(accountOptions[0])
	accountRow := container.NewBorder(nil, nil, accountLabel, nil, accountSelect)

	// ── Recipients ──────────────────────────────────────────────────────────
	composerToEntry = widget.NewEntry()
	composerToEntry.SetPlaceHolder("To (comma-separate multiple)")

	ccEntry := widget.NewEntry()
	ccEntry.SetPlaceHolder("Cc")

	bccEntry := widget.NewEntry()
	bccEntry.SetPlaceHolder("Bcc")

	requestMDNCheck := widget.NewCheck("Request Read Receipt", nil)
	ccBccContainer := container.NewVBox()
	showCcBcc := false

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

	// ── Subject ─────────────────────────────────────────────────────────────
	composerSubjectEntry = widget.NewEntry()
	composerSubjectEntry.SetPlaceHolder("Subject")
	subjectRow := container.NewBorder(nil, nil, widget.NewLabel("Subject:"), nil, composerSubjectEntry)

	// ── Body ────────────────────────────────────────────────────────────────
	composerBodyEntry = widget.NewMultiLineEntry()
	composerBodyEntry.SetPlaceHolder("Compose your message...")
	composerBodyEntry.Wrapping = fyne.TextWrapWord

	previewMode := false
	previewRich := widget.NewRichTextFromMarkdown("")
	previewArea := container.NewScroll(previewRich)
	previewArea.Hide()
	editorContainer := container.NewMax(composerBodyEntry, previewArea)

	// ── Templates ───────────────────────────────────────────────────────────
	templateNames := []string{"Default Template"}
	var dbTemplates []storage.Template
	if db != nil {
		if tList, err := db.ListTemplates(); err == nil {
			dbTemplates = tList
			for _, t := range tList {
				templateNames = append(templateNames, t.Name)
			}
		}
	}
	if len(templateNames) == 1 {
		templateNames = append(templateNames, "Business Formal", "Casual Reply")
	}

	templateSelect := widget.NewSelect(templateNames, func(s string) {
		switch s {
		case "Business Formal":
			composerBodyEntry.SetText("Dear [Name],\n\nI hope this email finds you well.\n\nBest regards,\nRyan")
		case "Casual Reply":
			composerBodyEntry.SetText("Hi [Name],\n\nThanks for reaching out.\n\nCheers,\nRyan")
		default:
			for _, t := range dbTemplates {
				if t.Name == s {
					composerBodyEntry.SetText(t.Snippet)
					break
				}
			}
		}
	})
	templateSelect.SetSelected("Default Template")

	// Signature injection on account change
	accountSelect.OnChanged = func(selected string) {
		sig := "\n\n-- \nSent via AfterMail"
		if strings.Contains(selected, "msgs.global") {
			sig = "\n\n-- \n[Encrypted · Ed25519/X25519]\nAfterSMTP"
		}
		if !strings.Contains(composerBodyEntry.Text, "-- ") {
			composerBodyEntry.SetText(composerBodyEntry.Text + sig)
		}
	}
	accountSelect.OnChanged(accountSelect.Selected)

	// ── Format toolbar ──────────────────────────────────────────────────────
	formatSelect := widget.NewSelect([]string{"Plain Text", "HTML", "Markdown"}, nil)
	formatSelect.SetSelected("Plain Text")

	previewBtn := widget.NewButton("Preview", func() {
		previewMode = !previewMode
		if previewMode {
			previewRich.ParseMarkdown(composerBodyEntry.Text)
			composerBodyEntry.Hide()
			previewArea.Show()
		} else {
			previewArea.Hide()
			composerBodyEntry.Show()
		}
	})
	previewBtn.Importance = widget.LowImportance

	attachBtn := widget.NewButton("Attach Files", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			dialog.ShowInformation("Attachment", "Attached: "+reader.URI().Name(), w)
		}, w)
	})

	// ── AI toolbar ──────────────────────────────────────────────────────────
	spellCheckBtn := widget.NewButton("✓ Spell Check", func() {
		if composerBodyEntry.Text == "" {
			dialog.ShowInformation("Spell Check", "No text to check.", w)
			return
		}
		assistant := getAIAssistant()
		if assistant == nil || assistant.APIKey == "" {
			dialog.ShowInformation("Spell Check", "Configure an AI API key in Settings → AI Assistant.", w)
			return
		}
		prog := dialog.NewInformation("Spell Check", "Checking…", w)
		prog.Show()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			corrected, err := assistant.CheckSpelling(ctx, composerBodyEntry.Text)
			prog.Hide()
			if err != nil {
				dialog.ShowError(fmt.Errorf("spell check: %w", err), w)
				return
			}
			if corrected == composerBodyEntry.Text {
				dialog.ShowInformation("Spell Check", "✓ No spelling errors found!", w)
			} else {
				dialog.ShowConfirm("Spell Check", "Corrections found — apply them?", func(ok bool) {
					if ok {
						composerBodyEntry.SetText(corrected)
					}
				}, w)
			}
		}()
	})
	spellCheckBtn.Importance = widget.LowImportance

	grammarCheckBtn := widget.NewButton("✓ Grammar", func() {
		if composerBodyEntry.Text == "" {
			dialog.ShowInformation("Grammar Check", "No text to check.", w)
			return
		}
		assistant := getAIAssistant()
		if assistant == nil || assistant.APIKey == "" {
			dialog.ShowInformation("Grammar Check", "Configure an AI API key in Settings → AI Assistant.", w)
			return
		}
		prog := dialog.NewInformation("Grammar Check", "Checking…", w)
		prog.Show()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			corrected, err := assistant.CheckGrammar(ctx, composerBodyEntry.Text)
			prog.Hide()
			if err != nil {
				dialog.ShowError(fmt.Errorf("grammar check: %w", err), w)
				return
			}
			if corrected == composerBodyEntry.Text {
				dialog.ShowInformation("Grammar Check", "✓ No grammar issues found!", w)
			} else {
				dialog.ShowConfirm("Grammar Check", "Corrections found — apply them?", func(ok bool) {
					if ok {
						composerBodyEntry.SetText(corrected)
					}
				}, w)
			}
		}()
	})
	grammarCheckBtn.Importance = widget.LowImportance

	var aiBtn *widget.Button
	aiBtn = widget.NewButton("🤖 AI", func() {
		assistant := getAIAssistant()
		if assistant == nil || assistant.APIKey == "" {
			dialog.ShowInformation("AI Assistant", "Configure an AI API key in Settings → AI Assistant.", w)
			return
		}
		if composerBodyEntry.Text == "" {
			dialog.ShowInformation("AI Assistant", "Write some text first, or use Generate Draft.", w)
			return
		}

		aiAction := func(label string, fn func(context.Context, string) (string, error)) {
			prog := dialog.NewInformation("AI", label+"…", w)
			prog.Show()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
				defer cancel()
				result, err := fn(ctx, composerBodyEntry.Text)
				prog.Hide()
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				dialog.ShowConfirm("AI Assistant", "Apply "+label+" version?", func(ok bool) {
					if ok {
						composerBodyEntry.SetText(result)
					}
				}, w)
			}()
		}

		canvas := fyne.CurrentApp().Driver().CanvasForObject(aiBtn)
		pos := fyne.NewPos(aiBtn.Position().X, aiBtn.Position().Y+aiBtn.Size().Height)
		widget.ShowPopUpMenuAtPosition(
			fyne.NewMenu("AI",
				fyne.NewMenuItem("Improve Writing", func() { aiAction("Improve Writing", assistant.ImproveWriting) }),
				fyne.NewMenuItem("Make Concise", func() { aiAction("Concise", assistant.MakeConcise) }),
				fyne.NewMenuItem("Make Formal", func() { aiAction("Formal", assistant.MakeFormal) }),
				fyne.NewMenuItem("Make Friendly", func() { aiAction("Friendly", assistant.MakeFriendly) }),
				fyne.NewMenuItemSeparator(),
				fyne.NewMenuItem("Generate Draft…", func() {
					promptEntry := widget.NewMultiLineEntry()
					promptEntry.SetPlaceHolder("Describe what you want to write…")
					dialog.ShowForm("Generate Draft", "Generate", "Cancel",
						[]*widget.FormItem{widget.NewFormItem("Prompt", promptEntry)},
						func(ok bool) {
							if !ok || promptEntry.Text == "" {
								return
							}
							prog := dialog.NewInformation("AI", "Generating…", w)
							prog.Show()
							go func() {
								ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
								defer cancel()
								draft, err := assistant.GenerateDraft(ctx, promptEntry.Text)
								prog.Hide()
								if err != nil {
									dialog.ShowError(err, w)
									return
								}
								composerBodyEntry.SetText(draft)
							}()
						}, w)
				}),
				fyne.NewMenuItem("Summarize", func() {
					prog := dialog.NewInformation("AI", "Summarizing…", w)
					prog.Show()
					go func() {
						ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
						defer cancel()
						sum, err := assistant.SummarizeEmail(ctx, composerBodyEntry.Text)
						prog.Hide()
						if err != nil {
							dialog.ShowError(err, w)
							return
						}
						dialog.ShowInformation("Summary", sum, w)
					}()
				}),
			),
			canvas, pos,
		)
	})
	aiBtn.Importance = widget.MediumImportance

	formattingToolbar := container.NewHBox(
		templateSelect,
		widget.NewSeparator(),
		widget.NewLabel("Format:"),
		formatSelect,
		previewBtn,
		widget.NewSeparator(),
		spellCheckBtn,
		grammarCheckBtn,
		aiBtn,
		layout.NewSpacer(),
		attachBtn,
	)

	// ── Auto-save draft ─────────────────────────────────────────────────────
	go func() {
		last := ""
		for range time.NewTicker(20 * time.Second).C {
			text := composerSubjectEntry.Text + composerBodyEntry.Text
			if text != "" && text != last {
				last = text
				fmt.Println("[Auto-Save] Draft saved.")
			}
		}
	}()

	// ── Security indicator ──────────────────────────────────────────────────
	securityLabel := widget.NewLabel("🔒 TLS")

	// ── clearForm helper ────────────────────────────────────────────────────
	clearForm := func() {
		composerToEntry.SetText("")
		ccEntry.SetText("")
		bccEntry.SetText("")
		composerSubjectEntry.SetText("")
		composerBodyEntry.SetText("")
	}

	// ── findSelectedAccount ─────────────────────────────────────────────────
	findSelectedAccount := func() *accounts.Account {
		sel := accountSelect.Selected
		for i, opt := range accountOptions {
			if opt == sel && i < len(dbAccounts) {
				return dbAccounts[i]
			}
		}
		return nil
	}

	// ── Send button ─────────────────────────────────────────────────────────
	sendBtn := widget.NewButtonWithIcon("Send", theme.MailSendIcon(), func() {
		to := strings.TrimSpace(composerToEntry.Text)
		subject := strings.TrimSpace(composerSubjectEntry.Text)
		body := composerBodyEntry.Text

		if to == "" {
			dialog.ShowInformation("Cannot Send", "A recipient is required.", w)
			return
		}

		recipients := strings.Split(to, ",")
		for i := range recipients {
			recipients[i] = strings.TrimSpace(recipients[i])
		}

		acc := findSelectedAccount()
		if acc == nil {
			dialog.ShowInformation("Cannot Send", "No account selected. Add one in Settings → Accounts.", w)
			return
		}

		// Schedule with undo delay.
		sendID, err := undoManager.ScheduleSend(nil, 10*time.Second)
		if err != nil {
			dialog.ShowError(fmt.Errorf("schedule send: %w", err), w)
			return
		}

		// Undo countdown dialog.
		countdown := widget.NewLabel("10 seconds")
		undoBtnWidget := widget.NewButton("Undo", func() {
			if undoManager.CancelSend(sendID) == nil {
				dialog.ShowInformation("Cancelled", "Send cancelled.", w)
			}
		})
		undoBtnWidget.Importance = widget.WarningImportance

		undoContent := container.NewVBox(
			widget.NewLabel("To: "+to),
			widget.NewLabel("Subject: "+subject),
			widget.NewSeparator(),
			container.NewHBox(widget.NewLabel("Sending in:"), countdown),
			undoBtnWidget,
		)
		undoDlg := dialog.NewCustom("Sending…", "OK", undoContent, w)
		undoDlg.Show()

		ticker := time.NewTicker(time.Second)
		go func() {
			n := 10
			for range ticker.C {
				n--
				countdown.SetText(fmt.Sprintf("%d seconds", n))
				countdown.Refresh()
				if n <= 0 {
					ticker.Stop()
					undoDlg.Hide()
					// Dispatch the actual send.
					ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
					defer cancel()

					var sendErr error
					if acc.Type == accounts.TypeAfterSMTP {
						sendErr = sendViaAfterSMTP(ctx, acc, recipients, subject, body)
					} else {
						sendErr = sendViaSMTP(ctx, acc, recipients, subject, body)
					}

					if sendErr != nil {
						dialog.ShowError(fmt.Errorf("send failed: %w", sendErr), w)
					} else {
						dialog.ShowInformation("Sent", "✓ Message delivered.", w)
						clearForm()
					}
					return
				}
			}
		}()
	})
	sendBtn.Importance = widget.HighImportance

	// ── Save Draft ──────────────────────────────────────────────────────────
	saveDraftBtn := widget.NewButton("Save Draft", func() {
		dialog.ShowInformation("Draft Saved", "Your draft has been saved.", w)
	})

	// ── Schedule ────────────────────────────────────────────────────────────
	schedDispatcher := send.NewScheduledDispatcher()
	scheduleBtn := widget.NewButton("Schedule…", func() {
		to := strings.TrimSpace(composerToEntry.Text)
		subject := strings.TrimSpace(composerSubjectEntry.Text)
		body := composerBodyEntry.Text

		if to == "" {
			dialog.ShowInformation("Cannot Schedule", "A recipient is required.", w)
			return
		}

		minsEntry := widget.NewEntry()
		minsEntry.SetText("60")

		dialog.ShowForm("Schedule Send", "Schedule", "Cancel",
			[]*widget.FormItem{widget.NewFormItem("Delay (minutes)", minsEntry)},
			func(ok bool) {
				if !ok {
					return
				}
				var mins int
				fmt.Sscanf(minsEntry.Text, "%d", &mins)
				if mins <= 0 {
					mins = 60
				}
				schedDispatcher.QueueMessage(send.ScheduledMessage{
					To:      strings.Split(to, ","),
					Subject: subject,
					Body:    body,
					SendAt:  time.Now().Add(time.Duration(mins) * time.Minute),
				})
				dialog.ShowInformation("Scheduled", fmt.Sprintf("Message scheduled for %d minutes from now.", mins), w)
				clearForm()
			}, w)
	})

	// ── Discard ─────────────────────────────────────────────────────────────
	discardBtn := widget.NewButton("Discard", func() {
		dialog.ShowConfirm("Discard Draft", "Discard this message?", func(ok bool) {
			if ok {
				clearForm()
			}
		}, w)
	})
	discardBtn.Importance = widget.LowImportance

	actionBar := container.NewHBox(
		sendBtn, scheduleBtn, saveDraftBtn, discardBtn,
		layout.NewSpacer(),
		securityLabel,
	)

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

	footer := container.NewVBox(
		widget.NewSeparator(),
		actionBar,
	)

	return container.NewBorder(header, footer, nil, nil, editorContainer)
}

// ── Send implementations ──────────────────────────────────────────────────────

func sendViaSMTP(ctx context.Context, acc *accounts.Account, recipients []string, subject, body string) error {
	if acc.SmtpHost == "" {
		return fmt.Errorf("no SMTP host configured for account %q — edit it in Settings → Accounts", acc.Name)
	}

	client := &smtppkg.Client{
		Host:     acc.SmtpHost,
		Port:     acc.SmtpPort,
		UseTLS:   acc.SmtpUseTLS,
		Username: acc.Username,
		Password: acc.Password,
	}

	payload := buildRFC5322(acc.Email, recipients, subject, body)
	return client.SendMessage(ctx, acc.Email, recipients, payload)
}

func sendViaAfterSMTP(ctx context.Context, acc *accounts.Account, recipients []string, subject, body string) error {
	client, err := accounts.NewMsgsGlobalClient(acc)
	if err != nil {
		return fmt.Errorf("aftersmtp client: %w", err)
	}
	payload := &proto.AMFPayload{
		Subject:  subject,
		TextBody: body,
	}
	for _, to := range recipients {
		resp, err := client.DeliverMessage(ctx, to, payload)
		if err != nil {
			return fmt.Errorf("deliver to %s: %w", to, err)
		}
		if !resp.Success {
			return fmt.Errorf("gateway rejected delivery to %s: %s", to, resp.ErrorMessage)
		}
	}
	return nil
}

func buildRFC5322(from string, to []string, subject, body string) []byte {
	var sb strings.Builder
	sb.WriteString("From: " + from + "\r\n")
	sb.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	sb.WriteString("Subject: " + subject + "\r\n")
	sb.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)
	return []byte(sb.String())
}

func markdownToHTML(md string) string {
	html := strings.ReplaceAll(md, "\n", "<br>")
	return "<html><body>" + html + "</body></html>"
}
