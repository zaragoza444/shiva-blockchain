package bridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/onex-blockchain/onex/internal/legacy"
)

type CustomToken struct {
	ID        string `json:"id"`
	ChainID   string `json:"chainId"`
	Name      string `json:"name"`
	Symbol    string `json:"symbol"`
	Decimals  int    `json:"decimals"`
	Supply    string `json:"supply"`
	Creator   string `json:"creator"`
	CreatedAt int64  `json:"createdAt"`
}

type customTokenStore struct {
	mu   sync.Mutex
	path string
}

func (b *Bridge) customTokens() *customTokenStore {
	if b.custom == nil {
		b.custom = &customTokenStore{path: filepath.Join(legacy.HomeDir(), "custom-tokens.json")}
	}
	return b.custom
}

func (cts *customTokenStore) load() ([]CustomToken, error) {
	cts.mu.Lock()
	defer cts.mu.Unlock()
	data, err := os.ReadFile(cts.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []CustomToken
	return out, json.Unmarshal(data, &out)
}

func (cts *customTokenStore) save(tokens []CustomToken) error {
	cts.mu.Lock()
	defer cts.mu.Unlock()
	_ = os.MkdirAll(filepath.Dir(cts.path), 0o755)
	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cts.path, data, 0o644)
}

func (b *Bridge) ListCustomTokens() ([]CustomToken, error) {
	return b.customTokens().load()
}

func (b *Bridge) mergeCustomTokensIntoRegistry() {
	tokens, _ := b.customTokens().load()
	reg := b.registry()
	reg.mu.Lock()
	defer reg.mu.Unlock()
	seen := make(map[string]bool)
	for _, t := range reg.Tokens {
		seen[t.ChainID+":"+t.ID] = true
	}
	for _, ct := range tokens {
		key := ct.ChainID + ":" + ct.ID
		if seen[key] {
			continue
		}
		reg.Tokens = append(reg.Tokens, TokenInfo{
			ID: ct.ID, ChainID: ct.ChainID, Name: ct.Name,
			Symbol: ct.Symbol, Decimals: ct.Decimals,
		})
		seen[key] = true
	}
}

func sanitizeTokenID(symbol string) string {
	re := regexp.MustCompile(`[^A-Z0-9]`)
	s := strings.ToUpper(symbol)
	s = re.ReplaceAllString(s, "")
	if len(s) < 2 {
		s = "TKN" + newID()[:4]
	}
	if len(s) > 12 {
		s = s[:12]
	}
	return s
}

func (b *Bridge) CreateToken(chainID, name, symbol string, decimals int, supplyStr string) (*CustomToken, error) {
	return b.CreateTokenViaPlatform(chainID, name, symbol, decimals, supplyStr)
}
