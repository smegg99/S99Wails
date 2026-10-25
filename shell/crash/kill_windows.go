// shell/crash/kill_windows.go

//go:build windows

package crash

import "os"

// stop ends the process; Windows has no signal that makes the runtime write its stacks.
func stop(pid int) {
	kill(pid)
}

func kill(pid int) {
	if pid <= 0 {
		return
	}
	if p, err := os.FindProcess(pid); err == nil {
		_ = p.Kill()
	}
}
