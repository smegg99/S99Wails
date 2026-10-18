// shell/editkeys/editkeys.go

// Package editkeys sends undo, redo and empty-clipboard paste to the page as events:
// Wails on Linux drops some of these keys, and WebKitGTK 2.54 (X11) crashes on the paste.
package editkeys

import "github.com/wailsapp/wails/v3/pkg/application"

// Event reports one of the keys: "undo", "redo", or "paste" with nothing to paste.
const Event = "edit:key"

func init() {
	application.RegisterEvent[string](Event)
}

// GDK's keyvals: a lower-case letter's is its code point.
const (
	keyZ      = 'z'
	keyV      = 'v'
	keyInsert = 0xff63
)

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

// pastes matches ctrl+V, ctrl+shift+V or shift+Insert. key is the lower-cased
// keyval; other means Alt, Super or Meta is held.
func pastes(key uint, control, shift, other bool) bool {
	switch {
	case other:
		return false
	case key == keyV:
		return control
	case key == keyInsert:
		return shift && !control
	}
	return false
}
