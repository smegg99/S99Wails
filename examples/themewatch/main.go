// examples/themewatch/main.go
//
// Prints the theme once, then each theme:changed event. Change the OS accent or
// dark mode to see events; Ctrl-C quits.
package main

import (
	"fmt"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	theme "github.com/smegg99/s99wails/services/theme"
)

func main() {
	svc := &theme.Service{}
	app := application.New(application.Options{
		Name:     "themewatch example",
		Services: []application.Service{application.NewService(svc)},
	})

	app.Event.On(theme.EventThemeChanged, func(e *application.CustomEvent) {
		fmt.Printf("changed: %+v\n", e.Data)
	})
	stop := theme.RegisterThemeWatcher(app)
	defer stop()

	fmt.Printf("current: %+v\n", svc.GetTheme())
	fmt.Println("flip your OS accent or dark mode, Ctrl-C to quit")

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
