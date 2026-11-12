// shell/update/latest_edge_test.go

package update

import (
	"context"
	"errors"
	"testing"
)

func TestEmptyVersionAndUnsupportedPlatformMakeNoRequest(t *testing.T) {
	for _, tc := range []struct{ os, arch, version string }{
		{"linux", "amd64", ""}, {"linux", "amd64", "  "},
		{"android", "arm64", "1.4.3"}, {"js", "wasm", "1.4.3"},
	} {
		t.Run(tc.os+"/"+tc.version, func(t *testing.T) {
			g := newGitHub(t)
			goos, goarch = tc.os, tc.arch
			opts := g.options()
			opts.Version = tc.version
			if err := New(opts).Check(context.Background()); err != nil {
				t.Fatal(err)
			}
			if g.hits.Load() != 0 {
				t.Fatalf("requests = %d", g.hits.Load())
			}
		})
	}
}

func TestCheckSelectsByVersionAndSkipsDraftsAndOtherArchitectures(t *testing.T) {
	draft := release("v9.0.0", "app-9.0.0-linux-amd64.AppImage")
	draft["draft"] = true
	g := newGitHub(t, draft,
		release("v2.0.0", "app-2.0.0-linux-arm64.deb"),
		release("v1.5.0", "app-1.5.0-linux-amd64.AppImage"),
		release("v1.10.0", "app-1.10.0-linux-universal.tar.gz"),
	)
	s := New(g.options())
	if err := s.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s.latest == nil || s.latest.Version != "1.10.0" {
		t.Fatalf("release = %+v", s.latest)
	}
}

func TestTargetMapsDesktopPlatforms(t *testing.T) {
	for _, tc := range []struct{ os, arch, platform, targetArch string }{
		{"linux", "amd64", "linux", "x86_64"},
		{"windows", "amd64", "windows", "x86_64"},
		{"darwin", "arm64", "macos", "arm64"},
		{"linux", "arm64", "linux", "arm64"},
		{"android", "arm64", "", "arm64"},
		{"linux", "386", "linux", ""},
	} {
		platform, arch := target(tc.os, tc.arch)
		if platform != tc.platform || arch != tc.targetArch {
			t.Errorf("target(%q, %q) = %q, %q", tc.os, tc.arch, platform, arch)
		}
	}
}

func TestMemoryErrorsAreReportedWithoutLosingStoredNotes(t *testing.T) {
	failure := errors.New("store unavailable")
	store := &memoryStore{value: Memory{Notes: "1.4.3"}, ok: true, loadErr: failure}
	var got []error
	s := New(Options{Store: store, OnError: func(err error) { got = append(got, err) }})
	if err := s.Ignore("v1.5.0"); err != failure {
		t.Fatalf("load failure = %v", err)
	}
	store.loadErr = nil
	store.saveErr = failure
	if err := s.Ignore("v1.5.0"); err != failure {
		t.Fatalf("save failure = %v", err)
	}
	store.saveErr = nil
	if err := s.Ignore("v1.5.0"); err != nil {
		t.Fatal(err)
	}
	memory, _, _ := store.Load()
	if len(got) != 2 || memory != (Memory{Ignored: "1.5.0", Notes: "1.4.3"}) {
		t.Fatalf("errors = %v, memory = %+v", got, memory)
	}
}
