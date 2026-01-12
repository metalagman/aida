package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/metalagman/aida/pkg/config"
	"google.golang.org/genai"
)

type ModelInfo struct {
	Name             string
	DisplayName      string
	SupportedActions []string
}

const defaultPageSize = 100

func ListModels(ctx context.Context, provider string, cfg config.ProviderConfig) ([]ModelInfo, error) {
	provider = config.NormalizeProviderName(provider)
	if provider == "" {
		return nil, fmt.Errorf("provider name is required")
	}

	switch provider {
	case "aistudio":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("api_key is required for aistudio provider")
		}

		client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: cfg.APIKey})
		if err != nil {
			return nil, fmt.Errorf("create genai client: %w", err)
		}

		return listGenAIModels(ctx, client)
	default:
		return nil, fmt.Errorf("unsupported provider %q", provider)
	}
}

func listGenAIModels(ctx context.Context, client *genai.Client) ([]ModelInfo, error) {
	page, err := client.Models.List(ctx, &genai.ListModelsConfig{PageSize: defaultPageSize})
	if err != nil {
		return nil, err
	}

	var models []ModelInfo

	for {
		for _, model := range page.Items {
			if model == nil {
				continue
			}

			models = append(models, ModelInfo{
				Name:             model.Name,
				DisplayName:      model.DisplayName,
				SupportedActions: model.SupportedActions,
			})
		}

		if page.NextPageToken == "" {
			break
		}

		page, err = page.Next(ctx)
		if err != nil {
			return nil, err
		}
	}

	return models, nil
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