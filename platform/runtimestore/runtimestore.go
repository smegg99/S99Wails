// platform/runtimestore/runtimestore.go

// Package runtimestore keeps small per-user values under $XDG_RUNTIME_DIR, cleared at logout.
package runtimestore

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
)

// ErrUnavailable is a system with no usable runtime directory: no XDG_RUNTIME_DIR,
// or one that is not a private directory of this user.
var ErrUnavailable = errors.New("runtimestore: no private runtime directory")

// ErrNotFound is a key with no value stored.
var ErrNotFound = errors.New("runtimestore: not found")

var appName = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// Store is one application's directory inside the runtime directory.
type Store struct {
	dir string
}

// Open returns the store for app under $XDG_RUNTIME_DIR, creating it 0700.
func Open(app string) (*Store, error) {
	return OpenIn(os.Getenv("XDG_RUNTIME_DIR"), app)
}

// OpenIn is Open against an explicit runtime directory, for tests and harnesses.
func OpenIn(runtimeDir, app string) (*Store, error) {
	if !appName.MatchString(app) {
		return nil, fmt.Errorf("runtimestore: invalid app name %q", app)
	}
	if runtimeDir == "" || !filepath.IsAbs(runtimeDir) {
		return nil, ErrUnavailable
	}
	info, err := os.Stat(runtimeDir)
	if err != nil || !info.IsDir() || !private(info) {
		return nil, ErrUnavailable
	}
	dir := filepath.Join(runtimeDir, app)
	if err := os.Mkdir(dir, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, fmt.Errorf("runtimestore: %w", err)
	}
	info, err = os.Lstat(dir)
	if err != nil {
		return nil, fmt.Errorf("runtimestore: %w", err)
	}
	if !info.IsDir() || !private(info) {
		return nil, fmt.Errorf("runtimestore: %s is not a private directory", dir)
	}
	return &Store{dir: dir}, nil
}

// Dir reports the directory the values live in.
func (s *Store) Dir() string { return s.dir }

// Get reports the value stored under key, or ErrNotFound.
func (s *Store) Get(key string) (string, error) {
	data, err := os.ReadFile(s.path(key))
	if errors.Is(err, fs.ErrNotExist) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("runtimestore: %w", err)
	}
	return string(data), nil
}

// Set stores value under key, readable by this user only, replacing it whole.
func (s *Store) Set(key, value string) error {
	temp, err := os.CreateTemp(s.dir, ".set-*")
	if err != nil {
		return fmt.Errorf("runtimestore: %w", err)
	}
	defer os.Remove(temp.Name())
	if _, err := temp.WriteString(value); err != nil {
		temp.Close()
		return fmt.Errorf("runtimestore: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("runtimestore: %w", err)
	}
	if err := os.Rename(temp.Name(), s.path(key)); err != nil {
		return fmt.Errorf("runtimestore: %w", err)
	}
	return nil
}

// Delete removes the value under key; a key with no value is not an error.
func (s *Store) Delete(key string) error {
	if err := os.Remove(s.path(key)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("runtimestore: %w", err)
	}
	return nil
}

// path names a key's file by its hash, so any key is a safe file name.
func (s *Store) path(key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(s.dir, hex.EncodeToString(sum[:]))
}
