// webbench/matrix.go

package webbench

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// Config is one comparison row. WebKitGTK reads its render settings once at
// startup, so each row runs in its own child process.
type Config struct {
	Label string
	Env   map[string]string
}

// DefaultMatrix covers every render-path rung on both display backends. An empty
// value unsets the variable: WebKit tests for presence, so "0" still disables.
func DefaultMatrix() []Config {
	rungs := []struct {
		name string
		env  map[string]string
	}{
		{"auto", nil},
		{"nogbm", map[string]string{"WEBKIT_DMABUF_RENDERER_DISABLE_GBM": "1"}},
		{"shm", map[string]string{"WEBKIT_DMABUF_RENDERER_FORCE_SHM": "1"}},
		{"legacy", map[string]string{"WEBKIT_DISABLE_DMABUF_RENDERER": "1"}},
	}
	backends := []struct{ name, gdk string }{{"x11", "x11"}, {"native", ""}}

	configs := make([]Config, 0, len(rungs)*len(backends))
	for _, b := range backends {
		for _, r := range rungs {
			env := map[string]string{"GDK_BACKEND": b.gdk}
			// State every variable on every row so the parent shell's value is not inherited.
			for _, key := range []string{
				"WEBKIT_DISABLE_DMABUF_RENDERER",
				"WEBKIT_DMABUF_RENDERER_DISABLE_GBM",
				"WEBKIT_DMABUF_RENDERER_FORCE_SHM",
			} {
				env[key] = r.env[key]
			}
			configs = append(configs, Config{Label: r.name + " + " + b.name, Env: env})
		}
	}
	return configs
}

// ChildEnvVar marks a re-executed child process.
const ChildEnvVar = "S99WEBBENCH_CHILD"

// Matrix re-executes exe once per config; the child calls Run and prints
// Result.MarshalLine. Child stderr is passed through.
func Matrix(exe string, args []string, configs []Config, progress io.Writer) ([]Result, error) {
	results := make([]Result, 0, len(configs))
	for _, cfg := range configs {
		if progress != nil {
			fmt.Fprintf(progress, "running: %s\n", cfg.Label)
		}
		res, err := runChild(exe, args, cfg)
		if err != nil {
			res = Result{Label: cfg.Label, Error: err.Error()}
		}
		res.Label = cfg.Label
		results = append(results, res)
	}
	return results, nil
}

func runChild(exe string, args []string, cfg Config) (Result, error) {
	cmd := exec.Command(exe, args...)
	cmd.Env = childEnv(cfg.Env)
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return Result{}, fmt.Errorf("child: %w", err)
	}

	// The last JSON line on stdout is the result.
	var last string
	scan := bufio.NewScanner(strings.NewReader(string(out)))
	scan.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scan.Scan() {
		if line := strings.TrimSpace(scan.Text()); strings.HasPrefix(line, "{") {
			last = line
		}
	}
	if last == "" {
		return Result{}, fmt.Errorf("child printed no result")
	}
	var res Result
	if err := json.Unmarshal([]byte(last), &res); err != nil {
		return Result{}, fmt.Errorf("child result: %w", err)
	}
	return res, nil
}

func childEnv(overrides map[string]string) []string {
	drop := map[string]bool{ChildEnvVar: true}
	for k, v := range overrides {
		if v == "" {
			drop[k] = true
		}
	}
	env := make([]string, 0, len(os.Environ())+len(overrides)+1)
	for _, kv := range os.Environ() {
		if k, _, ok := strings.Cut(kv, "="); ok && drop[k] {
			continue
		}
		env = append(env, kv)
	}
	for k, v := range overrides {
		if v != "" {
			env = append(env, k+"="+v)
		}
	}
	return append(env, ChildEnvVar+"=1")
}

// Compare renders results fastest first, with each frame rate as a ratio to the slowest.
func Compare(results []Result) string {
	ordered := append([]Result(nil), results...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].FPS > ordered[j].FPS })

	worst := 0.0
	for _, r := range ordered {
		if r.FPS > 0 && (worst == 0 || r.FPS < worst) {
			worst = r.FPS
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%-38s %7s %8s %8s %9s %9s %9s\n",
		"configuration", "fps", "p95 ms", "worst", "load ms", "style ms", "gpu %")
	b.WriteString(strings.Repeat("-", 94) + "\n")
	for _, r := range ordered {
		ratio := ""
		if worst > 0 && r.FPS > 0 {
			ratio = fmt.Sprintf("  (%.1fx)", r.FPS/worst)
		}
		if r.Error != "" && r.FPS == 0 {
			fmt.Fprintf(&b, "%-38s %7s %8s %8s %9s %9s %9s   %s\n",
				truncate(r.Label, 38), "-", "-", "-", "-", "-", "-", r.Error)
			continue
		}
		fmt.Fprintf(&b, "%-38s %7.1f %8.1f %8.1f %9.0f %9.1f %9.1f%s\n",
			truncate(r.Label, 38), r.FPS, r.FrameP95MS, r.FrameWorstMS,
			r.LoadFinishedMS, r.StyleRecalcMS, r.CPU["WebKitGPUProcess"], ratio)
	}
	return b.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
