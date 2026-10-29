// shell/editkeys/editkeys_test.go
package editkeys

import (
	"errors"
	"testing"
)

// ctrl+Z undoes and ctrl+shift+Z redoes; Z alone, with Alt or Super, or a paste key is the page's.
func TestOnlyTheTwoChordsAreTaken(t *testing.T) {
	cases := []struct {
		key                   uint
		control, shift, other bool
		want                  string
	}{
		{'z', true, false, false, "undo"},
		{'z', true, true, false, "redo"},
		{'z', false, false, false, ""},
		{'z', false, true, false, ""},
		{'z', true, false, true, ""},
		{'y', true, false, false, ""},
		{'v', true, false, false, ""},
		{'v', true, true, false, ""},
		{0xff63, false, true, false, ""},
	}
	for _, c := range cases {
		if got := actionOf(c.key, c.control, c.shift, c.other); got != c.want {
			t.Errorf("actionOf(%#x, ctrl=%v, shift=%v, other=%v) = %q, want %q", c.key, c.control, c.shift, c.other, got, c.want)
		}
	}
}

// Watch(win, nil) is allowed: with no handler a failure is dropped instead of crashing the app.
func TestNilErrorHandlerDropsTheFailure(t *testing.T) {
	report(nil, errors.New("editkeys: no GTK window"))
	var got error
	report(func(err error) { got = err }, errors.New("boom"))
	if got == nil || got.Error() != "boom" {
		t.Fatalf("handler got %v", got)
	}
}
