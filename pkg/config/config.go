package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const ProviderAIStudio = "aistudio"

const (
	DirPerm       = 0o700
	FilePerm      = 0o600
	envSplitCount = 2
)

type Config struct {
	LLM             LLMConfig                 `mapstructure:"llm"              toml:"-"                yaml:"-"`
	Providers       map[string]ProviderConfig `mapstructure:"provider"         toml:"provider"         yaml:"provider"`
	DefaultProvider string `mapstructure:"default_provider" toml:"default_provider" yaml:"default_provider"`
	Mode            string                    `mapstructure:"mode"             toml:"mode"             yaml:"mode"`
	Shell           string                    `mapstructure:"shell"            toml:"shell"            yaml:"shell"`
}

type LLMConfig struct {
	Provider string `mapstructure:"provider" toml:"-" yaml:"-"`
	APIKey   string `mapstructure:"api_key"  toml:"-" yaml:"-"`
	Model    string `mapstructure:"model"    toml:"-" yaml:"-"`
}

type ProviderConfig struct {
	APIKey string `mapstructure:"api_key" toml:"api_key" yaml:"api_key"`
	Model  string `mapstructure:"model"   toml:"model"   yaml:"model"`
}

func Load() (*Config, error) {
	v := viper.New()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "aida")

	v.AddConfigPath(configDir)
	v.SetConfigName("config")

	v.SetEnvPrefix("AIDA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	// Bind keys so Viper knows to look for them in environment variables
	_ = v.BindEnv("llm.provider")
	_ = v.BindEnv("llm.api_key")
	_ = v.BindEnv("llm.model")
	_ = v.BindEnv("mode")
	_ = v.BindEnv("shell")
	_ = v.BindEnv("default_provider")

	v.SetDefault("mode", "confirm")
	v.SetDefault("shell", "/bin/sh")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	applyEnvOverrides(&cfg)
	migrateLegacyLLM(&cfg)
	normalizeProviders(&cfg)

	if cfg.DefaultProvider == "" && len(cfg.Providers) > 0 {
		cfg.DefaultProvider = FirstProviderName(cfg.Providers)
	}

	if cfg.DefaultProvider == "" {
		cfg.DefaultProvider = ProviderAIStudio
	}

	return &cfg, nil
}

func ResolveConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "aida")
	tomlPath := filepath.Join(configDir, "config.toml")
	yamlPath := filepath.Join(configDir, "config.yaml")

	if _, err := os.Stat(tomlPath); err == nil {
		return tomlPath, nil
	}

	if _, err := os.Stat(yamlPath); err == nil {
		return yamlPath, nil
	}

	return tomlPath, nil
}

func Save(cfg *Config) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("config is nil")
	}

	path, err := ResolveConfigPath()
	if err != nil {
		return "", err
	}

	if mkdirErr := os.MkdirAll(filepath.Dir(path), DirPerm); mkdirErr != nil {
		return "", fmt.Errorf("create config dir: %w", mkdirErr)
	}

	ext := strings.ToLower(filepath.Ext(path))

	var data []byte

	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(cfg)
	default:
		data, err = toml.Marshal(cfg)
	}

	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, FilePerm); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}

	return path, nil
}

func NormalizeProviderName(input string) string {
	normalized := strings.ToLower(strings.TrimSpace(input))

	switch normalized {
	case ProviderAIStudio, "google", "googleai", "google-ai-studio":
		return ProviderAIStudio
	default:
		return normalized
	}
}

func DefaultModelForProvider(input string) string {
	switch NormalizeProviderName(input) {
	case ProviderAIStudio:
		return "gemini-3-flash"
	default:
		return ""
	}
}

func (c *Config) ActiveProvider() (string, ProviderConfig, error) {
	if c == nil {
		return "", ProviderConfig{}, fmt.Errorf("config is nil")
	}

	name := NormalizeProviderName(c.DefaultProvider)

	if name == "" && len(c.Providers) > 0 {
		name = FirstProviderName(c.Providers)
	}

	if name == "" {
		return "", ProviderConfig{}, fmt.Errorf("no providers configured")
	}

	if provider, ok := c.Providers[name]; ok {
		return name, provider, nil
	}

	return "", ProviderConfig{}, fmt.Errorf("default provider %q not configured", name)
}

func (c *Config) FindProvider(name string) (ProviderConfig, bool) {
	if c == nil {
		return ProviderConfig{}, false
	}

	name = NormalizeProviderName(name)

	if name == "" {
		return ProviderConfig{}, false
	}

	if provider, ok := c.Providers[name]; ok {
		return provider, true
	}

	return ProviderConfig{}, false
}

func (c *Config) UpsertProvider(name string, provider ProviderConfig) string {
	if c == nil {
		return ""
	}

	name = NormalizeProviderName(name)
	if name == "" {
		return ""
	}

	if provider.Model == "" {
		provider.Model = DefaultModelForProvider(name)
	}

	if c.Providers == nil {
		c.Providers = make(map[string]ProviderConfig)
	}

	if existing, ok := c.Providers[name]; ok {
		if provider.APIKey != "" {
			existing.APIKey = provider.APIKey
		}

		if provider.Model != "" {
			existing.Model = provider.Model
		}

		c.Providers[name] = existing

		return name
	}

	c.Providers[name] = provider

	if c.DefaultProvider == "" {
		c.DefaultProvider = name
	}

	return name
}

func RemoveProvider(cfg *Config, name string) bool {
	if cfg == nil {
		return false
	}

	name = NormalizeProviderName(name)
	if name == "" {
		return false
	}

	if _, ok := cfg.Providers[name]; !ok {
		return false
	}

	delete(cfg.Providers, name)

	if cfg.DefaultProvider == name {
		cfg.DefaultProvider = ""

		if len(cfg.Providers) > 0 {
			cfg.DefaultProvider = FirstProviderName(cfg.Providers)
		}
	}

	return true
}

func normalizeProviders(cfg *Config) {
	if cfg == nil {
		return
	}

	if cfg.Providers == nil {
		cfg.Providers = make(map[string]ProviderConfig)

		return
	}

	normalized := make(map[string]ProviderConfig)

	for name, provider := range cfg.Providers {
		normalizedName := NormalizeProviderName(name)
		if normalizedName == "" {
			continue
		}

		normalizeSingleProvider(normalized, normalizedName, provider)
	}

	cfg.Providers = normalized

	if cfg.DefaultProvider != "" {
		cfg.DefaultProvider = NormalizeProviderName(cfg.DefaultProvider)
	}
}

func normalizeSingleProvider(normalized map[string]ProviderConfig, name string, provider ProviderConfig) {
	if provider.Model == "" {
		provider.Model = DefaultModelForProvider(name)
	}

	if existing, ok := normalized[name]; ok {
		switch {
		case existing.APIKey == "" && provider.APIKey != "":
			existing.APIKey = provider.APIKey
		case existing.Model == "" && provider.Model != "":
			existing.Model = provider.Model
		}

		normalized[name] = existing
	} else {
		normalized[name] = provider
	}
}

func migrateLegacyLLM(cfg *Config) {
	if cfg == nil {
		return
	}

	if len(cfg.Providers) > 0 {
		return
	}

	if cfg.LLM.Provider == "" && cfg.LLM.APIKey == "" && cfg.LLM.Model == "" {
		return
	}

	name := NormalizeProviderName(cfg.LLM.Provider)
	if name == "" {
		name = ProviderAIStudio
	}

	if cfg.Providers == nil {
		cfg.Providers = make(map[string]ProviderConfig)
	}

	cfg.Providers[name] = ProviderConfig{
		APIKey: cfg.LLM.APIKey,
		Model:  cfg.LLM.Model,
	}

	if cfg.DefaultProvider == "" {
		cfg.DefaultProvider = name
	}
}

func applyEnvOverrides(cfg *Config) {
	if cfg == nil {
		return
	}

	// 1. Specific provider overrides (AIDA_PROVIDER_<NAME>_API_KEY / AIDA_PROVIDER_<NAME>_MODEL)
	applySpecificProviderEnvOverrides(cfg)

	// 2. Default provider override
	if cfg.DefaultProvider == "" {
		// Viper should have already handled AIDA_DEFAULT_PROVIDER via BindEnv
		// But if it's still empty, we fallback to our logic.
	}

	// 3. Generic active provider overrides (AIDA_LLM_API_KEY / AIDA_LLM_MODEL)
	envProvider := NormalizeProviderName(cfg.LLM.Provider)
	if envProvider != "" {
		cfg.DefaultProvider = envProvider
	}

	apiKey := cfg.LLM.APIKey
	model := cfg.LLM.Model

	if envProvider == "" && (apiKey != "" || model != "") {
		if cfg.DefaultProvider != "" {
			envProvider = cfg.DefaultProvider
		} else if len(cfg.Providers) > 0 {
			envProvider = FirstProviderName(cfg.Providers)
		} else {
			envProvider = ProviderAIStudio
			cfg.DefaultProvider = envProvider
		}
	}

	if envProvider != "" {
		cfg.UpsertProvider(envProvider, ProviderConfig{
			APIKey: apiKey,
			Model:  model,
		})
	}

	cfg.DefaultProvider = NormalizeProviderName(cfg.DefaultProvider)
}

func applySpecificProviderEnvOverrides(cfg *Config) {
	prefix := "AIDA_PROVIDER_"

	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, prefix) {
			continue
		}

		parts := strings.SplitN(env, "=", envSplitCount)
		if len(parts) != envSplitCount {
			continue
		}

		key := parts[0]
		value := parts[1]
		remaining := strings.TrimPrefix(key, prefix)

		switch {
		case strings.HasSuffix(remaining, "_API_KEY"):
			name := strings.TrimSuffix(remaining, "_API_KEY")
			cfg.UpsertProvider(NormalizeProviderName(name), ProviderConfig{APIKey: value})
		case strings.HasSuffix(remaining, "_MODEL"):
			name := strings.TrimSuffix(remaining, "_MODEL")
			cfg.UpsertProvider(NormalizeProviderName(name), ProviderConfig{Model: value})
		}
	}
}

func FirstProviderName(providers map[string]ProviderConfig) string {
	if len(providers) == 0 {
		return ""
	}

	keys := make([]string, 0, len(providers))

	for key := range providers {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys[0]
}