// env.go
//
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
