// services/browser/browser.go

// Package browser opens URLs in the system browser; target="_blank" in a Wails
// webview navigates the webview instead.
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

type Service struct {
	log *s99logger.Logger
}

// New builds the service. A nil log disables logging.
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
