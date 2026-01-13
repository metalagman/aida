package openai

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/metalagman/aida/internal/llm/command"
	"google.golang.org/adk/model"
)

const (
	openAITimeout = 30 * time.Second
)

type Provider struct {
	opts  Options
	model model.LLM
}

func NewProvider(apiKey string, model string) (*Provider, error) {
	opts := NewOptions(apiKey, model)
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	openAIModel, err := NewOpenAIModel(opts.apiKey, opts.model)
	if err != nil {
		return nil, err
	}

	return &Provider{
		opts:  opts,
		model: openAIModel,
	}, nil
}

// NewProviderWithClient creates an OpenAI provider with a custom HTTP client.
func NewProviderWithClient(apiKey string, model string, client *http.Client) (*Provider, error) {
	opts := NewOptions(
		apiKey,
		model,
		WithClient(client),
	)
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	openAIModel, err := NewOpenAIModelWithClient(opts.apiKey, opts.model, opts.client)
	if err != nil {
		return nil, err
	}

	return &Provider{
		opts:  opts,
		model: openAIModel,
	}, nil
}

func (p *Provider) GenerateCommand(ctx context.Context, prompt string) (string, error) {
	return command.GenerateCommandWithModel(ctx, p.model, prompt)
}

func (p *Provider) Name() string {
	return "openai"
}
