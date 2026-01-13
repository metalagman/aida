package llm

import (
	"context"
	"fmt"

	"github.com/metalagman/aida/internal/config"
	"github.com/metalagman/aida/internal/llm/provider"
	"github.com/metalagman/aida/internal/llm/providers/aistudio"
	"github.com/metalagman/aida/internal/llm/providers/openai"
)

type Provider = provider.Provider

// NewProvider constructs a provider based on config.
func NewProvider(ctx context.Context, cfg *config.Config) (Provider, error) {
	name, active, err := cfg.ActiveProvider()
	if err != nil {
		return nil, err
	}

	switch name {
	case "aistudio":
		return aistudio.NewProvider(ctx, active.APIKey, active.Model)
	case "openai":
		return openai.NewProvider(active.APIKey, active.Model)
	default:
		return nil, fmt.Errorf("unsupported llm provider %q", name)
	}
}
