package aistudio

import (
	"context"
	"fmt"
	"strings"

	"github.com/metalagman/aida/pkg/llm/command"
	"google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"
)

type Provider struct {
	opts  Options
	model model.LLM
}

func NewProvider(ctx context.Context, apiKey string, modelName string) (*Provider, error) {
	opts := NewOptions(
		modelName,
		WithApiKey(apiKey),
	)
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	var cfg *genai.ClientConfig
	if strings.TrimSpace(opts.apiKey) != "" {
		cfg = &genai.ClientConfig{APIKey: opts.apiKey}
	}

	m, err := gemini.NewModel(ctx, opts.model, cfg)
	if err != nil {
		return nil, fmt.Errorf("create gemini model: %w", err)
	}

	return &Provider{
		opts:  opts,
		model: m,
	}, nil
}

func (p *Provider) GenerateCommand(ctx context.Context, prompt string) (string, error) {
	return command.GenerateCommandWithModel(ctx, p.model, prompt)
}

func (p *Provider) Name() string {
	return "aistudio"
}
