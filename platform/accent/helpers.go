// platform/accent/helpers.go

package accent

import (
	"fmt"
	"strings"
)

// Info holds a point-in-time snapshot of device-level theme settings.
type Info struct {
	AccentColor          string `json:"accent_color"`
	DarkMode             bool   `json:"dark_mode"`
	Scheme               Scheme `json:"scheme"`
	AccentWatchSupported bool   `json:"accent_watch_supported"`
}

// Snapshot returns the device theme from native OS APIs only, with no GUI framework.
func Snapshot() Info {
	info := Info{Scheme: SchemeNoPreference}

	if scheme, ok := DetectScheme(); ok {
		info.Scheme = scheme
		info.DarkMode = scheme == SchemeDark
	}

	if hex := GetAccentColor().Hex(); hex != "" {
		info.AccentColor = hex
	}

	info.AccentWatchSupported = info.AccentColor != ""
	return info
}

// NormalizeToHex turns rgb() or rgba() into #RRGGBB and returns hex unchanged.
func NormalizeToHex(color string) string {
	color = strings.TrimSpace(color)
	if strings.HasPrefix(color, "#") {
		return color
	}
	var r, g, b int

	if n, _ := fmt.Sscanf(color, "rgb(%d,%d,%d", &r, &g, &b); n == 3 {
		return fmt.Sprintf("#%02x%02x%02x", clamp(r), clamp(g), clamp(b))
	}
	if n, _ := fmt.Sscanf(color, "rgba(%d,%d,%d", &r, &g, &b); n == 3 {
		return fmt.Sprintf("#%02x%02x%02x", clamp(r), clamp(g), clamp(b))
	}
	if n, _ := fmt.Sscanf(color, "rgb(%d, %d, %d", &r, &g, &b); n == 3 {
		return fmt.Sprintf("#%02x%02x%02x", clamp(r), clamp(g), clamp(b))
	}
	if n, _ := fmt.Sscanf(color, "rgba(%d, %d, %d", &r, &g, &b); n == 3 {
		return fmt.Sprintf("#%02x%02x%02x", clamp(r), clamp(g), clamp(b))
	}
	return ""
}

func clamp(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}
