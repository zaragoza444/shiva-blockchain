package main

import (
	"fmt"
	"strings"
)

type Chain struct {
	Slug              string `json:"slug"`
	Name              string `json:"name"`
	ChainID           int64  `json:"chainId"`
	RPCURL            string `json:"rpcUrl"`
	Explorer          string `json:"explorer"`
	NativeSymbol      string `json:"nativeSymbol"`
	DexChainID        string `json:"dexChainId"`
	TokenType         string `json:"tokenType"`
	LiquiditySupported bool  `json:"liquiditySupported"`
	Live              bool  `json:"live"`
}

func supportedChains() []Chain {
	return []Chain{
		{Slug: "bsc", Name: "BNB Smart Chain", ChainID: 56, RPCURL: "https://bsc-dataseed.binance.org", Explorer: "https://bscscan.com", NativeSymbol: "BNB", DexChainID: "bsc", TokenType: "erc20", LiquiditySupported: true, Live: true},
		{Slug: "eth", Name: "Ethereum", ChainID: 1, RPCURL: "https://ethereum-rpc.publicnode.com", Explorer: "https://etherscan.io", NativeSymbol: "ETH", DexChainID: "ethereum", TokenType: "erc20", Live: true},
		{Slug: "base", Name: "Base", ChainID: 8453, RPCURL: "https://mainnet.base.org", Explorer: "https://basescan.org", NativeSymbol: "ETH", DexChainID: "base", TokenType: "erc20", Live: true},
		{Slug: "polygon", Name: "Polygon", ChainID: 137, RPCURL: "https://polygon-rpc.com", Explorer: "https://polygonscan.com", NativeSymbol: "MATIC", DexChainID: "polygon", TokenType: "erc20", Live: true},
		{Slug: "arbitrum", Name: "Arbitrum One", ChainID: 42161, RPCURL: "https://arb1.arbitrum.io/rpc", Explorer: "https://arbiscan.io", NativeSymbol: "ETH", DexChainID: "arbitrum", TokenType: "erc20", Live: true},
		{Slug: "optimism", Name: "Optimism", ChainID: 10, RPCURL: "https://mainnet.optimism.io", Explorer: "https://optimistic.etherscan.io", NativeSymbol: "ETH", DexChainID: "optimism", TokenType: "erc20", Live: true},
		{Slug: "avalanche", Name: "Avalanche C-Chain", ChainID: 43114, RPCURL: "https://api.avax.network/ext/bc/C/rpc", Explorer: "https://snowtrace.io", NativeSymbol: "AVAX", DexChainID: "avalanche", TokenType: "erc20", Live: true},
		{Slug: "linea", Name: "Linea", ChainID: 59144, RPCURL: "https://rpc.linea.build", Explorer: "https://lineascan.build", NativeSymbol: "ETH", DexChainID: "linea", TokenType: "erc20", Live: true},
		{Slug: "blast", Name: "Blast", ChainID: 81457, RPCURL: "https://rpc.blast.io", Explorer: "https://blastscan.io", NativeSymbol: "ETH", DexChainID: "blast", TokenType: "erc20", Live: true},
		{Slug: "scroll", Name: "Scroll", ChainID: 534352, RPCURL: "https://rpc.scroll.io", Explorer: "https://scrollscan.com", NativeSymbol: "ETH", DexChainID: "scroll", TokenType: "erc20", Live: true},
		{Slug: "spl", Name: "Solana (SPL)", ChainID: 0, RPCURL: "", Explorer: "https://solscan.io", NativeSymbol: "SOL", DexChainID: "solana", TokenType: "spl", Live: true},
		{Slug: "sui", Name: "Sui", ChainID: 0, RPCURL: "", Explorer: "https://suiscan.xyz", NativeSymbol: "SUI", DexChainID: "sui", TokenType: "sui", Live: true},
	}
}

func chainBySlug(slug string) (Chain, error) {
	slug = normalizeChainSlug(slug)
	for _, c := range supportedChains() {
		if c.Slug == slug {
			return c, nil
		}
	}
	return Chain{}, fmt.Errorf("unsupported chain: %s", slug)
}

func chainByID(id int64) (Chain, error) {
	for _, c := range supportedChains() {
		if c.ChainID == id {
			return c, nil
		}
	}
	return Chain{}, fmt.Errorf("unsupported chain id: %d", id)
}

func normalizeChainSlug(slug string) string {
	switch slug {
	case "", "bnb", "bsc":
		return "bsc"
	case "ethereum", "mainnet":
		return "eth"
	case "matic":
		return "polygon"
	case "arb":
		return "arbitrum"
	case "op":
		return "optimism"
	case "avax":
		return "avalanche"
	default:
		return slug
	}
}

func deployChain(slug string) (Chain, error) {
	c, err := chainBySlug(slug)
	if err != nil {
		return c, err
	}
	if c.ChainID == 0 || c.RPCURL == "" {
		return Chain{}, fmt.Errorf("%s deploy uses EVM — pick BSC, Ethereum, Base, or Polygon in the wizard", c.Name)
	}
	return c, nil
}

func defaultChain(cfg Config) Chain {
	c, err := chainByID(cfg.ChainID)
	if err != nil {
		c, _ = chainBySlug("bsc")
	}
	if cfg.RPCURL != "" && c.Slug == "bsc" {
		c.RPCURL = cfg.RPCURL
	}
	if cfg.Explorer != "" && c.Slug == "bsc" {
		c.Explorer = cfg.Explorer
	}
	return c
}

func chainIDFromDex(dex string) int64 {
	for _, c := range supportedChains() {
		if strings.EqualFold(c.DexChainID, dex) || strings.EqualFold(c.Slug, dex) {
			return c.ChainID
		}
	}
	return 0
}

func explorerTokenURL(explorer, token string) string {
	return strings.TrimRight(explorer, "/") + "/token/" + token
}

func explorerTxURL(explorer, hash string) string {
	return strings.TrimRight(explorer, "/") + "/tx/" + hash
}

func resolveChainSlug(slugs ...string) string {
	for _, slug := range slugs {
		if s := normalizeChainSlug(slug); s != "" {
			return s
		}
	}
	return "bsc"
}

func readyProbeAddress(chain Chain) string {
	switch chain.ChainID {
	case 56:
		return "0x55d398326f99059fF775485246999027B3197955"
	case 1, 8453, 42161, 10, 59144, 81457, 534352:
		return "0xdAC17F958D2ee523a2206206994597C13D831ec7"
	case 137:
		return "0xc2132D05D31c914a87C6611C10748AEb04B58e8F"
	case 43114:
		return "0x9702230A8Ea53601f5cD2dc00fDBc13d4dF4A8c7"
	default:
		return "0x55d398326f99059fF775485246999027B3197955"
	}
}
