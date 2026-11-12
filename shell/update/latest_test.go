// shell/update/latest_test.go

package update

import (
	"context"
	"net/http"
	"testing"
)

func TestCheckFindsTheNewestStableReleaseWithABuildForThisPlatform(t *testing.T) {
	pre := release("v1.6.0", "app-1.6.0-linux-amd64.AppImage")
	pre["prerelease"] = true
	g := newGitHub(t, pre, release("v1.5.1", "app-1.5.1-windows-amd64.exe"), release("v1.5.0", "app-1.5.0-linux-amd64.AppImage"))
	s := New(g.options())
	emitted := events(t, s)
	if err := s.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := Release{Version: "1.5.0", URL: "https://example.test/releases/v1.5.0"}
	if s.latest == nil || *s.latest != want {
		t.Fatalf("release = %+v, want %+v", s.latest, want)
	}
	if len(*emitted) != 1 || (*emitted)[0] != want {
		t.Fatalf("events = %+v", *emitted)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.agent != "App/1.4.3" || g.authorization != "" || g.hits.Load() != 1 {
		t.Fatalf("agent = %q, authorization = %q, requests = %d", g.agent, g.authorization, g.hits.Load())
	}
}

func TestCheckReportsNothingWhenTheRunningVersionIsNewest(t *testing.T) {
	g := newGitHub(t, release("v1.4.3", "app-1.4.3-linux-amd64.AppImage"))
	opts := g.options()
	opts.Version = "v1.4.3"
	s := New(opts)
	emitted := events(t, s)
	if err := s.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s.latest != nil || len(*emitted) != 0 {
		t.Fatalf("release = %+v, events = %+v", s.latest, *emitted)
	}
}

func TestDownloadURLReplacesTheReleasePage(t *testing.T) {
	g := newGitHub(t, release("v1.5.0", "app-1.5.0-linux-amd64.AppImage"))
	opts := g.options()
	opts.DownloadURL = "https://example.test/#downloads"
	s := New(opts)
	if err := s.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s.latest == nil || s.latest.URL != opts.DownloadURL {
		t.Fatalf("release = %+v", s.latest)
	}
}

func TestIgnoredVersionStaysQuietUntilANewerOne(t *testing.T) {
	g := newGitHub(t, release("v1.5.0", "app-1.5.0-linux-amd64.AppImage"))
	opts := g.options()
	opts.Store = &memoryStore{value: Memory{Ignored: "1.5.0"}, ok: true}
	s := New(opts)
	emitted := events(t, s)
	if err := s.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(*emitted) != 0 {
		t.Fatalf("ignored version emitted %+v", *emitted)
	}
	g.set(http.StatusOK, release("v1.5.1", "app-1.5.1-linux-amd64.AppImage"))
	if err := s.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(*emitted) != 1 || (*emitted)[0].Version != "1.5.1" {
		t.Fatalf("events = %+v", *emitted)
	}
}

func TestDisabledCheckMakesNoRequest(t *testing.T) {
	g := newGitHub(t)
	opts := g.options()
	opts.Enabled = func() bool { return false }
	s := New(opts)
	if err := s.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if g.hits.Load() != 0 {
		t.Fatalf("requests = %d", g.hits.Load())
	}
}

func TestFailedCheckOnlyReachesOnError(t *testing.T) {
	g := newGitHub(t)
	g.set(http.StatusForbidden)
	opts := g.options()
	var failures []error
	opts.OnError = func(err error) { failures = append(failures, err) }
	s := New(opts)
	emitted := events(t, s)
	err := s.Check(context.Background())
	if err == nil || len(failures) != 1 || failures[0] != err {
		t.Fatalf("error = %v, reported = %v", err, failures)
	}
	if s.latest != nil || len(*emitted) != 0 {
		t.Fatalf("release = %+v, events = %+v", s.latest, *emitted)
	}
}
