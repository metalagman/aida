package cmd

import "github.com/metalagman/aida/internal/config"

func normalizeProvider(input string) string {
	return config.NormalizeProviderName(input)
}
