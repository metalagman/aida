//go:build integration && opencode

package runtimeexec_test

import (
	"testing"

	"github.com/normahq/norma/pkg/runtime/agentconfig"
)

func TestACPIntegration_OpenCode_CountGoFiles(t *testing.T) {
	t.Helper()
	requireACPIntegrationOptIn(t)
	requireBinary(t, "opencode")

	command := generateACPCommand(t, "opencode", agentconfig.Config{
		Type: agentconfig.AgentTypeOpenCodeACP,
		OpenCodeACP: &agentconfig.ACPConfig{
			Model: "opencode/big-pickle",
		},
	})

	assertCountGoFilesCommand(t, command)
}
