// examples/tray/main.go

// A tray icon with a menu: left-click toggles the window, closing the
// window hides it to the tray, quit via the menu.
package main

import (
	_ "embed"
	"log"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/smegg99/s99wails/tray"
)

//go:embed icon.png
var icon []byte

func main() {
	app := application.New(application.Options{Name: "tray example", Icon: icon})
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "tray example", Width: 480, Height: 320,
	})

	systemTray := app.SystemTray.New()
	manager := tray.NewManager(systemTray, tray.Options{
		Icons:   tray.IconsFunc(func(dark bool) ([]byte, error) { return icon, nil }),
		Tooltip: func() string { return "tray example" },
	})
	manager.RegisterEvents(app)
	defer manager.Close()

	var quitting atomic.Bool
	manager.SetMenu(func() *application.Menu {
		menu := app.NewMenu()
		menu.Add("Show").OnClick(func(_ *application.Context) { tray.Show(window) })
		menu.AddSeparator()
		menu.Add("Quit").OnClick(func(_ *application.Context) {
			quitting.Store(true)
			app.Quit()
		})
		return menu
	})
	tray.OnClickToggle(systemTray, window)
	tray.CloseToTray(window, func() bool { return true }, quitting.Load)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
