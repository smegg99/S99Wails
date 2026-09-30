// traygen/main_test.go

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunFullPipeline(t *testing.T) {
	out := t.TempDir()
	if err := run(filepath.Join("testdata", "trayicons.cue"), out); err != nil {
		t.Fatalf("run: %v", err)
	}

	for _, rel := range []string{
		"dark/windows/app-tray-dark-idle.png",
		"dark/linux/scalable/status/app-tray-dark-running.svg",
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

func TestRunWritesAppIcon(t *testing.T) {
	out := t.TempDir()
	if err := run(filepath.Join("testdata", "appicon.cue"), out); err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, rel := range []string{"appicon/app-tray-app.png", "appicon/app-tray-app-16px.png"} {
		if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
}

func TestRunPrunesAppIcon(t *testing.T) {
	out := t.TempDir()
	if err := run(filepath.Join("testdata", "appicon.cue"), out); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(out, "appicon", "app-tray-app-99px.png")
	if err := os.WriteFile(stale, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(filepath.Join("testdata", "appicon.cue"), out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("stale app icon survived re-run")
	}

	// Dropping the block drops the whole generated tree with it.
	if err := run(filepath.Join("testdata", "trayicons.cue"), out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "appicon")); !os.IsNotExist(err) {
		t.Error("appicon/ survived a config without appIcon")
	}
}

func TestRunRefusesAppIconInsideGeneratedTree(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"logo-dark.svg", "logo-light.svg"} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	source := filepath.Join(dir, "appicon", "icon.svg")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("<svg/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, "trayicons.cue")
	cue := `prefix: "app-tray"
themes: {
	dark: source:  "logo-dark.svg"
	light: source: "logo-light.svg"
}
states: idle: {}
appIcon: source: "appicon/icon.svg"
`
	if err := os.WriteFile(config, []byte(cue), 0o644); err != nil {
		t.Fatal(err)
	}

	err := run(config, dir)
	if err == nil || !strings.Contains(err.Error(), "inside the generated appicon/ directory") {
		t.Fatalf("run = %v, want the source refused", err)
	}
	if _, err := os.Stat(source); err != nil {
		t.Errorf("the refused run removed the source: %v", err)
	}
}
