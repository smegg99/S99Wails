// examples/accentwatch/main.go

// Watches the OS accent and color scheme; change them in system settings, Ctrl-C quits.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/smegg99/s99wails/v2/platform/accent"
)

func main() {
	stop := accent.Watch(func(info accent.Info) {
		fmt.Printf("changed: scheme=%s dark=%v accent=%s\n", info.Scheme, info.DarkMode, info.AccentColor)
	})
	defer stop()

	fmt.Println("watching, Ctrl-C to quit")
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	<-ctx.Done()
}
