// examples/locale/main.go

package main

import (
	"fmt"
	"testing/fstest"

	"github.com/smegg99/s99wails/locale"
)

func main() {
	// Same shape as the `loc build` output in generated/backend, which an app embeds.
	locales, err := locale.New(fstest.MapFS{
		"en.json": {Data: []byte(`{"tray.show": "Show", "tray.quit": "Quit"}`)},
		"pl.json": {Data: []byte(`{"tray.show": "Pokaż"}`)},
	}, "en")
	if err != nil {
		panic(err)
	}

	fmt.Println("en:", locales.T("tray.show"))

	locales.SetLocale("pl-PL") // language matching resolves to pl
	fmt.Println("pl:", locales.T("tray.show"))
	fmt.Println("pl fallback:", locales.T("tray.quit")) // missing in pl, falls back to en
}
