// shell/editkeys/editkeys.go

// Package editkeys sends ctrl+Z and ctrl+shift+Z to the page as events, which Wails on Linux
// may drop, and keeps a failed clipboard TARGETS reply from crashing WebKitGTK 2.54 (X11).
package editkeys

import "github.com/wailsapp/wails/v3/pkg/application"

// Event reports one of the keys: "undo" or "redo".
const Event = "edit:key"

func init() {
	application.RegisterEvent[string](Event)
}

// GDK's keyval for z: a lower-case letter's is its code point.
const keyZ = 'z'

// actionOf names what a key press does: "undo", "redo", or "" for a key the
// page keeps. key is the lower-cased keyval; other is Alt, Super or Meta held.
func actionOf(key uint, control, shift, other bool) string {
	if key != keyZ || !control || other {
		return ""
	}
	if shift {
		return "redo"
	}
	return "undo"
}
