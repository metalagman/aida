package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"

	"github.com/metalagman/aida/internal/config"
	"github.com/metalagman/aida/internal/runner"
	"github.com/metalagman/aida/internal/runtimeexec"
	"github.com/spf13/cobra"
)

type cliOptions struct {
	profile string
	model   string
	yolo    bool
	quiet   bool
	dryRun  bool
	shell   string
}

var rootCmd = NewRootCmd()

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}

		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func NewRootCmd() *cobra.Command {
	initDotEnv()

	opts := &cliOptions{}
	cmd := &cobra.Command{
		Use:           "aida [prompt] [-- prompt]",
		Short:         "Generate and run a single shell command from a prompt",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			prompt := PromptFromArgs(args, cmd.ArgsLenAtDash())
			if strings.TrimSpace(prompt) == "" {
				return cmd.Help()
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			cfg, loadErr := config.LoadProfile(resolveProfile(opts.profile))
			if loadErr != nil {
				return loadErr
			}

			if err := applyOverrides(cfg, opts); err != nil {
				return err
			}

			if err := cfg.ValidateSelectedRuntime(); err != nil {
				return err
			}

			r := setupRunner(cmd, opts, cfg)

			prompt = formatPromptWithShell(prompt, cfg.Aida.Shell)

			command, err := runtimeexec.GenerateCommand(ctx, cfg, prompt)
			if err != nil {
				return err
			}

			if err := r.Run(ctx, command); err != nil {
				if errors.Is(err, runner.ErrCancelled) {
					return nil
				}

				return err
			}

			return nil
		},
	}

	setupFlags(cmd, opts)
	cmd.AddCommand(newInitCmd())

	return cmd
}

func setupRunner(cmd *cobra.Command, opts *cliOptions, cfg *config.Config) runner.Runner {
	mode := runner.RunMode(cfg.Aida.Mode)

	switch {
	case opts.dryRun:
		mode = runner.ModeDryRun
	case opts.quiet:
		mode = runner.ModeQuiet
	case opts.yolo:
		mode = runner.ModeYOLO
	case mode == "":
		mode = runner.ModeConfirm
	}

	executor := runner.ShellExecutor{Shell: cfg.Aida.Shell}

	return runner.Runner{
		Mode:     mode,
		Stdout:   cmd.OutOrStdout(),
		Stderr:   cmd.ErrOrStderr(),
		PromptIn: promptInput(cmd.InOrStdin()),
		ExecIn:   cmd.InOrStdin(),
		Executor: executor,
	}
}

func setupFlags(cmd *cobra.Command, opts *cliOptions) {
	cmd.Flags().StringVar(&opts.profile, "profile", "", "config profile name")
	cmd.Flags().StringVar(&opts.model, "model", "", "LLM model name")
	cmd.Flags().StringVar(&opts.shell, "shell", "", "Shell executable for running commands")
	cmd.Flags().BoolVar(&opts.yolo, "yolo", false, "Run without confirmation")
	cmd.Flags().BoolVar(&opts.quiet, "quiet", false, "Run silently")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Print command without running")
}

func PromptFromArgs(args []string, dashIndex int) string {
	if dashIndex >= 0 && dashIndex < len(args) {
		args = args[dashIndex:]
	}

	return strings.Join(args, " ")
}

func formatPromptWithShell(prompt, fallbackShell string) string {
	shell := fallbackShell
	if shell == "" {
		shell = "/bin/sh"
	}

	wd, err := os.Getwd()
	if err != nil {
		wd = ""
	}

	return fmt.Sprintf(
		"OS: %s\nArch: %s\nPWD: %s\nShell: %s\nRequest: %s",
		runtime.GOOS,
		runtime.GOARCH,
		wd,
		shell,
		prompt,
	)
}

func applyOverrides(cfg *config.Config, opts *cliOptions) error {
	if opts == nil || cfg == nil {
		return nil
	}

	if opts.shell != "" {
		cfg.SetShell(opts.shell)
	}

	if cfg.Aida.Shell == "" {
		cfg.Aida.Shell = "/bin/sh"
	}

	return applyProviderRuntimeOverrides(cfg, opts)
}

func promptInput(r io.Reader) io.ReadCloser {
	if rc, ok := r.(io.ReadCloser); ok {
		return rc
	}

	return io.NopCloser(r)
}

func resolveProfile(flagValue string) string {
	if strings.TrimSpace(flagValue) != "" {
		return strings.TrimSpace(flagValue)
	}

	return strings.TrimSpace(os.Getenv("AIDA_PROFILE"))
}

func applyProviderRuntimeOverrides(cfg *config.Config, opts *cliOptions) error {
	if opts == nil || strings.TrimSpace(opts.model) == "" {
		return nil
	}

	providerName, err := cfg.ActiveProviderID()
	if err != nil {
		return err
	}

	return cfg.SetProviderModel(providerName, opts.model)
}
