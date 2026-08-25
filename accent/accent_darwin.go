//go:build darwin

// accent/accent_darwin.go

package accent

import (
	"image/color"
	"os/exec"
	"strconv"
	"strings"
)

// getAccentColor prefers AppleHighlightColor (RGB floats) and falls back to the
// AppleAccentColor index.
func getAccentColor() Color {
	if c, ok := highlightColor(); ok {
		return c
	}
	if c, ok := accentIndexColor(); ok {
		return c
	}
	return Color{OK: false, Source: AppleDefaultsSource}
}

func highlightColor() (Color, bool) {
	out, err := exec.Command("defaults", "read", "-g", "AppleHighlightColor").Output()
	if err != nil {
		return Color{}, false
	}
	// Format: "0.968627 0.831373 1.000000 Purple"
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 3 {
		return Color{}, false
	}
	rgb := [3]uint8{}
	for i := 0; i < 3; i++ {
		f, err := strconv.ParseFloat(fields[i], 64)
		if err != nil || f < 0 || f > 1 {
			return Color{}, false
		}
		rgb[i] = uint8(f*255 + 0.5)
	}
	return Color{
		RGBA:   color.RGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: 0xFF},
		OK:     true,
		Source: AppleDefaultsSource,
	}, true
}

func accentIndexColor() (Color, bool) {
	out, err := exec.Command("defaults", "read", "-g", "AppleAccentColor").Output()
	if err != nil {
		// Unset means the default blue accent.
		return appleSystemColor(4), true
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return Color{}, false
	}
	return appleSystemColor(n), true
}

// appleSystemColor maps the AppleAccentColor index to Apple's system accent colors.
func appleSystemColor(index int) Color {
	var c color.RGBA
	switch index {
	case -1:
		c = color.RGBA{0x8E, 0x8E, 0x93, 0xFF} // graphite
	case 0:
		c = color.RGBA{0xFF, 0x3B, 0x30, 0xFF} // red
	case 1:
		c = color.RGBA{0xFF, 0x95, 0x00, 0xFF} // orange
	case 2:
		c = color.RGBA{0xFF, 0xCC, 0x00, 0xFF} // yellow
	case 3:
		c = color.RGBA{0x34, 0xC7, 0x59, 0xFF} // green
	case 5:
		c = color.RGBA{0xAF, 0x52, 0xDE, 0xFF} // purple
	case 6:
		c = color.RGBA{0xFF, 0x2D, 0x55, 0xFF} // pink
	default:
		c = color.RGBA{0x00, 0x7A, 0xFF, 0xFF} // blue
	}
	return Color{RGBA: c, OK: true, Source: AppleDefaultsSource}
}
