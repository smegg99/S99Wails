// locale/translator.go

package locale

import (
	"context"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/smegg99/s99logger"
)

// Translator adapts the catalogue to s99logger; event attributes become template
// data. A nil catalogue yields a nil Translator (no translation).
func (l *Locales) Translator() s99logger.Translator {
	if l == nil {
		return nil
	}
	return translator{locales: l}
}

type translator struct{ locales *Locales }

// Translate localizes one event. A named lang does not change the active locale.
func (t translator) Translate(_ context.Context, lang string, event s99logger.Event) (string, error) {
	localizer := t.locales.Current()
	if lang != "" {
		localizer = goi18n.NewLocalizer(t.locales.bundle, lang, t.locales.defaultLocale)
	}

	data := make(map[string]any, len(event.Attrs()))
	for _, attr := range event.Attrs() {
		data[attr.Key] = attr.Value
	}

	cfg := &goi18n.LocalizeConfig{MessageID: string(event.MessageID()), TemplateData: data}
	if count, plural := event.PluralCount(); plural {
		cfg.PluralCount = count
		if _, taken := data["PluralCount"]; !taken {
			data["PluralCount"] = count
		}
	}
	return t.locales.localizeWith(localizer, cfg), nil
}
