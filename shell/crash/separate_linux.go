// shell/crash/separate_linux.go

//go:build linux

package crash

import (
	"os"
	"strconv"
	"syscall"
)

// separate reopens w as its own file description, so the runtime making its crash output blocking leaves w alone.
func separate(w *os.File) (*os.File, bool) {
	conn, err := w.SyscallConn()
	if err != nil {
		return w, false
	}
	var fd uintptr
	if err := conn.Control(func(f uintptr) { fd = f }); err != nil {
		return w, false
	}
	f, err := os.OpenFile("/proc/self/fd/"+strconv.FormatUint(uint64(fd), 10), os.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return w, false
	}
	return f, true
}
