package llm

import (
	"strings"
)

func SanitizeCommand(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}

	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}

	return extractFromCodeFence(trimmed)
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
