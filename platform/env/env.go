// platform/env/env.go

// Package env sets process-environment workarounds before the Wails GUI initializes.
package env

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// launchedViaAppImage reports whether the process is running inside an AppImage.
func launchedViaAppImage() bool {
	if os.Getenv("APPIMAGE") != "" || os.Getenv("APPDIR") != "" {
		return true
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	return strings.Contains(filepath.ToSlash(filepath.Clean(exe)), "/tmp/.mount_")
}

func setDefault(key, value string) {
	if _, ok := os.LookupEnv(key); !ok {
		os.Setenv(key, value)
	}
}

// Override variables Setup reads when Options.RenderPathVar or DebugVar is empty.
const (
	defaultRenderPathVar = "S99_RENDER"
	defaultDebugVar      = "S99_RENDER_DEBUG"
)

// Options configures Setup. Each field is an independent opt-in.
type Options struct {
	// PreferX11 calls PreferX11OnPlasmaWayland, for GPUs that flicker under Wayland+DMABUF.
	PreferX11 bool

	// RenderPath picks how composited frames reach the UI process. The zero
	// value is RenderPathAuto, WebKit's own zero-copy default.
	RenderPath RenderPath

	// AppImageRenderPath applies inside an AppImage, where the host GPU stack is
	// unknown. RenderPathNoGBM is the suggested value.
	AppImageRenderPath RenderPath

	// RenderPathVar names the variable holding a rung name (auto, nogbm, shm,
	// legacy). Empty uses DefaultRenderPathVar.
	RenderPathVar string

	// DebugVar set to "1" calls DebugRendering. Empty uses DefaultDebugVar.
	DebugVar string
}

// Result reports what Setup did. Off Linux only AppImage is set.
type Result struct {
	AppImage       bool       // launched from an AppImage
	RenderPath     RenderPath // the rung that ended up selected
	RenderPathFrom string     // what selected it: default, appimage, env, var
	DMABUFDisabled bool       // WEBKIT_DISABLE_DMABUF_RENDERER ended up set
	X11Preferred   bool       // GDK_BACKEND was defaulted to x11
}

// Setup resolves the render path (environment, RenderPathVar, AppImageRenderPath in an AppImage, RenderPath) and applies PreferX11.
func Setup(o Options) Result {
	res := Result{AppImage: launchedViaAppImage()}
	if runtime.GOOS != "linux" {
		return res
	}

	debugVar := o.DebugVar
	if debugVar == "" {
		debugVar = defaultDebugVar
	}
	if os.Getenv(debugVar) == "1" {
		debugRendering()
	}

	path, from := o.resolveRenderPath()
	applyRenderPath(path)
	res.RenderPath, res.RenderPathFrom = path, from
	res.DMABUFDisabled = explicitlySet("WEBKIT_DISABLE_DMABUF_RENDERER")

	if o.PreferX11 {
		res.X11Preferred = preferX11OnPlasmaWayland()
	}
	return res
}

func (o Options) resolveRenderPath() (path RenderPath, from string) {
	if p, ok := renderPathFromEnv(); ok {
		return p, "env"
	}
	varName := o.RenderPathVar
	if varName == "" {
		varName = defaultRenderPathVar
	}
	if p, ok := parseRenderPath(os.Getenv(varName)); ok {
		return p, "var"
	}
	if launchedViaAppImage() {
		return o.AppImageRenderPath, "appimage"
	}
	return o.RenderPath, "default"
}

func explicitlySet(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok
}

// preferX11OnPlasmaWayland sets an unset GDK_BACKEND to x11 on Plasma Wayland, where GTK 3 under XWayland paints at 60 Hz and one scale.
func preferX11OnPlasmaWayland() bool {
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
