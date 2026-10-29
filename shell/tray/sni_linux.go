//go:build linux

// shell/tray/sni_linux.go

package tray

import (
	"errors"
	"fmt"
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

// sni is a StatusNotifierItem with a dbusmenu on its own bus connection, since Wails publishes only a pixmap.
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

// start exports the item and registers it with the panel, now and whenever a panel restarts.
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

// watch registers with the current panel and every new one until the connection closes.
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

func (s *sni) close() {
	s.mu.Lock()
	conn := s.conn
	s.conn, s.props, s.menuProps = nil, nil, nil
	s.mu.Unlock()
	if conn != nil {
		s.report(conn.Close())
	}
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
