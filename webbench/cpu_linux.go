//go:build linux

// webbench/cpu_linux.go

package webbench

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// sampleCPU returns a stop function giving each named process's average CPU.
// It matches by name because WebKit spawns its processes after load starts.
func sampleCPU(names []string) func() map[string]float64 {
	type mark struct {
		firstTicks, lastTicks uint64
		firstAt, lastAt       time.Time
	}
	marks := map[string]*mark{}
	stop := make(chan struct{})
	done := make(chan map[string]float64, 1)

	go func() {
		tick := time.NewTicker(400 * time.Millisecond)
		defer tick.Stop()
		for {
			now := time.Now()
			for name, ticks := range procTicks(names) {
				m := marks[name]
				if m == nil {
					m = &mark{firstTicks: ticks, firstAt: now}
					marks[name] = m
				}
				m.lastTicks, m.lastAt = ticks, now
			}
			select {
			case <-stop:
				hz := float64(clockTicks)
				out := map[string]float64{}
				for name, m := range marks {
					span := m.lastAt.Sub(m.firstAt).Seconds()
					if span < 0.5 {
						continue
					}
					out[name] = round1(float64(m.lastTicks-m.firstTicks) / hz / span * 100)
				}
				done <- out
				return
			case <-tick.C:
			}
		}
	}()

	return func() map[string]float64 {
		close(stop)
		select {
		case out := <-done:
			return out
		case <-time.After(2 * time.Second):
			return nil
		}
	}
}

// clockTicks is USER_HZ, 100 on supported Linux targets; sysconf would need cgo.
const clockTicks = 100

// procTicks sums utime+stime per process name across all matching processes.
func procTicks(names []string) map[string]uint64 {
	// Match the truncated comm name and report the full name.
	want := map[string]string{}
	for _, n := range names {
		want[comm(n)] = n
	}
	out := map[string]uint64{}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return out
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(e.Name()); err != nil {
			continue
		}
		raw, err := os.ReadFile(filepath.Join("/proc", e.Name(), "comm"))
		if err != nil {
			continue
		}
		full, ok := want[strings.TrimSpace(string(raw))]
		if !ok {
			continue
		}
		stat, err := os.ReadFile(filepath.Join("/proc", e.Name(), "stat"))
		if err != nil {
			continue
		}
		// The comm field is parenthesised and may contain spaces, so fields are
		// counted from after the closing paren: utime is 14, stime is 15.
		close := strings.LastIndex(string(stat), ")")
		if close < 0 {
			continue
		}
		fields := strings.Fields(string(stat)[close+1:])
		if len(fields) < 13 {
			continue
		}
		utime, _ := strconv.ParseUint(fields[11], 10, 64)
		stime, _ := strconv.ParseUint(fields[12], 10, 64)
		out[full] += utime + stime
	}
	return out
}

// comm truncates name to 15 characters, as the kernel does (TASK_COMM_LEN 16).
func comm(name string) string {
	if len(name) > 15 {
		return name[:15]
	}
	return name
}

func round1(f float64) float64 { return float64(int(f*10+0.5)) / 10 }
