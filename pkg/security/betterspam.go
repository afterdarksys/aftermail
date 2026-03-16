package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const betterSpamBaseURL = "https://betterspam.com/api/v1"

// BetterSpamClient is the client for betterspam.com
type BetterSpamClient struct {
	HTTPClient *http.Client
}

// NewBetterSpamClient creates a new BetterSpam client
func NewBetterSpamClient() *BetterSpamClient {
	return &BetterSpamClient{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// addHeaders adds standard headers to bypass rate limits using AfterDark Ecosystem origin
func (c *BetterSpamClient) addHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://afterdarksys.com")
}

// CheckMailRequest payload for checking mail
type CheckMailRequest struct {
	RawEmail string `json:"raw_email"`
}

// MailCheckResult represents spam analysis results
type MailCheckResult struct {
	IsSpam      bool    `json:"is_spam"`
	Score       float64 `json:"score"`
	Required    float64 `json:"required"`
	Action      string  `json:"action"`
	SpamAssassin *SpamScore `json:"spamassassin,omitempty"`
	Rspamd       *SpamScore `json:"rspamd,omitempty"`
}

type SpamScore struct {
	Score float64 `json:"score"`
	Spam  bool    `json:"spam"`
}

// LookupEmailResponse represents the email reputation response
type LookupEmailResponse struct {
	Email      string  `json:"email"`
	Reputation string  `json:"reputation"`
	Score      float64 `json:"score"`
	IsBlacklisted bool `json:"is_blacklisted"`
}

// LookupDomainResponse represents the domain reputation response
type LookupDomainResponse struct {
	Domain     string  `json:"domain"`
	Reputation string  `json:"reputation"`
	IsBlacklisted bool `json:"is_blacklisted"`
}

// CheckMail checks if a raw email is spam
func (c *BetterSpamClient) CheckMail(rawEmail string) (*MailCheckResult, error) {
	payload := CheckMailRequest{RawEmail: rawEmail}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", betterSpamBaseURL+"/mail/check", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	c.addHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("betterspam error: status %d. Response: %s", resp.StatusCode, string(body))
	}

	var result MailCheckResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// LookupEmail checks the reputation of an email address
func (c *BetterSpamClient) LookupEmail(email string) (*LookupEmailResponse, error) {
	req, err := http.NewRequest("GET", betterSpamBaseURL+"/lookup/email/"+email, nil)
	if err != nil {
		return nil, err
	}
	c.addHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result LookupEmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("betterspam error: status %d", resp.StatusCode)
	}

	return &result, nil
}

// LookupDomain checks the reputation of a domain
func (c *BetterSpamClient) LookupDomain(domain string) (*LookupDomainResponse, error) {
	req, err := http.NewRequest("GET", betterSpamBaseURL+"/lookup/domain/"+domain, nil)
	if err != nil {
		return nil, err
	}
	c.addHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result LookupDomainResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("betterspam error: status %d", resp.StatusCode)
	}

	return &result, nil
}
