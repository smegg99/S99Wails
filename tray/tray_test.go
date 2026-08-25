// tray/tray_test.go

package tray

import (
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestSetIconsSwapsAndReapplies(t *testing.T) {
	first := IconsFunc(func(dark bool) ([]byte, error) { return []byte{1}, nil })
	second := IconsFunc(func(dark bool) ([]byte, error) { return []byte{2}, nil })

	// nil tray: apply() must skip SetIcon instead of panicking (test seam).
	m := NewManager(nil, Options{Icons: first})
	defer m.Close()
	time.Sleep(50 * time.Millisecond) // let the initial apply drain

	m.SetIcons(second)
	time.Sleep(50 * time.Millisecond)

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.opts.Icons == nil {
		t.Fatal("icons not set")
	}
	got, _ := m.opts.Icons.Icon(false)
	if len(got) != 1 || got[0] != 2 {
		t.Fatalf("Icons not swapped, got %v", got)
	}
	if m.lastIcon != nil {
		t.Fatal("lastIcon dedupe cache not cleared on SetIcons")
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	m := NewManager(nil, Options{})
	m.Close()
	m.Close()                                                             // must not panic
	m.SetIcons(IconsFunc(func(bool) ([]byte, error) { return nil, nil })) // after Close: no panic
}

func TestNilTrayTooltipAndMenuDoNotPanic(t *testing.T) {
	m := NewManager(nil, Options{Tooltip: func() string { return "t" }})
	defer m.Close()

	m.SetMenu(func() *application.Menu { return nil })
	m.RebuildMenu()
}
