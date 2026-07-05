// locale_test.go
package locale

import (
	"testing"
	"testing/fstest"
)

func testFS() fstest.MapFS {
	return fstest.MapFS{
		"en.json": {Data: []byte(`{"tray.show": "Show", "tray.quit": "Quit"}`)},
		"pl.json": {Data: []byte(`{"tray.show": "Pokaż"}`)},
	}
}

func TestNewAndT(t *testing.T) {
	l, err := New(testFS(), "en")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := l.T("tray.show"); got != "Show" {
		t.Errorf("T(tray.show) = %q, want Show", got)
	}
}

func TestSetLocaleAndFallback(t *testing.T) {
	l, err := New(testFS(), "en")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	l.SetLocale("pl")
	if got := l.T("tray.show"); got != "Pokaż" {
		t.Errorf("pl T(tray.show) = %q, want Pokaż", got)
	}
	// Missing in pl -> falls back to the default locale.
	if got := l.T("tray.quit"); got != "Quit" {
		t.Errorf("pl T(tray.quit) = %q, want Quit (fallback)", got)
	}
	// Region matching: pl-PL resolves to pl.
	l.SetLocale("pl-PL")
	if got := l.T("tray.show"); got != "Pokaż" {
		t.Errorf("pl-PL T(tray.show) = %q, want Pokaż", got)
	}
}

func TestUnknownMessageReturnsID(t *testing.T) {
	l, err := New(testFS(), "en")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := l.T("nope.missing"); got != "nope.missing" {
		t.Errorf("T(missing) = %q, want the ID back", got)
	}
}

func TestNewEmptyFS(t *testing.T) {
	if _, err := New(fstest.MapFS{}, "en"); err == nil {
		t.Error("New with no locale files should fail")
	}
}

func TestAvailable(t *testing.T) {
	l, err := New(testFS(), "en")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got := l.Available()
	if len(got) != 2 || got[0] != "en" || got[1] != "pl" {
		t.Errorf("Available() = %v, want [en pl]", got)
	}
}

func TestOnChange(t *testing.T) {
	l, err := New(testFS(), "en")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	fired := 0
	l.OnChange(func() { fired++ })
	l.SetLocale("pl")
	l.SetLocale("en")
	if fired != 2 {
		t.Errorf("OnChange fired %d times, want 2", fired)
	}
}

func TestTfAndTn(t *testing.T) {
	fs := testFS()
	fs["en.json"].Data = []byte(`{
		"greet": "Hello {{.Name}}",
		"items": {"one": "{{.PluralCount}} item", "other": "{{.PluralCount}} items"}
	}`)
	l, err := New(fs, "en")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := l.Tf("greet", map[string]any{"Name": "Karol"}); got != "Hello Karol" {
		t.Errorf("Tf = %q", got)
	}
	if got := l.Tn("items", 1, nil); got != "1 item" {
		t.Errorf("Tn(1) = %q", got)
	}
	if got := l.Tn("items", 5, nil); got != "5 items" {
		t.Errorf("Tn(5) = %q", got)
	}
}

func TestTnDoesNotMutateCallerMap(t *testing.T) {
	fs := testFS()
	fs["en.json"].Data = []byte(`{"items": {"one": "{{.PluralCount}} item", "other": "{{.PluralCount}} items"}}`)
	l, err := New(fs, "en")
	if err != nil {
		t.Fatal(err)
	}
	shared := map[string]any{"Name": "x"}
	_ = l.Tn("items", 3, shared)
	if _, ok := shared["PluralCount"]; ok {
		t.Error("Tn mutated the caller's map")
	}
}
