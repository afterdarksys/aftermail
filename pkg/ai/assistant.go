package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Provider represents an AI provider
type Provider string

const (
	ProviderAnthropic  Provider = "anthropic"
	ProviderOpenRouter Provider = "openrouter"
)

// Assistant handles AI operations for email composition
type Assistant struct {
	Provider Provider
	APIKey   string
	Model    string
}

// NewAssistant creates a new AI assistant
func NewAssistant(provider Provider, apiKey, model string) *Assistant {
	if model == "" {
		if provider == ProviderAnthropic {
			model = "claude-sonnet-4-20250514"
		} else {
			model = "anthropic/claude-sonnet-4"
		}
	}

	return &Assistant{
		Provider: provider,
		APIKey:   apiKey,
		Model:    model,
	}
}

// AnthropicRequest represents Anthropic API request
type AnthropicRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

// Message represents a message in the conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicResponse represents Anthropic API response
type AnthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// CheckSpelling checks spelling in the given text
func (a *Assistant) CheckSpelling(ctx context.Context, text string) (string, error) {
	prompt := fmt.Sprintf(`Please check the following text for spelling errors and return ONLY the corrected text, nothing else:

%s`, text)

	return a.query(ctx, prompt)
}

// CheckGrammar checks grammar in the given text
func (a *Assistant) CheckGrammar(ctx context.Context, text string) (string, error) {
	prompt := fmt.Sprintf(`Please check the following text for grammar errors and return ONLY the corrected text, nothing else:

%s`, text)

	return a.query(ctx, prompt)
}

// ImproveWriting improves the writing quality
func (a *Assistant) ImproveWriting(ctx context.Context, text string) (string, error) {
	prompt := fmt.Sprintf(`Please improve the following email text for clarity, professionalism, and impact. Return ONLY the improved text:

%s`, text)

	return a.query(ctx, prompt)
}

// MakeConcise makes the text more concise
func (a *Assistant) MakeConcise(ctx context.Context, text string) (string, error) {
	prompt := fmt.Sprintf(`Please make the following text more concise while preserving all important information. Return ONLY the concise version:

%s`, text)

	return a.query(ctx, prompt)
}

// MakeFormal makes the text more formal
func (a *Assistant) MakeFormal(ctx context.Context, text string) (string, error) {
	prompt := fmt.Sprintf(`Please rewrite the following text in a more formal, professional tone. Return ONLY the formal version:

%s`, text)

	return a.query(ctx, prompt)
}

// MakeFriendly makes the text more friendly
func (a *Assistant) MakeFriendly(ctx context.Context, text string) (string, error) {
	prompt := fmt.Sprintf(`Please rewrite the following text in a more friendly, casual tone. Return ONLY the friendly version:

%s`, text)

	return a.query(ctx, prompt)
}

// GenerateDraft generates an email draft from a brief description
func (a *Assistant) GenerateDraft(ctx context.Context, description string) (string, error) {
	prompt := fmt.Sprintf(`Generate a professional email based on this description:

%s

Return ONLY the email body, no subject line or greetings unless specified in the description.`, description)

	return a.query(ctx, prompt)
}

// SummarizeEmail summarizes an email
func (a *Assistant) SummarizeEmail(ctx context.Context, emailBody string) (string, error) {
	prompt := fmt.Sprintf(`Provide a brief summary of this email in 1-2 sentences:

%s`, emailBody)

	return a.query(ctx, prompt)
}

// query sends a query to the AI provider
func (a *Assistant) query(ctx context.Context, prompt string) (string, error) {
	switch a.Provider {
	case ProviderAnthropic:
		return a.queryAnthropic(ctx, prompt)
	case ProviderOpenRouter:
		return a.queryOpenRouter(ctx, prompt)
	default:
		return "", fmt.Errorf("unsupported provider: %s", a.Provider)
	}
}

// queryAnthropic queries the Anthropic API
func (a *Assistant) queryAnthropic(ctx context.Context, prompt string) (string, error) {
	reqBody := AnthropicRequest{
		Model:     a.Model,
		MaxTokens: 4096,
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp AnthropicResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("API error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return apiResp.Content[0].Text, nil
}

// queryOpenRouter queries the OpenRouter API
func (a *Assistant) queryOpenRouter(ctx context.Context, prompt string) (string, error) {
	reqBody := map[string]interface{}{
		"model": a.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.APIKey)
	req.Header.Set("HTTP-Referer", "https://aftermail.dev")
	req.Header.Set("X-Title", "AfterMail")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("API error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return apiResp.Choices[0].Message.Content, nil
}
