//go:build linux

// shell/tray/sni_linux_test.go

package tray

import (
	"bufio"
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 0x10, G: 0x20, B: 0x30, A: 0x80})
	img.SetNRGBA(1, 0, color.NRGBA{R: 0xff, A: 0xff})
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestPixmapIsStraightARGBInNetworkOrder(t *testing.T) {
	px, err := toPixmap(testPNG(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x80, 0x10, 0x20, 0x30, 0xff, 0xff, 0, 0}
	if px.Width != 2 || px.Height != 1 || !bytes.Equal(px.Pixels, want) {
		t.Fatalf("pixmap = %+v", px)
	}
	if _, err := toPixmap([]byte("<svg/>")); err == nil {
		t.Fatal("an SVG was taken as a bitmap")
	}
}

// Setters that run before the item is on the bus must land in what it exports.
func TestStateSetBeforeExportIsPublished(t *testing.T) {
	item := newSNI("app", Options{})
	if err := item.setIcon(testPNG(t)); err != nil {
		t.Fatal(err)
	}
	item.setThemedIcon("/run/user/1000/icons", "latest")
	item.setTooltip("Connected")
	spec := item.itemSpec()[itemIface]
	if spec["IconName"].Value != "latest" || spec["IconThemePath"].Value != "/run/user/1000/icons" {
		t.Fatalf("themed icon = %v %v", spec["IconName"].Value, spec["IconThemePath"].Value)
	}
	if icons := spec["IconPixmap"].Value.([]pixmap); len(icons) != 1 || icons[0].Width != 2 {
		t.Fatalf("pixmap = %v", icons)
	}
	if tip := spec["ToolTip"].Value.(tooltip); tip.Title != "app" || tip.Text != "Connected" {
		t.Fatalf("tooltip = %+v", tip)
	}
}

func TestLayoutListsEntriesAndRevisionMovesOnlyOnChange(t *testing.T) {
	item := newSNI("app", Options{})
	item.setMenu([]Item{{Label: "Open"}, {Separator: true}, {Label: "Quit"}})
	item.setMenu([]Item{{Label: "Open"}, {Separator: true}, {Label: "Quit"}})
	if item.revision != 1 {
		t.Fatalf("revision = %d after an identical rebuild", item.revision)
	}
	root, _ := item.node(0, -1)
	if len(root.Children) != 3 {
		t.Fatalf("children = %d", len(root.Children))
	}
	sep := root.Children[1].Value().(menuNode)
	if sep.Properties["type"].Value() != "separator" {
		t.Fatalf("separator = %+v", sep)
	}
	if _, ok := item.node(4, 0); ok {
		t.Fatal("an entry past the menu exists")
	}
}

// privateBus starts a session bus for this test only.
func privateBus(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("dbus-daemon"); err != nil {
		t.Skip("dbus-daemon is not installed")
	}
	socket := filepath.Join(t.TempDir(), "bus")
	cmd := exec.Command("dbus-daemon", "--session", "--nofork", "--print-address", "--address=unix:path="+socket)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
	address, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(address)
}

// watcher is a StatusNotifierWatcher that records registrations.
type watcher struct{ got chan string }

func (w watcher) RegisterStatusNotifierItem(service string) *dbus.Error {
	w.got <- service
	return nil
}

func TestItemOnTheBus(t *testing.T) {
	address := privateBus(t)
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", address)

	var mu sync.Mutex
	var errs []error
	clicked := make(chan string, 4)
	item := newSNI("app", Options{
		OnClick: func() { clicked <- "icon" },
		OnError: func(err error) { mu.Lock(); errs = append(errs, err); mu.Unlock() },
	})
	if err := item.setIcon(testPNG(t)); err != nil {
		t.Fatal(err)
	}
	item.setThemedIcon("/tmp/theme", "app-light")
	item.setMenu([]Item{{Label: "Open", OnClick: func() { clicked <- "open" }}, {Separator: true}, {Label: "Quit"}})

	// No panel yet: start must succeed and stay quiet.
	if err := item.start(); err != nil {
		t.Fatal(err)
	}
	defer item.close()

	host, err := dbus.Connect(address)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	panel := watcher{got: make(chan string, 2)}
	if err := host.Export(panel, watcherPath, watcherName); err != nil {
		t.Fatal(err)
	}
	if _, err := host.RequestName(watcherName, dbus.NameFlagDoNotQueue); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-panel.got:
		if got != item.busName {
			t.Fatalf("registered %q", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the item did not register with a panel that started after it")
	}

	obj := host.Object(item.busName, itemPath)
	read := func(name string) any {
		t.Helper()
		v, err := obj.GetProperty(itemIface + "." + name)
		if err != nil {
			t.Fatal(name, err)
		}
		return v.Value()
	}
	if read("IconName") != "app-light" || read("IconThemePath") != "/tmp/theme" {
		t.Fatal("themed icon not exported")
	}
	if read("Menu") != menuPath {
		t.Fatal("menu path not exported")
	}

	item.setThemedIcon("/tmp/theme", "app-dark")
	item.setTooltip("Connected")
	if read("IconName") != "app-dark" {
		t.Fatal("themed icon change not exported")
	}
	if tip := read("ToolTip").([]any); tip[3] != "Connected" {
		t.Fatalf("tooltip = %v", tip)
	}

	menu := host.Object(item.busName, menuPath)
	var revision uint32
	var layout menuNode
	if err := menu.Call(menuIface+".GetLayout", 0, int32(0), int32(-1), []string{}).Store(&revision, &layout); err != nil {
		t.Fatal(err)
	}
	if len(layout.Children) != 3 {
		t.Fatalf("layout = %+v", layout)
	}
	if err := menu.Call(menuIface+".Event", 0, int32(1), "clicked", dbus.MakeVariant(""), uint32(0)).Err; err != nil {
		t.Fatal(err)
	}
	if err := obj.Call(itemIface+".Activate", 0, int32(0), int32(0)).Err; err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for len(seen) < 2 {
		select {
		case what := <-clicked:
			seen[what] = true
		case <-time.After(5 * time.Second):
			t.Fatalf("clicks seen: %v", seen)
		}
	}

	item.close()
	var owner string
	if err := host.BusObject().Call("org.freedesktop.DBus.GetNameOwner", 0, item.busName).Store(&owner); err == nil {
		t.Fatalf("bus name still owned by %s after close", owner)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(errs) != 0 {
		t.Fatalf("errors: %v", errs)
	}
}

// A panic in an app's menu action must crash the process, so the crash watcher shows its trace.
func TestMenuActionPanicCrashesTheProcess(t *testing.T) {
	if os.Getenv("S99_TRAY_PANIC") != "" {
		newSNI("app", Options{}).run(func() { panic("boom") })
		time.Sleep(time.Second)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestMenuActionPanicCrashesTheProcess$")
	cmd.Env = append(os.Environ(), "S99_TRAY_PANIC=1")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "panic: boom") {
		t.Fatalf("err = %v, output %q; want the process to die of the panic", err, out)
	}
}

// An OnError that rebuilds the menu runs while a publish fails; it must not wait on the item's lock.
func TestErrorHandlerMayCallBackIntoTheItem(t *testing.T) {
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", privateBus(t))
	menu := []Item{{Label: "Open"}}
	var item *sni
	reported := make(chan error, 8)
	item = newSNI("app", Options{OnError: func(err error) {
		item.setMenu(menu)
		item.setTooltip("")
		reported <- err
	}})
	if err := item.start(); err != nil {
		t.Fatal(err)
	}
	item.mu.Lock()
	conn := item.conn
	item.mu.Unlock()
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		item.setMenu([]Item{{Label: "Open"}, {Label: "Quit"}})
		item.setTooltip("Connected")
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a failing publish deadlocked on the error handler")
	}
	item.close()
	if len(reported) == 0 {
		t.Fatal("the failed publish was not reported")
	}
}
