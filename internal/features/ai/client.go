package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client defines the OpenAI-compatible chat completion client contract.
type Client interface {
	Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

type openAIClient struct {
	endpoint   string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewClient creates a new OpenAI-compatible HTTP Client.
func NewClient(endpoint, apiKey, model string) Client {
	return &openAIClient{
		endpoint:   strings.TrimRight(endpoint, "/"),
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: 45 * time.Second},
	}
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func (c *openAIClient) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if c.endpoint == "" {
		return "", fmt.Errorf("openai endpoint not configured")
	}

	payload := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}
	raw, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("openai API error (%d): %s", resp.StatusCode, string(body))
	}

	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("openai API returned empty choices")
	}
	return parsed.Choices[0].Message.Content, nil
}
