package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const betterPhishBaseURL = "https://betterphish.io/api/v1"

// BetterPhishClient is the client for betterphish.io
type BetterPhishClient struct {
	HTTPClient *http.Client
}

// NewBetterPhishClient creates a new client
func NewBetterPhishClient() *BetterPhishClient {
	return &BetterPhishClient{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// addHeaders adds standard headers to bypass rate limits using AfterDark Ecosystem origin
func (c *BetterPhishClient) addHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://afterdarksys.com")
}

// ValidateEmailRequest is the payload for email submission
type ValidateEmailRequest struct {
	EmailBase64 string `json:"email_base64"`
	Source      string `json:"source"`
	Recipient   string `json:"recipient,omitempty"`
}

// ValidateEmailResponse is the response from email submission
type ValidateEmailResponse struct {
	Success      bool   `json:"success"`
	SubmissionID string `json:"submission_id"`
	Message      string `json:"message"`
	Error        string `json:"error,omitempty"`
}

// AIValidateRequest is the payload for AI validation
type AIValidateRequest struct {
	URL          string `json:"url"`
	EmailContent string `json:"email_content,omitempty"`
	Subject      string `json:"subject,omitempty"`
}

// AIValidateResponse is the response from AI validation
type AIValidateResponse struct {
	IsPhishing     bool    `json:"is_phishing"`
	Confidence     float64 `json:"confidence"`
	Recommendation string  `json:"recommendation"`
	URL            string  `json:"url"`
	Error          string  `json:"error,omitempty"`
}

// SubmitEmail submits a raw base64 encoded email to BetterPhish for analysis
func (c *BetterPhishClient) SubmitEmail(emailBase64, recipient string) (*ValidateEmailResponse, error) {
	payload := ValidateEmailRequest{
		EmailBase64: emailBase64,
		Source:      "adsmail",
		Recipient:   recipient,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", betterPhishBaseURL+"/submit/email", bytes.NewBuffer(data))
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

	var result ValidateEmailResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return &result, fmt.Errorf("betterphish error: %s (status %d)", result.Error, resp.StatusCode)
	}

	return &result, nil
}

// Validate validates a URL and optional email content for phishing
func (c *BetterPhishClient) Validate(url, emailContent, subject string) (*AIValidateResponse, error) {
	payload := AIValidateRequest{
		URL:          url,
		EmailContent: emailContent,
		Subject:      subject,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", betterPhishBaseURL+"/validate", bytes.NewBuffer(data))
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

	var result AIValidateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return &result, fmt.Errorf("betterphish validate error: %s (status %d)", result.Error, resp.StatusCode)
	}

	return &result, nil
}

// ReportPhishing reports a suspicious URL
func (c *BetterPhishClient) ReportPhishing(url, emailContent string) error {
	payload := map[string]interface{}{
		"url":       url,
		"submitter": "adsmail",
		"body":      emailContent,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", betterPhishBaseURL+"/submit", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	c.addHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to report phishing: status %d", resp.StatusCode)
	}

	return nil
}
