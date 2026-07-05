// main.go
//
// traygen pregenerates tray icons from labeled SVGs: one icon per
// theme x state, plus typed Go accessors and a preview contact sheet.
// Consume as a Go tool dependency:
//
//	//go:generate go tool traygen
//
// next to a trayicons.cue (see schema.cue for the format).
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
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

	variants, err := BuildVariants(cfg, filepath.Dir(configPath))
	if err != nil {
		return err
	}

	r, err := NewRenderer(context.Background())
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	// Theme dirs are wholly generated; prune them so removed states or sizes
	// don't leave stale assets behind (outDir may also hold the sources, so
	// never remove more than the generated trees).
	for _, theme := range cfg.ThemeNames() {
		if err := os.RemoveAll(filepath.Join(outDir, theme)); err != nil {
			return fmt.Errorf("clean %s: %w", theme, err)
		}
	}

	if err := WriteAssets(cfg, variants, r, outDir); err != nil {
		return err
	}
	if err := GenerateGo(cfg, outDir); err != nil {
		return err
	}
	if err := WriteSheet(cfg, variants, r, outDir); err != nil {
		return err
	}

	fmt.Printf("traygen: %d variants (%d themes x %d states) -> %s\n",
		len(variants), len(cfg.Themes), len(cfg.States), outDir)
	return nil
}
