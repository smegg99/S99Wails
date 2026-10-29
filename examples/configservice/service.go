// examples/configservice/service.go

package configservice

import "github.com/wailsapp/wails/v3/pkg/application"

const (
	EventConfigChanged      = "config:changed"
	EventPreferencesChanged = "config:preferences:changed"
)

// Service is the Wails-bound config service, registered with application.NewService.
type Service struct {
	// OnLanguageChanged receives the configured language after every successful
	// write, for example locale.SetLocale.
	OnLanguageChanged func(language string)
}

func (s *Service) GetConfig() Config {
	return Get()
}

// SetConfig replaces the entire config on disk and in memory.
func (s *Service) SetConfig(c Config) error {
	if err := SetConfig(c); err != nil {
		return err
	}
	s.afterWrite()
	return nil
}

func (s *Service) GetPreferences() Preferences {
	return Get().Preferences
}

func (s *Service) SetPreferences(p Preferences) error {
	if err := SetPreferences(p); err != nil {
		return err
	}
	s.afterWrite()
	return nil
}

func (s *Service) GetLoggerConfig() LoggerConfig {
	return Get().Logger
}

func (s *Service) SetLoggerConfig(lc LoggerConfig) error {
	if err := SetLoggerConfig(lc); err != nil {
		return err
	}
	s.afterWrite()
	return nil
}

func (s *Service) afterWrite() {
	if s.OnLanguageChanged != nil {
		if lang := Get().Preferences.Language; lang != "" {
			s.OnLanguageChanged(lang)
		}
	}
	if app := application.Get(); app != nil {
		updated := Get()
		app.Event.Emit(EventPreferencesChanged, updated.Preferences)
		app.Event.Emit(EventConfigChanged, updated)
	}
}
