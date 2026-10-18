// shell/editkeys/editkeys_other.go

//go:build !(linux && cgo && gtk3)

package editkeys

import "github.com/wailsapp/wails/v3/pkg/application"

// Watch is only wired for the GTK3 window; elsewhere the keys stay the webview's.
func Watch(*application.WebviewWindow, func(error)) {}
