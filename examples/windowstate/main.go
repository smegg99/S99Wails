// examples/windowstate/main.go

// Move or resize the window, quit, run again: the geometry comes back from
// windowstate.json in the working directory.
package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/smegg99/s99wails/windowstate"
)

type fileStore string

func (f fileStore) Load() (windowstate.State, bool, error) {
	var state windowstate.State
	data, err := os.ReadFile(string(f))
	if os.IsNotExist(err) {
		return state, false, nil
	}
	if err != nil {
		return state, false, err
	}
	return state, true, json.Unmarshal(data, &state)
}

func (f fileStore) Save(state windowstate.State) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(string(f), data, 0o644)
}

func main() {
	store := fileStore("windowstate.json")

	app := application.New(application.Options{Name: "windowstate example"})

	// Configure applies saved geometry before the window opens.
	opts := application.WebviewWindowOptions{
		Title: "windowstate example", Width: 800, Height: 600,
	}
	if err := windowstate.Configure(&opts, store); err != nil {
		log.Println("restore:", err)
	}
	window := app.Window.NewWithOptions(opts)

	defer windowstate.Track(window, store, windowstate.Options{})()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
