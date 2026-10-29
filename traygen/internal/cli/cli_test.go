// traygen/internal/cli/cli_test.go

package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// run builds a one-command tool whose work returns work and returns the exit code and both streams.
func run(t *testing.T, env []string, work error, args ...string) (int, string, string) {
	t.Helper()
	out, data := &bytes.Buffer{}, &bytes.Buffer{}
	a := New(out, data, append([]string{"LANG=C", "NO_COLOR=1"}, env...))
	var flag string
	root := a.Root(&cobra.Command{
		Use:  "tool",
		Args: a.NoArgs,
		RunE: func(*cobra.Command, []string) error { return work },
	})
	root.Flags().StringVar(&flag, "out", "", "")
	code := a.Run(context.Background(), root, args)
	return code, out.String(), data.String()
}

func TestWorkThatFailsExitsOneAndSaysWhy(t *testing.T) {
	code, out, _ := run(t, nil, errors.New("disk full"))
	if code != 1 || !strings.Contains(out, "[x]") || !strings.Contains(out, "disk full") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestUsageErrorsExitTwo(t *testing.T) {
	for _, args := range [][]string{{"--nope"}, {"stray"}, {"--lang", "de"}, {"-out", "x"}} {
		code, out, _ := run(t, nil, nil, args...)
		if code != 2 || !strings.Contains(out, "[x]") {
			t.Errorf("%q: code %d, out %q", args, code, out)
		}
	}
}

func TestDoneExitsZero(t *testing.T) {
	if code, out, _ := run(t, nil, nil, "--out", "x"); code != 0 || out != "" {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestLangRendersHelpInPolishWhereverItComes(t *testing.T) {
	for _, args := range [][]string{{"--lang", "pl", "--help"}, {"--help", "--lang", "pl"}} {
		code, _, help := run(t, nil, nil, args...)
		if code != 0 || !strings.Contains(help, "Użycie:") || !strings.Contains(help, "Flagi:") ||
			!strings.Contains(help, "pomoc dla tool") {
			t.Errorf("%q: code %d, help %q", args, code, help)
		}
	}
}

func TestEnvironmentPicksTheLanguage(t *testing.T) {
	_, _, help := run(t, []string{"S99WAILS_LANG=pl"}, nil, "--help")
	if !strings.Contains(help, "Użycie:") {
		t.Fatalf("help %q", help)
	}
	_, _, help = run(t, []string{"S99WAILS_LANG=en"}, nil, "--help")
	if !strings.Contains(help, "Usage:") {
		t.Fatalf("help %q", help)
	}
}

func TestPolishFailureLineUsesPolishWords(t *testing.T) {
	_, out, _ := run(t, nil, errors.New("disk full"), "--lang", "pl")
	if !strings.Contains(out, "błąd") {
		t.Fatalf("out %q", out)
	}
}
