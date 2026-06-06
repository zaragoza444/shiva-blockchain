package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	cfg := LoadConfig()
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatal(err)
	}

	srv := NewServer(cfg)
	mux := http.NewServeMux()

	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/ready", srv.handleReady)
	mux.HandleFunc("/api/config", srv.handleConfig)
	mux.HandleFunc("/api/deploy", srv.handleDeploy)
	mux.HandleFunc("/api/tokens/register", srv.handleRegister)
	mux.HandleFunc("/api/tokens/", srv.routeTokens)
	mux.HandleFunc("/api/tokens", srv.handleTokens)
	mux.HandleFunc("/api/bscscan/", srv.handleBSCScan)
	mux.HandleFunc("/api/price/", srv.handlePrice)
	mux.HandleFunc("/api/market/bnb", srv.handleMarketBNB)
	mux.HandleFunc("/api/liquidity/quote", srv.handleLiquidityQuote)
	mux.HandleFunc("/api/liquidity/pair", srv.handleLiquidityPair)
	mux.HandleFunc("/api/liquidity/register", srv.handleLiquidityRegister)
	mux.HandleFunc("/api/liquidity", srv.handleLiquidityList)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		serveStatic(w, r, cfg.WebDir, cfg.Env)
	})

	handler := chain(
		mux,
		withMaxBody(cfg.MaxBodyBytes),
		srv.withAPIAuth,
		withSecurityHeaders,
		withCORS(cfg.CORSOrigins),
		withRequestLog,
	)

	httpSrv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("BSC Token Launcher [%s] listening on %s", cfg.Env, cfg.Listen)
	if cfg.IsProduction() {
		log.Printf("production mode: apiKey=%t cors=%d origins backend=%t bscscan=%t",
			cfg.APIKey != "", len(cfg.CORSOrigins), cfg.DeployerKey != "", cfg.BSCScanAPIKey != "")
	} else {
		log.Printf("dev mode: http://127.0.0.1%s", strings.TrimPrefix(cfg.Listen, ":"))
	}

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
	log.Println("stopped")
}

func (s *Server) routeTokens(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/tokens/")
	if path == "" || path == "/" {
		s.handleTokens(w, r)
		return
	}
	s.handleTokenDetail(w, r)
}

func serveStatic(w http.ResponseWriter, r *http.Request, webDir, env string) {
	if r.URL.Path == "/" {
		http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
		return
	}
	clean := filepath.Clean(r.URL.Path)
	if strings.Contains(clean, "..") {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	full := filepath.Join(webDir, strings.TrimPrefix(clean, "/"))
	if _, err := os.Stat(full); err != nil {
		http.NotFound(w, r)
		return
	}
	if strings.HasSuffix(full, ".js") || strings.HasSuffix(full, ".css") {
		if env != "production" {
			w.Header().Set("Cache-Control", "no-cache")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=3600")
		}
	}
	http.ServeFile(w, r, full)
}
