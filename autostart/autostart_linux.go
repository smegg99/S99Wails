//go:build linux

// autostart/autostart_linux.go

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func desktopFilePath(name string) (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "autostart", name+".desktop"), nil
}

func enable(opts Options) error {
	path, err := desktopFilePath(opts.Name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	execParts := make([]string, len(opts.Exec))
	for i, arg := range opts.Exec {
		execParts[i] = quoteExecArg(arg)
	}

	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Exec=%s
Terminal=false
X-GNOME-Autostart-enabled=true
`, sanitizeEntryValue(opts.DisplayName), strings.Join(execParts, " "))

	return os.WriteFile(path, []byte(content), 0o644)
}

// quoteExecArg double-quotes an Exec argument and backslash-escapes ", `, $ and \
// per the Desktop Entry spec.
func quoteExecArg(arg string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range arg {
		switch r {
		case '"', '`', '$', '\\':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	b.WriteByte('"')
	return b.String()
}

// sanitizeEntryValue strips control characters (notably newlines, which
// would otherwise inject additional desktop-entry keys) from a value.
func sanitizeEntryValue(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
}

func disable(opts Options) error {
	path, err := desktopFilePath(opts.Name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func enabled(opts Options) (bool, error) {
	path, err := desktopFilePath(opts.Name)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}
