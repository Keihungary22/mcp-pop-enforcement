package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"
)

var (
	ErrJWKSKeyNotFound = errors.New("JWKS signing key not found")
	ErrInvalidJWKSKey  = errors.New("invalid JWKS key")
)

// jwksDocument represents a JSON Web Key Set.
// jwksDocumentはJSON Web Key Setを表す。
type jwksDocument struct {
	Keys []rsaJWK `json:"keys"`
}

// rsaJWK represents an RSA public key from a JWKS document.
// rsaJWKはJWKS内のRSA Public Keyを表す。
type rsaJWK struct {
	KID string `json:"kid"`
	KTY string `json:"kty"`
	ALG string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// JWKSResolver resolves JWT signing keys by kid.
// JWKSResolverはkidを使ってJWT Signing Keyを解決する。
type JWKSResolver struct {
	url        string
	httpClient *http.Client

	mu   sync.RWMutex
	keys map[string]*rsa.PublicKey
}

// NewJWKSResolver creates a JWKS resolver.
// NewJWKSResolverはJWKS Resolverを作成する。
func NewJWKSResolver(url string) *JWKSResolver {
	return &JWKSResolver{
		url: url,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		keys: make(map[string]*rsa.PublicKey),
	}
}

// Resolve returns the RSA signing key matching kid.
//
// The local cache is checked first. If the key is unknown,
// the JWKS endpoint is refreshed once before returning an error.
//
// Resolveはkidに一致するRSA Signing Keyを返す。
// Cacheに存在しない場合はJWKSを再取得してから判定する。
func (r *JWKSResolver) Resolve(
	ctx context.Context,
	kid string,
) (*rsa.PublicKey, error) {
	if kid == "" {
		return nil, fmt.Errorf(
			"%w: missing kid",
			ErrJWKSKeyNotFound,
		)
	}

	r.mu.RLock()
	key := r.keys[kid]
	r.mu.RUnlock()

	if key != nil {
		return key, nil
	}

	if err := r.refresh(ctx); err != nil {
		return nil, err
	}

	r.mu.RLock()
	key = r.keys[kid]
	r.mu.RUnlock()

	if key == nil {
		return nil, fmt.Errorf(
			"%w: %s",
			ErrJWKSKeyNotFound,
			kid,
		)
	}

	return key, nil
}

func (r *JWKSResolver) refresh(
	ctx context.Context,
) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		r.url,
		nil,
	)
	if err != nil {
		return fmt.Errorf(
			"create JWKS request: %w",
			err,
		)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf(
			"fetch JWKS: %w",
			err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)

		return fmt.Errorf(
			"fetch JWKS: unexpected status %d",
			resp.StatusCode,
		)
	}

	var document jwksDocument

	if err := json.NewDecoder(resp.Body).Decode(&document); err != nil {
		return fmt.Errorf(
			"decode JWKS: %w",
			err,
		)
	}

	keys := make(map[string]*rsa.PublicKey)

	for _, jwk := range document.Keys {
		if jwk.KTY != "RSA" {
			continue
		}

		if jwk.Use != "" && jwk.Use != "sig" {
			continue
		}

		if jwk.ALG != "" && jwk.ALG != "RS256" {
			continue
		}

		publicKey, err := rsaPublicKey(jwk)
		if err != nil {
			continue
		}

		keys[jwk.KID] = publicKey
	}

	r.mu.Lock()
	r.keys = keys
	r.mu.Unlock()

	return nil
}

func rsaPublicKey(
	jwk rsaJWK,
) (*rsa.PublicKey, error) {
	if jwk.KID == "" || jwk.N == "" || jwk.E == "" {
		return nil, ErrInvalidJWKSKey
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: modulus",
			ErrInvalidJWKSKey,
		)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: exponent",
			ErrInvalidJWKSKey,
		)
	}

	if len(eBytes) == 0 || len(eBytes) > 8 {
		return nil, fmt.Errorf(
			"%w: exponent length",
			ErrInvalidJWKSKey,
		)
	}

	exponent := 0

	for _, value := range eBytes {
		exponent = exponent<<8 + int(value)
	}

	if exponent < 3 {
		return nil, fmt.Errorf(
			"%w: exponent value",
			ErrInvalidJWKSKey,
		)
	}

	modulus := new(big.Int).SetBytes(nBytes)

	if modulus.Sign() <= 0 {
		return nil, fmt.Errorf(
			"%w: modulus value",
			ErrInvalidJWKSKey,
		)
	}

	return &rsa.PublicKey{
		N: modulus,
		E: exponent,
	}, nil
}
