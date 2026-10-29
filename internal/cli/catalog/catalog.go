// internal/cli/catalog/catalog.go

// Package catalog holds the message catalogue of the module's commands.
package catalog

import (
	"embed"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"

	"github.com/smegg99/s99locale/lang"
)

// EnvVar overrides the OS locale for the commands alone.
const EnvVar = "S99WAILS_LANG"

// Files holds the generated locales/<locale>.json bundles.
//
//go:embed locales/*.json
var Files embed.FS

var bundle = newBundle()

func newBundle() *i18n.Bundle {
	built := i18n.NewBundle(language.Make(DefaultLocale))
	for _, name := range Paths() {
		if _, err := built.LoadMessageFileFS(Files, name); err != nil {
			panic("catalog: load " + name + ": " + err.Error())
		}
	}
	return built
}

// Paths names every bundle in Files, the default locale's first.
func Paths() []string {
	paths := make([]string, 0, len(Locales))
	for _, locale := range Locales {
		paths = append(paths, "locales/"+locale+".json")
	}
	return paths
}

// Localizer renders in one locale, falling back to the default.
func Localizer(locale string) *i18n.Localizer {
	if locale == "" {
		locale = DefaultLocale
	}
	return i18n.NewLocalizer(bundle, locale, DefaultLocale)
}

// FromEnv is the language the commands render in when no --lang was given, read from env.
func FromEnv(env []string) string {
	return lang.Detect(lang.Options{Var: EnvVar, Supported: Locales, Default: DefaultLocale, LookupEnv: lookup(env)})
}

// Supported reports whether a language has a catalogue, and its canonical name.
func Supported(value string) (string, bool) { return lang.Match(Locales, value) }

func lookup(env []string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		for i := len(env) - 1; i >= 0; i-- {
			if key, value, ok := strings.Cut(env[i], "="); ok && key == name {
				return value, true
			}
		}
		return "", false
	}
}
