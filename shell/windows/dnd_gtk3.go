//go:build linux && cgo && gtk3

package windows

/*
#cgo pkg-config: gtk+-3.0
#include "dnd_gtk3.h"
*/
import "C"

import "unsafe"

// The functions below run on GTK's GUI thread only.

func dndWayland() bool { return C.s99DndWayland() != 0 }

func dndAttach(native unsafe.Pointer, handle uintptr) bool {
	return C.s99DndAttach(native, C.uintptr_t(handle)) != 0
}

// dndLocal reports the pointer in the window's webview coordinates, logical pixels.
func dndLocal(native unsafe.Pointer) (x, y int, held, inside, ok bool) {
	var px, py, down, in C.int
	ok = C.s99DndLocal(native, &px, &py, &down, &in) != 0
	return int(px), int(py), down != 0, in != 0, ok
}

func dndBegin(native unsafe.Pointer) bool { return C.s99DndBegin(native) != 0 }
func dndCancel()                          { C.s99DndCancel() }

// dndImage takes the page's PNG of the dragged tab, held at x, y logical pixels.
func dndImage(png []byte, scale, x, y int) bool {
	if len(png) == 0 {
		return false
	}
	return C.s99DndImage(unsafe.Pointer(&png[0]), C.int(len(png)), C.int(scale), C.int(x), C.int(y)) != 0
}
func dndIcon(show bool)            { C.s99DndIcon(cBool(show)) }
func dndFloat(show bool, x, y int) { C.s99DndFloat(cBool(show), C.int(x), C.int(y)) }
func dndClear()                    { C.s99DndClear() }

func cBool(v bool) C.int {
	if v {
		return 1
	}
	return 0
}

//export s99wailsDndMotion
func s99wailsDndMotion(handle C.uintptr_t, x, y C.int) {
	dndPost(dndEvent{kind: dndMotion, handle: uintptr(handle), x: int(x), y: int(y)})
}

//export s99wailsDndLeave
func s99wailsDndLeave(handle C.uintptr_t) {
	dndPost(dndEvent{kind: dndLeave, handle: uintptr(handle)})
}

//export s99wailsDndDrop
func s99wailsDndDrop(handle C.uintptr_t, x, y C.int) {
	dndPost(dndEvent{kind: dndDrop, handle: uintptr(handle), x: int(x), y: int(y)})
}

//export s99wailsDndFailed
func s99wailsDndFailed(result C.int) {
	dndPost(dndEvent{kind: dndFailed, result: int(result)})
}

//export s99wailsDndEnd
func s99wailsDndEnd() { dndPost(dndEvent{kind: dndEnd}) }
