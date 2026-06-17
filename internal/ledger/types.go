package ledger

// SourceKind identifies where a ledger entry originated.
type SourceKind string

const (
	SourceBank      SourceKind = "bank"
	SourceOneX      SourceKind = "onex"
	SourceEVM       SourceKind = "evm"
	SourcePortfolio SourceKind = "portfolio"
	SourceImport    SourceKind = "import"
)

// Mode distinguishes verified on-chain/bank data from simulated portfolio balances.
type Mode string

const (
	ModeReal       Mode = "real"
	ModeSimulated  Mode = "simulated"
	ModeBank       Mode = "bank"
	ModeFiat       Mode = "fiat"
)

// Entry is a normalized ledger line in canonical real units.
type Entry struct {
	ID           string     `json:"id"`
	Source       SourceKind `json:"source"`
	Mode         Mode       `json:"mode"`
	ChainID      string     `json:"chainId,omitempty"`
	Asset        string     `json:"asset"`
	TokenKey     string     `json:"tokenKey,omitempty"`
	Atomic       string     `json:"atomic,omitempty"`
	Human        string     `json:"human"`
	FiatCurrency string     `json:"fiatCurrency"`
	FiatValue    float64    `json:"fiatValue"`
	FiatUSD      float64    `json:"fiatUsd"`
	Account      string     `json:"account,omitempty"`
	Timestamp    int64      `json:"timestamp,omitempty"`
	Reference    string     `json:"reference,omitempty"`
}

// Snapshot is a point-in-time unified ledger across all sources.
type Snapshot struct {
	At        int64              `json:"at"`
	Mode      string             `json:"mode"`
	Entries   []Entry            `json:"entries"`
	TotalUSD  float64            `json:"totalUsd"`
	BySource  map[string]float64 `json:"bySourceUsd"`
	ByFiat    map[string]float64 `json:"byFiat"`
}

// ConvertRequest converts an amount between ledger assets.
type ConvertRequest struct {
	FromAsset    string  `json:"fromAsset"`
	ToAsset      string  `json:"toAsset"`
	Amount       string  `json:"amount"`
	FiatCurrency string  `json:"fiatCurrency,omitempty"`
}

// ConvertResult holds the converted amount and valuation.
type ConvertResult struct {
	FromAsset    string  `json:"fromAsset"`
	ToAsset      string  `json:"toAsset"`
	FromAmount   string  `json:"fromAmount"`
	ToAmount     string  `json:"toAmount"`
	Rate         float64 `json:"rate"`
	FiatCurrency string  `json:"fiatCurrency"`
	FiatValue    float64 `json:"fiatValue"`
	FiatUSD      float64 `json:"fiatUsd"`
	Mode         Mode    `json:"mode"`
}

// TokenMeta describes a fungible asset for conversion.
type TokenMeta struct {
	ChainID  string
	TokenID  string
	Symbol   string
	Decimals int
	Native   bool
}

// PriceQuote is a spot price in USD.
type PriceQuote struct {
	USD float64
}

// BankFile is the JSON schema for bank ledger import (file or API).
type BankFile struct {
	Accounts []BankAccount `json:"accounts"`
}

type BankAccount struct {
	ID       string `json:"id,omitempty"`
	IBAN     string `json:"iban,omitempty"`
	Name     string `json:"name,omitempty"`
	Currency string `json:"currency"`
	Balance  string `json:"balance"`
}

// ImportFile accepts arbitrary external ledger rows for normalization.
type ImportFile struct {
	Entries []ImportRow `json:"entries"`
}

type ImportRow struct {
	Source    string `json:"source,omitempty"`
	Asset     string `json:"asset"`
	Amount    string `json:"amount"`
	Currency  string `json:"currency,omitempty"`
	Account   string `json:"account,omitempty"`
	Reference string `json:"reference,omitempty"`
}
