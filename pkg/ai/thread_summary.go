package ai

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ThreadMessage is the minimal representation of one message in a thread.
type ThreadMessage struct {
	From    string
	To      string
	Date    time.Time
	Subject string
	Body    string
}

// ThreadSummary holds the output of a thread analysis.
type ThreadSummary struct {
	// OneLiner is a one-sentence TL;DR of the whole thread.
	OneLiner string
	// Arc is a 2-3 sentence narrative of how the conversation evolved.
	Arc string
	// PendingItems lists unresolved questions or action items.
	PendingItems []string
	// KeyDecisions lists important conclusions or decisions reached.
	KeyDecisions []string
	// WaitingOn identifies who needs to act next, if determinable.
	WaitingOn string
	// ChangeSinceLastRead summarizes only what is new since a given time (may be empty).
	ChangeSinceLastRead string
}

// SummarizeThread produces a concise summary of an email thread.
func (a *Assistant) SummarizeThread(ctx context.Context, messages []ThreadMessage) (*ThreadSummary, error) {
	if len(messages) == 0 {
		return &ThreadSummary{OneLiner: "Empty thread."}, nil
	}

	prompt := buildThreadSummaryPrompt(messages, time.Time{})
	raw, err := a.query(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("thread summary failed: %w", err)
	}

	return parseThreadSummary(raw), nil
}

// WhatChangedSinceLastRead returns a focused summary of only the messages after lastRead.
func (a *Assistant) WhatChangedSinceLastRead(ctx context.Context, messages []ThreadMessage, lastRead time.Time) (*ThreadSummary, error) {
	if len(messages) == 0 {
		return &ThreadSummary{OneLiner: "No new messages."}, nil
	}

	// Partition: messages before and after lastRead
	var before, after []ThreadMessage
	for _, m := range messages {
		if m.Date.After(lastRead) {
			after = append(after, m)
		} else {
			before = append(before, m)
		}
	}

	if len(after) == 0 {
		return &ThreadSummary{OneLiner: "Nothing new since you last read this thread."}, nil
	}

	prompt := buildThreadSummaryPrompt(messages, lastRead)
	raw, err := a.query(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("thread delta summary failed: %w", err)
	}

	summary := parseThreadSummary(raw)

	// Also generate a focused "what changed" blurb using only new messages
	if len(before) > 0 {
		deltaPrompt := buildDeltaPrompt(before, after)
		delta, err := a.query(ctx, deltaPrompt)
		if err == nil {
			summary.ChangeSinceLastRead = strings.TrimSpace(delta)
		}
	}

	return summary, nil
}

func buildThreadSummaryPrompt(messages []ThreadMessage, lastRead time.Time) string {
	var sb strings.Builder
	sb.WriteString("Summarize this email thread.\n\n")

	if !lastRead.IsZero() {
		sb.WriteString(fmt.Sprintf("The user last read this thread at %s. Mark messages after that as [NEW].\n\n",
			lastRead.Format("Jan 2, 3:04 PM")))
	}

	sb.WriteString("Thread:\n")
	sb.WriteString(strings.Repeat("─", 60) + "\n")
	for i, m := range messages {
		marker := ""
		if !lastRead.IsZero() && m.Date.After(lastRead) {
			marker = " [NEW]"
		}
		sb.WriteString(fmt.Sprintf("[%d]%s %s → %s (%s)\n%s\n\n",
			i+1, marker, m.From, m.To, m.Date.Format("Jan 2, 3:04 PM"), m.Body))
	}
	sb.WriteString(strings.Repeat("─", 60) + "\n\n")

	sb.WriteString(`Return your analysis as plain text with these exact labeled sections (no JSON, no markdown):

ONE_LINER: (one sentence TL;DR)
ARC: (2-3 sentences on how the conversation evolved)
PENDING: (bullet list of unresolved questions/actions, or "None")
DECISIONS: (bullet list of conclusions reached, or "None")
WAITING_ON: (who needs to act next, or "Nobody")
`)
	return sb.String()
}

func buildDeltaPrompt(before, after []ThreadMessage) string {
	var sb strings.Builder
	sb.WriteString("Given this background context:\n\n")
	for _, m := range before {
		sb.WriteString(fmt.Sprintf("%s → %s: %s\n", m.From, m.To, truncate(m.Body, 200)))
	}
	sb.WriteString("\nThe following new messages arrived:\n\n")
	for _, m := range after {
		sb.WriteString(fmt.Sprintf("%s → %s (%s): %s\n",
			m.From, m.To, m.Date.Format("Jan 2, 3:04 PM"), m.Body))
	}
	sb.WriteString("\nIn one or two sentences, what changed or was added since the earlier messages? Be specific.")
	return sb.String()
}

func parseThreadSummary(raw string) *ThreadSummary {
	s := &ThreadSummary{}
	lines := strings.Split(raw, "\n")

	var currentSection string
	var pendingBuf, decisionsBuf []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "ONE_LINER:"):
			s.OneLiner = strings.TrimSpace(strings.TrimPrefix(line, "ONE_LINER:"))
			currentSection = "one_liner"
		case strings.HasPrefix(line, "ARC:"):
			s.Arc = strings.TrimSpace(strings.TrimPrefix(line, "ARC:"))
			currentSection = "arc"
		case strings.HasPrefix(line, "PENDING:"):
			currentSection = "pending"
			rest := strings.TrimSpace(strings.TrimPrefix(line, "PENDING:"))
			if rest != "" && rest != "None" {
				pendingBuf = append(pendingBuf, strings.TrimPrefix(rest, "- "))
			}
		case strings.HasPrefix(line, "DECISIONS:"):
			currentSection = "decisions"
			rest := strings.TrimSpace(strings.TrimPrefix(line, "DECISIONS:"))
			if rest != "" && rest != "None" {
				decisionsBuf = append(decisionsBuf, strings.TrimPrefix(rest, "- "))
			}
		case strings.HasPrefix(line, "WAITING_ON:"):
			s.WaitingOn = strings.TrimSpace(strings.TrimPrefix(line, "WAITING_ON:"))
			currentSection = "waiting"
		default:
			// continuation lines
			switch currentSection {
			case "arc":
				s.Arc += " " + line
			case "pending":
				if line != "None" {
					pendingBuf = append(pendingBuf, strings.TrimPrefix(line, "- "))
				}
			case "decisions":
				if line != "None" {
					decisionsBuf = append(decisionsBuf, strings.TrimPrefix(line, "- "))
				}
			}
		}
	}

	s.PendingItems = pendingBuf
	s.KeyDecisions = decisionsBuf
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
