package gui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/afterdarksys/aftermail/pkg/fingerprint"
	"github.com/afterdarksys/aftermail/pkg/send"
	"github.com/afterdarksys/aftermail/pkg/storage"
)

// buildInsightsView builds the Sender Insights tab.
// Combines the Send Optimizer and Sender Fingerprinting features.
func buildInsightsView(w fyne.Window, db *storage.DB) fyne.CanvasObject {
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Send Optimizer", theme.MailSendIcon(), buildSendOptimizerView(w, db)),
		container.NewTabItemWithIcon("Sender Profiles", theme.AccountIcon(), buildFingerprintView(w, db)),
	)
	return tabs
}

// ─── Send Optimizer ───────────────────────────────────────────────────────────

func buildSendOptimizerView(w fyne.Window, db *storage.DB) fyne.CanvasObject {
	recipientEntry := widget.NewEntry()
	recipientEntry.SetPlaceHolder("recipient@example.com")

	resultCard := newInsightCard("Enter a recipient email to see the optimal send time.", "")

	analyzeBtn := widget.NewButtonWithIcon("Analyze", theme.SearchIcon(), func() {
		email := strings.TrimSpace(recipientEntry.Text)
		if email == "" {
			resultCard.setContent("Enter an email address first.", "")
			return
		}

		records := loadResponseRecords(db, email)
		profile := send.BuildProfile(email, records)
		suggestion := send.SuggestSendTime(profile, time.Now())
		summary := send.ProfileSummary(profile)

		title := fmt.Sprintf("Best time: %s", suggestion.RecommendedAt.Format("Mon Jan 2 at 3:04 PM"))
		body := fmt.Sprintf(
			"%s\n\nConfidence: %.0f%%\n\n%s",
			suggestion.Reason,
			suggestion.Confidence*100,
			summary,
		)
		resultCard.setContent(title, body)
	})

	header := widget.NewLabelWithStyle(
		"Send Optimizer",
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)
	subtitle := widget.NewLabelWithStyle(
		"Analyzes your contact's reply patterns to recommend the best time to send for maximum response rate.",
		fyne.TextAlignLeading,
		fyne.TextStyle{Italic: true},
	)
	subtitle.Wrapping = fyne.TextWrapWord

	inputRow := container.NewBorder(nil, nil,
		widget.NewLabel("Recipient:"),
		analyzeBtn,
		recipientEntry,
	)

	return container.NewVBox(
		header,
		subtitle,
		widget.NewSeparator(),
		inputRow,
		resultCard.widget,
		layout.NewSpacer(),
	)
}

// ─── Sender Fingerprinting ────────────────────────────────────────────────────

func buildFingerprintView(w fyne.Window, db *storage.DB) fyne.CanvasObject {
	senderEntry := widget.NewEntry()
	senderEntry.SetPlaceHolder("sender@example.com")

	resultCard := newInsightCard("Enter a sender email to analyze their writing profile.", "")

	riskBar := widget.NewProgressBar()
	riskBar.Min = 0
	riskBar.Max = 1
	riskLabel := widget.NewLabel("")

	analyzeBtn := widget.NewButtonWithIcon("Analyze Latest Message", theme.SearchIcon(), func() {
		email := strings.TrimSpace(senderEntry.Text)
		if email == "" {
			resultCard.setContent("Enter a sender email address first.", "")
			return
		}

		samples, latest := loadSenderSamples(db, email)
		if len(samples) < 3 {
			resultCard.setContent(
				"Not enough history",
				fmt.Sprintf("Only %d message(s) from %s. Need at least 3 to build a profile.", len(samples), email),
			)
			riskBar.SetValue(0)
			riskLabel.SetText("")
			return
		}

		baseline := fingerprint.Build(email, samples)

		if latest == nil {
			resultCard.setContent("Profile built", fmt.Sprintf(
				"Built baseline from %d messages.\nNo new message to score — a new message from this contact will be analyzed automatically.",
				len(samples),
			))
			return
		}

		score := fingerprint.Score(baseline, *latest)
		riskBar.SetValue(score.Total)
		riskLabel.SetText(fmt.Sprintf("Risk: %s (%.0f%%)", strings.ToUpper(score.Risk), score.Total*100))

		var lines []string
		lines = append(lines, fmt.Sprintf("Baseline from %d messages · Avg %d words/sentence · Vocab richness %.0f%%",
			baseline.SampleCount, int(baseline.AvgWordsPerSentence), baseline.VocabRichness*100))

		if len(score.Flags) > 0 {
			lines = append(lines, "\nAnomalies detected:")
			for _, f := range score.Flags {
				lines = append(lines, "  • "+f)
			}
		} else {
			lines = append(lines, "\nNo anomalies — message is consistent with this sender's historical style.")
		}

		resultCard.setContent(
			fmt.Sprintf("Style analysis: %s", email),
			strings.Join(lines, "\n"),
		)
	})

	header := widget.NewLabelWithStyle(
		"Sender Behavioral Profiles",
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)
	subtitle := widget.NewLabelWithStyle(
		"Builds a per-sender writing style baseline to detect account takeover and AI-generated impersonation attacks.",
		fyne.TextAlignLeading,
		fyne.TextStyle{Italic: true},
	)
	subtitle.Wrapping = fyne.TextWrapWord

	inputRow := container.NewBorder(nil, nil,
		widget.NewLabel("Sender:"),
		analyzeBtn,
		senderEntry,
	)

	riskRow := container.NewBorder(nil, nil,
		riskLabel,
		nil,
		riskBar,
	)

	return container.NewVBox(
		header,
		subtitle,
		widget.NewSeparator(),
		inputRow,
		riskRow,
		resultCard.widget,
		layout.NewSpacer(),
	)
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

// insightCard is a simple display card with a title and multi-line body.
type insightCard struct {
	titleLabel *widget.Label
	bodyLabel  *widget.Label
	widget     fyne.CanvasObject
}

func newInsightCard(title, body string) *insightCard {
	tl := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	bl := widget.NewLabel(body)
	bl.Wrapping = fyne.TextWrapWord

	c := &insightCard{titleLabel: tl, bodyLabel: bl}
	c.widget = container.NewVBox(widget.NewSeparator(), tl, bl, widget.NewSeparator())
	return c
}

func (c *insightCard) setContent(title, body string) {
	c.titleLabel.SetText(title)
	c.bodyLabel.SetText(body)
}

// loadResponseRecords fetches sent/received message pairs for building a send profile.
// Queries the messages table for sent messages and their replies.
func loadResponseRecords(db *storage.DB, contactEmail string) []send.MessageRecord {
	if db == nil {
		return nil
	}
	// Full implementation queries messages table joining on thread_id to find reply pairs:
	//   SELECT sent.received_at, reply.received_at
	//   FROM messages sent
	//   JOIN messages reply ON sent.thread_id = reply.thread_id
	//     AND reply.received_at > sent.received_at
	//     AND reply.sender LIKE ?
	//   WHERE sent.sender NOT LIKE ?
	// Stubbed until messages table is fully populated from IMAP sync.
	_ = contactEmail
	return nil
}

// loadSenderSamples returns historical message bodies from a sender for fingerprinting,
// plus the most recent message as "latest" to score against the baseline.
func loadSenderSamples(db *storage.DB, senderEmail string) ([]fingerprint.MessageSample, *fingerprint.MessageSample) {
	if db == nil {
		return nil, nil
	}
	// Full implementation:
	//   SELECT body_plain FROM messages WHERE sender LIKE ? ORDER BY received_at DESC LIMIT 50
	// Returns all but the most recent as baseline, most recent as the message to score.
	_ = senderEmail
	return nil, nil
}
