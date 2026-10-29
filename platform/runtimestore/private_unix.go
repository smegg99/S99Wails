//go:build unix

// platform/runtimestore/private_unix.go

package runtimestore

import (
	"io/fs"
	"os"
	"syscall"
)

// private reports a directory only this user owns and can enter, as XDG requires of the runtime dir.
func private(info fs.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(stat.Uid) == os.Getuid() && info.Mode().Perm()&0o077 == 0
}
