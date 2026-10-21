// shell/tray/window.go

package tray

import (
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Show shows and focuses the window. Nil-safe.
func Show(win *application.WebviewWindow) {
	if win == nil {
		return
	}
	win.Show()
	win.Focus()
}

// Toggle hides a visible window and shows a hidden one.
func Toggle(win *application.WebviewWindow) {
	if win == nil {
		return
	}
	if win.IsVisible() {
		win.Hide()
		return
	}
	Show(win)
}

// CloseToTray hides the window instead of closing it while enabled() is true.
// quitting() lets an explicit tray Quit action bypass the hook.
func CloseToTray(win *application.WebviewWindow, enabled func() bool, quitting func() bool) {
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if quitting != nil && quitting() {
			return
		}
		if enabled != nil && enabled() {
			win.Hide()
			e.Cancel()
		}
	})
}

func recoveredError(r any) error {
	if err, ok := r.(error); ok {
		return err
	}
	return fmt.Errorf("panic: %v", r)
}
