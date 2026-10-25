// shell/crash/oom_other.go

//go:build !linux

package crash

// oomKills has no source outside Linux, so a kill is never named as out of memory.
func oomKills() int64 {
	return 0
}
