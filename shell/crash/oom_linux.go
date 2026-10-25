// shell/crash/oom_linux.go

package crash

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// oomKills is the kernel's count of processes it killed for memory; a rise during a run names the cause.
func oomKills() int64 {
	f, err := os.Open("/proc/vmstat")
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
