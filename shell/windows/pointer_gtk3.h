// shell/windows/pointer_gtk3.h
#include <gtk/gtk.h>
#include <gdk/gdkx.h>
static int pointerSample(int *x, int *y, int *held) {
 GdkDisplay *d = gdk_display_get_default();
 if (!d || !GDK_IS_X11_DISPLAY(d)) return 0;
 GdkDevice *p = gdk_seat_get_pointer(gdk_display_get_default_seat(d));
 GdkModifierType mask = 0;
 gdk_device_get_position(p, NULL, x, y);
 gdk_window_get_device_position(gdk_get_default_root_window(), p, NULL, NULL, &mask);
 *held = (mask & GDK_BUTTON1_MASK) != 0;
 return 1;
}
static int pointerWindow(void *widget, int *x, int *y) {
 if (!widget || !GTK_IS_WINDOW(widget) || !gtk_widget_get_visible(GTK_WIDGET(widget))) return 0;
 GdkWindow *w = gtk_widget_get_window(GTK_WIDGET(widget));
 if (!w) return 0;
 GdkDisplay *display = gdk_window_get_display(w);
 if (!GDK_IS_X11_DISPLAY(display)) return 0;
 Display *xd = GDK_DISPLAY_XDISPLAY(display);
 Window expected = GDK_WINDOW_XID(w);
 Window root = GDK_WINDOW_XID(gdk_screen_get_root_window(gdk_window_get_screen(w)));
 Window current = root, child, queryRoot;
 int rootX, rootY, localX, localY, found = 0;
 unsigned int mask;
 // A WebKit child may belong to another X client and have no GdkWindow.
 gdk_x11_display_error_trap_push(display);
 for (int depth = 0; depth < 64 && current != None; depth++) {
  if (current == expected) { found = 1; break; }
  if (!XQueryPointer(xd, current, &queryRoot, &child, &rootX, &rootY, &localX, &localY, &mask)) break;
  current = child;
 }
 if (gdk_x11_display_error_trap_pop(display)) return 0;
 if (!found) return 0;
 gdk_window_get_origin(w, x, y);
 return 1;
}
