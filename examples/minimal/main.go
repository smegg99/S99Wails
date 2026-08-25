// examples/minimal/main.go

// Minimal desktop shell using the s99wails packages. Build with:
//
//	go build -tags gtk3 ./examples/minimal
package main

import (
	"embed"
	_ "embed"
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"sync/atomic"

	"github.com/smegg99/s99logger"
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/smegg99/s99wails/env"
	"github.com/smegg99/s99wails/locale"
	browserservice "github.com/smegg99/s99wails/services/browser"
	loggerservice "github.com/smegg99/s99wails/services/logger"
	themeservice "github.com/smegg99/s99wails/services/theme"
	"github.com/smegg99/s99wails/tray"
	"github.com/smegg99/s99wails/windowstate"
)

//go:embed locales
var localesFS embed.FS

//go:embed icon.png
var icon []byte

//go:embed assets
var assetsFS embed.FS

func main() {
	env.ApplyLinuxWebkitWorkarounds()

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
	if _, err := windowstate.Manage(window, jsonFileStore("windowstate.json"), windowstate.Options{}); err != nil {
		log.Println("window state:", err)
	}

	systemTray := app.SystemTray.New()
	manager := tray.NewManager(systemTray, tray.Options{
		Icons:              tray.IconsFunc(func(dark bool) ([]byte, error) { return icon, nil }),
		Tooltip:            func() string { return locales.T("app.name") },
		ThemeChangedEvents: []string{themeservice.EventThemeChanged},
	})
	manager.RegisterEvents(app)
	defer manager.Close()

	// Re-localize the tray menu whenever the language changes.
	locales.OnChange(manager.RebuildMenu)

	// Wails event listeners fire on different goroutines, so the quit flag
	// shared between the tray menu and the closing hook must be atomic.
	var quitting atomic.Bool
	manager.SetMenu(func() *application.Menu {
		menu := app.NewMenu()
		menu.Add(locales.T("tray.show")).OnClick(func(_ *application.Context) { tray.Show(window) })
		menu.AddSeparator()
		menu.Add(locales.T("tray.quit")).OnClick(func(_ *application.Context) {
			quitting.Store(true)
			app.Quit()
		})
		return menu
	})
	tray.OnClickToggle(systemTray, window)
	tray.CloseToTray(window, func() bool { return true }, quitting.Load)

	stopThemeWatcher := themeservice.RegisterThemeWatcher(app)
	defer stopThemeWatcher()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// jsonFileStore is the smallest possible windowstate.Store: one JSON file.
// Real apps typically back the Store with their config instead.
type jsonFileStore string

func (f jsonFileStore) Load() (windowstate.State, bool, error) {
	var state windowstate.State
	data, err := os.ReadFile(string(f))
	if os.IsNotExist(err) {
		return state, false, nil
	}
	if err != nil {
		return state, false, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, false, err
	}
	return state, true, nil
}

func (f jsonFileStore) Save(state windowstate.State) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(string(f), data, 0o644)
}
