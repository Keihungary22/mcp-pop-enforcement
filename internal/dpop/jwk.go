package dpop

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"
)

// JWK represents the public EC key carried in a DPoP proof header.
// JWKはDPoP Proof Headerに含まれる公開EC Keyを表す。
type JWK struct {
	KTY string `json:"kty"`
	CRV string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
	D   string `json:"d,omitempty"`
}

// PublicKey converts the JWK into a P-256 ECDSA public key.
// PublicKeyはJWKをP-256 ECDSA Public Keyへ変換する。
func (j JWK) PublicKey() (*ecdsa.PublicKey, error) {
	if j.D != "" {
		return nil, fmt.Errorf("DPoP JWK must not contain private key material")
	}

	if j.KTY != "EC" || j.CRV != "P-256" {
		return nil, fmt.Errorf("DPoP JWK must use EC P-256")
	}

	xBytes, err := base64.RawURLEncoding.DecodeString(j.X)
	if err != nil {
		return nil, fmt.Errorf("decode JWK x coordinate: %w", err)
	}

	yBytes, err := base64.RawURLEncoding.DecodeString(j.Y)
	if err != nil {
		return nil, fmt.Errorf("decode JWK y coordinate: %w", err)
	}

	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)

	curve := elliptic.P256()
	if !curve.IsOnCurve(x, y) {
		return nil, fmt.Errorf("DPoP JWK point is not on P-256 curve")
	}

	return &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}, nil
}

// Thumbprint returns the RFC 7638 SHA-256 JWK thumbprint.
// ThumbprintはRFC 7638 SHA-256 JWK Thumbprintを返す。
func (j JWK) Thumbprint() (string, error) {
	if _, err := j.PublicKey(); err != nil {
		return "", err
	}

	canonical := fmt.Sprintf(
		`{"crv":"%s","kty":"%s","x":"%s","y":"%s"}`,
		j.CRV,
		j.KTY,
		j.X,
		j.Y,
	)

	sum := sha256.Sum256([]byte(canonical))

	return base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

// AccessTokenHash returns the RFC 9449 ath value for an access token.
// AccessTokenHashはRFC 9449のath値をAccess Tokenから計算する。
func AccessTokenHash(accessToken string) string {
	sum := sha256.Sum256([]byte(accessToken))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
