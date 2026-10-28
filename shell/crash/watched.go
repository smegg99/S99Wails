// shell/crash/watched.go

package crash

import (
	"encoding/json"
	"strings"
	"time"
)

var (
	// hangAfter is how long a beating source may stay silent before the process counts as not responding.
	hangAfter = 15 * time.Second
	// stopAfter is how long a stopped process has to write its stacks before it is killed outright.
	stopAfter = 5 * time.Second
	// killedGrace lets a logout's SIGTERM reach the watcher before it reports a kill.
	killedGrace = time.Second
	checkEvery  = time.Second
)

// screen is what the watcher shows; the Wails one runs the app's frontend.
type screen interface {
	show(r Report)
	hide()
	quit()
}

// controls carry the screen's Wait and Kill to the watcher loop.
type controls struct {
	wait chan struct{}
	kill chan struct{}
}

func newControls() controls {
	return controls{wait: make(chan struct{}, 1), kill: make(chan struct{}, 1)}
}

func (controls) signal(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

// watched is the watcher's view of the process: what it described, its recent log, its beats and its end.
type watched struct {
	opts    Options
	pid     int
	ctl     controls
	details Details
	log     *ring
	trace   strings.Builder
	beats   map[string]time.Time
	bye     bool
	// up is set once the app is running; an exit before that (a second instance) is never a kill.
	up bool
	// quitting is set once the app starts to quit; its cleanup may block every beat, the page's too.
	quitting bool
	oom      int64

	hung       bool
	stopping   bool
	quietUntil time.Time
}

func newWatched(opts Options, pid int, ctl controls) *watched {
	return &watched{opts: opts, pid: pid, ctl: ctl, log: newRing(ringLines), beats: map[string]time.Time{}, oom: oomKills()}
}

// loop follows the process until the pipe ends and returns the watcher's exit code.
func (w *watched) loop(events <-chan event, ui screen) int {
	check := time.NewTicker(checkEvery)
	defer check.Stop()
	var force <-chan time.Time
	for {
		select {
		case e := <-events:
			if e.kind == eventEnd {
				return w.end(ui)
			}
			w.take(e, ui)
		case <-check.C:
			if !w.hung && time.Now().After(w.quietUntil) && w.stalled() {
				w.hung = true
				ui.show(w.report(KindHang, "", true))
			}
		case <-w.ctl.wait:
			if w.hung && !w.stopping {
				w.hung = false
				w.quietUntil = time.Now().Add(hangAfter)
				ui.hide()
			}
		case <-w.ctl.kill:
			if w.hung && !w.stopping {
				w.stopping = true
				stop(w.pid)
				force = time.After(stopAfter)
			}
		case <-force:
			kill(w.pid)
		}
	}
}

func (w *watched) take(e event, ui screen) {
	switch e.kind {
	case lineDetails:
		var d Details
		if json.Unmarshal([]byte(e.payload), &d) == nil {
			w.details = d
		}
	case lineLog:
		w.log.add(e.payload)
	case lineBeat:
		if w.quitting {
			return
		}
		w.beats[e.payload] = time.Now()
		if w.hung && !w.stopping && !w.stalled() {
			w.hung = false
			ui.hide()
		}
	case lineRest:
		delete(w.beats, e.payload)
	case lineBye:
		w.bye = true
	case lineUp:
		w.up = true
	case lineQuit:
		w.quitting = true
		clear(w.beats)
	case eventTrace:
		w.trace.WriteString(e.payload)
	}
}

// stalled is whether any source that has beaten went silent for hangAfter.
func (w *watched) stalled() bool {
	for _, last := range w.beats {
		if time.Since(last) > hangAfter {
			return true
		}
	}
	return false
}

// end decides what the end of the pipe means: a stopped hang, a crash, a clean or early exit, or a kill.
func (w *watched) end(ui screen) int {
	trace := strings.TrimSpace(w.trace.String())
	switch {
	case w.stopping:
		ui.show(w.report(KindHang, trace, false))
	case trace != "":
		ui.show(w.report(KindCrash, trace, false))
	case w.bye || !w.up:
		ui.quit()
	default:
		time.Sleep(killedGrace)
		r := w.report(KindKilled, "", false)
		r.OOM = oomKills() > w.oom
		ui.show(r)
	}
	return 0
}

func (w *watched) report(kind Kind, trace string, alive bool) Report {
	r := build(w.opts, Failure{Kind: kind, Err: trace}, w.details, w.log.snapshot(), time.Now())
	r.Alive = alive
	return r
}
