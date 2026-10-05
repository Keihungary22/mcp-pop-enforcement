package enforcement

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Keihungary22/mcp-pop-enforcement/internal/auth"
	"github.com/Keihungary22/mcp-pop-enforcement/internal/dpop"
	"github.com/golang-jwt/jwt/v5"
)

const (
	testIssuer      = "https://issuer.example"
	testAudience    = "mcp://resource"
	testScope       = "mcp:invoke"
	testExpectedHTU = "http://127.0.0.1:9100/mcp"
)

func generateTestKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}

	return key
}

func publicJWK(key *ecdsa.PrivateKey) dpop.JWK {
	return dpop.JWK{
		KTY: "EC",
		CRV: "P-256",
		X: base64.RawURLEncoding.EncodeToString(
			key.PublicKey.X.FillBytes(make([]byte, 32)),
		),
		Y: base64.RawURLEncoding.EncodeToString(
			key.PublicKey.Y.FillBytes(make([]byte, 32)),
		),
	}
}

func signAccessToken(
	t *testing.T,
	issuerKey *ecdsa.PrivateKey,
	jkt string,
	scope string,
	includeCNF bool,
) string {
	t.Helper()

	now := time.Now()

	claims := auth.Claims{
		Scope: scope,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testIssuer,
			Audience:  jwt.ClaimStrings{testAudience},
			IssuedAt:  jwt.NewNumericDate(now.Add(-time.Minute)),
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
		},
	}

	if includeCNF {
		claims.CNF = &auth.Confirmation{
			JKT: jkt,
		}
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodES256,
		claims,
	)

	signed, err := token.SignedString(issuerKey)
	if err != nil {
		t.Fatalf("sign access token: %v", err)
	}

	return signed
}

func accessTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func signDPoPProof(
	t *testing.T,
	clientKey *ecdsa.PrivateKey,
	accessToken string,
) string {
	t.Helper()

	now := time.Now()

	claims := dpop.ProofClaims{
		HTM: http.MethodPost,
		HTU: testExpectedHTU,
		ATH: accessTokenHash(accessToken),
		RegisteredClaims: jwt.RegisteredClaims{
			ID:       "proof-jti",
			IssuedAt: jwt.NewNumericDate(now.Add(-time.Second)),
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodES256,
		claims,
	)

	token.Header["typ"] = "dpop+jwt"
	token.Header["jwk"] = publicJWK(clientKey)

	signed, err := token.SignedString(clientKey)
	if err != nil {
		t.Fatalf("sign DPoP proof: %v", err)
	}

	return signed
}

func newMiddleware(
	t *testing.T,
	issuerKey *ecdsa.PrivateKey,
) *Middleware {
	t.Helper()

	tokenValidator := auth.NewValidator(
		&issuerKey.PublicKey,
		testIssuer,
		testAudience,
		testScope,
	)

	proofVerifier := dpop.NewVerifier(
		5*time.Minute,
		30*time.Second,
	)

	replayStore := dpop.NewMemoryReplayStore(
		5 * time.Minute,
	)

	return NewMiddleware(
		tokenValidator,
		proofVerifier,
		replayStore,
		testExpectedHTU,
	)
}

func executeRequest(
	t *testing.T,
	middleware *Middleware,
	authorization string,
	proof string,
) (int, bool) {
	t.Helper()

	called := false

	next := http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusNoContent)
		},
	)

	handler := middleware.Wrap(next)

	request := httptest.NewRequest(
		http.MethodPost,
		testExpectedHTU,
		nil,
	)

	if authorization != "" {
		request.Header.Set(
			"Authorization",
			authorization,
		)
	}

	if proof != "" {
		request.Header.Set("DPoP", proof)
	}

	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	return response.Code, called
}

func TestMiddlewareAllowsValidDPoPBoundRequest(t *testing.T) {
	issuerKey := generateTestKey(t)
	clientKey := generateTestKey(t)

	jkt, err := publicJWK(clientKey).Thumbprint()
	if err != nil {
		t.Fatalf("calculate JWK thumbprint: %v", err)
	}

	accessToken := signAccessToken(
		t,
		issuerKey,
		jkt,
		testScope,
		true,
	)

	proof := signDPoPProof(
		t,
		clientKey,
		accessToken,
	)

	status, called := executeRequest(
		t,
		newMiddleware(t, issuerKey),
		"DPoP "+accessToken,
		proof,
	)

	if status != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			status,
		)
	}

	if !called {
		t.Fatal("expected downstream handler to be called")
	}
}

func TestMiddlewareRejectsBearerAuthorization(t *testing.T) {
	issuerKey := generateTestKey(t)
	clientKey := generateTestKey(t)

	jkt, err := publicJWK(clientKey).Thumbprint()
	if err != nil {
		t.Fatalf("calculate JWK thumbprint: %v", err)
	}

	accessToken := signAccessToken(
		t,
		issuerKey,
		jkt,
		testScope,
		true,
	)

	status, called := executeRequest(
		t,
		newMiddleware(t, issuerKey),
		"Bearer "+accessToken,
		"",
	)

	if status != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			status,
		)
	}

	if called {
		t.Fatal("downstream handler must not be called")
	}
}

func TestMiddlewareRejectsMissingProof(t *testing.T) {
	issuerKey := generateTestKey(t)
	clientKey := generateTestKey(t)

	jkt, err := publicJWK(clientKey).Thumbprint()
	if err != nil {
		t.Fatalf("calculate JWK thumbprint: %v", err)
	}

	accessToken := signAccessToken(
		t,
		issuerKey,
		jkt,
		testScope,
		true,
	)

	status, called := executeRequest(
		t,
		newMiddleware(t, issuerKey),
		"DPoP "+accessToken,
		"",
	)

	if status != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			status,
		)
	}

	if called {
		t.Fatal("downstream handler must not be called")
	}
}

func TestMiddlewareRejectsTokenWithoutCNF(t *testing.T) {
	issuerKey := generateTestKey(t)
	clientKey := generateTestKey(t)

	accessToken := signAccessToken(
		t,
		issuerKey,
		"",
		testScope,
		false,
	)

	proof := signDPoPProof(
		t,
		clientKey,
		accessToken,
	)

	status, called := executeRequest(
		t,
		newMiddleware(t, issuerKey),
		"DPoP "+accessToken,
		proof,
	)

	if status != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			status,
		)
	}

	if called {
		t.Fatal("downstream handler must not be called")
	}
}

func TestMiddlewareRejectsInvalidProof(t *testing.T) {
	issuerKey := generateTestKey(t)
	clientKey := generateTestKey(t)

	jkt, err := publicJWK(clientKey).Thumbprint()
	if err != nil {
		t.Fatalf("calculate JWK thumbprint: %v", err)
	}

	accessToken := signAccessToken(
		t,
		issuerKey,
		jkt,
		testScope,
		true,
	)

	status, called := executeRequest(
		t,
		newMiddleware(t, issuerKey),
		"DPoP "+accessToken,
		"not-a-jwt",
	)

	if status != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			status,
		)
	}

	if called {
		t.Fatal("downstream handler must not be called")
	}
}

func TestMiddlewareRejectsKeyBindingMismatch(t *testing.T) {
	issuerKey := generateTestKey(t)

	boundKey := generateTestKey(t)
	proofKey := generateTestKey(t)

	boundJKT, err := publicJWK(boundKey).Thumbprint()
	if err != nil {
		t.Fatalf("calculate bound JWK thumbprint: %v", err)
	}

	accessToken := signAccessToken(
		t,
		issuerKey,
		boundJKT,
		testScope,
		true,
	)

	proof := signDPoPProof(
		t,
		proofKey,
		accessToken,
	)

	status, called := executeRequest(
		t,
		newMiddleware(t, issuerKey),
		"DPoP "+accessToken,
		proof,
	)

	if status != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			status,
		)
	}

	if called {
		t.Fatal("downstream handler must not be called")
	}
}

func TestMiddlewareRejectsInsufficientScope(t *testing.T) {
	issuerKey := generateTestKey(t)
	clientKey := generateTestKey(t)

	jkt, err := publicJWK(clientKey).Thumbprint()
	if err != nil {
		t.Fatalf("calculate JWK thumbprint: %v", err)
	}

	accessToken := signAccessToken(
		t,
		issuerKey,
		jkt,
		"mcp:read",
		true,
	)

	proof := signDPoPProof(
		t,
		clientKey,
		accessToken,
	)

	status, called := executeRequest(
		t,
		newMiddleware(t, issuerKey),
		"DPoP "+accessToken,
		proof,
	)

	if status != http.StatusForbidden {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusForbidden,
			status,
		)
	}

	if called {
		t.Fatal("downstream handler must not be called")
	}
}

func TestMiddlewareRejectsReplayedProof(t *testing.T) {
	issuerKey := generateTestKey(t)
	clientKey := generateTestKey(t)

	jkt, err := publicJWK(clientKey).Thumbprint()
	if err != nil {
		t.Fatalf("calculate JWK thumbprint: %v", err)
	}

	accessToken := signAccessToken(
		t,
		issuerKey,
		jkt,
		testScope,
		true,
	)

	proof := signDPoPProof(
		t,
		clientKey,
		accessToken,
	)

	middleware := newMiddleware(t, issuerKey)

	firstStatus, firstCalled := executeRequest(
		t,
		middleware,
		"DPoP "+accessToken,
		proof,
	)

	if firstStatus != http.StatusNoContent {
		t.Fatalf(
			"expected first status %d, got %d",
			http.StatusNoContent,
			firstStatus,
		)
	}

	if !firstCalled {
		t.Fatal("expected first request to reach downstream")
	}

	secondStatus, secondCalled := executeRequest(
		t,
		middleware,
		"DPoP "+accessToken,
		proof,
	)

	if secondStatus != http.StatusUnauthorized {
		t.Fatalf(
			"expected replay status %d, got %d",
			http.StatusUnauthorized,
			secondStatus,
		)
	}

	if secondCalled {
		t.Fatal("replayed proof must not reach downstream")
	}
}
