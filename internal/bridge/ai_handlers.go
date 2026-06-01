package bridge

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/onex-blockchain/onex/internal/ai"
)

func (s *Server) registerAIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/bridge/ai/status", s.handleAIStatus)
	mux.HandleFunc("/bridge/ai/chat", s.handleAIChat)
}

func (s *Server) handleAIStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	a := ai.NewAssistant()
	writeJSON(w, a.Status())
}

func (s *Server) handleAIChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req ai.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Context == "" {
		req.Context = s.buildWalletAIContext()
	}
	out := ai.NewAssistant().Chat(req)
	writeJSON(w, out)
}

func (s *Server) buildWalletAIContext() string {
	var b strings.Builder
	st, _ := s.b.Status()
	if st != nil {
		data, _ := json.Marshal(st)
		b.WriteString("bridge_status: ")
		b.Write(data)
		b.WriteByte('\n')
	}
	p, err := s.b.GetPortfolio()
	if err == nil && p != nil {
		data, _ := json.Marshal(map[string]interface{}{
			"address":  p.Address,
			"balances": p.Balances,
			"stakes":   len(p.Stakes),
			"loans":    len(p.Loans),
			"nfts":     len(p.NFTs),
			"tasks":    len(p.Tasks),
		})
		b.WriteString("portfolio: ")
		b.Write(data)
	}
	return b.String()
}
