package ai

import (
	"os"
	"strings"
)

// Config from environment (OpenAI-compatible APIs).
type Config struct {
	Enabled  bool
	APIKey   string
	BaseURL  string
	Model    string
	MaxTok   int
}

func LoadConfig() Config {
	key := strings.TrimSpace(os.Getenv("SHIVA_AI_API_KEY"))
	base := strings.TrimSpace(os.Getenv("SHIVA_AI_BASE_URL"))
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	model := strings.TrimSpace(os.Getenv("SHIVA_AI_MODEL"))
	if model == "" {
		model = "gpt-4o-mini"
	}
	enabled := strings.EqualFold(os.Getenv("SHIVA_AI_ENABLED"), "true") || key != ""
	return Config{
		Enabled: enabled,
		APIKey:  key,
		BaseURL: strings.TrimRight(base, "/"),
		Model:   model,
		MaxTok:  1024,
	}
}

func (c Config) CloudAvailable() bool {
	return c.Enabled && c.APIKey != ""
}
