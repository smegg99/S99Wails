// config_test.go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "trayicons.cue")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const validConfig = `
prefix: "app-tray"
slots: ["badge"]
themes: {
	dark:  source: "logo-dark.svg"
	light: source: "logo-light.svg"
}
states: {
	idle: {}
	error: slots: badge: "#FF0000"
}
`

func TestLoadConfigValid(t *testing.T) {
	cfg, err := LoadConfig(writeConfig(t, validConfig))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Prefix != "app-tray" {
		t.Errorf("prefix = %q", cfg.Prefix)
	}
	if cfg.GoPackage != "trayicons" {
		t.Errorf("default goPackage = %q, want trayicons", cfg.GoPackage)
	}
	if !cfg.TrayAdapter {
		t.Error("default trayAdapter should be true")
	}
	if len(cfg.Sizes) != 1 || cfg.Sizes[0] != 32 {
		t.Errorf("default sizes = %v, want [32]", cfg.Sizes)
	}
	if got := cfg.StateNames(); len(got) != 2 || got[0] != "error" || got[1] != "idle" {
		t.Errorf("StateNames = %v, want sorted [error idle]", got)
	}
	if got := cfg.ThemeNames(); len(got) != 2 || got[0] != "dark" || got[1] != "light" {
		t.Errorf("ThemeNames = %v", got)
	}
}

func TestLoadConfigRejectsMissingLightTheme(t *testing.T) {
	_, err := LoadConfig(writeConfig(t, `
prefix: "x"
slots: []
themes: dark: source: "d.svg"
states: idle: {}
`))
	if err == nil {
		t.Fatal("config without light theme should fail")
	}
}

func TestLoadConfigRejectsBadColor(t *testing.T) {
	_, err := LoadConfig(writeConfig(t, `
prefix: "x"
slots: ["badge"]
themes: { dark: source: "d.svg", light: source: "l.svg" }
states: error: slots: badge: "red"
`))
	if err == nil {
		t.Fatal("non-hex color should fail schema validation")
	}
}

func TestLoadConfigRejectsUndeclaredSlot(t *testing.T) {
	_, err := LoadConfig(writeConfig(t, `
prefix: "x"
slots: ["badge"]
themes: { dark: source: "d.svg", light: source: "l.svg" }
states: error: slots: dot: "#FF0000"
`))
	if err == nil || !strings.Contains(err.Error(), "undeclared slot") {
		t.Fatalf("want undeclared-slot error, got %v", err)
	}
}

func TestLoadConfigRejectsEmptySizes(t *testing.T) {
	_, err := LoadConfig(writeConfig(t, `
prefix: "x"
sizes: []
slots: []
themes: { dark: source: "d.svg", light: source: "l.svg" }
states: idle: {}
`))
	if err == nil {
		t.Fatal("empty sizes should fail")
	}
}
