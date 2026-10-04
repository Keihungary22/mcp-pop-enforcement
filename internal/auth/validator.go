package auth

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken      = errors.New("invalid access token")
	ErrInsufficientScope = errors.New("insufficient scope")
)

// Confirmation contains proof-of-possession key confirmation data.
// ConfirmationはProof-of-Possession KeyのConfirmation情報を保持する。
type Confirmation struct {
	JKT string `json:"jkt"`
}

// Claims represents the access-token claims required by the baseline.
// ClaimsはBaselineで必要となるAccess Token Claimを表す。
type Claims struct {
	Scope string        `json:"scope"`
	CNF   *Confirmation `json:"cnf,omitempty"`
	jwt.RegisteredClaims
}

// Validator validates signed OAuth access tokens.
// Validatorは署名済みOAuth Access Tokenを検証する。
type Validator struct {
	publicKey     *ecdsa.PublicKey
	requiredScope string
	parser        *jwt.Parser
}

// NewValidator creates an ES256 JWT validator.
// NewValidatorはES256 JWT Validatorを作成する。
func NewValidator(
	publicKey *ecdsa.PublicKey,
	issuer string,
	audience string,
	requiredScope string,
) *Validator {
	return &Validator{
		publicKey:     publicKey,
		requiredScope: requiredScope,
		parser: jwt.NewParser(
			jwt.WithValidMethods([]string{jwt.SigningMethodES256.Alg()}),
			jwt.WithIssuer(issuer),
			jwt.WithAudience(audience),
			jwt.WithExpirationRequired(),
			jwt.WithIssuedAt(),
		),
	}
}

// Validate validates the token and required scope.
// ValidateはTokenとRequired Scopeを検証する。
func (v *Validator) Validate(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := v.parser.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodES256 {
				return nil, fmt.Errorf("%w: unexpected signing algorithm", ErrInvalidToken)
			}

			return v.publicKey, nil
		},
	)
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("%w: token validation failed", ErrInvalidToken)
	}

	if claims.IssuedAt == nil {
		return nil, fmt.Errorf("%w: missing iat", ErrInvalidToken)
	}

	if v.requiredScope != "" && !hasScope(claims.Scope, v.requiredScope) {
		return nil, ErrInsufficientScope
	}

	return claims, nil
}

func hasScope(scopeClaim string, required string) bool {
	for _, scope := range strings.Fields(scopeClaim) {
		if scope == required {
			return true
		}
	}

	return false
}
