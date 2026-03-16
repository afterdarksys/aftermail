package ipfs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// Client handles IPFS operations
type Client struct {
	APIEndpoint string
	HTTPClient  *http.Client
}

// NewClient creates a new IPFS client
func NewClient(endpoint string) *Client {
	if endpoint == "" {
		endpoint = "http://127.0.0.1:5001" // Default IPFS API
	}

	return &Client{
		APIEndpoint: endpoint,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AddResponse represents IPFS add response
type AddResponse struct {
	Name string `json:"Name"`
	Hash string `json:"Hash"`
	Size string `json:"Size"`
}

// Add uploads data to IPFS and returns the CID
func (c *Client) Add(ctx context.Context, data []byte) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "message.bin")
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(data); err != nil {
		return "", fmt.Errorf("failed to write data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.APIEndpoint+"/api/v0/add", body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload to IPFS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("IPFS returned status %d", resp.StatusCode)
	}

	var addResp AddResponse
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return addResp.Hash, nil
}

// Get retrieves data from IPFS by CID
func (c *Client) Get(ctx context.Context, cid string) ([]byte, error) {
	url := fmt.Sprintf("%s/api/v0/cat?arg=%s", c.APIEndpoint, cid)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from IPFS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IPFS returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return data, nil
}

// Pin pins a CID to keep it in IPFS
func (c *Client) Pin(ctx context.Context, cid string) error {
	url := fmt.Sprintf("%s/api/v0/pin/add?arg=%s", c.APIEndpoint, cid)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to pin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pin failed with status %d", resp.StatusCode)
	}

	return nil
}

// Unpin unpins a CID
func (c *Client) Unpin(ctx context.Context, cid string) error {
	url := fmt.Sprintf("%s/api/v0/pin/rm?arg=%s", c.APIEndpoint, cid)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to unpin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unpin failed with status %d", resp.StatusCode)
	}

	return nil
}

// CheckHealth checks if IPFS daemon is running
func (c *Client) CheckHealth(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "POST", c.APIEndpoint+"/api/v0/id", nil)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("IPFS daemon not reachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("IPFS daemon unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
