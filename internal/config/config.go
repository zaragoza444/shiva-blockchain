package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/onex-blockchain/onex/internal/types"
	"github.com/onex-blockchain/onex/internal/legacy"
)

func init() {
	legacy.EnsureHomeMigrated()
}

type NodeConfig struct {
	DataDir      string
	GenesisPath  string
	ListenAddr   string
	APIAddr      string
	Mine         bool
	MinerAddr    types.Address
	Seed         string
	Seeds        []string
	TLSCert      string
	TLSKey       string
	NoExplorer   bool
	ChainID      string
	Difficulty   uint64
	BlockReward  uint64
	FaucetEnable bool
	FaucetKey    string
	CORSOrigins  []string
	APIKey       string
	EnableRPC    bool
}

func LoadGenesis(path string) (*types.GenesisConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var g types.GenesisConfig
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, err
	}
	if g.Difficulty == 0 {
		g.Difficulty = 3
	}
	if g.Reward == 0 {
		g.Reward = 50 * 100000000 // 50 ONEX
	}
	if g.NetworkID == 0 {
		if strings.Contains(strings.ToLower(g.ChainID), "testnet") {
			g.NetworkID = 9002
		} else {
			g.NetworkID = 9001
		}
	}
	return &g, nil
}

func LoadSeeds(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var seeds []string
	if err := json.Unmarshal(data, &seeds); err != nil {
		return nil, err
	}
	return seeds, nil
}

func DefaultDataDir() string {
	return legacy.HomeDir()
}

func ResolveGenesis(genesisFlag string) string {
	if genesisFlag != "" {
		return genesisFlag
	}
	for _, p := range []string{"configs/genesis.json", "../configs/genesis.json"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "configs/genesis.json"
}

func GenesisAlloc(g *types.GenesisConfig) map[types.Address]uint64 {
	out := make(map[types.Address]uint64, len(g.Alloc))
	for k, v := range g.Alloc {
		out[types.Address(k)] = v
	}
	return out
}

func ParseCORSOrigins(env string) []string {
	if env == "" {
		return []string{"*"}
	}
	var out []string
	for _, p := range strings.Split(env, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func Validate(cfg *NodeConfig) error {
	if cfg.DataDir == "" {
		cfg.DataDir = DefaultDataDir()
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":30303"
	}
	if cfg.APIAddr == "" {
		cfg.APIAddr = ":8545"
	}
	if cfg.Mine && cfg.MinerAddr == "" {
		return fmt.Errorf("mining enabled but no miner address (-miner)")
	}
	return nil
}
