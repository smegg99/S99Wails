//go:build !linux || !cgo || !gtk3

package windows

import "unsafe"

func Pointer() (x, y int, held, supported bool)   { return 0, 0, false, false }
func Under(unsafe.Pointer) (x, y int, under bool) { return 0, 0, false }
