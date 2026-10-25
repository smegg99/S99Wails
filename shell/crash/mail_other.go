// shell/crash/mail_other.go

//go:build !linux && !windows

package crash

func sendMail(mail) error {
	return errNoAttach
}
