package faucet

import (
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/onex-blockchain/onex/internal/types"
	"github.com/onex-blockchain/onex/internal/wallet"
)

type Service struct {
	mu       sync.Mutex
	wallet   *wallet.Wallet
	amount   uint64
	cooldown time.Duration
	last     map[types.Address]time.Time
	sendFn   func(*types.Transaction) error
	nonceFn  func(types.Address) uint64
}

func New(w *wallet.Wallet, amount uint64, cooldown time.Duration, send func(*types.Transaction) error, nonce func(types.Address) uint64) *Service {
	return &Service{
		wallet:   w,
		amount:   amount,
		cooldown: cooldown,
		last:     make(map[types.Address]time.Time),
		sendFn:   send,
		nonceFn:  nonce,
	}
}

func FromEnvKey(hexKey string, amount uint64) (*wallet.Wallet, error) {
	return wallet.ImportPrivate(hexKey)
}

func (f *Service) Drip(to types.Address) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t, ok := f.last[to]; ok && time.Since(t) < f.cooldown {
		return fmt.Errorf("cooldown active, try again later")
	}
	nonce := f.nonceFn(f.wallet.Address)
	tx, err := wallet.BuildTransfer(f.wallet, to, f.amount, 0, nonce)
	if err != nil {
		return err
	}
	if err := f.sendFn(tx); err != nil {
		return err
	}
	f.last[to] = time.Now()
	return nil
}

func LoadWalletFromHex(hexKey string) (*wallet.Wallet, error) {
	key := hexKey
	if len(key) > 0 && key[:2] != "0x" {
		if _, err := hex.DecodeString(key); err != nil {
			return nil, err
		}
	}
	return wallet.ImportPrivate(key)
}
