package ledger

import (
	"context"
	"testing"
)

func TestConvertAmountCryptoToFiat(t *testing.T) {
	prices := map[string]PriceQuote{
		"ETH": {USD: 3000},
	}
	res, err := ConvertAmount(ConvertRequest{
		FromAsset: "ETH",
		ToAsset:   "USD",
		Amount:    "2",
	}, prices, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.FiatUSD < 5999 || res.FiatUSD > 6001 {
		t.Fatalf("expected ~6000 USD got %v", res.FiatUSD)
	}
	if res.ToAmount != "6000" {
		t.Fatalf("expected 6000 got %s", res.ToAmount)
	}
}

func TestConvertAmountFiatCross(t *testing.T) {
	prices := map[string]PriceQuote{}
	res, err := ConvertAmount(ConvertRequest{
		FromAsset:    "EUR",
		ToAsset:      "USD",
		Amount:       "100",
		FiatCurrency: "USD",
	}, prices, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.FiatUSD < 107 || res.FiatUSD > 109 {
		t.Fatalf("expected ~108 USD got %v", res.FiatUSD)
	}
}

func TestParseBankJSON(t *testing.T) {
	data := []byte(`{"accounts":[{"currency":"USD","balance":"1500.25","name":"Checking","iban":"US00TEST"}]}`)
	entries, err := parseBankJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry got %d", len(entries))
	}
	if entries[0].Mode != ModeBank || entries[0].Asset != "USD" {
		t.Fatalf("unexpected entry: %+v", entries[0])
	}
}

func TestValueEntry(t *testing.T) {
	prices := map[string]PriceQuote{"BTC": {USD: 50000}}
	e := ValueEntry(Entry{
		Asset: "BTC",
		Human: "0.5",
	}, prices)
	if e.FiatUSD != 25000 {
		t.Fatalf("expected 25000 got %v", e.FiatUSD)
	}
}

func TestBankProviderResolved(t *testing.T) {
	cfg := BankProviderConfig{
		Provider:         "auto",
		PlaidClientID:    "id",
		PlaidAccessToken: "tok",
	}
	if cfg.ResolvedProvider() != "plaid" {
		t.Fatalf("expected plaid got %s", cfg.ResolvedProvider())
	}
	cfg2 := BankProviderConfig{Provider: "auto", TrueLayerToken: "tl"}
	if cfg2.ResolvedProvider() != "truelayer" {
		t.Fatalf("expected truelayer got %s", cfg2.ResolvedProvider())
	}
}

func TestEngineReadProductionSkipsPortfolio(t *testing.T) {
	eng := NewEngine()
	cfg := Config{Mode: "production", FiatCurrency: "USD"}
	snap := eng.Read(context.Background(), ReadInput{
		Config: cfg,
		Source: "all",
		Portfolio: map[string]string{
			"ethereum:ETH": "1000000000000000000",
		},
		Prices: map[string]PriceQuote{"ETH": {USD: 3000}},
	})
	for _, e := range snap.Entries {
		if e.Source == SourcePortfolio {
			t.Fatal("production mode should not include portfolio entries")
		}
	}
}
