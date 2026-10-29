// shell/tray/theme.go

package tray

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// applyThemedIcon publishes the SVG theme beside the bitmap, or clears it so the host draws the bitmap.
func (m *Manager) applyThemedIcon(tray themedBackend, icons Icons, dark bool) {
	themed, ok := icons.(ThemedIcons)
	if !ok {
		tray.setThemedIcon("", "")
		return
	}
	files, name, err := themed.ThemedIcon(dark)
	if err != nil {
		tray.setThemedIcon("", "")
		m.opts.report(err)
		return
	}
	if m.themeDir == "" {
		m.themeDir, err = os.MkdirTemp("", fmt.Sprintf("s99wails-tray-%d-", os.Getpid()))
		if err != nil {
			tray.setThemedIcon("", "")
			m.opts.report(err)
			return
		}
	}
	if err := writeTheme(files, m.themeDir); err != nil {
		tray.setThemedIcon("", "")
		m.opts.report(err)
		return
	}
	tray.setThemedIcon(m.themeDir, name)
}

func writeTheme(files fs.FS, dir string) error {
	if files == nil {
		return fmt.Errorf("tray icon theme is nil")
	}
	return fs.WalkDir(files, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dir, filepath.FromSlash(path))
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("tray icon theme contains symlink %s", path)
		}
		data, err := fs.ReadFile(files, path)
		if err != nil {
			return err
		}
		if existing, err := os.ReadFile(target); err == nil && bytes.Equal(existing, data) {
			return nil
		}
		// Hosts may read icons while a theme changes; publish complete files.
		file, err := os.CreateTemp(filepath.Dir(target), ".icon-*")
		if err != nil {
			return err
		}
		defer os.Remove(file.Name())
		if _, err := file.Write(data); err != nil {
			file.Close()
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
		return os.Rename(file.Name(), target)
	})
}
