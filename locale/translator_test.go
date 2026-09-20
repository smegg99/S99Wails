// locale/translator_test.go

package locale

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/smegg99/s99logger"
)

// loadTranslatorFS is a catalogue with one templated message in two languages.
func loadTranslatorFS(t *testing.T) *Locales {
	t.Helper()
	l, err := New(fstest.MapFS{
		"en.json": {Data: []byte(`{"greeting": "Hello, {{.Name}}"}`)},
		"pl.json": {Data: []byte(`{"greeting": "Witaj, {{.Name}}"}`)},
	}, "en")
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestTranslatorLocalizesAnEventAndItsAttributes(t *testing.T) {
	locales := loadTranslatorFS(t)
	translator := locales.Translator()

	got, err := translator.Translate(context.Background(), "",
		s99logger.NewEvent("greeting", s99logger.String("Name", "Ada")))
	if err != nil {
		t.Fatal(err)
	}
	if got != "Hello, Ada" {
		t.Errorf("message = %q", got)
	}
}

// A named language must not change the active locale.
func TestTranslatorDoesNotDisturbTheActiveLocale(t *testing.T) {
	locales := loadTranslatorFS(t)
	fired := 0
	locales.OnChange(func() { fired++ })

	if _, err := locales.Translator().Translate(context.Background(), "pl",
		s99logger.NewEvent("greeting", s99logger.String("Name", "Ada"))); err != nil {
		t.Fatal(err)
	}
	if fired != 0 {
		t.Errorf("SetLocale fired %d times during a translation", fired)
	}
	if got := locales.T("greeting"); got == "" {
		t.Error("the active catalogue stopped answering")
	}
}

func TestNilCatalogueHasNoTranslator(t *testing.T) {
	var none *Locales
	if none.Translator() != nil {
		t.Error("a nil catalogue produced a translator")
	}
}
