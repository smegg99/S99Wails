// shell/tray/theme_test.go

package tray

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"
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
	var reported error
	tray := &fakeTray{}
	m := &Manager{tray: tray, icons: failingThemedIcons{}, opts: Options{OnError: func(err error) { reported = err }}}
	defer func() {
		if m.themeDir != "" {
			os.RemoveAll(m.themeDir)
		}
	}()
	m.apply()
	if string(tray.icon) != "bitmap" {
		t.Fatal("bitmap fallback was not applied")
	}
	if tray.themePath != "" || tray.iconName != "" {
		t.Fatalf("broken theme published: %q %q", tray.themePath, tray.iconName)
	}
	if reported == nil {
		t.Fatal("invalid SVG theme was not reported")
	}
}

type goodThemedIcons struct{}

func (goodThemedIcons) Icon(bool) ([]byte, error) { return []byte("bitmap"), nil }
func (goodThemedIcons) ThemedIcon(dark bool) (fs.FS, string, error) {
	name := "app-light"
	if dark {
		name = "app-dark"
	}
	return fstest.MapFS{name + ".svg": {Data: []byte("<svg/>")}}, name, nil
}

func TestThemedIconIsWrittenAndPublishedBesideTheBitmap(t *testing.T) {
	tray := &fakeTray{}
	m := &Manager{tray: tray, dark: true, icons: goodThemedIcons{}}
	defer os.RemoveAll(m.themeDir)
	m.apply()
	if string(tray.icon) != "bitmap" || tray.iconName != "app-dark" || tray.themePath != m.themeDir {
		t.Fatalf("published %q %q %q", tray.icon, tray.iconName, tray.themePath)
	}
	if _, err := os.Stat(filepath.Join(m.themeDir, "app-dark.svg")); err != nil {
		t.Fatal(err)
	}
}

// Close returns only after the apply goroutine closed the tray and removed the theme it wrote.
func TestCloseWaitsForTheThemeDirectoryToGo(t *testing.T) {
	tray := &fakeTray{}
	m := start(tray, Options{Icons: goodThemedIcons{}})
	eventually(t, "themed icon not applied", func() bool {
		tray.mu.Lock()
		defer tray.mu.Unlock()
		return tray.themePath != ""
	})
	tray.mu.Lock()
	dir := tray.themePath
	tray.mu.Unlock()
	m.Close()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("theme directory still there after Close: %v", err)
	}
	if !tray.isClosed() {
		t.Fatal("tray still open after Close")
	}
}
