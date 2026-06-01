package chain

import (
	"fmt"
	"sort"

	"github.com/onex-blockchain/onex/internal/crypto"
	"github.com/onex-blockchain/onex/internal/types"
)

func ComputeStateRoot(state types.ChainState) string {
	if len(state) == 0 {
		return crypto.Hash([]byte("empty"))
	}
	addrs := make([]string, 0, len(state))
	for addr := range state {
		addrs = append(addrs, string(addr))
	}
	sort.Strings(addrs)
	var parts []byte
	for _, addr := range addrs {
		acc := state[types.Address(addr)]
		parts = append(parts, []byte(fmt.Sprintf("%s:%d:%d;", addr, acc.Balance, acc.Nonce))...)
	}
	return crypto.Hash(parts)
}

func ApplyTransaction(state types.ChainState, tx *types.Transaction, blockReward uint64, miner types.Address) error {
	if tx.From == "" {
		// coinbase / genesis allocation
		acc := state[tx.To]
		acc.Balance += tx.Amount
		state[tx.To] = acc
		return nil
	}
	from := state[tx.From]
	if from.Balance < tx.Amount+tx.Fee {
		return fmt.Errorf("insufficient balance")
	}
	if from.Nonce != tx.Nonce {
		return fmt.Errorf("invalid nonce")
	}
	if !crypto.VerifyTx(tx.From, crypto.TxPayload(tx), tx.Signature) {
		return fmt.Errorf("invalid signature")
	}
	from.Balance -= tx.Amount + tx.Fee
	from.Nonce++
	state[tx.From] = from

	to := state[tx.To]
	to.Balance += tx.Amount
	state[tx.To] = to

	if tx.Fee > 0 && miner != "" {
		m := state[miner]
		m.Balance += tx.Fee
		state[miner] = m
	}
	return nil
}

func ApplyBlockReward(state types.ChainState, miner types.Address, reward uint64) {
	if miner == "" || reward == 0 {
		return
	}
	m := state[miner]
	m.Balance += reward
	state[miner] = m
}
