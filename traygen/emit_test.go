// emit_test.go
package main

import (
	"bytes"
	"context"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadTestConfig(t *testing.T) *Config {
	t.Helper()
	cfg, err := LoadConfig(filepath.Join("testdata", "trayicons.cue"))
	if err != nil {
		t.Fatalf("LoadConfig(testdata): %v", err)
	}
	return cfg
}

func TestBuildVariants(t *testing.T) {
	cfg := loadTestConfig(t)
	variants, err := BuildVariants(cfg, "testdata")
	if err != nil {
		t.Fatalf("BuildVariants: %v", err)
	}
	// 2 themes x 3 states
	if len(variants) != 6 {
		t.Fatalf("got %d variants, want 6", len(variants))
	}

	byName := map[string]Variant{}
	for _, v := range variants {
		byName[v.Name] = v
	}

	// idle hides the badge
	idle := byName["app-tray-dark-idle"]
	if !strings.Contains(string(idle.SVG), `display="none"`) {
		t.Error("idle badge should be hidden")
	}
	// running recolors it; light theme override wins over the state color
	if !strings.Contains(string(byName["app-tray-dark-running"].SVG), "fill:#42A5F5") {
		t.Error("dark running badge should be #42A5F5")
	}
	if !strings.Contains(string(byName["app-tray-light-running"].SVG), "fill:#1E88E5") {
		t.Error("light running badge should use the theme override #1E88E5")
	}
}

func TestWriteAssets(t *testing.T) {
	cfg := loadTestConfig(t)
	variants, err := BuildVariants(cfg, "testdata")
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewRenderer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	out := t.TempDir()
	if err := WriteAssets(cfg, variants, r, out); err != nil {
		t.Fatalf("WriteAssets: %v", err)
	}

	// primary size uses the bare name, extra sizes get -<N>px
	mustExist := []string{
		"dark/windows/app-tray-dark-idle.png",
		"dark/windows/app-tray-dark-idle-24px.png",
		"dark/linux/app-tray-dark-idle.svg",
		"dark/linux-png-fallback/app-tray-dark-idle.png",
		"light/linux-png-fallback/app-tray-light-error.png",
	}
	for _, rel := range mustExist {
		if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}

	// PNG really is the primary size
	data, err := os.ReadFile(filepath.Join(out, "dark/windows/app-tray-dark-idle.png"))
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 32 {
		t.Errorf("primary png is %dpx, want 32", b.Dx())
	}
}

func TestBuildVariantsSourcePrecedence(t *testing.T) {
	dir := t.TempDir()
	write := func(name, marker string) {
		svg := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:inkscape="http://www.inkscape.org/namespaces/inkscape" data-marker="` + marker + `"><rect inkscape:label="badge" style="fill:#000000"/></svg>`
		if err := os.WriteFile(filepath.Join(dir, name), []byte(svg), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("theme.svg", "from-theme")
	write("state.svg", "from-state")
	write("override.svg", "from-override")

	cfg := &Config{
		Prefix: "p",
		Slots:  []string{"badge"},
		Sizes:  []int{32},
		Themes: map[string]Theme{
			"dark":  {Source: "theme.svg", StateSources: map[string]string{"special": "override.svg"}},
			"light": {Source: "theme.svg"},
		},
		States: map[string]State{
			"plain":   {},
			"special": {Source: "state.svg"},
		},
	}

	variants, err := BuildVariants(cfg, dir)
	if err != nil {
		t.Fatalf("BuildVariants: %v", err)
	}
	byName := map[string]string{}
	for _, v := range variants {
		byName[v.Name] = string(v.SVG)
	}

	if !strings.Contains(byName["p-dark-plain"], "from-theme") {
		t.Error("plain state should use theme.Source")
	}
	if !strings.Contains(byName["p-light-special"], "from-state") {
		t.Error("state.Source should beat theme.Source")
	}
	if !strings.Contains(byName["p-dark-special"], "from-override") {
		t.Error("theme.StateSources should beat state.Source")
	}
}
