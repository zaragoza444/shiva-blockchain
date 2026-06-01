package ai

import (
	"strings"

	"github.com/onex-blockchain/onex/internal/legacy"
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
	key := legacy.EnvOrLegacy("ONEX_AI_API_KEY", "SHIVA_AI_API_KEY")
	base := legacy.EnvOrLegacy("ONEX_AI_BASE_URL", "SHIVA_AI_BASE_URL")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	model := legacy.EnvOrLegacy("ONEX_AI_MODEL", "SHIVA_AI_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}
	enabled := strings.EqualFold(legacy.EnvOrLegacy("ONEX_AI_ENABLED", "SHIVA_AI_ENABLED"), "true") || key != ""
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
