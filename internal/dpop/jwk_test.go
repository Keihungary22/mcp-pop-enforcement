package dpop

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func generateDPoPTestKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate DPoP test key: %v", err)
	}

	return key
}

func jwkFromPrivateKey(key *ecdsa.PrivateKey) JWK {
	publicKey := key.PublicKey

	return JWK{
		KTY: "EC",
		CRV: "P-256",
		X: base64.RawURLEncoding.EncodeToString(
			publicKey.X.FillBytes(make([]byte, 32)),
		),
		Y: base64.RawURLEncoding.EncodeToString(
			publicKey.Y.FillBytes(make([]byte, 32)),
		),
	}
}

func TestAccessTokenHash(t *testing.T) {
	const expected = "ungWv48Bz-pBQUDeXa4iI7ADYaOWF3qctBD_YfIAFa0"

	actual := AccessTokenHash("abc")

	if actual != expected {
		t.Fatalf("expected %s, got %s", expected, actual)
	}
}

func TestJWKPublicKeyAndThumbprint(t *testing.T) {
	key := generateDPoPTestKey(t)
	jwk := jwkFromPrivateKey(key)

	publicKey, err := jwk.PublicKey()
	if err != nil {
		t.Fatalf("PublicKey() returned error: %v", err)
	}

	if publicKey.X.Cmp(key.PublicKey.X) != 0 {
		t.Fatal("unexpected public-key x coordinate")
	}

	if publicKey.Y.Cmp(key.PublicKey.Y) != 0 {
		t.Fatal("unexpected public-key y coordinate")
	}

	first, err := jwk.Thumbprint()
	if err != nil {
		t.Fatalf("Thumbprint() returned error: %v", err)
	}

	second, err := jwk.Thumbprint()
	if err != nil {
		t.Fatalf("second Thumbprint() returned error: %v", err)
	}

	if first == "" {
		t.Fatal("expected non-empty JWK thumbprint")
	}

	if first != second {
		t.Fatal("expected stable JWK thumbprint")
	}
}

func TestJWKRejectsPrivateKeyMaterial(t *testing.T) {
	key := generateDPoPTestKey(t)
	jwk := jwkFromPrivateKey(key)
	jwk.D = "private-material"

	if _, err := jwk.PublicKey(); err == nil {
		t.Fatal("expected private JWK material to be rejected")
	}
}
