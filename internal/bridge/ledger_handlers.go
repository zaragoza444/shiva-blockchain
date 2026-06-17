package bridge

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/onex-blockchain/onex/internal/ledger"
)

func (s *Server) registerLedgerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/bridge/ledger/status", s.handleLedgerStatus)
	mux.HandleFunc("/bridge/ledger/real", s.handleLedgerReal)
	mux.HandleFunc("/bridge/ledger/read", s.handleLedgerRead)
	mux.HandleFunc("/bridge/ledger/convert", s.handleLedgerConvert)
	mux.HandleFunc("/bridge/ledger/import", s.handleLedgerImport)
	// Legacy Shiva paths
	mux.HandleFunc("/bridge/shiva-ledger/status", s.handleLedgerStatus)
	mux.HandleFunc("/bridge/shiva-ledger/real", s.handleLedgerReal)
	mux.HandleFunc("/bridge/shiva-ledger/read", s.handleLedgerRead)
	mux.HandleFunc("/bridge/shiva-ledger/convert", s.handleLedgerConvert)
	mux.HandleFunc("/bridge/shiva-ledger/import", s.handleLedgerImport)
}

func (s *Server) handleLedgerStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.b.LedgerStatus())
}

func (s *Server) handleLedgerReal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	evm := r.URL.Query().Get("evm")
	snap, err := s.b.ReadRealLedger(r.Context(), "all", evm, s.b.LoadLatestImport())
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, snap)
}

func (s *Server) handleLedgerRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	source := r.URL.Query().Get("source")
	evm := r.URL.Query().Get("evm")
	snap, err := s.b.ReadRealLedger(r.Context(), source, evm, s.b.LoadLatestImport())
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, snap)
}

func (s *Server) handleLedgerConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req ledger.ConvertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := s.b.ConvertLedger(req)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, res)
}

func (s *Server) handleLedgerImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	path, err := s.b.SaveLedgerImport(body)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	snap, _ := s.b.ReadRealLedger(r.Context(), "import", "", body)
	writeJSON(w, map[string]interface{}{
		"status":   "imported",
		"path":     path,
		"entries":  len(snap.Entries),
		"snapshot": snap,
	})
}
