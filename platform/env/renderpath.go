// platform/env/renderpath.go

// WebKitGTK render path rungs. Compare them on a given machine with:
//
//	go run github.com/smegg99/s99wails/v2/cmd/s99webbench --dir frontend/.output/public --matrix

package env

import "os"

// RenderPath selects how composited frames reach the UI process. The zero value
// is Auto.
type RenderPath int

const (
	// RenderPathAuto sets no variables: full DMABUF, zero-copy.
	RenderPathAuto RenderPath = iota

	// RenderPathNoGBM keeps DMABUF but skips GBM allocation. Try it first for
	// garbled, black or stale frames.
	RenderPathNoGBM

	// RenderPathSHM keeps DMABUF compositing with shared-memory buffers (one copy per frame).
	RenderPathSHM

	// RenderPathLegacy disables the DMABUF renderer. Measured at about 5x lower
	// frame rate than Auto (Mesa 26, RDNA4); try NoGBM and SHM first.
	RenderPathLegacy
)

var renderPathVars = map[RenderPath][]string{
	RenderPathNoGBM:  {"WEBKIT_DMABUF_RENDERER_DISABLE_GBM"},
	RenderPathSHM:    {"WEBKIT_DMABUF_RENDERER_FORCE_SHM"},
	RenderPathLegacy: {"WEBKIT_DISABLE_DMABUF_RENDERER"},
}

var renderPathNames = map[RenderPath]string{
	RenderPathAuto:   "auto",
	RenderPathNoGBM:  "nogbm",
	RenderPathSHM:    "shm",
	RenderPathLegacy: "legacy",
}

// String is the name accepted by the override variable.
func (p RenderPath) String() string {
	if name, ok := renderPathNames[p]; ok {
		return name
	}
	return "auto"
}

// ParseRenderPath reads a rung by name. Unknown names return Auto and false.
func ParseRenderPath(s string) (RenderPath, bool) {
	for path, name := range renderPathNames {
		if s == name {
			return path, true
		}
	}
	return RenderPathAuto, false
}

// applyRenderPath sets the rung's variables without overriding user-set ones.
func applyRenderPath(p RenderPath) {
	for _, key := range renderPathVars[p] {
		setDefault(key, "1")
	}
}

// renderPathFromEnv reports the rung already set in the environment. Legacy is
// checked first: WebKit tests it for presence, so any value disables the renderer.
func renderPathFromEnv() (RenderPath, bool) {
	for _, p := range []RenderPath{RenderPathLegacy, RenderPathSHM, RenderPathNoGBM} {
		for _, key := range renderPathVars[p] {
			if _, ok := os.LookupEnv(key); ok {
				return p, true
			}
		}
	}
	return RenderPathAuto, false
}

// DebugRendering enables WebKitGTK's frame counter and damage overlays. Call
// before the GUI initializes.
func DebugRendering() {
	setDefault("WEBKIT_SHOW_FPS", "1")
	setDefault("WEBKIT_SHOW_DAMAGE", "1")
}
