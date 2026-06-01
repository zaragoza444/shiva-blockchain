package mempool

import (
	"fmt"
	"sync"

	"github.com/onex-blockchain/onex/internal/types"
)

type Pool struct {
	mu   sync.Mutex
	txs  map[string]types.Transaction
	keys []string
}

func New() *Pool {
	return &Pool{txs: make(map[string]types.Transaction)}
}

func txKey(tx types.Transaction) string {
	return fmt.Sprintf("%s:%d", tx.From, tx.Nonce)
}

func (p *Pool) Add(tx types.Transaction) {
	p.mu.Lock()
	defer p.mu.Unlock()
	k := txKey(tx)
	if _, ok := p.txs[k]; ok {
		return
	}
	p.txs[k] = tx
	p.keys = append(p.keys, k)
}

func (p *Pool) Pending(max int) []types.Transaction {
	p.mu.Lock()
	defer p.mu.Unlock()
	if max <= 0 || max > len(p.keys) {
		max = len(p.keys)
	}
	out := make([]types.Transaction, 0, max)
	for i := 0; i < max; i++ {
		out = append(out, p.txs[p.keys[i]])
	}
	return out
}

func (p *Pool) RemoveIncluded(txs []types.Transaction) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, tx := range txs {
		k := txKey(tx)
		delete(p.txs, k)
	}
	keys := make([]string, 0, len(p.txs))
	for k := range p.txs {
		keys = append(keys, k)
	}
	p.keys = keys
}

func (p *Pool) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.txs)
}
