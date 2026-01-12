package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	openAIBaseURL = "https://api.openai.com/v1"
)

type openAIModelList struct {
	Data []openAIModel `json:"data"`
}

type openAIModel struct {
	ID string `json:"id"`
}

func listOpenAIModels(ctx context.Context, apiKey string) ([]ModelInfo, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("api_key is required for openai provider")
	}

	client := &http.Client{Timeout: openAITimeout}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openAIBaseURL+"/models", nil)
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

	var list openAIModelList
	if err := json.Unmarshal(respBody, &list); err != nil {
		return nil, fmt.Errorf("parse openai response: %w", err)
	}

	models := make([]ModelInfo, 0, len(list.Data))

	for _, model := range list.Data {
		if strings.TrimSpace(model.ID) == "" {
			continue
		}

		models = append(models, ModelInfo{
			Name:             model.ID,
			DisplayName:      model.ID,
			SupportedActions: []string{"generateContent"},
		})
	}

	return models, nil
}
