// accent/accentwatch_linux.go
//go:build linux

package accent

import (
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

// debounceWindow coalesces the burst of SettingChanged signals one change emits.
const debounceWindow = 200 * time.Millisecond

func watchAccentChanges(onChange func()) (stop func()) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return func() {}
	}

	rule := "type='signal',interface='org.freedesktop.portal.Settings',member='SettingChanged',path='/org/freedesktop/portal/desktop'"
	if err := conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, rule).Err; err != nil {
		return func() {}
	}

	ch := make(chan *dbus.Signal, 16)
	conn.Signal(ch)

	done := make(chan struct{})
	trigger := make(chan struct{}, 1)

	// Producer: fan dbus signals into a single coalesce channel.
	go func() {
		defer func() { _ = recover() }()
		for {
			select {
			case sig, ok := <-ch:
				if !ok {
					return
				}
				if sig == nil || sig.Name != "org.freedesktop.portal.Settings.SettingChanged" {
					continue
				}
				if len(sig.Body) < 2 {
					continue
				}
				ns, _ := sig.Body[0].(string)
				if ns != "org.freedesktop.appearance" {
					continue
				}
				select {
				case trigger <- struct{}{}:
				default:
				}
			case <-done:
				return
			}
		}
	}()

	// Consumer: debounce, then fire onChange after a quiet period.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				_ = r
			}
		}()
		var timer *time.Timer
		for {
			select {
			case <-trigger:
				if timer == nil {
					timer = time.NewTimer(debounceWindow)
				} else {
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					timer.Reset(debounceWindow)
				}
			case <-timerC(timer):
				timer = nil
				func() {
					defer func() { _ = recover() }()
					onChange()
				}()
			case <-done:
				if timer != nil {
					timer.Stop()
				}
				return
			}
		}
	}()

	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
			conn.RemoveSignal(ch)
			_ = conn.BusObject().Call("org.freedesktop.DBus.RemoveMatch", 0, rule).Err
		})
	}
}

// timerC returns the timer's channel or nil so the select arm is
// disabled when no debounce timer is currently armed.
func timerC(t *time.Timer) <-chan time.Time {
	if t == nil {
		return nil
	}
	return t.C
}
