// shell/windows/dnd_gtk3.h
// Native Wayland tab drag: GTK drag and drop with a private target between the app's own windows.
#include <gtk/gtk.h>
#ifdef GDK_WINDOWING_WAYLAND
#include <gdk/gdkwayland.h>
#endif
#include <stdint.h>

extern void s99wailsDndMotion(uintptr_t handle, int x, int y);
extern void s99wailsDndLeave(uintptr_t handle);
extern void s99wailsDndDrop(uintptr_t handle, int x, int y);
extern void s99wailsDndFailed(int result);
extern void s99wailsDndEnd(void);

static GtkTargetList *s99TabTargets(void) {
 static GtkTargetList *list;
 if (!list) {
  static const GtkTargetEntry entries[] = {{"application/x-s99wails-tab", GTK_TARGET_SAME_APP, 0}};
  list = gtk_target_list_new(entries, 1);
 }
 return list;
}

static int s99DndWayland(void) {
#ifdef GDK_WINDOWING_WAYLAND
 GdkDisplay *d = gdk_display_get_default();
 return d && GDK_IS_WAYLAND_DISPLAY(d);
#else
 return 0;
#endif
}

static GtkWidget *s99FindWebView(GtkWidget *widget) {
 GType type = g_type_from_name("WebKitWebView");
 if (!widget) return NULL;
 if (type && G_TYPE_CHECK_INSTANCE_TYPE(widget, type)) return widget;
 if (!GTK_IS_CONTAINER(widget)) return NULL;
 GtkWidget *found = NULL;
 GList *children = gtk_container_get_children(GTK_CONTAINER(widget));
 for (GList *l = children; l && !found; l = l->next) found = s99FindWebView(GTK_WIDGET(l->data));
 g_list_free(children);
 return found;
}

static int s99Ours(GtkWidget *widget, GdkDragContext *context) {
 return gtk_drag_dest_find_target(widget, context, s99TabTargets()) != GDK_NONE;
}

// The handlers run before WebKit's class handlers and stop the emission, so
// WebKit never sees a tab drag as HTML drag and drop.
static gboolean s99DestMotion(GtkWidget *w, GdkDragContext *c, gint x, gint y, guint time, gpointer handle) {
 if (!s99Ours(w, c)) return FALSE;
 gdk_drag_status(c, GDK_ACTION_MOVE, time);
 s99wailsDndMotion((uintptr_t)handle, x, y);
 g_signal_stop_emission_by_name(w, "drag-motion");
 return TRUE;
}
static void s99DestLeave(GtkWidget *w, GdkDragContext *c, guint time, gpointer handle) {
 if (!s99Ours(w, c)) return;
 s99wailsDndLeave((uintptr_t)handle);
 g_signal_stop_emission_by_name(w, "drag-leave");
}
static gboolean s99DestDrop(GtkWidget *w, GdkDragContext *c, gint x, gint y, guint time, gpointer handle) {
 if (!s99Ours(w, c)) return FALSE;
 s99wailsDndDrop((uintptr_t)handle, x, y);
 gtk_drag_finish(c, TRUE, FALSE, time);
 g_signal_stop_emission_by_name(w, "drag-drop");
 return TRUE;
}

static GdkDragContext *s99Active;

static gboolean s99SourceFailed(GtkWidget *w, GdkDragContext *c, GtkDragResult result, gpointer data) {
 if (c != s99Active) return FALSE;
 s99wailsDndFailed((int)result);
 return TRUE; // no snap-back animation: the tab never left the strip
}
static void s99SourceEnd(GtkWidget *w, GdkDragContext *c, gpointer data) {
 if (c != s99Active) return;
 s99Active = NULL;
 s99wailsDndEnd();
}

// s99DndAttach makes a window's webview a drop site for tabs and its window a
// drag source. Idempotent; returns 0 while the window has no webview yet.
static int s99DndAttach(void *window, uintptr_t handle) {
 if (!window || !GTK_IS_WINDOW(window)) return 0;
 if (g_object_get_data(G_OBJECT(window), "s99wails-dnd")) return 1;
 GtkWidget *view = s99FindWebView(GTK_WIDGET(window));
 if (!view) return 0;
 g_signal_connect(view, "drag-motion", G_CALLBACK(s99DestMotion), (gpointer)handle);
 g_signal_connect(view, "drag-leave", G_CALLBACK(s99DestLeave), (gpointer)handle);
 g_signal_connect(view, "drag-drop", G_CALLBACK(s99DestDrop), (gpointer)handle);
 g_signal_connect(window, "drag-failed", G_CALLBACK(s99SourceFailed), NULL);
 g_signal_connect(window, "drag-end", G_CALLBACK(s99SourceEnd), NULL);
 g_object_set_data(G_OBJECT(window), "s99wails-dnd", (gpointer)handle);
 return 1;
}

// s99DndLocal reads the pointer relative to the window's webview. During the
// implicit grab of a held button Wayland still reports it, also outside.
static int s99DndLocal(void *window, int *x, int *y, int *held, int *inside) {
 if (!window || !GTK_IS_WINDOW(window)) return 0;
 GtkWidget *win = GTK_WIDGET(window);
 GdkWindow *gdkWindow = gtk_widget_get_window(win);
 GtkWidget *view = s99FindWebView(win);
 if (!gdkWindow || !view) return 0;
 GdkDevice *pointer = gdk_seat_get_pointer(gdk_display_get_default_seat(gtk_widget_get_display(win)));
 GdkModifierType mask = 0;
 gint wx = 0, wy = 0, vx = 0, vy = 0;
 gdk_window_get_device_position(gdkWindow, pointer, &wx, &wy, &mask);
 if (!gtk_widget_translate_coordinates(win, view, wx, wy, &vx, &vy)) return 0;
 GtkAllocation a;
 gtk_widget_get_allocation(view, &a);
 *x = vx;
 *y = vy;
 *held = (mask & GDK_BUTTON1_MASK) != 0;
 *inside = vx >= 0 && vy >= 0 && vx < a.width && vy < a.height;
 return 1;
}

// s99DndBegin starts the drag from the window holding the implicit grab.
static int s99DndBegin(void *window) {
 if (!window || !GTK_IS_WINDOW(window) || s99Active) return 0;
 GdkDragContext *c = gtk_drag_begin_with_coordinates(GTK_WIDGET(window), s99TabTargets(), GDK_ACTION_MOVE, 1, NULL, -1, -1);
 if (!c) return 0;
 s99Active = c;
 return 1;
}

static void s99DndCancel(void) {
 if (s99Active) gtk_drag_cancel(s99Active);
}

// The page's picture of the dragged tab: the drag icon on Wayland, a popup window on X11.
static GtkWidget *s99Chip, *s99Blank, *s99Float;
static int s99ChipX, s99ChipY, s99IconShown = -1;

static int s99DndImage(const void *png, int length, int scale, int x, int y) {
 GdkPixbufLoader *loader = gdk_pixbuf_loader_new_with_type("png", NULL);
 if (!loader) return 0;
 if (!gdk_pixbuf_loader_write(loader, png, length, NULL) || !gdk_pixbuf_loader_close(loader, NULL)) {
  g_object_unref(loader);
  return 0;
 }
 cairo_surface_t *surface = gdk_cairo_surface_create_from_pixbuf(gdk_pixbuf_loader_get_pixbuf(loader), scale > 0 ? scale : 1, NULL);
 g_object_unref(loader);
 GtkWidget *image = gtk_image_new_from_surface(surface);
 cairo_surface_destroy(surface);
 g_object_ref_sink(image);
 gtk_widget_show(image);
 if (s99Chip) g_object_unref(s99Chip);
 s99Chip = image;
 s99ChipX = x;
 s99ChipY = y;
 s99IconShown = -1;
 return 1;
}

// s99DndIcon shows the picture under the Wayland drag, or a blank one.
static void s99DndIcon(int show) {
 if (!s99Active) return;
 show = show && s99Chip;
 if (show == s99IconShown) return;
 if (!s99Blank) {
  cairo_surface_t *empty = cairo_image_surface_create(CAIRO_FORMAT_ARGB32, 1, 1);
  s99Blank = gtk_image_new_from_surface(empty);
  cairo_surface_destroy(empty);
  g_object_ref_sink(s99Blank);
  gtk_widget_show(s99Blank);
 }
 if (show) gtk_drag_set_icon_widget(s99Active, s99Chip, s99ChipX, s99ChipY);
 else gtk_drag_set_icon_widget(s99Active, s99Blank, 0, 0);
 s99IconShown = show;
}

static gboolean s99FloatDraw(GtkWidget *w, cairo_t *cr, gpointer data) {
 cairo_save(cr);
 cairo_set_operator(cr, CAIRO_OPERATOR_SOURCE);
 cairo_set_source_rgba(cr, 0, 0, 0, 0);
 cairo_paint(cr);
 cairo_restore(cr);
 return FALSE;
}

// s99DndFloat shows the input-less picture at x, y in screen coordinates, or hides it.
// The input shape is set after showing: GTK resets one set before.
static void s99DndFloat(int show, int x, int y) {
 if (!show || !s99Chip) {
  if (s99Float) gtk_widget_hide(s99Float);
  return;
 }
 if (!s99Float) {
  s99Float = gtk_window_new(GTK_WINDOW_POPUP);
  gtk_window_set_type_hint(GTK_WINDOW(s99Float), GDK_WINDOW_TYPE_HINT_DND);
  GdkScreen *screen = gtk_widget_get_screen(s99Float);
  GdkVisual *visual = gdk_screen_get_rgba_visual(screen);
  if (visual && gdk_screen_is_composited(screen)) {
   gtk_widget_set_visual(s99Float, visual);
   gtk_widget_set_app_paintable(s99Float, TRUE);
   g_signal_connect(s99Float, "draw", G_CALLBACK(s99FloatDraw), NULL);
  }
  gtk_widget_realize(s99Float);
 }
 GtkWidget *child = gtk_bin_get_child(GTK_BIN(s99Float));
 if (child != s99Chip) {
  if (child) gtk_container_remove(GTK_CONTAINER(s99Float), child);
  gtk_container_add(GTK_CONTAINER(s99Float), s99Chip);
  gtk_window_resize(GTK_WINDOW(s99Float), 1, 1);
 }
 gtk_window_move(GTK_WINDOW(s99Float), x - s99ChipX, y - s99ChipY);
 if (!gtk_widget_get_visible(s99Float)) {
  gtk_widget_show(s99Float);
  cairo_region_t *none = cairo_region_create();
  gtk_widget_input_shape_combine_region(s99Float, none);
  cairo_region_destroy(none);
 }
}

// s99DndClear drops the picture once the drag is over.
static void s99DndClear(void) {
 if (s99Float) {
  gtk_widget_destroy(s99Float);
  s99Float = NULL;
 }
 if (s99Chip) {
  g_object_unref(s99Chip);
  s99Chip = NULL;
 }
 s99IconShown = -1;
}
