package enforcement

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/Keihungary22/mcp-pop-enforcement/internal/auth"
	"github.com/Keihungary22/mcp-pop-enforcement/internal/dpop"
)

// Middleware combines access-token, DPoP proof, and replay validation.
// MiddlewareはAccess Token、DPoP Proof、Replay検証を統合する。
type Middleware struct {
	tokenValidator *auth.Validator
	proofVerifier  *dpop.Verifier
	replayStore    dpop.ReplayStore
	expectedHTU    string
}

// NewMiddleware creates DPoP-bound resource protection middleware.
// NewMiddlewareはDPoP-bound Resource Protection Middlewareを作成する。
func NewMiddleware(
	tokenValidator *auth.Validator,
	proofVerifier *dpop.Verifier,
	replayStore dpop.ReplayStore,
	expectedHTU string,
) *Middleware {
	return &Middleware{
		tokenValidator: tokenValidator,
		proofVerifier:  proofVerifier,
		replayStore:    replayStore,
		expectedHTU:    expectedHTU,
	}
}

// Wrap validates the DPoP-bound access token before forwarding.
// WrapはForward前にDPoP-bound Access Tokenを検証する。
func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accessToken, ok := dpopAuthorizationToken(
			r.Header.Values("Authorization"),
		)
		if !ok {
			writeDPoPUnauthorized(
				w,
				`invalid_token`,
				"missing or malformed DPoP authorization",
			)
			return
		}

		claims, err := m.tokenValidator.Validate(accessToken)
		if err != nil {
			if errors.Is(err, auth.ErrInsufficientScope) {
				w.Header().Set(
					"WWW-Authenticate",
					`DPoP error="insufficient_scope"`,
				)
				http.Error(
					w,
					"insufficient scope",
					http.StatusForbidden,
				)
				return
			}

			writeDPoPUnauthorized(
				w,
				`invalid_token`,
				"invalid access token",
			)
			return
		}

		if claims.CNF == nil || claims.CNF.JKT == "" {
			writeDPoPUnauthorized(
				w,
				`invalid_token`,
				"access token is not DPoP-bound",
			)
			return
		}

		proofValues := r.Header.Values("DPoP")
		if len(proofValues) != 1 || strings.TrimSpace(proofValues[0]) == "" {
			writeDPoPUnauthorized(
				w,
				`invalid_dpop_proof`,
				"missing or malformed DPoP proof",
			)
			return
		}

		result, err := m.proofVerifier.Verify(
			proofValues[0],
			accessToken,
			r.Method,
			m.expectedHTU,
		)
		if err != nil {
			writeDPoPUnauthorized(
				w,
				`invalid_dpop_proof`,
				"invalid DPoP proof",
			)
			return
		}

		if subtle.ConstantTimeCompare(
			[]byte(claims.CNF.JKT),
			[]byte(result.JKT),
		) != 1 {
			writeDPoPUnauthorized(
				w,
				`invalid_dpop_proof`,
				"DPoP key binding mismatch",
			)
			return
		}

		if !m.replayStore.CheckAndStore(
			result.JKT,
			result.JTI,
		) {
			writeDPoPUnauthorized(
				w,
				`invalid_dpop_proof`,
				"DPoP proof replay detected",
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func dpopAuthorizationToken(values []string) (string, bool) {
	if len(values) != 1 {
		return "", false
	}

	parts := strings.Fields(values[0])
	if len(parts) != 2 {
		return "", false
	}

	if !strings.EqualFold(parts[0], "DPoP") {
		return "", false
	}

	if parts[1] == "" {
		return "", false
	}

	return parts[1], true
}

func writeDPoPUnauthorized(
	w http.ResponseWriter,
	errorCode string,
	message string,
) {
	w.Header().Set(
		"WWW-Authenticate",
		`DPoP error="`+errorCode+`"`,
	)

	http.Error(
		w,
		message,
		http.StatusUnauthorized,
	)
}
