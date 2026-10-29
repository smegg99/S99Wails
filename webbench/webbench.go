// webbench/webbench.go

// Package webbench measures page load, frame timing and style cost inside
// WebKitGTK, one process per environment, using an injected probe.
package webbench

import (
	_ "embed"
	"encoding/json"
	"time"
)

// probeJS sets window.__s99bench when finished; the runner polls for it.
//
//go:embed probe.js
var probeJS string

// GPUPolicy mirrors WebKitHardwareAccelerationPolicy. The zero value is
// Always, matching the Wails default.
type GPUPolicy int

const (
	GPUAlways GPUPolicy = iota
	GPUOnDemand
	GPUNever
)

// Options is one measurement.
type Options struct {
	// URL is required.
	URL string

	// Workload is "page" (scroll the largest scroller and animate an overlay)
	// or "synthetic" (build a 1500-row list first, then the same).
	Workload string

	// FrameWindow below about 3s lets JIT warm-up frames dominate the mean.
	FrameWindow time.Duration

	// Width and Height should match the app's real window: a smaller window
	// composites less and reports better numbers.
	Width, Height int

	// GPUPolicy is passed to webkit_settings_set_hardware_acceleration_policy.
	GPUPolicy GPUPolicy

	// Timeout bounds the whole run, including load. On expiry the partial result
	// is returned with Timeout set.
	Timeout time.Duration

	// ConsoleToStdout mirrors the page console into the process stdout.
	ConsoleToStdout bool
}

func (o *Options) withDefaults() {
	if o.Workload == "" {
		o.Workload = "page"
	}
	if o.FrameWindow <= 0 {
		o.FrameWindow = 5 * time.Second
	}
	if o.Width <= 0 {
		o.Width = 1280
	}
	if o.Height <= 0 {
		o.Height = 820
	}
	if o.Timeout <= 0 {
		o.Timeout = o.FrameWindow + 40*time.Second
	}
}

// Result is one run's numbers. Zero means the page did not report the value.
type Result struct {
	Label string `json:"label,omitempty"`

	// Env is read back from the process environment, not from the request.
	Env map[string]string `json:"env"`

	// LoadFinishedMS is wall time from load_uri to WebKit's FINISHED.
	LoadFinishedMS float64 `json:"load_finished_ms"`
	DOMContentMS   float64 `json:"dom_content_loaded_ms"`
	LoadEventMS    float64 `json:"load_event_ms"`
	FirstPaintMS   float64 `json:"first_paint_ms"`
	FCPMS          float64 `json:"first_contentful_paint_ms"`

	// Steady state under the workload.
	FPS           float64 `json:"fps"`
	FrameP50MS    float64 `json:"frame_p50_ms"`
	FrameP95MS    float64 `json:"frame_p95_ms"`
	FrameWorstMS  float64 `json:"frame_worst_ms"`
	FramesCounted int     `json:"frames_counted"`

	// StyleRecalcMS is the cost of one full style invalidation (root attribute
	// toggle plus forced layout).
	StyleRecalcMS  float64 `json:"style_recalc_ms"`
	LayoutThrashMS float64 `json:"layout_thrash_ms"`
	ScriptMS       float64 `json:"script_ms"`
	ReactivityMS   float64 `json:"reactivity_ms"`

	Nodes       int `json:"nodes"`
	CSSRules    int `json:"css_rules"`
	Stylesheets int `json:"stylesheets"`

	// CPU is sampled from /proc per process (renderer, GPU) during the frame window.
	CPU map[string]float64 `json:"cpu_percent,omitempty"`

	Timeout bool   `json:"timeout,omitempty"`
	Error   string `json:"error,omitempty"`
}

// MarshalLine is the child-process wire format: one JSON object per line.
func (r Result) MarshalLine() string {
	b, err := json.Marshal(r)
	if err != nil {
		return `{"error":"marshal"}`
	}
	return string(b)
}
