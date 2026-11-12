// shell/update/service.go

package update

import (
	"context"
	"errors"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var firstDelay, checkEvery = 10 * time.Second, 6 * time.Hour

// Download opens the pending release's download page in the system browser.
func (s *Service) Download() error {
	s.mu.Lock()
	var url string
	if s.latest != nil {
		url = s.latest.URL
	}
	s.mu.Unlock()
	if url == "" {
		return errors.New("update: no download available")
	}
	app := application.Get()
	if app == nil {
		return errors.New("update: application not initialized")
	}
	return app.Browser.OpenURL(url)
}

// Schedule checks after ten seconds and every six hours until the app stops or stop is called.
func Schedule(s *Service) (stop func()) {
	parent := context.Background()
	if app := application.Get(); app != nil {
		parent = app.Context()
	}
	ctx, cancel := context.WithCancel(parent)
	delay, interval := firstDelay, checkEvery
	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				if ctx.Err() != nil {
					return
				}
				_ = s.Check(ctx)
				timer.Reset(interval)
			}
		}
	}()
	return cancel
}
