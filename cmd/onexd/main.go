package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/onex-blockchain/onex/internal/config"
	"github.com/onex-blockchain/onex/internal/legacy"
	"github.com/onex-blockchain/onex/internal/node"
	"github.com/onex-blockchain/onex/internal/types"
)

func main() {
	dataDir := flag.String("datadir", "", "data directory")
	genesis := flag.String("genesis", "", "genesis JSON path")
	listen := flag.String("listen", ":30303", "P2P listen address")
	api := flag.String("api", ":8545", "HTTP API address")
	mine := flag.Bool("mine", false, "enable mining")
	miner := flag.String("miner", "", "miner address (hex)")
	seed := flag.String("seed", "", "seed peer host:port")
	seedsFile := flag.String("seeds", "", "JSON file with seed peers")
	tlsCert := flag.String("tls-cert", "", "TLS certificate file")
	tlsKey := flag.String("tls-key", "", "TLS private key file")
	noExplorer := flag.Bool("no-explorer", false, "disable block explorer UI")
	faucet := flag.Bool("faucet", false, "enable testnet faucet")
	cors := flag.String("cors", "", "CORS allowed origins (comma-separated, default *)")
	apiKey := flag.String("api-key", "", "API key for POST /api/v1/tx and /rpc (or ONEX_API_KEY)")
	noRPC := flag.Bool("no-rpc", false, "disable JSON-RPC /rpc endpoint")
	flag.Parse()

	cfg := &config.NodeConfig{
		DataDir:      config.DefaultDataDir(),
		GenesisPath:  config.ResolveGenesis(*genesis),
		ListenAddr:   *listen,
		APIAddr:      *api,
		Mine:         *mine,
		NoExplorer:   *noExplorer,
		FaucetEnable: *faucet,
		TLSCert:      *tlsCert,
		TLSKey:       *tlsKey,
	}
	if *dataDir != "" {
		cfg.DataDir = *dataDir
	}
	if *miner != "" {
		cfg.MinerAddr = types.Address(*miner)
	}
	if *seed != "" {
		cfg.Seed = *seed
	}
	if *seedsFile != "" {
		s, err := config.LoadSeeds(*seedsFile)
		if err != nil {
			log.Fatalf("seeds file: %v", err)
		}
		cfg.Seeds = s
	}
	cfg.FaucetKey = legacy.EnvOrLegacy("ONEX_FAUCET_PRIVATE_KEY", "SHIVA_FAUCET_PRIVATE_KEY")
	cfg.CORSOrigins = config.ParseCORSOrigins(legacy.EnvOrLegacy("ONEX_CORS_ORIGINS", "SHIVA_CORS_ORIGINS"))
	if *cors != "" {
		cfg.CORSOrigins = config.ParseCORSOrigins(*cors)
	}
	cfg.APIKey = *apiKey
	if cfg.APIKey == "" {
		cfg.APIKey = legacy.EnvOrLegacy("ONEX_API_KEY", "SHIVA_API_KEY")
	}
	cfg.EnableRPC = !*noRPC

	n, err := node.New(cfg)
	if err != nil {
		log.Fatalf("node: %v", err)
	}

	go func() {
		if err := n.Run(); err != nil {
			log.Fatalf("api: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("shutting down...")
	n.Stop()
}
