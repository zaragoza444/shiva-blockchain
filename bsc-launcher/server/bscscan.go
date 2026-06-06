package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const bscScanV2Base = "https://api.etherscan.io/v2/api"

type BSCScanTokenInfo struct {
	ContractAddress string `json:"contractAddress"`
	TokenName       string `json:"tokenName"`
	Symbol          string `json:"symbol"`
	Divisor         string `json:"divisor"`
	TotalSupply     string `json:"totalSupply"`
	Holders         string `json:"holders"`
	TxCount         string `json:"txCount"`
	Creator         string `json:"creator"`
	IsContract      bool   `json:"isContract,omitempty"`
	IsWallet        bool   `json:"isWallet,omitempty"`
	Error           string `json:"error,omitempty"`
}

type cacheEntry struct {
	at   time.Time
	data interface{}
}

type bscScanClient struct {
	apiKey  string
	chainID int64
	ttl     time.Duration
	mu      sync.Mutex
	cache   map[string]cacheEntry
}

func newBSCScanClient(apiKey string) *bscScanClient {
	return &bscScanClient{
		apiKey:  apiKey,
		chainID: 56,
		ttl:     90 * time.Second,
		cache:   make(map[string]cacheEntry),
	}
}

func (c *bscScanClient) getCached(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.cache[key]
	if !ok || time.Since(e.at) > c.ttl {
		return nil, false
	}
	return e.data, true
}

func (c *bscScanClient) setCached(key string, data interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = cacheEntry{at: time.Now(), data: data}
}

func (c *bscScanClient) TokenInfoForChain(chainID int64, address string) (*BSCScanTokenInfo, error) {
	if chainID == 0 {
		chainID = c.chainID
	}
	key := fmt.Sprintf("tokeninfo:%d:%s", chainID, strings.ToLower(address))
	if v, ok := c.getCached(key); ok {
		return v.(*BSCScanTokenInfo), nil
	}

	info := &BSCScanTokenInfo{ContractAddress: address}
	if c.apiKey == "" {
		c.setCached(key, info)
		return info, nil
	}

	q := url.Values{}
	q.Set("chainid", fmt.Sprintf("%d", chainID))
	q.Set("module", "token")
	q.Set("action", "tokeninfo")
	q.Set("contractaddress", address)
	q.Set("apikey", c.apiKey)

	body, err := c.request(q)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Status  string          `json:"status"`
		Message string          `json:"message"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if resp.Status != "1" {
		msg := resp.Message
		if len(resp.Result) > 0 && string(resp.Result) != "[]" {
			msg = strings.Trim(string(resp.Result), `"`)
		}
		if msg != "" && !strings.EqualFold(msg, "OK") {
			return nil, fmt.Errorf("bscscan: %s", msg)
		}
	}

	var items []BSCScanTokenInfo
	if err := json.Unmarshal(resp.Result, &items); err == nil && len(items) > 0 {
		info = &items[0]
	}

	c.setCached(key, info)
	return info, nil
}

func (c *bscScanClient) TokenInfo(address string) (*BSCScanTokenInfo, error) {
	return c.TokenInfoForChain(c.chainID, address)
}

func (c *bscScanClient) request(q url.Values) ([]byte, error) {
	u := bscScanV2Base + "?" + q.Encode()
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 15 * time.Second}
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
		return nil, fmt.Errorf("bscscan http %d", res.StatusCode)
	}
	return body, nil
}
