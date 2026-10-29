// shell/crash/trywrite_unix_test.go

//go:build !windows

package crash

import (
	"os"
	"strings"
	"testing"
)

// Only a line within POSIX's 512-byte PIPE_BUF is written at once, so none can land half-written on macOS.
func TestTryWriteLeavesALongLineToTheQueue(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	if !tryWrite(w, "short\n") {
		t.Fatal("a short line was not written at once")
	}
	if tryWrite(w, strings.Repeat("x", 600)+"\n") {
		t.Fatal("a line past PIPE_BUF was written at once")
	}
	w.Close()
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	if got := string(buf[:n]); got != "short\n" {
		t.Fatalf("pipe holds %q, want only the short line", got)
	}
}
