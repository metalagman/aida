package runner

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

var ErrCancelled = errors.New("command canceled")

type Executor interface {
	Execute(ctx context.Context, command string, stdout, stderr io.Writer, stdin io.Reader) error
}

type ShellExecutor struct {
	Shell string
}

func (e ShellExecutor) Execute(ctx context.Context, command string, stdout, stderr io.Writer, stdin io.Reader) error {
	shell := e.Shell
	if shell == "" {
		shell = "/bin/sh"
	}

	cmd := exec.CommandContext(ctx, shell, "-c", command)

	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = stdin

	return cmd.Run()
}

type RunMode string

const (
	ModeYOLO    RunMode = "yolo"
	ModeConfirm RunMode = "confirm"
	ModeQuiet   RunMode = "quiet"
	ModeDryRun  RunMode = "dry-run"
)

type Runner struct {
	Mode     RunMode
	Stdout   io.Writer
	Stderr   io.Writer
	PromptIn io.ReadCloser
	ExecIn   io.Reader
	Executor Executor
}

func (r Runner) Run(ctx context.Context, command string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return errors.New("empty command generated")
	}

	if command == "UNABLE_TO_RUN_LOCAL" {
		if r.Mode != ModeQuiet {
			_, _ = fmt.Fprintln(r.Stdout, "Unable to process the request locally with shell scripting tools.")
		}

		return ErrCancelled
	}

	switch r.Mode {
	case ModeDryRun:
		_, _ = fmt.Fprintln(r.Stdout, command)

		return nil
	case ModeQuiet:
		return r.Executor.Execute(ctx, command, io.Discard, io.Discard, r.execIn())
	case ModeYOLO:
		return r.runWithConfirmation(ctx, command, false)
	default:
		return r.runWithConfirmation(ctx, command, true)
	}
}

const (
	colorReset = "\033[0m"
	colorCyan  = "\033[36m"
)

func (r Runner) runWithConfirmation(ctx context.Context, command string, forceConfirm bool) error {
	if forceConfirm {
		if err := r.confirm(ctx, command); err != nil {
			return err
		}
	} else {
		_, _ = fmt.Fprintf(r.Stdout, "Running: %s`%s`%s\n", colorCyan, command, colorReset)
	}

	return r.Executor.Execute(ctx, command, r.Stdout, r.Stderr, r.execIn())
}

func (r Runner) confirm(ctx context.Context, command string) error {
	_, _ = fmt.Fprintf(r.Stdout, "I would run %s`%s`%s, confirm? [y/N] ", colorCyan, command, colorReset)

	type readResult struct {
		answer string
		err    error
	}

	done := make(chan readResult, 1)
	promptIn := r.promptIn()

	go func() {
		reader := bufio.NewReader(promptIn)

		answer, err := reader.ReadString('\n')

		done <- readResult{answer, err}
	}()

	select {
	case <-ctx.Done():
		_ = promptIn.Close()

		<-done

		_, _ = fmt.Fprintln(r.Stdout)

		return ErrCancelled
	case res := <-done:
		if res.err != nil && !errors.Is(res.err, io.EOF) {
			return fmt.Errorf("read confirmation: %w", res.err)
		}

		answer := strings.TrimSpace(strings.ToLower(res.answer))
		if answer == "y" || answer == "yes" {
			return nil
		}

		_, _ = fmt.Fprintln(r.Stdout, "Canceled.")

		return ErrCancelled
	}
}

func (r Runner) promptIn() io.ReadCloser {
	if r.PromptIn != nil {
		return r.PromptIn
	}

	if r.ExecIn == nil {
		return io.NopCloser(strings.NewReader(""))
	}

	return io.NopCloser(r.ExecIn)
}

func (r Runner) execIn() io.Reader {
	if r.ExecIn != nil {
		return r.ExecIn
	}

	if r.PromptIn != nil {
		return r.PromptIn
	}

	return strings.NewReader("")
}
