package config

import (
	"os"
	"strconv"
	"strings"
	"sync"

	"nvim-engine/internal/provider/p_error"
)

const (
	DefaultWorkers    = 10
	DefaultCapacity   = 20
	DefaultMaxRetries = 2
	DefaultLogLevel   = "debug"
)

type EngineConfig struct {
	Workers    int
	Capacity   int
	MaxRetries int
}

type ProvidersConfig struct {
	Order []string

	GeminiAPIKeys []string
	GeminiModel   string
	GeminiURL     string

	AnthropicAPIKeys []string
	AnthropicModel   string
	AnthropicURL     string

	OpenAIAPIKeys []string
	OpenAIModel   string
	OpenAIURL     string

	OllamaModel string
	OllamaURL   string
}

type LoggerConfig struct {
	Level string
	Path  string
}

type Config struct {
	Engine    EngineConfig
	Providers ProvidersConfig
	Logger    LoggerConfig
}

var (
	instance *Config
	once     sync.Once
)

func Get() *Config {
	once.Do(func() { instance = Load() })

	return instance
}

func Load() *Config {
	return &Config{
		Providers: ProvidersConfig{
			Order: getEnvAsSlice("PROVIDER_ORDER"),

			GeminiAPIKeys: getEnvAsSlice("GEMINI_API_KEYS"),
			GeminiModel:   getEnvOrDefault("GEMINI_MODEL", "gemini-3.8-flash"),
			GeminiURL:     getEnvOrDefault("GEMINI_URL", "https://generativelanguage.googleapis.com/v1beta/models"),

			AnthropicAPIKeys: getEnvAsSlice("ANTHROPIC_API_KEYS"),
			AnthropicModel:   getEnvOrDefault("ANTHROPIC_MODEL", "claude-sonnet-5-5"),
			AnthropicURL:     getEnvOrDefault("ANTHROPIC_URL", "https://api.anthropic.com/v1/messages"),

			OpenAIAPIKeys: getEnvAsSlice("OPENAI_API_KEYS"),
			OpenAIModel:   getEnvOrDefault("OPENAI_MODEL", "gpt-6-astra"),
			OpenAIURL:     getEnvOrDefault("OPENAI_URL", "https://api.openai.com/v1/chat/completions"),

			OllamaModel: getEnvOrDefault("OLLAMA_MODEL", ""),
			OllamaURL:   getEnvOrDefault("OLLAMA_URL", ""),
		},
		Engine: EngineConfig{
			Workers:    getEnvAsInt("AI_ENGINE_WORKERS", DefaultWorkers),
			Capacity:   getEnvAsInt("AI_ENGINE_CAPACITY", DefaultCapacity),
			MaxRetries: max(1, getEnvAsInt("AI_ENGINE_MAX_RETRIES", DefaultMaxRetries)),
		},
		Logger: LoggerConfig{
			Level: getEnvOrDefault("LOG_LEVEL", DefaultLogLevel),
			Path:  getEnvOrDefault("LOG_PATH", ""),
		},
	}
}

func (c *Config) Validate() *p_error.ProviderError {
	hasCloudKey := len(c.Providers.GeminiAPIKeys) > 0 || len(c.Providers.AnthropicAPIKeys) > 0 || len(c.Providers.OpenAIAPIKeys) > 0
	hasOllama := c.Providers.OllamaModel != "" && c.Providers.OllamaURL != ""

	if !hasCloudKey && !hasOllama {
		return &p_error.ProviderError{
			Code:     p_error.ErrConfig,
			Provider: "Configuration",
			Message:  p_error.FriendlyMessages[p_error.ErrConfig],
		}
	}

	return nil
}

func getEnvAsSlice(key string) []string {
	val := os.Getenv(key)
	if val == "" {
		return nil
	}

	var keys []string
	for _, k := range strings.Split(val, ",") {
		if k = strings.TrimSpace(k); k != "" {
			keys = append(keys, k)
		}
	}

	return keys
}

func getEnvOrDefault(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}

	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	n, err := strconv.Atoi(strings.TrimSpace(val))
	if err != nil || n <= 0 {
		return fallback
	}

	return n
}
