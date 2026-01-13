package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/metalagman/aida/pkg/config"
	"github.com/metalagman/aida/pkg/llm/provider"
	"github.com/metalagman/aida/pkg/llm/providers/aistudio"
	"github.com/metalagman/aida/pkg/llm/providers/openai"
)

type ModelInfo = provider.ModelInfo

func ListModels(ctx context.Context, provider string, cfg config.ProviderConfig) ([]ModelInfo, error) {
	provider = config.NormalizeProviderName(provider)
	if provider == "" {
		return nil, fmt.Errorf("provider name is required")
	}

	switch provider {
	case "aistudio":
		return aistudio.ListModels(ctx, cfg)
	case "openai":
		return openai.ListModels(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported provider %q", provider)
	}
}

func FilterModelsForGenerateContent(models []ModelInfo) []ModelInfo {
	var filtered []ModelInfo

	for _, model := range models {
		if supportsGenerateContent(model.SupportedActions) {
			filtered = append(filtered, model)
		}
	}

	return filtered
}

func supportsGenerateContent(actions []string) bool {
	for _, action := range actions {
		if strings.Contains(strings.ToLower(action), "generatecontent") {
			return true
		}
	}

	return false
}

func DisplayModelName(name string) string {
	return strings.TrimPrefix(name, "models/")
}
