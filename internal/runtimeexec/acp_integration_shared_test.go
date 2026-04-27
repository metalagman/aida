package runtimeexec_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/metalagman/aida/internal/config"
	"github.com/metalagman/aida/internal/runtimeexec"
	"github.com/normahq/runtime/agentconfig"
	runtimeconfig "github.com/normahq/runtime/appconfig"
)

const (
	acpIntegrationEnv     = "AIDA_RUN_ACP_INTEGRATION"
	acpIntegrationPrompt  = "count go files in current directory"
	acpIntegrationTimeout = 2 * time.Minute
)

var (
	goFilePatternDiscovery = regexp.MustCompile(`(?i)(\.go\b|\*\.go\b|-e\s+go\b|--extension(?:=|\s+)go\b)`)
	countPatternDiscovery  = regexp.MustCompile(`(?i)(wc\s+-l\b|grep\s+-c\b|awk\b.*\b(nr|NR)\b)`)
)

func requireACPIntegrationOptIn(t *testing.T) {
	t.Helper()

	if strings.TrimSpace(os.Getenv(acpIntegrationEnv)) != "1" {
		t.Skipf("%s=1 is required to run ACP integration tests", acpIntegrationEnv)
	}
}

func requireBinary(t *testing.T, binary string) {
	t.Helper()

	if _, err := exec.LookPath(binary); err != nil {
		t.Fatalf("%s binary not found in PATH: %v", binary, err)
	}
}

func generateACPCommand(t *testing.T, providerID string, providerCfg agentconfig.Config) string {
	t.Helper()

	workspace := newACPIntegrationWorkspace(t)
	t.Chdir(workspace)

	cfg := &config.Config{
		Runtime: runtimeconfig.RuntimeConfig{
			Providers: map[string]agentconfig.Config{
				providerID: providerCfg,
			},
		},
		Aida: config.AidaConfig{
			Provider: providerID,
			Mode:     "confirm",
			Shell:    "/bin/sh",
		},
	}

	if err := cfg.ValidateSelectedRuntime(); err != nil {
		t.Fatalf("ValidateSelectedRuntime() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), acpIntegrationTimeout)
	defer cancel()

	command, err := runtimeexec.GenerateCommand(ctx, cfg, formatACPIntegrationPrompt(t, acpIntegrationPrompt))
	if err != nil {
		t.Fatalf("%s GenerateCommand() error = %v", providerID, err)
	}

	return strings.TrimSpace(command)
}

func newACPIntegrationWorkspace(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	files := []string{
		"main.go",
		"pkg/foo/foo.go",
		"internal/bar/bar.go",
		"README.md",
		"scripts/build.sh",
	}

	for _, relPath := range files {
		path := filepath.Join(dir, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", path, err)
		}

		content := "placeholder\n"
		if strings.HasSuffix(relPath, ".go") {
			content = "package placeholder\n"
		}

		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", path, err)
		}
	}

	return dir
}

func formatACPIntegrationPrompt(t *testing.T, prompt string) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	return strings.TrimSpace("OS: " + runtime.GOOS +
		"\nArch: " + runtime.GOARCH +
		"\nPWD: " + wd +
		"\nShell: /bin/sh" +
		"\nRequest: " + prompt)
}

func assertCountGoFilesCommand(t *testing.T, command string) {
	t.Helper()

	if command == "" {
		t.Fatal("generated command is empty")
	}

	if command == "UNABLE_TO_RUN_LOCAL" {
		t.Fatalf("generated command unexpectedly returned %q", command)
	}

	if strings.Contains(command, "```") {
		t.Fatalf("generated command contains markdown fences: %q", command)
	}

	if strings.Contains(command, "\n") {
		t.Fatalf("generated command contains newlines: %q", command)
	}

	if !goFilePatternDiscovery.MatchString(command) {
		t.Fatalf("generated command does not appear to target go files: %q", command)
	}

	if !countPatternDiscovery.MatchString(command) {
		t.Fatalf("generated command does not appear to count files: %q", command)
	}

	discoveryMarkers := []string{"find ", "rg ", "fd ", "git ls-files", "locate ", "ls *.go", "ls ./*.go"}
	for _, marker := range discoveryMarkers {
		if strings.Contains(command, marker) {
			return
		}
	}

	t.Fatalf("generated command does not use a plausible file-discovery primitive: %q", command)
}
