// shell/crash/separate_other.go

//go:build !linux

package crash

import "os"

// separate shares w with the runtime here, so w turns blocking and lines go only through the queue.
func separate(w *os.File) (*os.File, bool) {
	return w, false
}
