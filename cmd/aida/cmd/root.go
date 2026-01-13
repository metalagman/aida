package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"

	"github.com/metalagman/aida/pkg/config"
	"github.com/metalagman/aida/pkg/llm"
	"github.com/metalagman/aida/pkg/runner"
	"github.com/spf13/cobra"
)

type cliOptions struct {
	provider string
	apiKey   string
	model    string
	yolo     bool
	quiet    bool
	dryRun   bool
	shell    string
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
		Args:          cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			cfg, loadErr := config.Load()
			if loadErr != nil {
				return loadErr
			}

			if err := applyOverrides(cfg, opts); err != nil {
				return err
			}

			provider, err := llm.NewProvider(ctx, cfg)
			if err != nil {
				return err
			}

			r := setupRunner(cmd, opts, cfg)

			prompt := PromptFromArgs(args, cmd.ArgsLenAtDash())
			if strings.TrimSpace(prompt) == "" {
				return errors.New("prompt is required")
			}

			prompt = formatPromptWithShell(prompt, cfg.Shell)

			if err := r.Run(ctx, prompt, provider); err != nil {
				if errors.Is(err, runner.ErrCancelled) {
					return nil
				}

				return err
			}

			return nil
		},
	}

	setupFlags(cmd, opts)
	cmd.AddCommand(newProvidersCmd())

	return cmd
}

func setupRunner(cmd *cobra.Command, opts *cliOptions, cfg *config.Config) runner.Runner {
	mode := runner.RunMode(cfg.Mode)

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

	executor := runner.ShellExecutor{Shell: cfg.Shell}

	return runner.Runner{
		Mode:     mode,
		Stdout:   cmd.OutOrStdout(),
		Stderr:   cmd.ErrOrStderr(),
		Stdin:    cmd.InOrStdin(),
		Executor: executor,
	}
}

func setupFlags(cmd *cobra.Command, opts *cliOptions) {
	cmd.Flags().StringVar(&opts.provider, "provider", "", "LLM provider (aistudio, openai)")
	cmd.Flags().StringVar(&opts.apiKey, "api-key", "", "LLM API key")
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

	if opts.provider != "" {
		normalized := normalizeProvider(opts.provider)
		if normalized == "" {
			return fmt.Errorf("unsupported provider %q", opts.provider)
		}

		cfg.DefaultProvider = normalized
	}

	if opts.shell != "" {
		cfg.Shell = opts.shell
	}

	if cfg.Shell == "" {
		cfg.Shell = "/bin/sh"
	}

	_ = os.Setenv("AIDA_SHELL", cfg.Shell)

	providerName := resolveProviderName(cfg, opts)

	if providerName != "" && (opts.apiKey != "" || opts.model != "") {
		cfg.UpsertProvider(providerName, config.ProviderConfig{
			APIKey: opts.apiKey,
			Model:  opts.model,
		})
	}

	return nil
}

func resolveProviderName(cfg *config.Config, opts *cliOptions) string {
	if opts.provider != "" {
		if normalized := normalizeProvider(opts.provider); normalized != "" {
			return normalized
		}

		return opts.provider
	}

	if opts.apiKey == "" && opts.model == "" {
		return ""
	}

	if cfg.DefaultProvider != "" {
		return cfg.DefaultProvider
	}

	if len(cfg.Providers) > 0 {
		return config.FirstProviderName(cfg.Providers)
	}

	cfg.DefaultProvider = "aistudio"

	return cfg.DefaultProvider
}
