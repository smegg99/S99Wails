// platform/runtimestore/runtimestore_test.go

package runtimestore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func runtimeDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestAValueSurvivesAReopen(t *testing.T) {
	dir := runtimeDir(t)
	store, err := OpenIn(dir, "goptivum-desktop")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := store.Set("session:http://127.0.0.1:8833", "token"); err != nil {
		t.Fatalf("set: %v", err)
	}
	again, err := OpenIn(dir, "goptivum-desktop")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	got, err := again.Get("session:http://127.0.0.1:8833")
	if err != nil || got != "token" {
		t.Fatalf("get = %q, %v; want token", got, err)
	}
}

func TestValuesAreThisUsersOnly(t *testing.T) {
	store, err := OpenIn(runtimeDir(t), "app")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := store.Set("key", "value"); err != nil {
		t.Fatalf("set: %v", err)
	}
	dirInfo, err := os.Stat(store.Dir())
	if err != nil {
		t.Fatal(err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0o700 {
		t.Errorf("dir mode = %o, want 700", perm)
	}
	entries, err := os.ReadDir(store.Dir())
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries = %v, %v; want one file", entries, err)
	}
	info, err := entries[0].Info()
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode = %o, want 600", perm)
	}
}

func TestAMissingKeyIsNotFound(t *testing.T) {
	store, err := OpenIn(runtimeDir(t), "app")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := store.Get("nothing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("get = %v, want ErrNotFound", err)
	}
	if err := store.Delete("nothing"); err != nil {
		t.Errorf("delete of a missing key = %v, want nil", err)
	}
}

func TestDeleteRemovesTheValue(t *testing.T) {
	store, err := OpenIn(runtimeDir(t), "app")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := store.Set("key", "value"); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete("key"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Get("key"); !errors.Is(err, ErrNotFound) {
		t.Errorf("get after delete = %v, want ErrNotFound", err)
	}
}

func TestNoUsableRuntimeDirectoryIsUnavailable(t *testing.T) {
	open := runtimeDir(t)
	if err := os.Chmod(open, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, dir := range map[string]string{
		"unset":    "",
		"relative": "run/user",
		"missing":  filepath.Join(t.TempDir(), "absent"),
		"shared":   open,
	} {
		if _, err := OpenIn(dir, "app"); !errors.Is(err, ErrUnavailable) {
			t.Errorf("%s: err = %v, want ErrUnavailable", name, err)
		}
	}
}

func TestAnAppNameThatIsNotAPlainNameIsRefused(t *testing.T) {
	for _, name := range []string{"", "../escape", "a/b", "Upper"} {
		if _, err := OpenIn(runtimeDir(t), name); err == nil {
			t.Errorf("%q was accepted", name)
		}
	}
}
