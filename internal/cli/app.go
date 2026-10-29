// internal/cli/app.go

// Package cli is the shared shape of the module's commands: Cobra for the tree and flags, S99Term for
// output, S99Locale for every message and S99Logger for the log lines.
package cli

import (
	"context"
	"errors"
	"io"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/smegg99/s99logger"
	loggeri18n "github.com/smegg99/s99logger/i18n"
	"github.com/smegg99/s99term"
	"github.com/spf13/cobra"

	"github.com/smegg99/s99wails/v2/internal/cli/catalog"
)

// App is a command's shared state: its two streams, its language, its console and its logger.
type App struct {
	// out carries human lines, written only through the console.
	out io.Writer
	// data carries help and machine output.
	data io.Writer
	env  []string

	language  string
	words     *i18n.Localizer
	describes []func(*i18n.Localizer)
	term      *s99term.Console
	log       *s99logger.Logger
}

// New returns an App writing human lines to out and data to data; env feeds language and colour detection.
func New(out, data io.Writer, env []string) *App {
	return &App{out: out, data: data, env: env}
}

// Data is the machine stream: JSON and the results a script reads.
func (a *App) Data() io.Writer { return a.data }

// Say is the localizer in use.
func (a *App) Say() *i18n.Localizer {
	if a.words == nil {
		a.words = catalog.Localizer(catalog.DefaultLocale)
	}
	return a.words
}

// Describe registers a command's own strings; they render again whenever the language changes.
func (a *App) Describe(f func(*i18n.Localizer)) {
	a.describes = append(a.describes, f)
	f(a.Say())
}

// Console is built on first use, so --lang can change its words during flag parsing.
func (a *App) Console() *s99term.Console {
	if a.term == nil {
		l := a.Say()
		a.term = s99term.New(s99term.Options{Out: a.out, Env: a.env, Words: s99term.Words{
			s99term.StatusOK:      catalog.CliStatusOk(l),
			s99term.StatusFailed:  catalog.CliStatusFailed(l),
			s99term.StatusSkipped: catalog.CliStatusSkipped(l),
			s99term.StatusWaiting: catalog.CliStatusWaiting(l),
			s99term.StatusRunning: catalog.CliStatusRunning(l),
			s99term.StatusWarning: catalog.CliStatusWarning(l),
			s99term.StatusStale:   catalog.CliStatusStale(l),
		}})
	}
	return a.term
}

// Log writes localized records as console lines.
func (a *App) Log() *s99logger.Logger {
	if a.log == nil {
		translator, err := loggeri18n.New(loggeri18n.Options{
			DefaultLanguage: catalog.DefaultLocale, FS: catalog.Files, Files: catalog.Paths(),
		})
		// The bundles are embedded, so a failure here is a build fault every test run shows.
		if err != nil {
			panic("cli: load the log catalogue: " + err.Error())
		}
		a.log = s99logger.New(a.Console().Sink(), s99logger.Options{
			Language: a.language, MinLevel: s99logger.LevelInfo, Translator: translator,
		})
	}
	return a.log
}

// localize renders every registered string in language and drops what holds the old words.
func (a *App) localize(language string) {
	a.language = language
	a.words = catalog.Localizer(language)
	for _, f := range a.describes {
		f(a.words)
	}
	a.reset()
}

func (a *App) reset() {
	if a.term != nil {
		a.term.Close()
	}
	a.term, a.log = nil, nil
}

// Run executes root with args and returns the exit code: 0 when done, 2 for a usage error, 1 for failed work.
func (a *App) Run(ctx context.Context, root *cobra.Command, args []string) int {
	root.SetArgs(args)
	err := root.ExecuteContext(ctx)
	defer a.reset()
	if err == nil {
		return 0
	}
	a.Log().Error(s99logger.NewEvent(catalog.KeyCliFailed, s99logger.Err(err)))
	var usage usageError
	if errors.As(err, &usage) {
		return 2
	}
	return 1
}

// usageError is a command line the command cannot run; it exits 2.
type usageError struct{ err error }

func (e usageError) Error() string { return e.err.Error() }
func (e usageError) Unwrap() error { return e.err }

// Usage marks err as a usage error.
func Usage(err error) error { return usageError{err: err} }
