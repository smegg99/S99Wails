// env/renderpath.go

// How WebKitGTK gets a composited frame from the web process to the UI process,
// as a ladder rather than a switch.
//
// The advice in circulation is binary: the DMABUF renderer misbehaves on some
// stack, so WEBKIT_DISABLE_DMABUF_RENDERER=1 and move on. That is a real fix for
// a real class of bug and it costs about five times the frame rate - measured
// with s99wails/webbench on Mesa 26 + RDNA4, 52.8fps against 9.9, the web
// process at 37% CPU against 98%. Jumping straight to the bottom rung to dodge
// a driver bug is the single most expensive thing a Linux Wails app does to
// itself.
//
// WebKitGTK 2.52 recognises finer knobs than that one, and they sit between the
// two extremes:
//
//	WEBKIT_DMABUF_RENDERER_DISABLE_GBM  keep the renderer, stop allocating
//	                                    through GBM. Targets modifier and
//	                                    format negotiation - the usual cause of
//	                                    garbled, black or stale frames.
//	WEBKIT_DMABUF_RENDERER_FORCE_SHM    keep the renderer, move the buffers to
//	                                    shared memory. Loses zero-copy, keeps
//	                                    the modern compositing path.
//	WEBKIT_DISABLE_DMABUF_RENDERER      the legacy fallback. Everything through
//	                                    the CPU.
//
// So a stack that cannot do full DMABUF has two rungs to try before the one
// that costs 5x. Which of them is fastest and which of them behaves is a
// question about a particular machine, and the answer is measured, not guessed:
//
//	go run github.com/smegg99/s99wails/cmd/s99webbench -dir frontend/.output/public -matrix
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

// String is the name a user types into the override variable.
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
