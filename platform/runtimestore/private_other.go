//go:build !unix

// platform/runtimestore/private_other.go

package runtimestore

import "io/fs"

// private is never true here: only unix systems have an XDG runtime directory.
func private(fs.FileInfo) bool { return false }
