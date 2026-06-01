package chain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/onex-blockchain/onex/internal/config"
	"github.com/onex-blockchain/onex/internal/crypto"
	"github.com/onex-blockchain/onex/internal/storage"
	"github.com/onex-blockchain/onex/internal/types"
)

func TestFinalizeAndMineBlock(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "chaindata")
	store, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	genesis := &types.GenesisConfig{
		ChainID:    "test",
		Difficulty: 2,
		Alloc:      map[string]uint64{},
		Reward: 100,
	}
	bc, err := New(store, genesis)
	if err != nil {
		t.Fatal(err)
	}
	miner, priv, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	_ = priv
	block, err := bc.FinalizeAndMineBlock(nil, miner)
	if err != nil {
		t.Fatal(err)
	}
	if block.Header.Index != 1 {
		t.Fatalf("expected index 1, got %d", block.Header.Index)
	}
	if block.Hash == "" {
		t.Fatal("empty block hash")
	}
}

func TestApplyBlockSync(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "chaindata")
	store, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	genesis := &types.GenesisConfig{
		ChainID:    "test",
		Difficulty: 2,
		Alloc: map[string]uint64{
			"a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890": 1000,
		},
		Reward: 50,
	}
	bc1, err := New(store, genesis)
	if err != nil {
		t.Fatal(err)
	}
	miner, _, _ := crypto.GenerateKeyPair()
	mined, err := bc1.FinalizeAndMineBlock(nil, miner)
	if err != nil {
		t.Fatal(err)
	}

	dir2 := filepath.Join(t.TempDir(), "chaindata2")
	store2, _ := storage.Open(dir2)
	bc2, err := New(store2, genesis)
	if err != nil {
		t.Fatal(err)
	}
	gen, _ := bc2.GetBlock(0)
	if err := bc2.ApplyBlock(gen); err != nil {
		t.Fatal(err)
	}
	if err := bc2.ApplyBlock(mined); err != nil {
		t.Fatalf("apply synced block: %v", err)
	}
	if bc2.Height() != 1 {
		t.Fatalf("height %d", bc2.Height())
	}
}

func TestGenesisFromConfig(t *testing.T) {
	path := filepath.Join("..", "..", "configs", "genesis.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("genesis.json not in test cwd")
	}
	g, err := config.LoadGenesis(path)
	if err != nil {
		t.Fatal(err)
	}
	if g.ChainID == "" {
		t.Fatal("empty chain id")
	}
}
