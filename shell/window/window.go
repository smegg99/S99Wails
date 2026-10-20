// shell/window/window.go

// Package window holds helpers for Wails v3 webview window options.
package window

import "github.com/wailsapp/wails/v3/pkg/application"

// Limits are resize constraints; a zero field is unset.
type Limits struct {
	MinWidth  int
	MinHeight int
	MaxWidth  int
	MaxHeight int
}

// Apply writes the non-zero limits onto opts. Call before app.Window.NewWithOptions.
func Apply(opts *application.WebviewWindowOptions, l Limits) {
	if l.MinWidth > 0 {
		opts.MinWidth = l.MinWidth
	}
	if l.MinHeight > 0 {
		opts.MinHeight = l.MinHeight
	}
	if l.MaxWidth > 0 {
		opts.MaxWidth = l.MaxWidth
	}
	if l.MaxHeight > 0 {
		opts.MaxHeight = l.MaxHeight
	}
}
