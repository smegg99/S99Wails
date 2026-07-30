// proxy/proxy_test.go

package proxy_test

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smegg99/s99ab/work/daemon"
	"github.com/smegg99/s99wails/proxy"
)

// fakeDaemon serves an echo endpoint and a small SSE stream on a unix socket.
func fakeDaemon(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "d")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	sock := filepath.Join(dir, "s.sock")

	ln, err := daemon.Listen(sock)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/echo", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.URL.Path)
	})
	mux.HandleFunc("/api/v1/stream", func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("daemon response is not flushable")
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: one\n\n")
		flusher.Flush()
		_, _ = io.WriteString(w, "data: two\n\n")
		flusher.Flush()
	})

	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
	return sock
}

// the proxy puts back the mount prefix Wails strips, so the daemon sees the
// path the webview actually asked for
func TestRestoresStrippedRoute(t *testing.T) {
	sock := fakeDaemon(t)
	ts := httptest.NewServer(proxy.New(proxy.Config{Socket: sock, Route: "/api"}))
	defer ts.Close()

	// Wails strips "/api" before calling the service.
	resp, err := http.Get(ts.URL + "/v1/echo")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "/api/v1/echo" {
		t.Errorf("daemon saw %q, want /api/v1/echo", body)
	}
}

// an SSE stream survives the proxy: the content type passes through and both
// frames arrive
func TestStreamsSSEThrough(t *testing.T) {
	sock := fakeDaemon(t)
	ts := httptest.NewServer(proxy.New(proxy.Config{Socket: sock, Route: "/api"}))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/v1/stream")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}

	var got []string
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		if strings.HasPrefix(sc.Text(), "data:") {
			got = append(got, sc.Text())
		}
	}
	if len(got) != 2 || got[0] != "data: one" || got[1] != "data: two" {
		t.Errorf("streamed frames = %v, want [data: one, data: two]", got)
	}
}
