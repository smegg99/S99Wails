//go:build !linux

// shell/tray/wails_other.go

package tray

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// wailsTray is Wails' own tray, used on Windows and macOS, which take pixels and have no themed icons.
type wailsTray struct {
	app      *application.App
	tray     *application.SystemTray
	template bool
}

func newPlatform(app *application.App, opts Options) backend {
	tray := app.SystemTray.New()
	if opts.OnClick != nil {
		tray.OnClick(opts.OnClick)
	}
	return &wailsTray{app: app, tray: tray, template: opts.TemplateIcon}
}

func (w *wailsTray) setIcon(icon []byte) error {
	if runtime.GOOS == "darwin" && w.template {
		w.tray.SetTemplateIcon(icon)
		return nil
	}
	w.tray.SetIcon(icon)
	return nil
}

func (w *wailsTray) setTooltip(text string) { w.tray.SetTooltip(text) }

func (w *wailsTray) setMenu(items []Item) {
	menu := w.app.NewMenu()
	for _, item := range items {
		if item.Separator {
			menu.AddSeparator()
			continue
		}
		click := item.OnClick
		menu.Add(item.Label).OnClick(func(*application.Context) {
			if click != nil {
				click()
			}
		})
	}
	w.tray.SetMenu(menu)
}

// close leaves the tray to Wails, which removes it when the application quits.
func (w *wailsTray) close() {}
