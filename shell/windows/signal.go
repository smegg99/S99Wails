// shell/windows/signal.go

package windows

import (
	"os"
	"os/signal"
	"syscall"
	"time"
)

// QuitOnSignal quits on SIGINT or SIGTERM as the tray does, exiting after grace or on a second signal; set DisableDefaultSignalHandler.
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
