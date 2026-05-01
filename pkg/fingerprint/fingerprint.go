// Package fingerprint builds per-sender writing style baselines and scores new
// messages for behavioural anomalies — useful for detecting account takeover
// and AI-generated impersonation attacks.
package fingerprint

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"
)

// Baseline is a statistical writing-style profile built from a sender's
// historical messages.
type Baseline struct {
	// Sender is the email address this baseline belongs to.
	Sender string `json:"sender"`

	// SampleCount is the number of messages used to build the baseline.
	SampleCount int `json:"sample_count"`

	// AvgWordsPerSentence is the mean sentence length in words.
	AvgWordsPerSentence float64 `json:"avg_words_per_sentence"`

	// AvgWordLength is the mean character count of words used.
	AvgWordLength float64 `json:"avg_word_length"`

	// VocabRichness is the ratio of unique words to total words (0.0–1.0).
	VocabRichness float64 `json:"vocab_richness"`

	// TopWords are the 50 most-frequent words, used for style fingerprinting.
	TopWords []string `json:"top_words"`

	// GreetingPattern is the most common opening word/phrase.
	GreetingPattern string `json:"greeting_pattern"`

	// SignaturePattern is the most common closing phrase.
	SignaturePattern string `json:"signature_pattern"`

	// AvgBodyLength is the mean body character count.
	AvgBodyLength float64 `json:"avg_body_length"`

	// PunctuationRate is the mean number of punctuation chars per word.
	PunctuationRate float64 `json:"punctuation_rate"`

	// ExclamationRate is mean exclamation marks per message.
	ExclamationRate float64 `json:"exclamation_rate"`

	// QuestionRate is mean question marks per message.
	QuestionRate float64 `json:"question_rate"`

	// CapitalisationRate is ratio of ALLCAPS words to total words.
	CapitalisationRate float64 `json:"capitalisation_rate"`
}

// AnomalyScore holds the result of comparing a message against a baseline.
type AnomalyScore struct {
	// Total is the overall anomaly score (0.0 = identical style, 1.0 = maximally different).
	Total float64 `json:"total"`

	// Signals is a breakdown of which dimensions contributed.
	Signals map[string]float64 `json:"signals"`

	// Risk is a human-readable risk label derived from Total.
	Risk string `json:"risk"`

	// Flags are specific observations worth surfacing to the user.
	Flags []string `json:"flags"`
}

// MessageSample is the minimal input needed to build or score a fingerprint.
type MessageSample struct {
	Body string
}

// Build constructs a Baseline from a slice of historical messages.
// At least 5 messages are recommended; fewer than 3 returns an empty baseline.
func Build(sender string, messages []MessageSample) *Baseline {
	if len(messages) < 3 {
		return &Baseline{Sender: sender, SampleCount: len(messages)}
	}

	b := &Baseline{
		Sender:      sender,
		SampleCount: len(messages),
	}

	var (
		totalWPS        float64
		totalWordLen    float64
		totalBodyLen    float64
		totalPunct      float64
		totalExcl       float64
		totalQuestion   float64
		totalCaps       float64
		wordCount       int
		wordFreq        = make(map[string]int)
		greetingFreq    = make(map[string]int)
		signatureFreq   = make(map[string]int)
	)

	for _, m := range messages {
		body := strings.TrimSpace(m.Body)
		totalBodyLen += float64(len(body))

		sentences := splitSentences(body)
		words := tokenize(body)
		if len(words) == 0 {
			continue
		}

		// Sentence length
		if len(sentences) > 0 {
			totalWPS += float64(len(words)) / float64(len(sentences))
		}

		// Word length, frequency, capitalisation
		for _, w := range words {
			wordFreq[strings.ToLower(w)]++
			totalWordLen += float64(len(w))
			wordCount++
			if len(w) > 1 && w == strings.ToUpper(w) {
				totalCaps++
			}
		}

		// Punctuation rates
		for _, ch := range body {
			if unicode.IsPunct(ch) {
				totalPunct++
			}
			if ch == '!' {
				totalExcl++
			}
			if ch == '?' {
				totalQuestion++
			}
		}

		// Greeting (first non-empty line)
		for _, line := range strings.Split(body, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				greeting := firstWords(line, 3)
				greetingFreq[greeting]++
				break
			}
		}

		// Signature (last non-empty line)
		bodyLines := strings.Split(body, "\n")
		for i := len(bodyLines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(bodyLines[i])
			if line != "" {
				sig := firstWords(line, 3)
				signatureFreq[sig]++
				break
			}
		}
	}

	n := float64(len(messages))
	b.AvgWordsPerSentence = totalWPS / n
	b.AvgBodyLength = totalBodyLen / n
	b.ExclamationRate = totalExcl / n
	b.QuestionRate = totalQuestion / n

	if wordCount > 0 {
		b.AvgWordLength = totalWordLen / float64(wordCount)
		b.PunctuationRate = totalPunct / float64(wordCount)
		b.CapitalisationRate = totalCaps / float64(wordCount)
		b.VocabRichness = float64(len(wordFreq)) / float64(wordCount)
	}

	b.TopWords = topN(wordFreq, 50)
	b.GreetingPattern = topKey(greetingFreq)
	b.SignaturePattern = topKey(signatureFreq)

	return b
}

// Score compares a single new message against a pre-built baseline.
// Returns an AnomalyScore with a total 0.0–1.0 and per-signal breakdown.
func Score(b *Baseline, msg MessageSample) *AnomalyScore {
	if b.SampleCount < 3 {
		return &AnomalyScore{
			Total:   0,
			Risk:    "unknown",
			Signals: map[string]float64{},
			Flags:   []string{"Insufficient baseline data (< 3 messages)"},
		}
	}

	body := strings.TrimSpace(msg.Body)
	words := tokenize(body)
	sentences := splitSentences(body)
	signals := map[string]float64{}
	flags := []string{}

	// --- Sentence length ---
	var wps float64
	if len(sentences) > 0 && len(words) > 0 {
		wps = float64(len(words)) / float64(len(sentences))
	}
	signals["sentence_length"] = clampedDiff(wps, b.AvgWordsPerSentence, b.AvgWordsPerSentence*0.5+1)

	// --- Body length ---
	bodyLenDiff := clampedDiff(float64(len(body)), b.AvgBodyLength, b.AvgBodyLength*0.5+100)
	signals["body_length"] = bodyLenDiff

	// --- Word length ---
	var avgWL float64
	if len(words) > 0 {
		var total float64
		for _, w := range words {
			total += float64(len(w))
		}
		avgWL = total / float64(len(words))
	}
	signals["word_length"] = clampedDiff(avgWL, b.AvgWordLength, b.AvgWordLength*0.3+0.5)

	// --- Vocab richness ---
	wordFreq := make(map[string]int)
	for _, w := range words {
		wordFreq[strings.ToLower(w)]++
	}
	var vocabRichness float64
	if len(words) > 0 {
		vocabRichness = float64(len(wordFreq)) / float64(len(words))
	}
	signals["vocab_richness"] = clampedDiff(vocabRichness, b.VocabRichness, 0.15)

	// --- Punctuation rate ---
	var punctCount float64
	var exclCount float64
	var questionCount float64
	var capsCount float64
	for _, ch := range body {
		if unicode.IsPunct(ch) {
			punctCount++
		}
		if ch == '!' {
			exclCount++
		}
		if ch == '?' {
			questionCount++
		}
	}
	for _, w := range words {
		if len(w) > 1 && w == strings.ToUpper(w) {
			capsCount++
		}
	}

	var punctRate float64
	if len(words) > 0 {
		punctRate = punctCount / float64(len(words))
		signals["punctuation"] = clampedDiff(punctRate, b.PunctuationRate, b.PunctuationRate*0.5+0.05)
	}
	signals["exclamation"] = clampedDiff(exclCount, b.ExclamationRate, b.ExclamationRate*0.5+1)
	signals["caps_usage"] = clampedDiff(capsCount, b.CapitalisationRate*float64(len(words)), b.CapitalisationRate*float64(len(words))*0.5+1)

	// --- Top word overlap ---
	topWordSet := make(map[string]bool, len(b.TopWords))
	for _, w := range b.TopWords {
		topWordSet[w] = true
	}
	var overlap int
	for w := range wordFreq {
		if topWordSet[w] {
			overlap++
		}
	}
	overlapRate := 0.0
	if len(b.TopWords) > 0 {
		overlapRate = float64(overlap) / float64(len(b.TopWords))
	}
	// Low overlap = unusual vocabulary for this sender
	signals["vocab_overlap"] = 1.0 - overlapRate

	// --- Greeting match ---
	greeting := ""
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			greeting = strings.ToLower(firstWords(line, 3))
			break
		}
	}
	greetingMatch := 0.0
	if strings.EqualFold(greeting, b.GreetingPattern) {
		greetingMatch = 1.0
	}
	signals["greeting"] = 1.0 - greetingMatch

	// --- Compute weighted total ---
	weights := map[string]float64{
		"sentence_length": 0.15,
		"body_length":     0.10,
		"word_length":     0.15,
		"vocab_richness":  0.10,
		"vocab_overlap":   0.25,
		"punctuation":     0.05,
		"exclamation":     0.05,
		"caps_usage":      0.05,
		"greeting":        0.10,
	}

	total := 0.0
	totalWeight := 0.0
	for k, w := range weights {
		if v, ok := signals[k]; ok {
			total += v * w
			totalWeight += w
		}
	}
	if totalWeight > 0 {
		total /= totalWeight
	}

	// --- Human-readable flags ---
	if signals["vocab_overlap"] > 0.7 {
		flags = append(flags, "Unusual vocabulary — very few of this sender's characteristic words are present")
	}
	if signals["sentence_length"] > 0.6 {
		flags = append(flags, "Sentence length significantly different from sender's norm")
	}
	if signals["body_length"] > 0.7 {
		flags = append(flags, "Message length is unusually different from this sender's average")
	}
	if signals["greeting"] > 0.5 && b.GreetingPattern != "" {
		flags = append(flags, fmt.Sprintf("Greeting differs from sender's usual pattern (%q)", b.GreetingPattern))
	}
	if signals["caps_usage"] > 0.6 {
		flags = append(flags, "Abnormal use of ALL-CAPS words")
	}

	risk := riskLabel(total)

	return &AnomalyScore{
		Total:   math.Round(total*100) / 100,
		Signals: signals,
		Risk:    risk,
		Flags:   flags,
	}
}

// riskLabel maps a 0–1 score to a human label.
func riskLabel(score float64) string {
	switch {
	case score < 0.25:
		return "normal"
	case score < 0.50:
		return "low"
	case score < 0.70:
		return "medium"
	case score < 0.85:
		return "high"
	default:
		return "critical"
	}
}

// clampedDiff returns how different `actual` is from `expected`, normalised by `scale`, clamped to [0,1].
func clampedDiff(actual, expected, scale float64) float64 {
	if scale == 0 {
		if actual == expected {
			return 0
		}
		return 1
	}
	diff := math.Abs(actual-expected) / scale
	if diff > 1 {
		return 1
	}
	return diff
}

func tokenize(text string) []string {
	var words []string
	for _, w := range strings.Fields(text) {
		w = strings.Trim(w, ".,!?;:\"'()[]{}—-")
		if len(w) > 0 {
			words = append(words, w)
		}
	}
	return words
}

func splitSentences(text string) []string {
	var sentences []string
	current := strings.Builder{}
	for _, ch := range text {
		current.WriteRune(ch)
		if ch == '.' || ch == '!' || ch == '?' {
			s := strings.TrimSpace(current.String())
			if s != "" {
				sentences = append(sentences, s)
			}
			current.Reset()
		}
	}
	if s := strings.TrimSpace(current.String()); s != "" {
		sentences = append(sentences, s)
	}
	return sentences
}

func firstWords(s string, n int) string {
	words := strings.Fields(s)
	if len(words) <= n {
		return strings.ToLower(s)
	}
	return strings.ToLower(strings.Join(words[:n], " "))
}

func topN(freq map[string]int, n int) []string {
	type kv struct {
		k string
		v int
	}
	var sorted []kv
	for k, v := range freq {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].v > sorted[j].v
	})
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "i": true, "you": true,
		"it": true, "is": true, "was": true, "be": true, "have": true,
		"that": true, "this": true, "my": true, "we": true, "are": true,
	}
	result := make([]string, 0, n)
	for _, kv := range sorted {
		if !stopWords[kv.k] {
			result = append(result, kv.k)
		}
		if len(result) >= n {
			break
		}
	}
	return result
}

func topKey(freq map[string]int) string {
	best := ""
	bestCount := 0
	for k, v := range freq {
		if v > bestCount {
			best = k
			bestCount = v
		}
	}
	return best
}

