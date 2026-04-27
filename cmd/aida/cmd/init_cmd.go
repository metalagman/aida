package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/metalagman/aida/internal/config"
	"github.com/normahq/runtime/agentconfig"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type initFlags struct {
	force bool
}

type initDocument struct {
	Runtime  runtimeSection        `yaml:"runtime"`
	Aida     config.AidaConfig     `yaml:"aida"`
	Profiles map[string]profileDoc `yaml:"profiles"`
}

type runtimeSection struct {
	Providers  map[string]agentconfig.Config `yaml:"providers"`
	MCPServers map[string]any                `yaml:"mcp_servers,omitempty"`
}

type profileDoc struct {
	Aida config.AidaConfig `yaml:"aida"`
}

type detectedProvider struct {
	ID       string
	Type     string
	Model    string
	Binaries []string
}

func newInitCmd() *cobra.Command {
	flags := &initFlags{}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate the canonical Aida config file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInit(cmd, flags)
		},
	}

	cmd.Flags().BoolVar(&flags.force, "force", false, "Overwrite an existing config file")

	return cmd
}

func runInit(cmd *cobra.Command, flags *initFlags) error {
	configPath, err := config.ResolveConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(configPath); err == nil && !flags.force {
		return fmt.Errorf("config already exists at %s; rerun with --force to overwrite", configPath)
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("stat config: %w", err)
	}

	if _, err := config.EnsureConfigDir(); err != nil {
		return err
	}

	content, err := buildInitConfig()
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, []byte(content), config.FilePerm); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", configPath)

	return nil
}

func buildInitConfig() (string, error) {
	detected := detectACPProviders()
	doc := initDocument{
		Runtime: runtimeSection{
			Providers:  buildInitProviders(detected),
			MCPServers: map[string]any{},
		},
		Aida: config.AidaConfig{
			Mode:  "confirm",
			Shell: "/bin/sh",
		},
		Profiles: map[string]profileDoc{
			"default": {
				Aida: config.AidaConfig{},
			},
		},
	}

	if len(detected) > 0 {
		doc.Aida.Provider = detected[0].ID
		doc.Profiles["default"] = profileDoc{
			Aida: config.AidaConfig{Provider: detected[0].ID},
		}
	}

	data, err := yaml.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("marshal init config: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("# Aida runtime config\n")
	sb.WriteString("# Edit this file directly after running `aida init`.\n")
	sb.WriteString("# ACP providers are preferred. `aida.provider` selects the active provider ID.\n")

	if len(detected) == 0 {
		sb.WriteString(
			"# No supported ACP CLI was detected in PATH. Set `aida.provider` manually after choosing a provider.\n",
		)
	}

	sb.WriteByte('\n')
	sb.Write(data)
	sb.WriteByte('\n')
	sb.WriteString(initReferenceComment)

	return sb.String(), nil
}

func buildInitProviders(detected []detectedProvider) map[string]agentconfig.Config {
	providers := make(map[string]agentconfig.Config, len(detected))

	for _, provider := range detected {
		cfg, ok := detectedProviderConfig(provider)
		if !ok {
			continue
		}

		providers[provider.ID] = cfg
	}

	return providers
}

func detectedProviderConfig(provider detectedProvider) (agentconfig.Config, bool) {
	cfg := agentconfig.Config{Type: provider.Type}

	switch provider.Type {
	case agentconfig.AgentTypeCodexACP:
		cfg.CodexACP = &agentconfig.ACPConfig{Model: provider.Model}
	case agentconfig.AgentTypeOpenCodeACP:
		cfg.OpenCodeACP = &agentconfig.ACPConfig{Model: provider.Model, Mode: "plan"}
	case agentconfig.AgentTypeCopilotACP:
		cfg.CopilotACP = &agentconfig.ACPConfig{Model: provider.Model}
	case agentconfig.AgentTypeGeminiACP:
		cfg.GeminiACP = &agentconfig.ACPConfig{Model: provider.Model, Mode: "plan"}
	case agentconfig.AgentTypeClaudeCodeACP:
		cfg.ClaudeCodeACP = &agentconfig.ACPConfig{Model: provider.Model}
	default:
		return agentconfig.Config{}, false
	}

	return cfg, true
}

func detectACPProviders() []detectedProvider {
	templates := []detectedProvider{
		{ID: "codex", Type: agentconfig.AgentTypeCodexACP, Model: "gpt-5.3-codex", Binaries: []string{"codex"}},
		{
			ID:       "opencode",
			Type:     agentconfig.AgentTypeOpenCodeACP,
			Model:    "opencode/big-pickle",
			Binaries: []string{"opencode"},
		},
		{ID: "copilot", Type: agentconfig.AgentTypeCopilotACP, Model: "gpt-5-codex", Binaries: []string{"copilot"}},
		{ID: "gemini", Type: agentconfig.AgentTypeGeminiACP, Model: "gemini-3-flash-preview", Binaries: []string{"gemini"}},
		{
			ID:       "claude_code",
			Type:     agentconfig.AgentTypeClaudeCodeACP,
			Model:    "claude-sonnet-4",
			Binaries: []string{"claudecode", "claude"},
		},
	}

	detected := make([]detectedProvider, 0, len(templates))

	for _, template := range templates {
		for _, binary := range template.Binaries {
			if _, err := exec.LookPath(binary); err == nil {
				detected = append(detected, template)

				break
			}
		}
	}

	return detected
}

const initReferenceComment = `# ---------------------------------------------------------------------------
# Full config shape reference
# ---------------------------------------------------------------------------
# runtime:
#   providers:
#     codex:
#       type: codex_acp
#       codex_acp:
#         model: gpt-5.3-codex
#     opencode:
#       type: opencode_acp
#       opencode_acp:
#         model: opencode/big-pickle
#         mode: plan
#     copilot:
#       type: copilot_acp
#       copilot_acp:
#         model: gpt-5-codex
#     gemini:
#       type: gemini_acp
#       gemini_acp:
#         model: gemini-3-flash-preview
#         mode: plan
#     claude_code:
#       type: claude_code_acp
#       claude_code_acp:
#         model: claude-sonnet-4
#     generic:
#       type: generic_acp
#       generic_acp:
#         cmd: ["custom-acp"]
#         extra_args: []
#         model: custom-model
#         mode: ""
#     openai:
#       type: openai
#       openai:
#         api_key: ""
#         model: gpt-4o-mini
#     aistudio:
#       type: aistudio
#       aistudio:
#         api_key: ""
#         model: gemini-2.5-flash
#     pool:
#       type: pool
#       pool:
#         members: [codex, openai]
#   mcp_servers: {}
# aida:
#   provider: codex
#   mode: confirm
#   shell: /bin/sh
# profiles:
#   default:
#     aida:
#       provider: codex
#   api:
#     aida:
#       provider: openai
#       mode: confirm
#       shell: /bin/sh
`
