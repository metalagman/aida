//go:build integration && gemini

package runtimeexec_test

import (
	"testing"

	"github.com/normahq/runtime/agentconfig"
)

func TestACPIntegration_Gemini_CountGoFiles(t *testing.T) {
	t.Helper()
	requireACPIntegrationOptIn(t)
	requireBinary(t, "gemini")

	command := generateACPCommand(t, "gemini", agentconfig.Config{
		Type: agentconfig.AgentTypeGeminiACP,
		GeminiACP: &agentconfig.ACPConfig{
			Model: "gemini-3-flash-preview",
		},
	})

	assertCountGoFilesCommand(t, command)
}
