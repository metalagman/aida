package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/aida/cmd/aida/cmd"
	"github.com/metalagman/aida/internal/config"
)

func TestInitWritesCanonicalConfig(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	binDir := t.TempDir()
	t.Setenv("PATH", binDir)
	writeExecutable(t, filepath.Join(binDir, "codex"))

	var stdout bytes.Buffer

	root := cmd.NewRootCmd()
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"init"})

	if err := root.Execute(); err != nil {
		t.Fatalf("root.Execute() error = %v", err)
	}

	configPath := filepath.Join(tmpHome, ".config", "aida", "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", configPath, err)
	}

	content := string(data)
	separator := "# ---------------------------------------------------------------------------"
	liveConfig, _, _ := strings.Cut(content, separator)

	for _, want := range []string{
		"aida:",
		"mode: confirm",
		"shell: /bin/sh",
		"type: codex_acp",
		"codex_acp:",
		"# Full config shape reference",
		"type: openai",
		"type: aistudio",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("generated config missing %q in output:\n%s", want, content)
		}
	}

	for _, unwanted := range []string{
		"\n    openai:",
		"\n    aistudio:",
		"\n    pool:",
		"\n  api:",
	} {
		if strings.Contains(liveConfig, unwanted) {
			t.Fatalf("generated live config unexpectedly contains %q:\n%s", unwanted, liveConfig)
		}
	}
}

func TestInitWritesPlanModeForOpenCodeAndGemini(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	binDir := t.TempDir()
	t.Setenv("PATH", binDir)
	writeExecutable(t, filepath.Join(binDir, "opencode"))
	writeExecutable(t, filepath.Join(binDir, "gemini"))

	root := cmd.NewRootCmd()
	root.SetArgs([]string{"init"})

	if err := root.Execute(); err != nil {
		t.Fatalf("root.Execute() error = %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}

	opencodeCfg, ok := cfg.Runtime.Providers["opencode"]
	if !ok {
		t.Fatalf("cfg.Runtime.Providers missing opencode: %#v", cfg.Runtime.Providers)
	}

	if got := opencodeCfg.OpenCodeACP.Mode; got != "plan" {
		t.Fatalf("opencode mode = %q, want %q", got, "plan")
	}

	geminiCfg, ok := cfg.Runtime.Providers["gemini"]
	if !ok {
		t.Fatalf("cfg.Runtime.Providers missing gemini: %#v", cfg.Runtime.Providers)
	}

	if got := geminiCfg.GeminiACP.Mode; got != "plan" {
		t.Fatalf("gemini mode = %q, want %q", got, "plan")
	}
}

func TestInitWritesProfilesForDetectedProviders(t *testing.T) {
	path := runInitWithDetectedProviders(t, "codex", "opencode", "gemini")
	liveConfig := readLiveInitConfig(t, path)

	assertContainsAll(t, liveConfig, []string{
		"profiles:",
		"\n    default:",
		"\n    codex:",
		"\n    opencode:",
		"\n    gemini:",
	})

	assertContainsNone(t, liveConfig, []string{
		"\n    openai:",
		"\n    aistudio:",
		"\n    pool:",
	})
}

func TestInitGeneratedProviderProfilesRoundTrip(t *testing.T) {
	runInitWithDetectedProviders(t, "codex", "opencode", "gemini")

	for _, tt := range []struct {
		name    string
		profile string
		want    string
	}{
		{name: "implicit default", profile: "", want: "codex"},
		{name: "default", profile: "default", want: "codex"},
		{name: "codex", profile: "codex", want: "codex"},
		{name: "opencode", profile: "opencode", want: "opencode"},
		{name: "gemini", profile: "gemini", want: "gemini"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.LoadProfile(tt.profile)
			if err != nil {
				t.Fatalf("config.LoadProfile(%q) error = %v", tt.profile, err)
			}

			if cfg.Aida.Provider != tt.want {
				t.Fatalf("cfg.Aida.Provider = %q, want %q", cfg.Aida.Provider, tt.want)
			}

			if err := cfg.ValidateSelectedRuntime(); err != nil {
				t.Fatalf("cfg.ValidateSelectedRuntime() error = %v", err)
			}
		})
	}
}

func TestInitDetectsClaudeCodeFromClaudeBinary(t *testing.T) {
	runInitWithDetectedProviders(t, "claude")

	cfg, err := config.LoadProfile("claude_code")
	if err != nil {
		t.Fatalf("config.LoadProfile(%q) error = %v", "claude_code", err)
	}

	if cfg.Aida.Provider != "claude_code" {
		t.Fatalf("cfg.Aida.Provider = %q, want %q", cfg.Aida.Provider, "claude_code")
	}

	provider, ok := cfg.Runtime.Providers["claude_code"]
	if !ok {
		t.Fatalf("cfg.Runtime.Providers missing claude_code: %#v", cfg.Runtime.Providers)
	}

	if provider.ClaudeCodeACP == nil {
		t.Fatalf("provider.ClaudeCodeACP is nil")
	}

	if got := provider.ClaudeCodeACP.Model; got != "claude-sonnet-4" {
		t.Fatalf("provider.ClaudeCodeACP.Model = %q, want %q", got, "claude-sonnet-4")
	}

	if err := cfg.ValidateSelectedRuntime(); err != nil {
		t.Fatalf("cfg.ValidateSelectedRuntime() error = %v", err)
	}
}

func TestInitDoesNotDetectClaudeCodeFromClaudeCodeBinary(t *testing.T) {
	path := runInitWithDetectedProviders(t, "claudecode")
	liveConfig := readLiveInitConfig(t, path)

	assertContainsNone(t, liveConfig, []string{
		"\n    claude_code:",
	})
}

func TestInitGeneratedConfigRoundTrips(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	binDir := t.TempDir()
	t.Setenv("PATH", binDir)
	writeExecutable(t, filepath.Join(binDir, "codex"))

	root := cmd.NewRootCmd()
	root.SetArgs([]string{"init"})

	if err := root.Execute(); err != nil {
		t.Fatalf("root.Execute() error = %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}

	if cfg.Aida.Provider != "codex" {
		t.Fatalf("cfg.Aida.Provider = %q, want %q", cfg.Aida.Provider, "codex")
	}

	if err := cfg.ValidateSelectedRuntime(); err != nil {
		t.Fatalf("cfg.ValidateSelectedRuntime() error = %v", err)
	}
}

func TestInitWithoutDetectedProvidersIncludesWarning(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("PATH", t.TempDir())

	root := cmd.NewRootCmd()
	root.SetArgs([]string{"init"})

	if err := root.Execute(); err != nil {
		t.Fatalf("root.Execute() error = %v", err)
	}

	path, err := config.ResolveConfigPath()
	if err != nil {
		t.Fatalf("ResolveConfigPath() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}

	content := string(data)
	if !strings.Contains(content, "# No supported ACP CLI was detected in PATH.") {
		t.Fatalf("generated config missing no-provider warning:\n%s", content)
	}
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()

	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func runInitWithDetectedProviders(t *testing.T, binaries ...string) string {
	t.Helper()

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	binDir := t.TempDir()
	t.Setenv("PATH", binDir)

	for _, binary := range binaries {
		writeExecutable(t, filepath.Join(binDir, binary))
	}

	root := cmd.NewRootCmd()
	root.SetArgs([]string{"init"})

	if err := root.Execute(); err != nil {
		t.Fatalf("root.Execute() error = %v", err)
	}

	path, err := config.ResolveConfigPath()
	if err != nil {
		t.Fatalf("ResolveConfigPath() error = %v", err)
	}

	return path
}

func readLiveInitConfig(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}

	separator := "# ---------------------------------------------------------------------------"
	liveConfig, _, _ := strings.Cut(string(data), separator)

	return liveConfig
}

func assertContainsAll(t *testing.T, content string, values []string) {
	t.Helper()

	for _, value := range values {
		if !strings.Contains(content, value) {
			t.Fatalf("generated live config missing %q:\n%s", value, content)
		}
	}
}

func assertContainsNone(t *testing.T, content string, values []string) {
	t.Helper()

	for _, value := range values {
		if strings.Contains(content, value) {
			t.Fatalf("generated live config unexpectedly contains %q:\n%s", value, content)
		}
	}
}
