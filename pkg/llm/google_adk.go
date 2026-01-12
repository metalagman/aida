package llm

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"
)

type GoogleADKProvider struct {
	model model.LLM
}

func NewGoogleADKProvider(ctx context.Context, apiKey string, modelName string) (*GoogleADKProvider, error) {
	m, err := gemini.NewModel(ctx, modelName, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		return nil, fmt.Errorf("create gemini model: %w", err)
	}

	return &GoogleADKProvider{
		model: m,
	}, nil
}

func (p *GoogleADKProvider) GenerateCommand(ctx context.Context, prompt string) (string, error) {
	req := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role: "user",
				Parts: []*genai.Part{
					{Text: prompt},
				},
			},
		},
		Config: &genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{
					{Text: "You are a shell command generator. " +
						"Output ONLY the raw shell command, no markdown fences, no explanation. " +
						"If you cannot fulfill the request, output UNABLE_TO_RUN_LOCAL."},
				},
			},
		},
	}

	var sb strings.Builder

	for resp, err := range p.model.GenerateContent(ctx, req, false) {
		if err != nil {
			return "", fmt.Errorf("generate content: %w", err)
		}

		if resp.Content == nil {
			continue
		}

		for _, part := range resp.Content.Parts {
			if part.Text != "" {
				sb.WriteString(part.Text)
			}
		}
	}

	return SanitizeCommand(sb.String()), nil
}

func (p *GoogleADKProvider) Name() string {
	return "aistudio"
}
