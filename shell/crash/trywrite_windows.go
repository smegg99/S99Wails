// shell/crash/trywrite_windows.go

package crash

import "os"

// tryWrite has no non-blocking pipe write on Windows, so every line goes through the queue.
func tryWrite(*os.File, string) bool { return false }
