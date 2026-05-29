package api

import (
	"encoding/json"
	"net/http"

	"github.com/shiva-blockchain/shiva/internal/ai"
)

func (s *Server) handleAIStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, ai.NewAssistant().Status())
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
		peers := 0
		if s.net != nil {
			peers = s.net.PeerCount()
		}
		mempool := 0
		if s.pool != nil {
			mempool = s.pool.Len()
		}
		req.Context = ai.BuildChainContext(
			s.bc.ChainID(),
			s.bc.NetworkID(),
			s.bc.Height(),
			peers,
			s.mining,
			mempool,
		)
	}
	writeJSON(w, ai.NewAssistant().Chat(req))
}
