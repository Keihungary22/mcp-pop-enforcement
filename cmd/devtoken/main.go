package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Keihungary22/mcp-pop-enforcement/internal/auth"
	"github.com/Keihungary22/mcp-pop-enforcement/internal/dpop"
	"github.com/golang-jwt/jwt/v5"
)

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

func writePrivateKey(path string, key *ecdsa.PrivateKey) error {
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}

	data := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: der,
	})

	return os.WriteFile(path, data, 0o600)
}

func writePublicKey(path string, key *ecdsa.PublicKey) error {
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return err
	}

	data := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	})

	return os.WriteFile(path, data, 0o644)
}

func main() {
	outputDir := flag.String(
		"output-dir",
		".dev/keys",
		"directory for development keys",
	)
	issuer := flag.String(
		"issuer",
		"https://issuer.example",
		"token issuer",
	)
	audience := flag.String(
		"audience",
		"mcp://resource",
		"token audience",
	)
	scope := flag.String(
		"scope",
		"mcp:invoke",
		"token scope",
	)
	ttl := flag.Duration(
		"ttl",
		5*time.Minute,
		"token lifetime",
	)

	flag.Parse()

	issuerKey, err := ecdsa.GenerateKey(
		elliptic.P256(),
		rand.Reader,
	)
	if err != nil {
		panic(err)
	}

	clientKey, err := ecdsa.GenerateKey(
		elliptic.P256(),
		rand.Reader,
	)
	if err != nil {
		panic(err)
	}

	if err := os.MkdirAll(*outputDir, 0o700); err != nil {
		panic(err)
	}

	issuerPrivatePath := filepath.Join(
		*outputDir,
		"issuer-private.pem",
	)
	issuerPublicPath := filepath.Join(
		*outputDir,
		"issuer-public.pem",
	)
	clientPrivatePath := filepath.Join(
		*outputDir,
		"client-private.pem",
	)

	if err := writePrivateKey(
		issuerPrivatePath,
		issuerKey,
	); err != nil {
		panic(err)
	}

	if err := writePublicKey(
		issuerPublicPath,
		&issuerKey.PublicKey,
	); err != nil {
		panic(err)
	}

	if err := writePrivateKey(
		clientPrivatePath,
		clientKey,
	); err != nil {
		panic(err)
	}

	jkt, err := publicJWK(clientKey).Thumbprint()
	if err != nil {
		panic(err)
	}

	now := time.Now()

	claims := auth.Claims{
		Scope: *scope,
		CNF: &auth.Confirmation{
			JKT: jkt,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    *issuer,
			Subject:   "mock-client",
			Audience:  jwt.ClaimStrings{*audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(*ttl)),
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodES256,
		claims,
	)

	signed, err := token.SignedString(issuerKey)
	if err != nil {
		panic(err)
	}

	fmt.Println(signed)
}
