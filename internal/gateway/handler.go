package gateway

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Handler forwards requests to a configured MCP upstream.
type Handler struct {
	proxy *httputil.ReverseProxy
}

// New creates an MCP forwarding handler using standard HTTP transport.
func New(upstreamURL string) (*Handler, error) {
	return NewWithSocket(upstreamURL, "")
}

// NewWithSocket creates an MCP forwarding handler.
// If socketPath is set, upstream HTTP requests use a Unix domain socket.
func NewWithSocket(upstreamURL, socketPath string) (*Handler, error) {
	target, err := url.Parse(upstreamURL)
	if err != nil {
		return nil, fmt.Errorf("parse upstream MCP URL: %w", err)
	}

	if target.Scheme == "" || target.Host == "" {
		return nil, fmt.Errorf("upstream MCP URL must contain scheme and host")
	}

	if socketPath != "" && target.Scheme != "http" {
		return nil, fmt.Errorf("Unix socket upstream requires HTTP scheme")
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = target.Path
			req.URL.RawPath = target.RawPath
			req.Host = target.Host
		},
		ErrorHandler: func(w http.ResponseWriter, req *http.Request, err error) {
			log.Printf("upstream request failed: %v", err)
			http.Error(
				w,
				"upstream MCP service unavailable",
				http.StatusBadGateway,
			)
		},
	}

	if socketPath != "" {
		transport := http.DefaultTransport.(*http.Transport).Clone()

		transport.Proxy = nil
		transport.DialContext = func(
			ctx context.Context,
			_, _ string,
		) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", socketPath)
		}

		proxy.Transport = transport
	}

	return &Handler{proxy: proxy}, nil
}

// ServeHTTP forwards an MCP request to the configured upstream.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
	h.proxy.ServeHTTP(w, r)
}
