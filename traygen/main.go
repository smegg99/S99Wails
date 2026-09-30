// traygen/main.go

// traygen generates tray icons per theme and state from labeled SVGs, with Go
// accessors and preview sheets. Run next to a trayicons.cue (format: schema.cue):
//
//	//go:generate go tool traygen
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	configPath := flag.String("config", "trayicons.cue", "path to the trayicons CUE config")
	outDir := flag.String("out", "", "output directory (default: the config file's directory)")
	flag.Parse()

	dir := *outDir
	if dir == "" {
		dir = filepath.Dir(*configPath)
	}

	if err := run(*configPath, dir); err != nil {
		log.Fatal(err)
	}
}

func run(configPath, outDir string) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return err
	}
	if err := checkTraySources(cfg, configPath, outDir); err != nil {
		return err
	}

	variants, err := BuildVariants(cfg, filepath.Dir(configPath))
	if err != nil {
		return err
	}

	r, err := NewRenderer(context.Background())
	if err != nil {
		return err
	}
	defer r.Close()

	if cfg.AppIcon != nil {
		if err := checkAppIconSource(cfg, configPath, outDir); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	// Remove only the generated trees; outDir may also hold the sources.
	for _, dir := range append(cfg.ThemeNames(), appIconDir) {
		if err := os.RemoveAll(filepath.Join(outDir, dir)); err != nil {
			return fmt.Errorf("clean %s: %w", dir, err)
		}
	}

	if err := WriteAssets(cfg, variants, r, outDir); err != nil {
		return err
	}
	if cfg.AppIcon != nil {
		if err := WriteAppIcon(cfg, r, filepath.Dir(configPath), outDir); err != nil {
			return err
		}
	}
	if err := GenerateGo(cfg, outDir); err != nil {
		return err
	}
	if err := WriteSheet(cfg, variants, r, outDir); err != nil {
		return err
	}

	fmt.Printf("traygen: %d variants (%d themes x %d states) -> %s\n",
		len(variants), len(cfg.Themes), len(cfg.States), outDir)
	if cfg.AppIcon != nil {
		fmt.Printf("traygen: app icon at %v px -> %s\n",
			cfg.AppIcon.Sizes, filepath.Join(outDir, appIconDir))
	}
	return nil
}

// checkTraySources protects editable artwork from the generated-tree cleanup.
func checkTraySources(cfg *Config, configPath, outDir string) error {
	var sources []string
	for _, state := range cfg.States {
		sources = append(sources, state.Source)
	}
	for _, theme := range cfg.Themes {
		sources = append(sources, theme.Source)
		for _, source := range theme.StateSources {
			sources = append(sources, source)
		}
		for _, platform := range theme.Platforms {
			sources = append(sources, platform.Source)
			for _, source := range platform.StateSources {
				sources = append(sources, source)
			}
			for _, source := range platform.SizeSources {
				sources = append(sources, source)
			}
		}
	}
	for _, source := range sources {
		if source == "" {
			continue
		}
		path, err := filepath.Abs(filepath.Join(filepath.Dir(configPath), source))
		if err != nil {
			return err
		}
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			path = resolved
		}
		for _, dir := range append(cfg.ThemeNames(), appIconDir) {
			generated, err := filepath.Abs(filepath.Join(outDir, dir))
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(generated, path)
			if err != nil {
				return err
			}
			if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return fmt.Errorf("tray source %s is inside generated directory %s", source, dir)
			}
		}
	}
	return nil
}

// checkAppIconSource refuses a source inside <out>/appicon/, which run removes first.
func checkAppIconSource(cfg *Config, configPath, outDir string) error {
	source, err := filepath.Abs(filepath.Join(filepath.Dir(configPath), cfg.AppIcon.Source))
	if err != nil {
		return fmt.Errorf("app icon: %w", err)
	}
	generated, err := filepath.Abs(filepath.Join(outDir, appIconDir))
	if err != nil {
		return fmt.Errorf("app icon: %w", err)
	}
	rel, err := filepath.Rel(generated, source)
	if err != nil {
		return fmt.Errorf("app icon: %w", err)
	}
	if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("app icon source %s is inside the generated %s/ directory, which every run removes", cfg.AppIcon.Source, appIconDir)
	}
	return nil
}
