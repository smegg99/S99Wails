// shell/tray/tray_test.go

package tray

import (
	"sync"
	"testing"
	"time"
)

// fakeTray records what the Manager pushes to the platform tray.
type fakeTray struct {
	mu        sync.Mutex
	icon      []byte
	icons     int
	themePath string
	iconName  string
	tooltip   string
	items     []Item
	closed    bool
}

func (f *fakeTray) setIcon(icon []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.icon = icon
	f.icons++
	return nil
}

func (f *fakeTray) setThemedIcon(themePath, name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.themePath, f.iconName = themePath, name
}

func (f *fakeTray) setTooltip(text string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tooltip = text
}

func (f *fakeTray) setMenu(items []Item) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.items = items
}

func (f *fakeTray) close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
}

func (f *fakeTray) isClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

func (f *fakeTray) lastIcon() []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.icon
}

func eventually(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal(what)
}

func TestSetIconsSwapsAndReapplies(t *testing.T) {
	first := IconsFunc(func(dark bool) ([]byte, error) { return []byte{1}, nil })
	second := IconsFunc(func(dark bool) ([]byte, error) { return []byte{2}, nil })
	tray := &fakeTray{}
	m := start(tray, Options{Icons: first})
	defer m.Close()
	eventually(t, "first icon not applied", func() bool { return string(tray.lastIcon()) == "\x01" })

	m.SetIcons(second)
	eventually(t, "icons not swapped", func() bool { return string(tray.lastIcon()) == "\x02" })
}

func TestSameIconIsNotPushedTwice(t *testing.T) {
	tray := &fakeTray{}
	m := &Manager{tray: tray, icons: IconsFunc(func(bool) ([]byte, error) { return []byte{1}, nil })}
	m.apply()
	m.apply()
	if tray.icons != 1 {
		t.Fatalf("icon pushed %d times", tray.icons)
	}
}

func TestCloseIsIdempotentAndClosesTheTray(t *testing.T) {
	tray := &fakeTray{}
	m := start(tray, Options{})
	m.Close()
	m.Close()                                                             // must not panic
	m.SetIcons(IconsFunc(func(bool) ([]byte, error) { return nil, nil })) // after Close: no panic
	eventually(t, "tray not closed", tray.isClosed)
}

func TestMenuAndTooltipAreRebuilt(t *testing.T) {
	label := "Show"
	tray := &fakeTray{}
	m := start(tray, Options{Tooltip: func() string { return label + " tip" }})
	defer m.Close()

	m.SetMenu(func() []Item { return []Item{{Label: label}, {Separator: true}} })
	label = "Pokaz"
	m.RebuildMenu()

	tray.mu.Lock()
	defer tray.mu.Unlock()
	if len(tray.items) != 2 || tray.items[0].Label != "Pokaz" || !tray.items[1].Separator {
		t.Fatalf("menu = %+v", tray.items)
	}
	if tray.tooltip != "Pokaz tip" {
		t.Fatalf("tooltip = %q", tray.tooltip)
	}
}
