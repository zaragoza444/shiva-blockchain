package main

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// TokenFeatures mirrors the 20lab-style wizard configuration.
type TokenFeatures struct {
	Chain             string   `json:"chain"`
	Flags             uint32   `json:"flags"`
	Owner             string   `json:"owner"`
	Recipient         string   `json:"recipient"`
	MaxSupply         string   `json:"maxSupply"`
	MaxWalletPct      float64  `json:"maxWalletPct"`
	MaxTxPct          float64  `json:"maxTxPct"`
	AntiBotCooldown   int64    `json:"antiBotCooldown"`
	LiquidityTaxPct   float64  `json:"liquidityTaxPct"`
	DividendTaxPct    float64  `json:"dividendTaxPct"`
	BurnTaxPct        float64  `json:"burnTaxPct"`
	LiquidityWallet   string   `json:"liquidityWallet"`
	DividendWallet    string   `json:"dividendWallet"`
	WalletTaxAccounts []string `json:"walletTaxAccounts"`
	WalletTaxBps      []int    `json:"walletTaxBps"`
}

type InitParams struct {
	Name              string
	Symbol            string
	Decimals          uint8
	InitialSupply     *big.Int
	Owner             common.Address
	Recipient         common.Address
	Flags             uint32
	MaxSupply         *big.Int
	MaxWallet         *big.Int
	MaxTx             *big.Int
	AntiBotCooldown   *big.Int
	LiquidityTaxBps   uint16
	DividendTaxBps    uint16
	BurnTaxBps        uint16
	LiquidityWallet   common.Address
	DividendWallet    common.Address
	WalletTaxAccounts []common.Address
	WalletTaxRates    []uint16
}

func pctToBps(pct float64) uint16 {
	if pct <= 0 {
		return 0
	}
	bps := int(pct * 100)
	if bps > 2500 {
		bps = 2500
	}
	return uint16(bps)
}

func parseOptionalAddress(s string, fallback common.Address) common.Address {
	s = strings.TrimSpace(s)
	if s == "" || !common.IsHexAddress(s) {
		return fallback
	}
	return common.HexToAddress(s)
}

func buildInitParams(name, symbol string, decimals int, supplyRaw *big.Int, features TokenFeatures, deployer string) (InitParams, error) {
	owner := parseOptionalAddress(features.Owner, common.HexToAddress(deployer))
	recipient := parseOptionalAddress(features.Recipient, owner)

	maxSupply := new(big.Int).Set(supplyRaw)
	if strings.TrimSpace(features.MaxSupply) != "" {
		ms, err := parseSupply(features.MaxSupply, decimals)
		if err != nil {
			return InitParams{}, err
		}
		maxSupply = ms
	} else if features.Flags&0x1 != 0 {
		maxSupply = new(big.Int).Mul(supplyRaw, big.NewInt(2))
	}

	maxWalletPct := features.MaxWalletPct
	if maxWalletPct <= 0 {
		maxWalletPct = 2
	}
	maxTxPct := features.MaxTxPct
	if maxTxPct <= 0 {
		maxTxPct = 1
	}
	maxWallet := new(big.Int)
	maxTx := new(big.Int)
	if features.Flags&(1<<4) != 0 {
		maxWallet = pctOfSupply(supplyRaw, maxWalletPct)
	}
	if features.Flags&(1<<5) != 0 {
		maxTx = pctOfSupply(supplyRaw, maxTxPct)
	}

	cooldown := features.AntiBotCooldown
	if cooldown <= 0 {
		cooldown = 30
	}

	walletAccounts := make([]common.Address, 0, len(features.WalletTaxAccounts))
	walletRates := make([]uint16, 0, len(features.WalletTaxBps))
	for i, a := range features.WalletTaxAccounts {
		a = strings.TrimSpace(a)
		if a == "" || !common.IsHexAddress(a) {
			continue
		}
		walletAccounts = append(walletAccounts, common.HexToAddress(a))
		bps := 0
		if i < len(features.WalletTaxBps) {
			bps = features.WalletTaxBps[i]
		}
		if bps > 2500 {
			bps = 2500
		}
		walletRates = append(walletRates, uint16(bps))
	}
	if len(walletAccounts) != len(walletRates) {
		return InitParams{}, fmt.Errorf("wallet tax config mismatch")
	}
	if len(walletAccounts) > 5 {
		return InitParams{}, fmt.Errorf("max 5 wallet taxes")
	}

	liqBps := pctToBps(features.LiquidityTaxPct)
	divBps := pctToBps(features.DividendTaxPct)
	burnBps := pctToBps(features.BurnTaxPct)
	totalTax := int(liqBps) + int(divBps) + int(burnBps)
	for _, w := range walletRates {
		totalTax += int(w)
	}
	if totalTax > 2500 {
		return InitParams{}, fmt.Errorf("total tax cannot exceed 25%%")
	}

	return InitParams{
		Name:              name,
		Symbol:            symbol,
		Decimals:          uint8(decimals),
		InitialSupply:     supplyRaw,
		Owner:             owner,
		Recipient:         recipient,
		Flags:             features.Flags,
		MaxSupply:         maxSupply,
		MaxWallet:         maxWallet,
		MaxTx:             maxTx,
		AntiBotCooldown:   big.NewInt(cooldown),
		LiquidityTaxBps:   liqBps,
		DividendTaxBps:    divBps,
		BurnTaxBps:        burnBps,
		LiquidityWallet:   parseOptionalAddress(features.LiquidityWallet, owner),
		DividendWallet:    parseOptionalAddress(features.DividendWallet, owner),
		WalletTaxAccounts: walletAccounts,
		WalletTaxRates:    walletRates,
	}, nil
}

func pctOfSupply(supply *big.Int, pct float64) *big.Int {
	if pct <= 0 {
		return big.NewInt(0)
	}
	num := new(big.Int).Mul(supply, big.NewInt(int64(pct*100)))
	return num.Div(num, big.NewInt(10000))
}
