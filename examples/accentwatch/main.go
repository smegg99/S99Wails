// examples/accentwatch/main.go

// Watches the OS for accent and color-scheme changes. Flip the accent color
// or dark mode in your system settings, Ctrl-C to quit.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/smegg99/s99wails/accent"
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
