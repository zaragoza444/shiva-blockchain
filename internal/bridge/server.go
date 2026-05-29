package bridge

import (
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/shiva-blockchain/shiva/internal/rpc"
	"github.com/shiva-blockchain/shiva/internal/types"
	"github.com/shiva-blockchain/shiva/internal/wallet"
)

//go:embed static/wallet/*
var walletFS embed.FS

type Server struct {
	b *Bridge
}

func NewServer(b *Bridge) *Server {
	return &Server{b: b}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok", "service": "shiva-bridge"})
	})
	mux.HandleFunc("/bridge/status", s.handleStatus)
	mux.HandleFunc("/bridge/wallet/load", s.handleWalletLoad)
	mux.HandleFunc("/bridge/wallet/create", s.handleWalletCreate)
	mux.HandleFunc("/bridge/wallet/balance", s.handleWalletBalance)
	mux.HandleFunc("/bridge/wallet/send", s.handleWalletSend)
	mux.HandleFunc("/rpc", s.handleRPC)
	s.registerDeFiRoutes(mux)
	s.registerAIRoutes(mux)

	sub, _ := fs.Sub(walletFS, "static/wallet")
	mux.Handle("/wallet/", http.StripPrefix("/wallet/", http.FileServer(http.FS(sub))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/wallet/", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})
	return cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
	}))
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	st, _ := s.b.Status()
	writeJSON(w, st)
}

func (s *Server) handleWalletLoad(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	path := req.Path
	if path == "" {
		path = s.b.Config().WalletPath
	}
	if err := s.b.LoadWallet(path); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]string{"address": s.b.WalletAddress(), "path": path})
}

func (s *Server) handleWalletCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	path := s.b.Config().WalletPath
	wlt, err := wallet.Create(path)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	_ = s.b.LoadWallet(path)
	writeJSON(w, map[string]string{"address": string(wlt.Address), "path": path})
}

func (s *Server) handleWalletBalance(w http.ResponseWriter, r *http.Request) {
	addr, bal, nonce, err := s.b.WalletBalance()
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"address": addr, "balance": bal, "nonce": nonce})
}

func (s *Server) handleWalletSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		To     string `json:"to"`
		Amount string `json:"amount"`
		Fee    string `json:"fee"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	amount, err := rpc.ParseAmount(req.Amount)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	feeStr := req.Fee
	if feeStr == "" {
		feeStr = "0.001"
	}
	fee, err := rpc.ParseAmount(feeStr)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	res, err := s.b.Send(types.Address(strings.ToLower(normalizeAddr(req.To))), amount, fee)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, res)
}

func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	var req struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
		ID     interface{}     `json:"id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	walletMethods := strings.HasPrefix(req.Method, "shiva_") ||
		req.Method == "eth_requestAccounts" || req.Method == "eth_accounts"
	if walletMethods {
		result, err := s.b.HandleWalletRPC(req.Method, req.Params)
		if err != nil {
			writeJSON(w, map[string]interface{}{
				"jsonrpc": "2.0", "id": req.ID,
				"error": map[string]interface{}{"code": -32000, "message": err.Error()},
			})
			return
		}
		writeJSON(w, map[string]interface{}{"jsonrpc": "2.0", "id": req.ID, "result": result})
		return
	}
	out, code, err := s.b.node.ProxyRPC(body)
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"jsonrpc": "2.0", "id": req.ID,
			"error": map[string]interface{}{"code": -32000, "message": err.Error()},
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(out)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) Start(addr string) error {
	log.Printf("shiva-bridge: wallet UI http://127.0.0.1%s/wallet/", addr)
	log.Printf("shiva-bridge: JSON-RPC http://127.0.0.1%s/rpc -> %s", addr, s.b.Config().NodeURL)
	return http.ListenAndServe(addr, s.Handler())
}
