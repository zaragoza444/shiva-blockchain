package bridge

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/onex-blockchain/onex/internal/bridge/amm"
	"github.com/onex-blockchain/onex/internal/bridge/tokenplatform"
	"github.com/onex-blockchain/onex/internal/types"
	"github.com/onex-blockchain/onex/internal/wallet"
)

type Bridge struct {
	mu       sync.RWMutex
	cfg      Config
	node     *NodeClient
	wallet   *wallet.Wallet
	reg      *Registry
	store    *PortfolioStore
	custom   *customTokenStore
	amm      *amm.Store
	platform *tokenplatform.Store
}

func New(cfg Config) *Bridge {
	return &Bridge{
		cfg:  cfg,
		node: NewNodeClient(cfg.NodeURL),
	}
}

func (b *Bridge) Config() Config {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.cfg
}

func (b *Bridge) Node() *NodeClient {
	return b.node
}

func (b *Bridge) LoadWallet(path string) error {
	w, err := wallet.Load(path)
	if err != nil {
		return err
	}
	b.mu.Lock()
	b.wallet = w
	if path != "" {
		b.cfg.WalletPath = path
	}
	b.mu.Unlock()
	return nil
}

func (b *Bridge) WalletAddress() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.wallet == nil {
		return ""
	}
	return string(b.wallet.Address)
}

func (b *Bridge) WalletBalance() (types.Address, uint64, uint64, error) {
	if err := b.EnsureWallet(); err != nil {
		return "", 0, 0, err
	}
	b.mu.RLock()
	addr := b.wallet.Address
	b.mu.RUnlock()
	bal, nonce, err := b.node.Balance(addr)
	return addr, bal, nonce, err
}

func (b *Bridge) EnsureWallet() error {
	b.mu.RLock()
	w := b.wallet
	path := b.cfg.WalletPath
	b.mu.RUnlock()
	if w != nil {
		return nil
	}
	if path == "" {
		return fmt.Errorf("no wallet configured")
	}
	return b.LoadWallet(path)
}

func (b *Bridge) Status() (map[string]interface{}, error) {
	out := map[string]interface{}{
		"nodeUrl":   b.cfg.NodeURL,
		"listen":    b.cfg.Listen,
		"nodeOk":    false,
		"wallet":    b.WalletAddress(),
		"walletPath": b.cfg.WalletPath,
	}
	st, err := b.node.Status()
	if err != nil {
		out["nodeError"] = err.Error()
		return out, nil
	}
	out["nodeOk"] = true
	out["chainId"] = st.ChainID
	out["networkId"] = st.NetworkID
	out["height"] = st.Height
	out["peers"] = st.Peers
	return out, nil
}

func (b *Bridge) Send(to types.Address, amount, fee uint64) (map[string]string, error) {
	if err := b.EnsureWallet(); err != nil {
		return nil, err
	}
	b.mu.RLock()
	w := b.wallet
	b.mu.RUnlock()
	_, nonce, err := b.node.Balance(w.Address)
	if err != nil {
		return nil, err
	}
	tx, err := wallet.BuildTransfer(w, to, amount, fee, nonce)
	if err != nil {
		return nil, err
	}
	if err := b.node.SubmitTx(tx); err != nil {
		return nil, err
	}
	return map[string]string{"status": "accepted", "from": string(w.Address)}, nil
}

func (b *Bridge) HandleWalletRPC(method string, params json.RawMessage) (interface{}, error) {
	switch method {
	case "onex_requestAccounts", "eth_requestAccounts":
		if err := b.EnsureWallet(); err != nil {
			return nil, err
		}
		return []string{b.WalletAddress()}, nil
	case "onex_accounts", "eth_accounts":
		addr := b.WalletAddress()
		if addr == "" {
			return []string{}, nil
		}
		return []string{addr}, nil
	case "onex_getBalance", "eth_getBalance":
		var args []string
		if err := json.Unmarshal(params, &args); err != nil || len(args) < 1 {
			return nil, fmt.Errorf("address required")
		}
		addr := types.Address(normalizeAddr(args[0]))
		bal, nonce, err := b.node.Balance(addr)
		if err != nil {
			return nil, err
		}
		if method == "eth_getBalance" {
			return fmt.Sprintf("0x%x", bal), nil
		}
		return map[string]interface{}{"balance": bal, "nonce": nonce}, nil
	case "onex_sendTransaction":
		if err := b.EnsureWallet(); err != nil {
			return nil, err
		}
		var tx types.Transaction
		if err := json.Unmarshal(params, &tx); err != nil {
			var wrap struct {
				Tx types.Transaction `json:"tx"`
			}
			if err2 := json.Unmarshal(params, &wrap); err2 != nil {
				return nil, fmt.Errorf("invalid tx")
			}
			tx = wrap.Tx
		}
		b.mu.RLock()
		w := b.wallet
		b.mu.RUnlock()
		if tx.Signature == "" {
			if err := w.SignTransaction(&tx); err != nil {
				return nil, err
			}
		}
		if err := b.node.SubmitTx(&tx); err != nil {
			return nil, err
		}
		return map[string]string{"status": "accepted"}, nil
	default:
		return nil, fmt.Errorf("wallet rpc not handled: %s", method)
	}
}

func normalizeAddr(s string) string {
	return strings.TrimPrefix(strings.TrimSpace(s), "0x")
}
