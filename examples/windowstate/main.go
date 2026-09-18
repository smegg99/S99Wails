// examples/windowstate/main.go

// Move or resize the window, quit, run again: the geometry comes back from
// windowstate.json in the working directory.
package main

import (
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/smegg99/s99wails/v2/shell/windowstate"
)

func main() {
	store := windowstate.FileStore("windowstate.json")

	app := application.New(application.Options{Name: "windowstate example"})

	// Configure applies saved geometry before the window opens.
	opts := application.WebviewWindowOptions{
		Title: "windowstate example", Width: 800, Height: 600,
	}
	if _, err := windowstate.Configure(&opts, store); err != nil {
		log.Println("restore:", err)
	}
	window := app.Window.NewWithOptions(opts)

	defer windowstate.Track(window, store, windowstate.Options{})()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
