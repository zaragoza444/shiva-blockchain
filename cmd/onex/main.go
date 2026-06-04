package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/onex-blockchain/onex/internal/rpc"
	"github.com/onex-blockchain/onex/internal/types"
	"github.com/onex-blockchain/onex/internal/wallet"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "wallet-create":
		runWalletCreate(os.Args[2:])
	case "balance":
		runBalance(os.Args[2:])
	case "send":
		runSend(os.Args[2:])
	case "token-create":
		runTokenCreate(os.Args[2:])
	case "token-list":
		runTokenList(os.Args[2:])
	case "token-wrap":
		runTokenWrap(os.Args[2:])
	case "token-info":
		runTokenInfo(os.Args[2:])
	case "platform-status":
		runPlatformStatus(os.Args[2:])
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`OneX CLI

  onex wallet-create [-out PATH]
  onex balance -address HEX [-api URL]
  onex send -wallet PATH -to HEX -amount DECIMAL [-fee DECIMAL] [-api URL]

Token Platform (requires onex-bridge running):
  onex token-create -name NAME -symbol SYM -supply AMT [-chain CHAIN] [-decimals N] [-bridge URL]
  onex token-list [-bridge URL]
  onex token-wrap -from-chain CHAIN -from-token ID -to-chain CHAIN -amount AMT [-bridge URL]
  onex token-info -chain CHAIN -id TOKEN_ID [-bridge URL]
  onex platform-status [-bridge URL]`)
}

func runWalletCreate(args []string) {
	fs := flag.NewFlagSet("wallet-create", flag.ExitOnError)
	out := fs.String("out", "", "wallet JSON path")
	_ = fs.Parse(args)

	path := *out
	if path == "" {
		path = wallet.DefaultWalletPath("default")
	}
	w, err := wallet.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Wallet created\n  address: %s\n  file: %s\n", w.Address, path)
}

func runBalance(args []string) {
	fs := flag.NewFlagSet("balance", flag.ExitOnError)
	addr := fs.String("address", "", "address (hex)")
	api := fs.String("api", "http://127.0.0.1:8545", "node API base URL")
	_ = fs.Parse(args)
	if *addr == "" {
		log.Fatal("usage: onex balance -address HEX")
	}
	url := strings.TrimRight(*api, "/") + "/api/v1/balance/" + strings.TrimSpace(*addr)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		Balance uint64 `json:"balance"`
		Nonce   uint64 `json:"nonce"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Balance: %s\nNonce: %d\n", wallet.FormatBalance(out.Balance), out.Nonce)
}

func runSend(args []string) {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	walletPath := fs.String("wallet", "", "wallet JSON file")
	to := fs.String("to", "", "recipient address")
	amountStr := fs.String("amount", "", "amount in ONEX e.g. 1.5")
	feeStr := fs.String("fee", "0.001", "fee in ONEX")
	api := fs.String("api", "http://127.0.0.1:8545", "node API base URL")
	_ = fs.Parse(args)
	if *walletPath == "" || *to == "" || *amountStr == "" {
		log.Fatal("usage: onex send -wallet PATH -to HEX -amount DECIMAL")
	}
	w, err := wallet.Load(*walletPath)
	if err != nil {
		log.Fatal(err)
	}
	amount, err := rpc.ParseAmount(*amountStr)
	if err != nil {
		log.Fatal(err)
	}
	fee, err := rpc.ParseAmount(*feeStr)
	if err != nil {
		log.Fatal(err)
	}
	balURL := strings.TrimRight(*api, "/") + "/api/v1/balance/" + string(w.Address)
	resp, err := http.Get(balURL)
	if err != nil {
		log.Fatal(err)
	}
	var bal struct {
		Nonce uint64 `json:"nonce"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&bal)
	resp.Body.Close()

	tx, err := wallet.BuildTransfer(w, types.Address(strings.ToLower(*to)), amount, fee, bal.Nonce)
	if err != nil {
		log.Fatal(err)
	}
	body, _ := json.Marshal(tx)
	txURL := strings.TrimRight(*api, "/") + "/api/v1/tx"
	res, err := http.Post(txURL, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]string
	_ = json.NewDecoder(res.Body).Decode(&out)
	if res.StatusCode >= 400 {
		log.Fatalf("submit failed: %v", out)
	}
	fmt.Printf("Transaction submitted: %s\n", out["status"])
}
