//go:build linux

// shell/tray/sni_linux.go

package tray

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	itemIface   = "org.kde.StatusNotifierItem"
	menuIface   = "com.canonical.dbusmenu"
	watcherName = "org.kde.StatusNotifierWatcher"

	itemPath    = dbus.ObjectPath("/StatusNotifierItem")
	menuPath    = dbus.ObjectPath("/StatusNotifierMenu")
	watcherPath = dbus.ObjectPath("/StatusNotifierWatcher")

	// dbusmenuVersion is the dbusmenu protocol version, not the layout revision.
	dbusmenuVersion = uint32(3)
)

// pixmap is one StatusNotifierItem image: ARGB32 pixels in network byte order.
type pixmap struct {
	Width, Height int32
	Pixels        []byte
}

// tooltip is the StatusNotifierItem ToolTip: icon name, icons, title, text.
type tooltip struct {
	IconName string
	Icons    []pixmap
	Title    string
	Text     string
}

// menuNode is one dbusmenu layout entry: id, properties and child nodes.
type menuNode struct {
	ID         int32
	Properties map[string]dbus.Variant
	Children   []dbus.Variant
}

type menuProperties struct {
	ID         int32
	Properties map[string]dbus.Variant
}

type menuEvent struct {
	ID        int32
	EventID   string
	Data      dbus.Variant
	Timestamp uint32
}

// sni is a StatusNotifierItem with a dbusmenu on its own session-bus connection.
// Wails' Linux tray publishes only a pixmap, which the panel cannot draw as SVG.
type sni struct {
	name    string
	busName string
	onClick func()
	report  func(error)

	// mu guards the state below and the export handles. Setters before the
	// export only record state; export publishes the latest of it.
	mu        sync.Mutex
	conn      *dbus.Conn
	props     *prop.Properties
	menuProps *prop.Properties
	icons     []pixmap
	themePath string
	iconName  string
	text      string
	items     []Item
	revision  uint32
}

func newPlatform(app *application.App, opts Options) backend {
	item := newSNI(app.Config().Name, opts)
	opts.report(item.start())
	return item
}

func newSNI(name string, opts Options) *sni {
	return &sni{
		name:    name,
		busName: fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid()),
		onClick: opts.OnClick,
		report:  opts.report,
		icons:   []pixmap{},
	}
}

// start exports the item and registers it with the panel, now and whenever a
// panel (re)starts.
func (s *sni) start() error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("tray: session bus: %w", err)
	}
	if err := s.export(conn); err != nil {
		conn.Close()
		return err
	}
	if err := conn.AddMatchSignal(
		dbus.WithMatchSender("org.freedesktop.DBus"),
		dbus.WithMatchInterface("org.freedesktop.DBus"),
		dbus.WithMatchMember("NameOwnerChanged"),
		dbus.WithMatchArg(0, watcherName),
	); err != nil {
		s.report(fmt.Errorf("tray: watch for the panel: %w", err))
	}
	signals := make(chan *dbus.Signal, 8)
	conn.Signal(signals)
	go s.watch(conn, signals)
	return nil
}

func (s *sni) export(conn *dbus.Conn) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := conn.Export(itemMethods{s}, itemPath, itemIface); err != nil {
		return fmt.Errorf("tray: export item: %w", err)
	}
	if err := conn.Export(menuMethods{s}, menuPath, menuIface); err != nil {
		return fmt.Errorf("tray: export menu: %w", err)
	}
	props, err := prop.Export(conn, itemPath, s.itemSpec())
	if err != nil {
		return fmt.Errorf("tray: export item properties: %w", err)
	}
	menuProps, err := prop.Export(conn, menuPath, menuSpec())
	if err != nil {
		return fmt.Errorf("tray: export menu properties: %w", err)
	}
	if err := exportIntrospection(conn, props, menuProps); err != nil {
		return err
	}
	reply, err := conn.RequestName(s.busName, dbus.NameFlagDoNotQueue)
	if err != nil {
		return fmt.Errorf("tray: request %s: %w", s.busName, err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return fmt.Errorf("tray: bus name %s is taken", s.busName)
	}
	s.conn, s.props, s.menuProps = conn, props, menuProps
	return nil
}

func exportIntrospection(conn *dbus.Conn, props, menuProps *prop.Properties) error {
	item := introspect.Node{Name: string(itemPath), Interfaces: []introspect.Interface{
		introspect.IntrospectData,
		prop.IntrospectData,
		{
			Name:       itemIface,
			Methods:    introspect.Methods(itemMethods{}),
			Properties: props.Introspection(itemIface),
			Signals: []introspect.Signal{
				{Name: "NewTitle"}, {Name: "NewIcon"}, {Name: "NewToolTip"}, {Name: "NewMenu"},
				{Name: "NewStatus", Args: []introspect.Arg{{Name: "status", Type: "s"}}},
				{Name: "NewIconThemePath", Args: []introspect.Arg{{Name: "icon_theme_path", Type: "s"}}},
			},
		},
	}}
	menu := introspect.Node{Name: string(menuPath), Interfaces: []introspect.Interface{
		introspect.IntrospectData,
		prop.IntrospectData,
		{
			Name:       menuIface,
			Methods:    introspect.Methods(menuMethods{}),
			Properties: menuProps.Introspection(menuIface),
			Signals: []introspect.Signal{
				{Name: "LayoutUpdated", Args: []introspect.Arg{{Name: "revision", Type: "u"}, {Name: "parent", Type: "i"}}},
				{Name: "ItemsPropertiesUpdated", Args: []introspect.Arg{
					{Name: "updatedProps", Type: "a(ia{sv})"}, {Name: "removedProps", Type: "a(ias)"}}},
			},
		},
	}}
	for path, node := range map[dbus.ObjectPath]*introspect.Node{itemPath: &item, menuPath: &menu} {
		if err := conn.Export(introspect.NewIntrospectable(node), path, "org.freedesktop.DBus.Introspectable"); err != nil {
			return fmt.Errorf("tray: export introspection: %w", err)
		}
	}
	return nil
}

// watch registers with the current panel and again with every new one, until
// the connection closes.
func (s *sni) watch(conn *dbus.Conn, signals <-chan *dbus.Signal) {
	s.register(conn)
	for sig := range signals {
		if sig.Name != "org.freedesktop.DBus.NameOwnerChanged" || len(sig.Body) < 3 {
			continue
		}
		name, _ := sig.Body[0].(string)
		owner, _ := sig.Body[2].(string)
		if name == watcherName && owner != "" {
			s.register(conn)
		}
	}
}

func (s *sni) register(conn *dbus.Conn) {
	call := conn.Object(watcherName, watcherPath).Call(watcherName+".RegisterStatusNotifierItem", 0, s.busName)
	var busErr dbus.Error
	// With no panel yet, watch registers when one starts.
	if errors.As(call.Err, &busErr) && busErr.Name == "org.freedesktop.DBus.Error.ServiceUnknown" {
		return
	}
	if call.Err != nil {
		s.report(fmt.Errorf("tray: register with the panel: %w", call.Err))
	}
}

func (s *sni) itemSpec() map[string]map[string]*prop.Prop {
	read := func(v any) *prop.Prop { return &prop.Prop{Value: v, Emit: prop.EmitTrue} }
	return map[string]map[string]*prop.Prop{itemIface: {
		"Category":            read("ApplicationStatus"),
		"Id":                  read(s.name),
		"Title":               read(s.name),
		"Status":              read("Active"),
		"WindowId":            read(int32(0)),
		"IconName":            read(s.iconName),
		"IconThemePath":       read(s.themePath),
		"IconPixmap":          read(s.icons),
		"OverlayIconName":     read(""),
		"OverlayIconPixmap":   read([]pixmap{}),
		"AttentionIconName":   read(""),
		"AttentionIconPixmap": read([]pixmap{}),
		"AttentionMovieName":  read(""),
		"ToolTip":             read(s.tooltip()),
		"ItemIsMenu":          read(false),
		"Menu":                read(menuPath),
	}}
}

func menuSpec() map[string]map[string]*prop.Prop {
	read := func(v any) *prop.Prop { return &prop.Prop{Value: v, Emit: prop.EmitTrue} }
	return map[string]map[string]*prop.Prop{menuIface: {
		"Version":       read(dbusmenuVersion),
		"TextDirection": read("ltr"),
		"Status":        read("normal"),
		"IconThemePath": read([]string{}),
	}}
}

// tooltip titles the tooltip with the application and puts the text under it.
func (s *sni) tooltip() tooltip {
	return tooltip{Icons: []pixmap{}, Title: s.name, Text: s.text}
}

func (s *sni) setIcon(icon []byte) error {
	px, err := toPixmap(icon)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.icons = []pixmap{px}
	s.publish(itemIface, "IconPixmap", s.icons, "NewIcon")
	return nil
}

func (s *sni) setThemedIcon(themePath, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.themePath == themePath && s.iconName == name {
		return
	}
	s.themePath, s.iconName = themePath, name
	s.publish(itemIface, "IconThemePath", themePath, "")
	s.publish(itemIface, "IconName", name, "NewIcon")
}

func (s *sni) setTooltip(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.text == text {
		return
	}
	s.text = text
	s.publish(itemIface, "ToolTip", s.tooltip(), "NewToolTip")
}

func (s *sni) setMenu(items []Item) {
	s.mu.Lock()
	defer s.mu.Unlock()
	same := sameLayout(s.items, items)
	s.items = items
	if same {
		return
	}
	s.revision++
	if s.conn != nil {
		s.report(s.conn.Emit(menuPath, menuIface+".LayoutUpdated", s.revision, int32(0)))
	}
}

// publish sets an exported property and sends the item signal hosts listen
// for. Before the export there is nothing to send; the caller holds mu.
func (s *sni) publish(iface, name string, value any, signal string) {
	if s.props == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			s.report(fmt.Errorf("tray: publish %s: %w", name, recoveredError(r)))
		}
	}()
	s.props.SetMust(iface, name, value)
	if signal != "" {
		s.report(s.conn.Emit(itemPath, itemIface+"."+signal))
	}
}

func (s *sni) close() {
	s.mu.Lock()
	conn := s.conn
	s.conn, s.props, s.menuProps = nil, nil, nil
	s.mu.Unlock()
	if conn != nil {
		s.report(conn.Close())
	}
}

// run calls an application action off the bus goroutine. Wails' window and
// app methods dispatch to the GUI thread themselves.
func (s *sni) run(action func()) {
	if action == nil {
		return
	}
	go action()
}

func (s *sni) click(id int32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := int(id) - 1
	if index < 0 || index >= len(s.items) {
		return false
	}
	if item := s.items[index]; !item.Separator {
		s.run(item.OnClick)
	}
	return true
}

// node is a menu entry's layout; id 0 is the root, entries count from 1. The
// caller holds mu.
func (s *sni) node(id int32, depth int32) (menuNode, bool) {
	if id == 0 {
		root := menuNode{ID: 0, Properties: map[string]dbus.Variant{
			"children-display": dbus.MakeVariant("submenu"),
		}, Children: []dbus.Variant{}}
		if depth != 0 {
			for i := range s.items {
				child, _ := s.node(int32(i+1), 0)
				root.Children = append(root.Children, dbus.MakeVariant(child))
			}
		}
		return root, true
	}
	index := int(id) - 1
	if index < 0 || index >= len(s.items) {
		return menuNode{}, false
	}
	item := s.items[index]
	properties := map[string]dbus.Variant{"label": dbus.MakeVariant(item.Label)}
	if item.Separator {
		properties = map[string]dbus.Variant{"type": dbus.MakeVariant("separator")}
	}
	return menuNode{ID: id, Properties: properties, Children: []dbus.Variant{}}, true
}

func sameLayout(a, b []Item) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Label != b[i].Label || a[i].Separator != b[i].Separator {
			return false
		}
	}
	return true
}

func noItem(id int32) *dbus.Error {
	return dbus.NewError("org.freedesktop.DBus.Error.InvalidArgs", []any{fmt.Sprintf("no menu item %d", id)})
}

// toPixmap converts a PNG to the item's ARGB32 network-order pixels.
func toPixmap(icon []byte) (pixmap, error) {
	img, err := png.Decode(bytes.NewReader(icon))
	if err != nil {
		return pixmap{}, fmt.Errorf("tray: icon is not a PNG: %w", err)
	}
	bounds := img.Bounds()
	rgba := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, bounds.Min, draw.Src)
	pixels := make([]byte, len(rgba.Pix))
	for i := 0; i < len(rgba.Pix); i += 4 {
		pixels[i] = rgba.Pix[i+3]
		pixels[i+1] = rgba.Pix[i]
		pixels[i+2] = rgba.Pix[i+1]
		pixels[i+3] = rgba.Pix[i+2]
	}
	return pixmap{Width: int32(bounds.Dx()), Height: int32(bounds.Dy()), Pixels: pixels}, nil
}

// itemMethods is org.kde.StatusNotifierItem.
type itemMethods struct{ s *sni }

func (m itemMethods) Activate(x, y int32) *dbus.Error {
	m.s.run(m.s.onClick)
	return nil
}

// SecondaryActivate (middle click) does nothing; the host opens the menu itself.
func (m itemMethods) SecondaryActivate(x, y int32) *dbus.Error { return nil }

func (m itemMethods) ContextMenu(x, y int32) *dbus.Error { return nil }

func (m itemMethods) Scroll(delta int32, orientation string) *dbus.Error { return nil }

// menuMethods is com.canonical.dbusmenu.
type menuMethods struct{ s *sni }

func (m menuMethods) GetLayout(parent, depth int32, names []string) (uint32, menuNode, *dbus.Error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	node, ok := m.s.node(parent, depth)
	if !ok {
		return 0, menuNode{}, noItem(parent)
	}
	return m.s.revision, node, nil
}

func (m menuMethods) GetGroupProperties(ids []int32, names []string) ([]menuProperties, *dbus.Error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	if len(ids) == 0 {
		for i := 0; i <= len(m.s.items); i++ {
			ids = append(ids, int32(i))
		}
	}
	out := []menuProperties{}
	for _, id := range ids {
		if node, ok := m.s.node(id, 0); ok {
			out = append(out, menuProperties{ID: id, Properties: node.Properties})
		}
	}
	return out, nil
}

func (m menuMethods) GetProperty(id int32, name string) (dbus.Variant, *dbus.Error) {
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	node, ok := m.s.node(id, 0)
	if !ok {
		return dbus.Variant{}, noItem(id)
	}
	value, ok := node.Properties[name]
	if !ok {
		return dbus.Variant{}, dbus.NewError("org.freedesktop.DBus.Error.InvalidArgs", []any{"no property " + name})
	}
	return value, nil
}

func (m menuMethods) Event(id int32, eventID string, data dbus.Variant, timestamp uint32) *dbus.Error {
	if eventID == "clicked" && !m.s.click(id) {
		return noItem(id)
	}
	return nil
}

func (m menuMethods) EventGroup(events []menuEvent) ([]int32, *dbus.Error) {
	missing := []int32{}
	for _, event := range events {
		if event.EventID == "clicked" && !m.s.click(event.ID) {
			missing = append(missing, event.ID)
		}
	}
	return missing, nil
}

func (m menuMethods) AboutToShow(id int32) (bool, *dbus.Error) { return false, nil }

func (m menuMethods) AboutToShowGroup(ids []int32) ([]int32, []int32, *dbus.Error) {
	return []int32{}, []int32{}, nil
}
