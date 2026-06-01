package bridge

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

// MarketQuote is a USD spot quote for display in the wallet.
type MarketQuote struct {
	USD         float64 `json:"usd"`
	USD24hChange float64 `json:"usd24hChange"`
}

type marketCache struct {
	mu    sync.Mutex
	at    time.Time
	ttl   time.Duration
	quotes map[string]MarketQuote
}

var globalMarket = &marketCache{ttl: 90 * time.Second}

// symbol -> CoinGecko API id
var cgIDBySymbol = map[string]string{
	"BTC":  "bitcoin",
	"ETH":  "ethereum",
	"USDT": "tether",
	"USDC": "usd-coin",
	"WBTC": "wrapped-bitcoin",
	"BNB":  "binancecoin",
	"MATIC": "matic-network",
	"AVAX": "avalanche-2",
	"SOL":  "solana",
	"TRX":  "tron",
}

// synthetic USD when not listed on CoinGecko (derived from configs/swap-pairs.json pegs).
var syntheticUSD = map[string]float64{
	"ONEX":  0.01,
	"tONEX": 0.01,
	"wONEX": 0.01,
	"sONEX": 0.0095,
	"sETH":   0, // filled from ETH
	"sUSDT":  0,
	"sBNB":   0,
	"ALL":    0.00042,
}

func (b *Bridge) MarketPrices() map[string]MarketQuote {
	globalMarket.mu.Lock()
	if time.Since(globalMarket.at) < globalMarket.ttl && len(globalMarket.quotes) > 0 {
		out := copyQuotes(globalMarket.quotes)
		globalMarket.mu.Unlock()
		b.applySynthetic(out)
		return out
	}
	globalMarket.mu.Unlock()

	fetched, err := fetchCoinGeckoPrices()
	out := map[string]MarketQuote{}
	if err == nil {
		out = fetched
	}

	globalMarket.mu.Lock()
	globalMarket.quotes = out
	globalMarket.at = time.Now()
	globalMarket.mu.Unlock()

	b.applySynthetic(out)
	return out
}

func copyQuotes(m map[string]MarketQuote) map[string]MarketQuote {
	out := make(map[string]MarketQuote, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (b *Bridge) applySynthetic(out map[string]MarketQuote) {
	for sym, usd := range syntheticUSD {
		if sym == "sETH" || sym == "sUSDT" || sym == "sBNB" {
			continue
		}
		if _, ok := out[sym]; !ok && usd > 0 {
			out[sym] = MarketQuote{USD: usd}
		}
	}
	if eth, ok := out["ETH"]; ok {
		if q, ok2 := out["sETH"]; !ok2 || q.USD == 0 {
			out["sETH"] = MarketQuote{USD: eth.USD * 0.98, USD24hChange: eth.USD24hChange}
		}
	}
	if usdt, ok := out["USDT"]; ok {
		if q, ok2 := out["sUSDT"]; !ok2 || q.USD == 0 {
			out["sUSDT"] = MarketQuote{USD: usdt.USD * 0.99, USD24hChange: usdt.USD24hChange}
		}
	}
	if bnb, ok := out["BNB"]; ok {
		if q, ok2 := out["sBNB"]; !ok2 || q.USD == 0 {
			out["sBNB"] = MarketQuote{USD: bnb.USD * 0.98, USD24hChange: bnb.USD24hChange}
		}
	}
}

func fetchCoinGeckoPrices() (map[string]MarketQuote, error) {
	ids := make([]string, 0, len(cgIDBySymbol))
	for _, id := range cgIDBySymbol {
		ids = append(ids, id)
	}
	url := "https://api.coingecko.com/api/v3/simple/price?ids=" + strings.Join(ids, ",") +
		"&vs_currencies=usd&include_24hr_change=true"

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("coingecko status %d", resp.StatusCode)
	}

	var raw map[string]struct {
		USD          float64 `json:"usd"`
		USD24hChange float64 `json:"usd_24h_change"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	idToSym := make(map[string]string, len(cgIDBySymbol))
	for sym, id := range cgIDBySymbol {
		idToSym[id] = sym
	}

	out := make(map[string]MarketQuote)
	for id, row := range raw {
		sym, ok := idToSym[id]
		if !ok {
			continue
		}
		out[sym] = MarketQuote{USD: row.USD, USD24hChange: row.USD24hChange}
	}
	return out, nil
}

// ChartPoint is one USD price sample for sparkline / detail charts.
type ChartPoint struct {
	T int64   `json:"t"`
	V float64 `json:"v"`
}

type chartCacheEntry struct {
	at   time.Time
	pts  []ChartPoint
}

var chartCache sync.Map

func coingeckoIDForSymbol(sym string) string {
	if id, ok := cgIDBySymbol[sym]; ok {
		return id
	}
	switch sym {
	case "sETH":
		return "ethereum"
	case "sUSDT":
		return "tether"
	case "sBNB":
		return "binancecoin"
	}
	return ""
}

func syntheticBaseUSD(sym string, quotes map[string]MarketQuote) float64 {
	if q, ok := quotes[sym]; ok && q.USD > 0 {
		return q.USD
	}
	if u, ok := syntheticUSD[sym]; ok && u > 0 {
		return u
	}
	return 0
}

// MarketChart returns USD price history for a token symbol.
func (b *Bridge) MarketChart(symbol, days string) ([]ChartPoint, error) {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	if sym == "" {
		return nil, fmt.Errorf("symbol required")
	}
	switch days {
	case "1", "7", "30":
	default:
		days = "7"
	}
	key := sym + ":" + days
	if v, ok := chartCache.Load(key); ok {
		e := v.(chartCacheEntry)
		if time.Since(e.at) < 5*time.Minute && len(e.pts) > 0 {
			return e.pts, nil
		}
	}

	quotes := b.MarketPrices()
	cgID := coingeckoIDForSymbol(sym)
	var pts []ChartPoint
	var err error
	if cgID != "" {
		pts, err = fetchCoinGeckoChart(cgID, days)
	}
	if err != nil || len(pts) < 2 {
		base := syntheticBaseUSD(sym, quotes)
		if base <= 0 {
			if parent := chartParentSymbol(sym); parent != "" {
				base = syntheticBaseUSD(parent, quotes)
			}
		}
		if base <= 0 {
			return nil, fmt.Errorf("no chart data for %s", sym)
		}
		pts = syntheticChart(base, days)
	}
	chartCache.Store(key, chartCacheEntry{at: time.Now(), pts: pts})
	return pts, nil
}

func chartParentSymbol(sym string) string {
	switch sym {
	case "sETH":
		return "ETH"
	case "sUSDT":
		return "USDT"
	case "sBNB":
		return "BNB"
	case "tONEX", "wONEX", "sONEX":
		return "ONEX"
	}
	return ""
}

func fetchCoinGeckoChart(cgID, days string) ([]ChartPoint, error) {
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/%s/market_chart?vs_currency=usd&days=%s", cgID, days)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("coingecko chart status %d", resp.StatusCode)
	}
	var raw struct {
		Prices [][]float64 `json:"prices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := make([]ChartPoint, 0, len(raw.Prices))
	for _, p := range raw.Prices {
		if len(p) < 2 {
			continue
		}
		out = append(out, ChartPoint{T: int64(p[0]), V: p[1]})
	}
	// Downsample for UI
	if len(out) > 96 {
		step := len(out) / 96
		if step < 1 {
			step = 1
		}
		trim := make([]ChartPoint, 0, 96)
		for i := 0; i < len(out); i += step {
			trim = append(trim, out[i])
		}
		if trim[len(trim)-1].T != out[len(out)-1].T {
			trim = append(trim, out[len(out)-1])
		}
		out = trim
	}
	return out, nil
}

func syntheticChart(base float64, days string) []ChartPoint {
	n := 48
	now := time.Now().UnixMilli()
	var span int64 = 7 * 86400000
	switch days {
	case "1":
		span = 86400000
	case "30":
		span = 30 * 86400000
	}
	out := make([]ChartPoint, n)
	for i := 0; i < n; i++ {
		t := now - span + int64(i)*span/int64(n-1)
		wave := 1 + 0.035*math.Sin(float64(i)*0.55)
		out[i] = ChartPoint{T: t, V: base * wave}
	}
	return out
}
