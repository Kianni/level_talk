package config

import (
	"errors"
	"os"
)

// Config holds runtime configuration.
type Config struct {
	Port             string
	DBDSN            string
	LLMAPIKey        string
	LLMModel         string
	ElevenLabsAPIKey string
	ElevenLabsVoice  string
}

// Load parses environment variables into Config and validates required values.
func Load() (Config, error) {
	cfg := Config{
		Port:             getEnv("PORT", "8080"),
		DBDSN:            os.Getenv("DB_DSN"),
		LLMAPIKey:        os.Getenv("LLM_API_KEY"),
		LLMModel:         os.Getenv("LLM_MODEL"),
		ElevenLabsAPIKey: os.Getenv("ELEVENLABS_API_KEY"),
		ElevenLabsVoice:  os.Getenv("ELEVENLABS_VOICE_ID"),
	}

	if cfg.DBDSN == "" {
		return Config{}, errors.New("DB_DSN is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
