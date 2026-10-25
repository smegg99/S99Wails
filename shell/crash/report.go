// shell/crash/report.go

package crash

import (
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	// mailBytes keeps a mailto URL inside what mail clients and URL handlers accept.
	mailBytes = 6000
)

// Report is what the screen shows; Text is the plain-text report that Copy, Save and Email send.
type Report struct {
	Kind    Kind   `json:"kind"`
	App     string `json:"app"`
	Version string `json:"version"`
	Lang    string `json:"lang"`
	File    string `json:"file"`
	LogFile string `json:"logFile"`
	// Error is the failure for the screen: the whole error, or a crash trace's first line.
	Error string `json:"error"`
	Text  string `json:"text"`
	// Email and Logs say whether those actions can run.
	Email bool `json:"email"`
	Logs  bool `json:"logs"`
	// Alive is set while a hung process still runs, so the screen offers Wait and Kill.
	Alive bool `json:"alive"`
	// OOM says the kernel killed the process for memory.
	OOM bool `json:"oom"`
}

// build is the report for f; log is the recent log, oldest first.
func build(opts Options, f Failure, d Details, log []string, now time.Time) Report {
	r := Report{
		Kind: f.Kind, App: opts.App.Name, Version: opts.Version, Lang: d.Lang, File: f.File, LogFile: d.LogFile,
		Error: summary(f), Email: opts.ReportEmail != "", Logs: exists(d.LogFile),
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", r.App, r.Version)
	fmt.Fprintf(&b, "Kind: %s\n", f.Kind)
	fmt.Fprintf(&b, "Time: %s\n", now.Format(time.RFC3339))
	fmt.Fprintf(&b, "System: %s/%s, %s\n", runtime.GOOS, runtime.GOARCH, runtime.Version())
	if session := session(); session != "" {
		fmt.Fprintf(&b, "Session: %s\n", session)
	}
	if f.File != "" {
		fmt.Fprintf(&b, "File: %s\n", f.File)
	}
	for _, field := range d.Fields {
		fmt.Fprintf(&b, "%s: %s\n", field.Name, field.Value)
	}
	if d.LogFile != "" {
		fmt.Fprintf(&b, "Log file: %s\n", d.LogFile)
	}
	if err := strings.TrimSpace(f.Err); err != "" {
		fmt.Fprintf(&b, "\n%s\n", err)
	}
	if len(log) > 0 {
		fmt.Fprintf(&b, "\nRecent log, %d lines:\n%s\n", len(log), strings.Join(log, "\n"))
	}
	r.Text = b.String()
	return r
}

// mailto addresses the report to to; a long report is cut at a line, since Save keeps the whole text.
func mailto(to string, r Report) string {
	body := r.Text
	if len(body) > mailBytes {
		cut := strings.LastIndexByte(body[:mailBytes], '\n')
		if cut < 0 {
			cut = mailBytes
		}
		body = body[:cut] + "\n[cut, the saved report has the full text]\n"
	}
	return "mailto:" + to + "?subject=" + escape(subject(r)) + "&body=" + escape(body)
}

// escape is query escaping with %20 for spaces, which mailto needs and QueryEscape does not give.
func escape(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

// summary is what the screen shows as the error: a hang or a kill has none, a crash its first line.
func summary(f Failure) string {
	switch f.Kind {
	case KindCrash:
		return firstLine(f.Err)
	case KindHang, KindKilled:
		return ""
	}
	return strings.TrimSpace(f.Err)
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

func session() string {
	var parts []string
	for _, key := range []string{"XDG_SESSION_TYPE", "XDG_CURRENT_DESKTOP"} {
		if v := os.Getenv(key); v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, ", ")
}

func exists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
