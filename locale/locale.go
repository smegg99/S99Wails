// locale/locale.go

// Package locale wraps go-i18n for native UI strings, loading S99Locale backend JSON.
package locale

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// Locales holds a message bundle and the active localizer. It is safe for
// concurrent use.
type Locales struct {
	bundle        *goi18n.Bundle
	defaultLocale string
	available     []string
	current       atomic.Pointer[goi18n.Localizer]

	mu       sync.Mutex
	onChange []func()
}

// New loads every *.json message file from the root of localeFS (typically an
// embed.FS narrowed with fs.Sub) and selects defaultLocale.
func New(localeFS fs.FS, defaultLocale string) (*Locales, error) {
	tag, err := language.Parse(defaultLocale)
	if err != nil {
		return nil, fmt.Errorf("parse default locale %q: %w", defaultLocale, err)
	}

	bundle := goi18n.NewBundle(tag)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	files, err := fs.Glob(localeFS, "*.json")
	if err != nil {
		return nil, fmt.Errorf("glob locale files: %w", err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no *.json locale files found")
	}
	available := make([]string, 0, len(files))
	for _, file := range files {
		if _, err := bundle.LoadMessageFileFS(localeFS, file); err != nil {
			return nil, fmt.Errorf("load locale file %s: %w", file, err)
		}
		available = append(available, strings.TrimSuffix(file, ".json"))
	}
	sort.Strings(available)

	l := &Locales{bundle: bundle, defaultLocale: defaultLocale, available: available}
	l.current.Store(goi18n.NewLocalizer(bundle, defaultLocale))
	return l, nil
}

// SetLocale switches the active language (pl-PL matches pl, unknown falls back
// to the default) and then fires OnChange callbacks.
func (l *Locales) SetLocale(code string) {
	if code == "" {
		code = l.defaultLocale
	}
	l.current.Store(goi18n.NewLocalizer(l.bundle, code, l.defaultLocale))

	l.mu.Lock()
	callbacks := append([]func(){}, l.onChange...)
	l.mu.Unlock()
	for _, cb := range callbacks {
		cb()
	}
}

// OnChange registers a callback run after every SetLocale.
func (l *Locales) OnChange(cb func()) {
	if cb == nil {
		return
	}
	l.mu.Lock()
	l.onChange = append(l.onChange, cb)
	l.mu.Unlock()
}

// Available lists the loaded locale codes, sorted.
func (l *Locales) Available() []string {
	return append([]string{}, l.available...)
}

func (l *Locales) Current() *goi18n.Localizer {
	return l.current.Load()
}

// T returns the message for messageID, or messageID when no translation exists.
func (l *Locales) T(messageID string) string {
	return l.localize(&goi18n.LocalizeConfig{MessageID: messageID})
}

// Tf localizes a message with template data.
func (l *Locales) Tf(messageID string, data map[string]any) string {
	return l.localize(&goi18n.LocalizeConfig{MessageID: messageID, TemplateData: data})
}

// Tn localizes a plural message; count is also exposed as {{.PluralCount}}.
func (l *Locales) Tn(messageID string, count int, data map[string]any) string {
	// Copy so the caller's map is not mutated.
	merged := make(map[string]any, len(data)+1)
	for k, v := range data {
		merged[k] = v
	}
	if _, ok := merged["PluralCount"]; !ok {
		merged["PluralCount"] = count
	}
	return l.localize(&goi18n.LocalizeConfig{MessageID: messageID, PluralCount: count, TemplateData: merged})
}

func (l *Locales) localize(cfg *goi18n.LocalizeConfig) string {
	return l.localizeWith(l.Current(), cfg)
}

// localizeWith uses a given localizer without changing the active one.
func (l *Locales) localizeWith(loc *goi18n.Localizer, cfg *goi18n.LocalizeConfig) string {
	// go-i18n can return a fallback message together with an error; prefer the message.
	msg, err := loc.Localize(cfg)
	if msg != "" {
		return msg
	}
	if err != nil {
		return cfg.MessageID
	}
	return msg
}
