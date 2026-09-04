// examples/localechange/main.go

// OnChange fires after every SetLocale; in an app you wire it to
// trayManager.RebuildMenu so native menus re-localize.
package main

import (
	"fmt"
	"testing/fstest"

	"github.com/smegg99/s99wails/v2/locale"
)

func main() {
	locales, err := locale.New(fstest.MapFS{
		"en.json": {Data: []byte(`{"tray.show": "Show"}`)},
		"pl.json": {Data: []byte(`{"tray.show": "Pokaż"}`)},
	}, "en")
	if err != nil {
		panic(err)
	}

	fmt.Println("available:", locales.Available())

	locales.OnChange(func() {
		fmt.Println("locale changed, menu would rebuild as:", locales.T("tray.show"))
	})

	locales.SetLocale("pl")
	locales.SetLocale("en")
}
