// proxy/proxy.go

// Package proxy reverse-proxies Wails webview requests to an s99ab daemon on a
// unix socket, flushing every write so SSE streams pass through.
//
//	application.NewServiceWithOptions(
//		proxy.New(proxy.Config{Socket: sock, Route: "/api"}),
//		application.ServiceOptions{Route: "/api"},
//	)
package proxy

import (
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/smegg99/s99ab/work/daemon"
)

// Config wires a Service to one daemon.
type Config struct {
	// Socket is the daemon's unix socket.
	Socket string

	// Route must match ServiceOptions.Route. Wails strips it, so the Service
	// prepends it again before forwarding.
	Route string
}

type Service struct {
	rp *httputil.ReverseProxy
}

func New(cfg Config) *Service {
	route := strings.TrimSuffix(cfg.Route, "/")
	rp := &httputil.ReverseProxy{
		Transport: daemon.Transport(cfg.Socket),
		// Flush every write so SSE events are not buffered.
		FlushInterval: -1,
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.URL.Scheme = "http"
			pr.Out.URL.Host = "daemon" // ignored by the unix transport, kept valid
			if route != "" {
				pr.Out.URL.Path = route + pr.Out.URL.Path
				if pr.Out.URL.RawPath != "" {
					pr.Out.URL.RawPath = route + pr.Out.URL.RawPath
				}
			}
		},
	}
	return &Service{rp: rp}
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.rp.ServeHTTP(w, r)
}
