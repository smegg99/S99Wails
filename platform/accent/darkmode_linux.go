//go:build linux

// platform/accent/darkmode_linux.go

package accent

import "github.com/godbus/dbus/v5"

func detectScheme() (Scheme, bool) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return SchemeNoPreference, false
	}

	obj := conn.Object("org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop")
	var v dbus.Variant
	call := obj.Call("org.freedesktop.portal.Settings.Read", 0, "org.freedesktop.appearance", "color-scheme")
	if call.Err != nil {
		return SchemeNoPreference, false
	}
	if err := call.Store(&v); err != nil {
		return SchemeNoPreference, false
	}

	val := v.Value()
	if vv, ok := val.(dbus.Variant); ok {
		val = vv.Value()
	}

	var n int64
	switch t := val.(type) {
	case uint32:
		n = int64(t)
	case int32:
		n = int64(t)
	case int64:
		n = t
	case uint64:
		n = int64(t)
	default:
		return SchemeNoPreference, false
	}

	// org.freedesktop.appearance color-scheme: 0 no preference, 1 dark, 2 light.
	switch n {
	case 1:
		return SchemeDark, true
	case 2:
		return SchemeLight, true
	default:
		return SchemeNoPreference, true
	}
}
