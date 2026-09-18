// shell/windowstate/filestore.go

package windowstate

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// FileStore keeps the geometry in one JSON file, created with its directory on
// first save.
func FileStore(path string) Store { return fileStore(path) }

type fileStore string

// Load reports ok=false for a missing or unreadable file, so a bad one is
// replaced on the next save rather than warned about on every start.
func (f fileStore) Load() (State, bool, error) {
	var state State
	data, err := os.ReadFile(string(f))
	if errors.Is(err, os.ErrNotExist) {
		return state, false, nil
	}
	if err != nil {
		return state, false, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, false, nil
	}
	return state, true, nil
}

// Save writes through a temporary file in the same directory, so an interrupted
// save leaves the previous geometry rather than half of a new one.
func (f fileStore) Save(state State) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	path := string(f)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".window-*")
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
	return os.Rename(file.Name(), path)
}
