// traygen/native_test.go

package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFitSVGPreservesAspectAndInset(t *testing.T) {
	r, err := NewRenderer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	svg, err := FitSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 50"><rect width="100" height="50" fill="#ff0000"/></svg>`), 32, .125)
	if err != nil {
		t.Fatal(err)
	}
	data, err := r.PNG(svg, 32)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	for y := range 32 {
		for x := range 32 {
			_, _, _, a := img.At(x, y).RGBA()
			inside := x >= 4 && x < 28 && y >= 10 && y < 22
			if (a != 0) != inside {
				t.Fatalf("alpha at %d,%d = %d; want painted=%v", x, y, a, inside)
			}
		}
	}
}

func TestSizeArtworkReachesWindowsICO(t *testing.T) {
	dir := t.TempDir()
	for name, shape := range map[string]string{
		"default.svg": `<rect width="32" height="32" inkscape:label="badge" style="fill:#000000"/>`,
		"tiny.svg":    `<rect x="8" y="8" width="16" height="16" inkscape:label="badge" style="fill:#000000"/>`,
	} {
		svg := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:inkscape="http://www.inkscape.org/namespaces/inkscape" viewBox="0 0 32 32">` + shape + `</svg>`
		if err := os.WriteFile(filepath.Join(dir, name), []byte(svg), 0600); err != nil {
			t.Fatal(err)
		}
	}
	config := validConfig + `
platforms: windows: {sizes: [16,32,256], padding: 0}
themes: {
	dark: {source: "default.svg", platforms: windows: sizeSources: "16": "tiny.svg"}
	light: source: "default.svg"
}`
	config = strings.ReplaceAll(strings.ReplaceAll(config, "logo-dark.svg", "default.svg"), "logo-light.svg", "default.svg")
	path := filepath.Join(dir, "trayicons.cue")
	if err := os.WriteFile(path, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	variants, err := BuildVariants(cfg, dir)
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
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "dark/windows/app-tray-dark-error.ico"))
	if err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint16(data) != 0 || binary.LittleEndian.Uint16(data[2:]) != 1 || binary.LittleEndian.Uint16(data[4:]) != 3 {
		t.Fatalf("bad ICO header: %v", data[:6])
	}
	next := 6 + 16*3
	for i, size := range []int{16, 32, 256} {
		entry := data[6+i*16 : 6+(i+1)*16]
		if entry[0] != byte(size) || entry[1] != byte(size) || binary.LittleEndian.Uint16(entry[6:]) != 32 {
			t.Fatalf("bad frame entry %d: %v", size, entry)
		}
		length := int(binary.LittleEndian.Uint32(entry[8:]))
		offset := int(binary.LittleEndian.Uint32(entry[12:]))
		if offset != next || offset+length > len(data) {
			t.Fatalf("bad frame offset/length: %d/%d", offset, length)
		}
		img, err := png.Decode(bytes.NewReader(data[offset : offset+length]))
		if err != nil {
			t.Fatal(err)
		}
		if img.Bounds().Dx() != size || img.Bounds().Dy() != size {
			t.Fatalf("wrong frame bounds: %v", img.Bounds())
		}
		r, g, b, a := img.At(size/2, size/2).RGBA()
		if r != 65535 || g != 0 || b != 0 || a != 65535 {
			t.Fatalf("slot color not applied: %d %d %d %d", r, g, b, a)
		}
		_, _, _, corner := img.At(1, 1).RGBA()
		if (corner == 0) != (size == 16) {
			t.Fatalf("size %d did not use its own artwork", size)
		}
		next = offset + length
	}
	if next != len(data) {
		t.Fatal("unexpected bytes after final frame")
	}
}

func TestTemplatePNGPreservesAlpha(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	for x, a := range []uint8{0, 127, 255} {
		source.SetNRGBA(x, 0, color.NRGBA{R: 250, G: 30, B: 80, A: a})
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, source); err != nil {
		t.Fatal(err)
	}
	data, err := templatePNG(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	for x, want := range []uint8{0, 127, 255} {
		r, g, b, a := img.At(x, 0).RGBA()
		if r != 0 || g != 0 || b != 0 || uint8(a>>8) != want {
			t.Fatalf("pixel %d = %d %d %d %d", x, r, g, b, a)
		}
	}
}

func TestPlatformConfigRejectsInvalidOverrides(t *testing.T) {
	for name, extra := range map[string]string{
		"unknown":           `themes: dark: platforms: android: source: "a.svg"`,
		"unknown-state":     `themes: dark: platforms: windows: stateSources: missing: "a.svg"`,
		"unlisted-size":     `themes: dark: platforms: windows: sizeSources: "17": "a.svg"`,
		"noncanonical-size": `themes: dark: platforms: windows: sizeSources: "016": "a.svg"`,
		"duplicate":         `platforms: windows: sizes: [16,16]`,
		"oversize":          `platforms: windows: sizes: [257]`,
		"padding":           `platforms: darwin: padding: 0.5`,
		"template":          `platforms: linux: template: true`,
		"old-sizes":         `sizes: [32]`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := LoadConfig(writeConfig(t, validConfig+extra)); err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
}

func TestRunProtectsEditableSizeSource(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "dark", "small.svg")
	if err := os.MkdirAll(filepath.Dir(source), 0700); err != nil {
		t.Fatal(err)
	}
	const artwork = `<svg viewBox="0 0 16 16"/>`
	if err := os.WriteFile(source, []byte(artwork), 0600); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, "trayicons.cue")
	if err := os.WriteFile(config, []byte(validConfig+`themes: dark: platforms: windows: sizeSources: "16": "dark/small.svg"`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := run(context.Background(), config, dir); err == nil || !strings.Contains(err.Error(), "inside generated directory") {
		t.Fatalf("source was not protected: %v", err)
	}
	data, err := os.ReadFile(source)
	if err != nil || string(data) != artwork {
		t.Fatalf("source changed: %q, %v", data, err)
	}
}
