// shell/mousenav/mousenav_gtk3.go

//go:build linux && cgo && gtk3

package mousenav

/*
#cgo pkg-config: gtk+-3.0
#include <gtk/gtk.h>
#include <stdint.h>

extern gboolean mousenavButton(GtkWidget*, GdkEventButton*, gpointer);
extern void mousenavDestroyed(GtkWidget*, gpointer);

static GtkWidget* mousenavFindWebView(GtkWidget* widget) {
	if (g_strcmp0(G_OBJECT_TYPE_NAME(widget), "WebKitWebView") == 0) return widget;
	if (!GTK_IS_CONTAINER(widget)) return NULL;
	GList* children = gtk_container_get_children(GTK_CONTAINER(widget));
	GtkWidget* found = NULL;
	for (GList* l = children; l != NULL && found == NULL; l = l->next) found = mousenavFindWebView(GTK_WIDGET(l->data));
	g_list_free(children);
	return found;
}

static gboolean mousenavWatch(void* window, uintptr_t id) {
	GtkWidget* view = mousenavFindWebView(GTK_WIDGET(window));
	if (view == NULL) return FALSE;
	g_signal_connect(view, "button-press-event", G_CALLBACK(mousenavButton), (gpointer)id);
	g_signal_connect(view, "button-release-event", G_CALLBACK(mousenavButton), (gpointer)id);
	g_signal_connect(window, "destroy", G_CALLBACK(mousenavDestroyed), (gpointer)id);
	return TRUE;
}
*/
import "C"

import (
	"errors"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// ErrNoWebView reports a window with no WebKitWebView inside it to watch.
var ErrNoWebView = errors.New("mousenav: no webview in the window")

// Each callback belongs to its native window. GTK destruction, rather than
// a cancellable close request, removes it.
var windows sync.Map // uint -> *application.WebviewWindow

// Watch sends the window's side buttons as EventStep from the first time it is
// shown; the page never receives them, since WebKitGTK hands it both as one.
func Watch(win *application.WebviewWindow, onError func(error)) {
	windows.Store(win.ID(), win)
	var once sync.Once
	win.OnWindowEvent(events.Common.WindowShow, func(*application.WindowEvent) {
		once.Do(func() {
			application.InvokeSync(func() {
				if C.mousenavWatch(win.NativeWindow(), C.uintptr_t(win.ID())) == 0 {
					onError(ErrNoWebView)
				}
			})
		})
	})
}

// cgo writes a C prototype from these names, so none of them may be blank.
//
//export mousenavButton
func mousenavButton(widget *C.GtkWidget, event *C.GdkEventButton, data C.gpointer) C.gboolean {
	step := stepOf(uint(event.button))
	if step == 0 {
		return C.FALSE
	}
	if event._type == C.GDK_BUTTON_RELEASE {
		if win, ok := windows.Load(uint(uintptr(data))); ok {
			win.(*application.WebviewWindow).DispatchWailsEvent(&application.CustomEvent{Name: EventStep, Data: step})
		}
	}
	return C.TRUE
}

//export mousenavDestroyed
func mousenavDestroyed(widget *C.GtkWidget, data C.gpointer) {
	windows.Delete(uint(uintptr(data)))
}
