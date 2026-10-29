// shell/crash/crash.go

// Package crash shows the app's own frontend as an error screen after a failed start, panic, hang or kill.
package crash

import (
	"encoding/json"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	// Query is what the screen's window URL carries, so the frontend knows to draw the screen.
	Query = "s99crash"
	// reportEvent carries the new Report whenever it changes while the screen is open.
	reportEvent = "s99crash:report"
)

// Kind says why the screen opened.
type Kind string

const (
	// KindConfig means the settings file could not be read; Failure.File names it.
	KindConfig Kind = "config"
	// KindStartup means another failure before the main window opened.
	KindStartup Kind = "startup"
	// KindCrash means the process died from a panic or a fatal runtime error.
	KindCrash Kind = "crash"
	// KindHang means the process stopped responding; Report.Alive says whether it still runs.
	KindHang Kind = "hang"
	// KindKilled means the process ended without a trace and without a clean exit.
	KindKilled Kind = "killed"
)

// Options configures the screen's application; set them once and pass the same value to Watch and Show.
type Options struct {
	// App is what the screen starts with: Name, Icon, Logger, Assets, Linux and Mac. Services and SingleInstance are replaced.
	App     application.Options
	Version string
	// ReportEmail receives "Email report"; empty hides the action.
	ReportEmail string
	// Width and Height are the screen's size and its minimum, best the main window's minimum; zero means 960x640.
	Width, Height int
}

// Failure is one reason to show the screen.
type Failure struct {
	Kind Kind
	File string
	// Err is the error text, or the whole trace for KindCrash.
	Err string
}

// Field is one extra line in the report, such as the render path.
type Field struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Details is what the app learns while it starts; the screen reads the last value given to Describe.
type Details struct {
	Lang    string  `json:"lang,omitempty"`
	LogFile string  `json:"logFile,omitempty"`
	Fields  []Field `json:"fields,omitempty"`
}

var (
	mu    sync.Mutex
	known Details
)

// Describe records what the screen should know; with Watch running the watcher learns it too.
func Describe(d Details) {
	mu.Lock()
	known = d
	mu.Unlock()
	if line, err := json.Marshal(d); err == nil {
		send(lineDetails, string(line))
	}
}

func described() Details {
	mu.Lock()
	defer mu.Unlock()
	return known
}
