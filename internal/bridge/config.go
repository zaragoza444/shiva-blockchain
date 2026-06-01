package bridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/onex-blockchain/onex/internal/legacy"
)

type Config struct {
	NodeURL     string `json:"nodeUrl"`
	Listen      string `json:"listen"`
	WalletPath  string `json:"walletPath"`
	ProjectRoot string `json:"projectRoot"`
}

func DefaultConfig() Config {
	return Config{
		NodeURL:    "http://127.0.0.1:8545",
		Listen:     ":9338",
		WalletPath: filepath.Join(legacy.HomeDir(), "wallets", "default.json"),
	}
}

func ConfigPath() string {
	return filepath.Join(legacy.HomeDir(), "bridge.json")
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		path = ConfigPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			_ = SaveConfig(path, cfg)
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.NodeURL == "" {
		cfg.NodeURL = DefaultConfig().NodeURL
	}
	if cfg.Listen == "" {
		cfg.Listen = DefaultConfig().Listen
	}
	if cfg.WalletPath == "" {
		cfg.WalletPath = DefaultConfig().WalletPath
	}
	cfg.WalletPath = legacy.RewriteText(cfg.WalletPath)
	applyEnvOverrides(&cfg)
	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := legacy.EnvOrLegacy("ONEX_NODE_URL", "SHIVA_NODE_URL"); v != "" {
		cfg.NodeURL = v
	}
	if v := legacy.EnvOrLegacy("ONEX_BRIDGE_LISTEN", "SHIVA_BRIDGE_LISTEN"); v != "" {
		cfg.Listen = v
	}
	if v := legacy.EnvOrLegacy("ONEX_WALLET_PATH", "SHIVA_WALLET_PATH"); v != "" {
		cfg.WalletPath = legacy.RewriteText(v)
	}
	if v := legacy.EnvOrLegacy("ONEX_PROJECT_ROOT", "SHIVA_PROJECT_ROOT"); v != "" {
		cfg.ProjectRoot = v
	}
	if p := strings.TrimSpace(os.Getenv("PORT")); p != "" && legacy.EnvOrLegacy("ONEX_BRIDGE_LISTEN", "SHIVA_BRIDGE_LISTEN") == "" {
		cfg.Listen = ":" + strings.TrimPrefix(p, ":")
	}
	cfg.NodeURL = normalizeURL(cfg.NodeURL)
}

func normalizeURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return u
	}
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u
	}
	return "https://" + u
}

func SaveConfig(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
