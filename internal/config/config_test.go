package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg := Load()

	if cfg.Engine.Workers != DefaultWorkers {
		t.Errorf("Workers = %d, want %d", cfg.Engine.Workers, DefaultWorkers)
	}
	if cfg.Engine.Capacity != DefaultCapacity {
		t.Errorf("Capacity = %d, want %d", cfg.Engine.Capacity, DefaultCapacity)
	}
	if cfg.Engine.MaxRetries != DefaultMaxRetries {
		t.Errorf("MaxRetries = %d, want %d", cfg.Engine.MaxRetries, DefaultMaxRetries)
	}
	if cfg.Logger.Level != DefaultLogLevel {
		t.Errorf("Logger.Level = %q, want %q", cfg.Logger.Level, DefaultLogLevel)
	}
}

func TestLoadReadsDocumentedEnv(t *testing.T) {
	t.Setenv("AI_ENGINE_WORKERS", "4")
	t.Setenv("AI_ENGINE_CAPACITY", "1000")
	t.Setenv("AI_ENGINE_MAX_RETRIES", "5")
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("LOG_PATH", "/tmp/engine-test.log")
	t.Setenv("PROVIDER_ORDER", "openai, anthropic")
	t.Setenv("GEMINI_API_KEYS", "a, b ,, c")

	cfg := Load()

	if cfg.Engine.Workers != 4 || cfg.Engine.Capacity != 1000 || cfg.Engine.MaxRetries != 5 {
		t.Errorf("engine config not read: %+v", cfg.Engine)
	}
	if cfg.Logger.Level != "warn" || cfg.Logger.Path != "/tmp/engine-test.log" {
		t.Errorf("logger config not read: %+v", cfg.Logger)
	}
	if len(cfg.Providers.Order) != 2 || cfg.Providers.Order[0] != "openai" {
		t.Errorf("PROVIDER_ORDER = %v", cfg.Providers.Order)
	}
	if len(cfg.Providers.GeminiAPIKeys) != 3 {
		t.Errorf("GEMINI_API_KEYS = %v, want 3 trimmed keys", cfg.Providers.GeminiAPIKeys)
	}
}

func TestLoadRejectsBadNumbers(t *testing.T) {
	for _, val := range []string{"nope", "0", "-3", ""} {
		t.Setenv("AI_ENGINE_WORKERS", val)

		if got := Load().Engine.Workers; got != DefaultWorkers {
			t.Errorf("AI_ENGINE_WORKERS=%q gave Workers = %d, want %d", val, got, DefaultWorkers)
		}
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"no providers at all", Config{}, true},
		{"a cloud key is enough", Config{Providers: ProvidersConfig{OpenAIAPIKeys: []string{"k"}}}, false},
		{"ollama needs both", Config{Providers: ProvidersConfig{OllamaModel: "llama3"}}, true},
		{"ollama model and url", Config{Providers: ProvidersConfig{OllamaModel: "llama3", OllamaURL: "http://x"}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.Validate(); (got != nil) != tt.wantErr {
				t.Errorf("Validate() = %v, wantErr %v", got, tt.wantErr)
			}
		})
	}
}
