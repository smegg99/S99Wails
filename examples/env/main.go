// examples/env/main.go

package main

import (
	"fmt"
	"os"

	"github.com/smegg99/s99wails/v2/platform/env"
)

func main() {
	result := env.Setup(env.Options{})

	fmt.Println("running from AppImage:", result.AppImage)
	fmt.Println("render path:", result.RenderPath, "selected by:", result.RenderPathFrom)
	fmt.Println("WEBKIT_DISABLE_DMABUF_RENDERER =", os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER"))
	fmt.Println("GDK_BACKEND defaulted to x11:", result.X11Preferred)
}
