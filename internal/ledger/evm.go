package ledger

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"
)

// EVMChain is minimal chain metadata for on-chain balance reads.
type EVMChain struct {
	ID   string
	RPC  string
	Type string
}

// knownContracts maps chainId:symbol -> ERC-20 contract (mainnet).
var knownContracts = map[string]string{
	"ethereum:USDT":  "0xdAC17F958D2ee523a2206206994597C13D831ec7",
	"ethereum:USDC":  "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
	"ethereum:WBTC":  "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599",
	"bsc:USDT":       "0x55d398326f99059fF775485246999027B3197955",
	"bsc:USDT-BSC":   "0x55d398326f99059fF775485246999027B3197955",
	"polygon:USDC":   "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174",
	"polygon:USDC-POLY": "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174",
}

// ReadEVMBalance fetches real native or ERC-20 balance from chain RPC.
func ReadEVMBalance(ctx context.Context, chain EVMChain, holder, symbol string, decimals int, contract string) (string, error) {
	if chain.RPC == "" || chain.Type != "evm" {
		return "", fmt.Errorf("no evm rpc for %s", chain.ID)
	}
	holder = normalizeEVMAddr(holder)
	if holder == "" {
		return "", fmt.Errorf("evm address required")
	}
	if contract == "" {
		contract = knownContracts[chain.ID+":"+strings.ToUpper(symbol)]
	}
	if contract != "" {
		return erc20Balance(ctx, chain.RPC, contract, holder)
	}
	return nativeBalance(ctx, chain.RPC, holder)
}

func ReadEVMEntries(ctx context.Context, chains []EVMChain, holder string, tokens []TokenMeta) ([]Entry, error) {
	if holder == "" {
		return nil, nil
	}
	var out []Entry
	now := time.Now().Unix()
	for _, t := range tokens {
		chain := findChain(chains, t.ChainID)
		if chain == nil || chain.Type != "evm" || chain.RPC == "" {
			continue
		}
		atomic, err := ReadEVMBalance(ctx, *chain, holder, t.Symbol, t.Decimals, "")
		if err != nil || atomic == "" || atomic == "0" {
			continue
		}
		human := atomicToHumanStr(atomic, t.Decimals)
		out = append(out, Entry{
			ID:        fmt.Sprintf("evm-%s-%s", t.ChainID, t.TokenID),
			Source:    SourceEVM,
			Mode:      ModeReal,
			ChainID:   t.ChainID,
			Asset:     t.Symbol,
			TokenKey:  t.ChainID + ":" + t.TokenID,
			Atomic:    atomic,
			Human:     human,
			Account:   holder,
			Timestamp: now,
			Reference: "eth_rpc",
		})
	}
	return out, nil
}

func findChain(chains []EVMChain, id string) *EVMChain {
	for i := range chains {
		if chains[i].ID == id {
			return &chains[i]
		}
	}
	return nil
}

func nativeBalance(ctx context.Context, rpcURL, holder string) (string, error) {
	var result string
	err := ethCall(ctx, rpcURL, "eth_getBalance", []interface{}{toHexAddr(holder), "latest"}, &result)
	if err != nil {
		return "", err
	}
	return hexToDecimal(result)
}

func erc20Balance(ctx context.Context, rpcURL, contract, holder string) (string, error) {
	// balanceOf(address) selector 0x70a08231
	addr := strings.TrimPrefix(normalizeEVMAddr(holder), "0x")
	if len(addr) != 40 {
		return "", fmt.Errorf("invalid holder address")
	}
	data := "0x70a08231" + strings.Repeat("0", 24) + addr
	var result string
	err := ethCall(ctx, rpcURL, "eth_call", []interface{}{
		map[string]string{"to": toHexAddr(contract), "data": data},
		"latest",
	}, &result)
	if err != nil {
		return "", err
	}
	return hexToDecimal(result)
}

func ethCall(ctx context.Context, rpcURL, method string, params []interface{}, out interface{}) error {
	body, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	var wrap struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return err
	}
	if wrap.Error != nil {
		return fmt.Errorf("rpc: %s", wrap.Error.Message)
	}
	return json.Unmarshal(wrap.Result, out)
}

func hexToDecimal(hexStr string) (string, error) {
	hexStr = strings.TrimSpace(hexStr)
	if hexStr == "" || hexStr == "0x" || hexStr == "0x0" {
		return "0", nil
	}
	hexStr = strings.TrimPrefix(hexStr, "0x")
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", err
	}
	n := new(big.Int).SetBytes(b)
	return n.String(), nil
}

func atomicToHumanStr(atomic string, decimals int) string {
	n, ok := new(big.Int).SetString(strings.TrimSpace(atomic), 10)
	if !ok {
		return "0"
	}
	if decimals <= 0 {
		return n.String()
	}
	div := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	whole := new(big.Int).Div(n, div)
	rem := new(big.Int).Mod(n, div)
	if rem.Sign() == 0 {
		return whole.String()
	}
	frac := rem.String()
	for len(frac) < decimals {
		frac = "0" + frac
	}
	frac = strings.TrimRight(frac, "0")
	if frac == "" {
		return whole.String()
	}
	return whole.String() + "." + frac
}

func normalizeEVMAddr(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if !strings.HasPrefix(s, "0x") {
		s = "0x" + s
	}
	return strings.ToLower(s)
}

func toHexAddr(s string) string {
	s = normalizeEVMAddr(s)
	if s == "" {
		return s
	}
	return s
}
