// shell/crash/trywrite_unix.go

//go:build !windows

package crash

import (
	"os"

	"golang.org/x/sys/unix"
)

// tryWrite writes line now if the pipe has room; a line under PIPE_BUF goes out whole or not at all.
func tryWrite(f *os.File, line string) bool {
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
