// browser.go
//
// Package browser is a Wails-bound service for opening URLs in the user's
// default system browser. It is the one place a frontend hands an external
// link to the desktop: a target="_blank" inside a Wails webview navigates the
// webview, not the machine's browser, so a link that means to LEAVE the app
// goes through here instead.
//
// The per-platform launch details are the Wails browser manager's; this only
// bridges the call to the frontend bindings and logs it through s99logger.
package browser

import (
	"fmt"

	"github.com/smegg99/s99logger"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	MsgOpenURL       s99logger.MessageID = "opening url in default browser"
	MsgOpenURLFailed s99logger.MessageID = "failed to open url in default browser"
)

// Service opens URLs in the user's default browser. It is bound to the
// frontend, which calls OpenURL with the address to leave for.
type Service struct {
	log *s99logger.Logger
}

// New creates the service. log may be nil to disable logging.
func New(log *s99logger.Logger) *Service {
	return &Service{log: log}
}

func (s *Service) OpenURL(url string) error {
	if s.log != nil {
		s.log.Info(s99logger.NewEvent(MsgOpenURL, s99logger.String("url", url)))
	}

	app := application.Get()
	if app == nil {
		return fmt.Errorf("open url: application not initialized")
	}
	if err := app.Browser.OpenURL(url); err != nil {
		if s.log != nil {
			s.log.Error(s99logger.NewEvent(MsgOpenURLFailed,
				s99logger.String("url", url),
				s99logger.Err(err),
			))
		}
		return err
	}

	return nil
}
