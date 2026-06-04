package bridge

import (
	"encoding/json"
	"net/http"
)

func (s *Server) registerPlatformRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/bridge/platform/status", s.handlePlatformStatus)
	mux.HandleFunc("/bridge/platform/tokens", s.handlePlatformTokens)
	mux.HandleFunc("/bridge/platform/token", s.handlePlatformTokenDetail)
	mux.HandleFunc("/bridge/platform/deploy", s.handlePlatformDeploy)
	mux.HandleFunc("/bridge/platform/wrap", s.handlePlatformWrap)
	mux.HandleFunc("/bridge/platform/wraps", s.handlePlatformWraps)
}

func (s *Server) handlePlatformStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	st, err := s.b.PlatformStatus()
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, st)
}

func (s *Server) handlePlatformTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	list, err := s.b.ListPlatformTokens()
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, list)
}

func (s *Server) handlePlatformTokenDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	chainID := r.URL.Query().Get("chain")
	tokenID := r.URL.Query().Get("id")
	if chainID == "" || tokenID == "" {
		http.Error(w, "chain and id required", http.StatusBadRequest)
		return
	}
	tok, err := s.b.GetPlatformToken(chainID, tokenID)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, tok)
}

func (s *Server) handlePlatformDeploy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ChainID  string `json:"chainId"`
		Name     string `json:"name"`
		Symbol   string `json:"symbol"`
		Decimals int    `json:"decimals"`
		Supply   string `json:"supply"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.ChainID == "" {
		req.ChainID = "onex-mainnet-1"
	}
	if req.Decimals == 0 {
		req.Decimals = 8
	}
	tok, err := s.b.DeployPlatformToken(req.ChainID, req.Name, req.Symbol, req.Decimals, req.Supply)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, tok)
}

func (s *Server) handlePlatformWrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		OriginChainID string `json:"originChainId"`
		OriginTokenID string `json:"originTokenId"`
		TargetChainID string `json:"targetChainId"`
		Amount        string `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rec, wrapped, err := s.b.WrapPlatformToken(req.OriginChainID, req.OriginTokenID, req.TargetChainID, req.Amount)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{
		"wrap":    rec,
		"wrapped": wrapped,
	})
}

func (s *Server) handlePlatformWraps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	list, err := s.b.ListWrapRecords()
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, list)
}
