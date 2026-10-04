package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGatewayForwardsRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" {
			t.Fatalf("unexpected upstream path: %s", r.URL.Path)
		}

		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	handler, err := New(upstream.URL + "/mcp")
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "http://gateway.test/mcp", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGatewayRejectsInvalidUpstreamURL(t *testing.T) {
	_, err := New("not-a-valid-upstream")

	if err == nil {
		t.Fatal("expected invalid upstream URL to return an error")
	}
}
