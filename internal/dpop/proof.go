package dpop

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidProof = errors.New("invalid DPoP proof")

// ProofClaims represents the claims carried by a DPoP proof JWT.
// ProofClaimsはDPoP Proof JWTに含まれるClaimを表す。
type ProofClaims struct {
	HTM string `json:"htm"`
	HTU string `json:"htu"`
	ATH string `json:"ath"`
	jwt.RegisteredClaims
}

// VerificationResult contains information derived from a valid DPoP proof.
// VerificationResultは有効なDPoP Proofから得られた情報を保持する。
type VerificationResult struct {
	JKT string
	JTI string
}

// Verifier validates DPoP proof JWTs.
// VerifierはDPoP Proof JWTを検証する。
type Verifier struct {
	maxAge     time.Duration
	futureSkew time.Duration
	now        func() time.Time
}

// NewVerifier creates a DPoP proof verifier.
// NewVerifierはDPoP Proof Verifierを作成する。
func NewVerifier(maxAge time.Duration, futureSkew time.Duration) *Verifier {
	return &Verifier{
		maxAge:     maxAge,
		futureSkew: futureSkew,
		now:        time.Now,
	}
}

// Verify validates a DPoP proof for a protected-resource request.
// VerifyはProtected Resource Requestに対するDPoP Proofを検証する。
func (v *Verifier) Verify(
	proofJWT string,
	accessToken string,
	method string,
	targetURI string,
) (*VerificationResult, error) {
	if proofJWT == "" {
		return nil, fmt.Errorf("%w: missing proof", ErrInvalidProof)
	}

	claims := &ProofClaims{}
	var proofJWK JWK

	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodES256.Alg()}),
	)

	token, err := parser.ParseWithClaims(
		proofJWT,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodES256 {
				return nil, fmt.Errorf(
					"%w: unsupported signing algorithm",
					ErrInvalidProof,
				)
			}

			typ, ok := token.Header["typ"].(string)
			if !ok || typ != "dpop+jwt" {
				return nil, fmt.Errorf(
					"%w: invalid typ header",
					ErrInvalidProof,
				)
			}

			rawJWK, ok := token.Header["jwk"]
			if !ok {
				return nil, fmt.Errorf(
					"%w: missing jwk header",
					ErrInvalidProof,
				)
			}

			encodedJWK, err := json.Marshal(rawJWK)
			if err != nil {
				return nil, fmt.Errorf(
					"%w: encode jwk header",
					ErrInvalidProof,
				)
			}

			if err := json.Unmarshal(encodedJWK, &proofJWK); err != nil {
				return nil, fmt.Errorf(
					"%w: decode jwk header",
					ErrInvalidProof,
				)
			}

			publicKey, err := proofJWK.PublicKey()
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrInvalidProof, err)
			}

			return publicKey, nil
		},
	)
	if err != nil || !token.Valid {
		return nil, fmt.Errorf(
			"%w: proof signature or structure validation failed",
			ErrInvalidProof,
		)
	}

	if claims.ID == "" {
		return nil, fmt.Errorf("%w: missing jti", ErrInvalidProof)
	}

	if claims.IssuedAt == nil {
		return nil, fmt.Errorf("%w: missing iat", ErrInvalidProof)
	}

	if claims.HTM == "" || claims.HTM != method {
		return nil, fmt.Errorf("%w: htm mismatch", ErrInvalidProof)
	}

	proofURI, err := normalizeTargetURI(claims.HTU)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid htu", ErrInvalidProof)
	}

	requestURI, err := normalizeTargetURI(targetURI)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid request URI", ErrInvalidProof)
	}

	if proofURI != requestURI {
		return nil, fmt.Errorf("%w: htu mismatch", ErrInvalidProof)
	}

	now := v.now()
	issuedAt := claims.IssuedAt.Time

	if issuedAt.Before(now.Add(-v.maxAge)) {
		return nil, fmt.Errorf("%w: proof is too old", ErrInvalidProof)
	}

	if issuedAt.After(now.Add(v.futureSkew)) {
		return nil, fmt.Errorf("%w: proof iat is in the future", ErrInvalidProof)
	}

	if claims.ATH == "" {
		return nil, fmt.Errorf("%w: missing ath", ErrInvalidProof)
	}

	expectedATH := AccessTokenHash(accessToken)

	if subtle.ConstantTimeCompare(
		[]byte(claims.ATH),
		[]byte(expectedATH),
	) != 1 {
		return nil, fmt.Errorf("%w: ath mismatch", ErrInvalidProof)
	}

	thumbprint, err := proofJWK.Thumbprint()
	if err != nil {
		return nil, fmt.Errorf(
			"%w: calculate JWK thumbprint",
			ErrInvalidProof,
		)
	}

	return &VerificationResult{
		JKT: thumbprint,
		JTI: claims.ID,
	}, nil
}

func normalizeTargetURI(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("target URI must be absolute")
	}

	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return parsed.String(), nil
}
