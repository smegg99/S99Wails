// cmd/s99webbench/main.go

//go:build linux

// Command s99webbench measures a page in WebKitGTK and compares render paths; it needs cgo, gtk3, webkit2gtk-4.1 and a display.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/smegg99/s99term"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/smegg99/s99wails/v2/internal/cli"
	"github.com/smegg99/s99wails/v2/internal/cli/catalog"
	"github.com/smegg99/s99wails/v2/webbench"
)

type options struct {
	url, dir, workload, gpu string
	frames                  time.Duration
	width, height           int
	matrix, json, console   bool
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	a := cli.New(os.Stderr, os.Stdout, os.Environ())
	code := a.Run(ctx, newRoot(a), os.Args[1:])
	stop()
	os.Exit(code)
}

func newRoot(a *cli.App) *cobra.Command {
	var o options
	cmd := &cobra.Command{
		Use:  "s99webbench",
		Args: a.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if o.url == "" && o.dir == "" {
				return cli.Usage(errors.New(catalog.WebbenchErrTarget(a.Say())))
			}
			// A matrix run re-executes itself once per configuration.
			if o.matrix && os.Getenv(webbench.ChildEnvVar) == "" {
				return runMatrix(a, childArgs(cmd), o.json)
			}
			return measure(a, o)
		},
	}
	f := cmd.Flags()
	f.StringVar(&o.url, "url", "", "")
	f.StringVar(&o.dir, "dir", "", "")
	f.StringVar(&o.workload, "workload", "page", "")
	f.DurationVar(&o.frames, "frames", 5*time.Second, "")
	f.IntVar(&o.width, "width", 1280, "")
	f.IntVar(&o.height, "height", 820, "")
	f.StringVar(&o.gpu, "gpu", "always", "")
	f.BoolVar(&o.matrix, "matrix", false, "")
	f.BoolVar(&o.json, "json", false, "")
	f.BoolVar(&o.console, "console", false, "")
	a.Describe(func(l *i18n.Localizer) {
		cmd.Short, cmd.Long = catalog.WebbenchShort(l), catalog.WebbenchLong(l)
		for name, usage := range map[string]string{
			"url": catalog.WebbenchFlagUrl(l), "dir": catalog.WebbenchFlagDir(l),
			"workload": catalog.WebbenchFlagWorkload(l), "frames": catalog.WebbenchFlagFrames(l),
			"width": catalog.WebbenchFlagWidth(l), "height": catalog.WebbenchFlagHeight(l),
			"gpu": catalog.WebbenchFlagGpu(l), "matrix": catalog.WebbenchFlagMatrix(l),
			"json": catalog.WebbenchFlagJson(l), "console": catalog.WebbenchFlagConsole(l),
		} {
			f.Lookup(name).Usage = usage
		}
	})
	return a.Root(cmd)
}

// measure runs one measurement in this process.
func measure(a *cli.App, o options) error {
	target := o.url
	if o.dir != "" {
		served, stop, err := webbench.Serve(o.dir)
		if err != nil {
			return err
		}
		defer stop()
		target = served
	}
	res, err := webbench.Run(webbench.Options{
		URL:             target,
		Workload:        o.workload,
		FrameWindow:     o.frames,
		Width:           o.width,
		Height:          o.height,
		GPUPolicy:       parsePolicy(o.gpu),
		ConsoleToStdout: o.console,
	})
	if err != nil {
		return err
	}
	// A matrix child always prints the wire format; its parent parses the last JSON line.
	if o.json || os.Getenv(webbench.ChildEnvVar) != "" {
		_, err := fmt.Fprintln(a.Data(), res.MarshalLine())
		return err
	}
	a.Console().Print(result(a, res))
	return nil
}

// runMatrix runs one child per default configuration, each a live step, then prints the comparison.
func runMatrix(a *cli.App, args []string, asJSON bool) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	var results []webbench.Result
	for _, cfg := range webbench.DefaultMatrix() {
		step := a.Console().Begin(cfg.Label)
		res := webbench.Child(exe, args, cfg)
		if res.Error != "" {
			step.End(s99term.StatusFailed, s99term.F("error", res.Error))
		} else {
			step.End(s99term.StatusOK, s99term.F("fps", fmt.Sprintf("%.1f", res.FPS)))
		}
		results = append(results, res)
	}
	if asJSON {
		enc := json.NewEncoder(a.Data())
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}
	a.Console().Print(comparison(a, results))
	return nil
}

// childArgs are the flags this run was given, minus --matrix, so a child measures instead of recursing.
func childArgs(cmd *cobra.Command) []string {
	var args []string
	cmd.Flags().Visit(func(f *pflag.Flag) {
		if f.Name != "matrix" {
			args = append(args, "--"+f.Name+"="+f.Value.String())
		}
	})
	return args
}

func parsePolicy(s string) webbench.GPUPolicy {
	switch s {
	case "never":
		return webbench.GPUNever
	case "ondemand":
		return webbench.GPUOnDemand
	default:
		return webbench.GPUAlways
	}
}
