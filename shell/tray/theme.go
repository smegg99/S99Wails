// shell/tray/theme.go

package tray

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func (m *Manager) applyThemedIcon(icons Icons, dark bool) {
	themed, ok := icons.(ThemedIcons)
	if !ok {
		m.tray.SetThemedIcon("", "")
		return
	}
	files, name, err := themed.ThemedIcon(dark)
	if err != nil {
		m.tray.SetThemedIcon("", "")
		m.errorf(err)
		return
	}
	if m.themeDir == "" {
		m.themeDir, err = os.MkdirTemp("", fmt.Sprintf("s99wails-tray-%d-", os.Getpid()))
		if err != nil {
			m.tray.SetThemedIcon("", "")
			m.errorf(err)
			return
		}
	}
	if err := writeTheme(files, m.themeDir); err != nil {
		m.tray.SetThemedIcon("", "")
		m.errorf(err)
		return
	}
	m.tray.SetThemedIcon(m.themeDir, name)
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
