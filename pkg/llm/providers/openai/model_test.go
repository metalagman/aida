package openai_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/metalagman/aida/pkg/llm/providers/openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

func TestOpenAIModel_GenerateContent(t *testing.T) {
	server := newOpenAIModelTestServer(t)
	defer server.Close()

	openai.SetOpenAIBaseURL(server.URL)
	defer openai.SetOpenAIBaseURL("https://api.openai.com/v1")

	openAIModel, err := openai.NewOpenAIModel("test-key", "gpt-4o")
	require.NoError(t, err)

	req := newOpenAIModelRequest()

	var got *adkmodel.LLMResponse

	for resp, err := range openAIModel.GenerateContent(context.Background(), req, false) {
		require.NoError(t, err)

		got = resp
	}

	require.NotNil(t, got)
	require.NotNil(t, got.Content)
	require.Len(t, got.Content.Parts, 1)
	assert.Equal(t, "ls -la", got.Content.Parts[0].Text)
}

func newOpenAIModelTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	type requestMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type requestPayload struct {
		Model    string           `json:"model"`
		Messages []requestMessage `json:"messages"`
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload requestPayload
		require.NoError(t, json.Unmarshal(body, &payload))

		assert.Equal(t, "gpt-4o", payload.Model)
		require.Len(t, payload.Messages, 2)
		assert.Equal(t, "system", payload.Messages[0].Role)
		assert.Equal(t, "system prompt", payload.Messages[0].Content)
		assert.Equal(t, "user", payload.Messages[1].Role)
		assert.Equal(t, "list files", payload.Messages[1].Content)

		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": "ls -la",
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func newOpenAIModelRequest() *adkmodel.LLMRequest {
	return &adkmodel.LLMRequest{
		Contents: []*genai.Content{
			{
				Role: "user",
				Parts: []*genai.Part{
					{Text: "list files"},
				},
			},
		},
		Config: &genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{
					{Text: "system prompt"},
				},
			},
		},
	}
}
