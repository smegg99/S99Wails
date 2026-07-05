// theme.go
//
// Package theme exposes the OS dark mode and accent color to the frontend.
package theme

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/smegg99/s99wails/accent"
)

const (
	// EventThemeChanged is emitted to the frontend whenever the system theme
	// or accent color changes.
	EventThemeChanged = "theme:changed"
)

type Service struct{}

// GetTheme returns the current device theme info.
func (s *Service) GetTheme() accent.Info {
	return currentTheme()
}

// RegisterThemeWatcher emits EventThemeChanged on theme or accent changes. stop
// ends the accent watcher; the Wails listener lives as long as the app.
func RegisterThemeWatcher(app *application.App) (stop func()) {
	app.Event.OnApplicationEvent(events.Common.ThemeChanged, func(_ *application.ApplicationEvent) {
		app.Event.Emit(EventThemeChanged, currentTheme())
	})

	return accent.Watch(func(info accent.Info) {
		app.Event.Emit(EventThemeChanged, enrich(info))
	})
}

func currentTheme() accent.Info {
	return enrich(accent.Snapshot())
}

// enrich fills inconclusive native detection with the Wails environment.
func enrich(info accent.Info) accent.Info {
	app := application.Get()
	if app == nil || app.Env == nil {
		return info
	}

	if info.Scheme == accent.SchemeNoPreference {
		info.DarkMode = app.Env.IsDarkMode()
	}

	if info.AccentColor == "" {
		if wailsAccent := app.Env.GetAccentColor(); wailsAccent != "" {
			info.AccentColor = accent.NormalizeToHex(wailsAccent)
			info.AccentWatchSupported = info.AccentColor != ""
		}
	}

	return info
}
