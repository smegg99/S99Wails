package windows

import (
	"os"
	"os/signal"
	"syscall"
	"time"
)

// QuitOnSignal replaces Wails' handler (set DisableDefaultSignalHandler): SIGINT or SIGTERM quits
// as the tray's Quit does, and the process exits once grace passes or on a second signal.
func QuitOnSignal(s *Service, grace time.Duration) {
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go s.quitOnSignals(signals, grace, os.Exit)
}

func (s *Service) quitOnSignals(signals <-chan os.Signal, grace time.Duration, exit func(int)) {
	<-signals
	go s.quit()
	timer := time.NewTimer(grace)
	defer timer.Stop()
	select {
	case <-signals:
	case <-timer.C:
	}
	exit(1)
}
