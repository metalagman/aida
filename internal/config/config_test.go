package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metalagman/aida/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestHome(t *testing.T) string {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")

	t.Cleanup(func() {
		os.Setenv("HOME", origHome)
	})
	os.Setenv("HOME", tmpDir)

	return tmpDir
}

func assertConfigLoaded(t *testing.T, cfg *config.Config) {
	require.Equal(t, "aistudio", cfg.DefaultProvider)
	require.Len(t, cfg.Providers, 1)

	provider, ok := cfg.Providers["aistudio"]
	require.True(t, ok)
	assert.Equal(t, "test-key", provider.APIKey)
	assert.Equal(t, "gemini-2.0-flash-exp", provider.Model)
	assert.Equal(t, "confirm", cfg.Mode)
	assert.Equal(t, "/bin/sh", cfg.Shell)
}

func TestLoad_TOML(t *testing.T) {
	tmpDir := setupTestHome(t)
	configDir := filepath.Join(tmpDir, ".config", "aida")
	err := os.MkdirAll(configDir, 0o755)
	require.NoError(t, err)

	configContent := `
[provider.aistudio]
api_key = "test-key"
model = "gemini-2.0-flash-exp"
`
	err = os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(configContent), 0o644)
	require.NoError(t, err)

	cfg, err := config.Load()
	require.NoError(t, err)
	assertConfigLoaded(t, cfg)
}

func TestLoad_YAML(t *testing.T) {
	tmpDir := setupTestHome(t)
	configDir := filepath.Join(tmpDir, ".config", "aida")
	err := os.MkdirAll(configDir, 0o755)
	require.NoError(t, err)

	configContent := `
provider:
  aistudio:
    api_key: test-key
    model: gemini-2.0-flash-exp
`
	err = os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0o644)
	require.NoError(t, err)

	cfg, err := config.Load()
	require.NoError(t, err)
	assertConfigLoaded(t, cfg)
}

func TestLoad_Defaults(t *testing.T) {
	_ = setupTestHome(t)
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Empty(t, cfg.DefaultProvider)
	assert.Empty(t, cfg.Providers)
	assert.Equal(t, "confirm", cfg.Mode)
	assert.Equal(t, "/bin/sh", cfg.Shell)
}

func TestLoad_EnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")

	t.Cleanup(func() {
		os.Setenv("HOME", origHome)
		os.Unsetenv("AIDA_MODE")
	})
	os.Setenv("HOME", tmpDir)
	os.Setenv("AIDA_MODE", "quiet")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "quiet", cfg.Mode)
	assert.Equal(t, "/bin/sh", cfg.Shell)
}

func TestLoad_SpecificProviderEnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")

	t.Cleanup(func() {
		os.Setenv("HOME", origHome)
		os.Unsetenv("AIDA_PROVIDER_AISTUDIO_API_KEY")
		os.Unsetenv("AIDA_PROVIDER_AISTUDIO_MODEL")
	})
	os.Setenv("HOME", tmpDir)
	os.Setenv("AIDA_PROVIDER_AISTUDIO_API_KEY", "specific-key")
	os.Setenv("AIDA_PROVIDER_AISTUDIO_MODEL", "gemini-ultra")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, "aistudio", cfg.DefaultProvider)
	require.Len(t, cfg.Providers, 1)

	provider, ok := cfg.Providers["aistudio"]
	require.True(t, ok)
	assert.Equal(t, "specific-key", provider.APIKey)
	assert.Equal(t, "gemini-ultra", provider.Model)
}

func TestLoad_EnvAPIKeyDoesNotOverrideModel(t *testing.T) {
	tmpDir := setupTestHome(t)
	configDir := filepath.Join(tmpDir, ".config", "aida")
	err := os.MkdirAll(configDir, 0o755)
	require.NoError(t, err)

	configContent := `
[provider.aistudio]
api_key = "file-key"
model = "gemini-2.0-flash-exp"
`
	err = os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(configContent), 0o644)
	require.NoError(t, err)

	t.Setenv("AIDA_PROVIDER_AISTUDIO_API_KEY", "env-key")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, "aistudio", cfg.DefaultProvider)

	provider, ok := cfg.Providers["aistudio"]
	require.True(t, ok)
	assert.Equal(t, "env-key", provider.APIKey)
	assert.Equal(t, "gemini-2.0-flash-exp", provider.Model)
}

func TestLoad_Overrides(t *testing.T) {
	tmpDir := setupTestHome(t)
	configDir := filepath.Join(tmpDir, ".config", "aida")
	err := os.MkdirAll(configDir, 0o755)
	require.NoError(t, err)

	configContent := `
mode = "confirm"
shell = "/usr/bin/env bash"
`
	err = os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(configContent), 0o644)
	require.NoError(t, err)

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "confirm", cfg.Mode)
	assert.Equal(t, "/usr/bin/env bash", cfg.Shell)
}

func TestSave_New(t *testing.T) {
	_ = setupTestHome(t)
	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"aistudio": {
				APIKey: "saved-key",
				Model:  "gemini-2.0-flash-exp",
			},
		},
		DefaultProvider: "aistudio",
		Mode:            "confirm",
		Shell:           "/bin/bash",
	}

	path, err := config.Save(cfg)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(path, "config.toml"))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "api_key")
	assert.Contains(t, string(data), "saved-key")
}

func TestUpsertProvider_DefaultModel(t *testing.T) {
	cfg := &config.Config{}
	cfg.UpsertProvider("aistudio", config.ProviderConfig{
		APIKey: "test-key",
	})

	require.Len(t, cfg.Providers, 1)

	provider, ok := cfg.Providers["aistudio"]
	require.True(t, ok)
	assert.Equal(t, "gemini-2.5-flash", provider.Model)
}

func TestSave_OverwriteYAML(t *testing.T) {
	tmpDir := setupTestHome(t)
	configDir := filepath.Join(tmpDir, ".config", "aida")
	err := os.MkdirAll(configDir, 0o700)
	require.NoError(t, err)

	yamlPath := filepath.Join(configDir, "config.yaml")
	err = os.WriteFile(yamlPath, []byte("provider:\n  aistudio:\n    api_key: old-key\n"), 0o600)
	require.NoError(t, err)

	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"aistudio": {
				APIKey: "yaml-key",
				Model:  "gemini-pro",
			},
		},
		DefaultProvider: "aistudio",
		Mode:            "confirm",
		Shell:           "/bin/sh",
	}

	path, err := config.Save(cfg)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(path, "config.yaml"))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "api_key")
	assert.Contains(t, string(data), "yaml-key")
}
