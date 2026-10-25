// shell/crash/mail_linux_test.go

package crash

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeMailer puts xdg-mime, answering handler, and an xdg-email that records its arguments on PATH.
func fakeMailer(t *testing.T, handler string) (args string) {
	t.Helper()
	bin := t.TempDir()
	args = filepath.Join(bin, "args")
	scripts := map[string]string{
		"xdg-mime":  "#!/bin/sh\necho " + handler + "\n",
		"xdg-email": "#!/bin/sh\nfor a in \"$@\"; do printf '%s\\n' \"$a\"; done > " + args + "\n",
	}
	for name, body := range scripts {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	return args
}

func TestEmailAttachesTheReportAndTheLogFile(t *testing.T) {
	args := fakeMailer(t, "org.mozilla.Thunderbird.desktop")
	log := filepath.Join(t.TempDir(), "app.log")
	if err := os.WriteFile(log, []byte("logged\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := Report{App: "Tuner", Version: "1.4.0", Kind: KindCrash, Error: "panic: boom", Text: "the whole report", LogFile: log}
	if err := (&Service{email: "me@example.com"}).attach(r); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(args)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Split(strings.TrimSpace(string(data)), "\n")
	attached := []string{}
	for i, arg := range got {
		if arg == "--attach" {
			attached = append(attached, got[i+1])
		}
	}
	if len(attached) != 2 || attached[1] != log || got[len(got)-1] != "me@example.com" {
		t.Fatalf("xdg-email got %q", got)
	}
	if report, err := os.ReadFile(attached[0]); err != nil || string(report) != "the whole report" {
		t.Fatalf("attached report %q: %v", report, err)
	}
	if !strings.Contains(string(data), "Tuner 1.4.0 crash report\n") || !strings.Contains(string(data), "The full report and the log file are attached.") {
		t.Fatalf("subject or body missing: %q", data)
	}
}

func TestEmailLeavesAClientWithoutAttachmentsToMailto(t *testing.T) {
	fakeMailer(t, "firefox.desktop")
	err := (&Service{email: "me@example.com"}).attach(Report{App: "Tuner", Kind: KindCrash, Text: "r"})
	if !errors.Is(err, errNoAttach) {
		t.Fatalf("err = %v, want errNoAttach", err)
	}
}

func TestWriteReportDropsOldReports(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	now := time.Now()
	first, err := writeReport(Report{App: "Tuner", Kind: KindCrash, Text: "old"}, now)
	if err != nil {
		t.Fatal(err)
	}
	old := now.Add(-keepReports - time.Hour)
	if err := os.Chtimes(first, old, old); err != nil {
		t.Fatal(err)
	}
	if _, err := writeReport(Report{App: "Tuner", Kind: KindHang, Text: "new"}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(first); !os.IsNotExist(err) {
		t.Fatalf("old report kept: %v", err)
	}
}
