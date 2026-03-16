package config

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

type EngineConfig struct {
	Workers  int
	Capacity int
}

type ProvidersConfig struct {
	GeminiAPIKey string
	GeminiModel  string
	GeminiURL    string

	AnthropicAPIKey string
	AnthropicModel  string
	AnthropicURL    string

	OpenAIAPIKey string
	OpenAIModel  string
	OpenAIURL    string
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
			Engine: EngineConfig{
				Workers:  getEnvAsInt("AI_ENGINE_WORKERS", 4),
				Capacity: getEnvAsInt("AI_ENGINE_CAPACITY", 1000),
			},
			Providers: ProvidersConfig{
				GeminiAPIKey: os.Getenv("GEMINI_API_KEY"),
				GeminiModel:  getEnvOrDefault("GEMINI_MODEL", "gemini-2.0-flash"),
				GeminiURL:    getEnvOrDefault("GEMINI_URL", "https://generativelanguage.googleapis.com/v1beta/models"),

				AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
				AnthropicModel:  getEnvOrDefault("ANTHROPIC_MODEL", "claude-3-5-sonnet-20241022"),
				AnthropicURL:    getEnvOrDefault("ANTHROPIC_URL", "https://api.anthropic.com/v1/messages"),

				OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"),
				OpenAIModel:  getEnvOrDefault("OPENAI_MODEL", "gpt-4o"),
				OpenAIURL:    getEnvOrDefault("OPENAI_URL", "https://api.openai.com/v1/chat/completions"),
			},
			Logger: LoggerConfig{
				Level: getEnvOrDefault("LOG_LEVEL", "debug"),
				Path:  getEnvOrDefault("LOG_PATH", filepath.Join(os.TempDir(), "nvim-engine.log")),
			},
		}
	})

	return instance
}

func getEnvOrDefault(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}

	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}

	return fallback
}
