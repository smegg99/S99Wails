// examples/tray/main.go

// A tray icon with a menu: left-click toggles the window, closing the
// window hides it to the tray, quit via the menu.
package main

import (
	_ "embed"
	"log"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/smegg99/s99wails/v2/shell/tray"
)

//go:embed icon.png
var icon []byte

func main() {
	app := application.New(application.Options{Name: "tray example", Icon: icon})
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "tray example", Width: 480, Height: 320,
	})

	manager := tray.NewManager(app, tray.Options{
		Icons:   tray.IconsFunc(func(dark bool) ([]byte, error) { return icon, nil }),
		Tooltip: func() string { return "tray example" },
		OnClick: func() { tray.Toggle(window) },
	})
	manager.RegisterEvents(app)
	defer manager.Close()

	var quitting atomic.Bool
	manager.SetMenu(func() []tray.Item {
		return []tray.Item{
			{Label: "Show", OnClick: func() { tray.Show(window) }},
			{Separator: true},
			{Label: "Quit", OnClick: func() {
				quitting.Store(true)
				app.Quit()
			}},
		}
	})
	tray.CloseToTray(window, func() bool { return true }, quitting.Load)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
