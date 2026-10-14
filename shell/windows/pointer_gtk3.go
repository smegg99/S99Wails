//go:build linux && cgo && gtk3

package windows

/*
#cgo pkg-config: gtk+-3.0 x11
#include "pointer_gtk3.h"
*/
import "C"

import "unsafe"

// Pointer and Under must run on GTK's GUI thread. Positions are logical pixels.
func Pointer() (x, y int, held, supported bool) {
	var px, py, down C.int
	ok := C.pointerSample(&px, &py, &down)
	return int(px), int(py), down != 0, ok != 0
}
func Under(native unsafe.Pointer) (x, y int, under bool) {
	var px, py C.int
	ok := C.pointerWindow(native, &px, &py)
	return int(px), int(py), ok != 0
}
