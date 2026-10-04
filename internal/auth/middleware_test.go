package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMiddlewareAllowsValidToken(t *testing.T) {
	key := generateTestKey(t)

	validator := NewValidator(
		&key.PublicKey,
		testIssuer,
		testAudience,
		testScope,
	)

	middleware := NewMiddleware(validator)

	now := time.Now()
	token := signTestToken(
		t,
		key,
		testIssuer,
		testAudience,
		testScope,
		now.Add(-time.Minute),
		now.Add(5*time.Minute),
	)

	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.Wrap(next)

	request := httptest.NewRequest(
		http.MethodPost,
		"http://gateway.test/mcp",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+token)

	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			response.Code,
		)
	}

	if !called {
		t.Fatal("expected downstream handler to be called")
	}
}

func TestMiddlewareRejectsMissingToken(t *testing.T) {
	key := generateTestKey(t)

	validator := NewValidator(
		&key.PublicKey,
		testIssuer,
		testAudience,
		testScope,
	)

	middleware := NewMiddleware(validator)

	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.Wrap(next)

	request := httptest.NewRequest(
		http.MethodPost,
		"http://gateway.test/mcp",
		nil,
	)

	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			response.Code,
		)
	}

	if called {
		t.Fatal("downstream handler must not be called")
	}
}

func TestMiddlewareRejectsInsufficientScope(t *testing.T) {
	key := generateTestKey(t)

	validator := NewValidator(
		&key.PublicKey,
		testIssuer,
		testAudience,
		testScope,
	)

	middleware := NewMiddleware(validator)

	now := time.Now()
	token := signTestToken(
		t,
		key,
		testIssuer,
		testAudience,
		"mcp:read",
		now.Add(-time.Minute),
		now.Add(5*time.Minute),
	)

	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.Wrap(next)

	request := httptest.NewRequest(
		http.MethodPost,
		"http://gateway.test/mcp",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+token)

	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusForbidden,
			response.Code,
		)
	}

	if called {
		t.Fatal("downstream handler must not be called")
	}
}

func TestMiddlewareRejectsInvalidToken(t *testing.T) {
	key := generateTestKey(t)

	validator := NewValidator(
		&key.PublicKey,
		testIssuer,
		testAudience,
		testScope,
	)

	middleware := NewMiddleware(validator)

	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler := middleware.Wrap(next)

	request := httptest.NewRequest(
		http.MethodPost,
		"http://gateway.test/mcp",
		nil,
	)
	request.Header.Set("Authorization", "Bearer not-a-valid-jwt")

	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			response.Code,
		)
	}

	if called {
		t.Fatal("downstream handler must not be called")
	}
}
