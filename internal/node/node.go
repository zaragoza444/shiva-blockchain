package node

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/onex-blockchain/onex/internal/api"
	"github.com/onex-blockchain/onex/internal/chain"
	"github.com/onex-blockchain/onex/internal/config"
	"github.com/onex-blockchain/onex/internal/faucet"
	"github.com/onex-blockchain/onex/internal/mempool"
	"github.com/onex-blockchain/onex/internal/network"
	"github.com/onex-blockchain/onex/internal/storage"
	"github.com/onex-blockchain/onex/internal/types"
)

type Node struct {
	cfg    *config.NodeConfig
	bc     *chain.Blockchain
	pool   *mempool.Pool
	net    *network.Server
	api    *api.Server
	faucet *faucet.Service
	cancel context.CancelFunc
}

func New(cfg *config.NodeConfig) (*Node, error) {
	if err := config.Validate(cfg); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, err
	}

	genesis, err := config.LoadGenesis(cfg.GenesisPath)
	if err != nil {
		return nil, fmt.Errorf("genesis: %w", err)
	}
	cfg.ChainID = genesis.ChainID
	cfg.Difficulty = genesis.Difficulty

	store, err := storage.Open(filepath.Join(cfg.DataDir, "chain"))
	if err != nil {
		return nil, err
	}
	bc, err := chain.New(store, genesis)
	if err != nil {
		return nil, err
	}

	pool := mempool.New()
	peerID := randomPeerID()
	netSrv := network.NewServer(cfg.ListenAddr, bc, genesis.ChainID, peerID)

	n := &Node{cfg: cfg, bc: bc, pool: pool, net: netSrv}

	netSrv.OnTransaction(func(tx types.Transaction) {
		if err := bc.ValidateTx(&tx); err == nil {
			pool.Add(tx)
		}
	})

	seeds := cfg.Seeds
	if cfg.Seed != "" {
		seeds = append([]string{cfg.Seed}, seeds...)
	}
	if len(seeds) == 0 {
		for _, p := range []string{
			filepath.Join("configs", "seeds.json"),
			filepath.Join("configs", "seeds-mainnet.json"),
		} {
			if s, err := config.LoadSeeds(p); err == nil {
				seeds = s
				break
			}
		}
	}

	if cfg.FaucetEnable && cfg.FaucetKey != "" {
		w, err := faucet.LoadWalletFromHex(cfg.FaucetKey)
		if err != nil {
			return nil, fmt.Errorf("faucet key: %w", err)
		}
		n.faucet = faucet.New(w, 10*100000000, time.Minute, func(tx *types.Transaction) error {
			pool.Add(*tx)
			netSrv.BroadcastTx(*tx)
			return nil
		}, bc.Nonce)
	}

	enableRPC := cfg.EnableRPC
	n.api = api.New(cfg.APIAddr, bc, pool, netSrv, n.faucet, cfg.Mine, cfg.NoExplorer, cfg.TLSCert, cfg.TLSKey, cfg.CORSOrigins, cfg.APIKey, enableRPC)

	ctx, cancel := context.WithCancel(context.Background())
	n.cancel = cancel

	if err := netSrv.Start(ctx); err != nil {
		cancel()
		return nil, err
	}
	netSrv.Bootstrap(ctx, seeds)

	if cfg.Mine {
		go n.mineLoop(ctx)
	}

	return n, nil
}

func (n *Node) Run() error {
	log.Printf("onexd: chain=%s height=%d data=%s", n.bc.ChainID(), n.bc.Height(), n.cfg.DataDir)
	return n.api.Start()
}

func (n *Node) Stop() {
	if n.cancel != nil {
		n.cancel()
	}
}

func (n *Node) mineLoop(ctx context.Context) {
	miner := n.cfg.MinerAddr
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			txs := n.pool.Pending(100)
			block, err := n.bc.FinalizeAndMineBlock(txs, miner)
			if err != nil {
				log.Printf("mine: %v", err)
				continue
			}
			n.pool.RemoveIncluded(txs)
			n.net.BroadcastBlock(block)
			log.Printf("mined block %d hash=%s", block.Header.Index, block.Hash[:16])
		}
	}
}

func randomPeerID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
