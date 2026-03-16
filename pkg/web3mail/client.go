package web3mail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles interaction with the mailblocks.io backend API
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// StakedEmail represents an email quarantined on IPFS awaiting stake resolution
type StakedEmail struct {
	ID          string    `json:"id"`
	Sender      string    `json:"sender"`
	IPFSCID     string    `json:"ipfs_cid"`
	StakeAmount float64   `json:"stake_amount"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewClient creates a new Web3Mail API client
func NewClient(endpoint string) *Client {
	if endpoint == "" {
		endpoint = "http://localhost:8080"
	}
	return &Client{
		BaseURL: endpoint,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// GetQuarantined fetches the list of staked emails pending review
func (c *Client) GetQuarantined(ctx context.Context, recipientAddr string) ([]StakedEmail, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/quarantine?recipient=%s", c.BaseURL, recipientAddr), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var emails []StakedEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return nil, err
	}

	return emails, nil
}

// ResolveStake calls the smart contract backend to refund or slash a sender
func (c *Client) ResolveStake(ctx context.Context, emailID string, action string) error {
	if action != "refund" && action != "slash" {
		return fmt.Errorf("invalid action: %s", action)
	}

	payload, _ := json.Marshal(map[string]string{
		"email_id": emailID,
		"action":   action,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/resolve", c.BaseURL), bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to resolve stake: %s", string(body))
	}

	return nil
}
