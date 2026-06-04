package tokenplatform

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/onex-blockchain/onex/internal/legacy"
)

// PlatformToken is a token deployed via the OneX Token Platform.
type PlatformToken struct {
	ID              string                 `json:"id"`
	ChainID         string                 `json:"chainId"`
	ChainType       string                 `json:"chainType"`
	Name            string                 `json:"name"`
	Symbol          string                 `json:"symbol"`
	Decimals        int                    `json:"decimals"`
	Supply          string                 `json:"supply"`
	Creator         string                 `json:"creator"`
	CreatedAt       int64                  `json:"createdAt"`
	ContractAddress string                 `json:"contractAddress,omitempty"`
	DeployStatus    string                 `json:"deployStatus"`
	DeployTxHash    string                 `json:"deployTxHash,omitempty"`
	DeployPayload   map[string]interface{} `json:"deployPayload,omitempty"`
	OriginKey       string                 `json:"originKey,omitempty"`
	IsWrapped       bool                   `json:"isWrapped,omitempty"`
	WrappedOn       []WrapRecord           `json:"wrappedOn,omitempty"`
}

// WrapRecord links an origin token to a wrapped deployment on another chain.
type WrapRecord struct {
	ID             string `json:"id"`
	OriginKey      string `json:"originKey"`
	TargetChainID  string `json:"targetChainId"`
	WrappedTokenID string `json:"wrappedTokenId"`
	WrappedSymbol  string `json:"wrappedSymbol"`
	Amount         string `json:"amount"`
	Status         string `json:"status"`
	CreatedAt      int64  `json:"createdAt"`
	BridgeTxHash   string `json:"bridgeTxHash,omitempty"`
}

// PlatformConfig is loaded from configs/token-platform.json.
type PlatformConfig struct {
	Version      int               `json:"version"`
	WrapPrefix   string            `json:"wrapPrefix"`
	HubChainID   string            `json:"hubChainId"`
	DeployFeeONEX string           `json:"deployFeeOnex"`
	ChainNotes   map[string]string `json:"chainNotes"`
}

func DefaultPlatformConfig() PlatformConfig {
	return PlatformConfig{
		Version:      1,
		WrapPrefix:   "w",
		HubChainID:   "onex-mainnet-1",
		DeployFeeONEX: "0.01",
		ChainNotes: map[string]string{
			"onex":    "Native OneX token registry",
			"evm":     "ERC-20 compatible deploy metadata",
			"solana":  "SPL mint metadata",
			"btc":     "Omni-layer style asset reference",
			"tron":    "TRC-20 compatible metadata",
		},
	}
}

type Store struct {
	mu       sync.Mutex
	path     string
	wrapPath string
}

func NewStore() *Store {
	home := legacy.HomeDir()
	return &Store{
		path:     filepath.Join(home, "platform-tokens.json"),
		wrapPath: filepath.Join(home, "platform-wraps.json"),
	}
}

func (s *Store) LoadTokens() ([]PlatformToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return loadJSONFile[PlatformToken](s.path)
}

func (s *Store) SaveTokens(tokens []PlatformToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return saveJSONFile(s.path, tokens)
}

func (s *Store) LoadWraps() ([]WrapRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return loadJSONFile[WrapRecord](s.wrapPath)
}

func (s *Store) SaveWraps(wraps []WrapRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return saveJSONFile(s.wrapPath, wraps)
}

func loadJSONFile[T any](path string) ([]T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []T
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func saveJSONFile(path string, v interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func LoadConfig(root string) PlatformConfig {
	cfg := DefaultPlatformConfig()
	if root == "" {
		return cfg
	}
	data, err := os.ReadFile(filepath.Join(root, "configs", "token-platform.json"))
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, &cfg)
	if cfg.WrapPrefix == "" {
		cfg.WrapPrefix = "w"
	}
	if cfg.HubChainID == "" {
		cfg.HubChainID = "onex-mainnet-1"
	}
	return cfg
}

func FindToken(tokens []PlatformToken, chainID, tokenID string) (*PlatformToken, int) {
	for i := range tokens {
		if tokens[i].ChainID == chainID && tokens[i].ID == tokenID {
			return &tokens[i], i
		}
	}
	return nil, -1
}

func FindByOriginKey(tokens []PlatformToken, originKey string) []PlatformToken {
	var out []PlatformToken
	for _, t := range tokens {
		if t.OriginKey == originKey || tokenKey(t.ChainID, t.ID) == originKey {
			out = append(out, t)
		}
	}
	return out
}

func tokenKey(chainID, tokenID string) string {
	return chainID + ":" + tokenID
}

func AppendWrapRecord(tokens []PlatformToken, idx int, rec WrapRecord) []PlatformToken {
	if idx < 0 || idx >= len(tokens) {
		return tokens
	}
	tokens[idx].WrappedOn = append(tokens[idx].WrappedOn, rec)
	return tokens
}

// StatusSummary aggregates platform stats.
type StatusSummary struct {
	Config         PlatformConfig   `json:"config"`
	TotalTokens    int              `json:"totalTokens"`
	TotalWraps     int              `json:"totalWraps"`
	ChainsSupported int             `json:"chainsSupported"`
	ByChainType    map[string]int   `json:"byChainType"`
	RecentTokens   []PlatformToken  `json:"recentTokens,omitempty"`
}

func BuildStatus(cfg PlatformConfig, tokens []PlatformToken, wraps []WrapRecord, chainCount int) StatusSummary {
	byType := map[string]int{}
	for _, t := range tokens {
		byType[t.ChainType]++
	}
	recent := tokens
	if len(recent) > 5 {
		recent = recent[len(recent)-5:]
	}
	return StatusSummary{
		Config:          cfg,
		TotalTokens:     len(tokens),
		TotalWraps:      len(wraps),
		ChainsSupported: chainCount,
		ByChainType:     byType,
		RecentTokens:    recent,
	}
}

func MergeWithCustom(platform []PlatformToken, custom []CustomTokenLite, chainTypes map[string]string) []PlatformToken {
	seen := make(map[string]bool, len(platform)+len(custom))
	out := make([]PlatformToken, 0, len(platform)+len(custom))
	for _, t := range platform {
		key := t.ChainID + ":" + t.ID
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, t)
	}
	for _, ct := range custom {
		key := ct.ChainID + ":" + ct.ID
		if seen[key] {
			continue
		}
		seen[key] = true
		ctype := chainTypes[ct.ChainID]
		if ctype == "" {
			ctype = "evm"
		}
		out = append(out, PlatformToken{
			ID: ct.ID, ChainID: ct.ChainID, ChainType: ctype,
			Name: ct.Name, Symbol: ct.Symbol, Decimals: ct.Decimals,
			Supply: ct.Supply, Creator: ct.Creator, CreatedAt: ct.CreatedAt,
			DeployStatus: "registered",
		})
	}
	return out
}

// CustomTokenLite is used to merge legacy custom-tokens.json into platform listings.
type CustomTokenLite struct {
	ID, ChainID, Name, Symbol, Supply, Creator string
	Decimals                                   int
	CreatedAt                                  int64
}

func ValidateWrapAmount(amount uint64) error {
	if amount == 0 {
		return fmt.Errorf("wrap amount must be > 0")
	}
	return nil
}
