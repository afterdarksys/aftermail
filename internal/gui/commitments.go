package gui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/afterdarksys/aftermail/pkg/storage"
)

// buildCommitmentsView builds the Commitment Ledger tab.
// It shows all tracked promises, action items, and deadlines extracted
// from email threads, with resolve/snooze controls.
func buildCommitmentsView(w fyne.Window, db *storage.DB) fyne.CanvasObject {
	if db == nil {
		return widget.NewLabel("Database not available.")
	}

	// --- Filter bar ---
	filterSelect := widget.NewSelect(
		[]string{"Open", "Overdue", "Resolved", "Snoozed", "All"},
		nil,
	)
	filterSelect.SetSelected("Open")

	// --- Commitment list ---
	var commitments []storage.Commitment
	var listWidget *widget.List

	refreshList := func() {
		var err error
		status := ""
		switch filterSelect.Selected {
		case "Open":
			status = "open"
		case "Resolved":
			status = "resolved"
		case "Snoozed":
			status = "snoozed"
		case "Overdue":
			var overdue []storage.Commitment
			overdue, err = db.OverdueCommitments()
			if err == nil {
				commitments = overdue
				listWidget.Refresh()
			}
			return
		}

		commitments, err = db.ListCommitments(status, "")
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		listWidget.Refresh()
	}

	filterSelect.OnChanged = func(_ string) { refreshList() }

	listWidget = widget.NewList(
		func() int { return len(commitments) },
		func() fyne.CanvasObject {
			return container.NewBorder(
				nil, nil, nil,
				container.NewHBox(
					widget.NewButtonWithIcon("", theme.ConfirmIcon(), nil),
					widget.NewButtonWithIcon("", theme.MoreHorizontalIcon(), nil),
				),
				container.NewVBox(
					widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
					widget.NewLabel(""),
				),
			)
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i >= len(commitments) {
				return
			}
			c := commitments[i]

			border := obj.(*fyne.Container)
			content := border.Objects[0].(*fyne.Container)
			buttons := border.Objects[1].(*fyne.Container)

			titleLabel := content.Objects[0].(*widget.Label)
			metaLabel := content.Objects[1].(*widget.Label)

			// Build title with kind badge
			kindBadge := kindEmoji(c.Kind)
			titleLabel.SetText(kindBadge + " " + truncateStr(c.Text, 80))
			titleLabel.TextStyle = fyne.TextStyle{Bold: true}

			// Meta line: sender + due date
			meta := fmt.Sprintf("From: %s  |  %s", c.Sender, formatDate(c.ExtractedAt))
			if c.HasDueDate && !c.DueDate.IsZero() {
				dueMark := ""
				if time.Now().After(c.DueDate) && c.Status == "open" {
					dueMark = " OVERDUE"
				}
				meta += fmt.Sprintf("  |  Due: %s%s", c.DueDate.Format("Jan 2"), dueMark)
			}
			metaLabel.SetText(meta)

			// Resolve button
			resolveBtn := buttons.Objects[0].(*widget.Button)
			resolveBtn.OnTapped = func() {
				if err := db.ResolveCommitment(c.ID); err != nil {
					dialog.ShowError(err, w)
					return
				}
				refreshList()
			}

			// Snooze / notes button
			moreBtn := buttons.Objects[1].(*widget.Button)
			moreBtn.OnTapped = func() {
				showCommitmentDetail(w, db, c, refreshList)
			}
		},
	)

	// --- Stats bar ---
	statsLabel := widget.NewLabel("")
	updateStats := func() {
		open, _ := db.ListCommitments("open", "")
		overdue, _ := db.OverdueCommitments()
		statsLabel.SetText(fmt.Sprintf("Open: %d  |  Overdue: %d", len(open), len(overdue)))
	}

	// Initial load
	refreshList()
	updateStats()

	// --- Header ---
	header := container.NewBorder(
		nil, nil,
		widget.NewLabelWithStyle("Commitment Ledger", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		statsLabel,
		container.NewHBox(
			widget.NewLabel("Show:"),
			filterSelect,
			layout.NewSpacer(),
		),
	)

	// --- Info panel ---
	infoText := "AI extracts promises, action items, and deadlines from your emails.\nMark items resolved when done, or snooze them for later."
	info := widget.NewLabelWithStyle(infoText, fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

	return container.NewBorder(
		container.NewVBox(header, info, widget.NewSeparator()),
		nil, nil, nil,
		listWidget,
	)
}

// showCommitmentDetail shows a detail dialog for a commitment with notes editing and snooze.
func showCommitmentDetail(w fyne.Window, db *storage.DB, c storage.Commitment, refresh func()) {
	kindLabel := widget.NewLabelWithStyle(
		kindEmoji(c.Kind)+" "+strings.ToUpper(strings.ReplaceAll(c.Kind, "_", " ")),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	textLabel := widget.NewLabel(c.Text)
	textLabel.Wrapping = fyne.TextWrapWord

	senderLabel := widget.NewLabel("From: " + c.Sender)
	subjectLabel := widget.NewLabel("Re: " + c.Subject)
	extractedLabel := widget.NewLabel("Extracted: " + formatDate(c.ExtractedAt))
	confidenceLabel := widget.NewLabel(fmt.Sprintf("Confidence: %.0f%%", c.Confidence*100))

	dueLabel := widget.NewLabel("")
	if c.HasDueDate && !c.DueDate.IsZero() {
		dueLabel.SetText("Due: " + c.DueDate.Format("Monday, January 2, 2006"))
	}

	notesEntry := widget.NewMultiLineEntry()
	notesEntry.SetPlaceHolder("Add personal notes about this commitment...")
	notesEntry.SetText(c.Notes)
	notesEntry.SetMinRowsVisible(3)

	form := container.NewVBox(
		kindLabel,
		widget.NewSeparator(),
		textLabel,
		widget.NewSeparator(),
		senderLabel, subjectLabel, extractedLabel, confidenceLabel,
		dueLabel,
		widget.NewSeparator(),
		widget.NewLabel("Notes:"),
		notesEntry,
	)

	dialog.ShowCustomConfirm(
		"Commitment Detail",
		"Resolve",
		"Snooze",
		form,
		func(resolve bool) {
			// Save notes first
			_ = db.UpdateCommitmentNotes(c.ID, notesEntry.Text)

			if resolve {
				_ = db.ResolveCommitment(c.ID)
			} else {
				_ = db.SnoozeCommitment(c.ID)
			}
			refresh()
		},
		w,
	)
}

func kindEmoji(kind string) string {
	switch kind {
	case "promise_made":
		return "📋"
	case "promise_received":
		return "🤝"
	case "question":
		return "❓"
	case "deadline":
		return "⏰"
	case "follow_up":
		return "🔔"
	case "request":
		return "📌"
	default:
		return "•"
	}
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	now := time.Now()
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return "Today " + t.Format("3:04 PM")
	}
	if t.Year() == now.Year() {
		return t.Format("Jan 2")
	}
	return t.Format("Jan 2, 2006")
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
