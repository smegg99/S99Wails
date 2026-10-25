// shell/crash/pipe.go

package crash

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// marker starts every line this package writes; everything else on the pipe is the runtime's crash output.
	marker = "\x00s99crash "
	// maxLine stays under PIPE_BUF, so a line is written at once and never splits the runtime's output.
	maxLine = 3500
	// queued lines wait for the writer; past this the newest are dropped instead of blocking the app.
	queued = 4096
)

const (
	lineDetails = "details"
	lineLog     = "log"
	lineBeat    = "beat"
	lineRest    = "rest"
	lineBye     = "bye"
)

// pipe writes framed lines to the watcher from one goroutine, so no caller ever blocks on it.
type pipe struct {
	w       *os.File
	lines   chan string
	done    chan struct{}
	dropped atomic.Int64

	mu     sync.RWMutex
	closed bool
}

var out atomic.Pointer[pipe]

func newPipe(w *os.File) *pipe {
	p := &pipe{w: w, lines: make(chan string, queued), done: make(chan struct{})}
	go p.run()
	return p
}

func (p *pipe) run() {
	defer close(p.done)
	for line := range p.lines {
		if n := p.dropped.Swap(0); n > 0 {
			p.write(frame(lineLog, fmt.Sprintf("[%d log lines dropped]", n)))
		}
		p.write(line)
	}
}

func (p *pipe) write(line string) {
	_, _ = p.w.WriteString(line)
}

func frame(kind, payload string) string {
	line := marker + kind
	if payload != "" {
		line += " " + strings.ReplaceAll(payload, "\n", " ")
	}
	if len(line) >= maxLine {
		line = line[:maxLine-1]
	}
	return line + "\n"
}

// send queues one line for the watcher; without Watch it does nothing.
func send(kind, payload string) {
	p := out.Load()
	if p == nil {
		return
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return
	}
	select {
	case p.lines <- frame(kind, payload):
	default:
		p.dropped.Add(1)
	}
}

// Done tells the watcher this is a clean exit. Call it last, after the logger is closed.
func Done() {
	p := out.Swap(nil)
	if p == nil {
		return
	}
	p.mu.Lock()
	p.closed = true
	close(p.lines)
	p.mu.Unlock()
	select {
	case <-p.done:
	case <-time.After(2 * time.Second):
	}
	p.write(frame(lineBye, ""))
}
