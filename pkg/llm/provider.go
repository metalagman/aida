package llm

import (
	"context"
	"errors"
	"fmt"

	"github.com/metalagman/aida/pkg/config"
)

// Provider generates a single shell command from a user prompt.
type Provider interface {
	GenerateCommand(ctx context.Context, prompt string) (string, error)
	Name() string
}

// NewProvider constructs a provider based on config.
func NewProvider(ctx context.Context, cfg *config.Config) (Provider, error) {
	name, active, err := cfg.ActiveProvider()
	if err != nil {
		return nil, err
	}

	switch name {
	case "aistudio":
		if active.APIKey == "" {
			return nil, errors.New("api_key is required for aistudio provider")
		}

		return NewGoogleADKProvider(ctx, active.APIKey, active.Model)
	default:
		return nil, fmt.Errorf("unsupported llm provider %q", name)
	}
}
