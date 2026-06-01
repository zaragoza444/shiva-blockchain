package wallet

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/onex-blockchain/onex/internal/crypto"
	"github.com/onex-blockchain/onex/internal/legacy"
	"github.com/onex-blockchain/onex/internal/types"
)

type Wallet struct {
	Address types.Address `json:"address"`
	Private string        `json:"privateKey"`
}

func Create(path string) (*Wallet, error) {
	pub, priv, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, err
	}
	w := &Wallet{Address: pub, Private: hex.EncodeToString(priv)}
	if path != "" {
		if err := Save(path, w); err != nil {
			return nil, err
		}
	}
	return w, nil
}

func Load(path string) (*Wallet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var w Wallet
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, err
	}
	return &w, nil
}

func Save(path string, w *Wallet) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (w *Wallet) PrivateKey() ([]byte, error) {
	return hex.DecodeString(w.Private)
}

func (w *Wallet) SignTransaction(tx *types.Transaction) error {
	priv, err := w.PrivateKey()
	if err != nil {
		return err
	}
	tx.From = w.Address
	sig, err := crypto.SignTx(priv, crypto.TxPayload(tx))
	if err != nil {
		return err
	}
	tx.Signature = sig
	return nil
}

func BuildTransfer(w *Wallet, to types.Address, amount, fee, nonce uint64) (*types.Transaction, error) {
	tx := &types.Transaction{
		To:     to,
		Amount: amount,
		Fee:    fee,
		Nonce:  nonce,
	}
	if err := w.SignTransaction(tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func ImportPrivate(hexKey string) (*Wallet, error) {
	priv, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, err
	}
	pub, err := crypto.PubFromPrivate(priv)
	if err != nil {
		return nil, err
	}
	return &Wallet{Address: pub, Private: hexKey}, nil
}

func DefaultWalletPath(name string) string {
	return filepath.Join(legacy.HomeDir(), "wallets", name+".json")
}

func FormatBalance(atomic uint64) string {
	const unit = 100000000
	whole := atomic / unit
	frac := atomic % unit
	return fmt.Sprintf("%d.%08d ONEX", whole, frac)
}
