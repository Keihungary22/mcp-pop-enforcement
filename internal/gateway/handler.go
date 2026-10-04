package gateway

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Handler forwards requests to a configured MCP upstream.
// HandlerはRequestを設定されたMCP Upstreamへ転送する。
type Handler struct {
	proxy *httputil.ReverseProxy
}

// New creates a new MCP forwarding handler.
// NewはMCP Forwarding Handlerを作成する。
func New(upstreamURL string) (*Handler, error) {
	target, err := url.Parse(upstreamURL)
	if err != nil {
		return nil, fmt.Errorf("parse upstream MCP URL: %w", err)
	}

	if target.Scheme == "" || target.Host == "" {
		return nil, fmt.Errorf("upstream MCP URL must contain scheme and host")
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
			http.Error(w, "upstream MCP service unavailable", http.StatusBadGateway)
		},
	}

	return &Handler{proxy: proxy}, nil
}

// ServeHTTP forwards a request to the MCP upstream.
// ServeHTTPはRequestをMCP Upstreamへ転送する。
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
	h.proxy.ServeHTTP(w, r)
}
