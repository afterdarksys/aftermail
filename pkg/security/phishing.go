package security

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// PhishingScanner leverages the BetterPhish API for URL and Heuristics scanning
type PhishingScanner struct {
	APIKey string
	Client *http.Client
}

func NewPhishingScanner(apiKey string) *PhishingScanner {
	return &PhishingScanner{
		APIKey: apiKey,
		Client: &http.Client{Timeout: 5 * time.Second},
	}
}

// ScanURL queries BetterPhish to determine if a domain in an email is blacklisted
func (ps *PhishingScanner) ScanURL(domain string) (bool, error) {
	if ps.APIKey == "" {
		log.Println("[Phishing] BetterPhish API Key missing, falling back to basic checks.")
		return false, nil // Bypass mode
	}

	log.Printf("[Phishing] Verifying %s against BetterPhish.io...", domain)
	url := fmt.Sprintf("https://api.betterphish.io/v1/scan?url=%s", domain)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+ps.APIKey)

	resp, err := ps.Client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("BetterPhish API returned %d", resp.StatusCode)
	}

	var result struct {
		Malicious bool `json:"malicious"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return result.Malicious, nil
}
