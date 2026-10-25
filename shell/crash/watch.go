// shell/crash/watch.go

package crash

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime/debug"
	"strconv"
	"strings"
)

// watcherEnv carries the watched process's pid to the watcher copy.
const watcherEnv = "S99_CRASH_WATCHER"

// Watch starts a watcher copy of this program that shows the screen when this process crashes, stops
// responding or is killed. Call it first in main: in the watcher copy it never returns.
func Watch(opts Options) error {
	if value := os.Getenv(watcherEnv); value != "" {
		pid, _ := strconv.Atoi(value)
		signal.Ignore(os.Interrupt)
		os.Exit(runWatcher(os.Stdin, opts, pid))
	}
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find executable: %w", err)
	}
	return spawn(exe, os.Args[1:])
}

// spawn starts the watcher as exe args and routes this process's crash output and lines to it.
func spawn(exe string, args []string) error {
	r, w, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("open crash pipe: %w", err)
	}
	cmd := exec.Command(exe, args...)
	cmd.Env = append(os.Environ(), watcherEnv+"="+strconv.Itoa(os.Getpid()))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = r, os.Stdout, os.Stderr
	err = cmd.Start()
	_ = r.Close()
	if err != nil {
		_ = w.Close()
		return fmt.Errorf("start crash watcher: %w", err)
	}
	if err := debug.SetCrashOutput(w, debug.CrashOptions{}); err != nil {
		_ = w.Close()
		return fmt.Errorf("set crash output: %w", err)
	}
	out.Store(newPipe(w))
	Describe(described())
	return nil
}

// runWatcher watches the pipe on its own goroutine; the screen, once needed, runs on this one.
func runWatcher(r io.Reader, opts Options, pid int) int {
	events := make(chan event, 256)
	go readPipe(r, events)
	ui := newWailsScreen(opts)
	w := newWatched(opts, pid, ui.ctl)
	result := make(chan int, 1)
	go func() { result <- w.loop(events, ui) }()
	select {
	case first := <-ui.open:
		if err := ui.run(first); err != nil {
			fmt.Fprintln(os.Stderr, "crash screen:", err)
			return 1
		}
		return 0
	case code := <-result:
		return code
	}
}

// event is one thing read from the pipe: a framed line, a piece of crash output, or the end.
type event struct {
	kind    string
	payload string
}

const (
	eventTrace = "trace"
	eventEnd   = "end"
)

// readPipe turns the pipe into events. A framed line can land inside the runtime's output, so the
// text before a marker is trace.
func readPipe(r io.Reader, events chan<- event) {
	in := bufio.NewReaderSize(r, 64<<10)
	for {
		line, err := in.ReadString('\n')
		if i := strings.Index(line, marker); i >= 0 {
			if i > 0 {
				events <- event{kind: eventTrace, payload: line[:i]}
			}
			kind, payload, _ := strings.Cut(strings.TrimSuffix(line[i+len(marker):], "\n"), " ")
			events <- event{kind: kind, payload: payload}
		} else if line != "" {
			events <- event{kind: eventTrace, payload: line}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				fmt.Fprintln(os.Stderr, "crash pipe:", err)
			}
			events <- event{kind: eventEnd}
			return
		}
	}
}
