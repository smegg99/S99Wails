// window.go
package tray

import (
	"fmt"
	"runtime"
	"strings"

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

// OnClickToggle makes a left click on the tray icon toggle the window,
// working around the Wails v3 alpha behavior on Linux where the DBus menu
// "opened" event fires the normal click handler (which would make
// right-clicking the tray icon toggle the window too).
func OnClickToggle(tray *application.SystemTray, win *application.WebviewWindow) {
	tray.OnClick(func() {
		if isLinuxTrayMenuOpenCallback() {
			return
		}
		Toggle(win)
	})
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

// isLinuxTrayMenuOpenCallback reports whether the current call stack
// originates from the Linux DBus tray menu "opened" event.
func isLinuxTrayMenuOpenCallback() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	var pcs [16]uintptr
	n := runtime.Callers(2, pcs[:])
	for _, pc := range pcs[:n] {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		if strings.Contains(fn.Name(), "github.com/wailsapp/wails/v3/pkg/application.(*linuxSystemTray).Event") {
			return true
		}
	}
	return false
}
