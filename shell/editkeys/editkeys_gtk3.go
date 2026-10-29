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

// GTK passes a failed TARGETS reply as (NULL, -1), which WebKitGTK reads as SIZE_MAX atoms.
// It becomes an empty list before any handler reads it; GTK never frees the list's one byte.
static gboolean editkeysTargets(GSignalInvocationHint* hint, guint count, const GValue* values, gpointer data) {
	GtkSelectionData* selection = count < 2 ? NULL : g_value_get_boxed(&values[1]);
	if (selection != NULL && gtk_selection_data_get_length(selection) < 0 &&
	    gtk_selection_data_get_target(selection) == gdk_atom_intern_static_string("TARGETS"))
		gtk_selection_data_set(selection, GDK_SELECTION_TYPE_ATOM, 32, NULL, 0);
	return TRUE;
}

static void editkeysHookTargets(void) {
	g_signal_add_emission_hook(g_signal_lookup("selection-received", GTK_TYPE_WIDGET), 0, editkeysTargets, NULL, NULL);
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

// Each callback belongs to its native window and is removed on GTK destruction, not on a cancellable close.
var windows sync.Map // uint -> *application.WebviewWindow

// The targets hook is the whole process's, added with the first window shown.
var hookTargets sync.Once

// Watch sends ctrl+Z and ctrl+shift+Z as Event and hooks failed TARGETS replies; a nil onError drops failures.
func Watch(win *application.WebviewWindow, onError func(error)) {
	windows.Store(win.ID(), win)
	var once sync.Once
	win.OnWindowEvent(events.Common.WindowShow, func(*application.WindowEvent) {
		once.Do(func() {
			application.InvokeSync(func() {
				hookTargets.Do(func() { C.editkeysHookTargets() })
				if C.editkeysWatch(win.NativeWindow(), C.uintptr_t(win.ID())) == 0 {
					report(onError, ErrNoWindow)
				}
			})
		})
	})
}

// cgo writes a C prototype from these names, so none may be blank.
//
//export editkeysPress
func editkeysPress(widget *C.GtkWidget, event *C.GdkEventKey, data C.gpointer) C.gboolean {
	other := C.guint(C.GDK_MOD1_MASK | C.GDK_SUPER_MASK | C.GDK_META_MASK)
	action := actionOf(uint(C.editkeysLower(event)), C.editkeysHas(event, C.GDK_CONTROL_MASK) != 0,
		C.editkeysHas(event, C.GDK_SHIFT_MASK) != 0, C.editkeysHas(event, other) != 0)
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
