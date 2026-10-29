// shell/tray/tray.go

// Package tray is a theme-aware tray icon with rebuildable menus; on Linux a StatusNotifierItem.
package tray

import (
	"bytes"
	"io/fs"
	"os"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/smegg99/s99wails/v2/platform/accent"
)

// Icons resolves the tray icon for the current OS theme.
type Icons interface {
	// Icon returns native bitmap bytes for the current OS and panel theme.
	Icon(dark bool) ([]byte, error)
}

// ThemedIcons supplies a Linux SVG icon theme together with the required bitmap fallback.
type ThemedIcons interface {
	Icons
	ThemedIcon(dark bool) (files fs.FS, name string, err error)
}

// IconsFunc adapts a plain function to the Icons interface.
type IconsFunc func(dark bool) ([]byte, error)

func (f IconsFunc) Icon(dark bool) ([]byte, error) { return f(dark) }

// Item is one tray menu entry: a labelled action, or a separator line.
type Item struct {
	Label     string
	Separator bool
	OnClick   func()
}

// Options configures a Manager.
type Options struct {
	Icons Icons
	// TemplateIcon asks macOS to tint black-and-transparent bitmap artwork.
	TemplateIcon bool
	// Tooltip is re-applied on every RebuildMenu, so locale changes pick it up.
	Tooltip func() string
	// OnClick runs on a primary click; the secondary click opens the menu.
	OnClick func()
	// OnError receives icon, bus and menu failures and recovered panics.
	// Optional; errors are dropped when nil.
	OnError func(error)
	// ThemeChangedEvents lists extra custom events that trigger dark-mode
	// re-detection, such as theme.EventThemeChanged.
	ThemeChangedEvents []string
}

func (o Options) report(err error) {
	if err != nil && o.OnError != nil {
		o.OnError(err)
	}
}

// reportPanic hands a recovered panic to OnError, or raises it again for the crash watcher when there is none.
func (o Options) reportPanic(r any) {
	if o.OnError == nil {
		panic(r)
	}
	o.OnError(recoveredError(r))
}

// closeWait bounds Close: a tray stuck in a main-thread call after app.Run returned must not hold exit.
var closeWait = 2 * time.Second

// backend is the platform tray the Manager drives.
type backend interface {
	setIcon(icon []byte) error
	setTooltip(text string)
	setMenu(items []Item)
	close()
}

// themedBackend is a tray the desktop can draw from an icon theme by name.
type themedBackend interface {
	setThemedIcon(themePath, name string)
}

// Manager drives the tray icon, tooltip and menu; safe for concurrent use, with icon updates coalesced by applyLoop.
type Manager struct {
	tray backend
	opts Options

	mu        sync.Mutex
	icons     Icons
	dark      bool
	closed    bool
	buildMenu func() []Item
	lastIcon  []byte
	themeDir  string

	pendingApply chan struct{}
	// stopped closes once applyLoop has closed the tray and removed the theme directory.
	stopped chan struct{}
}

// NewManager creates the tray with its first icon for the OS theme; app names the Linux StatusNotifierItem.
func NewManager(app *application.App, opts Options) *Manager {
	return start(newPlatform(app, opts), opts)
}

func start(tray backend, opts Options) *Manager {
	dark := false
	if d, ok := accent.DetectDarkMode(); ok {
		dark = d
	}
	m := &Manager{
		tray:         tray,
		opts:         opts,
		icons:        opts.Icons,
		dark:         dark,
		pendingApply: make(chan struct{}, 1),
		stopped:      make(chan struct{}),
	}
	go m.applyLoop()
	m.requestApply()
	m.applyTooltip()
	return m
}

// Close stops the apply goroutine and waits, at most closeWait, for it to remove the tray; never call it from OnError.
func (m *Manager) Close() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		select {
		case <-m.stopped:
		case <-time.After(closeWait):
		}
		return
	}
	m.closed = true
	m.mu.Unlock()
	close(m.pendingApply)
	select {
	case <-m.stopped:
	case <-time.After(closeWait):
	}
}

// SetMenu stores and applies the menu builder; RebuildMenu re-runs it after a locale change.
func (m *Manager) SetMenu(build func() []Item) {
	m.mu.Lock()
	m.buildMenu = build
	m.mu.Unlock()
	m.RebuildMenu()
}

// RebuildMenu re-runs the stored menu builder and swaps the tray menu.
func (m *Manager) RebuildMenu() {
	m.mu.Lock()
	build := m.buildMenu
	m.mu.Unlock()
	if build != nil {
		m.tray.setMenu(build())
	}
	m.applyTooltip()
}

// SetIcons swaps the icon set, for example from a traygen ForStatus adapter.
func (m *Manager) SetIcons(icons Icons) {
	m.mu.Lock()
	m.icons = icons
	m.lastIcon = nil
	m.mu.Unlock()
	m.requestApply()
}

func (m *Manager) applyTooltip() {
	if m.opts.Tooltip != nil {
		m.tray.setTooltip(m.opts.Tooltip())
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

// applyLoop owns all icon updates.
func (m *Manager) applyLoop() {
	defer close(m.stopped)
	defer func() {
		if r := recover(); r != nil {
			m.opts.reportPanic(r)
		}
		m.tray.close()
		if m.themeDir != "" {
			m.opts.report(os.RemoveAll(m.themeDir))
		}
	}()

	for range m.pendingApply {
		// A request buffered before Close would call into a main loop that may have stopped.
		if m.isClosed() {
			continue
		}
		m.apply()
	}
}

func (m *Manager) apply() {
	defer func() {
		if r := recover(); r != nil {
			m.opts.reportPanic(r)
		}
	}()

	m.mu.Lock()
	icons := m.icons
	dark := m.dark
	m.mu.Unlock()

	if icons == nil {
		return
	}

	icon, err := icons.Icon(dark)
	if err != nil {
		m.opts.report(err)
		return
	}

	m.mu.Lock()
	same := bytes.Equal(m.lastIcon, icon)
	if !same {
		m.lastIcon = icon
	}
	m.mu.Unlock()
	if !same {
		m.opts.report(m.tray.setIcon(icon))
	}
	if themed, ok := m.tray.(themedBackend); ok {
		m.applyThemedIcon(themed, icons, dark)
	}
}

func (m *Manager) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}
