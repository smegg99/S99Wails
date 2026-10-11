// shell/mousenav/mousenav_other.go

//go:build !(linux && cgo && gtk3)

package mousenav

import "github.com/wailsapp/wails/v3/pkg/application"

// Watch is only wired for the GTK3 webview; elsewhere the side buttons stay the webview's.
func Watch(*application.WebviewWindow, func(error)) {}
