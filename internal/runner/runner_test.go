package runner_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/metalagman/aida/internal/runner"
)

type fakeExecutor struct {
	called  bool
	command string
}

func (e *fakeExecutor) Execute(_ context.Context, command string, stdout, _ io.Writer, _ io.Reader) error {
	e.called = true
	e.command = command

	if command == "ls -la" {
		_, _ = fmt.Fprintln(stdout, "total 0")
	}

	return nil
}

type blockingReadCloser struct {
	release <-chan struct{}
}

func (r blockingReadCloser) Read([]byte) (int, error) {
	<-r.release

	return 0, io.EOF
}

func (r blockingReadCloser) Close() error {
	return nil
}

func TestRunnerConfirmContextCancel(t *testing.T) {
	var stdout bytes.Buffer

	exec := &fakeExecutor{}
	pr, pw := io.Pipe()

	t.Cleanup(func() {
		_ = pr.Close()
		_ = pw.Close()
	})

	r := runner.Runner{
		Mode:     runner.ModeConfirm,
		Stdout:   &stdout,
		PromptIn: pr,
		ExecIn:   strings.NewReader(""),
		Executor: exec,
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := r.Run(ctx, "ls")
	if err != runner.ErrCancelled {
		t.Fatalf("Run() error = %v, want %v", err, runner.ErrCancelled)
	}

	if exec.called {
		t.Fatal("Run() unexpectedly executed command after cancellation")
	}
}

func TestRunnerConfirmContextCancelDoesNotWaitForPromptRead(t *testing.T) {
	release := make(chan struct{})
	defer close(release)

	var stdout bytes.Buffer

	r := runner.Runner{
		Mode:     runner.ModeConfirm,
		Stdout:   &stdout,
		PromptIn: blockingReadCloser{release: release},
		ExecIn:   strings.NewReader(""),
		Executor: &fakeExecutor{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan error, 1)

	go func() {
		done <- r.Run(ctx, "ls")
	}()

	select {
	case err := <-done:
		if err != runner.ErrCancelled {
			t.Fatalf("Run() error = %v, want %v", err, runner.ErrCancelled)
		}
	case <-time.After(time.Second):
		t.Fatal("Run() blocked waiting for prompt read after context cancellation")
	}
}

func TestRunnerConfirmYes(t *testing.T) {
	var stdout bytes.Buffer

	exec := &fakeExecutor{}
	r := runner.Runner{
		Mode:     runner.ModeConfirm,
		Stdout:   &stdout,
		PromptIn: io.NopCloser(strings.NewReader("y\n")),
		ExecIn:   strings.NewReader(""),
		Executor: exec,
	}

	err := r.Run(context.Background(), "ls -la")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !exec.called {
		t.Fatal("Run() did not execute confirmed command")
	}

	if exec.command != "ls -la" {
		t.Fatalf("executed command = %q, want %q", exec.command, "ls -la")
	}

	got := stdout.String()
	if !strings.Contains(got, "I would run \x1b[36m`ls -la`\x1b[0m") {
		t.Fatalf("stdout = %q, want confirmation prompt", got)
	}

	if strings.Contains(got, "Running:") {
		t.Fatalf("stdout = %q, did not expect yolo output", got)
	}
}

func TestRunnerConfirmNo(t *testing.T) {
	var stdout bytes.Buffer

	exec := &fakeExecutor{}
	r := runner.Runner{
		Mode:     runner.ModeConfirm,
		Stdout:   &stdout,
		PromptIn: io.NopCloser(strings.NewReader("n\n")),
		ExecIn:   strings.NewReader(""),
		Executor: exec,
	}

	err := r.Run(context.Background(), "ls")
	if err != runner.ErrCancelled {
		t.Fatalf("Run() error = %v, want %v", err, runner.ErrCancelled)
	}

	if exec.called {
		t.Fatal("Run() unexpectedly executed rejected command")
	}

	if !strings.Contains(stdout.String(), "Canceled.") {
		t.Fatalf("stdout = %q, want cancellation message", stdout.String())
	}
}

func TestRunnerYOLO(t *testing.T) {
	var stdout bytes.Buffer

	exec := &fakeExecutor{}
	r := runner.Runner{
		Mode:     runner.ModeYOLO,
		Stdout:   &stdout,
		PromptIn: io.NopCloser(strings.NewReader("")),
		ExecIn:   strings.NewReader(""),
		Executor: exec,
	}

	err := r.Run(context.Background(), "ls")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !exec.called {
		t.Fatal("Run() did not execute yolo command")
	}

	if !strings.Contains(stdout.String(), "Running: \x1b[36m`ls`\x1b[0m") {
		t.Fatalf("stdout = %q, want yolo banner", stdout.String())
	}
}

func TestRunnerEmptyCommand(t *testing.T) {
	r := runner.Runner{Mode: runner.ModeConfirm}

	if err := r.Run(context.Background(), " "); err == nil {
		t.Fatal("Run() error = nil, want error for empty command")
	}
}

func TestRunnerUnableToRunLocal(t *testing.T) {
	var stdout bytes.Buffer

	exec := &fakeExecutor{}
	r := runner.Runner{
		Mode:     runner.ModeConfirm,
		Stdout:   &stdout,
		PromptIn: io.NopCloser(strings.NewReader("")),
		ExecIn:   strings.NewReader(""),
		Executor: exec,
	}

	err := r.Run(context.Background(), "UNABLE_TO_RUN_LOCAL")
	if err != runner.ErrCancelled {
		t.Fatalf("Run() error = %v, want %v", err, runner.ErrCancelled)
	}

	if exec.called {
		t.Fatal("Run() unexpectedly executed UNABLE_TO_RUN_LOCAL sentinel")
	}

	if !strings.Contains(stdout.String(), "Unable to process the request locally") {
		t.Fatalf("stdout = %q, want local failure message", stdout.String())
	}
}

func TestRunnerQuietSuppressesCommandOutput(t *testing.T) {
	var stdout bytes.Buffer

	exec := &fakeExecutor{}
	r := runner.Runner{
		Mode:     runner.ModeQuiet,
		Stdout:   &stdout,
		PromptIn: io.NopCloser(strings.NewReader("")),
		ExecIn:   strings.NewReader(""),
		Executor: exec,
	}

	err := r.Run(context.Background(), "ls -la")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty output in quiet mode", stdout.String())
	}

	if !exec.called {
		t.Fatal("Run() did not execute quiet command")
	}
}

func TestRunnerDryRunOutputsOnlyCommand(t *testing.T) {
	var stdout bytes.Buffer

	exec := &fakeExecutor{}
	r := runner.Runner{
		Mode:     runner.ModeDryRun,
		Stdout:   &stdout,
		PromptIn: io.NopCloser(strings.NewReader("")),
		ExecIn:   strings.NewReader(""),
		Executor: exec,
	}

	err := r.Run(context.Background(), "ls -la")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if stdout.String() != "ls -la\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "ls -la\n")
	}

	if exec.called {
		t.Fatal("Run() unexpectedly executed command in dry-run mode")
	}
}
