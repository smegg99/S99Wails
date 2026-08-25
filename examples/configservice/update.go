// examples/configservice/update.go

package configservice

import (
	"fmt"
	"sync"
)

func SetConfig(c Config) error {
	return patch(c)
}

// SetPreferences updates only the preferences section.
func SetPreferences(p Preferences) error {
	return patch(struct {
		Preferences Preferences `json:"preferences"`
	}{Preferences: p})
}

// SetLoggerConfig updates only the logger section.
func SetLoggerConfig(lc LoggerConfig) error {
	return patch(struct {
		Logger LoggerConfig `json:"logger"`
	}{Logger: lc})
}

// SetBackend updates only the backend section.
func SetBackend(b BackendConfig) error {
	return patch(struct {
		Backend BackendConfig `json:"backend"`
	}{Backend: b})
}

// patchMu serializes Set* calls; each patch is a read-modify-write of the file.
var patchMu sync.Mutex

// patch deep-merges v into the on-disk config, validates it and refreshes Global.
func patch(v any) error {
	patchMu.Lock()
	defer patchMu.Unlock()

	mu.RLock()
	l := loader
	mu.RUnlock()
	if l == nil {
		return fmt.Errorf("config not initialized")
	}

	if err := l.Patch(v); err != nil {
		return fmt.Errorf("patch config: %w", err)
	}

	return decode()
}
