// examples/env/main.go

package main

import (
	"fmt"
	"os"

	"github.com/smegg99/s99wails/v2/platform/env"
)

func main() {
	forced := env.ApplyLinuxWebkitWorkarounds()
	x11 := env.PreferX11OnPlasmaWayland()

	fmt.Println("running from AppImage:", env.LaunchedViaAppImage(), "(workaround forced:", forced, ")")
	fmt.Println("WEBKIT_DISABLE_DMABUF_RENDERER =", os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER"))
	fmt.Println("GDK_BACKEND defaulted to x11:", x11)
}
