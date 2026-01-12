package cmd

import "github.com/metalagman/aida/pkg/config"

func normalizeProvider(input string) string {
	return config.NormalizeProviderName(input)
}
