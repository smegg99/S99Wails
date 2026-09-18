// shell/windowstate/filestore_test.go

package windowstate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "window.json")
	store := FileStore(path)

	if _, ok, err := store.Load(); err != nil || ok {
		t.Fatalf("empty store: ok=%v err=%v", ok, err)
	}
	want := State{X: 3, Y: 4, Width: 640, Height: 480, Maximised: true}
	if err := store.Save(want); err != nil {
		t.Fatal(err)
	}
	got, ok, err := store.Load()
	if err != nil || !ok || got != want {
		t.Fatalf("got %+v ok=%v err=%v, want %+v", got, ok, err, want)
	}
	if info, err := os.Stat(filepath.Dir(path)); err != nil || info.Mode().Perm() != 0o700 {
		t.Errorf("directory mode = %v (err %v), want 0700", info.Mode().Perm(), err)
	}
}

// A state file that will not parse counts as nothing saved, so the next save
// replaces it instead of warning on every start.
func TestFileStoreUnreadableCountsAsUnsaved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "window.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	state, ok, err := FileStore(path).Load()
	if err != nil || ok || state != (State{}) {
		t.Fatalf("got %+v ok=%v err=%v, want zero state and ok=false", state, ok, err)
	}
}
