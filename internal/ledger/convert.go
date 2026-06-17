package ledger

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// FiatRates maps ISO currency codes to USD (1 unit of currency = X USD).
var FiatRates = map[string]float64{
	"USD": 1,
	"EUR": 1.08,
	"GBP": 1.27,
	"JPY": 0.0067,
	"CHF": 1.12,
	"AED": 0.27,
	"INR": 0.012,
	"CNY": 0.14,
	"BRL": 0.20,
	"NGN": 0.00065,
}

// ConvertAmount converts between crypto symbols and fiat currencies using live USD quotes.
func ConvertAmount(req ConvertRequest, prices map[string]PriceQuote, tokenBySymbol map[string]TokenMeta) (*ConvertResult, error) {
	from := strings.ToUpper(strings.TrimSpace(req.FromAsset))
	to := strings.ToUpper(strings.TrimSpace(req.ToAsset))
	if from == "" || to == "" {
		return nil, fmt.Errorf("fromAsset and toAsset required")
	}
	amt, err := strconv.ParseFloat(strings.TrimSpace(req.Amount), 64)
	if err != nil || amt < 0 {
		return nil, fmt.Errorf("invalid amount")
	}
	fiat := strings.ToUpper(strings.TrimSpace(req.FiatCurrency))
	if fiat == "" {
		fiat = "USD"
	}

	fromUSD := unitUSD(from, prices, fiat)
	toUSD := unitUSD(to, prices, fiat)
	if fromUSD <= 0 {
		return nil, fmt.Errorf("no price for %s", from)
	}
	if toUSD <= 0 {
		return nil, fmt.Errorf("no price for %s", to)
	}

	usdValue := amt * fromUSD
	toAmt := usdValue / toUSD
	rate := fromUSD / toUSD

	mode := ModeReal
	if isFiat(from) || isFiat(to) {
		mode = ModeFiat
	}
	_ = tokenBySymbol

	return &ConvertResult{
		FromAsset:    from,
		ToAsset:      to,
		FromAmount:   formatFloat(amt),
		ToAmount:     formatFloat(toAmt),
		Rate:         rate,
		FiatCurrency: fiat,
		FiatValue:    usdToFiat(usdValue, fiat),
		FiatUSD:      usdValue,
		Mode:         mode,
	}, nil
}

func unitUSD(asset string, prices map[string]PriceQuote, fiat string) float64 {
	if isFiat(asset) {
		rate, ok := FiatRates[asset]
		if !ok {
			return 0
		}
		return rate
	}
	if q, ok := prices[asset]; ok && q.USD > 0 {
		return q.USD
	}
	// Stablecoin fallback
	switch asset {
	case "USDT", "USDC", "DAI", "BUSD":
		return 1
	}
	return 0
}

func isFiat(sym string) bool {
	_, ok := FiatRates[strings.ToUpper(sym)]
	return ok
}

func usdToFiat(usd float64, fiat string) float64 {
	rate, ok := FiatRates[fiat]
	if !ok || rate <= 0 {
		return usd
	}
	return usd / rate
}

func formatFloat(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	s := fmt.Sprintf("%.12f", v)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-" {
		return "0"
	}
	return s
}

// ValueEntry computes fiat valuation for a ledger entry.
func ValueEntry(e Entry, prices map[string]PriceQuote) Entry {
	asset := strings.ToUpper(strings.TrimSpace(e.Asset))
	fiat := strings.ToUpper(strings.TrimSpace(e.FiatCurrency))
	if fiat == "" {
		fiat = "USD"
	}
	e.FiatCurrency = fiat

	human, _ := strconv.ParseFloat(strings.TrimSpace(e.Human), 64)
	if human <= 0 && e.Atomic != "" {
		human = atomicToHuman(e.Atomic, e.Asset)
	}
	if human <= 0 {
		return e
	}
	e.Human = formatFloat(human)

	usdPer := unitUSD(asset, prices, fiat)
	if usdPer <= 0 {
		return e
	}
	e.FiatUSD = human * usdPer
	e.FiatValue = usdToFiat(e.FiatUSD, fiat)
	return e
}

func atomicToHuman(atomic, asset string) float64 {
	n, err := strconv.ParseUint(strings.TrimSpace(atomic), 10, 64)
	if err != nil {
		return 0
	}
	dec := decimalsForSymbol(asset)
	div := math.Pow10(dec)
	if div <= 0 {
		return 0
	}
	return float64(n) / div
}

func decimalsForSymbol(sym string) int {
	switch strings.ToUpper(sym) {
	case "BTC", "ONEX", "WONEX", "SONEX", "WBTC":
		return 8
	case "ETH", "BNB", "MATIC", "AVAX", "ALL":
		return 18
	case "SOL":
		return 9
	case "TRX":
		return 6
	case "USDT", "USDC":
		return 6
	default:
		return 8
	}
}

// Summarize builds totals from valued entries.
func Summarize(entries []Entry, mode string) Snapshot {
	now := entries
	out := Snapshot{
		Mode:     mode,
		Entries:  now,
		BySource: make(map[string]float64),
		ByFiat:   make(map[string]float64),
	}
	for _, e := range entries {
		out.TotalUSD += e.FiatUSD
		out.BySource[string(e.Source)] += e.FiatUSD
		if e.FiatCurrency != "" {
			out.ByFiat[e.FiatCurrency] += e.FiatValue
		}
	}
	return out
}
