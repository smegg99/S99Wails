// program.go
//
// Package program is a Wails-bound service bridging the frontend to native,
// machine-level actions such as opening URLs in the user's default browser.
// The actual work is delegated to the Wails browser manager, which handles
// the per-platform launch details.
package program

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

// New creates the service. log may be nil to disable logging.
func New(log *s99logger.Logger) *Service {
	return &Service{log: log}
}

// OpenURL opens the given URL in the user's default system browser.
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
