package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWKSValidatorAcceptsValidRS256Token(t *testing.T) {
	privateKey, err := rsa.GenerateKey(
		rand.Reader,
		2048,
	)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	const kid = "keycloak-signing-key"

	n := base64.RawURLEncoding.EncodeToString(
		privateKey.PublicKey.N.Bytes(),
	)

	e := base64.RawURLEncoding.EncodeToString(
		[]byte{0x01, 0x00, 0x01},
	)

	server := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			document := map[string]any{
				"keys": []map[string]string{
					{
						"kid": kid,
						"kty": "RSA",
						"alg": "RS256",
						"use": "sig",
						"n":   n,
						"e":   e,
					},
				},
			}

			w.Header().Set(
				"Content-Type",
				"application/json",
			)

			_ = json.NewEncoder(w).Encode(document)
		}),
	)
	defer server.Close()

	resolver := NewJWKSResolver(server.URL)

	validator := NewJWKSValidator(
		resolver,
		"http://keycloak.example/realms/mcp-pop",
		"mcp-resource",
		"mcp:invoke",
	)

	now := time.Now()

	claims := Claims{
		Scope: "mcp:invoke",
		CNF: &Confirmation{
			JKT: "client-thumbprint",
		},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "http://keycloak.example/realms/mcp-pop",
			Audience: jwt.ClaimStrings{
				"mcp-resource",
			},
			IssuedAt: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(
				now.Add(5 * time.Minute),
			),
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodRS256,
		claims,
	)

	token.Header["kid"] = kid

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	result, err := validator.Validate(tokenString)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}

	if result.CNF == nil {
		t.Fatal("expected cnf claim")
	}

	if result.CNF.JKT != "client-thumbprint" {
		t.Fatalf(
			"unexpected cnf.jkt: %s",
			result.CNF.JKT,
		)
	}
}

func TestJWKSValidatorRejectsUnknownKID(t *testing.T) {
	privateKey, err := rsa.GenerateKey(
		rand.Reader,
		2048,
	)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	server := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			_, _ = w.Write(
				[]byte(`{"keys":[]}`),
			)
		}),
	)
	defer server.Close()

	validator := NewJWKSValidator(
		NewJWKSResolver(server.URL),
		"http://keycloak.example/realms/mcp-pop",
		"mcp-resource",
		"mcp:invoke",
	)

	now := time.Now()

	claims := Claims{
		Scope: "mcp:invoke",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "http://keycloak.example/realms/mcp-pop",
			Audience: jwt.ClaimStrings{
				"mcp-resource",
			},
			IssuedAt: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(
				now.Add(5 * time.Minute),
			),
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodRS256,
		claims,
	)

	token.Header["kid"] = "unknown-key"

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := validator.Validate(tokenString); err == nil {
		t.Fatal("expected unknown kid to be rejected")
	}
}
