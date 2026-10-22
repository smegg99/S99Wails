//go:build !linux || !cgo || !gtk3

package windows

import "unsafe"

func dndWayland() bool                                          { return false }
func dndAttach(unsafe.Pointer, uintptr) bool                    { return false }
func dndLocal(unsafe.Pointer) (x, y int, held, inside, ok bool) { return 0, 0, false, false, false }
func dndBegin(unsafe.Pointer) bool                              { return false }
func dndCancel()                                                {}
func dndImage([]byte, int, int, int) bool                       { return false }
func dndIcon(bool)                                              {}
func dndFloat(bool, int, int)                                   {}
func dndClear()                                                 {}
