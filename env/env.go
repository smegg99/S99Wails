// env/env.go

// Package env sets process-environment workarounds before the Wails GUI initializes.
package env

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ApplyLinuxWebkitWorkarounds disables the WebKitGTK DMABUF renderer unless the
// user set it; inside an AppImage it is forced and appImage is true.
func ApplyLinuxWebkitWorkarounds() (appImage bool) {
	if runtime.GOOS == "linux" {
		setDefault("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	}

	if LaunchedViaAppImage() {
		os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
		return true
	}
	return false
}

// LaunchedViaAppImage reports whether the process is running inside an AppImage.
func LaunchedViaAppImage() bool {
	if os.Getenv("APPIMAGE") != "" || os.Getenv("APPDIR") != "" {
		return true
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	return strings.Contains(filepath.ToSlash(filepath.Clean(exe)), "/tmp/.mount_")
}

// setDefault sets an environment variable only when it is not already set.
func setDefault(key, value string) {
	if _, ok := os.LookupEnv(key); !ok {
		os.Setenv(key, value)
	}
}

// Override variables Setup reads when Options.RenderPathVar or DebugVar is empty.
const (
	DefaultRenderPathVar = "S99_RENDER"
	DefaultDebugVar      = "S99_RENDER_DEBUG"
)

// Options configures Setup. Each field is an independent opt-in.
type Options struct {
	// PreferX11 calls PreferX11OnPlasmaWayland.
	PreferX11 bool

	RenderPath RenderPath

	// AppImageRenderPath applies inside an AppImage, where the host GPU stack is
	// unknown. RenderPathNoGBM is the suggested value.
	AppImageRenderPath RenderPath

	// RenderPathVar names the variable holding a rung name (auto, nogbm, shm,
	// legacy). Empty uses DefaultRenderPathVar.
	RenderPathVar string

	// DebugVar set to "1" calls DebugRendering. Empty uses DefaultDebugVar.
	DebugVar string

	// DisableDMABUF forces the legacy fallback.
	//
	// Deprecated: it is RenderPathLegacy under an older name, and it reads as
	// though the choice were binary when there are two cheaper rungs above it.
	// Set RenderPath instead.
	DisableDMABUF bool

	// GPUOptOutVar names an environment variable that, set to "1", keeps the
	// DMABUF renderer AND native Wayland.
	//
	// Deprecated: it is an opt-out from a default that should not have been the
	// default. Use RenderPathVar, which names every rung rather than only the
	// two ends of the ladder.
	GPUOptOutVar string
}

// Result reports what Setup did. Off Linux only AppImage is set.
type Result struct {
	AppImage       bool       // launched from an AppImage
	RenderPath     RenderPath // the rung that ended up selected
	RenderPathFrom string     // what selected it: default, appimage, env, var, legacy-option
	DMABUFDisabled bool       // WEBKIT_DISABLE_DMABUF_RENDERER ended up set
	X11Preferred   bool       // GDK_BACKEND was defaulted to x11
	GPUOptOut      bool       // the GPU opt-out variable was honored
}

// Setup applies the Linux desktop workarounds in one call, before the GUI
// initializes. It resolves the render path and the display backend together
// because they interact, in this precedence:
//
//  1. The user's own environment. Any of the renderer variables already set is
//     a decision that has been made; respect it and report which rung it names.
//  2. RenderPathVar, if the app named one and its value parses. This is the
//     escape hatch a user reaches for, so it beats everything the app chose.
//  3. GPU opt-out (deprecated GPUOptOutVar == "1"): full DMABUF and native
//     Wayland, no X11 default either, since the point is the GPU path.
//  4. AppImage: AppImageRenderPath, because the host GPU stack is unknown.
//  5. Otherwise: RenderPath, or RenderPathLegacy if the deprecated
//     DisableDMABUF is set.
//
// Then PreferX11. An explicit GDK_BACKEND always wins over it.
//
// A zero Options is inert in the sense that matters: it selects RenderPathAuto,
// which sets no variables at all. The old shape of this function defaulted to
// the slowest rung and asked apps to opt out; that was backwards, and every app
// that took the default paid about five times its frame rate for a workaround
// it had never measured a need for.
func Setup(o Options) Result {
	res := Result{AppImage: LaunchedViaAppImage()}
	if runtime.GOOS != "linux" {
		return res
	}

	debugVar := o.DebugVar
	if debugVar == "" {
		debugVar = DefaultDebugVar
	}
	if os.Getenv(debugVar) == "1" {
		DebugRendering()
	}

	path, from, skipX11 := o.resolveRenderPath()
	applyRenderPath(path)
	res.RenderPath, res.RenderPathFrom = path, from
	res.DMABUFDisabled = explicitlySet("WEBKIT_DISABLE_DMABUF_RENDERER")
	res.GPUOptOut = from == "gpu-optout"

	if o.PreferX11 && !skipX11 {
		res.X11Preferred = PreferX11OnPlasmaWayland()
	}
	return res
}

// resolveRenderPath is Setup's precedence, separated so it can be read and
// tested without touching the process environment's other halves. skipX11
// reports the one case that also decides the display backend: the deprecated
// GPU opt-out meant "the accelerated path, all of it", and XWayland is not that.
func (o Options) resolveRenderPath() (path RenderPath, from string, skipX11 bool) {
	if p, ok := renderPathFromEnv(); ok {
		return p, "env", false
	}
	varName := o.RenderPathVar
	if varName == "" {
		varName = DefaultRenderPathVar
	}
	if p, ok := ParseRenderPath(os.Getenv(varName)); ok {
		return p, "var", false
	}
	if o.GPUOptOutVar != "" && os.Getenv(o.GPUOptOutVar) == "1" {
		return RenderPathAuto, "gpu-optout", true
	}
	if res := LaunchedViaAppImage(); res {
		// A caller still on the deprecated boolean asked for the legacy rung
		// everywhere, and an AppImage was where it was least negotiable.
		if o.AppImageRenderPath == RenderPathAuto && o.DisableDMABUF {
			return RenderPathLegacy, "appimage", false
		}
		return o.AppImageRenderPath, "appimage", false
	}
	if o.DisableDMABUF {
		return RenderPathLegacy, "legacy-option", false
	}
	return o.RenderPath, "default", false
}

func explicitlySet(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok
}

// PreferX11OnPlasmaWayland defaults GDK_BACKEND=x11 on Plasma Wayland, where GTK
// lacks xdg-decoration and draws unthemed decorations. Reports whether it applied.
func PreferX11OnPlasmaWayland() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	wayland := os.Getenv("XDG_SESSION_TYPE") == "wayland" || os.Getenv("WAYLAND_DISPLAY") != ""
	plasma := strings.Contains(strings.ToUpper(os.Getenv("XDG_CURRENT_DESKTOP")), "KDE")
	if !wayland || !plasma {
		return false
	}
	if _, ok := os.LookupEnv("GDK_BACKEND"); ok {
		return false
	}
	os.Setenv("GDK_BACKEND", "x11")
	return true
}
