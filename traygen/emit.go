// traygen/emit.go

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Variant holds resolved artwork for one platform, theme, state and physical size.
type Variant struct {
	Theme, State, Platform, Name string
	Size                         int
	SVG                          []byte
}

func assetName(prefix, theme, state string) string {
	return fmt.Sprintf("%s-%s-%s", prefix, theme, state)
}

// BuildVariants resolves platform and size artwork before applying the state's slot colours.
func BuildVariants(cfg *Config, srcDir string) ([]Variant, error) {
	var variants []Variant
	for _, themeName := range cfg.ThemeNames() {
		theme := cfg.Themes[themeName]
		for _, stateName := range cfg.StateNames() {
			state := cfg.States[stateName]
			base := theme.Source
			if state.Source != "" {
				base = state.Source
			}
			if override := theme.StateSources[stateName]; override != "" {
				base = override
			}
			colors := map[string]string{}
			for slot, color := range state.Slots {
				colors[slot] = color
			}
			for slot, color := range theme.Colors[stateName] {
				colors[slot] = color
			}
			for _, platform := range platformNames {
				profile, _ := cfg.Platform(platform)
				sources := theme.Platforms[platform]
				source := base
				if sources.Source != "" {
					source = sources.Source
				}
				if override := sources.StateSources[stateName]; override != "" {
					source = override
				}
				sizes := profile.Sizes
				if platform == "linux" {
					sizes = append([]int{0}, sizes...)
				}
				for _, size := range sizes {
					selected := source
					if override := sources.SizeSources[strconv.Itoa(size)]; override != "" {
						selected = override
					}
					raw, err := os.ReadFile(filepath.Join(srcDir, selected))
					if err != nil {
						return nil, fmt.Errorf("%s/%s/%s/%d: read %s: %w", platform, themeName, stateName, size, selected, err)
					}
					processed, err := ApplySlots(raw, cfg.Slots, colors)
					if err != nil {
						return nil, fmt.Errorf("%s/%s/%s/%d (%s): %w", platform, themeName, stateName, size, selected, err)
					}
					canvas := size
					if canvas == 0 {
						canvas = profile.Sizes[0]
					}
					processed, err = FitSVG(processed, canvas, profile.Padding)
					if err != nil {
						return nil, fmt.Errorf("%s: fit %s: %w", platform, selected, err)
					}
					variants = append(variants, Variant{Theme: themeName, State: stateName, Platform: platform,
						Name: assetName(cfg.Prefix, themeName, stateName), Size: size, SVG: processed})
				}
			}
		}
	}
	return variants, nil
}

// WriteAssets writes SVG icon themes, native-size PNGs and multi-resolution Windows ICOs.
func WriteAssets(cfg *Config, variants []Variant, r *Renderer, outDir string) error {
	windows := map[string][]iconFrame{}
	for _, v := range variants {
		profile, _ := cfg.Platform(v.Platform)
		if v.Platform == "linux" {
			dir := "scalable/status"
			if v.Size > 0 {
				dir = fmt.Sprintf("%dx%d/status", v.Size, v.Size)
			}
			if err := writeFile(filepath.Join(outDir, v.Theme, "linux", dir, v.Name+".svg"), v.SVG); err != nil {
				return err
			}
			if v.Size == 0 {
				// Flat lookup is used by SNI hosts that do not read index.theme.
				if err := writeFile(filepath.Join(outDir, v.Theme, "linux", v.Name+".svg"), v.SVG); err != nil {
					return err
				}
				continue
			}
		}
		data, err := r.PNG(v.SVG, v.Size)
		if err != nil {
			return fmt.Errorf("%s/%s/%d: %w", v.Platform, v.Name, v.Size, err)
		}
		if profile.Template {
			data, err = templatePNG(data)
			if err != nil {
				return err
			}
		}
		dir := v.Platform
		if dir == "linux" {
			dir = "linux-png-fallback"
		}
		name := v.Name + ".png"
		if v.Size != profile.Sizes[0] {
			name = fmt.Sprintf("%s-%dpx.png", v.Name, v.Size)
		}
		if err := writeFile(filepath.Join(outDir, v.Theme, dir, name), data); err != nil {
			return err
		}
		if v.Platform == "windows" {
			key := filepath.Join(v.Theme, "windows", v.Name+".ico")
			windows[key] = append(windows[key], iconFrame{Size: v.Size, PNG: data})
		}
	}
	for path, frames := range windows {
		data, err := encodeICO(frames)
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(outDir, path), data); err != nil {
			return err
		}
	}
	for _, theme := range cfg.ThemeNames() {
		if err := writeFile(filepath.Join(outDir, theme, "linux", "index.theme"), iconTheme(cfg.Platforms.Linux.Sizes)); err != nil {
			return err
		}
	}
	return nil
}

func iconTheme(sizes []int) []byte {
	dirs := []string{"scalable/status"}
	for _, size := range sizes {
		dirs = append(dirs, fmt.Sprintf("%dx%d/status", size, size))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[Icon Theme]\nName=S99Wails Tray\nDirectories=%s\n\n", strings.Join(dirs, ","))
	b.WriteString("[scalable/status]\nSize=32\nType=Scalable\nMinSize=1\nMaxSize=512\nContext=Status\n\n")
	for i, size := range sizes {
		fmt.Fprintf(&b, "[%s]\nSize=%d\nType=Fixed\nContext=Status\n\n", dirs[i+1], size)
	}
	return []byte(b.String())
}

const appIconDir = "appicon"

func appIconName(prefix string) string { return prefix + "-app" }

// WriteAppIcon renders the application artwork independently of tray profiles.
func WriteAppIcon(cfg *Config, r *Renderer, srcDir, outDir string) error {
	svg, err := os.ReadFile(filepath.Join(srcDir, cfg.AppIcon.Source))
	if err != nil {
		return fmt.Errorf("app icon: read source: %w", err)
	}
	for i, size := range cfg.AppIcon.Sizes {
		suffix := ""
		if i > 0 {
			suffix = fmt.Sprintf("-%dpx", size)
		}
		data, err := r.PNG(svg, size)
		if err != nil {
			return fmt.Errorf("app icon: %w", err)
		}
		if err := writeFile(filepath.Join(outDir, appIconDir, appIconName(cfg.Prefix)+suffix+".png"), data); err != nil {
			return err
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
