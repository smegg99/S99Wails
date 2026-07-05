// config.go
package configservice

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/joho/godotenv"
	"github.com/smegg99/s99config"
)

const (
	configFileName = "config.json"
	keyringService = "YourApp" // CHANGE ME
)

var (
	Global Config

	mu          sync.RWMutex
	loader      *s99config.Loader
	resolvedCfg string
)

// Initialize loads .env if present, validates the config against the embedded
// schema (writing a default when missing) and returns the config path.
func Initialize() (string, error) {
	_ = godotenv.Load()

	path := resolveConfigPath()
	dir := filepath.Dir(path)

	l, err := s99config.New(
		definition,
		s99config.WithReferences(s99config.ReferenceOptions{
			ConfigDir:      dir,
			DataDir:        dir,
			KeyringService: keyringService,
		}),
	)
	if err != nil {
		return "", fmt.Errorf("compile config schema: %w", err)
	}

	if _, statErr := os.Stat(path); errors.Is(statErr, fs.ErrNotExist) {
		if err := l.WriteDefaults(path); err != nil {
			return path, fmt.Errorf("write default config %s: %w", path, err)
		}
	}

	if err := l.Load(path); err != nil {
		return path, fmt.Errorf("load config %s: %w", path, err)
	}

	mu.Lock()
	loader = l
	resolvedCfg = path
	mu.Unlock()

	return path, decode()
}

// Get returns a copy of the loaded configuration.
func Get() Config {
	mu.RLock()
	defer mu.RUnlock()
	return Global
}

func GetConfigPath() string {
	mu.RLock()
	defer mu.RUnlock()
	return resolvedCfg
}

// decode refreshes Global from the loader's current state.
func decode() error {
	mu.RLock()
	l := loader
	mu.RUnlock()
	if l == nil {
		return fmt.Errorf("config not initialized")
	}

	var next Config
	if err := l.Decode(&next); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}

	mu.Lock()
	Global = next
	mu.Unlock()
	return nil
}

// resolveConfigPath returns the config file path from CONFIG_PATH (file or
// directory) or the portable default in the working directory.
func resolveConfigPath() string {
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return filepath.Join(p, configFileName)
		}
		return p
	}
	return configFileName
}
