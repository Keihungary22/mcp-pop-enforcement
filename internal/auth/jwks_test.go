package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJWKSResolverResolvesRS256SigningKey(t *testing.T) {
	modulus := new(big.Int)
	modulus.SetString(
		"00c34f123456789abcdef123456789abcdef123456789abcdef",
		16,
	)

	n := base64.RawURLEncoding.EncodeToString(
		modulus.Bytes(),
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
						"kid": "signing-key",
						"kty": "RSA",
						"alg": "RS256",
						"use": "sig",
						"n":   n,
						"e":   e,
					},
					{
						"kid": "encryption-key",
						"kty": "RSA",
						"alg": "RSA-OAEP",
						"use": "enc",
						"n":   n,
						"e":   e,
					},
				},
			}

			w.Header().Set(
				"Content-Type",
				"application/json",
			)

			if err := json.NewEncoder(w).Encode(document); err != nil {
				t.Fatalf("encode JWKS: %v", err)
			}
		}),
	)
	defer server.Close()

	resolver := NewJWKSResolver(server.URL)

	key, err := resolver.Resolve(
		context.Background(),
		"signing-key",
	)
	if err != nil {
		t.Fatalf("resolve signing key: %v", err)
	}

	if key.E != 65537 {
		t.Fatalf(
			"expected exponent 65537, got %d",
			key.E,
		)
	}
}

func TestJWKSResolverRejectsUnknownKey(t *testing.T) {
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

	resolver := NewJWKSResolver(server.URL)

	_, err := resolver.Resolve(
		context.Background(),
		"unknown-key",
	)

	if err == nil {
		t.Fatal("expected unknown kid to be rejected")
	}
}

func TestJWKSResolverIgnoresEncryptionKeys(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			_, _ = w.Write(
				[]byte(`{
                    "keys": [
                        {
                            "kid": "enc-key",
                            "kty": "RSA",
                            "alg": "RSA-OAEP",
                            "use": "enc",
                            "n": "AQ",
                            "e": "AQAB"
                        }
                    ]
                }`),
			)
		}),
	)
	defer server.Close()

	resolver := NewJWKSResolver(server.URL)

	_, err := resolver.Resolve(
		context.Background(),
		"enc-key",
	)

	if err == nil {
		t.Fatal(
			"expected encryption-only key to be ignored",
		)
	}
}
