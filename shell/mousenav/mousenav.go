// shell/mousenav/mousenav.go

// Package mousenav turns a mouse's back and forward buttons into steps
// through the window's history, as a browser takes them.
package mousenav

import "github.com/wailsapp/wails/v3/pkg/application"

// eventStep reports a side button: -1 for back, 1 for forward.
const eventStep = "mouse:step"

func init() {
	application.RegisterEvent[int](eventStep)
}

// stepOf maps an X11 pointer button to a step: 8 is back, 9 is forward.
func stepOf(button uint) int {
	switch button {
	case 8:
		return -1
	case 9:
		return 1
	}
	return 0
}
