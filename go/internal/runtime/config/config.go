package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/internal/agent"
	"github.com/rexleimo/agno-go/internal/model"
	"gopkg.in/yaml.v3"
)

// Config represents the runtime configuration loaded from YAML and environment variables.
type Config struct {
	Server    ServerConfig       `yaml:"server"`
	Logging   LoggingConfig      `yaml:"logging"`
	Auth      AuthConfig         `yaml:"auth"`
	Providers ProvidersConfig    `yaml:"providers"`
	Memory    agent.MemoryConfig `yaml:"memory"`
	Bench     BenchConfig        `yaml:"bench"`
	Runtime   RuntimeConfig      `yaml:"runtime"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

type AuthConfig struct {
	APIKey string `yaml:"apiKey"`
	Header string `yaml:"header"`
}

type BenchConfig struct {
	Concurrency int           `yaml:"concurrency"`
	InputTokens int           `yaml:"input_tokens"`
	Duration    time.Duration `yaml:"duration"`
}

type RuntimeConfig struct {
	MaxConcurrentRequests int           `yaml:"maxConcurrentRequests"`
	RequestQueue          int           `yaml:"requestQueue"`
	RequestTimeout        time.Duration `yaml:"requestTimeout"`
	Router                RouterConfig  `yaml:"router"`
	GOMEMLIMIT            string        `yaml:"gomemlimit"`
	GOGC                  int           `yaml:"gogc"`
}

type RouterConfig struct {
	MaxProviderConcurrency int           `yaml:"maxProviderConcurrency"`
	ProviderTimeout        time.Duration `yaml:"providerTimeout"`
	ProviderRetries        int           `yaml:"providerRetries"`
	ProviderBackoff        time.Duration `yaml:"providerBackoff"`
}

type ProviderConfig struct {
	Endpoint   string             `yaml:"endpoint"`
	APIKey     string             `yaml:"-"`
	Status     model.Availability `yaml:"-"`
	MissingEnv []string           `yaml:"-"`
	Reason     string             `yaml:"-"`
}

type ProvidersConfig struct {
	OpenAI      ProviderConfig `yaml:"openai"`
	Gemini      ProviderConfig `yaml:"gemini"`
	GLM4        ProviderConfig `yaml:"glm4"`
	OpenRouter  ProviderConfig `yaml:"openrouter"`
	SiliconFlow ProviderConfig `yaml:"siliconflow"`
	Cerebras    ProviderConfig `yaml:"cerebras"`
	ModelScope  ProviderConfig `yaml:"modelscope"`
	Groq        ProviderConfig `yaml:"groq"`
	Ollama      ProviderConfig `yaml:"ollama"`
}

// Load reads configuration from YAML and an optional .env file.
func Load(configPath string) (*Config, error) {
	return LoadWithEnv(configPath, ".env")
}

// LoadWithEnv reads YAML config and merges env variables from the provided .env file.
func LoadWithEnv(configPath, envPath string) (*Config, error) {
	if envPath != "" {
		if err := loadEnvFile(envPath); err != nil {
			return nil, fmt.Errorf("load env: %w", err)
		}
	}
	cfgBytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(cfgBytes, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyAuthDefaults()
	cfg.applyProviderEnv()
	cfg.applyRuntimeEnv()
	return &cfg, nil
}

// ProviderStatuses exposes provider readiness for health checks.
func (c *Config) ProviderStatuses() []model.ProviderStatus {
	all := c.Providers.asMap()
	statuses := make([]model.ProviderStatus, 0, len(all))
	for provider, cfg := range all {
		statuses = append(statuses, model.ProviderStatus{
			Provider:     provider,
			Status:       cfg.Status,
			Capabilities: providerCapabilities(provider),
			MissingEnv:   cfg.MissingEnv,
			Reason:       cfg.Reason,
		})
	}
	return statuses
}

// ProviderConfigs returns provider configs keyed by provider enum.
func (c *Config) ProviderConfigs() map[agent.Provider]ProviderConfig {
	out := make(map[agent.Provider]ProviderConfig)
	for k, v := range c.Providers.asMap() {
		out[k] = *v
	}
	return out
}

func (c *Config) applyProviderEnv() {
	envKeys := requiredEnv()
	providers := c.Providers.asMap()
	for name, cfg := range providers {
		cfg.Endpoint = os.ExpandEnv(cfg.Endpoint)
		required := envKeys[name]
		missing := missingEnv(required)
		cfg.APIKey = firstNonEmpty(getEnv(required))

		// Ollama is endpoint-driven; treat empty endpoint as not-configured.
		if name == agent.ProviderOllama && strings.TrimSpace(cfg.Endpoint) == "" {
			missing = append(missing, "OLLAMA_ENDPOINT")
		}
		cfg.MissingEnv = missing
		if len(missing) > 0 {
			cfg.Status = model.ProviderNotConfigured
			cfg.Reason = fmt.Sprintf("missing env: %s", strings.Join(missing, ","))
		} else {
			cfg.Status = model.ProviderAvailable
			cfg.Reason = ""
		}
	}
}

func requiredEnv() map[agent.Provider][]string {
	return map[agent.Provider][]string{
		agent.ProviderOpenAI:      {"OPENAI_API_KEY"},
		agent.ProviderGemini:      {"GEMINI_API_KEY"},
		agent.ProviderGLM4:        {"GLM4_API_KEY"},
		agent.ProviderOpenRouter:  {"OPENROUTER_API_KEY"},
		agent.ProviderSiliconFlow: {"SILICONFLOW_API_KEY"},
		agent.ProviderCerebras:    {"CEREBRAS_API_KEY"},
		agent.ProviderModelScope:  {"MODELSCOPE_API_KEY"},
		agent.ProviderGroq:        {"GROQ_API_KEY"},
		agent.ProviderOllama:      {},
	}
}

func missingEnv(keys []string) []string {
	var missing []string
	for _, k := range keys {
		if strings.TrimSpace(os.Getenv(k)) == "" {
			missing = append(missing, k)
		}
	}
	return missing
}

func getEnv(keys []string) []string {
	values := make([]string, 0, len(keys))
	for _, k := range keys {
		values = append(values, os.Getenv(k))
	}
	return values
}

func (c *Config) applyAuthDefaults() {
	c.Auth.APIKey = strings.TrimSpace(firstNonEmpty([]string{c.Auth.APIKey, os.Getenv("AGNO_API_KEY")}))
	c.Auth.Header = strings.TrimSpace(c.Auth.Header)
	if c.Auth.Header == "" {
		c.Auth.Header = "X-API-Key"
	}
	if c.Auth.APIKey == "" {
		c.Auth.APIKey = "dev-local-key"
	}
}

func firstNonEmpty(values []string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func (c *Config) applyRuntimeEnv() {
	if limit := strings.TrimSpace(c.Runtime.GOMEMLIMIT); limit != "" {
		_ = os.Setenv("GOMEMLIMIT", limit)
	}
	if c.Runtime.GOGC > 0 {
		_ = os.Setenv("GOGC", strconv.Itoa(c.Runtime.GOGC))
	}
}

// loadEnvFile sets env vars from a .env file without overriding existing values.
func loadEnvFile(envPath string) error {
	abs := envPath
	if !filepath.IsAbs(envPath) {
		if wd, err := os.Getwd(); err == nil {
			abs = filepath.Join(wd, envPath)
		}
	}
	data, err := os.ReadFile(abs)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
	return nil
}

func (p *ProvidersConfig) asMap() map[agent.Provider]*ProviderConfig {
	return map[agent.Provider]*ProviderConfig{
		agent.ProviderOpenAI:      &p.OpenAI,
		agent.ProviderGemini:      &p.Gemini,
		agent.ProviderGLM4:        &p.GLM4,
		agent.ProviderOpenRouter:  &p.OpenRouter,
		agent.ProviderSiliconFlow: &p.SiliconFlow,
		agent.ProviderCerebras:    &p.Cerebras,
		agent.ProviderModelScope:  &p.ModelScope,
		agent.ProviderGroq:        &p.Groq,
		agent.ProviderOllama:      &p.Ollama,
	}
}

func providerCapabilities(p agent.Provider) []model.Capability {
	// All providers expose chat + embedding + streaming surfaces; specific
	// adapters will gate unavailable capabilities at call time.
	return []model.Capability{
		model.CapabilityChat,
		model.CapabilityEmbedding,
		model.CapabilityStreaming,
	}
}
