package categorization

import (
	"context"
	"fmt"
	"strings"

	"github.com/afterdarksys/aftermail/pkg/ai"
	"github.com/afterdarksys/aftermail/pkg/accounts"
)

// Category represents an email category
type Category struct {
	Name        string
	Keywords    []string
	Priority    int
	Color       string
	AutoArchive bool
}

// DefaultCategories returns the default category set
func DefaultCategories() []Category {
	return []Category{
		{Name: "Work", Keywords: []string{"meeting", "deadline", "project", "urgent"}, Priority: 1, Color: "#FF6B6B"},
		{Name: "Personal", Keywords: []string{"family", "friend", "personal"}, Priority: 2, Color: "#4ECDC4"},
		{Name: "Finance", Keywords: []string{"invoice", "payment", "bank", "receipt", "transaction"}, Priority: 1, Color: "#45B7D1"},
		{Name: "Shopping", Keywords: []string{"order", "shipping", "delivery", "purchase"}, Priority: 3, Color: "#FFA07A"},
		{Name: "Social", Keywords: []string{"facebook", "twitter", "linkedin", "notification"}, Priority: 4, Color: "#98D8C8"},
		{Name: "Newsletters", Keywords: []string{"unsubscribe", "newsletter", "digest"}, Priority: 5, Color: "#95E1D3", AutoArchive: true},
		{Name: "Promotions", Keywords: []string{"sale", "discount", "offer", "deal", "%"}, Priority: 5, Color: "#F6B93B", AutoArchive: true},
		{Name: "Spam", Keywords: []string{"viagra", "casino", "prize", "winner", "click here"}, Priority: 6, Color: "#E74C3C"},
	}
}

// SmartCategorizer uses AI and rules to categorize emails
type SmartCategorizer struct {
	AI         *ai.Assistant
	Categories []Category
	UseAI      bool
}

// NewSmartCategorizer creates a new categorizer
func NewSmartCategorizer(assistant *ai.Assistant, useAI bool) *SmartCategorizer {
	return &SmartCategorizer{
		AI:         assistant,
		Categories: DefaultCategories(),
		UseAI:      useAI,
	}
}

// CategorizeMessage categorizes a message
func (sc *SmartCategorizer) CategorizeMessage(ctx context.Context, msg *accounts.Message) (string, float64, error) {
	// First try rule-based categorization
	ruleCategory, confidence := sc.categorizeByRules(msg)
	if confidence > 0.8 {
		return ruleCategory, confidence, nil
	}

	// If AI is available and enabled, use it for better accuracy
	if sc.UseAI && sc.AI != nil {
		aiCategory, aiConfidence, err := sc.categorizeByAI(ctx, msg)
		if err == nil && aiConfidence > confidence {
			return aiCategory, aiConfidence, nil
		}
	}

	return ruleCategory, confidence, nil
}

// categorizeByRules uses keyword matching
func (sc *SmartCategorizer) categorizeByRules(msg *accounts.Message) (string, float64) {
	text := strings.ToLower(msg.Subject + " " + msg.BodyPlain)

	bestCategory := "Inbox"
	bestScore := 0.0

	for _, cat := range sc.Categories {
		score := 0.0
		for _, keyword := range cat.Keywords {
			if strings.Contains(text, strings.ToLower(keyword)) {
				score += 1.0
			}
		}

		// Normalize score
		if len(cat.Keywords) > 0 {
			score = score / float64(len(cat.Keywords))
		}

		// Consider sender domain for certain categories
		if cat.Name == "Social" && sc.isSocialDomain(msg.Sender) {
			score += 0.5
		}
		if cat.Name == "Finance" && sc.isFinanceDomain(msg.Sender) {
			score += 0.5
		}

		if score > bestScore {
			bestScore = score
			bestCategory = cat.Name
		}
	}

	// Cap confidence at 0.85 for rule-based
	confidence := bestScore
	if confidence > 0.85 {
		confidence = 0.85
	}

	return bestCategory, confidence
}

// categorizeByAI uses AI to categorize
func (sc *SmartCategorizer) categorizeByAI(ctx context.Context, msg *accounts.Message) (string, float64, error) {
	prompt := fmt.Sprintf(`Categorize this email into ONE of these categories: Work, Personal, Finance, Shopping, Social, Newsletters, Promotions, Spam, or Inbox.

From: %s
Subject: %s
Body: %s

Return ONLY the category name and confidence (0-1) in format: Category|Confidence
Example: Work|0.92`, msg.Sender, msg.Subject, truncate(msg.BodyPlain, 500))

	response, err := sc.AI.GenerateDraft(ctx, prompt)
	if err != nil {
		return "", 0, err
	}

	// Parse response
	parts := strings.Split(strings.TrimSpace(response), "|")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid AI response format")
	}

	category := strings.TrimSpace(parts[0])
	var confidence float64
	fmt.Sscanf(parts[1], "%f", &confidence)

	return category, confidence, nil
}

// BatchCategorize categorizes multiple messages
func (sc *SmartCategorizer) BatchCategorize(ctx context.Context, messages []*accounts.Message) (map[int64]string, error) {
	results := make(map[int64]string)

	for _, msg := range messages {
		category, _, err := sc.CategorizeMessage(ctx, msg)
		if err != nil {
			continue
		}
		results[msg.ID] = category
	}

	return results, nil
}

// LearnFromUserAction learns from user corrections
func (sc *SmartCategorizer) LearnFromUserAction(msg *accounts.Message, correctCategory string) {
	// TODO: Implement machine learning to improve categorization
	// For now, just add keywords from the subject
	for i, cat := range sc.Categories {
		if cat.Name == correctCategory {
			// Extract potential keywords from subject
			words := strings.Fields(strings.ToLower(msg.Subject))
			for _, word := range words {
				if len(word) > 4 && !contains(cat.Keywords, word) {
					sc.Categories[i].Keywords = append(sc.Categories[i].Keywords, word)
				}
			}
		}
	}
}

// GetSuggestedActions returns suggested actions for a message
func (sc *SmartCategorizer) GetSuggestedActions(msg *accounts.Message, category string) []string {
	var actions []string

	for _, cat := range sc.Categories {
		if cat.Name == category {
			if cat.AutoArchive {
				actions = append(actions, "Auto-archive")
			}
			if cat.Priority <= 2 {
				actions = append(actions, "Mark important")
			}
			if cat.Name == "Spam" {
				actions = append(actions, "Move to spam", "Block sender")
			}
		}
	}

	return actions
}

// Helper functions
func (sc *SmartCategorizer) isSocialDomain(email string) bool {
	socialDomains := []string{"facebook.com", "twitter.com", "linkedin.com", "instagram.com", "tiktok.com"}
	for _, domain := range socialDomains {
		if strings.Contains(email, domain) {
			return true
		}
	}
	return false
}

func (sc *SmartCategorizer) isFinanceDomain(email string) bool {
	financeDomains := []string{"paypal.com", "stripe.com", "bank", "credit", "venmo.com"}
	for _, domain := range financeDomains {
		if strings.Contains(email, domain) {
			return true
		}
	}
	return false
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
