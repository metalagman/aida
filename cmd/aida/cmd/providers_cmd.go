package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"

	"github.com/metalagman/aida/pkg/config"
	"github.com/metalagman/aida/pkg/llm"
	"github.com/spf13/cobra"
)

func newProvidersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "Manage configured providers",
	}

	cmd.AddCommand(newProvidersListCmd())
	cmd.AddCommand(newProvidersLogoutCmd())
	cmd.AddCommand(newProvidersModelsCmd())
	cmd.AddCommand(newProvidersSetModelCmd())
	cmd.AddCommand(newProvidersConfigureCmd())
	cmd.AddCommand(newProvidersDefaultCmd())

	return cmd
}

func newProvidersDefaultCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "default [provider]",
		Short: "Get or set the default provider",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				if cfg.DefaultProvider == "" {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No default provider set.")
				} else {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), cfg.DefaultProvider)
				}

				return nil
			}

			name := normalizeProvider(args[0])
			if name == "" {
				return fmt.Errorf("unsupported provider %q", args[0])
			}

			if _, ok := cfg.FindProvider(name); !ok {
				return fmt.Errorf("provider %q not configured", name)
			}

			cfg.DefaultProvider = name

			path, err := config.Save(cfg)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Set default provider to %s in %s\n", name, path)

			return nil
		},
	}
}

func newProvidersListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if len(cfg.Providers) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No providers configured.")

				return nil
			}

			names := make([]string, 0, len(cfg.Providers))

			for name := range cfg.Providers {
				names = append(names, name)
			}

			sort.Strings(names)

			for _, name := range names {
				display := name
				if name == cfg.DefaultProvider && name != "" {
					display = name + " (default)"
				}

				_, _ = fmt.Fprintln(cmd.OutOrStdout(), display)
			}

			return nil
		},
	}
}

func newProvidersLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout <provider>",
		Short: "Remove a configured provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := normalizeProvider(args[0])
			if name == "" {
				return fmt.Errorf("unsupported provider %q", args[0])
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if !config.RemoveProvider(cfg, name) {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Provider %s not configured.\n", name)

				return nil
			}

			path, err := config.Save(cfg)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed %s from %s\n", name, path)

			return nil
		},
	}
}

func newProvidersModelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models [provider]",
		Short: "List available models for a provider",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runProvidersModels,
	}

	cmd.Flags().Bool("all", false, "Show all models, not just generateContent-capable ones")
	cmd.Flags().String("api-key", "", "API key to use for listing models")

	return cmd
}

func runProvidersModels(cmd *cobra.Command, args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	apiKey, _ := cmd.Flags().GetString("api-key")

	providerName, provider, err := resolveProviderForModels(cfg, args)
	if err != nil {
		return err
	}

	if apiKey != "" {
		provider.APIKey = apiKey
	}

	all, _ := cmd.Flags().GetBool("all")

	models, err := llm.ListModels(ctx, providerName, provider)
	if err != nil {
		return err
	}

	if !all {
		models = llm.FilterModelsForGenerateContent(models)
	}

	if len(models) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No models found.")

		return nil
	}

	for _, model := range models {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), llm.DisplayModelName(model.Name))
	}

	return nil
}

func resolveProviderForModels(cfg *config.Config, args []string) (string, config.ProviderConfig, error) {
	if len(args) == 0 {
		return cfg.ActiveProvider()
	}

	providerName := normalizeProvider(args[0])
	if providerName == "" {
		return "", config.ProviderConfig{}, fmt.Errorf("unsupported provider %q", args[0])
	}

	provider, _ := cfg.FindProvider(providerName)

	return providerName, provider, nil
}

func newProvidersSetModelCmd() *cobra.Command {
	const minArgs = 1

	const maxArgs = 2

	cmd := &cobra.Command{
		Use:   "set-model <provider> [model]",
		Short: "Set the default model for a provider",
		Args:  cobra.RangeArgs(minArgs, maxArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := normalizeProvider(args[0])
			if name == "" {
				return fmt.Errorf("unsupported provider %q", args[0])
			}

			model, _ := cmd.Flags().GetString("model")
			if len(args) > 1 {
				model = args[1]
			}

			model = strings.TrimSpace(model)
			if model == "" {
				return fmt.Errorf("model is required (either as an argument or via --model flag)")
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if _, ok := cfg.FindProvider(name); !ok {
				return fmt.Errorf("provider %q not configured", name)
			}

			cfg.UpsertProvider(name, config.ProviderConfig{
				Model: model,
			})

			path, err := config.Save(cfg)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Set %s model to %s in %s\n", name, model, path)

			return nil
		},
	}

	cmd.Flags().String("model", "", "Model to set (can also be provided as a second argument)")

	return cmd
}

func newProvidersConfigureCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure <provider>",
		Short: "Configure provider credentials and defaults",
		Args:  cobra.ExactArgs(1),
		RunE:  runProvidersConfigure,
	}

	cmd.Flags().String("api-key", "", "API key to store (skips prompt)")
	cmd.Flags().String("model", "", "Default model to use (skips prompt)")

	return cmd
}

func runProvidersConfigure(cmd *cobra.Command, args []string) error {
	name := normalizeProvider(args[0])
	if name == "" {
		return fmt.Errorf("unsupported provider %q", args[0])
	}

	apiKey, _ := cmd.Flags().GetString("api-key")
	model, _ := cmd.Flags().GetString("model")

	if apiKey == "" {
		var err error

		apiKey, err = promptForAPIKey(cmd, name)
		if err != nil {
			return err
		}
	}

	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("api key is required")
	}

	if model == "" {
		var err error

		model, err = promptForModel(cmd, name)
		if err != nil {
			return err
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	cfg.UpsertProvider(name, config.ProviderConfig{
		APIKey: apiKey,
		Model:  model,
	})

	path, err := config.Save(cfg)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Configured %s in %s\n", name, path)

	return nil
}

func promptForAPIKey(cmd *cobra.Command, provider string) (string, error) {
	out := cmd.OutOrStdout()
	if hint := apiKeyHint(provider); hint != "" {
		_, _ = fmt.Fprintf(out, "Create an API key at: %s\n", hint)
	}

	_, _ = fmt.Fprint(out, "Enter API key: ")

	reader := bufio.NewReader(cmd.InOrStdin())

	key, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read API key: %w", err)
	}

	return strings.TrimSpace(key), nil
}

func apiKeyHint(provider string) string {
	switch normalizeProvider(provider) {
	case "aistudio":
		return "https://aistudio.google.com/api-keys"
	case "openai":
		return "https://platform.openai.com/api-keys"
	default:
		return ""
	}
}

func promptForModel(cmd *cobra.Command, provider string) (string, error) {
	out := cmd.OutOrStdout()
	defaultModel := config.DefaultModelForProvider(provider)

	if defaultModel != "" {
		_, _ = fmt.Fprintf(out, "Enter model (default: %s): ", defaultModel)
	} else {
		_, _ = fmt.Fprint(out, "Enter model (optional): ")
	}

	reader := bufio.NewReader(cmd.InOrStdin())

	value, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read model: %w", err)
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return defaultModel, nil
	}

	return value, nil
}
