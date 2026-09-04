// examples/localeplurals/main.go

package main

import (
	"fmt"
	"testing/fstest"

	"github.com/smegg99/s99wails/v2/locale"
)

func main() {
	locales, err := locale.New(fstest.MapFS{
		"en.json": {Data: []byte(`{
			"greet": "Hello {{.Name}}",
			"items": {"one": "{{.PluralCount}} item", "other": "{{.PluralCount}} items"}
		}`)},
	}, "en")
	if err != nil {
		panic(err)
	}

	fmt.Println(locales.Tf("greet", map[string]any{"Name": "Karol"}))
	fmt.Println(locales.Tn("items", 1, nil))
	fmt.Println(locales.Tn("items", 5, nil))
}
