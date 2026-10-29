// platform/env/env_test.go

package env

import (
	"os"
	"runtime"
	"testing"
)

// clearDisplayEnv unsets every variable Setup reads, with restore.
func clearDisplayEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"WEBKIT_DISABLE_DMABUF_RENDERER", "GDK_BACKEND",
		"XDG_SESSION_TYPE", "WAYLAND_DISPLAY", "XDG_CURRENT_DESKTOP",
		"APPIMAGE", "APPDIR", "THUG_GPU_TEST",
		// Remaining rung variables and the fallback override names.
		"WEBKIT_DMABUF_RENDERER_DISABLE_GBM", "WEBKIT_DMABUF_RENDERER_FORCE_SHM",
		"WEBKIT_SHOW_FPS", "WEBKIT_SHOW_DAMAGE",
		defaultRenderPathVar, defaultDebugVar,
	} {
		orig, had := os.LookupEnv(k)
		os.Unsetenv(k)
		t.Cleanup(func() {
			if had {
				os.Setenv(k, orig)
			} else {
				os.Unsetenv(k)
			}
		})
	}
}

// plasmaWayland makes PreferX11OnPlasmaWayland eligible to fire.
func plasmaWayland(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_SESSION_TYPE", "wayland")
	t.Setenv("XDG_CURRENT_DESKTOP", "KDE")
}

func TestSetupLegacyRenderPathAndPreferX11(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Setup is a no-op off Linux")
	}
	clearDisplayEnv(t)
	plasmaWayland(t)

	res := Setup(Options{PreferX11: true, RenderPath: RenderPathLegacy})

	if got := os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER"); got != "1" {
		t.Errorf("WEBKIT_DISABLE_DMABUF_RENDERER = %q, want 1", got)
	}
	if got := os.Getenv("GDK_BACKEND"); got != "x11" {
		t.Errorf("GDK_BACKEND = %q, want x11", got)
	}
	if !res.DMABUFDisabled || !res.X11Preferred {
		t.Errorf("result = %+v, want DMABUFDisabled and X11Preferred", res)
	}
}

func TestSetupRespectsExplicitDMABUFOverride(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Setup is a no-op off Linux")
	}
	clearDisplayEnv(t)
	plasmaWayland(t)
	// WebKit tests presence, so "0" still disables; Setup must keep the value.
	t.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "0")

	res := Setup(Options{PreferX11: true, RenderPath: RenderPathLegacy})

	if got := os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER"); got != "0" {
		t.Errorf("explicit override overwritten: got %q, want 0", got)
	}
	if !res.DMABUFDisabled || !res.X11Preferred {
		t.Errorf("result = %+v, want DMABUFDisabled (presence) and X11Preferred", res)
	}
}

func TestSetupZeroOptionsIsInert(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Setup is a no-op off Linux")
	}
	clearDisplayEnv(t)
	plasmaWayland(t)

	Setup(Options{})

	if _, set := os.LookupEnv("WEBKIT_DISABLE_DMABUF_RENDERER"); set {
		t.Error("zero Options must not touch WEBKIT_DISABLE_DMABUF_RENDERER")
	}
	if _, set := os.LookupEnv("GDK_BACKEND"); set {
		t.Error("zero Options must not touch GDK_BACKEND")
	}
}
