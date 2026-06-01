package bridge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/onex-blockchain/onex/internal/legacy"
)

type Portfolio struct {
	Address  string                       `json:"address"`
	Balances map[string]string            `json:"balances"` // tokenKey -> atomic amount string
	Deposits []DepositRecord              `json:"deposits"`
	Swaps    []SwapRecord                 `json:"swaps"`
	NFTs     []NFTAsset                   `json:"nfts"`
	Tasks    []TaskItem                   `json:"tasks"`
	Loans         []LoanRecord `json:"loans"`
	Stakes        []StakeRecord `json:"stakes"`
	CreatedTokens []string     `json:"createdTokens"`
}

type DepositRecord struct {
	ID        string `json:"id"`
	ChainID   string `json:"chainId"`
	TokenID   string `json:"tokenId"`
	Amount    string `json:"amount"`
	TxHash    string `json:"txHash,omitempty"`
	Status    string `json:"status"` // pending, confirmed
	CreatedAt int64  `json:"createdAt"`
}

type SwapRecord struct {
	ID        string `json:"id"`
	FromKey   string `json:"fromKey"`
	ToKey     string `json:"toKey"`
	FromAmt   string `json:"fromAmount"`
	ToAmt     string `json:"toAmount"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"createdAt"`
}

type NFTAsset struct {
	ID          string `json:"id"`
	ChainID     string `json:"chainId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ImageURL    string `json:"imageUrl"`
	Owner       string `json:"owner"`
	CreatedAt   int64  `json:"createdAt"`
}

type TaskItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	RewardKey   string `json:"rewardKey"`
	RewardAmt   string `json:"rewardAmount"`
	Status      string `json:"status"` // open, done
}

type LoanRecord struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"` // borrow, lend
	CollateralKey string `json:"collateralKey"`
	CollateralAmt string `json:"collateralAmount"`
	DebtKey      string  `json:"debtKey"`
	DebtAmt      string  `json:"debtAmount"`
	APY          float64 `json:"apy"`
	Status       string  `json:"status"` // active, repaid
	CreatedAt    int64   `json:"createdAt"`
}

type PortfolioStore struct {
	mu   sync.Mutex
	dir  string
	data map[string]*Portfolio
}

func NewPortfolioStore(dir string) *PortfolioStore {
	return &PortfolioStore{dir: dir, data: make(map[string]*Portfolio)}
}

func (ps *PortfolioStore) path(addr string) string {
	h := sha256.Sum256([]byte(addr))
	return filepath.Join(ps.dir, "portfolio_"+hex.EncodeToString(h[:8])+".json")
}

func (ps *PortfolioStore) Load(addr string) (*Portfolio, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if p, ok := ps.data[addr]; ok {
		return p.clone(), nil
	}
	_ = os.MkdirAll(ps.dir, 0o755)
	data, err := os.ReadFile(ps.path(addr))
	if err != nil {
		p := defaultPortfolio(addr)
		ps.data[addr] = p
		return p.clone(), nil
	}
	var p Portfolio
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	if legacy.MigrateBalanceKeys(p.Balances) {
		if data, err := json.MarshalIndent(p, "", "  "); err == nil {
			_ = os.WriteFile(ps.path(addr), data, 0o600)
		}
	}
	ps.data[addr] = &p
	return p.clone(), nil
}

func (ps *PortfolioStore) Save(p *Portfolio) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.data[p.Address] = p
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ps.path(p.Address), data, 0o600)
}

func defaultPortfolio(addr string) *Portfolio {
	return &Portfolio{
		Address: addr,
		Balances: map[string]string{
			"ethereum:USDT":  "5000000000",
			"ethereum:ETH":   "100000000",
			"bsc:BNB":        "2000000000",
			"polygon:USDC-POLY": "3000000000",
			"onex-mainnet-1:wONEX": "10000000000",
		},
		Tasks: defaultTasks(),
	}
}

func defaultTasks() []TaskItem {
	return []TaskItem{
		{ID: "welcome", Title: "Welcome to OneX", Description: "Complete your profile in the wallet", RewardKey: "onex-mainnet-1:ONEX", RewardAmt: "1000000000", Status: "open"},
		{ID: "first-swap", Title: "First swap", Description: "Swap any token to ONEX", RewardKey: "onex-mainnet-1:ONEX", RewardAmt: "500000000", Status: "open"},
		{ID: "mint-nft", Title: "Mint an NFT", Description: "Create your first OneX NFT", RewardKey: "onex-mainnet-1:wONEX", RewardAmt: "100000000", Status: "open"},
		{ID: "deposit", Title: "Bridge deposit", Description: "Record a cross-chain deposit", RewardKey: "onex-mainnet-1:ONEX", RewardAmt: "250000000", Status: "open"},
		{ID: "first-stake", Title: "Stake tokens", Description: "Stake ONEX or any supported token", RewardKey: "onex-mainnet-1:sONEX", RewardAmt: "200000000", Status: "open"},
		{ID: "create-token", Title: "Create a token", Description: "Launch your own token on OneX", RewardKey: "onex-mainnet-1:ONEX", RewardAmt: "1000000000", Status: "open"},
	}
}

func (p *Portfolio) clone() *Portfolio {
	b, _ := json.Marshal(p)
	var out Portfolio
	_ = json.Unmarshal(b, &out)
	return &out
}

func (p *Portfolio) GetBalance(key string) uint64 {
	s := p.Balances[key]
	if s == "" {
		return 0
	}
	var n uint64
	fmt.Sscanf(s, "%d", &n)
	return n
}

func (p *Portfolio) SetBalance(key string, amount uint64) {
	p.Balances[key] = fmt.Sprintf("%d", amount)
}

func (p *Portfolio) AddBalance(key string, delta uint64) {
	p.SetBalance(key, p.GetBalance(key)+delta)
}

func (p *Portfolio) SubBalance(key string, delta uint64) error {
	cur := p.GetBalance(key)
	if cur < delta {
		return fmt.Errorf("insufficient balance for %s", key)
	}
	p.SetBalance(key, cur-delta)
	return nil
}

func DepositAddress(walletAddr, chainID string) string {
	h := sha256.Sum256([]byte("deposit:" + walletAddr + ":" + chainID))
	return hex.EncodeToString(h[:])[:40]
}

func newID() string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(h[:8])
}
