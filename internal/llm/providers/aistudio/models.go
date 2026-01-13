package aistudio

import (
	"context"
	"fmt"
	"strings"

	"github.com/metalagman/aida/internal/config"
	"github.com/metalagman/aida/internal/llm/provider"
	"google.golang.org/genai"
)

const defaultPageSize = 100

func ListModels(ctx context.Context, cfg config.ProviderConfig) ([]provider.ModelInfo, error) {
	var clientConfig *genai.ClientConfig
	if strings.TrimSpace(cfg.APIKey) != "" {
		clientConfig = &genai.ClientConfig{APIKey: cfg.APIKey}
	}

	client, err := genai.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("create genai client: %w", err)
	}

	return listModels(ctx, client)
}

func listModels(ctx context.Context, client *genai.Client) ([]provider.ModelInfo, error) {
	page, err := client.Models.List(ctx, &genai.ListModelsConfig{PageSize: defaultPageSize})
	if err != nil {
		return nil, err
	}

	var models []provider.ModelInfo

	for {
		for _, model := range page.Items {
			if model == nil {
				continue
			}

			models = append(models, provider.ModelInfo{
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
