package main

import (
	"log"
	"net/http"
	"time"

	"github.com/Keihungary22/mcp-pop-enforcement/internal/auth"
	"github.com/Keihungary22/mcp-pop-enforcement/internal/config"
	"github.com/Keihungary22/mcp-pop-enforcement/internal/dpop"
	"github.com/Keihungary22/mcp-pop-enforcement/internal/enforcement"
	"github.com/Keihungary22/mcp-pop-enforcement/internal/gateway"
)

func main() {
	cfg := config.Load()

	var tokenValidator auth.TokenValidator

	if cfg.AccessTokenJWKSURL != "" {
		resolver := auth.NewJWKSResolver(
			cfg.AccessTokenJWKSURL,
		)

		tokenValidator = auth.NewJWKSValidator(
			resolver,
			cfg.ExpectedIssuer,
			cfg.ExpectedAudience,
			cfg.RequiredScope,
		)

		log.Printf(
			"access-token validation mode: JWKS (%s)",
			cfg.AccessTokenJWKSURL,
		)
	} else {
		if cfg.AccessTokenPublicKeyFile == "" {
			log.Fatal(
				"ACCESS_TOKEN_JWKS_URL or ACCESS_TOKEN_PUBLIC_KEY_FILE is required",
			)
		}

		publicKey, err := auth.LoadECDSAPublicKey(
			cfg.AccessTokenPublicKeyFile,
		)
		if err != nil {
			log.Fatalf(
				"load access-token public key: %v",
				err,
			)
		}

		tokenValidator = auth.NewValidator(
			publicKey,
			cfg.ExpectedIssuer,
			cfg.ExpectedAudience,
			cfg.RequiredScope,
		)

		log.Printf(
			"access-token validation mode: static public key",
		)
	}

	proofVerifier := dpop.NewVerifier(
		5*time.Minute,
		30*time.Second,
	)

	replayStore := dpop.NewMemoryReplayStore(
		5 * time.Minute,
	)

	enforcementMiddleware := enforcement.NewMiddleware(
		tokenValidator,
		proofVerifier,
		replayStore,
		cfg.ExpectedDPoPHTU,
	)

	mcpGateway, err := gateway.New(cfg.UpstreamMCPURL)
	if err != nil {
		log.Fatalf("create MCP gateway: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(
		w http.ResponseWriter,
		_ *http.Request,
	) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.Handle(
		"/mcp",
		enforcementMiddleware.Wrap(mcpGateway),
	)

	log.Printf(
		"PoP enforcement gateway listening on %s, upstream=%s",
		cfg.ListenAddr,
		cfg.UpstreamMCPURL,
	)

	if err := http.ListenAndServe(
		cfg.ListenAddr,
		mux,
	); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
