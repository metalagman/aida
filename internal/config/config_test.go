package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/aida/internal/config"
	"github.com/normahq/norma/pkg/runtime/agentconfig"
	runtimeconfig "github.com/normahq/norma/pkg/runtime/appconfig"
)

const openAIMCPUnsupportedError = `provider "openai": agent config schema validation failed: ` +
	`mcp_servers is not supported for type openai`

func TestLoadProfileDefaultProfileOverlay(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	path, err := config.ResolveConfigPath()
	if err != nil {
		t.Fatalf("ResolveConfigPath() error = %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}

	if err := os.WriteFile(path, []byte(`runtime:
  providers:
    codex:
      type: codex_acp
      codex_acp:
        model: gpt-5.3-codex
    openai:
      type: openai
      openai:
        api_key: file-key
        model: gpt-4o-mini
aida:
  provider: codex
  mode: confirm
  shell: /bin/sh
profiles:
  default:
    aida:
      provider: openai
`), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}

	if cfg.Aida.Provider != "openai" {
		t.Fatalf("cfg.Aida.Provider = %q, want %q", cfg.Aida.Provider, "openai")
	}

	if cfg.Aida.Mode != "confirm" {
		t.Fatalf("cfg.Aida.Mode = %q, want %q", cfg.Aida.Mode, "confirm")
	}

	if cfg.Aida.Shell != "/bin/sh" {
		t.Fatalf("cfg.Aida.Shell = %q, want %q", cfg.Aida.Shell, "/bin/sh")
	}
}

func TestLoadProfileExplicitMissingProfileFails(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	path, err := config.ResolveConfigPath()
	if err != nil {
		t.Fatalf("ResolveConfigPath() error = %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}

	if err := os.WriteFile(path, []byte(`runtime:
  providers:
    codex:
      type: codex_acp
      codex_acp:
        model: gpt-5.3-codex
`), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}

	_, err = config.LoadProfile("missing")
	if err == nil {
		t.Fatal("LoadProfile(missing) error = nil, want missing profile error")
	}

	if err.Error() != `top-level profile "missing" not found` {
		t.Fatalf("LoadProfile(missing) error = %q", err)
	}
}

func TestConfigRuntimeOverrides(t *testing.T) {
	cfg := &config.Config{
		Runtime: runtimeConfigWithProviders(),
		Aida: config.AidaConfig{
			Provider: "openai",
			Mode:     "confirm",
			Shell:    "/bin/sh",
		},
	}

	if err := cfg.SetProviderModel("openai", "gpt-5"); err != nil {
		t.Fatalf("SetProviderModel() error = %v", err)
	}

	provider := cfg.Runtime.Providers["openai"]
	if provider.OpenAI.Model != "gpt-5" {
		t.Fatalf("provider.OpenAI.Model = %q, want %q", provider.OpenAI.Model, "gpt-5")
	}
}

func TestLoadProfileRejectsAIDAProviderEnvOverride(t *testing.T) {
	const envAIDAProvider = "AIDA_PROVIDER"

	t.Setenv("HOME", t.TempDir())
	t.Setenv(envAIDAProvider, "openai")

	_, err := config.Load()
	if err == nil {
		t.Fatal("config.Load() error = nil, want unsupported env override error")
	}

	want := "environment variable AIDA_PROVIDER is not supported; use AIDA_PROFILE or config profiles instead"
	if err.Error() != want {
		t.Fatalf("config.Load() error = %q, want %q", err.Error(), want)
	}
}

func TestLoadProfileIgnoresInvalidInactiveProvider(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	path, err := config.ResolveConfigPath()
	if err != nil {
		t.Fatalf("ResolveConfigPath() error = %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}

	if err := os.WriteFile(path, []byte(`runtime:
  providers:
    codex:
      type: codex_acp
      codex_acp:
        model: gpt-5.3-codex
    openai:
      type: openai
      mcp_servers: [broken]
      openai:
        api_key: test-key
        model: gpt-4o-mini
  mcp_servers:
    broken:
      type: stdio
aida:
  provider: codex
`), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}

	if err := cfg.ValidateSelectedRuntime(); err != nil {
		t.Fatalf("ValidateSelectedRuntime() error = %v", err)
	}
}

func TestLoadProfileRejectsMalformedOldInitConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	path, err := config.ResolveConfigPath()
	if err != nil {
		t.Fatalf("ResolveConfigPath() error = %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}

	if err := os.WriteFile(path, []byte(`runtime:
  providers:
    codex:
      type: codex_acp
      codexacp:
        model: gpt-5.3-codex
aida:
  provider: codex
`), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}

	_, err = config.Load()
	if err == nil {
		t.Fatal("config.Load() error = nil, want malformed init error")
	}

	if !strings.Contains(err.Error(), "generated by a buggy version of aida init") {
		t.Fatalf("config.Load() error = %q, want malformed init message", err)
	}

	if !strings.Contains(err.Error(), "rerun `aida init --force`") {
		t.Fatalf("config.Load() error = %q, want remediation", err)
	}
}

func TestValidateSelectedRuntimeSelectedInvalidProviderFails(t *testing.T) {
	cfg := &config.Config{
		Runtime: runtimeconfig.RuntimeConfig{
			Providers: map[string]agentconfig.Config{
				"openai": {
					Type:       agentconfig.AgentTypeOpenAI,
					MCPServers: []string{"broken"},
					OpenAI: &agentconfig.LocalAPIConfig{
						APIKey: "test-key",
						Model:  "gpt-4o-mini",
					},
				},
			},
		},
		Aida: config.AidaConfig{
			Provider: "openai",
		},
	}

	err := cfg.ValidateSelectedRuntime()
	if err == nil || err.Error() != openAIMCPUnsupportedError {
		t.Fatalf("ValidateSelectedRuntime() error = %v, want %q", err, openAIMCPUnsupportedError)
	}
}

func TestValidateSelectedRuntimeSelectedProviderIgnoresInactiveBrokenMCP(t *testing.T) {
	cfg := &config.Config{
		Runtime: runtimeconfig.RuntimeConfig{
			Providers: map[string]agentconfig.Config{
				"codex": {
					Type: agentconfig.AgentTypeCodexACP,
					CodexACP: &agentconfig.ACPConfig{
						Model: "gpt-5.3-codex",
					},
				},
				"custom": {
					Type:       agentconfig.AgentTypeGenericACP,
					MCPServers: []string{"broken"},
					GenericACP: &agentconfig.ACPConfig{
						Cmd: []string{"custom-acp"},
					},
				},
			},
			MCPServers: map[string]agentconfig.MCPServerConfig{
				"broken": {
					Type: agentconfig.MCPServerTypeStdio,
				},
			},
		},
		Aida: config.AidaConfig{
			Provider: "codex",
		},
	}

	if err := cfg.ValidateSelectedRuntime(); err != nil {
		t.Fatalf("ValidateSelectedRuntime() error = %v", err)
	}
}

func TestValidateSelectedRuntimeSelectedPoolValidatesMembers(t *testing.T) {
	cfg := &config.Config{
		Runtime: runtimeconfig.RuntimeConfig{
			Providers: map[string]agentconfig.Config{
				"pool": {
					Type: agentconfig.AgentTypePool,
					PoolConfig: &agentconfig.PoolConfig{
						Members: []string{"codex", "openai"},
					},
				},
				"codex": {
					Type: agentconfig.AgentTypeCodexACP,
					CodexACP: &agentconfig.ACPConfig{
						Model: "gpt-5.3-codex",
					},
				},
				"openai": {
					Type:       agentconfig.AgentTypeOpenAI,
					MCPServers: []string{"broken"},
					OpenAI: &agentconfig.LocalAPIConfig{
						APIKey: "test-key",
						Model:  "gpt-4o-mini",
					},
				},
			},
		},
		Aida: config.AidaConfig{
			Provider: "pool",
		},
	}

	err := cfg.ValidateSelectedRuntime()
	if err == nil || err.Error() != openAIMCPUnsupportedError {
		t.Fatalf("ValidateSelectedRuntime() error = %v, want %q", err, openAIMCPUnsupportedError)
	}
}

func runtimeConfigWithProviders() runtimeconfig.RuntimeConfig {
	return runtimeconfig.RuntimeConfig{
		Providers: map[string]agentconfig.Config{
			"openai": {
				Type: agentconfig.AgentTypeOpenAI,
				OpenAI: &agentconfig.LocalAPIConfig{
					Model: "gpt-4o-mini",
				},
			},
			"codex": {
				Type: agentconfig.AgentTypeCodexACP,
				CodexACP: &agentconfig.ACPConfig{
					Model: "gpt-5.3-codex",
				},
			},
		},
	}
}
