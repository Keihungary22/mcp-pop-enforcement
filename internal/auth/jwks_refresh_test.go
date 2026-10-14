package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestJWKSResolverLimitsSequentialUnknownKIDRefreshes(t *testing.T) {
	var fetches atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		fetches.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"keys":[]}`))
	}))
	defer server.Close()

	resolver := NewJWKSResolver(server.URL)

	for i := 0; i < 10; i++ {
		_, err := resolver.Resolve(context.Background(), "unknown-key")

		if !errors.Is(err, ErrJWKSKeyNotFound) {
			t.Fatalf("expected unknown key error, got %v", err)
		}
	}

	if got := fetches.Load(); got != 1 {
		t.Fatalf("expected one JWKS fetch, got %d", got)
	}
}

func TestJWKSResolverCoalescesConcurrentUnknownKIDRefreshes(t *testing.T) {
	var fetches atomic.Int32

	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	var once sync.Once

	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		fetches.Add(1)
		once.Do(func() { close(requestStarted) })
		<-releaseRequest

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"keys":[]}`))
	}))
	defer server.Close()

	resolver := NewJWKSResolver(server.URL)

	const requests = 10

	start := make(chan struct{})
	var wg sync.WaitGroup

	errs := make(chan error, requests)

	for i := 0; i < requests; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			_, err := resolver.Resolve(context.Background(), "unknown-key")
			errs <- err
		}()
	}

	close(start)
	<-requestStarted

	close(releaseRequest)
	wg.Wait()
	close(errs)

	for err := range errs {
		if !errors.Is(err, ErrJWKSKeyNotFound) {
			t.Fatalf("expected unknown key error, got %v", err)
		}
	}

	if got := fetches.Load(); got != 1 {
		t.Fatalf("expected one concurrent JWKS fetch, got %d", got)
	}
}

func TestJWKSResolverAllowsKeyRotationAfterCooldown(t *testing.T) {
	var fetches atomic.Int32
	var rotated atomic.Bool

	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		fetches.Add(1)

		kid := "old-key"
		if rotated.Load() {
			kid = "new-key"
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(
			`{"keys":[{"kid":"` + kid +
				`","kty":"RSA","alg":"RS256","use":"sig",` +
				`"n":"AQAB","e":"AQAB"}]}`,
		))
	}))
	defer server.Close()

	resolver := NewJWKSResolver(server.URL)
	ctx := context.Background()

	if _, err := resolver.Resolve(ctx, "old-key"); err != nil {
		t.Fatalf("initial key resolution failed: %v", err)
	}

	rotated.Store(true)

	_, err := resolver.Resolve(ctx, "new-key")
	if !errors.Is(err, ErrJWKSKeyNotFound) {
		t.Fatalf("expected cooldown to block immediate refresh, got %v", err)
	}

	resolver.refreshMu.Lock()
	resolver.lastRefresh = time.Now().Add(-31 * time.Second)
	resolver.refreshMu.Unlock()

	if _, err := resolver.Resolve(ctx, "new-key"); err != nil {
		t.Fatalf("new key was not resolved after cooldown: %v", err)
	}

	if got := fetches.Load(); got != 2 {
		t.Fatalf("expected two JWKS fetches, got %d", got)
	}
}

func TestJWKSResolverRetriesAfterFetchFailure(t *testing.T) {
	var fetches atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		count := fetches.Add(1)

		if count == 1 {
			http.Error(w, "temporary failure", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(
			`{"keys":[{"kid":"recovered-key",` +
				`"kty":"RSA","alg":"RS256","use":"sig",` +
				`"n":"AQAB","e":"AQAB"}]}`,
		))
	}))
	defer server.Close()

	resolver := NewJWKSResolver(server.URL)
	ctx := context.Background()

	// First request fails.
	if _, err := resolver.Resolve(ctx, "recovered-key"); err == nil {
		t.Fatal("expected first JWKS fetch to fail")
	}

	// Immediate retry must not trigger another HTTP request.
	if _, err := resolver.Resolve(ctx, "recovered-key"); err == nil {
		t.Fatal("expected failure backoff to reject immediate retry")
	}

	if got := fetches.Load(); got != 1 {
		t.Fatalf("expected one fetch during backoff, got %d", got)
	}

	// Simulate expiration of the two-second backoff.
	resolver.refreshMu.Lock()
	resolver.lastFailure = time.Now().Add(-3 * time.Second)
	resolver.refreshMu.Unlock()

	if _, err := resolver.Resolve(ctx, "recovered-key"); err != nil {
		t.Fatalf("expected successful retry after backoff: %v", err)
	}

	if got := fetches.Load(); got != 2 {
		t.Fatalf("expected two JWKS fetch attempts, got %d", got)
	}
}

func TestJWKSResolverCanceledContextDoesNotStartBackoff(t *testing.T) {
	var fetches atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		fetches.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"keys":[]}`))
	}))
	defer server.Close()

	resolver := NewJWKSResolver(server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := resolver.Resolve(ctx, "unknown-key"); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}

	if _, err := resolver.Resolve(context.Background(), "unknown-key"); !errors.Is(err, ErrJWKSKeyNotFound) {
		t.Fatalf("expected normal refresh after cancellation, got %v", err)
	}

	if got := fetches.Load(); got != 1 {
		t.Fatalf("expected one JWKS fetch after cancellation, got %d", got)
	}
}

func TestJWKSResolverCachedKeySurvivesRefreshFailure(t *testing.T) {
	var fetches atomic.Int32
	var failing atomic.Bool

	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		fetches.Add(1)

		if failing.Load() {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(
			`{"keys":[{"kid":"cached-key",` +
				`"kty":"RSA","alg":"RS256","use":"sig",` +
				`"n":"AQAB","e":"AQAB"}]}`,
		))
	}))
	defer server.Close()

	resolver := NewJWKSResolver(server.URL)
	ctx := context.Background()

	if _, err := resolver.Resolve(ctx, "cached-key"); err != nil {
		t.Fatalf("initial key resolution failed: %v", err)
	}

	failing.Store(true)

	resolver.refreshMu.Lock()
	resolver.lastRefresh = time.Now().Add(-31 * time.Second)
	resolver.refreshMu.Unlock()

	if _, err := resolver.Resolve(ctx, "unknown-key"); err == nil {
		t.Fatal("expected JWKS refresh failure")
	}

	if _, err := resolver.Resolve(ctx, "cached-key"); err != nil {
		t.Fatalf("cached key unavailable after refresh failure: %v", err)
	}

	if got := fetches.Load(); got != 2 {
		t.Fatalf("expected two JWKS fetches, got %d", got)
	}
}

func TestJWKSResolverCancelWhileRefreshInFlight(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once

	unblock := func() {
		releaseOnce.Do(func() { close(release) })
	}
	defer unblock()

	var fetches atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if fetches.Add(1) == 1 {
			close(started)
		}

		<-release
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"keys":[]}`))
	}))
	defer server.Close()

	resolver := NewJWKSResolver(server.URL)

	firstDone := make(chan error, 1)
	go func() {
		_, err := resolver.Resolve(context.Background(), "first-unknown")
		firstDone <- err
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first JWKS request did not start")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	secondDone := make(chan error, 1)
	go func() {
		_, err := resolver.Resolve(ctx, "second-unknown")
		secondDone <- err
	}()

	// Give the second request time to reach the occupied refresh gate.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-secondDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("canceled request remained blocked behind JWKS refresh")
	}

	if got := fetches.Load(); got != 1 {
		t.Fatalf("expected only one JWKS HTTP request, got %d", got)
	}

	unblock()

	select {
	case err := <-firstDone:
		if !errors.Is(err, ErrJWKSKeyNotFound) {
			t.Fatalf("unexpected first request result: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first JWKS request did not complete")
	}
}
