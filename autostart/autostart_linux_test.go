// autostart_linux_test.go
//go:build linux

package autostart

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testOptions(t *testing.T) Options {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	return Options{
		Name:        "s99test",
		DisplayName: "S99 Test App",
		Exec:        []string{"/opt/my app/bin", "--home=$HOME", `--q="x"`},
	}
}

func TestEnableWritesEscapedDesktopFile(t *testing.T) {
	opts := testOptions(t)
	if err := Enable(opts); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	path, err := desktopFilePath(opts.Name)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("desktop file not written: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "Name=S99 Test App\n") {
		t.Errorf("Name line wrong:\n%s", content)
	}
	// Every arg quoted; $, " escaped per the desktop-entry spec.
	if !strings.Contains(content, `Exec="/opt/my app/bin" "--home=\$HOME" "--q=\"x\""`) {
		t.Errorf("Exec escaping wrong:\n%s", content)
	}
}

func TestDisplayNameNewlineCannotInjectKeys(t *testing.T) {
	opts := testOptions(t)
	opts.DisplayName = "Evil\nExec=rm -rf /"
	if err := Enable(opts); err != nil {
		t.Fatal(err)
	}
	path, _ := desktopFilePath(opts.Name)
	data, _ := os.ReadFile(path)
	// The payload must not become its own key line; it may survive as inert
	// text inside the (newline-stripped) Name value.
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "Exec=") && !strings.HasPrefix(line, `Exec="`) {
			t.Errorf("newline in DisplayName injected a key line %q:\n%s", line, data)
		}
	}
}

func TestEnableDisableEnabledRoundTrip(t *testing.T) {
	opts := testOptions(t)

	on, err := Enabled(opts)
	if err != nil || on {
		t.Fatalf("fresh env should be disabled, got on=%v err=%v", on, err)
	}
	if err := Enable(opts); err != nil {
		t.Fatal(err)
	}
	if on, _ := Enabled(opts); !on {
		t.Error("Enabled should report true after Enable")
	}
	if err := Disable(opts); err != nil {
		t.Fatal(err)
	}
	if on, _ := Enabled(opts); on {
		t.Error("Enabled should report false after Disable")
	}
	if err := Disable(opts); err != nil {
		t.Error("Disable of a missing entry must not error:", err)
	}
}

func TestOptionsRequireName(t *testing.T) {
	if err := Enable(Options{}); err == nil {
		t.Error("Enable without Name must error")
	}
}

func TestExecDefaultsToExecutable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPIMAGE", "")
	opts, err := Options{Name: "s99default"}.withDefaults()
	if err != nil {
		t.Fatal(err)
	}
	exe, _ := os.Executable()
	if len(opts.Exec) != 1 || opts.Exec[0] != exe {
		t.Errorf("Exec default = %v, want [%s]", opts.Exec, exe)
	}
	if opts.DisplayName != "s99default" {
		t.Errorf("DisplayName default = %q", opts.DisplayName)
	}
	_ = filepath.Dir(exe)
}
