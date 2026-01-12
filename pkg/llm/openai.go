package llm

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

const (
	openAITimeout = 30 * time.Second
)

func SetOpenAIBaseURL(url string) {
	openAIBaseURL = url
}

const openAISystemInstruction = "You are a shell command generator. " +
	"Output ONLY the raw shell command, no markdown fences, no explanation. " +
	"If you cannot fulfill the request, output UNABLE_TO_RUN_LOCAL."

type OpenAIProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func NewOpenAIProvider(apiKey string, model string) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("api_key is required for openai provider")
	}

	if model == "" {
		return nil, fmt.Errorf("model is required for openai provider")
	}

	return &OpenAIProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: openAITimeout},
	}, nil
}

func (p *OpenAIProvider) GenerateCommand(ctx context.Context, prompt string) (string, error) {
	payload := openAIChatRequest{
		Model: p.model,
		Messages: []openAIMessage{
			{
				Role:    "system",
				Content: openAISystemInstruction,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal openai request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIBaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create openai request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send openai request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read openai response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("openai request failed: %s", strings.TrimSpace(string(respBody)))
	}

	var parsed openAIChatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("parse openai response: %w", err)
	}

	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("openai response missing content")
	}

	return SanitizeCommand(parsed.Choices[0].Message.Content), nil
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
}

type openAIChatResponse struct {
	Choices []openAIChoice `json:"choices"`
}

type openAIChoice struct {
	Message openAIMessage `json:"message"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
