package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	def := defaultChain(s.cfg)
	writeJSON(w, map[string]interface{}{
		"status":  "ok",
		"service": "onex-token-lab",
		"env":     s.cfg.Env,
		"chainId": def.ChainID,
		"chains":  len(supportedChains()),
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	def := defaultChain(s.cfg)
	checks := map[string]string{"store": "ok", "rpc": "ok", "web": "ok"}
	status := http.StatusOK

	if _, err := s.store.Load(); err != nil {
		checks["store"] = err.Error()
		status = http.StatusServiceUnavailable
	}
	probe := readyProbeAddress(def)
	if _, err := s.isContractOn(ctx, def.RPCURL, probe); err != nil {
		checks["rpc"] = err.Error()
		status = http.StatusServiceUnavailable
	}
	if contractBytecodeHex() == "" {
		checks["web"] = "contract artifacts missing"
		status = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  map[bool]string{true: "ready", false: "degraded"}[status == http.StatusOK],
		"checks":  checks,
		"backend": s.cfg.DeployerKey != "",
		"bscscan": s.cfg.BSCScanAPIKey != "",
		"env":     s.cfg.Env,
	})
}
