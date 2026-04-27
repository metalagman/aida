//go:build integration && codex

package runtimeexec_test

import (
	"testing"

	"github.com/normahq/runtime/agentconfig"
)

func TestACPIntegration_Codex_CountGoFiles(t *testing.T) {
	t.Helper()
	requireACPIntegrationOptIn(t)
	requireBinary(t, "codex")
	requireBinary(t, "npx")

	command := generateACPCommand(t, "codex", agentconfig.Config{
		Type: agentconfig.AgentTypeCodexACP,
		CodexACP: &agentconfig.ACPConfig{
			Model: "gpt-5.3-codex",
		},
	})

	assertCountGoFilesCommand(t, command)
}
