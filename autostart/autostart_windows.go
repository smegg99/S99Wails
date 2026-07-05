// autostart_windows.go
//go:build windows

package autostart

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

// command joins the Exec args for the Run value. Arguments containing a double
// quote are rejected because they cannot be escaped there.
func command(opts Options) (string, error) {
	parts := make([]string, len(opts.Exec))
	for i, arg := range opts.Exec {
		if strings.Contains(arg, `"`) {
			return "", fmt.Errorf("autostart: exec argument %q contains a double quote, which cannot be represented in a Run registry command", arg)
		}
		if strings.ContainsAny(arg, " \t") {
			arg = `"` + arg + `"`
		}
		parts[i] = arg
	}
	return strings.Join(parts, " "), nil
}

func enable(opts Options) error {
	cmd, err := command(opts)
	if err != nil {
		return err
	}
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(opts.Name, cmd)
}

func disable(opts Options) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if err := k.DeleteValue(opts.Name); err != nil && err != registry.ErrNotExist {
		return err
	}
	return nil
}

func enabled(opts Options) (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer k.Close()
	_, _, err = k.GetStringValue(opts.Name)
	if err == registry.ErrNotExist {
		return false, nil
	}
	return err == nil, err
}
