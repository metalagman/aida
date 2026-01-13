package cmd_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/metalagman/aida/cmd/aida/cmd"
	"github.com/metalagman/aida/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestProvidersListAndLogout(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")

	t.Cleanup(func() {
		os.Setenv("HOME", origHome)
	})
	os.Setenv("HOME", tmpDir)

	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"aistudio": {APIKey: "test-key", Model: "gemini-2.0-flash-exp"},
		},
		DefaultProvider: "aistudio",
	}
	_, err := config.Save(cfg)
	require.NoError(t, err)

	var out bytes.Buffer

	root := cmd.NewRootCmd()
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"providers", "list"})

	err = root.Execute()
	require.NoError(t, err)
	require.Contains(t, out.String(), "aistudio (default)")

	out.Reset()

	root = cmd.NewRootCmd()
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"providers", "logout", "aistudio"})

	err = root.Execute()
	require.NoError(t, err)
	require.Contains(t, out.String(), "Removed aistudio")

	out.Reset()

	root = cmd.NewRootCmd()
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"providers", "list"})

	err = root.Execute()
	require.NoError(t, err)
	require.True(t, strings.Contains(out.String(), "No providers configured."))
}

func TestProvidersSetModel(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")

	t.Cleanup(func() {
		os.Setenv("HOME", origHome)
	})
	os.Setenv("HOME", tmpDir)

	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"aistudio": {APIKey: "test-key", Model: "gemini-3-flash"},
		},
		DefaultProvider: "aistudio",
	}
	_, err := config.Save(cfg)
	require.NoError(t, err)

	var out bytes.Buffer

	root := cmd.NewRootCmd()
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"providers", "set-model", "aistudio", "gemini-3-flash-preview"})

	err = root.Execute()
	require.NoError(t, err)
	require.Contains(t, out.String(), "Set aistudio model to gemini-3-flash-preview")

	loaded, err := config.Load()
	require.NoError(t, err)

	provider, ok := loaded.FindProvider("aistudio")
	require.True(t, ok)
	require.Equal(t, "gemini-3-flash-preview", provider.Model)
}

func TestProvidersConfigure(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")

	t.Cleanup(func() {
		os.Setenv("HOME", origHome)
	})
	os.Setenv("HOME", tmpDir)

	var out bytes.Buffer

	root := cmd.NewRootCmd()
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{
		"providers", "configure", "aistudio",
		"--api-key", "test-key",
		"--model", "gemini-3-flash-preview",
	})

	err := root.Execute()
	require.NoError(t, err)
	require.Contains(t, out.String(), "Configured aistudio")

	loaded, err := config.Load()
	require.NoError(t, err)

	provider, ok := loaded.FindProvider("aistudio")
	require.True(t, ok)
	require.Equal(t, "test-key", provider.APIKey)
	require.Equal(t, "gemini-3-flash-preview", provider.Model)
}

func TestProvidersConfigureEOFInputUsesDefaultModel(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")

	t.Cleanup(func() {
		os.Setenv("HOME", origHome)
	})
	os.Setenv("HOME", tmpDir)

	var out bytes.Buffer

	root := cmd.NewRootCmd()
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader("test-key"))
	root.SetArgs([]string{"providers", "configure", "aistudio"})

	err := root.Execute()
	require.NoError(t, err)
	require.Contains(t, out.String(), "Configured aistudio")

	loaded, err := config.Load()
	require.NoError(t, err)

	provider, ok := loaded.FindProvider("aistudio")
	require.True(t, ok)
	require.Equal(t, "test-key", provider.APIKey)
	require.Equal(t, "gemini-2.5-flash", provider.Model)
}
