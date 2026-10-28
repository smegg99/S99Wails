// shell/crash/service.go

package crash

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/smegg99/s99wails/v2/services/title"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Service is what the screen calls, by name, through the s99wails crash composable.
type Service struct {
	mu     sync.Mutex
	report Report
	name   string
	email  string
	ctl    controls
}

var (
	errNoApp     = errors.New("application not initialized")
	errNotHanged = errors.New("the app is not waiting on this screen")
	errAlive     = errors.New("the app still runs; wait for it or kill it")
)

func newService(opts Options, r Report, ctl controls) *Service {
	return &Service{report: r, name: opts.App.Name, email: opts.ReportEmail, ctl: ctl}
}

func (s *Service) Report() Report {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.report
}

func (s *Service) set(r Report) {
	s.mu.Lock()
	s.report = r
	s.mu.Unlock()
}

// Wait hides the not-responding screen and gives the app another hang period.
func (s *Service) Wait() error {
	if s.ctl.wait == nil {
		return errNotHanged
	}
	s.ctl.signal(s.ctl.wait)
	return nil
}

// Kill stops the app that is not responding; its stacks then fill the report.
func (s *Service) Kill() error {
	if s.ctl.kill == nil {
		return errNotHanged
	}
	s.ctl.signal(s.ctl.kill)
	return nil
}

// SetTitle names the screen's window: the app's name, then phrase, the way services/title joins them.
func (s *Service) SetTitle(ctx context.Context, phrase string) {
	window, ok := ctx.Value(application.WindowKey).(application.Window)
	if !ok {
		return
	}
	text := s.name
	if phrase != "" {
		text += title.DefaultSep + phrase
	}
	window.SetTitle(text)
}

// Copy puts the report on the clipboard and says whether it worked.
func (s *Service) Copy() bool {
	app := application.Get()
	return app != nil && app.Clipboard.SetText(s.Report().Text)
}

// Save asks where to write the report and returns the path, or "" when the dialog was cancelled.
func (s *Service) Save() (string, error) {
	app := application.Get()
	if app == nil {
		return "", errNoApp
	}
	r := s.Report()
	dialog := app.Dialog.SaveFile().SetFilename(reportName(r, time.Now())).CanCreateDirectories(true)
	if home, err := os.UserHomeDir(); err == nil {
		dialog.SetDirectory(home)
	}
	path, err := dialog.PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}
	if err := os.WriteFile(path, []byte(r.Text), 0o644); err != nil {
		return "", fmt.Errorf("write report: %w", err)
	}
	return path, nil
}

// Email opens the user's mail client addressed to Options.ReportEmail with the report and the log file
// attached. Where the client cannot take attachments, the report goes in a mailto body instead.
func (s *Service) Email() error {
	app := application.Get()
	if app == nil {
		return errNoApp
	}
	if s.email == "" {
		return errors.New("no report address configured")
	}
	r := s.Report()
	if err := s.attach(r); err == nil {
		return nil
	}
	return app.Browser.OpenURL(mailto(s.email, r))
}

// OpenLogs shows the log file in the file manager.
func (s *Service) OpenLogs() error {
	app := application.Get()
	if app == nil {
		return errNoApp
	}
	file := s.Report().LogFile
	if !exists(file) {
		return errors.New("no log file")
	}
	return app.Env.OpenFileManager(file, true)
}

// Restart starts the program again and closes the screen; refused while the hung app still runs.
func (s *Service) Restart() error {
	if s.Report().Alive {
		return errAlive
	}
	app := application.Get()
	if app == nil {
		return errNoApp
	}
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find executable: %w", err)
	}
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("restart: %w", err)
	}
	_ = cmd.Process.Release()
	app.Quit()
	return nil
}

// Close closes the screen; refused while the hung app still runs.
func (s *Service) Close() error {
	if s.Report().Alive {
		return errAlive
	}
	if app := application.Get(); app != nil {
		app.Quit()
	}
	return nil
}

func slug(name string) string {
	name = strings.ToLower(strings.Join(strings.Fields(name), "-"))
	if name == "" {
		return "report"
	}
	return filepath.Base(name)
}
