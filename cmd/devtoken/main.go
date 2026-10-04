package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	outputDir := flag.String("output-dir", ".dev/keys", "directory for development keys")
	issuer := flag.String("issuer", "https://issuer.example", "token issuer")
	audience := flag.String("audience", "mcp://resource", "token audience")
	scope := flag.String("scope", "mcp:invoke", "token scope")
	ttl := flag.Duration("ttl", 5*time.Minute, "token lifetime")
	flag.Parse()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}

	if err := os.MkdirAll(*outputDir, 0o700); err != nil {
		panic(err)
	}

	privateDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		panic(err)
	}

	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateDER,
	})

	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		panic(err)
	}

	publicPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicDER,
	})

	privatePath := filepath.Join(*outputDir, "issuer-private.pem")
	publicPath := filepath.Join(*outputDir, "issuer-public.pem")

	if err := os.WriteFile(privatePath, privatePEM, 0o600); err != nil {
		panic(err)
	}

	if err := os.WriteFile(publicPath, publicPEM, 0o644); err != nil {
		panic(err)
	}

	now := time.Now()

	claims := jwt.MapClaims{
		"iss":   *issuer,
		"sub":   "mock-client",
		"aud":   *audience,
		"scope": *scope,
		"iat":   now.Unix(),
		"exp":   now.Add(*ttl).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)

	signed, err := token.SignedString(privateKey)
	if err != nil {
		panic(err)
	}

	fmt.Println(signed)
}
