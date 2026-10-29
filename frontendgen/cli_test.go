// frontendgen/cli_test.go

package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/smegg99/s99wails/v2/internal/cli"
)

func execute(args ...string) (int, string) {
	out := &bytes.Buffer{}
	a := cli.New(out, &bytes.Buffer{}, []string{"LANG=C", "NO_COLOR=1", "S99WAILS_LANG=en"})
	return a.Run(context.Background(), newRoot(a), args), out.String()
}

func TestCommandEmitsIntoOutAndSaysHowMany(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "gen")
	code, out := execute("--out", dir, "--s99ab")
	if code != 0 || !strings.Contains(out, "Emitted the composables") || !strings.Contains(out, "files=13") {
		t.Fatalf("code %d, out %q", code, out)
	}
	if _, err := os.Stat(filepath.Join(dir, "tabsBridge.ts")); err != nil {
		t.Fatal(err)
	}
}

func TestCommandRefusesAStrayArgument(t *testing.T) {
	if code, out := execute("--out", t.TempDir(), "extra"); code != 2 {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestCommandFailsWhenItCannotWrite(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out := execute("--out", file); code != 1 {
		t.Fatalf("code %d, out %q", code, out)
	}
}
