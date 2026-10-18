// shell/editkeys/editkeys_gtk3.go

//go:build linux && cgo && gtk3

package editkeys

/*
#cgo pkg-config: gtk+-3.0
#include <gtk/gtk.h>
#include <stdint.h>

extern gboolean editkeysPress(GtkWidget*, GdkEventKey*, gpointer);
extern void editkeysDestroyed(GtkWidget*, gpointer);

// The toplevel hears a key before it passes it on to the webview inside it.
static gboolean editkeysWatch(void* window, uintptr_t id) {
	if (window == NULL || !GTK_IS_WINDOW(window)) return FALSE;
	g_signal_connect(window, "key-press-event", G_CALLBACK(editkeysPress), (gpointer)id);
	g_signal_connect(window, "destroy", G_CALLBACK(editkeysDestroyed), (gpointer)id);
	return TRUE;
}

static guint editkeysLower(GdkEventKey* event) { return gdk_keyval_to_lower(event->keyval); }
static gboolean editkeysHas(GdkEventKey* event, guint mask) { return (event->state & mask) != 0; }

// A clipboard nothing owns offers no target at all.
static gboolean editkeysClipboardEmpty(void) {
	GtkClipboard* clipboard = gtk_clipboard_get(GDK_SELECTION_CLIPBOARD);
	GdkAtom* targets = NULL;
	gint count = 0;
	if (!gtk_clipboard_wait_for_targets(clipboard, &targets, &count)) return TRUE;
	g_free(targets);
	return count == 0;
}
*/
import "C"

import (
	"errors"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// ErrNoWindow reports a window with no GTK toplevel to watch.
var ErrNoWindow = errors.New("editkeys: no GTK window")

// Each callback belongs to its native window. GTK destruction, rather than
// a cancellable close request, removes it.
var windows sync.Map // uint -> *application.WebviewWindow

// Watch sends the window's ctrl+Z and ctrl+shift+Z as Event, and a paste
// key while the clipboard holds nothing, from the first time it is shown.
func Watch(win *application.WebviewWindow, onError func(error)) {
	windows.Store(win.ID(), win)
	var once sync.Once
	win.OnWindowEvent(events.Common.WindowShow, func(*application.WindowEvent) {
		once.Do(func() {
			application.InvokeSync(func() {
				if C.editkeysWatch(win.NativeWindow(), C.uintptr_t(win.ID())) == 0 {
					onError(ErrNoWindow)
				}
			})
		})
	})
}

// cgo writes a C prototype from these names, so none of them may be blank.
//
//export editkeysPress
func editkeysPress(widget *C.GtkWidget, event *C.GdkEventKey, data C.gpointer) C.gboolean {
	other := C.guint(C.GDK_MOD1_MASK | C.GDK_SUPER_MASK | C.GDK_META_MASK)
	key, control, shift := uint(C.editkeysLower(event)), C.editkeysHas(event, C.GDK_CONTROL_MASK) != 0, C.editkeysHas(event, C.GDK_SHIFT_MASK) != 0
	held := C.editkeysHas(event, other) != 0
	action := actionOf(key, control, shift, held)
	if pastes(key, control, shift, held) {
		if C.editkeysClipboardEmpty() == 0 {
			return C.FALSE
		}
		action = "paste"
	}
	if action == "" {
		return C.FALSE
	}
	if win, ok := windows.Load(uint(uintptr(data))); ok {
		win.(*application.WebviewWindow).DispatchWailsEvent(&application.CustomEvent{Name: Event, Data: action})
	}
	return C.TRUE
}

//export editkeysDestroyed
func editkeysDestroyed(widget *C.GtkWidget, data C.gpointer) {
	windows.Delete(uint(uintptr(data)))
}
