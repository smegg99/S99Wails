// traygen/main.go

// traygen generates tray icons per theme and state from labeled SVGs, with Go accessors and previews, from a trayicons.cue.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/smegg99/s99term"
	"github.com/spf13/cobra"

	"github.com/smegg99/s99wails/traygen/v2/internal/catalog"
	"github.com/smegg99/s99wails/traygen/v2/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	a := cli.New(os.Stderr, os.Stdout, os.Environ())
	code := a.Run(ctx, newRoot(a), os.Args[1:])
	stop()
	os.Exit(code)
}

func newRoot(a *cli.App) *cobra.Command {
	var configPath, outDir string
	cmd := &cobra.Command{
		Use:  "traygen",
		Args: a.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir := outDir
			if dir == "" {
				dir = filepath.Dir(configPath)
			}
			done, err := run(cmd.Context(), configPath, dir)
			if err != nil {
				return err
			}
			l := a.Say()
			a.Console().Line(s99term.StatusOK, catalog.TraygenGenerated(l),
				s99term.F("variants", strconv.Itoa(done.variants)), s99term.F("themes", strconv.Itoa(done.themes)),
				s99term.F("states", strconv.Itoa(done.states)), s99term.F("out", dir))
			if done.appIcon != nil {
				a.Console().Line(s99term.StatusOK, catalog.TraygenAppIcon(l),
					s99term.F("sizes", fmt.Sprint(done.appIcon)), s99term.F("out", filepath.Join(dir, appIconDir)))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "trayicons.cue", "")
	cmd.Flags().StringVar(&outDir, "out", "", "")
	a.Describe(func(l *i18n.Localizer) {
		cmd.Short, cmd.Long = catalog.TraygenShort(l), catalog.TraygenLong(l)
		cmd.Flags().Lookup("config").Usage = catalog.TraygenFlagConfig(l)
		cmd.Flags().Lookup("out").Usage = catalog.TraygenFlagOut(l)
	})
	return a.Root(cmd)
}

// summary is what a run generated: tray variants per theme and state, and the app icon's sizes if any.
type summary struct {
	variants, themes, states int
	appIcon                  []int
}

func run(ctx context.Context, configPath, outDir string) (summary, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return summary{}, err
	}
	if err := checkTraySources(cfg, configPath, outDir); err != nil {
		return summary{}, err
	}

	variants, err := BuildVariants(cfg, filepath.Dir(configPath))
	if err != nil {
		return summary{}, err
	}

	r, err := NewRenderer(ctx)
	if err != nil {
		return summary{}, err
	}
	defer r.Close()

	if cfg.AppIcon != nil {
		if err := checkAppIconSource(cfg, configPath, outDir); err != nil {
			return summary{}, err
		}
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return summary{}, err
	}

	// Remove only the generated trees; outDir may also hold the sources.
	for _, dir := range append(cfg.ThemeNames(), appIconDir) {
		if err := os.RemoveAll(filepath.Join(outDir, dir)); err != nil {
			return summary{}, fmt.Errorf("clean %s: %w", dir, err)
		}
	}

	if err := WriteAssets(cfg, variants, r, outDir); err != nil {
		return summary{}, err
	}
	done := summary{variants: len(variants), themes: len(cfg.Themes), states: len(cfg.States)}
	if cfg.AppIcon != nil {
		if err := WriteAppIcon(cfg, r, filepath.Dir(configPath), outDir); err != nil {
			return summary{}, err
		}
		done.appIcon = cfg.AppIcon.Sizes
	}
	if err := GenerateGo(cfg, outDir); err != nil {
		return summary{}, err
	}
	if err := WriteSheet(cfg, variants, r, outDir); err != nil {
		return summary{}, err
	}
	return done, nil
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
