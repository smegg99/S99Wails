// shell/crash/mail_linux.go

package crash

import (
	"fmt"
	"os/exec"
	"regexp"
)

// attachers are the default mail clients xdg-email passes attachments to; any other gets a bare mailto.
var attachers = regexp.MustCompile(`(?i)thunderbird|icedove|betterbird|kmail|evolution`)

func sendMail(m mail) error {
	xdg, err := exec.LookPath("xdg-email")
	if err != nil {
		return errNoAttach
	}
	handler, err := exec.Command("xdg-mime", "query", "default", "x-scheme-handler/mailto").Output()
	if err != nil || !attachers.Match(handler) {
		return errNoAttach
	}
	args := []string{"--utf8", "--subject", m.subject, "--body", m.body}
	for _, file := range m.files {
		args = append(args, "--attach", file)
	}
	return within(func() error {
		if out, err := exec.Command(xdg, append(args, m.to)...).CombinedOutput(); err != nil {
			return fmt.Errorf("xdg-email: %w: %s", err, out)
		}
		return nil
	})
}
