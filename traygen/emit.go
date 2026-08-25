// traygen/emit.go

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// Variant is one fully-resolved theme x state icon.
type Variant struct {
	Theme, State, Name string
	SVG                []byte
}

func assetName(prefix, theme, state string) string {
	return fmt.Sprintf("%s-%s-%s", prefix, theme, state)
}

// BuildVariants resolves the source SVG for every theme x state (precedence:
// theme.stateSources > state.source > theme.source), applies slot colors
// (precedence: theme.colors > state.slots), and returns the processed SVGs.
func BuildVariants(cfg *Config, srcDir string) ([]Variant, error) {
	var variants []Variant
	for _, themeName := range cfg.ThemeNames() {
		theme := cfg.Themes[themeName]
		for _, stateName := range cfg.StateNames() {
			state := cfg.States[stateName]

			source := theme.Source
			if state.Source != "" {
				source = state.Source
			}
			if override, ok := theme.StateSources[stateName]; ok {
				source = override
			}

			raw, err := os.ReadFile(filepath.Join(srcDir, source))
			if err != nil {
				return nil, fmt.Errorf("theme %s state %s: read source: %w", themeName, stateName, err)
			}

			colors := map[string]string{}
			for slot, color := range state.Slots {
				colors[slot] = color
			}
			for slot, color := range theme.Colors[stateName] {
				colors[slot] = color
			}

			processed, err := ApplySlots(raw, cfg.Slots, colors)
			if err != nil {
				return nil, fmt.Errorf("theme %s state %s (%s): %w", themeName, stateName, source, err)
			}

			variants = append(variants, Variant{
				Theme: themeName,
				State: stateName,
				Name:  assetName(cfg.Prefix, themeName, stateName),
				SVG:   processed,
			})
		}
	}
	return variants, nil
}

// WriteAssets writes the per-platform asset tree:
//
//	<theme>/windows/<name>.png            (+ -<N>px for extra sizes)
//	<theme>/linux/<name>.svg
//	<theme>/linux-png-fallback/<name>.png (+ -<N>px for extra sizes)
func WriteAssets(cfg *Config, variants []Variant, r *Renderer, outDir string) error {
	for _, v := range variants {
		themeDir := filepath.Join(outDir, v.Theme)

		svgPath := filepath.Join(themeDir, "linux", v.Name+".svg")
		if err := writeFile(svgPath, v.SVG); err != nil {
			return err
		}

		for i, size := range cfg.Sizes {
			suffix := ""
			if i > 0 {
				suffix = fmt.Sprintf("-%dpx", size)
			}
			data, err := r.PNG(v.SVG, size)
			if err != nil {
				return fmt.Errorf("%s: %w", v.Name, err)
			}
			for _, dir := range []string{"windows", "linux-png-fallback"} {
				if err := writeFile(filepath.Join(themeDir, dir, v.Name+suffix+".png"), data); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
