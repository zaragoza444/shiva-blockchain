package api

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/onex-blockchain/onex/internal/chain"
	"github.com/onex-blockchain/onex/internal/faucet"
	"github.com/onex-blockchain/onex/internal/mempool"
	"github.com/onex-blockchain/onex/internal/network"
	"github.com/onex-blockchain/onex/internal/rpc"
	"github.com/onex-blockchain/onex/internal/types"
)

//go:embed static/explorer/*
var explorerFS embed.FS

type Server struct {
	addr       string
	bc         *chain.Blockchain
	pool       *mempool.Pool
	net        *network.Server
	faucet     *faucet.Service
	mining     bool
	noExplorer bool
	tlsCert    string
	tlsKey     string
	mw         *middleware
	enableRPC  bool
}

func New(addr string, bc *chain.Blockchain, pool *mempool.Pool, net *network.Server, faucet *faucet.Service, mining, noExplorer bool, tlsCert, tlsKey string, corsOrigins []string, apiKey string, enableRPC bool) *Server {
	return &Server{
		addr: addr, bc: bc, pool: pool, net: net, faucet: faucet,
		mining: mining, noExplorer: noExplorer, tlsCert: tlsCert, tlsKey: tlsKey,
		mw: newMiddleware(corsOrigins, apiKey), enableRPC: enableRPC,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)
	mux.HandleFunc("/api/v1/status", s.handleStatus)
	mux.HandleFunc("/api/v1/peers", s.handlePeers)
	mux.HandleFunc("/api/v1/blocks", s.handleBlocks)
	mux.HandleFunc("/api/v1/block/", s.handleBlock)
	mux.HandleFunc("/api/v1/balance/", s.handleBalance)
	mux.HandleFunc("/api/v1/tx", s.handleSubmitTx)
	mux.HandleFunc("/api/v1/faucet", s.handleFaucet)
	mux.HandleFunc("/api/v1/chain", s.handleChainMeta)
	mux.HandleFunc("/api/v1/ai/status", s.handleAIStatus)
	mux.HandleFunc("/api/v1/ai/chat", s.handleAIChat)

	if s.enableRPC {
		rpcHandler := rpc.New(s.bc, s.pool, s.net)
		mux.Handle("/rpc", rpcHandler)
	}

	if !s.noExplorer {
		sub, _ := fs.Sub(explorerFS, "static/explorer")
		mux.Handle("/explorer/", http.StripPrefix("/explorer/", http.FileServer(http.FS(sub))))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				http.Redirect(w, r, "/explorer/", http.StatusFound)
				return
			}
			http.NotFound(w, r)
		})
	}

	handler := s.mw.wrap(mux)
	log.Printf("api: listening on %s (rpc=%v)", s.addr, s.enableRPC)
	if s.tlsCert != "" && s.tlsKey != "" {
		return http.ListenAndServeTLS(s.addr, s.tlsCert, s.tlsKey, handler)
	}
	return http.ListenAndServe(s.addr, handler)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if _, err := s.bc.GetTip(); err != nil {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, map[string]string{"status": "ready"})
}

func (s *Server) handleChainMeta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"chainId":   s.bc.ChainID(),
		"networkId": s.bc.NetworkID(),
		"decimals":  types.CoinDecimals,
		"symbol":    "ONEX",
		"rpcMethods": []string{
			"onex_chainId", "onex_getBalance", "onex_getTransactionCount",
			"onex_sendTransaction", "eth_chainId", "eth_getBalance",
		},
		"wallet": map[string]interface{}{
			"type":        "ed25519",
			"addressLen":  64,
			"metaMask":    false,
			"useOneXWallet": true,
			"addChain": map[string]interface{}{
				"chainId":             fmtHexChainID(s.bc.NetworkID()),
				"chainName":           s.bc.ChainID(),
				"nativeCurrency":      map[string]string{"name": "OneX", "symbol": "ONEX", "decimals": "8"},
				"rpcUrls":             []string{"/rpc"},
				"blockExplorerUrls":   []string{"/explorer/"},
			},
		},
	})
}

func fmtHexChainID(id uint64) string {
	return "0x" + strconv.FormatUint(id, 16)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	tip, _ := s.bc.GetTip()
	hash := ""
	if tip != nil {
		hash = tip.Hash
	}
	peers := 0
	if s.net != nil {
		peers = s.net.PeerCount()
	}
	rpcURL := "/rpc"
	if r.Host != "" {
		scheme := "http"
		if r.TLS != nil || s.tlsCert != "" {
			scheme = "https"
		}
		rpcURL = scheme + "://" + r.Host + "/rpc"
	}
	writeJSON(w, types.APIStatus{
		ChainID:   s.bc.ChainID(),
		NetworkID: s.bc.NetworkID(),
		Height:    s.bc.Height(),
		Hash:      hash,
		Peers:     peers,
		Mining:    s.mining,
		RPCURL:    rpcURL,
	})
}

func (s *Server) handlePeers(w http.ResponseWriter, r *http.Request) {
	if s.net == nil {
		writeJSON(w, []types.PeerInfo{})
		return
	}
	writeJSON(w, s.net.Peers())
}

func (s *Server) handleBlocks(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			limit = n
		}
	}
	height := s.bc.Height()
	type blockSummary struct {
		Index   uint64 `json:"index"`
		Hash    string `json:"hash"`
		TxCount int    `json:"txCount"`
		Miner   string `json:"miner"`
	}
	var out []blockSummary
	for i := height; i > 0 && len(out) < limit; i-- {
		b, err := s.bc.GetBlock(i)
		if err != nil {
			break
		}
		out = append(out, blockSummary{
			Index: b.Header.Index, Hash: b.Hash,
			TxCount: len(b.Transactions), Miner: string(b.Header.Miner),
		})
	}
	if len(out) < limit {
		if b, err := s.bc.GetBlock(0); err == nil {
			out = append(out, blockSummary{
				Index: 0, Hash: b.Hash, TxCount: len(b.Transactions),
			})
		}
	}
	writeJSON(w, out)
}

func (s *Server) handleBlock(w http.ResponseWriter, r *http.Request) {
	idxStr := strings.TrimPrefix(r.URL.Path, "/api/v1/block/")
	idx, err := strconv.ParseUint(idxStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid index", 400)
		return
	}
	b, err := s.bc.GetBlock(idx)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}
	writeJSON(w, b)
}

func (s *Server) handleBalance(w http.ResponseWriter, r *http.Request) {
	addr := types.Address(strings.TrimPrefix(r.URL.Path, "/api/v1/balance/"))
	writeJSON(w, map[string]interface{}{
		"address": addr,
		"balance": s.bc.Balance(addr),
		"nonce":   s.bc.Nonce(addr),
	})
}

func (s *Server) handleSubmitTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var tx types.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := s.bc.ValidateTx(&tx); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	s.pool.Add(tx)
	if s.net != nil {
		s.net.BroadcastTx(tx)
	}
	writeJSON(w, map[string]string{"status": "accepted"})
}

func (s *Server) handleFaucet(w http.ResponseWriter, r *http.Request) {
	if s.faucet == nil {
		http.Error(w, "faucet disabled", 503)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req struct {
		Address string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	addr := types.Address(req.Address)
	if !addr.Valid() {
		writeJSON(w, map[string]string{"error": "invalid address"})
		return
	}
	if err := s.faucet.Drip(addr); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]string{"message": "faucet drip sent"})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
