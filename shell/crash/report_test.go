// shell/crash/report_test.go

package crash

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestReportCarriesTheFailureFieldsAndRecentLog(t *testing.T) {
	log := filepath.Join(t.TempDir(), "app.log")
	if err := os.WriteFile(log, []byte("on disk\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lines := []string{"line 1", "line 2"}
	opts := Options{App: application.Options{Name: "Tuner"}, Version: "1.4.0", ReportEmail: "a@b.c"}
	f := Failure{Kind: KindConfig, File: "/home/u/config.json", Err: "bad key tuner.x\nsecond line"}
	d := Details{Lang: "pl", LogFile: log, Fields: []Field{{Name: "Render path", Value: "native"}}}
	r := build(opts, f, d, lines, time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC))

	if r.Error != "bad key tuner.x\nsecond line" || r.File != f.File || r.Lang != "pl" || !r.Email || !r.Logs {
		t.Fatalf("report = %+v", r)
	}
	for _, want := range []string{"Tuner 1.4.0\n", "Kind: config\n", "Time: 2026-10-28T12:00:00Z\n", "File: /home/u/config.json\n", "Render path: native\n", "Log file: " + log + "\n", "second line", "Recent log, 2 lines:\nline 1\nline 2\n"} {
		if !strings.Contains(r.Text, want) {
			t.Errorf("text lacks %q:\n%s", want, r.Text)
		}
	}
}

func TestACrashShowsOnlyTheTracesFirstLine(t *testing.T) {
	r := build(Options{}, Failure{Kind: KindCrash, Err: "panic: boom\n\ngoroutine 7 [running]:"}, Details{}, nil, time.Now())
	if r.Error != "panic: boom" || !strings.Contains(r.Text, "goroutine 7 [running]:") {
		t.Fatalf("report = %+v", r)
	}
}

func TestReportWithoutAddressOrLogOffersNeither(t *testing.T) {
	r := build(Options{}, Failure{Kind: KindStartup, Err: "x"}, Details{LogFile: "/nonexistent/app.log"}, nil, time.Now())
	if r.Email || r.Logs || strings.Contains(r.Text, "Recent log") {
		t.Fatalf("report = %+v", r)
	}
}

func TestAHangOrKillHasNoErrorLine(t *testing.T) {
	for _, kind := range []Kind{KindHang, KindKilled} {
		if r := build(Options{}, Failure{Kind: kind, Err: "SIGQUIT: quit\ngoroutine 1"}, Details{}, nil, time.Now()); r.Error != "" {
			t.Fatalf("%s error = %q", kind, r.Error)
		}
	}
}

func TestRingKeepsTheNewestLinesInOrder(t *testing.T) {
	r := newRing(3)
	for _, line := range []string{"a", "b", "c", "d", "e"} {
		r.add(line)
	}
	if got := strings.Join(r.snapshot(), ","); got != "c,d,e" {
		t.Fatalf("snapshot = %s", got)
	}
}

func TestMailtoEscapesAndCutsALongReport(t *testing.T) {
	r := Report{App: "Tuner", Version: "1.4.0", Kind: KindCrash, Text: "a & b = c?\n" + strings.Repeat("x\n", mailBytes)}
	link := mailto("me@example.com", r)
	u, err := url.Parse(link)
	if err != nil || u.Scheme != "mailto" || u.Opaque != "me@example.com" {
		t.Fatalf("link %q: %v", link[:60], err)
	}
	if strings.Contains(link, "+") {
		t.Fatal("mailto must encode spaces as %20")
	}
	q := u.Query()
	if q.Get("subject") != "Tuner 1.4.0 crash report" {
		t.Fatalf("subject = %q", q.Get("subject"))
	}
	body := q.Get("body")
	if !strings.HasPrefix(body, "a & b = c?\n") || !strings.HasSuffix(body, "[cut, the saved report has the full text]\n") || len(body) > mailBytes+64 {
		t.Fatalf("body has %d bytes, starts %q", len(body), body[:20])
	}
}
