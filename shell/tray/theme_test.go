package tray

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestWriteThemePublishesFilesWithoutRewritingUnchangedArtwork(t *testing.T) {
	dir := t.TempDir()
	files := fstest.MapFS{
		"index.theme":             {Data: []byte("[Icon Theme]\nName=tray\n")},
		"scalable/status/cat.svg": {Data: []byte("<svg/>")},
	}
	if err := writeTheme(files, dir); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "scalable/status/cat.svg")
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && before.Mode().Perm() != 0600 {
		t.Fatalf("icon mode = %v", before.Mode())
	}
	if err := writeTheme(files, dir); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("unchanged icon was replaced")
	}
	files["scalable/status/cat.svg"].Data = []byte("<svg id=\"new\"/>")
	if err := writeTheme(files, dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != string(files["scalable/status/cat.svg"].Data) {
		t.Fatalf("updated artwork = %q, %v", data, err)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary files left behind: %v, %v", entries, err)
	}
}

func TestWriteThemeRejectsNilAndSymlinks(t *testing.T) {
	for name, files := range map[string]fs.FS{
		"nil":     nil,
		"symlink": fstest.MapFS{"icon.svg": {Mode: fs.ModeSymlink, Data: []byte("/somewhere")}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := writeTheme(files, t.TempDir()); err == nil {
				t.Fatal("invalid theme accepted")
			}
		})
	}
}

type failingThemedIcons struct{}

func (failingThemedIcons) Icon(bool) ([]byte, error) { return []byte("bitmap"), nil }
func (failingThemedIcons) ThemedIcon(bool) (fs.FS, string, error) {
	return fstest.MapFS{"bad.svg": {Mode: fs.ModeSymlink}}, "bad", nil
}

func TestThemeFailureRetainsBitmapAndReportsError(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux icon-theme API")
	}
	var reported error
	m := &Manager{tray: &application.SystemTray{}, opts: Options{Icons: failingThemedIcons{}, OnError: func(err error) { reported = err }}}
	defer func() {
		if m.themeDir != "" {
			os.RemoveAll(m.themeDir)
		}
	}()
	m.apply()
	if string(m.lastIcon) != "bitmap" {
		t.Fatal("bitmap fallback was not applied")
	}
	if reported == nil {
		t.Fatal("invalid SVG theme was not reported")
	}
}

func TestCloseRemovesOwnedThemeDirectory(t *testing.T) {
	dir, err := os.MkdirTemp(t.TempDir(), "tray-")
	if err != nil {
		t.Fatal(err)
	}
	m := &Manager{themeDir: dir, pendingApply: make(chan struct{}, 1)}
	go m.applyLoop()
	m.Close()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("Close left the owned icon theme behind")
}
