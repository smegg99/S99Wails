// cmd/s99webbench/main.go

// Command s99webbench measures a page inside WebKitGTK and compares render paths.
// It needs cgo, gtk3, webkit2gtk-4.1 and a display.
//
//	s99webbench -dir desktop/frontend/.output/public -matrix
//	s99webbench -url http://127.0.0.1:3400/
//	s99webbench -dir dist -workload synthetic -matrix
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/smegg99/s99wails/v2/webbench"
)

func main() {
	var (
		url      = flag.String("url", "", "URL to measure")
		dir      = flag.String("dir", "", "directory of a built SPA to serve over loopback and measure")
		workload = flag.String("workload", "page", "page (drive the page's own content) or synthetic (build a list first)")
		frames   = flag.Duration("frames", 5*time.Second, "how long to count frames")
		width    = flag.Int("width", 1280, "window width")
		height   = flag.Int("height", 820, "window height")
		gpu      = flag.String("gpu", "always", "hardware acceleration policy: always, ondemand, never")
		matrix   = flag.Bool("matrix", false, "run the default render-path comparison, one child process per configuration")
		asJSON   = flag.Bool("json", false, "print results as JSON")
		console  = flag.Bool("console", false, "mirror the page's console to stdout")
	)
	flag.Parse()

	if *url == "" && *dir == "" {
		fmt.Fprintln(os.Stderr, "s99webbench: one of -url or -dir is required")
		flag.Usage()
		os.Exit(2)
	}

	// A matrix run re-executes itself once per configuration.
	if *matrix && os.Getenv(webbench.ChildEnvVar) == "" {
		os.Exit(runMatrix())
	}

	target := *url
	if *dir != "" {
		served, stop, err := webbench.Serve(*dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "s99webbench:", err)
			os.Exit(1)
		}
		defer stop()
		target = served
	}

	res, err := webbench.Run(webbench.Options{
		URL:             target,
		Workload:        *workload,
		FrameWindow:     *frames,
		Width:           *width,
		Height:          *height,
		GPUPolicy:       parsePolicy(*gpu),
		ConsoleToStdout: *console,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "s99webbench:", err)
		os.Exit(1)
	}

	// A child in a matrix run always prints the wire format; its parent parses
	// the last JSON line and formats for the human.
	if *asJSON || os.Getenv(webbench.ChildEnvVar) != "" {
		fmt.Println(res.MarshalLine())
		return
	}
	fmt.Println(res.String())
	if len(res.CPU) > 0 {
		fmt.Printf("  cpu         ")
		for name, pct := range res.CPU {
			fmt.Printf("%s %.1f%%  ", name, pct)
		}
		fmt.Println()
	}
}

// runMatrix re-executes this binary once per configuration with -matrix dropped,
// so the child measures rather than recursing.
func runMatrix() int {
	child := make([]string, 0, len(os.Args))
	for _, a := range os.Args[1:] {
		if a == "-matrix" || a == "--matrix" || a == "-matrix=true" || a == "--matrix=true" {
			continue
		}
		child = append(child, a)
	}

	exe, err := os.Executable()
	if err != nil {
		exe = os.Args[0]
	}

	results, err := webbench.Matrix(exe, child, webbench.DefaultMatrix(), os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "s99webbench:", err)
		return 1
	}

	for _, a := range os.Args[1:] {
		if a == "-json" || a == "--json" {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(results)
			return 0
		}
	}
	fmt.Print("\n" + webbench.Compare(results))
	return 0
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
