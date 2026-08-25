// locale/example_test.go

package locale_test

import (
	"fmt"
	"testing/fstest"

	"github.com/smegg99/s99wails/locale"
)

func exampleFS() fstest.MapFS {
	return fstest.MapFS{
		"en.json": {Data: []byte(`{
			"tray.show": "Show",
			"tray.quit": "Quit",
			"greet": "Hello {{.Name}}",
			"items": {"one": "{{.PluralCount}} item", "other": "{{.PluralCount}} items"}
		}`)},
		"pl.json": {Data: []byte(`{"tray.show": "Pokaż"}`)},
	}
}

func Example() {
	locales, err := locale.New(exampleFS(), "en")
	if err != nil {
		panic(err)
	}

	fmt.Println(locales.Available())
	fmt.Println(locales.T("tray.show"))
	fmt.Println(locales.Tf("greet", map[string]any{"Name": "Karol"}))
	fmt.Println(locales.Tn("items", 1, nil), "/", locales.Tn("items", 5, nil))

	locales.SetLocale("pl-PL") // language matching resolves to pl
	fmt.Println(locales.T("tray.show"))
	fmt.Println(locales.T("tray.quit")) // missing in pl -> default-locale fallback
	// Output:
	// [en pl]
	// Show
	// Hello Karol
	// 1 item / 5 items
	// Pokaż
	// Quit
}

func ExampleLocales_OnChange() {
	locales, _ := locale.New(exampleFS(), "en")
	locales.OnChange(func() { fmt.Println("re-localize native UI (tray menus etc.)") })
	locales.SetLocale("pl")
	// Output:
	// re-localize native UI (tray menus etc.)
}
