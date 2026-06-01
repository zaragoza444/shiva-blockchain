package chain

import (
	"fmt"
	"sync"

	"github.com/onex-blockchain/onex/internal/consensus"
	"github.com/onex-blockchain/onex/internal/crypto"
	"github.com/onex-blockchain/onex/internal/storage"
	"github.com/onex-blockchain/onex/internal/types"
)

type Blockchain struct {
	mu          sync.RWMutex
	store       *storage.Store
	state       types.ChainState
	genesis     *types.GenesisConfig
	blockReward uint64
}

func New(store *storage.Store, genesis *types.GenesisConfig) (*Blockchain, error) {
	bc := &Blockchain{
		store:       store,
		state:       make(types.ChainState),
		genesis:     genesis,
		blockReward: genesis.Reward,
	}
	if store.Height() == 0 {
		alloc := make(map[types.Address]uint64)
		for k, v := range genesis.Alloc {
			alloc[types.Address(k)] = v
		}
		gen := types.NewGenesisBlock(alloc, genesis.Difficulty, genesis.ChainID)
		gen.Header.StateRoot = ComputeStateRoot(bc.state)
		for _, tx := range gen.Transactions {
			_ = ApplyTransaction(bc.state, &tx, 0, "")
		}
		gen.Header.StateRoot = ComputeStateRoot(bc.state)
		gen.Hash = crypto.Hash(crypto.BlockHeaderPayload(&gen.Header))
		consensus.MineBlock(gen)
		if err := bc.store.PutBlock(gen); err != nil {
			return nil, err
		}
	} else {
		if err := bc.rebuildState(); err != nil {
			return nil, err
		}
	}
	return bc, nil
}

func (bc *Blockchain) rebuildState() error {
	bc.state = make(types.ChainState)
	return bc.store.Iterate(0, func(b *types.Block) error {
		return bc.applyBlockInternal(b, false)
	})
}

func (bc *Blockchain) Height() uint64 {
	return bc.store.Height()
}

func (bc *Blockchain) GetBlock(i uint64) (*types.Block, error) {
	return bc.store.GetBlock(i)
}

func (bc *Blockchain) GetTip() (*types.Block, error) {
	return bc.store.GetTip()
}

func (bc *Blockchain) State() types.ChainState {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.state.Clone()
}

func (bc *Blockchain) Balance(addr types.Address) uint64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.state[addr].Balance
}

func (bc *Blockchain) Nonce(addr types.Address) uint64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.state[addr].Nonce
}

func (bc *Blockchain) ChainID() string {
	return bc.genesis.ChainID
}

func (bc *Blockchain) NetworkID() uint64 {
	if bc.genesis.NetworkID == 0 {
		return 9001
	}
	return bc.genesis.NetworkID
}

func (bc *Blockchain) Difficulty() uint64 {
	return bc.genesis.Difficulty
}

func (bc *Blockchain) BlockReward() uint64 {
	return bc.blockReward
}

// FinalizeAndMineBlock sets state root after executing txs, then mines PoW.
func (bc *Blockchain) FinalizeAndMineBlock(txs []types.Transaction, miner types.Address) (*types.Block, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	tip, err := bc.store.GetTip()
	if err != nil {
		return nil, err
	}

	sim := bc.state.Clone()
	for _, tx := range txs {
		if err := ApplyTransaction(sim, &tx, bc.blockReward, miner); err != nil {
			return nil, err
		}
	}
	ApplyBlockReward(sim, miner, bc.blockReward)

	b := &types.Block{
		Header: types.BlockHeader{
			Index:        tip.Header.Index + 1,
			Timestamp:    tip.Header.Timestamp + 1,
			PreviousHash: tip.Hash,
			Difficulty:   bc.genesis.Difficulty,
			Miner:        miner,
		},
		Transactions: txs,
	}
	b.Header.StateRoot = ComputeStateRoot(sim)
	b.Hash = crypto.Hash(crypto.BlockHeaderPayload(&b.Header))
	consensus.MineBlock(b)

	if err := bc.applyBlockInternal(b, true); err != nil {
		return nil, err
	}
	return b, nil
}

// ApplyBlock validates and applies a block from sync (no mining).
func (bc *Blockchain) ApplyBlock(b *types.Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	return bc.applyBlockInternal(b, true)
}

func (bc *Blockchain) applyBlockInternal(b *types.Block, persist bool) error {
	tip, err := bc.store.GetTip()
	if err != nil {
		return err
	}
	if b.Header.Index != tip.Header.Index+1 && !(b.Header.Index == 0 && tip.Header.Index == 0 && b.Hash != tip.Hash) {
		if b.Header.Index <= tip.Header.Index {
			if b.Hash == tip.Hash {
				return nil
			}
			return fmt.Errorf("block index %d not next after %d", b.Header.Index, tip.Header.Index)
		}
	}
	if b.Header.Index > 0 {
		if b.Header.PreviousHash != tip.Hash {
			return fmt.Errorf("invalid previous hash")
		}
		if !consensus.ValidProof(b) || !consensus.HashMatchesHeader(b) {
			return fmt.Errorf("invalid proof of work")
		}
	}

	sim := bc.state.Clone()
	for _, tx := range b.Transactions {
		if err := ApplyTransaction(sim, &tx, bc.blockReward, b.Header.Miner); err != nil {
			return fmt.Errorf("tx apply: %w", err)
		}
	}
	if b.Header.Index > 0 {
		ApplyBlockReward(sim, b.Header.Miner, bc.blockReward)
	}
	root := ComputeStateRoot(sim)
	if b.Header.StateRoot != root {
		return fmt.Errorf("state root mismatch: got %s want %s", b.Header.StateRoot, root)
	}

	for _, tx := range b.Transactions {
		if err := ApplyTransaction(bc.state, &tx, bc.blockReward, b.Header.Miner); err != nil {
			return err
		}
	}
	if b.Header.Index > 0 {
		ApplyBlockReward(bc.state, b.Header.Miner, bc.blockReward)
	}

	if persist {
		return bc.store.PutBlock(b)
	}
	return nil
}

func (bc *Blockchain) ValidateTx(tx *types.Transaction) error {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	sim := bc.state.Clone()
	return ApplyTransaction(sim, tx, bc.blockReward, "")
}
