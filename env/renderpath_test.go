// env/renderpath_test.go

package env

import (
	"os"
	"runtime"
	"testing"
)

// clearRenderEnv unsets renderer variables with restore; t.Setenv cannot unset.
func clearRenderEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"WEBKIT_DISABLE_DMABUF_RENDERER",
		"WEBKIT_DMABUF_RENDERER_DISABLE_GBM",
		"WEBKIT_DMABUF_RENDERER_FORCE_SHM",
		"WEBKIT_SHOW_FPS", "WEBKIT_SHOW_DAMAGE",
		"GDK_BACKEND", "XDG_SESSION_TYPE", "WAYLAND_DISPLAY", "XDG_CURRENT_DESKTOP",
		"APPIMAGE", "APPDIR", "S99_RENDER_TEST", "S99_DEBUG_TEST",
		// Fallback names a developer may have exported locally.
		DefaultRenderPathVar, DefaultDebugVar,
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

func requireLinux(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("Setup is a no-op off Linux")
	}
}

func TestSetupDefaultTouchesNothing(t *testing.T) {
	requireLinux(t)
	clearRenderEnv(t)

	res := Setup(Options{})

	if res.RenderPath != RenderPathAuto {
		t.Errorf("RenderPath = %v, want auto", res.RenderPath)
	}
	for _, k := range []string{
		"WEBKIT_DISABLE_DMABUF_RENDERER",
		"WEBKIT_DMABUF_RENDERER_DISABLE_GBM",
		"WEBKIT_DMABUF_RENDERER_FORCE_SHM",
	} {
		if _, ok := os.LookupEnv(k); ok {
			t.Errorf("%s was set by a zero Options", k)
		}
	}
}

func TestRenderPathRungsSetTheirOwnVariable(t *testing.T) {
	requireLinux(t)
	cases := []struct {
		path RenderPath
		key  string
	}{
		{RenderPathNoGBM, "WEBKIT_DMABUF_RENDERER_DISABLE_GBM"},
		{RenderPathSHM, "WEBKIT_DMABUF_RENDERER_FORCE_SHM"},
		{RenderPathLegacy, "WEBKIT_DISABLE_DMABUF_RENDERER"},
	}
	for _, tc := range cases {
		t.Run(tc.path.String(), func(t *testing.T) {
			clearRenderEnv(t)

			res := Setup(Options{RenderPath: tc.path})

			if res.RenderPath != tc.path {
				t.Errorf("RenderPath = %v, want %v", res.RenderPath, tc.path)
			}
			if os.Getenv(tc.key) != "1" {
				t.Errorf("%s = %q, want 1", tc.key, os.Getenv(tc.key))
			}
			// Exactly one rung, or the ladder is not a ladder.
			for _, other := range cases {
				if other.key == tc.key {
					continue
				}
				if _, ok := os.LookupEnv(other.key); ok {
					t.Errorf("%s was also set", other.key)
				}
			}
		})
	}
}

func TestExistingEnvironmentWins(t *testing.T) {
	requireLinux(t)
	clearRenderEnv(t)
	t.Setenv("WEBKIT_DMABUF_RENDERER_FORCE_SHM", "1")

	res := Setup(Options{RenderPath: RenderPathLegacy})

	if res.RenderPath != RenderPathSHM || res.RenderPathFrom != "env" {
		t.Errorf("got %v from %q, want shm from env", res.RenderPath, res.RenderPathFrom)
	}
	if _, ok := os.LookupEnv("WEBKIT_DISABLE_DMABUF_RENDERER"); ok {
		t.Error("the app's rung was applied over the user's")
	}
}

func TestRenderPathVarOverridesTheApp(t *testing.T) {
	requireLinux(t)
	clearRenderEnv(t)
	t.Setenv("S99_RENDER_TEST", "nogbm")

	res := Setup(Options{RenderPath: RenderPathLegacy, RenderPathVar: "S99_RENDER_TEST"})

	if res.RenderPath != RenderPathNoGBM || res.RenderPathFrom != "var" {
		t.Errorf("got %v from %q, want nogbm from var", res.RenderPath, res.RenderPathFrom)
	}
}

// A typo must not silently select a rung, least of all the slow one.
func TestUnknownRenderPathNameIsIgnored(t *testing.T) {
	requireLinux(t)
	clearRenderEnv(t)
	t.Setenv("S99_RENDER_TEST", "dmabuff")

	res := Setup(Options{RenderPath: RenderPathNoGBM, RenderPathVar: "S99_RENDER_TEST"})

	if res.RenderPath != RenderPathNoGBM || res.RenderPathFrom != "default" {
		t.Errorf("got %v from %q, want the app's nogbm", res.RenderPath, res.RenderPathFrom)
	}
}

func TestAppImageUsesItsOwnRung(t *testing.T) {
	requireLinux(t)
	clearRenderEnv(t)
	t.Setenv("APPIMAGE", "/tmp/app.AppImage")

	res := Setup(Options{RenderPath: RenderPathAuto, AppImageRenderPath: RenderPathNoGBM})

	if !res.AppImage {
		t.Fatal("AppImage not detected")
	}
	if res.RenderPath != RenderPathNoGBM || res.RenderPathFrom != "appimage" {
		t.Errorf("got %v from %q, want nogbm from appimage", res.RenderPath, res.RenderPathFrom)
	}
}

// The deprecated boolean has three other apps still on it; it must keep meaning
// what it meant, including inside an AppImage where it was least negotiable.
func TestDeprecatedDisableDMABUFStillMapsToLegacy(t *testing.T) {
	requireLinux(t)

	t.Run("plain", func(t *testing.T) {
		clearRenderEnv(t)
		res := Setup(Options{DisableDMABUF: true})
		if res.RenderPath != RenderPathLegacy || !res.DMABUFDisabled {
			t.Errorf("got %v, DMABUFDisabled=%v", res.RenderPath, res.DMABUFDisabled)
		}
	})

	t.Run("appimage", func(t *testing.T) {
		clearRenderEnv(t)
		t.Setenv("APPIMAGE", "/tmp/app.AppImage")
		res := Setup(Options{DisableDMABUF: true})
		if res.RenderPath != RenderPathLegacy {
			t.Errorf("got %v, want legacy", res.RenderPath)
		}
	})
}

func TestDebugVarTurnsOnTheOverlays(t *testing.T) {
	requireLinux(t)
	clearRenderEnv(t)
	t.Setenv("S99_DEBUG_TEST", "1")

	Setup(Options{DebugVar: "S99_DEBUG_TEST"})

	if os.Getenv("WEBKIT_SHOW_FPS") != "1" || os.Getenv("WEBKIT_SHOW_DAMAGE") != "1" {
		t.Error("debug overlays not enabled")
	}
}

func TestParseRenderPathRoundTrips(t *testing.T) {
	for _, p := range []RenderPath{RenderPathAuto, RenderPathNoGBM, RenderPathSHM, RenderPathLegacy} {
		got, ok := ParseRenderPath(p.String())
		if !ok || got != p {
			t.Errorf("%v round-tripped to %v (ok=%v)", p, got, ok)
		}
	}
	if _, ok := ParseRenderPath("nonsense"); ok {
		t.Error("an unknown name reported itself as known")
	}
}
