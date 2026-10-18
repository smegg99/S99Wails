// shell/editkeys/editkeys_test.go
package editkeys

import "testing"

// ctrl+Z undoes and ctrl+shift+Z redoes; Z alone, or with Alt or Super, is the page's.
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
	}
	for _, c := range cases {
		if got := actionOf(c.key, c.control, c.shift, c.other); got != c.want {
			t.Errorf("actionOf(%q, ctrl=%v, shift=%v, other=%v) = %q, want %q", rune(c.key), c.control, c.shift, c.other, got, c.want)
		}
	}
}

// ctrl+V, ctrl+shift+V and shift+Insert paste; V alone, ctrl+Insert (copy) or with Alt do not.
func TestThePasteKeys(t *testing.T) {
	cases := []struct {
		key                   uint
		control, shift, other bool
		want                  bool
	}{
		{'v', true, false, false, true},
		{'v', true, true, false, true},
		{keyInsert, false, true, false, true},
		{'v', false, false, false, false},
		{'v', true, false, true, false},
		{keyInsert, true, false, false, false},
		{keyInsert, true, true, false, false},
		{'c', true, false, false, false},
	}
	for _, c := range cases {
		if got := pastes(c.key, c.control, c.shift, c.other); got != c.want {
			t.Errorf("pastes(%#x, ctrl=%v, shift=%v, other=%v) = %v, want %v", c.key, c.control, c.shift, c.other, got, c.want)
		}
	}
}
