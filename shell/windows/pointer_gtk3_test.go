//go:build linux && cgo && gtk3

// shell/windows/pointer_gtk3_test.go
package windows

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPointerFindsWindowThroughForeignChild(t *testing.T) {
	for _, name := range []string{"cc", "pkg-config", "xvfb-run", "dbus-run-session"} {
		if _, err := exec.LookPath(name); err != nil {
			t.Skipf("native pointer test needs %s", name)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	flags, err := exec.CommandContext(ctx, "pkg-config", "--cflags", "--libs", "gtk+-3.0", "x11").Output()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	program := filepath.Join(dir, "pointer")
	args := append([]string{"testdata/pointer.c", "-o", program}, strings.Fields(string(flags))...)
	if out, err := exec.CommandContext(ctx, "cc", args...).CombinedOutput(); err != nil {
		t.Fatalf("compile native test: %v\n%s", err, out)
	}
	cmd := exec.CommandContext(ctx, "xvfb-run", "-a", "dbus-run-session", "--", program)
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"), "LANG=C", "GDK_BACKEND=x11", "NO_AT_BRIDGE=1",
		"XDG_RUNTIME_DIR=" + dir, "XDG_CONFIG_HOME=" + dir, "XDG_DATA_HOME=" + dir, "XDG_CACHE_HOME=" + dir,
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native pointer test: %v\n%s", err, out)
	}
}
