package main

import (
	"log"
	"net/http"

	"github.com/Keihungary22/mcp-pop-enforcement/internal/config"
	"github.com/Keihungary22/mcp-pop-enforcement/internal/gateway"
)

func main() {
	cfg := config.Load()

	mcpGateway, err := gateway.New(cfg.UpstreamMCPURL)
	if err != nil {
		log.Fatalf("create MCP gateway: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.Handle("/mcp", mcpGateway)

	log.Printf(
		"PoP enforcement gateway listening on %s, upstream=%s",
		cfg.ListenAddr,
		cfg.UpstreamMCPURL,
	)

	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
