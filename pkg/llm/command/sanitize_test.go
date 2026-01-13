package command_test

import (
	"testing"

	"github.com/metalagman/aida/pkg/llm/command"
)

func TestSanitizeCommand(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text",
			input: "ls -la",
			want:  "ls -la",
		},
		{
			name:  "code fence",
			input: "```\nls -la\n```",
			want:  "ls -la",
		},
		{
			name:  "fence with language",
			input: "```sh\nls -la\n```",
			want:  "ls -la",
		},
		{
			name:  "extra whitespace",
			input: "\n  ls -la  \n",
			want:  "ls -la",
		},
		{
			name:  "unable sentinel",
			input: "UNABLE_TO_RUN_LOCAL",
			want:  "UNABLE_TO_RUN_LOCAL",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := command.SanitizeCommand(tc.input)
			if got != tc.want {
				t.Fatalf("SanitizeCommand(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
