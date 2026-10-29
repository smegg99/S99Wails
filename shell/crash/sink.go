// shell/crash/sink.go

package crash

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/smegg99/s99logger"
)

// ringLines is how much recent log a report carries.
const ringLines = 1000

// local is this process's recent log, for Show; the watcher keeps its own copy from the pipe.
var local = newRing(ringLines)

// Sink keeps the report's recent log lines and streams them to the watcher without blocking; give it every level the report needs.
func Sink() s99logger.Sink {
	return sink{}
}

type sink struct{}

func (sink) Write(_ context.Context, rec s99logger.Record) error {
	line := format(rec)
	local.add(line)
	send(lineLog, line)
	return nil
}

func format(rec s99logger.Record) string {
	var b strings.Builder
	b.WriteString(rec.Time.Format("15:04:05.000"))
	b.WriteByte(' ')
	b.WriteString(strings.ToUpper(rec.Level.String()))
	b.WriteByte(' ')
	if rec.Message != "" {
		b.WriteString(rec.Message)
	} else {
		b.WriteString(string(rec.MessageID))
	}
	for _, attr := range rec.Attrs {
		fmt.Fprintf(&b, " %s=%v", attr.Key, attr.Value)
	}
	return strings.ReplaceAll(b.String(), "\n", " ")
}

// ring keeps the last size lines.
type ring struct {
	mu    sync.Mutex
	lines []string
	next  int
	full  bool
}

func newRing(size int) *ring {
	return &ring{lines: make([]string, size)}
}

func (r *ring) add(line string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lines[r.next] = line
	r.next = (r.next + 1) % len(r.lines)
	r.full = r.full || r.next == 0
}

// snapshot is the kept lines, oldest first.
func (r *ring) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.full {
		return append([]string(nil), r.lines[:r.next]...)
	}
	return append(append([]string(nil), r.lines[r.next:]...), r.lines[:r.next]...)
}
