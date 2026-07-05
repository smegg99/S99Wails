// main_test.go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunFullPipeline(t *testing.T) {
	out := t.TempDir()
	if err := run(filepath.Join("testdata", "trayicons.cue"), out); err != nil {
		t.Fatalf("run: %v", err)
	}

	for _, rel := range []string{
		"dark/windows/app-tray-dark-idle.png",
		"dark/linux/app-tray-dark-running.svg",
		"light/linux-png-fallback/app-tray-light-error.png",
		"icons_gen.go",
		"preview.png",
	} {
		if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
}

func TestRunBadConfig(t *testing.T) {
	if err := run(filepath.Join("testdata", "nope.cue"), t.TempDir()); err == nil {
		t.Fatal("missing config must error")
	}
}

func TestRunPrunesStaleAssets(t *testing.T) {
	out := t.TempDir()
	if err := run(filepath.Join("testdata", "trayicons.cue"), out); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(out, "dark", "windows", "app-tray-dark-gone.png")
	if err := os.WriteFile(stale, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(filepath.Join("testdata", "trayicons.cue"), out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("stale asset survived re-run")
	}
	if _, err := os.Stat(filepath.Join(out, "dark", "windows", "app-tray-dark-idle.png")); err != nil {
		t.Errorf("fresh asset missing after prune+regen: %v", err)
	}
}
