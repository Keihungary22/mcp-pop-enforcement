package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubTokenValidator struct {
	claims *Claims
	err    error
}

func (s *stubTokenValidator) Validate(
	_ string,
) (*Claims, error) {
	return s.claims, s.err
}

func TestIntrospectionValidatorAcceptsActiveToken(
	t *testing.T,
) {
	t.Parallel()

	var receivedToken string
	var receivedUser string
	var receivedPassword string

	server := httptest.NewServer(
		http.HandlerFunc(
			func(
				writer http.ResponseWriter,
				request *http.Request,
			) {
				if err := request.ParseForm(); err != nil {
					t.Fatalf(
						"parse introspection form: %v",
						err,
					)
				}

				receivedToken = request.Form.Get("token")

				receivedUser, receivedPassword, _ =
					request.BasicAuth()

				writer.Header().Set(
					"Content-Type",
					"application/json",
				)

				_, _ = writer.Write(
					[]byte(`{"active":true}`),
				)
			},
		),
	)
	defer server.Close()

	expectedClaims := &Claims{}

	validator := NewIntrospectionValidator(
		&stubTokenValidator{
			claims: expectedClaims,
		},
		server.URL,
		"mcp-resource",
		"resource-secret",
	)

	claims, err := validator.Validate(
		"test-access-token",
	)
	if err != nil {
		t.Fatalf(
			"expected active token to pass: %v",
			err,
		)
	}

	if claims != expectedClaims {
		t.Fatal(
			"expected claims from base validator",
		)
	}

	if receivedToken != "test-access-token" {
		t.Fatalf(
			"unexpected introspected token: %q",
			receivedToken,
		)
	}

	if receivedUser != "mcp-resource" {
		t.Fatalf(
			"unexpected client ID: %q",
			receivedUser,
		)
	}

	if receivedPassword != "resource-secret" {
		t.Fatal(
			"unexpected client secret",
		)
	}
}

func TestIntrospectionValidatorRejectsInactiveToken(
	t *testing.T,
) {
	t.Parallel()

	server := httptest.NewServer(
		http.HandlerFunc(
			func(
				writer http.ResponseWriter,
				_ *http.Request,
			) {
				writer.Header().Set(
					"Content-Type",
					"application/json",
				)

				_, _ = writer.Write(
					[]byte(`{"active":false}`),
				)
			},
		),
	)
	defer server.Close()

	validator := NewIntrospectionValidator(
		&stubTokenValidator{
			claims: &Claims{},
		},
		server.URL,
		"mcp-resource",
		"resource-secret",
	)

	_, err := validator.Validate(
		"revoked-token",
	)

	if !errors.Is(err, ErrInactiveToken) {
		t.Fatalf(
			"expected ErrInactiveToken, got %v",
			err,
		)
	}
}

func TestIntrospectionValidatorStopsWhenBaseValidationFails(
	t *testing.T,
) {
	t.Parallel()

	introspectionCalled := false

	server := httptest.NewServer(
		http.HandlerFunc(
			func(
				writer http.ResponseWriter,
				_ *http.Request,
			) {
				introspectionCalled = true
				writer.WriteHeader(
					http.StatusOK,
				)
			},
		),
	)
	defer server.Close()

	baseError := errors.New(
		"invalid JWT",
	)

	validator := NewIntrospectionValidator(
		&stubTokenValidator{
			err: baseError,
		},
		server.URL,
		"mcp-resource",
		"resource-secret",
	)

	_, err := validator.Validate(
		"invalid-token",
	)

	if !errors.Is(err, baseError) {
		t.Fatalf(
			"expected base validation error, got %v",
			err,
		)
	}

	if introspectionCalled {
		t.Fatal(
			"introspection must not run after base validation failure",
		)
	}
}
