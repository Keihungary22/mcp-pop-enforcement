package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// JWKSValidator validates RS256 access tokens using a JWKS endpoint.
// JWKSValidatorはJWKS Endpointを使用してRS256 Access Tokenを検証する。
type JWKSValidator struct {
	resolver         *JWKSResolver
	expectedIssuer   string
	expectedAudience string
	requiredScope    string
}

// NewJWKSValidator creates an RS256/JWKS access-token validator.
// NewJWKSValidatorはRS256/JWKS Access Token Validatorを作成する。
func NewJWKSValidator(
	resolver *JWKSResolver,
	expectedIssuer string,
	expectedAudience string,
	requiredScope string,
) *JWKSValidator {
	return &JWKSValidator{
		resolver:         resolver,
		expectedIssuer:   expectedIssuer,
		expectedAudience: expectedAudience,
		requiredScope:    requiredScope,
	}
}

// Validate validates a Keycloak-style RS256 access token.
// ValidateはKeycloak形式のRS256 Access Tokenを検証する。
func (v *JWKSValidator) Validate(
	tokenString string,
) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodRS256 {
				return nil, fmt.Errorf(
					"unexpected signing method: %s",
					token.Method.Alg(),
				)
			}

			kid, ok := token.Header["kid"].(string)
			if !ok || strings.TrimSpace(kid) == "" {
				return nil, fmt.Errorf(
					"missing JWT kid",
				)
			}

			return v.resolver.Resolve(
				context.Background(),
				kid,
			)
		},
		jwt.WithValidMethods(
			[]string{"RS256"},
		),
		jwt.WithIssuer(
			v.expectedIssuer,
		),
		jwt.WithAudience(
			v.expectedAudience,
		),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil || !token.Valid {
		return nil, fmt.Errorf(
			"%w: %v",
			ErrInvalidToken,
			err,
		)
	}

	if claims.IssuedAt == nil {
		return nil, fmt.Errorf(
			"%w: missing iat",
			ErrInvalidToken,
		)
	}

	if !containsScope(
		claims.Scope,
		v.requiredScope,
	) {
		return nil, ErrInsufficientScope
	}

	return claims, nil
}

func containsScope(
	scopeClaim string,
	requiredScope string,
) bool {
	for _, scope := range strings.Fields(scopeClaim) {
		if scope == requiredScope {
			return true
		}
	}

	return false
}
