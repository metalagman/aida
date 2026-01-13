package llm_test

import (
	"testing"

	"github.com/metalagman/aida/internal/llm"
)

func TestFilterModelsForGenerateContent(t *testing.T) {
	models := []llm.ModelInfo{
		{Name: "models/alpha", SupportedActions: []string{"generateContent"}},
		{Name: "models/beta", SupportedActions: []string{"countTokens"}},
		{Name: "models/gamma", SupportedActions: []string{"models.generateContent"}},
	}

	filtered := llm.FilterModelsForGenerateContent(models)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 models, got %d", len(filtered))
	}

	if filtered[0].Name != "models/alpha" || filtered[1].Name != "models/gamma" {
		t.Fatalf("unexpected models: %+v", filtered)
	}
}

func TestDisplayModelName(t *testing.T) {
	got := llm.DisplayModelName("models/gemini-2.0-flash")
	if got != "gemini-2.0-flash" {
		t.Fatalf("DisplayModelName mismatch: %q", got)
	}
}
