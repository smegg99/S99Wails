//go:build darwin

// accent/darkmode_darwin.go

package accent

import (
	"os/exec"
	"strings"
)

func detectScheme() (Scheme, bool) {
	// AppleInterfaceStyle is only set while dark mode is active; the read
	// fails with a non-zero exit in light mode.
	out, err := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle").Output()
	if err != nil {
		return SchemeLight, true
	}
	if strings.EqualFold(strings.TrimSpace(string(out)), "dark") {
		return SchemeDark, true
	}
	return SchemeLight, true
}
