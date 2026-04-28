package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/normahq/runtime/agentconfig"
	runtimeconfig "github.com/normahq/runtime/appconfig"
)

const (
	DirPerm  = 0o700
	FilePerm = 0o600

	ProviderAIStudio = "aistudio"
	ProviderOpenAI   = "openai"
	EnvAIDAProvider  = "AIDA_PROVIDER"

	defaultProfileName = "default"
)

//go:embed defaults.yaml
var defaultsYAML []byte

func DefaultModelForProvider(providerType string) string {
	switch NormalizeProviderName(providerType) {
	case ProviderAIStudio:
		return "gemini-2.5-flash"
	case ProviderOpenAI:
		return "gpt-4o-mini"
	default:
		return ""
	}
}

func NormalizeProviderName(input string) string {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case ProviderAIStudio, "google", "googleai", "google-ai-studio":
		return ProviderAIStudio
	case ProviderOpenAI, "open-ai":
		return ProviderOpenAI
	default:
		return ""
	}
}

type AidaConfig struct {
	Provider string `mapstructure:"provider" yaml:"provider"`
	Mode     string `mapstructure:"mode"     yaml:"mode"`
	Shell    string `mapstructure:"shell"    yaml:"shell"`
}

type ProviderConfig struct {
	APIKey string
	Model  string
}

type Config struct {
	Runtime runtimeconfig.RuntimeConfig `mapstructure:"runtime" yaml:"runtime"`
	Aida    AidaConfig                  `mapstructure:"aida"    yaml:"aida"`
	Profile string                      `mapstructure:"-"       yaml:"-"`
}

func Load() (*Config, error) {
	return LoadProfile(strings.TrimSpace(os.Getenv("AIDA_PROFILE")))
}

func LoadProfile(profile string) (*Config, error) {
	return loadProfile(strings.TrimSpace(profile))
}

func loadProfile(requestedProfile string) (*Config, error) {
	if err := validateUnsupportedEnvOverrides(); err != nil {
		return nil, err
	}

	path, err := ResolveConfigPath()
	if err != nil {
		return nil, err
	}

	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		cfg := defaultConfig()
		cfg.Profile = selectedProfileName(requestedProfile)

		return cfg, nil
	} else if statErr != nil {
		return nil, fmt.Errorf("stat config: %w", statErr)
	}

	settings, selectedProfile, err := runtimeconfig.LoadResolvedSettings(
		runtimeconfig.RuntimeLoadOptions{
			ConfigDir: filepath.Dir(filepath.Dir(path)),
			Profile:   requestedProfile,
		},
		runtimeconfig.AppLoadOptions{
			AppName:            "aida",
			DefaultsYAML:       defaultsYAML,
			UseDotConfigAppDir: true,
		},
	)
	if err != nil {
		return nil, err
	}

	if err := validateRequestedProfile(settings, requestedProfile); err != nil {
		return nil, err
	}

	if err := detectMalformedInitConfig(settings, path); err != nil {
		return nil, err
	}

	cfg := defaultConfig()
	if err := runtimeconfig.DecodeSettings(settings, cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	cfg.Profile = selectedProfile

	return cfg, nil
}

func ResolveConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(homeDir, ".config", "aida", "config.yaml"), nil
}

func EnsureConfigDir() (string, error) {
	path, err := ResolveConfigPath()
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, DirPerm); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}

	return dir, nil
}

func defaultConfig() *Config {
	return &Config{
		Aida: AidaConfig{
			Mode:  "confirm",
			Shell: "/bin/sh",
		},
	}
}

func selectedProfileName(requested string) string {
	if strings.TrimSpace(requested) != "" {
		return strings.TrimSpace(requested)
	}

	return defaultProfileName
}

func validateRequestedProfile(settings map[string]any, requestedProfile string) error {
	selected := strings.TrimSpace(requestedProfile)
	if selected == "" {
		return nil
	}

	rawProfiles, ok := settings["profiles"]
	if !ok || rawProfiles == nil {
		return fmt.Errorf("top-level profile %q not found", selected)
	}

	return nil
}

func extractMap(root map[string]any, key string) (map[string]any, bool) {
	raw, ok := root[key]
	if !ok {
		return nil, false
	}

	m, ok := toStringAnyMap(raw)

	return m, ok
}

func toStringAnyMap(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	case map[any]any:
		out := make(map[string]any, len(typed))
		for k, v := range typed {
			key, ok := k.(string)
			if !ok {
				return nil, false
			}

			out[key] = v
		}

		return out, true
	default:
		return nil, false
	}
}

var malformedInitProviderKeys = map[string]struct{}{
	"mcpservers":         {},
	"systeminstructions": {},
	"genericacp":         {},
	"geminiacp":          {},
	"codexacp":           {},
	"opencodeacp":        {},
	"copilotacp":         {},
	"claudecodeacp":      {},
	"extraargs":          {},
	"apikey":             {},
	"poolconfig":         {},
}

func detectMalformedInitConfig(settings map[string]any, path string) error {
	runtimeSettings, ok := extractMap(settings, "runtime")
	if !ok {
		return nil
	}

	providers, ok := extractMap(runtimeSettings, "providers")
	if !ok {
		return nil
	}

	for providerID, rawProvider := range providers {
		providerMap, ok := toStringAnyMap(rawProvider)
		if !ok {
			continue
		}

		if key, found := malformedProviderKey(providerMap); found {
			return fmt.Errorf("config file %q was generated by a buggy version of aida init and uses malformed key %q; "+
				"rerun `aida init --force` and reapply any local edits or secrets", path, providerID+"."+key)
		}
	}

	return nil
}

func malformedProviderKey(provider map[string]any) (string, bool) {
	for key, value := range provider {
		if _, ok := malformedInitProviderKeys[key]; ok {
			return key, true
		}

		nested, ok := toStringAnyMap(value)
		if !ok {
			continue
		}

		if nestedKey, found := malformedProviderKey(nested); found {
			return nestedKey, true
		}
	}

	return "", false
}

func validateUnsupportedEnvOverrides() error {
	if value, ok := os.LookupEnv(EnvAIDAProvider); ok && strings.TrimSpace(value) != "" {
		return fmt.Errorf("%s is not supported; use AIDA_PROFILE or config profiles instead", EnvAIDAProvider)
	}

	return nil
}

func (c *Config) ActiveProviderID() (string, error) {
	if c == nil {
		return "", fmt.Errorf("config is nil")
	}

	id := strings.TrimSpace(c.Aida.Provider)
	if id == "" {
		return "", fmt.Errorf("aida.provider is required")
	}

	if _, ok := c.Runtime.Providers[id]; !ok {
		return "", fmt.Errorf("provider %q is not defined in runtime.providers", id)
	}

	return id, nil
}

func (c *Config) ProviderConfig(id string) (agentconfig.Config, error) {
	if c == nil {
		return agentconfig.Config{}, fmt.Errorf("config is nil")
	}

	cfg, ok := c.Runtime.Providers[strings.TrimSpace(id)]
	if !ok {
		return agentconfig.Config{}, fmt.Errorf("provider %q is not defined in runtime.providers", id)
	}

	return cfg, nil
}

func (c *Config) ValidateSelectedRuntime() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}

	providerID, err := c.ActiveProviderID()
	if err != nil {
		return err
	}

	state := runtimeValidationState{
		validatedProviders: map[string]bool{},
		validatedMCP:       map[string]bool{},
	}

	return c.validateProviderScope(providerID, state, false)
}

func (c *Config) SetShell(shell string) {
	if c == nil {
		return
	}

	shell = strings.TrimSpace(shell)
	if shell == "" {
		return
	}

	c.Aida.Shell = shell
}

func (c *Config) SetProviderModel(id, model string) error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}

	model = strings.TrimSpace(model)
	if model == "" {
		return nil
	}

	provider, ok := c.Runtime.Providers[id]
	if !ok {
		return fmt.Errorf("provider %q is not defined in runtime.providers", id)
	}

	if provider.Type == agentconfig.AgentTypePool {
		return fmt.Errorf("--model is not supported for pool provider %q", id)
	}

	if err := setProviderModel(&provider, id, model); err != nil {
		return err
	}

	c.Runtime.Providers[id] = provider

	return nil
}

func setProviderModel(provider *agentconfig.Config, id, model string) error {
	if updated, err := setLocalAPIProviderModel(provider, model); updated || err != nil {
		return err
	}

	if updated, err := setACPProviderModel(provider, id, model); updated || err != nil {
		return err
	}

	return fmt.Errorf("unsupported provider type %q", provider.Type)
}

func setLocalAPIProviderModel(provider *agentconfig.Config, model string) (bool, error) {
	switch provider.Type {
	case agentconfig.AgentTypeOpenAI:
		if provider.OpenAI == nil {
			provider.OpenAI = &agentconfig.LocalAPIConfig{}
		}

		provider.OpenAI.Model = model

		return true, nil
	case agentconfig.AgentTypeAIStudio:
		if provider.AIStudio == nil {
			provider.AIStudio = &agentconfig.LocalAPIConfig{}
		}

		provider.AIStudio.Model = model

		return true, nil
	default:
		return false, nil
	}
}

func setACPProviderModel(provider *agentconfig.Config, id, model string) (bool, error) {
	blockName, block, ok := acpProviderModelBlock(provider)
	if !ok {
		return false, nil
	}

	if block == nil {
		return true, fmt.Errorf("%s block is required for provider %q", blockName, id)
	}

	block.Model = model

	return true, nil
}

func acpProviderModelBlock(provider *agentconfig.Config) (string, *agentconfig.ACPConfig, bool) {
	switch provider.Type {
	case agentconfig.AgentTypeGenericACP:
		return "generic_acp", provider.GenericACP, true
	case agentconfig.AgentTypeGeminiACP:
		return "gemini_acp", provider.GeminiACP, true
	case agentconfig.AgentTypeCodexACP:
		return "codex_acp", provider.CodexACP, true
	case agentconfig.AgentTypeOpenCodeACP:
		return "opencode_acp", provider.OpenCodeACP, true
	case agentconfig.AgentTypeCopilotACP:
		return "copilot_acp", provider.CopilotACP, true
	case agentconfig.AgentTypeClaudeCodeACP:
		return "claude_code_acp", provider.ClaudeCodeACP, true
	default:
		return "", nil, false
	}
}

type runtimeValidationState struct {
	validatedProviders map[string]bool
	validatedMCP       map[string]bool
}

func (c *Config) validateProviderScope(id string, state runtimeValidationState, nestedPoolMember bool) error {
	if state.validatedProviders[id] {
		return nil
	}

	provider, err := c.ProviderConfig(id)
	if err != nil {
		return err
	}

	if nestedPoolMember && agentconfig.IsPoolType(provider.Type) {
		return fmt.Errorf("provider %q: pool cannot contain nested pool %q", c.Aida.Provider, id)
	}

	if err := provider.Validate(); err != nil {
		return fmt.Errorf("provider %q: %w", id, err)
	}

	state.validatedProviders[id] = true

	if err := c.validateProviderMCPServers(id, provider, state); err != nil {
		return err
	}

	if !agentconfig.IsPoolType(provider.Type) {
		return nil
	}

	return c.validatePoolMembers(id, provider, state)
}

func (c *Config) validateProviderMCPServers(
	id string,
	provider agentconfig.Config,
	state runtimeValidationState,
) error {
	for _, serverID := range provider.MCPServers {
		serverID = strings.TrimSpace(serverID)
		if serverID == "" {
			continue
		}

		if state.validatedMCP[serverID] {
			continue
		}

		serverCfg, ok := c.Runtime.MCPServers[serverID]
		if !ok {
			return fmt.Errorf("provider %q: references unknown mcp server %q", id, serverID)
		}

		if err := agentconfig.ValidateMCPServerConfig(serverCfg); err != nil {
			return fmt.Errorf("provider %q: mcp %q: %w", id, serverID, err)
		}

		state.validatedMCP[serverID] = true
	}

	return nil
}

func (c *Config) validatePoolMembers(id string, provider agentconfig.Config, state runtimeValidationState) error {
	if provider.PoolConfig == nil {
		return fmt.Errorf("provider %q: pool block is required", id)
	}

	for _, memberID := range provider.PoolConfig.Members {
		memberID = strings.TrimSpace(memberID)
		if memberID == "" {
			continue
		}

		if memberID == id {
			return fmt.Errorf("provider %q: pool cannot reference itself", id)
		}

		if _, ok := c.Runtime.Providers[memberID]; !ok {
			return fmt.Errorf("provider %q: pool references unknown agent %q", id, memberID)
		}

		if err := c.validateProviderScope(memberID, state, true); err != nil {
			return err
		}
	}

	return nil
}
