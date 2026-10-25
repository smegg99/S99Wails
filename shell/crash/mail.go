// shell/crash/mail.go

package crash

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// keepReports is how long a report written for an email stays in the cache.
	keepReports = 14 * 24 * time.Hour
	// handOff is how long a mail client gets to refuse a message before it counts as taken.
	handOff = 10 * time.Second
)

// errNoAttach means this platform or mail client cannot take attachments.
var errNoAttach = errors.New("mail client cannot take attachments")

// mail is one message for the user's mail client.
type mail struct {
	to, subject, body string
	files             []string
}

// attach writes the report to the cache and hands it and the log file to the mail client.
func (s *Service) attach(r Report) error {
	path, err := writeReport(r, time.Now())
	if err != nil {
		return err
	}
	files := []string{path}
	if exists(r.LogFile) {
		files = append(files, r.LogFile)
	}
	return sendMail(mail{to: s.email, subject: subject(r), body: brief(r, len(files) > 1), files: files})
}

func subject(r Report) string {
	return fmt.Sprintf("%s %s %s report", r.App, r.Version, r.Kind)
}

// brief is the body beside the attachments: what happened, and what is attached.
func brief(r Report, withLog bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s, %s report.\n", r.App, r.Version, r.Kind)
	if r.Error != "" {
		fmt.Fprintf(&b, "\n%s\n", r.Error)
	}
	if withLog {
		b.WriteString("\nThe full report and the log file are attached.\n")
	} else {
		b.WriteString("\nThe full report is attached.\n")
	}
	return b.String()
}

func reportName(r Report, now time.Time) string {
	return fmt.Sprintf("%s-%s-%s.txt", slug(r.App), r.Kind, now.Format("20060102-150405"))
}

// writeReport saves the report under the user cache for a mail client to attach, and drops old ones.
func writeReport(r Report, now time.Time) (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("find cache dir: %w", err)
	}
	dir := filepath.Join(cache, slug(r.App), "reports")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create report dir: %w", err)
	}
	if old, err := os.ReadDir(dir); err == nil {
		for _, entry := range old {
			if info, err := entry.Info(); err == nil && now.Sub(info.ModTime()) > keepReports {
				_ = os.Remove(filepath.Join(dir, entry.Name()))
			}
		}
	}
	path := filepath.Join(dir, reportName(r, now))
	if err := os.WriteFile(path, []byte(r.Text), 0o644); err != nil {
		return "", fmt.Errorf("write report: %w", err)
	}
	return path, nil
}

// within runs send and returns its error if it fails within handOff; a client still open after that took the message.
func within(send func() error) error {
	done := make(chan error, 1)
	go func() { done <- send() }()
	select {
	case err := <-done:
		return err
	case <-time.After(handOff):
		return nil
	}
}
