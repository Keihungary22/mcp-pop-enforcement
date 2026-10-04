package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// LoadECDSAPublicKey loads a P-256 public key from a PEM file.
// LoadECDSAPublicKeyはPEM FileからP-256 Public Keyを読み込む。
func LoadECDSAPublicKey(path string) (*ecdsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read access-token public key: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("decode access-token public key: invalid PEM")
	}

	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse access-token public key: %w", err)
	}

	key, ok := parsed.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("access-token public key is not ECDSA")
	}

	if key.Curve.Params().Name != elliptic.P256().Params().Name {
		return nil, fmt.Errorf("access-token public key must use P-256")
	}

	return key, nil
}
