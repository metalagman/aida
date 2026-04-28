package runtimeexec

import (
	"fmt"
	"strings"
)

var rejectedOutputPrefixes = []string{
	"command:",
	"cmd:",
	"run ",
	"i would run",
	"here is",
	"here's",
	"you can run",
}

// NormalizeCommand extracts a single executable shell command from provider
// output and rejects prose or multi-line responses.
func NormalizeCommand(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("provider agent returned empty output")
	}

	if strings.HasPrefix(trimmed, "```") {
		trimmed = extractFromCodeFence(trimmed)
	}

	if trimmed == "UNABLE_TO_RUN_LOCAL" {
		return trimmed, nil
	}

	lines := nonEmptyLines(trimmed)
	if len(lines) != 1 {
		return "", fmt.Errorf("provider agent returned non-command output")
	}

	command := strings.TrimSpace(lines[0])
	lower := strings.ToLower(command)

	for _, prefix := range rejectedOutputPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return "", fmt.Errorf("provider agent returned non-command output")
		}
	}

	if strings.HasPrefix(command, "- ") || strings.HasPrefix(command, "* ") {
		return "", fmt.Errorf("provider agent returned non-command output")
	}

	return command, nil
}

func extractFromCodeFence(trimmed string) string {
	idx := strings.Index(trimmed, "\n")
	if idx == -1 {
		return strings.TrimSpace(strings.Trim(trimmed, "`"))
	}

	body := trimmed[idx+1:]
	lastFenceIdx := strings.LastIndex(body, "```")

	if lastFenceIdx != -1 {
		body = body[:lastFenceIdx]
	}

	return strings.TrimSpace(body)
}

func nonEmptyLines(input string) []string {
	rawLines := strings.Split(input, "\n")
	lines := make([]string, 0, len(rawLines))

	for _, line := range rawLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		lines = append(lines, line)
	}

	return lines
}

func snippet(input string) string {
	const maxLen = 80

	text := strings.Join(nonEmptyLines(input), " ")
	if len(text) <= maxLen {
		return text
	}

	return text[:maxLen] + "..."
}
