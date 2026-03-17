package accounts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

const (
	// Microsoft Graph API scopes
	graphMailReadScope   = "https://graph.microsoft.com/Mail.Read"
	graphMailSendScope   = "https://graph.microsoft.com/Mail.Send"
	graphMailReadWriteScope = "https://graph.microsoft.com/Mail.ReadWrite"
)

// OutlookClient handles Microsoft Graph API interactions using OAuth2
type OutlookClient struct {
	account *Account
	config  *oauth2.Config
	client  *http.Client
}

// NewOutlookClient creates a new Microsoft Graph API client
func NewOutlookClient(account *Account, onTokenRefresh func(*oauth2.Token)) (*OutlookClient, error) {
	config := &oauth2.Config{
		ClientID:     account.OAuthClientID,
		ClientSecret: account.OAuthClientSecret,
		Endpoint:     microsoft.AzureADEndpoint("common"),
		Scopes: []string{
			graphMailReadScope,
			graphMailSendScope,
			graphMailReadWriteScope,
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

	return &OutlookClient{
		account: account,
		config:  config,
		client:  client,
	}, nil
}

// GetAuthURL returns the OAuth2 authorization URL for user consent
func (o *OutlookClient) GetAuthURL() string {
	return o.config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}

// ExchangeCode exchanges an authorization code for tokens
func (o *OutlookClient) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := o.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Update account with new tokens
	o.account.OAuthAccessToken = token.AccessToken
	o.account.OAuthRefreshToken = token.RefreshToken
	o.account.OAuthExpiry = token.Expiry

	return token, nil
}

// GraphMessage represents a Microsoft Graph email message
type GraphMessage struct {
	ID                   string              `json:"id"`
	CreatedDateTime      string              `json:"createdDateTime"`
	ReceivedDateTime     string              `json:"receivedDateTime"`
	Subject              string              `json:"subject"`
	BodyPreview          string              `json:"bodyPreview"`
	Body                 *GraphBody          `json:"body"`
	From                 *GraphRecipient     `json:"from"`
	ToRecipients         []*GraphRecipient   `json:"toRecipients"`
	CcRecipients         []*GraphRecipient   `json:"ccRecipients"`
	BccRecipients        []*GraphRecipient   `json:"bccRecipients"`
	HasAttachments       bool                `json:"hasAttachments"`
	Attachments          []*GraphAttachment  `json:"attachments,omitempty"`
	IsRead               bool                `json:"isRead"`
	IsDraft              bool                `json:"isDraft"`
	InternetMessageID    string              `json:"internetMessageId"`
}

// GraphBody represents the message body
type GraphBody struct {
	ContentType string `json:"contentType"` // "text" or "html"
	Content     string `json:"content"`
}

// GraphRecipient represents an email recipient
type GraphRecipient struct {
	EmailAddress *GraphEmailAddress `json:"emailAddress"`
}

// GraphEmailAddress represents an email address with name
type GraphEmailAddress struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// GraphAttachment represents an attachment
type GraphAttachment struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	ContentType      string `json:"contentType"`
	Size             int64  `json:"size"`
	IsInline         bool   `json:"isInline"`
	ContentBytes     string `json:"contentBytes,omitempty"` // Base64
}

// GraphListMessagesResponse represents Graph API list response
type GraphListMessagesResponse struct {
	Context      string          `json:"@odata.context"`
	NextLink     string          `json:"@odata.nextLink,omitempty"`
	Value        []*GraphMessage `json:"value"`
}

// FetchMessages retrieves messages from Outlook/Microsoft 365
func (o *OutlookClient) FetchMessages(ctx context.Context, maxResults int) ([]*Message, error) {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/messages?$top=%d&$orderby=receivedDateTime DESC", maxResults)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Graph API returned status %d", resp.StatusCode)
	}

	var listResp GraphListMessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	messages := make([]*Message, 0, len(listResp.Value))
	for _, graphMsg := range listResp.Value {
		msg, err := o.convertGraphMessage(graphMsg)
		if err != nil {
			// Log error but continue with other messages
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// GetMessage retrieves a specific message by ID with attachments
func (o *OutlookClient) GetMessage(ctx context.Context, messageID string) (*Message, error) {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/me/messages/%s?$expand=attachments", messageID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Graph API returned status %d", resp.StatusCode)
	}

	var graphMsg GraphMessage
	if err := json.NewDecoder(resp.Body).Decode(&graphMsg); err != nil {
		return nil, fmt.Errorf("failed to decode message: %w", err)
	}

	return o.convertGraphMessage(&graphMsg)
}

// convertGraphMessage converts a Graph API message to our unified Message type
func (o *OutlookClient) convertGraphMessage(graphMsg *GraphMessage) (*Message, error) {
	msg := &Message{
		AccountID:  o.account.ID,
		RemoteID:   graphMsg.ID,
		Protocol:   "outlook",
		Subject:    graphMsg.Subject,
		Recipients: []string{},
		Flags:      []string{},
	}

	// Set sender
	if graphMsg.From != nil && graphMsg.From.EmailAddress != nil {
		msg.Sender = fmt.Sprintf("%s <%s>", graphMsg.From.EmailAddress.Name, graphMsg.From.EmailAddress.Address)
	}

	// Set recipients
	for _, to := range graphMsg.ToRecipients {
		if to.EmailAddress != nil {
			msg.Recipients = append(msg.Recipients, to.EmailAddress.Address)
		}
	}

	// Set body
	if graphMsg.Body != nil {
		if graphMsg.Body.ContentType == "html" {
			msg.BodyHTML = graphMsg.Body.Content
		} else {
			msg.BodyPlain = graphMsg.Body.Content
		}
	}

	// Parse timestamp
	if graphMsg.ReceivedDateTime != "" {
		timestamp, err := time.Parse(time.RFC3339, graphMsg.ReceivedDateTime)
		if err == nil {
			msg.ReceivedAt = timestamp
		}
	}

	// Set flags
	if graphMsg.IsRead {
		msg.Flags = append(msg.Flags, "\\Seen")
	}
	if graphMsg.IsDraft {
		msg.Flags = append(msg.Flags, "\\Draft")
	}

	// Extract attachments
	if graphMsg.HasAttachments && len(graphMsg.Attachments) > 0 {
		msg.Attachments = make([]Attachment, 0, len(graphMsg.Attachments))
		for _, att := range graphMsg.Attachments {
			msg.Attachments = append(msg.Attachments, Attachment{
				Filename:    att.Name,
				ContentType: att.ContentType,
				Size:        att.Size,
			})
		}
	}

	return msg, nil
}

// SendMessage sends a message via Microsoft Graph API
func (o *OutlookClient) SendMessage(ctx context.Context, to, subject, body string) error {
	message := map[string]interface{}{
		"subject": subject,
		"body": map[string]string{
			"contentType": "Text",
			"content":     body,
		},
		"toRecipients": []map[string]interface{}{
			{
				"emailAddress": map[string]string{
					"address": to,
				},
			},
		},
	}

	payload := map[string]interface{}{
		"message": message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	url := "https://graph.microsoft.com/v1.0/me/sendMail"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Graph API returned status %d", resp.StatusCode)
	}

	return nil
}
