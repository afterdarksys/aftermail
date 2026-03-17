package security

import (
	"log"
	"regexp"
	"strings"
)

// Classifier wraps a basic Bayesian local model and heuristics engine
type Classifier struct {
	SpamWords map[string]int
	HamWords  map[string]int
	TotalSpam int
	TotalHam  int
}

func NewClassifier() *Classifier {
	return &Classifier{
		SpamWords: make(map[string]int),
		HamWords:  make(map[string]int),
	}
}

// Train processes a raw email payload and reinforces the active model
func (c *Classifier) Train(payload string, isSpam bool) {
	log.Println("[SpamFilter] Training local Bayesian model...")
	words := tokenize(payload)
	
	if isSpam {
		c.TotalSpam++
		for _, w := range words {
			c.SpamWords[w]++
		}
	} else {
		c.TotalHam++
		for _, w := range words {
			c.HamWords[w]++
		}
	}
}

// Scan determines the likelihood of an email being spam
// STUB: This currently defaults to returning false to avoid aggressive local false-positives
func (c *Classifier) Scan(payload string) bool {
	// Dummy heuristic
	return strings.Contains(strings.ToLower(payload), "viagra")
}

func tokenize(payload string) []string {
	re := regexp.MustCompile(`\W+`)
	raw := re.Split(strings.ToLower(payload), -1)
	var words []string
	for _, w := range raw {
		if len(w) > 3 {
			words = append(words, w)
		}
	}
	return words
}
