//go:build darwin

// platform/autostart/autostart_darwin.go

package autostart

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func plistPath(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", name+".plist"), nil
}

func enable(opts Options) error {
	path, err := plistPath(opts.Name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	var label strings.Builder
	if err := xml.EscapeText(&label, []byte(opts.Name)); err != nil {
		return err
	}

	var args strings.Builder
	for _, arg := range opts.Exec {
		var escaped strings.Builder
		if err := xml.EscapeText(&escaped, []byte(arg)); err != nil {
			return err
		}
		args.WriteString("\t\t<string>" + escaped.String() + "</string>\n")
	}

	content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
%s	</array>
	<key>RunAtLoad</key>
	<true/>
</dict>
</plist>
`, label.String(), args.String())

	return os.WriteFile(path, []byte(content), 0o644)
}

func disable(opts Options) error {
	path, err := plistPath(opts.Name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func enabled(opts Options) (bool, error) {
	path, err := plistPath(opts.Name)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}
