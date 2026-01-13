package provider

import "context"

// Provider generates a single shell command from a user prompt.
type Provider interface {
	GenerateCommand(ctx context.Context, prompt string) (string, error)
	Name() string
}

type ModelInfo struct {
	Name             string
	DisplayName      string
	SupportedActions []string
}
