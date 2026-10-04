package dpop

import (
	"crypto/ecdsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testAccessToken = "test-access-token"
	testMethod      = "POST"
	testHTU         = "https://api.example.com/mcp"
)

func validProofClaims(now time.Time) ProofClaims {
	return ProofClaims{
		HTM: testMethod,
		HTU: testHTU,
		ATH: AccessTokenHash(testAccessToken),
		RegisteredClaims: jwt.RegisteredClaims{
			ID:       "proof-jti-123",
			IssuedAt: jwt.NewNumericDate(now.Add(-time.Minute)),
		},
	}
}

func signTestProof(
	t *testing.T,
	signingKey *ecdsa.PrivateKey,
	headerJWK JWK,
	claims ProofClaims,
	typ string,
	includeJWK bool,
) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["typ"] = typ

	if includeJWK {
		token.Header["jwk"] = headerJWK
	}

	signed, err := token.SignedString(signingKey)
	if err != nil {
		t.Fatalf("sign DPoP proof: %v", err)
	}

	return signed
}

func newTestVerifier(now time.Time) *Verifier {
	verifier := NewVerifier(
		5*time.Minute,
		30*time.Second,
	)

	verifier.now = func() time.Time {
		return now
	}

	return verifier
}

func TestVerifierAcceptsValidProof(t *testing.T) {
	now := time.Now()
	key := generateDPoPTestKey(t)
	jwk := jwkFromPrivateKey(key)

	proof := signTestProof(
		t,
		key,
		jwk,
		validProofClaims(now),
		"dpop+jwt",
		true,
	)

	verifier := newTestVerifier(now)

	result, err := verifier.Verify(
		proof,
		testAccessToken,
		testMethod,
		testHTU+"?ignored=true",
	)
	if err != nil {
		t.Fatalf("Verify() returned error: %v", err)
	}

	expectedJKT, err := jwk.Thumbprint()
	if err != nil {
		t.Fatalf("calculate expected thumbprint: %v", err)
	}

	if result.JKT != expectedJKT {
		t.Fatalf(
			"expected JKT %s, got %s",
			expectedJKT,
			result.JKT,
		)
	}

	if result.JTI != "proof-jti-123" {
		t.Fatalf("unexpected JTI: %s", result.JTI)
	}
}

func TestVerifierRejectsInvalidProofs(t *testing.T) {
	now := time.Now()

	key := generateDPoPTestKey(t)
	attackerKey := generateDPoPTestKey(t)
	jwk := jwkFromPrivateKey(key)

	tests := []struct {
		name       string
		mutate     func(*ProofClaims)
		signingKey *ecdsa.PrivateKey
		headerJWK  JWK
		typ        string
		includeJWK bool
	}{
		{
			name:       "wrong typ",
			signingKey: key,
			headerJWK:  jwk,
			typ:        "JWT",
			includeJWK: true,
		},
		{
			name:       "missing jwk",
			signingKey: key,
			headerJWK:  jwk,
			typ:        "dpop+jwt",
			includeJWK: false,
		},
		{
			name:       "invalid signature",
			signingKey: attackerKey,
			headerJWK:  jwk,
			typ:        "dpop+jwt",
			includeJWK: true,
		},
		{
			name: "wrong htm",
			mutate: func(claims *ProofClaims) {
				claims.HTM = "GET"
			},
			signingKey: key,
			headerJWK:  jwk,
			typ:        "dpop+jwt",
			includeJWK: true,
		},
		{
			name: "wrong htu",
			mutate: func(claims *ProofClaims) {
				claims.HTU = "https://other.example.com/mcp"
			},
			signingKey: key,
			headerJWK:  jwk,
			typ:        "dpop+jwt",
			includeJWK: true,
		},
		{
			name: "stale iat",
			mutate: func(claims *ProofClaims) {
				claims.IssuedAt = jwt.NewNumericDate(
					now.Add(-10 * time.Minute),
				)
			},
			signingKey: key,
			headerJWK:  jwk,
			typ:        "dpop+jwt",
			includeJWK: true,
		},
		{
			name: "future iat",
			mutate: func(claims *ProofClaims) {
				claims.IssuedAt = jwt.NewNumericDate(
					now.Add(time.Minute),
				)
			},
			signingKey: key,
			headerJWK:  jwk,
			typ:        "dpop+jwt",
			includeJWK: true,
		},
		{
			name: "missing iat",
			mutate: func(claims *ProofClaims) {
				claims.IssuedAt = nil
			},
			signingKey: key,
			headerJWK:  jwk,
			typ:        "dpop+jwt",
			includeJWK: true,
		},
		{
			name: "missing jti",
			mutate: func(claims *ProofClaims) {
				claims.ID = ""
			},
			signingKey: key,
			headerJWK:  jwk,
			typ:        "dpop+jwt",
			includeJWK: true,
		},
		{
			name: "missing ath",
			mutate: func(claims *ProofClaims) {
				claims.ATH = ""
			},
			signingKey: key,
			headerJWK:  jwk,
			typ:        "dpop+jwt",
			includeJWK: true,
		},
		{
			name: "wrong ath",
			mutate: func(claims *ProofClaims) {
				claims.ATH = AccessTokenHash("different-token")
			},
			signingKey: key,
			headerJWK:  jwk,
			typ:        "dpop+jwt",
			includeJWK: true,
		},
		{
			name:       "private key material in jwk",
			signingKey: key,
			headerJWK: func() JWK {
				privateJWK := jwk
				privateJWK.D = "private-material"
				return privateJWK
			}(),
			typ:        "dpop+jwt",
			includeJWK: true,
		},
	}

	verifier := newTestVerifier(now)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := validProofClaims(now)

			if tt.mutate != nil {
				tt.mutate(&claims)
			}

			proof := signTestProof(
				t,
				tt.signingKey,
				tt.headerJWK,
				claims,
				tt.typ,
				tt.includeJWK,
			)

			if _, err := verifier.Verify(
				proof,
				testAccessToken,
				testMethod,
				testHTU,
			); err == nil {
				t.Fatal("expected invalid DPoP proof to be rejected")
			}
		})
	}
}

func TestVerifierRejectsMalformedProof(t *testing.T) {
	verifier := newTestVerifier(time.Now())

	if _, err := verifier.Verify(
		"not-a-jwt",
		testAccessToken,
		testMethod,
		testHTU,
	); err == nil {
		t.Fatal("expected malformed DPoP proof to be rejected")
	}
}
