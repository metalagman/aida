//nolint:testpackage // white-box coverage for init helpers is intentional here.
package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/aida/internal/config"
)

func TestBuildInitConfigIncludesRemoteProviderTypes(t *testing.T) {
	emptyPath := t.TempDir()
	t.Setenv("PATH", emptyPath)

	content, err := buildInitConfig()
	if err != nil {
		t.Fatalf("buildInitConfig() error = %v", err)
	}

	for _, want := range []string{
		"runtime:",
		"providers:",
		"type: openai",
		"type: aistudio",
		"profiles:",
		"# No supported ACP CLI was detected in PATH.",
		"#       type: generic_acp",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("buildInitConfig() missing %q in output:\n%s", want, content)
		}
	}
}

func TestRunInitWritesCanonicalConfig(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	binDir := t.TempDir()
	t.Setenv("PATH", binDir)
	writeExecutable(t, filepath.Join(binDir, "codex"))

	cmd := newInitCmd()
	if err := runInit(cmd, &initFlags{}); err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	configPath := filepath.Join(tmpHome, ".config", "aida", "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", configPath, err)
	}

	content := string(data)

	liveConfig, _, _ := strings.Cut(
		content,
		"# ---------------------------------------------------------------------------",
	)
	for _, want := range []string{
		"aida:",
		"mode: confirm",
		"shell: /bin/sh",
		"type: codex_acp",
		"codex_acp:",
		"# Full config shape reference",
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

func TestRunInitWritesPlanModeForOpenCodeAndGemini(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	binDir := t.TempDir()
	t.Setenv("PATH", binDir)
	writeExecutable(t, filepath.Join(binDir, "opencode"))
	writeExecutable(t, filepath.Join(binDir, "gemini"))

	cmd := newInitCmd()
	if err := runInit(cmd, &initFlags{}); err != nil {
		t.Fatalf("runInit() error = %v", err)
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
		t.Fatalf("opencode mode = %q, want plan", got)
	}

	geminiCfg, ok := cfg.Runtime.Providers["gemini"]
	if !ok {
		t.Fatalf("cfg.Runtime.Providers missing gemini: %#v", cfg.Runtime.Providers)
	}

	if got := geminiCfg.GeminiACP.Mode; got != "plan" {
		t.Fatalf("gemini mode = %q, want plan", got)
	}
}

func TestRunInitGeneratedConfigRoundTrips(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	binDir := t.TempDir()
	t.Setenv("PATH", binDir)
	writeExecutable(t, filepath.Join(binDir, "codex"))

	if err := runInit(newInitCmd(), &initFlags{}); err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}

	if got := cfg.Aida.Provider; got != "codex" {
		t.Fatalf("cfg.Aida.Provider = %q, want codex", got)
	}

	if err := cfg.ValidateSelectedRuntime(); err != nil {
		t.Fatalf("cfg.ValidateSelectedRuntime() error = %v", err)
	}
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()

	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
