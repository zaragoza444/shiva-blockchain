package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	pancakeRouter  = "0x10ED43C718714eb63d5aA57B78B54704E256024E"
	pancakeFactory = "0xcA143Ce32Fe78f1f7019d7d551a6402fC5350c73"
	wbnbAddress    = "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c"
	usdtAddress    = "0x55d398326f99059fF775485246999027B3197955"
)

type DexConfig struct {
	Router  string            `json:"router"`
	Factory string            `json:"factory"`
	WBNB    string            `json:"wbnb"`
	USDT    string            `json:"usdt"`
	Name    string            `json:"name"`
	Quotes  []QuoteToken      `json:"quotes"`
	RouterABI []map[string]interface{} `json:"routerAbi"`
	FactoryABI []map[string]interface{} `json:"factoryAbi"`
}

type QuoteToken struct {
	ID       string `json:"id"`
	Symbol   string `json:"symbol"`
	Address  string `json:"address"`
	Decimals int    `json:"decimals"`
}

type LiquidityRecord struct {
	TokenAddress string `json:"tokenAddress"`
	QuoteID      string `json:"quoteId"`
	PairAddress  string `json:"pairAddress"`
	TokenAmount  string `json:"tokenAmount"`
	QuoteAmount  string `json:"quoteAmount"`
	TxHash       string `json:"txHash"`
	Creator      string `json:"creator"`
	CreatedAt    int64  `json:"createdAt"`
}

type LiquidityStore struct {
	mu   sync.Mutex
	path string
}

func NewLiquidityStore(dataDir string) *LiquidityStore {
	return &LiquidityStore{path: filepath.Join(dataDir, "liquidity.json")}
}

func (s *LiquidityStore) Load() ([]LiquidityRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []LiquidityRecord{}, nil
		}
		return nil, err
	}
	var list []LiquidityRecord
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (s *LiquidityStore) Save(list []LiquidityRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

func (s *LiquidityStore) Add(rec LiquidityRecord) error {
	list, err := s.Load()
	if err != nil {
		return err
	}
	if rec.CreatedAt == 0 {
		rec.CreatedAt = time.Now().Unix()
	}
	list = append(list, rec)
	return s.Save(list)
}

func loadABIFile(name string) ([]map[string]interface{}, error) {
	path := filepath.Join(projectRoot(), "abi", name)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var abiArr []map[string]interface{}
	if err := json.Unmarshal(data, &abiArr); err != nil {
		return nil, err
	}
	return abiArr, nil
}

func dexConfig() (DexConfig, error) {
	routerABI, err := loadABIFile("PancakeRouter.json")
	if err != nil {
		return DexConfig{}, err
	}
	factoryABI, err := loadABIFile("PancakeFactory.json")
	if err != nil {
		return DexConfig{}, err
	}
	return DexConfig{
		Router:     pancakeRouter,
		Factory:    pancakeFactory,
		WBNB:       wbnbAddress,
		USDT:       usdtAddress,
		Name:       "PancakeSwap V2",
		RouterABI:  routerABI,
		FactoryABI: factoryABI,
		Quotes: []QuoteToken{
			{ID: "bnb", Symbol: "BNB", Address: wbnbAddress, Decimals: 18},
			{ID: "usdt", Symbol: "USDT", Address: usdtAddress, Decimals: 18},
		},
	}, nil
}

func quoteAddress(quoteID string) (common.Address, error) {
	switch strings.ToLower(quoteID) {
	case "bnb", "wbnb":
		return common.HexToAddress(wbnbAddress), nil
	case "usdt":
		return common.HexToAddress(usdtAddress), nil
	default:
		return common.Address{}, fmt.Errorf("unsupported quote: %s", quoteID)
	}
}

func (s *Server) getPairAddress(ctx context.Context, tokenAddr, quoteID string) (string, error) {
	quote, err := quoteAddress(quoteID)
	if err != nil {
		return "", err
	}
	token := common.HexToAddress(tokenAddr)

	client, err := s.rpcClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	factoryABI, err := loadABIFile("PancakeFactory.json")
	if err != nil {
		return "", err
	}
	parsed, err := abi.JSON(strings.NewReader(mustJSON(factoryABI)))
	if err != nil {
		return "", err
	}

	data, err := parsed.Pack("getPair", token, quote)
	if err != nil {
		return "", err
	}
	factory := common.HexToAddress(pancakeFactory)
	out, err := client.CallContract(ctx, ethereum.CallMsg{To: &factory, Data: data}, nil)
	if err != nil {
		return "", err
	}
	vals, err := parsed.Unpack("getPair", out)
	if err != nil || len(vals) == 0 {
		return "", fmt.Errorf("getPair failed")
	}
	pair := vals[0].(common.Address)
	if pair == (common.Address{}) {
		return "", nil
	}
	return pair.Hex(), nil
}

func mustJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
