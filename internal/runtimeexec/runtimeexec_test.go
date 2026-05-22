package runtimeexec_test

import (
	"context"
	"errors"
	"io"
	"iter"
	"strings"
	"testing"

	"github.com/metalagman/aida/internal/config"
	"github.com/metalagman/aida/internal/runtimeexec"
	"github.com/normahq/norma/pkg/runtime/agentconfig"
	runtimeconfig "github.com/normahq/norma/pkg/runtime/appconfig"
	adkagent "google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

func TestContentTextUsesVisibleText(t *testing.T) {
	t.Parallel()

	got := runtimeexec.ContentTextForTest(genai.NewContentFromText("hello", genai.RoleModel))
	if got != "hello" {
		t.Fatalf("ContentTextForTest() = %q, want hello", got)
	}
}

func TestBuildUserPrompt(t *testing.T) {
	t.Parallel()

	got := runtimeexec.BuildUserPromptForTest("OS: linux\nRequest: count go files")

	if !strings.Contains(got, "Return ONLY the raw shell command text") {
		t.Fatalf("BuildUserPromptForTest() missing command contract: %q", got)
	}

	if !strings.Contains(got, "OS: linux\nRequest: count go files") {
		t.Fatalf("BuildUserPromptForTest() missing request context: %q", got)
	}
}

func TestGenerateCommandClosesAgentOnSuccess(t *testing.T) {
	cfg := testRuntimeConfig()
	agent, closed := newClosableTestAgent(t, nil)

	runtimeexec.SetBuildAgentRuntimeFuncForTest(
		func(context.Context, *config.Config, string, string, io.Writer) (adkagent.Agent, error) {
			return agent, nil
		},
	)
	runtimeexec.SetRunAgentTurnFuncForTest(func(context.Context, adkagent.Agent, string) (string, error) {
		return "find . -name '*.go' | wc -l", nil
	})
	t.Cleanup(runtimeexec.ResetBuildAgentRuntimeFuncForTest)
	t.Cleanup(runtimeexec.ResetRunAgentTurnFuncForTest)

	got, err := runtimeexec.GenerateCommand(context.Background(), cfg, "Request: count go files")
	if err != nil {
		t.Fatalf("GenerateCommand() error = %v", err)
	}

	if got != "find . -name '*.go' | wc -l" {
		t.Fatalf("GenerateCommand() = %q, want sanitized command", got)
	}

	if !*closed {
		t.Fatal("GenerateCommand() did not close the built agent")
	}
}

func TestGenerateCommandClosesAgentOnRunError(t *testing.T) {
	cfg := testRuntimeConfig()
	agent, closed := newClosableTestAgent(t, nil)
	wantErr := errors.New("boom")

	runtimeexec.SetBuildAgentRuntimeFuncForTest(
		func(_ context.Context, _ *config.Config, _, _ string, stderr io.Writer) (adkagent.Agent, error) {
			_, _ = io.WriteString(stderr, "run boom")

			return agent, nil
		},
	)
	runtimeexec.SetRunAgentTurnFuncForTest(func(context.Context, adkagent.Agent, string) (string, error) {
		return "", wantErr
	})
	t.Cleanup(runtimeexec.ResetBuildAgentRuntimeFuncForTest)
	t.Cleanup(runtimeexec.ResetRunAgentTurnFuncForTest)

	_, err := runtimeexec.GenerateCommand(context.Background(), cfg, "Request: count go files")
	if !errors.Is(err, wantErr) {
		t.Fatalf("GenerateCommand() error = %v, want %v", err, wantErr)
	}

	if !strings.Contains(err.Error(), "provider stderr: run boom") {
		t.Fatalf("GenerateCommand() error = %q, want provider stderr", err)
	}

	if !*closed {
		t.Fatal("GenerateCommand() did not close the built agent after run error")
	}
}

func TestGenerateCommandFailsOnCloseErrorAfterSuccess(t *testing.T) {
	cfg := testRuntimeConfig()
	agent, closed := newClosableTestAgent(t, errors.New("close failed"))

	runtimeexec.SetBuildAgentRuntimeFuncForTest(
		func(_ context.Context, _ *config.Config, _, _ string, stderr io.Writer) (adkagent.Agent, error) {
			_, _ = io.WriteString(stderr, "close boom")

			return agent, nil
		},
	)
	runtimeexec.SetRunAgentTurnFuncForTest(func(context.Context, adkagent.Agent, string) (string, error) {
		return "find . -name '*.go' | wc -l", nil
	})
	t.Cleanup(runtimeexec.ResetBuildAgentRuntimeFuncForTest)
	t.Cleanup(runtimeexec.ResetRunAgentTurnFuncForTest)

	_, err := runtimeexec.GenerateCommand(context.Background(), cfg, "Request: count go files")
	if err == nil {
		t.Fatal("GenerateCommand() error = nil, want close error")
	}

	if !strings.Contains(err.Error(), "close provider runtime: close failed") {
		t.Fatalf("GenerateCommand() error = %q, want close error", err)
	}

	if !strings.Contains(err.Error(), "provider stderr: close boom") {
		t.Fatalf("GenerateCommand() error = %q, want provider stderr", err)
	}

	if !*closed {
		t.Fatal("GenerateCommand() did not attempt to close the built agent")
	}
}

func TestGenerateCommandIncludesProviderStderrOnBuildError(t *testing.T) {
	cfg := testRuntimeConfig()
	wantErr := errors.New("build boom")

	runtimeexec.SetBuildAgentRuntimeFuncForTest(
		func(_ context.Context, _ *config.Config, _, _ string, stderr io.Writer) (adkagent.Agent, error) {
			_, _ = io.WriteString(stderr, "build stderr")

			return nil, wantErr
		},
	)
	t.Cleanup(runtimeexec.ResetBuildAgentRuntimeFuncForTest)

	_, err := runtimeexec.GenerateCommand(context.Background(), cfg, "Request: count go files")
	if !errors.Is(err, wantErr) {
		t.Fatalf("GenerateCommand() error = %v, want %v", err, wantErr)
	}

	if !strings.Contains(err.Error(), "provider stderr: build stderr") {
		t.Fatalf("GenerateCommand() error = %q, want provider stderr", err)
	}
}

func TestNormalizeCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{
			name:  "plain command",
			input: "find . -name '*.go' | wc -l",
			want:  "find . -name '*.go' | wc -l",
		},
		{
			name:  "fenced command",
			input: "```sh\nfind . -name '*.go' | wc -l\n```",
			want:  "find . -name '*.go' | wc -l",
		},
		{
			name:  "unable to run local",
			input: "UNABLE_TO_RUN_LOCAL",
			want:  "UNABLE_TO_RUN_LOCAL",
		},
		{
			name:    "prose prefix rejected",
			input:   "I would run find . -name '*.go' | wc -l",
			wantErr: "provider agent returned non-command output",
		},
		{
			name:    "label rejected",
			input:   "command: find . -name '*.go' | wc -l",
			wantErr: "provider agent returned non-command output",
		},
		{
			name:    "multiple lines rejected",
			input:   "find . -name '*.go'\nwc -l",
			wantErr: "provider agent returned non-command output",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertNormalizedCommand(t, tc.input, tc.want, tc.wantErr)
		})
	}
}

func assertNormalizedCommand(t *testing.T, input, want, wantErr string) {
	t.Helper()

	got, err := runtimeexec.NormalizeCommandForTest(input)
	if wantErr != "" {
		if err == nil {
			t.Fatalf("NormalizeCommandForTest(%q) error = nil, want %q", input, wantErr)
		}

		if err.Error() != wantErr {
			t.Fatalf("NormalizeCommandForTest(%q) error = %q, want %q", input, err, wantErr)
		}

		return
	}

	if err != nil {
		t.Fatalf("NormalizeCommandForTest(%q) error = %v", input, err)
	}

	if got != want {
		t.Fatalf("NormalizeCommandForTest(%q) = %q, want %q", input, got, want)
	}
}

func TestSeqFromEventsForTest(t *testing.T) {
	t.Parallel()

	events := runtimeexec.SeqFromEventsForTest(
		&session.Event{LLMResponse: model.LLMResponse{Content: genai.NewContentFromText("one", genai.RoleModel)}},
		&session.Event{LLMResponse: model.LLMResponse{Content: genai.NewContentFromText("two", genai.RoleModel)}},
	)

	got := make([]string, 0, 2)

	for ev, err := range events {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got = append(got, runtimeexec.ContentTextForTest(ev.Content))
	}

	if len(got) != 2 || got[1] != "two" {
		t.Fatalf("unexpected event texts: %#v", got)
	}
}

func TestSeqFromEventsForTestCanBeWrappedWithError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	events := func(yield func(*session.Event, error) bool) {
		yield(nil, wantErr)
	}

	for _, err := range events {
		if !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want %v", err, wantErr)
		}

		return
	}

	t.Fatal("expected error event")
}

func testRuntimeConfig() *config.Config {
	return &config.Config{
		Runtime: runtimeconfig.RuntimeConfig{
			Providers: map[string]agentconfig.Config{
				"codex": {
					Type: agentconfig.AgentTypeCodexACP,
					CodexACP: &agentconfig.ACPConfig{
						Model: "gpt-5.3-codex",
					},
				},
			},
		},
		Aida: config.AidaConfig{
			Provider: "codex",
			Mode:     "confirm",
			Shell:    "/bin/sh",
		},
	}
}

type closableAgent struct {
	adkagent.Agent

	closed   *bool
	closeErr error
}

func (a *closableAgent) Close() error {
	*a.closed = true

	return a.closeErr
}

func newClosableTestAgent(t *testing.T, closeErr error) (adkagent.Agent, *bool) {
	t.Helper()

	closed := new(bool)

	base, err := adkagent.New(adkagent.Config{
		Name:        "test-agent",
		Description: "test agent",
		Run: func(adkagent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(func(*session.Event, error) bool) {}
		},
	})
	if err != nil {
		t.Fatalf("agent.New() error = %v", err)
	}

	return &closableAgent{
		Agent:    base,
		closed:   closed,
		closeErr: closeErr,
	}, closed
}
