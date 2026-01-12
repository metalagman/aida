package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/metalagman/aida/pkg/config"
	"github.com/metalagman/aida/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIProvider_GenerateCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": "ls -la",
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm.SetOpenAIBaseURL(server.URL)
	defer llm.SetOpenAIBaseURL("https://api.openai.com/v1")

	p, err := llm.NewOpenAIProvider("test-key", "gpt-4o")
	require.NoError(t, err)

	cmd, err := p.GenerateCommand(context.Background(), "list files")
	require.NoError(t, err)
	assert.Equal(t, "ls -la", cmd)
}

func TestNewOpenAIProvider(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		p, err := llm.NewOpenAIProvider("key", "model")
		require.NoError(t, err)
		assert.NotNil(t, p)
		assert.Equal(t, "openai", p.Name())
	})

	t.Run("missing api key", func(t *testing.T) {
		_, err := llm.NewOpenAIProvider("", "model")
		assert.Error(t, err)
	})

	t.Run("missing model", func(t *testing.T) {
		_, err := llm.NewOpenAIProvider("key", "")
		assert.Error(t, err)
	})
}

func TestListOpenAIModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "gpt-4o"},
				{"id": "gpt-4o-mini"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm.SetOpenAIBaseURL(server.URL)
	defer llm.SetOpenAIBaseURL("https://api.openai.com/v1")

	models, err := llm.ListModels(context.Background(), "openai", config.ProviderConfig{APIKey: "test-key"})
	require.NoError(t, err)
	assert.Len(t, models, 2)
	assert.Equal(t, "gpt-4o", models[0].Name)
	assert.Equal(t, "gpt-4o-mini", models[1].Name)
}
