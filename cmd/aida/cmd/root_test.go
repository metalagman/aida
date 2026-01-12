package cmd_test

import (
	"testing"

	"github.com/metalagman/aida/cmd/aida/cmd"
)

func TestPromptFromArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		dashIndex int
		want      string
	}{
		{
			name:      "no dash uses all args",
			args:      []string{"find", "files"},
			dashIndex: -1,
			want:      "find files",
		},
		{
			name:      "dash uses args after dash",
			args:      []string{"--", "find", "files"},
			dashIndex: 1,
			want:      "find files",
		},
		{
			name:      "dash index past args uses all args",
			args:      []string{"find", "files"},
			dashIndex: 5,
			want:      "find files",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := cmd.PromptFromArgs(tc.args, tc.dashIndex)
			if got != tc.want {
				t.Fatalf("PromptFromArgs(%v, %d) = %q, want %q", tc.args, tc.dashIndex, got, tc.want)
			}
		})
	}
}
