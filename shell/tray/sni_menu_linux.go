//go:build linux

// shell/tray/sni_menu_linux.go

package tray

import (
	"fmt"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
)

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

func menuSpec() map[string]map[string]*prop.Prop {
	read := func(v any) *prop.Prop { return &prop.Prop{Value: v, Emit: prop.EmitTrue} }
	return map[string]map[string]*prop.Prop{menuIface: {
		"Version":       read(dbusmenuVersion),
		"TextDirection": read("ltr"),
		"Status":        read("normal"),
		"IconThemePath": read([]string{}),
	}}
}

func (s *sni) setMenu(items []Item) {
	s.mu.Lock()
	same := sameLayout(s.items, items)
	s.items = items
	if same {
		s.mu.Unlock()
		return
	}
	s.revision++
	var err error
	if s.conn != nil {
		err = s.conn.Emit(menuPath, menuIface+".LayoutUpdated", s.revision, int32(0))
	}
	s.mu.Unlock()
	s.report(err)
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
