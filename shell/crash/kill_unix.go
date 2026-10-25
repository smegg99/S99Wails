// shell/crash/kill_unix.go

//go:build !windows

package crash

import "syscall"

// stop asks the Go runtime to write every goroutine's stack and exit.
func stop(pid int) {
	if pid > 0 {
		_ = syscall.Kill(pid, syscall.SIGQUIT)
	}
}

func kill(pid int) {
	if pid > 0 {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}
}
