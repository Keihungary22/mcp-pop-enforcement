package auth

import (
	"errors"
	"net/http"
	"strings"
)

// Middleware protects the downstream MCP handler.
// MiddlewareはDownstream MCP Handlerを保護する。
type Middleware struct {
	validator *Validator
}

// NewMiddleware creates access-token validation middleware.
// NewMiddlewareはAccess Token Validation Middlewareを作成する。
func NewMiddleware(validator *Validator) *Middleware {
	return &Middleware{validator: validator}
}

// Wrap validates the access token before forwarding the request.
// WrapはRequestをForwardする前にAccess Tokenを検証する。
func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			w.Header().Set("WWW-Authenticate", `Bearer`)
			http.Error(w, "missing or malformed access token", http.StatusUnauthorized)
			return
		}

		_, err := m.validator.Validate(tokenString)
		if err != nil {
			if errors.Is(err, ErrInsufficientScope) {
				w.Header().Set(
					"WWW-Authenticate",
					`Bearer error="insufficient_scope"`,
				)
				http.Error(w, "insufficient scope", http.StatusForbidden)
				return
			}

			w.Header().Set(
				"WWW-Authenticate",
				`Bearer error="invalid_token"`,
			)
			http.Error(w, "invalid access token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)

	if len(parts) != 2 {
		return "", false
	}

	if !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	if parts[1] == "" {
		return "", false
	}

	return parts[1], true
}
