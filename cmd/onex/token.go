package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

const defaultBridge = "http://127.0.0.1:9338"

func bridgeURL(flagVal string) string {
	if v := strings.TrimSpace(flagVal); v != "" {
		return strings.TrimRight(v, "/")
	}
	if v := os.Getenv("ONEX_BRIDGE_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return defaultBridge
}

func bridgePost(base, path string, body interface{}) (map[string]interface{}, int) {
	data, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, base+path, bytes.NewReader(data))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if k := strings.TrimSpace(os.Getenv("ONEX_API_KEY")); k != "" {
		req.Header.Set("X-OneX-Api-Key", k)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]interface{}
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		out = map[string]interface{}{"error": string(raw)}
	}
	return out, resp.StatusCode
}

func bridgeGet(base, path string) (interface{}, int) {
	resp, err := http.Get(base + path)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return string(raw), resp.StatusCode
	}
	return out, resp.StatusCode
}

func runTokenCreate(args []string) {
	fs := flag.NewFlagSet("token-create", flag.ExitOnError)
	chain := fs.String("chain", "onex-mainnet-1", "chain id")
	name := fs.String("name", "", "token name")
	symbol := fs.String("symbol", "", "token symbol")
	decimals := fs.Int("decimals", 8, "decimals")
	supply := fs.String("supply", "", "total supply")
	bridge := fs.String("bridge", "", "bridge URL")
	_ = fs.Parse(args)
	if *name == "" || *symbol == "" || *supply == "" {
		log.Fatal("usage: onex token-create -name NAME -symbol SYM -supply AMOUNT [-chain CHAIN] [-decimals N] [-bridge URL]")
	}
	base := bridgeURL(*bridge)
	out, code := bridgePost(base, "/bridge/platform/deploy", map[string]interface{}{
		"chainId": *chain, "name": *name, "symbol": *symbol,
		"decimals": *decimals, "supply": *supply,
	})
	printJSON(out, code)
}

func runTokenList(args []string) {
	fs := flag.NewFlagSet("token-list", flag.ExitOnError)
	bridge := fs.String("bridge", "", "bridge URL")
	_ = fs.Parse(args)
	out, code := bridgeGet(bridgeURL(*bridge), "/bridge/platform/tokens")
	printJSON(out, code)
}

func runTokenWrap(args []string) {
	fs := flag.NewFlagSet("token-wrap", flag.ExitOnError)
	fromChain := fs.String("from-chain", "", "origin chain id")
	fromToken := fs.String("from-token", "", "origin token id")
	toChain := fs.String("to-chain", "", "target chain id")
	amount := fs.String("amount", "", "amount to wrap")
	bridge := fs.String("bridge", "", "bridge URL")
	_ = fs.Parse(args)
	if *fromChain == "" || *fromToken == "" || *toChain == "" || *amount == "" {
		log.Fatal("usage: onex token-wrap -from-chain CHAIN -from-token ID -to-chain CHAIN -amount AMT [-bridge URL]")
	}
	base := bridgeURL(*bridge)
	out, code := bridgePost(base, "/bridge/platform/wrap", map[string]string{
		"originChainId": *fromChain, "originTokenId": *fromToken,
		"targetChainId": *toChain, "amount": *amount,
	})
	printJSON(out, code)
}

func runTokenInfo(args []string) {
	fs := flag.NewFlagSet("token-info", flag.ExitOnError)
	chain := fs.String("chain", "", "chain id")
	id := fs.String("id", "", "token id")
	bridge := fs.String("bridge", "", "bridge URL")
	_ = fs.Parse(args)
	if *chain == "" || *id == "" {
		log.Fatal("usage: onex token-info -chain CHAIN -id TOKEN_ID [-bridge URL]")
	}
	path := fmt.Sprintf("/bridge/platform/token?chain=%s&id=%s", *chain, *id)
	out, code := bridgeGet(bridgeURL(*bridge), path)
	printJSON(out, code)
}

func runPlatformStatus(args []string) {
	fs := flag.NewFlagSet("platform-status", flag.ExitOnError)
	bridge := fs.String("bridge", "", "bridge URL")
	_ = fs.Parse(args)
	out, code := bridgeGet(bridgeURL(*bridge), "/bridge/platform/status")
	printJSON(out, code)
}

func printJSON(v interface{}, code int) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
	if code >= 400 {
		os.Exit(1)
	}
}
