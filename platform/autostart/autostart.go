// platform/autostart/autostart.go

// Package autostart registers per-user launch on login (XDG .desktop, Run key, LaunchAgent).
package autostart

import (
	"fmt"
	"os"
)

// Options identifies the autostart entry.
type Options struct {
	// Name is the machine-friendly identifier: the .desktop/.plist file name
	// and the registry value name. Required.
	Name string
	// DisplayName is the human-readable name where the platform shows one.
	// Defaults to Name.
	DisplayName string
	// Exec defaults to the current executable, or $APPIMAGE inside an AppImage
	// because the mount point changes per run.
	Exec []string
}

// Enable registers the autostart entry, replacing one with the same Name.
func Enable(opts Options) error {
	opts, err := opts.withDefaults()
	if err != nil {
		return err
	}
	return enable(opts)
}

// Disable removes the autostart entry; a missing one is not an error.
func Disable(opts Options) error {
	opts, err := opts.withDefaults()
	if err != nil {
		return err
	}
	return disable(opts)
}

// Enabled reports whether an autostart entry with this Name exists.
func Enabled(opts Options) (bool, error) {
	opts, err := opts.withDefaults()
	if err != nil {
		return false, err
	}
	return enabled(opts)
}

func (o Options) withDefaults() (Options, error) {
	if o.Name == "" {
		return o, fmt.Errorf("autostart: Name is required")
	}
	if o.DisplayName == "" {
		o.DisplayName = o.Name
	}
	if len(o.Exec) == 0 {
		exe := os.Getenv("APPIMAGE")
		if exe == "" {
			var err error
			exe, err = os.Executable()
			if err != nil {
				return o, fmt.Errorf("autostart: resolve executable: %w", err)
			}
		}
		o.Exec = []string{exe}
	}
	return o, nil
}
