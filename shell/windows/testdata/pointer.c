// shell/windows/testdata/pointer.c
#include "../pointer_gtk3.h"
#include <assert.h>

int main(int argc, char **argv) {
 gtk_init(&argc, &argv);
 GtkWidget *widget = gtk_window_new(GTK_WINDOW_TOPLEVEL);
 gtk_window_set_default_size(GTK_WINDOW(widget), 400, 300);
 gtk_widget_show_all(widget);
 while (gtk_events_pending()) gtk_main_iteration();
 GdkWindow *w = gtk_widget_get_window(widget);
 Display *other = XOpenDisplay(NULL);
 assert(other);
 Window root = DefaultRootWindow(other);
 Window child = XCreateSimpleWindow(other, GDK_WINDOW_XID(w), 0, 0, 400, 300, 0, 0, 0);
 XMapWindow(other, child);
 XWarpPointer(other, None, child, 0, 0, 0, 0, 50, 50);
 XSync(other, False);
 int x, y;
 assert(pointerWindow(widget, &x, &y));
 int px, py, held;
 assert(pointerSample(&px, &py, &held));
 assert(px - x == 50 && py - y == 50);

 // An unrelated top-level covering the same coordinates must block the hit.
 Window cover = XCreateSimpleWindow(other, root, x, y, 400, 300, 0, 0, 0);
 XMapRaised(other, cover);
 XSync(other, False);
 assert(!pointerWindow(widget, &x, &y));
 XDestroyWindow(other, cover);
 XSync(other, False);
 assert(pointerWindow(widget, &x, &y));

 XDestroyWindow(other, child);
 XSync(other, False);
 assert(pointerWindow(widget, &x, &y));
 gtk_widget_hide(widget);
 assert(!pointerWindow(widget, &x, &y));
 XCloseDisplay(other);
 gtk_widget_destroy(widget);
 return 0;
}
