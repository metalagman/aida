package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/metalagman/aida/internal/config"
	"github.com/metalagman/aida/internal/llm/provider"
)

type openAIModelList struct {
	Data []openAIModel `json:"data"`
}

type openAIModel struct {
	ID string `json:"id"`
}

func ListModels(ctx context.Context, cfg config.ProviderConfig) ([]provider.ModelInfo, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("api_key is required for openai provider")
	}

	client, err := openAIClient()
	if err != nil {
		return nil, err
	}

	respBody, err := fetchModelList(ctx, client, apiKey)
	if err != nil {
		return nil, err
	}

	return parseModelList(respBody)
}

func openAIClient() (*http.Client, error) {
	factory := getOpenAIHTTPClientFactory()

	client := factory()
	if client == nil {
		return nil, fmt.Errorf("openai http client factory returned nil")
	}

	return client, nil
}

func fetchModelList(ctx context.Context, client *http.Client, apiKey string) ([]byte, error) {
	endpoint := getOpenAIBaseURL() + "/models"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create openai request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send openai request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read openai response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("openai request failed: %s", strings.TrimSpace(string(respBody)))
	}

	return respBody, nil
}

func parseModelList(respBody []byte) ([]provider.ModelInfo, error) {
	var list openAIModelList
	if err := json.Unmarshal(respBody, &list); err != nil {
		return nil, fmt.Errorf("parse openai response: %w", err)
	}

	models := make([]provider.ModelInfo, 0, len(list.Data))

	for _, model := range list.Data {
		if strings.TrimSpace(model.ID) == "" {
			continue
		}

		models = append(models, provider.ModelInfo{
			Name:             model.ID,
			DisplayName:      model.ID,
			SupportedActions: []string{"generateContent"},
		})
	}

	return models, nil
}
