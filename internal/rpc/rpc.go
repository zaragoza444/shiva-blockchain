package rpc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/onex-blockchain/onex/internal/chain"
	"github.com/onex-blockchain/onex/internal/crypto"
	"github.com/onex-blockchain/onex/internal/mempool"
	"github.com/onex-blockchain/onex/internal/network"
	"github.com/onex-blockchain/onex/internal/types"
	"github.com/onex-blockchain/onex/internal/legacy"
)

type Handler struct {
	bc   *chain.Blockchain
	pool *mempool.Pool
	net  *network.Server
}

func New(bc *chain.Blockchain, pool *mempool.Pool, net *network.Server) *Handler {
	return &Handler{bc: bc, pool: pool, net: net}
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}
	if body[0] == '[' {
		var batch []request
		if err := json.Unmarshal(body, &batch); err != nil {
			writeErr(w, nil, -32700, err.Error())
			return
		}
		out := make([]response, 0, len(batch))
		for _, req := range batch {
			out = append(out, h.dispatch(req))
		}
		writeJSON(w, out)
		return
	}
	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		writeErr(w, nil, -32700, err.Error())
		return
	}
	writeJSON(w, h.dispatch(req))
}

func (h *Handler) dispatch(req request) response {
	if req.JSONRPC == "" {
		req.JSONRPC = "2.0"
	}
	res, err := h.call(req.Method, req.Params)
	if err != nil {
		return response{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32000, Message: err.Error()}}
	}
	return response{JSONRPC: "2.0", ID: req.ID, Result: res}
}

func (h *Handler) call(method string, params json.RawMessage) (interface{}, error) {
	method = legacy.NormalizeRPCMethod(method)
	switch method {
	case "onex_chainId":
		return h.bc.ChainID(), nil
	case "onex_getNetworkId", "net_version":
		return fmt.Sprintf("%d", h.bc.NetworkID()), nil
	case "eth_chainId":
		return fmt.Sprintf("0x%x", h.bc.NetworkID()), nil
	case "web3_clientVersion":
		return "OneX/1.0", nil
	case "onex_getBalance", "eth_getBalance":
		var args []string
		if err := json.Unmarshal(params, &args); err != nil || len(args) < 1 {
			return nil, fmt.Errorf("address required")
		}
		addr := normalizeAddress(args[0])
		if !addr.Valid() {
			return nil, fmt.Errorf("invalid address")
		}
		bal := h.bc.Balance(addr)
		if strings.HasPrefix(method, "eth_") {
			return fmt.Sprintf("0x%x", bal), nil
		}
		return map[string]interface{}{
			"address": addr,
			"balance": bal,
			"nonce":   h.bc.Nonce(addr),
		}, nil
	case "onex_getTransactionCount", "eth_getTransactionCount":
		var args []string
		if err := json.Unmarshal(params, &args); err != nil || len(args) < 1 {
			return nil, fmt.Errorf("address required")
		}
		addr := normalizeAddress(args[0])
		n := h.bc.Nonce(addr)
		if strings.HasPrefix(method, "eth_") {
			return fmt.Sprintf("0x%x", n), nil
		}
		return n, nil
	case "onex_sendTransaction", "eth_sendTransaction":
		var tx types.Transaction
		if err := json.Unmarshal(params, &tx); err == nil && tx.To != "" {
			return h.submitTx(&tx)
		}
		var wrapped struct {
			Tx types.Transaction `json:"tx"`
		}
		if err := json.Unmarshal(params, &wrapped); err != nil {
			return nil, fmt.Errorf("invalid transaction params")
		}
		return h.submitTx(&wrapped.Tx)
	case "onex_sendRawTransaction":
		var args []string
		if err := json.Unmarshal(params, &args); err != nil || len(args) < 1 {
			return nil, fmt.Errorf("signed tx json required")
		}
		var tx types.Transaction
		if err := json.Unmarshal([]byte(args[0]), &tx); err != nil {
			return nil, err
		}
		return h.submitTx(&tx)
	default:
		return nil, fmt.Errorf("method not found: %s", method)
	}
}

func (h *Handler) submitTx(tx *types.Transaction) (interface{}, error) {
	if err := h.bc.ValidateTx(tx); err != nil {
		return nil, err
	}
	h.pool.Add(*tx)
	if h.net != nil {
		h.net.BroadcastTx(*tx)
	}
	// pseudo tx hash for wallet UX
	hash := crypto.Hash(crypto.TxPayload(tx))
	return map[string]string{"txHash": hash, "status": "accepted"}, nil
}

func normalizeAddress(s string) types.Address {
	s = strings.TrimPrefix(strings.TrimSpace(s), "0x")
	return types.Address(strings.ToLower(s))
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, id json.RawMessage, code int, msg string) {
	writeJSON(w, response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: msg},
	})
}

// ParseAmount converts decimal ONEX string to atomic units (8 decimals).
func ParseAmount(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty amount")
	}
	parts := strings.SplitN(s, ".", 2)
	whole, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	frac := uint64(0)
	if len(parts) == 2 {
		fracStr := parts[1]
		if len(fracStr) > 8 {
			return 0, fmt.Errorf("too many decimal places")
		}
		for len(fracStr) < 8 {
			fracStr += "0"
		}
		frac, err = strconv.ParseUint(fracStr, 10, 64)
		if err != nil {
			return 0, err
		}
	}
	return whole*100000000 + frac, nil
}
