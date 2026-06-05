package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type marketClient struct {
	ttl   time.Duration
	mu    sync.Mutex
	cache map[string]cacheEntry
}

func newMarketClient() *marketClient {
	return &marketClient{
		ttl:   90 * time.Second,
		cache: make(map[string]cacheEntry),
	}
}

func (c *marketClient) getCached(key string) (float64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.cache[key]
	if !ok || time.Since(e.at) > c.ttl {
		return 0, false
	}
	v, ok := e.data.(float64)
	return v, ok
}

func (c *marketClient) setCached(key string, v float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = cacheEntry{at: time.Now(), data: v}
}

func (c *marketClient) BNBUSD() (float64, error) {
	if v, ok := c.getCached("bnb"); ok {
		return v, nil
	}
	u := "https://api.coingecko.com/api/v3/simple/price?ids=binancecoin&vs_currencies=usd"
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}
	var out struct {
		Binancecoin struct {
			USD float64 `json:"usd"`
		} `json:"binancecoin"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return 0, err
	}
	if out.Binancecoin.USD <= 0 {
		return 0, fmt.Errorf("bnb price unavailable")
	}
	c.setCached("bnb", out.Binancecoin.USD)
	return out.Binancecoin.USD, nil
}
