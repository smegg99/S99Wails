// shell/update/update.go

// Package update checks desktop releases and tracks which versions an installation has handled.
package update

import (
	"net/http"
	"sync"

	"github.com/smegg99/s99base/release/ghreleases"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Options supplies the release source, installation state and application callbacks.
type Options struct {
	Owner, Repo string
	Version     string
	DownloadURL string
	Changelog   string
	Store       Store
	Enabled     func() bool
	Existing    func() bool
	APIBase     string
	HTTPClient  *http.Client
	OnError     func(error)
}

// Release is a newer version and the page from which it can be downloaded.
type Release struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

// Service exposes release checks and installation state to the desktop frontend.
type Service struct {
	opts           Options
	client         *ghreleases.Client
	checking       sync.Mutex
	mu             sync.Mutex
	latest         *Release
	memory         Memory
	loaded, stored bool
	emit           func(string, any)
}

// New constructs a service without reading storage or contacting GitHub.
func New(opts Options) *Service {
	return &Service{
		opts: opts,
		client: ghreleases.New(ghreleases.Options{
			Owner: opts.Owner, Repo: opts.Repo, BaseURL: opts.APIBase,
			UserAgent: opts.Repo + "/" + opts.Version, HTTPClient: opts.HTTPClient,
		}),
		emit: func(name string, data any) {
			if app := application.Get(); app != nil {
				app.Event.Emit(name, data)
			}
		},
	}
}

func (s *Service) report(err error) error {
	if err != nil && s.opts.OnError != nil {
		s.opts.OnError(err)
	}
	return err
}
