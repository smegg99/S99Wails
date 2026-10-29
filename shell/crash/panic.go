// shell/crash/panic.go

package crash

import (
	"os"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ending orders PanicHandler and Done, which both end the pipe.
var ending sync.Mutex

// PanicHandler, as application.Options.PanicHandler, reports a panic Wails recovered as a crash.
func PanicHandler(d *application.PanicDetails) {
	// Never unlocked: a second panic, or Done, waits for the exit instead of racing the first trace.
	ending.Lock()
	text := "panic: " + d.Error.Error() + "\n\n" + d.StackTrace + "\n"
	p := out.Swap(nil)
	if p == nil {
		_, _ = os.Stderr.WriteString(text)
		os.Exit(2)
	}
	finish(p, text, time.Second)
	os.Exit(2)
}
