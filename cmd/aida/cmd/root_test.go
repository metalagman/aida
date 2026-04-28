package cmd_test

import (
	"bytes"
	"strings"
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

func TestRootCmdShowsHelpWithoutPrompt(t *testing.T) {
	t.Run("bare root", func(t *testing.T) {
		var stdout bytes.Buffer

		root := cmd.NewRootCmd()
		root.SetOut(&stdout)
		root.SetErr(&stdout)
		root.SetArgs(nil)

		if err := root.Execute(); err != nil {
			t.Fatalf("Execute() error = %v, want nil", err)
		}

		if !strings.Contains(stdout.String(), "Usage:\n  aida [prompt] [-- prompt] [flags]") {
			t.Fatalf("help output = %q, want root usage", stdout.String())
		}
	})

	t.Run("flags without prompt", func(t *testing.T) {
		var stdout bytes.Buffer

		root := cmd.NewRootCmd()
		root.SetOut(&stdout)
		root.SetErr(&stdout)
		root.SetArgs([]string{"--profile", "api"})

		if err := root.Execute(); err != nil {
			t.Fatalf("Execute() error = %v, want nil", err)
		}

		if !strings.Contains(stdout.String(), "Available Commands:") {
			t.Fatalf("help output = %q, want root help", stdout.String())
		}
	})

	t.Run("bare dash", func(t *testing.T) {
		var stdout bytes.Buffer

		root := cmd.NewRootCmd()
		root.SetOut(&stdout)
		root.SetErr(&stdout)
		root.SetArgs([]string{"--"})

		if err := root.Execute(); err != nil {
			t.Fatalf("Execute() error = %v, want nil", err)
		}

		if !strings.Contains(stdout.String(), "Generate and run a single shell command from a prompt") {
			t.Fatalf("help output = %q, want root help", stdout.String())
		}
	})
}

func TestRootCmdRejectsRemovedFlags(t *testing.T) {
	t.Run("provider", func(t *testing.T) {
		root := cmd.NewRootCmd()
		root.SetArgs([]string{"--provider", "openai", "--", "list"})

		err := root.Execute()
		if err == nil || !strings.Contains(err.Error(), "unknown flag: --provider") {
			t.Fatalf("Execute() error = %v, want unknown provider flag", err)
		}
	})

	t.Run("api-key", func(t *testing.T) {
		root := cmd.NewRootCmd()
		root.SetArgs([]string{"--api-key", "test-key", "--", "list"})

		err := root.Execute()
		if err == nil || !strings.Contains(err.Error(), "unknown flag: --api-key") {
			t.Fatalf("Execute() error = %v, want unknown api-key flag", err)
		}
	})
}
