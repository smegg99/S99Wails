// shell/crash/service_test.go

package crash

import (
	"errors"
	"os"
	"testing"
)

func TestServiceRefusesCloseAndRestartWhileTheAppRuns(t *testing.T) {
	s := newService(Options{}, Report{Alive: true}, newControls())
	if err := s.Close(); !errors.Is(err, errAlive) {
		t.Fatalf("Close() = %v, want errAlive", err)
	}
	if err := s.Restart(); !errors.Is(err, errAlive) {
		t.Fatalf("Restart() = %v, want errAlive", err)
	}
}

func TestBecomeWatcherUnsetsTheVariable(t *testing.T) {
	t.Setenv(watcherEnv, "42")
	pid, ok := becomeWatcher()
	if pid != 42 || !ok || os.Getenv(watcherEnv) != "" {
		t.Fatalf("becomeWatcher() = %d, %v; variable %q", pid, ok, os.Getenv(watcherEnv))
	}
}
