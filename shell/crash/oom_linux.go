// shell/crash/oom_linux.go

package crash

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// oomKills counts the OOM kills in this process's cgroup, which the watcher shares with the app.
func oomKills() int64 {
	return cgroupOOMKills("/proc/self/cgroup", "/sys/fs/cgroup")
}

// cgroupOOMKills reads oom_kill from memory.events of the unified cgroup named in procCgroup.
func cgroupOOMKills(procCgroup, root string) int64 {
	data, err := os.ReadFile(procCgroup)
	if err != nil {
		return 0
	}
	path, ok := "", false
	for line := range strings.Lines(string(data)) {
		if path, ok = strings.CutPrefix(strings.TrimSpace(line), "0::"); ok {
			break
		}
	}
	if !ok {
		return 0
	}
	f, err := os.Open(filepath.Join(root, filepath.FromSlash(path), "memory.events"))
	if err != nil {
		return 0
	}
	defer f.Close()
	lines := bufio.NewScanner(f)
	for lines.Scan() {
		if value, ok := strings.CutPrefix(lines.Text(), "oom_kill "); ok {
			n, _ := strconv.ParseInt(value, 10, 64)
			return n
		}
	}
	return 0
}
