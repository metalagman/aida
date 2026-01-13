package command

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/metalagman/aida/internal/llm/templater"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

const systemInstructionTemplate = `You are a shell command generator. Output ONLY the raw shell command,
no markdown fences, no explanation. If you cannot fulfill the request,
output UNABLE_TO_RUN_LOCAL.

Environment:
- OS: {{.OS}}
- Arch: {{.Arch}}
- Shell: {{.Shell}}
- CWD: {{.CWD}}`

const defaultGenerateTimeout = 60 * time.Second

func GenerateCommandWithModel(ctx context.Context, llmModel model.LLM, prompt string) (string, error) {
	if llmModel == nil {
		return "", fmt.Errorf("model is required")
	}

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, defaultGenerateTimeout)
		defer cancel()
	}

	systemInstruction, err := templater.Render(systemInstructionTemplate, map[string]string{
		"OS":    runtime.GOOS,
		"Arch":  runtime.GOARCH,
		"Shell": defaultString(os.Getenv("AIDA_SHELL"), os.Getenv("SHELL"), "unknown"),
		"CWD":   defaultString(currentDir(), "unknown"),
	})
	if err != nil {
		return "", err
	}

	req := &model.LLMRequest{
		Model: llmModel.Name(),
		Contents: []*genai.Content{
			{
				Role: "user",
				Parts: []*genai.Part{
					{Text: prompt},
				},
			},
		},
		Config: &genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{
					{Text: systemInstruction},
				},
			},
		},
	}

	var sb strings.Builder

	for resp, err := range llmModel.GenerateContent(ctx, req, false) {
		if err != nil {
			return "", fmt.Errorf("generate content: %w", err)
		}

		if resp == nil || resp.Content == nil {
			continue
		}

		for _, part := range resp.Content.Parts {
			if part.Text != "" {
				sb.WriteString(part.Text)
			}
		}
	}

	return SanitizeCommand(sb.String()), nil
}

func currentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	return dir
}

func defaultString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}
