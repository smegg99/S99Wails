// shell/crash/watch_test.go

package crash

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	helperOut  = "S99_CRASH_TEST_OUT"
	helperRole = "S99_CRASH_TEST_ROLE"
)

// TestMain runs the helper roles: a watcher that records what it would show, and a watched process
// that panics, has a panic Wails recovered, exits before it is up, waits to be killed, exits cleanly or hangs.
func TestMain(m *testing.M) {
	out := os.Getenv(helperOut)
	switch {
	case out != "" && os.Getenv(watcherEnv) != "":
		hangAfter, checkEvery, killedGrace = time.Second, 50*time.Millisecond, 10*time.Millisecond
		pid, _ := strconv.Atoi(os.Getenv(watcherEnv))
		events := make(chan event, 256)
		go readPipe(os.Stdin, events)
		ctl := newControls()
		code := newWatched(Options{}, pid, ctl).loop(events, &fileScreen{path: out, ctl: ctl})
		record(out, "exit")
		os.Exit(code)
	case out != "":
		exe, _ := os.Executable()
		if err := spawn(exe, nil); err != nil {
			os.Exit(3)
		}
		Describe(Details{Lang: "pl", LogFile: "/tmp/app.log"})
		send(lineLog, "last words")
		switch os.Getenv(helperRole) {
		case "early":
			os.Exit(0)
		case "kill":
			send(lineUp, "")
		case "wails-panic":
			send(lineUp, "")
			PanicHandler(&application.PanicDetails{Error: errors.New("boom"), StackTrace: "goroutine 7 [running]:"})
		case "panic":
			time.Sleep(200 * time.Millisecond)
			go func() { panic("boom") }()
		case "bye":
			Done()
			os.Exit(0)
		case "hang":
			send(lineUp, "")
			send(lineBeat, sourceNative)
		}
		select {}
	}
	os.Exit(m.Run())
}

// fileScreen writes each step to a file and, like a user, asks to stop a process that hangs.
type fileScreen struct {
	path string
	ctl  controls
}

func (s *fileScreen) show(r Report) {
	data, _ := json.Marshal(r)
	record(s.path, "show "+string(data))
	if r.Alive {
		s.ctl.signal(s.ctl.kill)
	}
}

func (s *fileScreen) hide() { record(s.path, "hide") }
func (s *fileScreen) quit() { record(s.path, "quit") }

func record(path, line string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line + "\n")
}

// watchRole runs a watched process in role; act gets it once started. It returns the watcher's steps.
func watchRole(t *testing.T, role string, act func(*exec.Cmd)) []string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the helpers signal processes")
	}
	out := filepath.Join(t.TempDir(), "steps")
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), helperOut+"="+out, helperRole+"="+role)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	act(cmd)
	_ = cmd.Wait()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		data, _ := os.ReadFile(out)
		if strings.HasSuffix(string(data), "exit\n") {
			return strings.Split(strings.TrimSpace(string(data)), "\n")
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("the watcher did not finish")
	return nil
}

func shown(t *testing.T, steps []string, i int) Report {
	t.Helper()
	data, ok := strings.CutPrefix(steps[i], "show ")
	if !ok {
		t.Fatalf("step %d = %q, want a shown report; steps %q", i, steps[i], steps)
	}
	var r Report
	if err := json.Unmarshal([]byte(data), &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestWatcherShowsAGoroutinePanic(t *testing.T) {
	steps := watchRole(t, "panic", func(*exec.Cmd) {})
	r := shown(t, steps, 0)
	if r.Kind != KindCrash || r.Error != "panic: boom" || r.Lang != "pl" || !strings.Contains(r.Text, "last words") {
		t.Fatalf("report = %+v", r)
	}
}

func TestWatcherShowsAPanicWailsRecovered(t *testing.T) {
	steps := watchRole(t, "wails-panic", func(*exec.Cmd) {})
	r := shown(t, steps, 0)
	if r.Kind != KindCrash || r.Error != "panic: boom" || !strings.Contains(r.Text, "goroutine 7 [running]:") ||
		!strings.Contains(r.Text, "last words") {
		t.Fatalf("report = %+v", r)
	}
}

func TestWatcherShowsAKill(t *testing.T) {
	steps := watchRole(t, "kill", func(cmd *exec.Cmd) {
		time.Sleep(500 * time.Millisecond)
		_ = cmd.Process.Kill()
	})
	r := shown(t, steps, 0)
	if r.Kind != KindKilled || r.Alive || !strings.Contains(r.Text, "last words") {
		t.Fatalf("report = %+v", r)
	}
}

func TestWatcherStaysQuietAfterACleanExit(t *testing.T) {
	steps := watchRole(t, "bye", func(*exec.Cmd) {})
	if len(steps) != 2 || steps[0] != "quit" {
		t.Fatalf("steps = %q, want quit then exit", steps)
	}
}

func TestWatcherStaysQuietWhenTheAppExitsBeforeItIsUp(t *testing.T) {
	steps := watchRole(t, "early", func(*exec.Cmd) {})
	if len(steps) != 2 || steps[0] != "quit" || steps[1] != "exit" {
		t.Fatalf("steps = %q, want quit then exit", steps)
	}
}

func TestWatcherStopsAHangAndShowsItsStacks(t *testing.T) {
	steps := watchRole(t, "hang", func(*exec.Cmd) {})
	waiting, stopped := shown(t, steps, 0), shown(t, steps, 1)
	if waiting.Kind != KindHang || !waiting.Alive {
		t.Fatalf("first report = %+v, want a live hang", waiting)
	}
	if stopped.Kind != KindHang || stopped.Alive || !strings.Contains(stopped.Text, "goroutine") {
		t.Fatalf("second report = %+v, want the stopped hang with stacks", stopped)
	}
}

func TestReadPipeSplitsLinesOutOfTheTrace(t *testing.T) {
	in := "panic: x\ngorou" + frame(lineLog, "inside") + "tine 1\n" + frame(lineBye, "")
	events := make(chan event, 16)
	readPipe(strings.NewReader(in), events)
	close(events)
	var trace strings.Builder
	var kinds []string
	for e := range events {
		if e.kind == eventTrace {
			trace.WriteString(e.payload)
			continue
		}
		kinds = append(kinds, e.kind+":"+e.payload)
	}
	if trace.String() != "panic: x\ngoroutine 1\n" || strings.Join(kinds, ",") != "log:inside,bye:,end:" {
		t.Fatalf("trace %q, lines %q", trace.String(), kinds)
	}
}

// fakeScreen records calls for the loop tests.
type fakeScreen struct{ steps chan string }

func (s fakeScreen) show(r Report) {
	s.steps <- "show " + string(r.Kind) + " alive=" + strconv.FormatBool(r.Alive)
}
func (s fakeScreen) hide() { s.steps <- "hide" }
func (s fakeScreen) quit() { s.steps <- "quit" }

func next(t *testing.T, steps <-chan string) string {
	t.Helper()
	select {
	case step := <-steps:
		return step
	case <-time.After(5 * time.Second):
		t.Fatal("no step")
		return ""
	}
}

func TestLoopHidesAHangWhenBeatsReturnAndAfterWait(t *testing.T) {
	defer func(h, c time.Duration) { hangAfter, checkEvery = h, c }(hangAfter, checkEvery)
	hangAfter, checkEvery = 200*time.Millisecond, 20*time.Millisecond
	events := make(chan event)
	ui := fakeScreen{steps: make(chan string, 8)}
	ctl := newControls()
	w := newWatched(Options{}, 0, ctl)
	done := make(chan int)
	go func() { done <- w.loop(events, ui) }()

	events <- event{kind: lineBeat, payload: sourceNative}
	if step := next(t, ui.steps); step != "show hang alive=true" {
		t.Fatalf("step = %q", step)
	}
	events <- event{kind: lineBeat, payload: sourceNative}
	if step := next(t, ui.steps); step != "hide" {
		t.Fatalf("step = %q, want the screen hidden when beats return", step)
	}
	if step := next(t, ui.steps); step != "show hang alive=true" {
		t.Fatalf("step = %q, want the hang shown again", step)
	}
	ctl.signal(ctl.wait)
	if step := next(t, ui.steps); step != "hide" {
		t.Fatalf("step = %q, want Wait to hide the screen", step)
	}
	events <- event{kind: lineRest, payload: sourceNative}
	events <- event{kind: lineBye}
	events <- event{kind: eventEnd}
	if step := next(t, ui.steps); step != "quit" {
		t.Fatalf("step = %q", step)
	}
	if code := <-done; code != 0 {
		t.Fatalf("code = %d", code)
	}
}

func TestFrameIsOneShortLine(t *testing.T) {
	line := frame(lineLog, strings.Repeat("x", 2*maxLine)+"\nsecond")
	if len(line) != maxLine || strings.Count(line, "\n") != 1 || !strings.HasPrefix(line, marker+"log ") {
		t.Fatalf("frame is %d bytes with %d newlines", len(line), strings.Count(line, "\n"))
	}
}
