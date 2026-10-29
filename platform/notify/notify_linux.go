//go:build linux

// platform/notify/notify_linux.go
// Package notify sends notifications through the existing desktop session bus.
package notify

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/godbus/dbus/v5"
)

const (
	service = "org.freedesktop.Notifications"
	path    = "/org/freedesktop/Notifications"
	method  = service + ".Notify"
)

// Desktop sends notifications as one application over one bus connection, reopened after a failure.
type Desktop struct {
	app string

	mu      sync.Mutex
	conn    *dbus.Conn
	closed  bool
	actions map[uint32]func()
}

// New names the application every notification comes from.
func New(app string) *Desktop { return &Desktop{app: app} }

// Send shows a notification with a title and one line, for the server's own time.
func (d *Desktop) Send(ctx context.Context, title, body string) error {
	return d.send(ctx, title, body, "", nil)
}

// SendAction opens an application destination when the notification is activated.
func (d *Desktop) SendAction(ctx context.Context, title, body, label string, activate func()) error {
	return d.send(ctx, title, body, label, activate)
}

func (d *Desktop) send(ctx context.Context, title, body, label string, activate func()) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	conn, err := d.bus(ctx)
	if err != nil {
		return err
	}
	actions := []string{}
	if activate != nil {
		actions = []string{"default", label}
	}
	// Register the reply before the signal reader can handle an immediate activation.
	d.mu.Lock()
	var id uint32
	err = conn.Object(service, path).CallWithContext(ctx, method, 0,
		d.app, uint32(0), "", title, body, actions, map[string]dbus.Variant{}, int32(-1)).Store(&id)
	if err == nil && activate != nil {
		d.actions[id] = activate
	}
	d.mu.Unlock()
	if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		d.drop(conn)
	}
	return err
}

func (d *Desktop) signals(conn *dbus.Conn, signals <-chan *dbus.Signal) {
	for signal := range signals {
		if signal == nil || len(signal.Body) < 2 {
			continue
		}
		id, ok := signal.Body[0].(uint32)
		if !ok {
			continue
		}
		d.mu.Lock()
		var action func()
		if d.conn == conn {
			if signal.Name == service+".ActionInvoked" && signal.Body[1] == "default" {
				action = d.actions[id]
			}
			if signal.Name == service+".NotificationClosed" || action != nil {
				delete(d.actions, id)
			}
		}
		d.mu.Unlock()
		if action != nil {
			action()
		}
	}
}

// Close ends the bus connection, if one is open.
func (d *Desktop) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.closed = true
	if d.conn != nil {
		_ = d.conn.Close()
		d.conn = nil
	}
}

func (d *Desktop) bus(ctx context.Context) (*dbus.Conn, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil, io.ErrClosedPipe
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if d.conn != nil && d.conn.Connected() {
		return d.conn, nil
	}
	// Never the autolaunching kind, which starts a bus of its own through
	// dbus-launch when the desktop has none.
	conn, err := dbus.SessionBusPrivateNoAutoStartup()
	if err != nil {
		return nil, err
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	if err := conn.Auth(nil); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := conn.Hello(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	signals := make(chan *dbus.Signal, 32)
	conn.Signal(signals)
	if err := conn.AddMatchSignalContext(ctx, dbus.WithMatchObjectPath(path), dbus.WithMatchInterface(service), dbus.WithMatchSender(service)); err != nil {
		_ = conn.Close()
		return nil, err
	}
	d.conn = conn
	d.actions = map[uint32]func(){}
	go d.signals(conn, signals)
	return conn, nil
}

// drop forgets a connection a call failed on, so the next Send opens a new one.
func (d *Desktop) drop(conn *dbus.Conn) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.conn == conn {
		_ = conn.Close()
		d.conn = nil
	}
}
