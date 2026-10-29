// examples/minimal/main.go

// Minimal desktop shell on the s99wails packages; build with -tags gtk3.
package main

import (
	"embed"
	_ "embed"
	"io/fs"
	"log"
	"os"
	"sync/atomic"

	"github.com/smegg99/s99logger"
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/smegg99/s99wails/v2/locale"
	"github.com/smegg99/s99wails/v2/platform/env"
	browserservice "github.com/smegg99/s99wails/v2/services/browser"
	loggerservice "github.com/smegg99/s99wails/v2/services/logger"
	themeservice "github.com/smegg99/s99wails/v2/services/theme"
	"github.com/smegg99/s99wails/v2/shell/tray"
	"github.com/smegg99/s99wails/v2/shell/windowstate"
)

//go:embed locales
var localesFS embed.FS

//go:embed icon.png
var icon []byte

//go:embed assets
var assetsFS embed.FS

func main() {
	env.Setup(env.Options{})

	localeDir, err := fs.Sub(localesFS, "locales")
	if err != nil {
		log.Fatal(err)
	}
	locales, err := locale.New(localeDir, "en")
	if err != nil {
		log.Fatal(err)
	}

	logger := s99logger.New(s99logger.NewConsoleSink(os.Stderr), s99logger.Options{Service: "example"})

	assets, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		log.Fatal(err)
	}

	app := application.New(application.Options{
		Name: locales.T("app.name"),
		Icon: icon,
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Services: []application.Service{
			application.NewService(loggerservice.New(logger)),
			application.NewService(browserservice.New(logger)),
			application.NewService(&themeservice.Service{}),
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: locales.T("app.name"),
		URL:   "/",
	})

	// Persist window geometry across runs next to the binary.
	if _, err := windowstate.Manage(window, windowstate.FileStore("windowstate.json"), windowstate.Options{}); err != nil {
		log.Println("window state:", err)
	}

	manager := tray.NewManager(app, tray.Options{
		Icons:              tray.IconsFunc(func(dark bool) ([]byte, error) { return icon, nil }),
		Tooltip:            func() string { return locales.T("app.name") },
		OnClick:            func() { tray.Toggle(window) },
		ThemeChangedEvents: []string{themeservice.EventThemeChanged},
	})
	manager.RegisterEvents(app)
	defer manager.Close()

	// Re-localize the tray menu whenever the language changes.
	locales.OnChange(manager.RebuildMenu)

	// Wails event listeners fire on different goroutines, so the quit flag
	// shared between the tray menu and the closing hook must be atomic.
	var quitting atomic.Bool
	manager.SetMenu(func() []tray.Item {
		return []tray.Item{
			{Label: locales.T("tray.show"), OnClick: func() { tray.Show(window) }},
			{Separator: true},
			{Label: locales.T("tray.quit"), OnClick: func() {
				quitting.Store(true)
				app.Quit()
			}},
		}
	})
	tray.CloseToTray(window, func() bool { return true }, quitting.Load)

	stopThemeWatcher := themeservice.RegisterThemeWatcher(app)
	defer stopThemeWatcher()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
