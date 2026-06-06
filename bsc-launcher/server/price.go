package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type PriceQuote struct {
	PriceUSD       float64 `json:"priceUsd"`
	PriceChange24h float64 `json:"priceChange24h"`
	LiquidityUSD   float64 `json:"liquidityUsd"`
	MarketCap      float64 `json:"marketCap"`
	DexID          string  `json:"dexId"`
	PairAddress    string  `json:"pairAddress"`
	HasLiquidity   bool    `json:"hasLiquidity"`
}

type priceClient struct {
	ttl   time.Duration
	mu    sync.Mutex
	cache map[string]cacheEntry
}

func newPriceClient() *priceClient {
	return &priceClient{
		ttl:   60 * time.Second,
		cache: make(map[string]cacheEntry),
	}
}

func (c *priceClient) getCached(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.cache[key]
	if !ok || time.Since(e.at) > c.ttl {
		return nil, false
	}
	return e.data, true
}

func (c *priceClient) setCached(key string, data interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = cacheEntry{at: time.Now(), data: data}
}

func (c *priceClient) Quote(dexChainID, address string) (*PriceQuote, error) {
	if dexChainID == "" {
		dexChainID = "bsc"
	}
	key := "price:" + dexChainID + ":" + strings.ToLower(address)
	if v, ok := c.getCached(key); ok {
		return v.(*PriceQuote), nil
	}

	u := "https://api.dexscreener.com/latest/dex/tokens/" + address
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 12 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dexscreener http %d", res.StatusCode)
	}

	var payload struct {
		Pairs []struct {
			ChainID       string `json:"chainId"`
			DexID         string `json:"dexId"`
			PairAddress   string `json:"pairAddress"`
			PriceUsd      string `json:"priceUsd"`
			PriceChange   struct {
				H24 float64 `json:"h24"`
			} `json:"priceChange"`
			Liquidity struct {
				USD float64 `json:"usd"`
			} `json:"liquidity"`
			MarketCap float64 `json:"marketCap"`
			Fdv       float64 `json:"fdv"`
		} `json:"pairs"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	quote := &PriceQuote{}
	var bestLiq float64
	for _, p := range payload.Pairs {
		if !strings.EqualFold(p.ChainID, dexChainID) && p.ChainID != fmt.Sprintf("%d", chainIDFromDex(dexChainID)) {
			continue
		}
		if p.Liquidity.USD < bestLiq {
			continue
		}
		bestLiq = p.Liquidity.USD
		price, _ := strconv.ParseFloat(p.PriceUsd, 64)
		quote.PriceUSD = price
		quote.PriceChange24h = p.PriceChange.H24
		quote.LiquidityUSD = p.Liquidity.USD
		quote.MarketCap = p.MarketCap
		if quote.MarketCap == 0 {
			quote.MarketCap = p.Fdv
		}
		quote.DexID = p.DexID
		quote.PairAddress = p.PairAddress
		quote.HasLiquidity = p.Liquidity.USD > 0
	}

	c.setCached(key, quote)
	return quote, nil
}
