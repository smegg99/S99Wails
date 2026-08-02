//go:build linux

// webbench/run_linux.go

package webbench

/*
#cgo pkg-config: gtk+-3.0 webkit2gtk-4.1
#include <gtk/gtk.h>
#include <webkit2/webkit2.h>
#include <stdlib.h>

extern void s99benchLoadChanged(WebKitWebView *view, WebKitLoadEvent ev, gpointer data);
extern gboolean s99benchPoll(gpointer data);
extern gboolean s99benchBail(gpointer data);
extern void s99benchEvalDone(GObject *src, GAsyncResult *res, gpointer data);

// g_signal_connect is a macro, so the connection has to be made from C.
static void s99bench_connect_load(WebKitWebView *view) {
	g_signal_connect(view, "load-changed", G_CALLBACK(s99benchLoadChanged), NULL);
}

static void s99bench_eval(WebKitWebView *view, const char *script) {
	webkit_web_view_evaluate_javascript(view, script, -1, NULL, NULL, NULL,
		(GAsyncReadyCallback)s99benchEvalDone, NULL);
}

// Returns a newly allocated C string the caller frees; no GObject crosses to Go.
static char *s99bench_eval_finish(WebKitWebView *view, GAsyncResult *res) {
	GError *err = NULL;
	JSCValue *val = webkit_web_view_evaluate_javascript_finish(view, res, &err);
	if (val == NULL) {
		if (err != NULL) g_error_free(err);
		return NULL;
	}
	char *s = jsc_value_to_string(val);
	g_object_unref(val);
	return s;
}
*/
import "C"

import (
	"encoding/json"
	"errors"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

// active is package state because only one run per process is valid: the render
// path environment is read once at GTK and WebKit startup.
var active *runState

type runState struct {
	view    *C.WebKitWebView
	opts    Options
	started time.Time
	loaded  time.Duration
	result  Result
	done    bool
	polling bool
}

// Run loads opts.URL in a WebKitGTK window and returns the probe's measurements.
// Call it from the main goroutine; it locks the OS thread for GTK.
func Run(opts Options) (Result, error) {
	opts.withDefaults()
	if opts.URL == "" {
		return Result{}, errors.New("webbench: URL is required")
	}
	if active != nil {
		return Result{}, errors.New("webbench: one run per process; use Matrix for several")
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if C.gtk_init_check(nil, nil) == 0 {
		return Result{}, errors.New("webbench: gtk_init_check failed (no display?)")
	}

	st := &runState{opts: opts, started: time.Now()}
	st.result.Env = readEnv()
	active = st
	defer func() { active = nil }()

	window := (*C.GtkWindow)(unsafe.Pointer(C.gtk_window_new(C.GTK_WINDOW_TOPLEVEL)))
	C.gtk_window_set_default_size(window, C.gint(opts.Width), C.gint(opts.Height))
	C.gtk_window_set_title(window, C.CString("s99webbench"))

	view := (*C.WebKitWebView)(unsafe.Pointer(C.webkit_web_view_new()))
	st.view = view

	settings := C.webkit_web_view_get_settings(view)
	C.webkit_settings_set_hardware_acceleration_policy(settings, policyToC(opts.GPUPolicy))
	if opts.ConsoleToStdout {
		C.webkit_settings_set_enable_write_console_messages_to_stdout(settings, C.TRUE)
	}

	C.gtk_container_add((*C.GtkContainer)(unsafe.Pointer(window)), (*C.GtkWidget)(unsafe.Pointer(view)))
	// The window must be shown: an unmapped webview does not composite.
	C.gtk_widget_show_all((*C.GtkWidget)(unsafe.Pointer(window)))

	C.s99bench_connect_load(view)

	cURL := C.CString(opts.URL)
	defer C.free(unsafe.Pointer(cURL))
	C.webkit_web_view_load_uri(view, cURL)

	stopCPU := sampleCPU([]string{"WebKitWebProcess", "WebKitGPUProcess", "WebKitNetworkProcess"})

	C.g_timeout_add(C.guint(opts.Timeout.Milliseconds()), C.GSourceFunc(C.s99benchBail), nil)
	C.gtk_main()

	st.result.CPU = stopCPU()
	st.result.LoadFinishedMS = float64(st.loaded.Microseconds()) / 1000
	return st.result, nil
}

func policyToC(p GPUPolicy) C.WebKitHardwareAccelerationPolicy {
	switch p {
	case GPUNever:
		return C.WEBKIT_HARDWARE_ACCELERATION_POLICY_NEVER
	case GPUOnDemand:
		return C.WEBKIT_HARDWARE_ACCELERATION_POLICY_ON_DEMAND
	default:
		return C.WEBKIT_HARDWARE_ACCELERATION_POLICY_ALWAYS
	}
}

//export s99benchLoadChanged
func s99benchLoadChanged(view *C.WebKitWebView, ev C.WebKitLoadEvent, data C.gpointer) {
	_, _ = view, data
	if active == nil || ev != C.WEBKIT_LOAD_FINISHED || active.polling {
		return
	}
	active.loaded = time.Since(active.started)
	active.polling = true

	script := strings.NewReplacer(
		"__WORKLOAD__", active.opts.Workload,
		"__FRAME_MS__", strconv.Itoa(int(active.opts.FrameWindow.Milliseconds())),
	).Replace(probeJS)

	cs := C.CString(script)
	defer C.free(unsafe.Pointer(cs))
	C.s99bench_eval(active.view, cs)

	C.g_timeout_add(250, C.GSourceFunc(C.s99benchPoll), nil)
}

//export s99benchPoll
func s99benchPoll(data C.gpointer) C.gboolean {
	_ = data
	if active == nil || active.done {
		return C.FALSE
	}
	cs := C.CString(`window.__s99bench ? JSON.stringify(window.__s99bench) : ""`)
	defer C.free(unsafe.Pointer(cs))
	C.s99bench_eval(active.view, cs)
	return C.TRUE
}

//export s99benchEvalDone
func s99benchEvalDone(src *C.GObject, res *C.GAsyncResult, data C.gpointer) {
	_, _ = src, data
	if active == nil || active.done {
		return
	}
	out := C.s99bench_eval_finish(active.view, res)
	if out == nil {
		return
	}
	defer C.g_free(C.gpointer(out))

	text := C.GoString(out)
	// Injection resolves to undefined and unfinished polls to "": not results.
	if text == "" || text == "undefined" || text == "null" || !strings.HasPrefix(text, "{") {
		return
	}
	var probe probeResult
	if err := json.Unmarshal([]byte(text), &probe); err != nil {
		return
	}
	active.done = true
	probe.into(&active.result)
	C.gtk_main_quit()
}

//export s99benchBail
func s99benchBail(data C.gpointer) C.gboolean {
	_ = data
	if active != nil && !active.done {
		active.result.Timeout = true
		active.result.Error = "timed out before the probe reported"
		C.gtk_main_quit()
	}
	return C.FALSE
}

// probeResult is the object probe.js reports; Run adds load time, CPU and env.
type probeResult struct {
	Nodes       int     `json:"nodes"`
	CSSRules    int     `json:"css_rules"`
	Stylesheets int     `json:"stylesheets"`
	DCL         float64 `json:"dom_content_loaded_ms"`
	Load        float64 `json:"load_event_ms"`
	FP          float64 `json:"first_paint_ms"`
	FCP         float64 `json:"first_contentful_paint_ms"`
	FPS         float64 `json:"fps"`
	P50         float64 `json:"frame_p50_ms"`
	P95         float64 `json:"frame_p95_ms"`
	Worst       float64 `json:"frame_worst_ms"`
	Frames      int     `json:"frames_counted"`
	StyleRecalc float64 `json:"style_recalc_ms"`
	Thrash      float64 `json:"layout_thrash_ms"`
	Script      float64 `json:"script_ms"`
	Reactivity  float64 `json:"reactivity_ms"`
	ProbeError  string  `json:"probe_error"`
}

func (p probeResult) into(r *Result) {
	r.Nodes, r.CSSRules, r.Stylesheets = p.Nodes, p.CSSRules, p.Stylesheets
	r.DOMContentMS, r.LoadEventMS, r.FirstPaintMS, r.FCPMS = p.DCL, p.Load, p.FP, p.FCP
	r.FPS, r.FrameP50MS, r.FrameP95MS, r.FrameWorstMS, r.FramesCounted = p.FPS, p.P50, p.P95, p.Worst, p.Frames
	r.StyleRecalcMS, r.LayoutThrashMS, r.ScriptMS, r.ReactivityMS = p.StyleRecalc, p.Thrash, p.Script, p.Reactivity
	if p.ProbeError != "" {
		r.Error = p.ProbeError
	}
}

// envKeys are the render-path variables reported with every result.
var envKeys = []string{
	"WEBKIT_DISABLE_DMABUF_RENDERER",
	"WEBKIT_DISABLE_COMPOSITING_MODE",
	"WEBKIT_SKIA_ENABLE_CPU_RENDERING",
	"GDK_BACKEND",
	"XDG_SESSION_TYPE",
	"LIBGL_ALWAYS_SOFTWARE",
}

func readEnv() map[string]string {
	env := map[string]string{}
	for _, k := range envKeys {
		if v, ok := os.LookupEnv(k); ok {
			env[k] = v
		}
	}
	return env
}
