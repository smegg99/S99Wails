// shell/update/latest.go

package update

import (
	"context"
	"runtime"
	"strings"

	"github.com/smegg99/s99base/release/ghreleases"
	"github.com/smegg99/s99base/release/version"
)

const availableEvent = "s99update:available"

var goos, goarch = runtime.GOOS, runtime.GOARCH

// Check finds the newest stable release with a download for this desktop platform.
func (s *Service) Check(ctx context.Context) error {
	platform, arch := target(goos, goarch)
	if platform == "" || arch == "" || strings.TrimSpace(s.opts.Version) == "" || (s.opts.Enabled != nil && !s.opts.Enabled()) {
		return nil
	}
	if !s.checking.TryLock() {
		return nil
	}
	defer s.checking.Unlock()
	releases, err := s.client.List(ctx, 30)
	if err != nil {
		return s.report(err)
	}
	var found *Release
	for _, r := range ghreleases.Stable(releases) {
		if version.Compare(r.Version, s.opts.Version) <= 0 || (found != nil && version.Compare(r.Version, found.Version) <= 0) {
			continue
		}
		for _, asset := range r.Assets {
			if asset.Platform == platform && (asset.Arch == arch || asset.Arch == "universal") {
				url := s.opts.DownloadURL
				if url == "" {
					url = r.HTMLURL
				}
				found = &Release{Version: version.Trim(r.Version), URL: url}
				break
			}
		}
	}
	if s.opts.Enabled != nil && !s.opts.Enabled() {
		return nil
	}
	s.mu.Lock()
	err = s.loadLocked()
	if err != nil {
		s.mu.Unlock()
		return s.report(err)
	}
	s.latest = found
	notify := found != nil && found.Version != s.memory.Ignored
	s.mu.Unlock()
	if notify {
		s.emit(availableEvent, *found)
	}
	return nil
}

func target(os, arch string) (string, string) {
	switch os {
	case "linux", "windows":
	case "darwin":
		os = "macos"
	default:
		os = ""
	}
	switch arch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
	default:
		arch = ""
	}
	return os, arch
}
