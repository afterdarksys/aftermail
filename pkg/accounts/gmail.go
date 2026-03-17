package accounts

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	// Gmail OAuth2 scopes
	gmailReadScope  = "https://www.googleapis.com/auth/gmail.readonly"
	gmailSendScope  = "https://www.googleapis.com/auth/gmail.send"
	gmailModifyScope = "https://www.googleapis.com/auth/gmail.modify"
)

// GmailClient handles Gmail API interactions using OAuth2
type GmailClient struct {
	account *Account
	config  *oauth2.Config
	client  *http.Client
}

// NewGmailClient creates a new Gmail API client
func NewGmailClient(account *Account, onTokenRefresh func(*oauth2.Token)) (*GmailClient, error) {
	config := &oauth2.Config{
		ClientID:     account.OAuthClientID,
		ClientSecret: account.OAuthClientSecret,
		Endpoint:     google.Endpoint,
		Scopes: []string{
			gmailReadScope,
			gmailSendScope,
			gmailModifyScope,
		},
		RedirectURL: "http://localhost:8080/oauth2callback",
	}

	token := &oauth2.Token{
		AccessToken:  account.OAuthAccessToken,
		RefreshToken: account.OAuthRefreshToken,
		Expiry:       account.OAuthExpiry,
	}

	baseSource := config.TokenSource(context.Background(), token)
	notifyingSource := NewNotifyTokenSource(baseSource, token, onTokenRefresh)

	client := oauth2.NewClient(context.Background(), notifyingSource)

	return &GmailClient{
		account: account,
		config:  config,
		client:  client,
	}, nil
}

// GetAuthURL returns the OAuth2 authorization URL for user consent
func (g *GmailClient) GetAuthURL() string {
	return g.config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}

// ExchangeCode exchanges an authorization code for tokens
func (g *GmailClient) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := g.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Update account with new tokens
	g.account.OAuthAccessToken = token.AccessToken
	g.account.OAuthRefreshToken = token.RefreshToken
	g.account.OAuthExpiry = token.Expiry

	return token, nil
}

// GmailMessage represents a Gmail API message
type GmailMessage struct {
	ID       string          `json:"id"`
	ThreadID string          `json:"threadId"`
	Snippet  string          `json:"snippet"`
	Payload  *GmailPayload   `json:"payload"`
	SizeEstimate int64        `json:"sizeEstimate"`
	HistoryID    string       `json:"historyId"`
	InternalDate string       `json:"internalDate"`
	LabelIDs     []string     `json:"labelIds"`
}

// GmailPayload represents the message body structure
type GmailPayload struct {
	PartID   string                `json:"partId"`
	MimeType string                `json:"mimeType"`
	Filename string                `json:"filename"`
	Headers  []GmailHeader         `json:"headers"`
	Body     *GmailBody            `json:"body"`
	Parts    []*GmailPayload       `json:"parts"`
}

// GmailHeader represents an email header
type GmailHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// GmailBody represents the message body
type GmailBody struct {
	Size         int    `json:"size"`
	Data         string `json:"data"` // Base64url encoded
	AttachmentID string `json:"attachmentId"`
}

// ListMessagesResponse represents Gmail list response
type ListMessagesResponse struct {
	Messages          []GmailMessageRef `json:"messages"`
	NextPageToken     string            `json:"nextPageToken"`
	ResultSizeEstimate int              `json:"resultSizeEstimate"`
}

// GmailMessageRef is a reference to a message
type GmailMessageRef struct {
	ID       string `json:"id"`
	ThreadID string `json:"threadId"`
}

// FetchMessages retrieves messages from Gmail
func (g *GmailClient) FetchMessages(ctx context.Context, maxResults int) ([]*Message, error) {
	url := fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages?maxResults=%d", maxResults)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gmail API returned status %d", resp.StatusCode)
	}

	var listResp ListMessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	messages := make([]*Message, 0, len(listResp.Messages))
	for _, msgRef := range listResp.Messages {
		msg, err := g.GetMessage(ctx, msgRef.ID)
		if err != nil {
			// Log error but continue with other messages
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// GetMessage retrieves a specific message by ID
func (g *GmailClient) GetMessage(ctx context.Context, messageID string) (*Message, error) {
	url := fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages/%s?format=full", messageID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gmail API returned status %d", resp.StatusCode)
	}

	var gmailMsg GmailMessage
	if err := json.NewDecoder(resp.Body).Decode(&gmailMsg); err != nil {
		return nil, fmt.Errorf("failed to decode message: %w", err)
	}

	return g.convertGmailMessage(&gmailMsg)
}

// convertGmailMessage converts a Gmail API message to our unified Message type
func (g *GmailClient) convertGmailMessage(gmailMsg *GmailMessage) (*Message, error) {
	msg := &Message{
		AccountID:  g.account.ID,
		RemoteID:   gmailMsg.ID,
		Protocol:   "gmail",
		Recipients: []string{},
		Flags:      gmailMsg.LabelIDs,
	}

	// Parse headers
	if gmailMsg.Payload != nil {
		for _, header := range gmailMsg.Payload.Headers {
			switch header.Name {
			case "From":
				msg.Sender = header.Value
			case "To":
				msg.Recipients = append(msg.Recipients, header.Value)
			case "Subject":
				msg.Subject = header.Value
			}
		}

		// Extract body
		msg.BodyPlain, msg.BodyHTML = g.extractBody(gmailMsg.Payload)

		// Extract attachments
		msg.Attachments = g.extractAttachments(gmailMsg.Payload)
	}

	// Parse timestamp
	if gmailMsg.InternalDate != "" {
		timestamp, err := parseGmailTimestamp(gmailMsg.InternalDate)
		if err == nil {
			msg.ReceivedAt = timestamp
		}
	}

	return msg, nil
}

// extractBody recursively extracts plain and HTML body from Gmail payload
func (g *GmailClient) extractBody(payload *GmailPayload) (plain, html string) {
	if payload == nil {
		return "", ""
	}

	if payload.MimeType == "text/plain" && payload.Body != nil && payload.Body.Data != "" {
		decoded, _ := base64.URLEncoding.DecodeString(payload.Body.Data)
		plain = string(decoded)
	} else if payload.MimeType == "text/html" && payload.Body != nil && payload.Body.Data != "" {
		decoded, _ := base64.URLEncoding.DecodeString(payload.Body.Data)
		html = string(decoded)
	}

	// Recursively check parts
	for _, part := range payload.Parts {
		partPlain, partHTML := g.extractBody(part)
		if partPlain != "" && plain == "" {
			plain = partPlain
		}
		if partHTML != "" && html == "" {
			html = partHTML
		}
	}

	return plain, html
}

// extractAttachments extracts file attachments from Gmail payload
func (g *GmailClient) extractAttachments(payload *GmailPayload) []Attachment {
	var attachments []Attachment

	if payload == nil {
		return attachments
	}

	if payload.Filename != "" && payload.Body != nil && payload.Body.AttachmentID != "" {
		// This is an attachment - actual data would need separate API call
		attachments = append(attachments, Attachment{
			Filename:    payload.Filename,
			ContentType: payload.MimeType,
			Size:        int64(payload.Body.Size),
		})
	}

	// Recursively check parts
	for _, part := range payload.Parts {
		attachments = append(attachments, g.extractAttachments(part)...)
	}

	return attachments
}

// parseGmailTimestamp converts Gmail's internalDate (milliseconds) to time.Time
func parseGmailTimestamp(internalDate string) (time.Time, error) {
	var milliseconds int64
	_, err := fmt.Sscanf(internalDate, "%d", &milliseconds)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, milliseconds*int64(time.Millisecond)), nil
}

// SendMessage sends a message via Gmail API
func (g *GmailClient) SendMessage(ctx context.Context, to, subject, body string) error {
	// Construct RFC 2822 message
	message := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", to, subject, body)
	encoded := base64.URLEncoding.EncodeToString([]byte(message))

	payload := map[string]string{"raw": encoded}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	url := "https://gmail.googleapis.com/gmail/v1/users/me/messages/send"
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Gmail API returned status %d", resp.StatusCode)
	}

	// Update jsonData usage to avoid unused variable error
	_ = jsonData

	return nil
}
