// shell/update/fake_test.go

package update

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

type memoryStore struct {
	mu               sync.Mutex
	value            Memory
	ok               bool
	loadErr, saveErr error
}

func (m *memoryStore) Load() (Memory, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.value, m.ok, m.loadErr
}

func (m *memoryStore) Save(value Memory) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.saveErr != nil {
		return m.saveErr
	}
	m.value, m.ok = value, true
	return nil
}

type githubServer struct {
	mu                   sync.Mutex
	body                 []map[string]any
	status               int
	hits                 atomic.Int32
	server               *httptest.Server
	agent, authorization string
}

func release(tag, asset string) map[string]any {
	return map[string]any{
		"tag_name": tag, "html_url": "https://example.test/releases/" + tag,
		"assets": []map[string]string{{"name": asset, "browser_download_url": "https://example.test/" + asset}},
	}
}

func newGitHub(t *testing.T, releases ...map[string]any) *githubServer {
	t.Helper()
	g := &githubServer{body: releases, status: http.StatusOK}
	g.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.mu.Lock()
		defer g.mu.Unlock()
		g.hits.Add(1)
		g.agent, g.authorization = r.UserAgent(), r.Header.Get("Authorization")
		if r.Method != http.MethodGet || r.URL.Path != "/repos/smegg99/App/releases" || r.URL.RawQuery != "per_page=30" {
			t.Errorf("request = %s %s", r.Method, r.URL)
		}
		w.WriteHeader(g.status)
		_ = json.NewEncoder(w).Encode(g.body)
	}))
	t.Cleanup(g.server.Close)
	previousOS, previousArch := goos, goarch
	goos, goarch = "linux", "amd64"
	t.Cleanup(func() { goos, goarch = previousOS, previousArch })
	return g
}

func (g *githubServer) options() Options {
	return Options{Owner: "smegg99", Repo: "App", Version: "1.4.3", APIBase: g.server.URL, HTTPClient: g.server.Client(), Store: &memoryStore{}}
}

func (g *githubServer) set(status int, releases ...map[string]any) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.status, g.body = status, releases
}

func events(t *testing.T, s *Service) *[]Release {
	t.Helper()
	var got []Release
	s.emit = func(name string, data any) {
		if name != "s99update:available" {
			t.Errorf("event = %q", name)
		}
		r, ok := data.(Release)
		if !ok {
			t.Errorf("event payload = %T", data)
			return
		}
		got = append(got, r)
	}
	return &got
}
