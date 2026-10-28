// shell/crash/panic.go

package crash

import (
	"os"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// PanicHandler, as application.Options.PanicHandler, reports a panic Wails recovered as a crash.
func PanicHandler(d *application.PanicDetails) {
	text := "panic: " + d.Error.Error() + "\n\n" + d.StackTrace + "\n"
	p := out.Swap(nil)
	if p == nil {
		_, _ = os.Stderr.WriteString(text)
		os.Exit(2)
	}
	drain(p, time.Second)
	p.write(text)
	os.Exit(2)
}
