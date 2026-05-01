// Package commitments extracts and tracks action items and promises from email threads.
package commitments

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Kind classifies what type of commitment was found.
type Kind string

const (
	KindPromiseMade     Kind = "promise_made"     // "I will send the report by Friday"
	KindPromiseReceived Kind = "promise_received"  // "I'll get back to you next week"
	KindQuestion        Kind = "question"          // Unanswered question waiting on reply
	KindDeadline        Kind = "deadline"          // An explicit date/deadline mentioned
	KindFollowUp        Kind = "follow_up"         // "Let me know if you have questions"
	KindRequest         Kind = "request"           // Someone asked you to do something
)

// Status tracks the lifecycle of a commitment.
type Status string

const (
	StatusOpen     Status = "open"
	StatusResolved Status = "resolved"
	StatusExpired  Status = "expired"
	StatusSnoozed  Status = "snoozed"
)

// Commitment represents a single extracted action item or promise.
type Commitment struct {
	ID          int64     `json:"id"`
	MessageID   string    `json:"message_id"`
	ThreadID    string    `json:"thread_id"`
	Sender      string    `json:"sender"`
	Recipient   string    `json:"recipient"`
	Subject     string    `json:"subject"`
	Kind        Kind      `json:"kind"`
	Text        string    `json:"text"`        // The extracted commitment text
	DueDate     time.Time `json:"due_date"`    // Zero if no date mentioned
	HasDueDate  bool      `json:"has_due_date"`
	Status      Status    `json:"status"`
	Confidence  float64   `json:"confidence"` // 0.0–1.0
	ExtractedAt time.Time `json:"extracted_at"`
	ResolvedAt  time.Time `json:"resolved_at"`
	Notes       string    `json:"notes"`
}

// ExtractionResult holds all commitments found in a single message.
type ExtractionResult struct {
	MessageID   string       `json:"message_id"`
	Commitments []Commitment `json:"commitments"`
}

// MessageInput is the minimal email data needed for extraction.
type MessageInput struct {
	ID        string
	ThreadID  string
	Sender    string
	Recipient string
	Subject   string
	Body      string
	Date      time.Time
}

// AIExtractor uses an AI backend to extract commitments.
type AIExtractor struct {
	query func(ctx context.Context, prompt string) (string, error)
}

// NewAIExtractor creates an extractor that calls the provided query function.
// Pass Assistant.Query or any function with the same signature.
func NewAIExtractor(queryFn func(ctx context.Context, prompt string) (string, error)) *AIExtractor {
	return &AIExtractor{query: queryFn}
}

// Extract identifies all commitments in the given message.
func (e *AIExtractor) Extract(ctx context.Context, msg MessageInput) (*ExtractionResult, error) {
	prompt := buildExtractionPrompt(msg)
	raw, err := e.query(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("commitment extraction failed: %w", err)
	}

	// The model returns JSON — parse it
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var parsed struct {
		Commitments []struct {
			Kind       string  `json:"kind"`
			Text       string  `json:"text"`
			DueDate    string  `json:"due_date"`
			Confidence float64 `json:"confidence"`
		} `json:"commitments"`
	}

	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		// Graceful degradation: return empty result rather than hard error
		return &ExtractionResult{MessageID: msg.ID}, nil
	}

	result := &ExtractionResult{MessageID: msg.ID}
	now := time.Now()

	for _, c := range parsed.Commitments {
		commitment := Commitment{
			MessageID:   msg.ID,
			ThreadID:    msg.ThreadID,
			Sender:      msg.Sender,
			Recipient:   msg.Recipient,
			Subject:     msg.Subject,
			Kind:        Kind(c.Kind),
			Text:        c.Text,
			Confidence:  c.Confidence,
			Status:      StatusOpen,
			ExtractedAt: now,
		}

		if c.DueDate != "" && c.DueDate != "null" {
			// Try common date formats
			for _, layout := range []string{"2006-01-02", "January 2, 2006", "Jan 2, 2006"} {
				if t, err := time.Parse(layout, c.DueDate); err == nil {
					commitment.DueDate = t
					commitment.HasDueDate = true
					break
				}
			}
		}

		result.Commitments = append(result.Commitments, commitment)
	}

	return result, nil
}

// IsExpired returns true if a dated commitment is past its due date and still open.
func (c *Commitment) IsExpired() bool {
	return c.Status == StatusOpen && c.HasDueDate && time.Now().After(c.DueDate)
}

// IsMine returns true if the commitment is something the given user address must act on.
func (c *Commitment) IsMine(myEmail string) bool {
	switch c.Kind {
	case KindPromiseMade, KindRequest:
		return strings.EqualFold(c.Recipient, myEmail)
	case KindQuestion:
		return strings.EqualFold(c.Recipient, myEmail)
	default:
		return false
	}
}

func buildExtractionPrompt(msg MessageInput) string {
	return fmt.Sprintf(`Analyze the following email and extract all commitments, action items, promises, questions awaiting answers, and deadlines.

Email:
  From: %s
  To: %s
  Subject: %s
  Date: %s
  Body:
%s

Return a JSON object with this exact structure (no other text):
{
  "commitments": [
    {
      "kind": "promise_made|promise_received|question|deadline|follow_up|request",
      "text": "exact short description of the commitment",
      "due_date": "YYYY-MM-DD or null",
      "confidence": 0.0-1.0
    }
  ]
}

Rules:
- Only extract real actionable commitments, not vague pleasantries.
- "promise_made" = the FROM person promised to do something.
- "promise_received" = someone promised the recipient something.
- "question" = an explicit question in the email that hasn't been answered yet.
- "deadline" = a specific date mentioned as a deadline or target.
- "request" = the FROM person asked the recipient to do something.
- "follow_up" = a soft follow-up reminder ("let me know", "get back to me").
- Confidence: 1.0 = crystal clear, 0.5 = inferred, 0.3 = possible but uncertain.
- If there are no commitments, return {"commitments": []}.`, msg.Sender, msg.Recipient, msg.Subject, msg.Date.Format("Mon Jan 2, 2006"), msg.Body)
}
