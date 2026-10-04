package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testIssuer   = "https://issuer.example"
	testAudience = "mcp://resource"
	testScope    = "mcp:invoke"
)

func generateTestKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}

	return key
}

func signTestToken(
	t *testing.T,
	privateKey *ecdsa.PrivateKey,
	issuer string,
	audience string,
	scope string,
	issuedAt time.Time,
	expiresAt time.Time,
) string {
	t.Helper()

	claims := Claims{
		Scope: scope,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)

	signed, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}

	return signed
}

// TestValidatorValidToken verifies that a valid access token is accepted.
// TestValidatorValidTokenは有効なAccess Tokenが受理されることを確認する。
func TestValidatorValidToken(t *testing.T) {
	key := generateTestKey(t)

	validator := NewValidator(
		&key.PublicKey,
		testIssuer,
		testAudience,
		testScope,
	)

	now := time.Now()

	token := signTestToken(
		t,
		key,
		testIssuer,
		testAudience,
		"mcp:read mcp:invoke",
		now.Add(-time.Minute),
		now.Add(5*time.Minute),
	)

	claims, err := validator.Validate(token)
	if err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}

	if claims.Scope != "mcp:read mcp:invoke" {
		t.Fatalf("unexpected scope: %s", claims.Scope)
	}
}

// TestValidatorRejectsInvalidTokens verifies baseline token rejection cases.
// TestValidatorRejectsInvalidTokensはBaselineで拒否すべきTokenを確認する。
func TestValidatorRejectsInvalidTokens(t *testing.T) {
	key := generateTestKey(t)
	attackerKey := generateTestKey(t)

	validator := NewValidator(
		&key.PublicKey,
		testIssuer,
		testAudience,
		testScope,
	)

	now := time.Now()

	tests := []struct {
		name      string
		token     string
		targetErr error
	}{
		{
			name:      "malformed token",
			token:     "not-a-jwt",
			targetErr: ErrInvalidToken,
		},
		{
			name: "invalid signature",
			token: signTestToken(
				t,
				attackerKey,
				testIssuer,
				testAudience,
				testScope,
				now.Add(-time.Minute),
				now.Add(5*time.Minute),
			),
			targetErr: ErrInvalidToken,
		},
		{
			name: "expired token",
			token: signTestToken(
				t,
				key,
				testIssuer,
				testAudience,
				testScope,
				now.Add(-10*time.Minute),
				now.Add(-time.Minute),
			),
			targetErr: ErrInvalidToken,
		},
		{
			name: "wrong issuer",
			token: signTestToken(
				t,
				key,
				"https://wrong-issuer.example",
				testAudience,
				testScope,
				now.Add(-time.Minute),
				now.Add(5*time.Minute),
			),
			targetErr: ErrInvalidToken,
		},
		{
			name: "wrong audience",
			token: signTestToken(
				t,
				key,
				testIssuer,
				"mcp://wrong-resource",
				testScope,
				now.Add(-time.Minute),
				now.Add(5*time.Minute),
			),
			targetErr: ErrInvalidToken,
		},
		{
			name: "insufficient scope",
			token: signTestToken(
				t,
				key,
				testIssuer,
				testAudience,
				"mcp:read",
				now.Add(-time.Minute),
				now.Add(5*time.Minute),
			),
			targetErr: ErrInsufficientScope,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validator.Validate(tt.token)

			if !errors.Is(err, tt.targetErr) {
				t.Fatalf(
					"expected error %v, got %v",
					tt.targetErr,
					err,
				)
			}
		})
	}
}
