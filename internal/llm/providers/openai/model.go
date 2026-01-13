package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strings"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// Model adapts OpenAI's chat completions to the ADK model.LLM interface.
type Model struct {
	name   string
	apiKey string
	client *http.Client
}

// NewOpenAIModel creates a model.LLM adapter backed by OpenAI chat completions.
func NewOpenAIModel(apiKey string, modelName string) (*Model, error) {
	return NewOpenAIModelWithClient(apiKey, modelName, nil)
}

// NewOpenAIModelWithClient creates a model.LLM adapter with a custom HTTP client.
func NewOpenAIModelWithClient(apiKey string, modelName string, client *http.Client) (*Model, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("api_key is required for openai provider")
	}

	if strings.TrimSpace(modelName) == "" {
		return nil, fmt.Errorf("model is required for openai provider")
	}

	if client == nil {
		client = &http.Client{Timeout: openAITimeout}
	}

	return &Model{
		name:   modelName,
		apiKey: apiKey,
		client: client,
	}, nil
}

func (m *Model) Name() string {
	return m.name
}

func (m *Model) GenerateContent(
	ctx context.Context,
	req *model.LLMRequest,
	stream bool,
) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		resp, err := m.generate(ctx, req)
		yield(resp, err)
	}
}

//nolint:cyclop,funlen
func (m *Model) generate(ctx context.Context, req *model.LLMRequest) (*model.LLMResponse, error) {
	payload, err := buildChatRequest(req, m.name)
	if err != nil {
		return nil, err
	}

	respBody, err := m.doChatRequest(ctx, payload)
	if err != nil {
		return nil, err
	}

	content, err := parseChatResponse(respBody)
	if err != nil {
		return nil, err
	}

	return &model.LLMResponse{
		Content:      content,
		TurnComplete: true,
	}, nil
}

func buildChatRequest(req *model.LLMRequest, modelName string) (openAIChatRequest, error) {
	messages := openAIMessagesFromRequest(req)
	if len(messages) == 0 {
		return openAIChatRequest{}, fmt.Errorf("openai request missing content")
	}

	payload := openAIChatRequest{
		Model:    modelName,
		Messages: messages,
	}

	applyRequestConfig(&payload, req)

	return payload, nil
}

func applyRequestConfig(payload *openAIChatRequest, req *model.LLMRequest) {
	if req == nil || req.Config == nil {
		return
	}

	if req.Config.Temperature != nil {
		payload.Temperature = float64(*req.Config.Temperature)
	}

	if req.Config.TopP != nil {
		payload.TopP = float64(*req.Config.TopP)
	}

	if req.Config.MaxOutputTokens > 0 {
		payload.MaxTokens = req.Config.MaxOutputTokens
	}

	if len(req.Config.StopSequences) > 0 {
		payload.Stop = req.Config.StopSequences
	}
}

func (m *Model) doChatRequest(ctx context.Context, payload openAIChatRequest) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal openai request: %w", err)
	}

	endpoint := getOpenAIBaseURL() + "/chat/completions"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create openai request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+m.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send openai request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read openai response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("openai request failed: %s", strings.TrimSpace(string(respBody)))
	}

	return respBody, nil
}

func parseChatResponse(respBody []byte) (*genai.Content, error) {
	var parsed openAIChatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("parse openai response: %w", err)
	}

	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return nil, fmt.Errorf("openai response missing content")
	}

	return &genai.Content{
		Role: "model",
		Parts: []*genai.Part{
			{Text: parsed.Choices[0].Message.Content},
		},
	}, nil
}

func openAIMessagesFromRequest(req *model.LLMRequest) []openAIMessage {
	if req == nil {
		return nil
	}

	var messages []openAIMessage

	if req.Config != nil && req.Config.SystemInstruction != nil {
		text := contentText(req.Config.SystemInstruction)
		if strings.TrimSpace(text) != "" {
			messages = append(messages, openAIMessage{
				Role:    "system",
				Content: text,
			})
		}
	}

	for _, content := range req.Contents {
		if content == nil {
			continue
		}

		text := contentText(content)
		if strings.TrimSpace(text) == "" {
			continue
		}

		messages = append(messages, openAIMessage{
			Role:    openAIRole(content.Role),
			Content: text,
		})
	}

	return messages
}

func contentText(content *genai.Content) string {
	if content == nil {
		return ""
	}

	var sb strings.Builder

	for _, part := range content.Parts {
		if part == nil || part.Text == "" {
			continue
		}

		sb.WriteString(part.Text)
	}

	return sb.String()
}

func openAIRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "assistant", "model":
		return "assistant"
	case "system":
		return "system"
	default:
		return "user"
	}
}

type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	MaxTokens   int32           `json:"max_tokens,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
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
