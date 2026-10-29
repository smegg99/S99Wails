// traygen/cli_test.go

package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smegg99/s99wails/traygen/v2/internal/cli"
)

func execute(args ...string) (int, string) {
	out := &bytes.Buffer{}
	a := cli.New(out, &bytes.Buffer{}, []string{"LANG=C", "NO_COLOR=1", "S99WAILS_LANG=en"})
	return a.Run(context.Background(), newRoot(a), args), out.String()
}

func TestCommandGeneratesAndSaysWhat(t *testing.T) {
	code, out := execute("--config", filepath.Join("testdata", "appicon.cue"), "--out", t.TempDir())
	if code != 0 || !strings.Contains(out, "Generated the tray icons") || !strings.Contains(out, "variants=") ||
		!strings.Contains(out, "Generated the app icon") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestCommandFailsOnABadConfig(t *testing.T) {
	if code, out := execute("--config", filepath.Join("testdata", "nope.cue"), "--out", t.TempDir()); code != 1 {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestCommandRefusesTheOldSingleDashFlags(t *testing.T) {
	if code, out := execute("-config", "trayicons.cue"); code != 2 {
		t.Fatalf("code %d, out %q", code, out)
	}
}
