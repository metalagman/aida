package runtimeexec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"iter"
	"os"
	"strings"

	"github.com/metalagman/aida/internal/config"
	"github.com/normahq/norma/pkg/runtime/agentfactory"
	"github.com/normahq/norma/pkg/runtime/mcpregistry"
	adkagent "google.golang.org/adk/agent"
	adkrunner "google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

const (
	runnerAppName = "aida"
	runnerUserID  = "aida-user"
)

const shellCommandContract = `You are a shell command generator.
	Return ONLY the raw shell command text that should be executed.
	Return exactly one command line.
Do not execute the request mentally and do not return the result of running the command.
Do not call tools, inspect the filesystem, or query the environment beyond the request context you were given.
Decide on the command from the request text alone.
Do not include markdown fences, explanations, labels, quotes, or extra prose.
If the request cannot be fulfilled locally with shell tools, return UNABLE_TO_RUN_LOCAL.`

type runtimeCloser interface {
	Close() error
}

func GenerateCommand(ctx context.Context, cfg *config.Config, prompt string) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("config is nil")
	}

	providerID, err := cfg.ActiveProviderID()
	if err != nil {
		return "", err
	}

	workingDir, err := currentWorkingDir()
	if err != nil {
		return "", err
	}

	var stderrBuf bytes.Buffer

	agentRuntime, err := buildAgentRuntimeFunc(ctx, cfg, providerID, workingDir, &stderrBuf)
	if err != nil {
		return "", withProviderStderr(err, stderrBuf.String())
	}

	command, runErr := runAgentTurnFunc(ctx, agentRuntime, buildUserPrompt(prompt))
	if runErr != nil {
		_ = closeRuntime(agentRuntime)

		return "", withProviderStderr(runErr, stderrBuf.String())
	}

	if err := closeRuntime(agentRuntime); err != nil {
		return "", withProviderStderr(fmt.Errorf("close provider runtime: %w", err), stderrBuf.String())
	}

	return command, nil
}

func buildAgentRuntime(
	ctx context.Context,
	cfg *config.Config,
	providerID,
	workingDir string,
	stderr io.Writer,
) (adkagent.Agent, error) {
	factory := agentfactory.New(
		cfg.Runtime.Providers,
		mcpregistry.New(cfg.Runtime.MCPServers),
		agentfactory.WithStderrWriter(stderr),
	)

	agentRuntime, err := factory.Build(ctx, agentfactory.BuildRequest{
		AgentID:          providerID,
		Name:             providerID,
		WorkingDirectory: workingDir,
	})
	if err != nil {
		return nil, fmt.Errorf("build provider runtime: %w", err)
	}

	return agentRuntime, nil
}

func runAgentTurn(ctx context.Context, ag adkagent.Agent, prompt string) (string, error) {
	if ag == nil {
		return "", fmt.Errorf("agent is nil")
	}

	sessionService := session.InMemoryService()

	r, err := adkrunner.New(adkrunner.Config{
		AppName:        runnerAppName,
		Agent:          ag,
		SessionService: sessionService,
	})
	if err != nil {
		return "", fmt.Errorf("create adk runner: %w", err)
	}

	created, err := sessionService.Create(ctx, &session.CreateRequest{
		AppName: runnerAppName,
		UserID:  runnerUserID,
	})
	if err != nil {
		return "", fmt.Errorf("create adk session: %w", err)
	}

	initialContent := genai.NewContentFromText(prompt, genai.RoleUser)
	events := r.Run(
		ctx,
		runnerUserID,
		created.Session.ID(),
		initialContent,
		adkagent.RunConfig{},
	)

	lastText, err := collectLastEventText(events)
	if err != nil {
		return "", err
	}

	command, err := NormalizeCommand(lastText)
	if err != nil {
		return "", fmt.Errorf("%w: %q", err, snippet(lastText))
	}

	return command, nil
}

func collectLastEventText(events iter.Seq2[*session.Event, error]) (string, error) {
	lastText := ""

	for ev, runErr := range events {
		if runErr != nil {
			return "", fmt.Errorf("run provider agent: %w", runErr)
		}

		if ev == nil || ev.Content == nil {
			continue
		}

		text := contentText(ev.Content)
		if strings.TrimSpace(text) == "" {
			continue
		}

		lastText = text
	}

	if strings.TrimSpace(lastText) == "" {
		return "", fmt.Errorf("provider agent returned empty output")
	}

	return lastText, nil
}

func contentText(content *genai.Content) string {
	if content == nil {
		return ""
	}

	result := ""

	for _, part := range content.Parts {
		if part == nil {
			continue
		}

		result += part.Text
	}

	return result
}

func buildUserPrompt(prompt string) string {
	trimmedPrompt := strings.TrimSpace(prompt)
	if trimmedPrompt == "" {
		return shellCommandContract
	}

	return shellCommandContract + "\n\n" + trimmedPrompt
}

func closeRuntime(agentRuntime adkagent.Agent) error {
	closer, ok := agentRuntime.(runtimeCloser)
	if !ok {
		return nil
	}

	return closer.Close()
}

func withProviderStderr(err error, stderr string) error {
	if err == nil {
		return nil
	}

	trimmed := strings.TrimSpace(stderr)
	if trimmed == "" {
		return err
	}

	return fmt.Errorf("%w | provider stderr: %s", err, trimmed)
}

func currentWorkingDir() (string, error) {
	dir, err := osGetwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	return dir, nil
}

var osGetwd = func() (string, error) {
	return os.Getwd()
}

var runAgentTurnFunc = runAgentTurn
var buildAgentRuntimeFunc = buildAgentRuntime
