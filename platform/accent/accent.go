// platform/accent/accent.go

package accent

import (
	"fmt"
	"image/color"
)

// ColorSource identifies the platform mechanism used to retrieve the accent color.
type ColorSource string

const (
	XDGPortalSource       ColorSource = "xdg-portal"
	WindowsRegistrySource ColorSource = "windows-registry"
	AppleDefaultsSource   ColorSource = "apple-defaults"
)

// Scheme is the OS color-scheme preference. Unlike a plain dark-mode bool it
// preserves "no preference", which is a real state on Linux.
type Scheme string

const (
	SchemeNoPreference Scheme = "no-preference"
	SchemeLight        Scheme = "light"
	SchemeDark         Scheme = "dark"
)

// Color holds a resolved platform accent color.
type Color struct {
	RGBA   color.RGBA
	OK     bool
	Source ColorSource
}

// Hex returns the color as a #rrggbb string, or "" if not available.
func (c Color) Hex() string {
	if !c.OK {
		return ""
	}
	return fmt.Sprintf("#%02x%02x%02x", c.RGBA.R, c.RGBA.G, c.RGBA.B)
}

// GetAccentColor returns the platform accent color when available.
func GetAccentColor() Color {
	return getAccentColor()
}

// DetectScheme returns the OS color-scheme preference. The second return
// value is false when the check is inconclusive (unsupported platform).
func DetectScheme() (Scheme, bool) {
	return detectScheme()
}

// DetectDarkMode reports dark mode from native APIs; ok is false when
// unsupported or the user has no preference.
func DetectDarkMode() (dark bool, ok bool) {
	scheme, ok := detectScheme()
	if !ok || scheme == SchemeNoPreference {
		return false, false
	}
	return scheme == SchemeDark, true
}

// WatchAccentChanges calls onChange on accent or color-scheme changes until stop.
func WatchAccentChanges(onChange func()) (stop func()) {
	return watchAccentChanges(onChange)
}
