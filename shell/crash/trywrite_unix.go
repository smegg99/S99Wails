// shell/crash/trywrite_unix.go

//go:build !windows

package crash

import (
	"os"

	"golang.org/x/sys/unix"
)

// pipeAtomic is POSIX's minimum PIPE_BUF (macOS's own): a write that long is never split.
const pipeAtomic = 512

// tryWrite writes a line of at most pipeAtomic bytes now if the pipe has room; longer lines go through the queue.
func tryWrite(f *os.File, line string) bool {
	if len(line) > pipeAtomic {
		return false
	}
	conn, err := f.SyscallConn()
	if err != nil {
		return false
	}
	written := false
	err = conn.Write(func(fd uintptr) bool {
		n, err := unix.Write(int(fd), []byte(line))
		written = err == nil && n == len(line)
		return true
	})
	return err == nil && written
}
