package config_test

import (
	"testing"

	"github.com/metalagman/aida/pkg/config"
)

func TestNormalizeProviderName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "openai", want: "openai"},
		{input: "OpenAI", want: "openai"},
		{input: "open-ai", want: "openai"},
		{input: "google-ai-studio", want: "aistudio"},
	}

	for _, test := range tests {
		if got := config.NormalizeProviderName(test.input); got != test.want {
			t.Errorf("NormalizeProviderName(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestDefaultModelForProvider(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "openai", want: "gpt-4o-mini"},
		{input: "aistudio", want: "gemini-3-flash"},
		{input: "unknown", want: ""},
	}

	for _, test := range tests {
		if got := config.DefaultModelForProvider(test.input); got != test.want {
			t.Errorf("DefaultModelForProvider(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}
