package runner_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/metalagman/aida/pkg/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeProvider struct {
	command string
	err     error
}

func (p fakeProvider) GenerateCommand(ctx context.Context, _ string) (string, error) {
	if p.err != nil {
		return "", p.err
	}

	return p.command, nil
}

type fakeExecutor struct {
	called  bool
	command string
}

func (e *fakeExecutor) Execute(_ context.Context, command string, _, _ io.Writer, _ io.Reader) error {
	e.called = true
	e.command = command

	return nil
}

func TestRunnerConfirmYes(t *testing.T) {
	var stdout bytes.Buffer

	exec := &fakeExecutor{}
	r := runner.Runner{
		Mode:     runner.ModeConfirm,
		Stdout:   &stdout,
		Stdin:    strings.NewReader("y\n"),
		Executor: exec,
	}

	err := r.Run(context.Background(), "list files", fakeProvider{command: "ls -la"})
	require.NoError(t, err)
	assert.True(t, exec.called)
	assert.Equal(t, "ls -la", exec.command)
	assert.Contains(t, stdout.String(), "I would run \x1b[36m`ls -la`\x1b[0m")
	assert.Contains(t, stdout.String(), "Running: \x1b[36m`ls -la`\x1b[0m")
}

func TestRunnerConfirmNo(t *testing.T) {
	var stdout bytes.Buffer

	exec := &fakeExecutor{}
	r := runner.Runner{
		Mode:     runner.ModeConfirm,
		Stdout:   &stdout,
		Stdin:    strings.NewReader("n\n"),
		Executor: exec,
	}

	err := r.Run(context.Background(), "list files", fakeProvider{command: "ls"})
	assert.ErrorIs(t, err, runner.ErrCancelled)
	assert.False(t, exec.called)
	assert.Contains(t, stdout.String(), "Canceled.")
}

func TestRunnerYOLO(t *testing.T) {
	var stdout bytes.Buffer

	exec := &fakeExecutor{}
	r := runner.Runner{
		Mode:     runner.ModeYOLO,
		Stdout:   &stdout,
		Stdin:    strings.NewReader(""),
		Executor: exec,
	}

	err := r.Run(context.Background(), "list files", fakeProvider{command: "ls"})
	require.NoError(t, err)
	assert.True(t, exec.called)
	assert.NotContains(t, stdout.String(), "confirm?")
}

func TestRunnerEmptyCommand(t *testing.T) {
	r := runner.Runner{Mode: runner.ModeConfirm}
	err := r.Run(context.Background(), "noop", fakeProvider{command: " "})
	assert.Error(t, err)
}

func TestRunnerUnableToRunLocal(t *testing.T) {
	var stdout bytes.Buffer

	exec := &fakeExecutor{}
	r := runner.Runner{
		Mode:     runner.ModeConfirm,
		Stdout:   &stdout,
		Stdin:    strings.NewReader(""),
		Executor: exec,
	}

	err := r.Run(context.Background(), "nope", fakeProvider{command: "UNABLE_TO_RUN_LOCAL"})
	assert.ErrorIs(t, err, runner.ErrCancelled)
	assert.False(t, exec.called)
	assert.Contains(t, stdout.String(), "Unable to process the request locally")
}

func TestRunnerQuietOutputsNothing(t *testing.T) {
	var stdout bytes.Buffer

	exec := &fakeExecutor{}
	r := runner.Runner{
		Mode:     runner.ModeQuiet,
		Stdout:   &stdout,
		Stdin:    strings.NewReader(""),
		Executor: exec,
	}

	err := r.Run(context.Background(), "list files", fakeProvider{command: "ls -la"})
	require.NoError(t, err)
	assert.Equal(t, "", stdout.String())
	assert.True(t, exec.called)
}

func TestRunnerPrintOutputsOnlyCommand(t *testing.T) {
	var stdout bytes.Buffer

	exec := &fakeExecutor{}
	r := runner.Runner{
		Mode:     runner.ModePrint,
		Stdout:   &stdout,
		Stdin:    strings.NewReader(""),
		Executor: exec,
	}

	err := r.Run(context.Background(), "list files", fakeProvider{command: "ls -la"})
	require.NoError(t, err)
	assert.Equal(t, "ls -la\n", stdout.String())
	assert.False(t, exec.called)
}