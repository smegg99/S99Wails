// shell/crash/oom_linux_test.go

package crash

import (
	"os"
	"path/filepath"
	"testing"
)

// The count comes from memory.events of the cgroup this process is in, not from the whole system.
func TestOOMKillsReadTheOwnCgroup(t *testing.T) {
	dir := t.TempDir()
	self := filepath.Join(dir, "cgroup")
	root := filepath.Join(dir, "sys")
	scope := filepath.Join(root, "user.slice", "app.scope")
	if err := os.MkdirAll(scope, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, text string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(self, "1:net_cls:/\n0::/user.slice/app.scope\n")
	write(filepath.Join(root, "memory.events"), "oom 9\noom_kill 9\n")
	write(filepath.Join(scope, "memory.events"), "low 0\noom 1\noom_kill 2\noom_group_kill 0\n")
	if got := cgroupOOMKills(self, root); got != 2 {
		t.Fatalf("oom kills = %d, want the scope's 2", got)
	}
	write(self, "4:memory:/user.slice\n")
	if got := cgroupOOMKills(self, root); got != 0 {
		t.Fatalf("oom kills = %d without a unified cgroup, want 0", got)
	}
}
