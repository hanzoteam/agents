// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/mattermost/mattermost-plugin-agents/v2/embeddings"
	"github.com/mattermost/mattermost-plugin-agents/v2/llm"
)

const (
	tokenUsageLogToPluginEnvKey = "MM_FEATUREFLAGS_AI_TOKEN_USAGE_LOG_TO_PLUGIN" // #nosec G101 -- env var key name, not a credential
	tokenUsageLogToFileEnvKey   = "MM_FEATUREFLAGS_AI_TOKEN_USAGE_LOG_TO_FILE"   // #nosec G101 -- env var key name, not a credential
)

type Config struct {
	Services                        []llm.ServiceConfig              `json:"services"`
	Bots                            []llm.BotConfig                  `json:"bots"`
	DefaultBotName                  string                           `json:"defaultBotName"`
	TranscriptGenerator             string                           `json:"transcriptBackend"`
	EnableTokenUsageLogging         bool                             `json:"enableTokenUsageLogging"`
	EnableCallSummary               bool                             `json:"enableCallSummary"`
	EnableTokenUsageLogToPlugin     *bool                            `json:"enableTokenUsageLogToPlugin,omitempty"`
	EnableTokenUsageLogToFile       *bool                            `json:"enableTokenUsageLogToFile,omitempty"`
	AllowedUpstreamHostnames        string                           `json:"allowedUpstreamHostnames"`
	AllowUnsafeLinks                bool                             `json:"allowUnsafeLinks"`
	EnableChannelMentionToolCalling bool                             `json:"enableChannelMentionToolCalling"`
	AllowNativeWebSearchInChannels  bool                             `json:"allowNativeWebSearchInChannels"`
	EmbeddingSearchConfig           embeddings.EmbeddingSearchConfig `json:"embeddingSearchConfig"`
	MCP                             MCPConfig                        `json:"mcp"`
	WebSearch                       WebSearchConfig                  `json:"webSearch"`
	TelemetryOutput                 string                           `json:"telemetryOutput"`
	OpenTelemetryEndpoint           string                           `json:"openTelemetryEndpoint"`
}

type WebSearchConfig struct {
	Enabled        bool                  `json:"enabled"`
	Provider       string                `json:"provider"`
	Google         WebSearchGoogleConfig `json:"google"`
	Brave          WebSearchBraveConfig  `json:"brave"`
	DomainDenylist []string              `json:"domainDenylist"`
}

type WebSearchGoogleConfig struct {
	APIKey         string `json:"apiKey"`
	SearchEngineID string `json:"searchEngineId"`
	ResultLimit    int    `json:"resultLimit"`
	APIURL         string `json:"apiURL"`
}

type WebSearchBraveConfig struct {
	APIKey       string `json:"apiKey"`
	APIURL       string `json:"apiURL"`
	ResultLimit  int    `json:"resultLimit"`
	PollTimeout  int    `json:"pollTimeout"`
	PollInterval int    `json:"pollInterval"`
}

func (c *Config) Clone() *Config {
	clone, err := DeepCopyJSON(*c)
	if err != nil {
		panic(fmt.Sprintf("failed to clone configuration: %v", err))
	}

	return &clone
}

// GetServiceByID returns the service configuration for the given ID
func (c *Config) GetServiceByID(id string) (llm.ServiceConfig, bool) {
	for i := range c.Services {
		if c.Services[i].ID == id {
			return c.Services[i], true
		}
	}
	return llm.ServiceConfig{}, false
}

type UpdateListener func()

type Container struct {
	cfg       atomic.Pointer[Config]
	listeners []UpdateListener
}

// Config retruns the whole configuration readonly.
// Avoid using this method, prefer using config though interfaces.
func (c *Container) Config() *Config {
	return c.cfg.Load()
}

func (c *Container) GetTranscriptGenerator() string {
	return c.cfg.Load().TranscriptGenerator
}

func (c *Container) GetBots() []llm.BotConfig {
	return c.cfg.Load().Bots
}

func (c *Container) GetDefaultBotName() string {
	return c.cfg.Load().DefaultBotName
}

func (c *Container) EnableTokenUsageLogging() bool {
	return c.cfg.Load().EnableTokenUsageLogging
}

func (c *Container) EnableTokenUsageLogToPlugin() bool {
	cfg := c.cfg.Load()
	if cfg == nil || !cfg.EnableTokenUsageLogging {
		return false
	}

	if enabled, ok := parseBooleanEnv(tokenUsageLogToPluginEnvKey); ok {
		return enabled
	}

	return false
}

func (c *Container) EnableTokenUsageLogToFile() bool {
	cfg := c.cfg.Load()
	if cfg == nil || !cfg.EnableTokenUsageLogging {
		return false
	}

	if enabled, ok := parseBooleanEnv(tokenUsageLogToFileEnvKey); ok {
		return enabled
	}

	return true
}

func parseBooleanEnv(key string) (bool, bool) {
	raw, ok := os.LookupEnv(key)
	if !ok {
		return false, false
	}

	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return false, false
	}

	return parsed, true
}

func (c *Container) MCP() MCPConfig {
	return c.cfg.Load().MCP
}

func (c *Container) AllowUnsafeLinks() bool {
	cfg := c.cfg.Load()
	if cfg == nil {
		return false
	}

	return cfg.AllowUnsafeLinks
}

func (c *Container) EnableChannelMentionToolCalling() bool {
	cfg := c.cfg.Load()
	if cfg == nil {
		return false
	}

	return cfg.EnableChannelMentionToolCalling
}

func (c *Container) AllowNativeWebSearchInChannels() bool {
	cfg := c.cfg.Load()
	if cfg == nil {
		return false
	}

	return cfg.AllowNativeWebSearchInChannels
}

func (c *Container) RegisterUpdateListener(listener UpdateListener) {
	c.listeners = append(c.listeners, listener)
}

func (c *Container) EmbeddingSearchConfig() embeddings.EmbeddingSearchConfig {
	return c.cfg.Load().EmbeddingSearchConfig
}

// GetServiceByID returns the service configuration for the given ID
func (c *Container) GetServiceByID(id string) (llm.ServiceConfig, bool) {
	cfg := c.cfg.Load()
	if cfg == nil {
		return llm.ServiceConfig{}, false
	}
	return cfg.GetServiceByID(id)
}

// serviceKeyEnv names the variable a service takes its API key from when the
// stored configuration leaves it empty: AGENTS_KEY_ followed by the service id.
// Anything not a letter or digit becomes an underscore, because a service id is
// free-form and a variable name is not.
func serviceKeyEnv(id string) string {
	var b strings.Builder
	b.WriteString("AGENTS_KEY_")
	for _, r := range strings.ToUpper(id) {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

// supplyServiceKeys fills in any API key the stored configuration leaves empty
// from the environment, so a credential can arrive from KMS as an environment
// variable and never be written down.
//
// It runs on the copy this process holds, not on what is stored. The admin API
// reads the database directly, so it keeps returning the empty string it was
// given, and an admin saving the settings page writes that empty string back
// rather than persisting the key into the config table.
func supplyServiceKeys(cfg *Config) {
	for i := range cfg.Services {
		if cfg.Services[i].APIKey == "" {
			cfg.Services[i].APIKey = os.Getenv(serviceKeyEnv(cfg.Services[i].ID))
		}
	}
}

// Updates the current configuration
// The new configuration is deep-copied to ensure the new and old
// configurations are independent of each other.
func (c *Container) Update(newConfig *Config) {
	if newConfig == nil {
		c.cfg.Store(nil)
		return
	}
	// Create a deep copy of the new configuration
	clone, err := DeepCopyJSON(*newConfig)
	if err != nil {
		panic(fmt.Sprintf("failed to deep copy configuration: %v", err))
	}

	supplyServiceKeys(&clone)

	// Update the atomic pointer with the new configuration
	c.cfg.Store(&clone)

	// Notify all listeners about the configuration change
	for _, listener := range c.listeners {
		listener()
	}
}

// StorePersistedConfigWithoutNotify updates in-memory configuration from a value read back from
// persistent storage without notifying update listeners. Use when the current call stack may
// already be servicing a listener (for example after SaveConfig during legacy migration) to
// avoid re-entrant listener invocation and deadlocks.
func (c *Container) StorePersistedConfigWithoutNotify(newConfig *Config) error {
	if newConfig == nil {
		c.cfg.Store(nil)
		return nil
	}
	clone, err := DeepCopyJSON(*newConfig)
	if err != nil {
		return fmt.Errorf("failed to deep copy configuration: %w", err)
	}
	supplyServiceKeys(&clone)
	c.cfg.Store(&clone)
	return nil
}

// DeepCopyJSON creates a deep copy of JSON-serializable structs
func DeepCopyJSON[T any](src T) (T, error) {
	var dst T
	data, err := json.Marshal(src)
	if err != nil {
		return dst, err
	}
	err = json.Unmarshal(data, &dst)
	return dst, err
}
