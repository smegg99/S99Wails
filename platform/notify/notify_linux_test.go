// platform/notify/notify_linux_test.go
package notify_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/smegg99/s99wails/v2/platform/notify"
)

// server is a notification server: it keeps what it was asked to show.
type server struct{ shown chan [3]string }

func (s *server) Notify(app string, _ uint32, _ string, title, body string, _ []string,
	_ map[string]dbus.Variant, _ int32) (uint32, *dbus.Error) {
	s.shown <- [3]string{app, title, body}
	return 1, nil
}

// busConfig is a session bus that starts no service on demand, so no
// notification server installed on this machine is ever started by a test.
const busConfig = `<!DOCTYPE busconfig PUBLIC "-//freedesktop//DTD D-Bus Bus Configuration 1.0//EN"
 "http://www.freedesktop.org/standards/dbus/1.0/busconfig.dtd">
<busconfig>
  <type>session</type>
  <listen>unix:dir=%s</listen>
  <auth>EXTERNAL</auth>
  <policy context="default">
    <allow send_destination="*" eavesdrop="true"/>
    <allow eavesdrop="true"/>
    <allow own="*"/>
  </policy>
</busconfig>`

// bus starts a private session bus and points this test at it.
func bus(t *testing.T) string {
	t.Helper()
	daemon, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("no dbus-daemon")
	}
	dir := t.TempDir()
	config := filepath.Join(dir, "bus.conf")
	if err := os.WriteFile(config, []byte(fmt.Sprintf(busConfig, dir)), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(daemon, "--config-file="+config, "--nofork", "--print-address")
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	address, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	address = strings.TrimSpace(address)
	// Never the desktop's own bus: an empty address would fall back to it.
	if !strings.HasPrefix(address, "unix:path="+dir) {
		t.Fatalf("private bus address = %q", address)
	}
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", address)
	t.Setenv("XDG_RUNTIME_DIR", dir)
	return address
}

// A notification reaches the desktop's notification server, from the application it names.
func TestANotificationReachesTheNotificationServer(t *testing.T) {
	bus(t)
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	srv := &server{shown: make(chan [3]string, 1)}
	if err := conn.Export(srv, "/org/freedesktop/Notifications", "org.freedesktop.Notifications"); err != nil {
		t.Fatal(err)
	}
	if reply, err := conn.RequestName("org.freedesktop.Notifications", dbus.NameFlagDoNotQueue); err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		t.Fatalf("RequestName = %v, %v", reply, err)
	}

	desktop := notify.New("Goptivum")
	t.Cleanup(desktop.Close)
	if err := desktop.Send(t.Context(), "dedicated-arrango is not answering", "A new solve is refused."); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-srv.shown:
		if got != [3]string{"Goptivum", "dedicated-arrango is not answering", "A new solve is refused."} {
			t.Errorf("shown = %q", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("nothing was shown")
	}
}

// Send returns an error, without hanging, when no notification server is present.
func TestNoNotificationServerIsAnError(t *testing.T) {
	bus(t)
	desktop := notify.New("Goptivum")
	t.Cleanup(desktop.Close)
	if err := desktop.Send(t.Context(), "title", "body"); err == nil {
		t.Fatal("Send without a notification server = nil, want an error")
	}
}

func serve(t *testing.T, implementation any) {
	t.Helper()
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if err := conn.Export(implementation, "/org/freedesktop/Notifications", "org.freedesktop.Notifications"); err != nil {
		t.Fatal(err)
	}
	if reply, err := conn.RequestName("org.freedesktop.Notifications", dbus.NameFlagDoNotQueue); err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		t.Fatalf("RequestName = %v, %v", reply, err)
	}
}

func TestSendRecoversWhenNotificationServiceAppears(t *testing.T) {
	bus(t)
	desktop := notify.New("Example")
	t.Cleanup(desktop.Close)
	if err := desktop.Send(t.Context(), "missing", "body"); err == nil {
		t.Fatal("missing service succeeded")
	}
	srv := &server{shown: make(chan [3]string, 16)}
	serve(t, srv)
	var callers sync.WaitGroup
	for range 16 {
		callers.Go(func() {
			if err := desktop.Send(t.Context(), "ready", "body"); err != nil {
				t.Error(err)
			}
		})
	}
	callers.Wait()
	if len(srv.shown) != 16 {
		t.Fatalf("received %d notifications", len(srv.shown))
	}
	desktop.Close()
	if err := desktop.Send(t.Context(), "closed", "body"); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("Send after Close = %v", err)
	}
}

type blockedServer struct{ release chan struct{} }

func (s *blockedServer) Notify(_ string, _ uint32, _ string, _, _ string, _ []string,
	_ map[string]dbus.Variant, _ int32) (uint32, *dbus.Error) {
	<-s.release
	return 1, nil
}

func TestSendDeadlineAndCancellation(t *testing.T) {
	bus(t)
	srv := &blockedServer{release: make(chan struct{})}
	t.Cleanup(func() { close(srv.release) })
	serve(t, srv)
	desktop := notify.New("Example")
	t.Cleanup(desktop.Close)
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	if err := desktop.Send(ctx, "blocked", "body"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Send deadline = %v", err)
	}
	ctx, cancel = context.WithCancel(t.Context())
	cancel()
	if err := desktop.Send(ctx, "cancelled", "body"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Send cancellation = %v", err)
	}
}

func TestDeadlineAlsoBoundsBusAuthentication(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "bus")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+socket)
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			accepted <- conn
		}
	}()
	desktop := notify.New("Example")
	t.Cleanup(desktop.Close)
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- desktop.Send(ctx, "title", "body") }()
	select {
	case conn := <-accepted:
		t.Cleanup(func() { _ = conn.Close() })
	case <-time.After(time.Second):
		t.Fatal("no bus connection")
	}
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("silent bus accepted a notification")
		}
	case <-time.After(time.Second):
		t.Fatal("bus authentication ignored the deadline")
	}
}

func TestDefaultActionReachesOnlyItsNotification(t *testing.T) {
	bus(t)
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	srv := &server{shown: make(chan [3]string, 1)}
	if err := conn.Export(srv, pathForTest, "org.freedesktop.Notifications"); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.RequestName("org.freedesktop.Notifications", dbus.NameFlagDoNotQueue); err != nil {
		t.Fatal(err)
	}
	desktop := notify.New("test")
	t.Cleanup(desktop.Close)
	activated := make(chan struct{}, 2)
	if err := desktop.SendAction(t.Context(), "title", "body", "Open", func() { activated <- struct{}{} }); err != nil {
		t.Fatal(err)
	}
	if err := conn.Emit(pathForTest, "org.freedesktop.Notifications.ActionInvoked", uint32(2), "default"); err != nil {
		t.Fatal(err)
	}
	if err := conn.Emit(pathForTest, "org.freedesktop.Notifications.ActionInvoked", uint32(1), "default"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-activated:
	case <-time.After(5 * time.Second):
		t.Fatal("notification action was not dispatched")
	}
}

const pathForTest dbus.ObjectPath = "/org/freedesktop/Notifications"
