package config

import (
	"errors"
	"os"
	"strings"
	"sync"
)

type EngineConfig struct {
	Workers  int
	Capacity int
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
	once.Do(func() {
		instance = &Config{
			Providers: ProvidersConfig{
				Order: getEnvAsSlice("PROVIDER_ORDER"),

				GeminiAPIKeys: getEnvAsSlice("GEMINI_API_KEYS"),
				GeminiModel:   getEnvOrDefault("GEMINI_MODEL", "gemini-2.0-flash"),
				GeminiURL:     getEnvOrDefault("GEMINI_URL", "https://generativelanguage.googleapis.com/v1beta/models"),

				AnthropicAPIKeys: getEnvAsSlice("ANTHROPIC_API_KEYS"),
				AnthropicModel:   getEnvOrDefault("ANTHROPIC_MODEL", "claude-3-5-sonnet-20241022"),
				AnthropicURL:     getEnvOrDefault("ANTHROPIC_URL", "https://api.anthropic.com/v1/messages"),

				OpenAIAPIKeys: getEnvAsSlice("OPENAI_API_KEYS"),
				OpenAIModel:   getEnvOrDefault("OPENAI_MODEL", "gpt-4o"),
				OpenAIURL:     getEnvOrDefault("OPENAI_URL", "https://api.openai.com/v1/chat/completions"),

				OllamaModel: getEnvOrDefault("OLLAMA_MODEL", ""),
				OllamaURL:   getEnvOrDefault("OLLAMA_URL", ""),
			},
			Engine: EngineConfig{
				Workers:  10,
				Capacity: 20,
			},
		}
	})

	return instance
}

func (c *Config) Validate() error {
	hasCloudKey := len(c.Providers.GeminiAPIKeys) > 0 || len(c.Providers.AnthropicAPIKeys) > 0 || len(c.Providers.OpenAIAPIKeys) > 0
	hasOllama := c.Providers.OllamaModel != "" && c.Providers.OllamaURL != ""

	if !hasCloudKey && !hasOllama {
		return errors.New("no API keys found and Ollama is not configured! AI generation will fail")
	}

	return nil
}

func getEnvAsSlice(key string) []string {
	val := os.Getenv(key)
	if val == "" {
		return nil
	}

	rawKeys := strings.Split(val, ",")
	var keys []string
	for _, k := range rawKeys {
		k = strings.TrimSpace(k)
		if k != "" {
			keys = append(keys, k)
		}
	}
	return keys
}

func getEnvOrDefault(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}

	return fallback
}
