// shell/tray/tray.go

// Package tray wraps the Wails v3 system tray with theme-aware icons and rebuildable menus.
package tray

import (
	"bytes"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/smegg99/s99wails/v2/platform/accent"
)

// Icons resolves the tray icon for the current OS theme.
type Icons interface {
	// Icon returns the PNG bytes to show while the OS theme is dark (true)
	// or light (false).
	Icon(dark bool) ([]byte, error)
}

// IconsFunc adapts a plain function to the Icons interface.
type IconsFunc func(dark bool) ([]byte, error)

func (f IconsFunc) Icon(dark bool) ([]byte, error) { return f(dark) }

// Options configures a Manager.
type Options struct {
	Icons Icons
	// Tooltip supplies the (localized) tray tooltip. It is applied when set
	// and re-applied on every RebuildMenu, so locale changes pick it up.
	Tooltip func() string
	// OnError receives icon-loading failures and recovered panics from the
	// apply loop. Optional; errors are dropped when nil.
	OnError func(error)
	// ThemeChangedEvents lists extra custom events that trigger dark-mode
	// re-detection, such as theme.EventThemeChanged.
	ThemeChangedEvents []string
}

// Manager drives the tray icon from the OS theme. Methods are safe for
// concurrent use; icon updates are serialized and coalesced by applyLoop.
type Manager struct {
	tray *application.SystemTray
	opts Options

	mu        sync.Mutex
	dark      bool
	closed    bool
	buildMenu func() *application.Menu
	lastIcon  []byte

	pendingApply chan struct{}
}

func NewManager(tray *application.SystemTray, opts Options) *Manager {
	dark := false
	if d, ok := accent.DetectDarkMode(); ok {
		dark = d
	}
	m := &Manager{
		tray:         tray,
		opts:         opts,
		dark:         dark,
		pendingApply: make(chan struct{}, 1),
	}
	go m.applyLoop()
	m.requestApply()
	m.applyTooltip()
	return m
}

// Close stops the apply goroutine. The Manager must not be used afterwards.
func (m *Manager) Close() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.closed = true
	m.mu.Unlock()
	close(m.pendingApply)
}

// SetMenu stores the menu builder and applies it. Call RebuildMenu after a
// locale change to re-run the builder with newly localized strings.
func (m *Manager) SetMenu(build func() *application.Menu) {
	m.mu.Lock()
	m.buildMenu = build
	m.mu.Unlock()
	m.RebuildMenu()
}

// RebuildMenu re-runs the stored menu builder and swaps the tray menu.
func (m *Manager) RebuildMenu() {
	if m.tray == nil {
		return
	}
	m.mu.Lock()
	build := m.buildMenu
	m.mu.Unlock()
	if build != nil {
		m.tray.SetMenu(build())
	}
	m.applyTooltip()
}

// SetIcons swaps the icon set, for example from a traygen ForStatus adapter.
func (m *Manager) SetIcons(icons Icons) {
	m.mu.Lock()
	m.opts.Icons = icons
	m.lastIcon = nil
	m.mu.Unlock()
	m.requestApply()
}

func (m *Manager) applyTooltip() {
	if m.tray == nil {
		return
	}
	if m.opts.Tooltip != nil {
		m.tray.SetTooltip(m.opts.Tooltip())
	}
}

func (m *Manager) UpdateTheme(dark bool) {
	m.mu.Lock()
	if m.dark == dark {
		m.mu.Unlock()
		return
	}
	m.dark = dark
	m.mu.Unlock()
	m.requestApply()
}

// RegisterEvents refreshes the icon on OS theme changes.
func (m *Manager) RegisterEvents(app *application.App) {
	// Re-detect from the OS instead of reading the event payload.
	detectAndUpdate := func() {
		defer func() { _ = recover() }()
		dark := false
		if d, ok := accent.DetectDarkMode(); ok {
			dark = d
		} else if a := application.Get(); a != nil && a.Env != nil {
			dark = a.Env.IsDarkMode()
		}
		m.UpdateTheme(dark)
	}

	app.Event.OnApplicationEvent(events.Common.ThemeChanged, func(_ *application.ApplicationEvent) {
		detectAndUpdate()
	})

	for _, name := range m.opts.ThemeChangedEvents {
		app.Event.On(name, func(_ *application.CustomEvent) {
			detectAndUpdate()
		})
	}
}

func (m *Manager) errorf(err error) {
	if err != nil && m.opts.OnError != nil {
		m.opts.OnError(err)
	}
}

// requestApply coalesces rapid calls into at most one extra refresh.
func (m *Manager) requestApply() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return
	}
	select {
	case m.pendingApply <- struct{}{}:
	default:
	}
}

// applyLoop owns all SetIcon calls.
func (m *Manager) applyLoop() {
	defer func() {
		if r := recover(); r != nil {
			m.errorf(recoveredError(r))
		}
	}()

	for range m.pendingApply {
		m.apply()
	}
}

func (m *Manager) apply() {
	defer func() {
		if r := recover(); r != nil {
			m.errorf(recoveredError(r))
		}
	}()

	if m.tray == nil {
		return
	}

	m.mu.Lock()
	icons := m.opts.Icons
	dark := m.dark
	m.mu.Unlock()

	if icons == nil {
		return
	}

	icon, err := icons.Icon(dark)
	if err != nil {
		m.errorf(err)
		return
	}

	// SetIcon dispatches to the GUI main thread; skip identical re-applies.
	m.mu.Lock()
	same := bytes.Equal(m.lastIcon, icon)
	if !same {
		m.lastIcon = icon
	}
	m.mu.Unlock()
	if same {
		return
	}

	m.tray.SetIcon(icon)
}
