// accent/helpers_test.go

package accent

import (
	"image/color"
	"testing"
)

func TestNormalizeToHex(t *testing.T) {
	cases := map[string]string{
		"#aabbcc":              "#aabbcc",
		"rgb(255,0,128)":       "#ff0080",
		"rgba(255,0,128,0.5)":  "#ff0080",
		"rgb(255, 0, 128)":     "#ff0080",
		"rgba(255, 0, 128, 1)": "#ff0080",
		"rgb(999,-5,0)":        "#ff0000",
		"not-a-color":          "",
	}
	for in, want := range cases {
		if got := NormalizeToHex(in); got != want {
			t.Errorf("NormalizeToHex(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestColorHex(t *testing.T) {
	c := Color{RGBA: color.RGBA{R: 0xF0, G: 0xB4, B: 0x29, A: 0xFF}, OK: true}
	if got := c.Hex(); got != "#f0b429" {
		t.Errorf("Hex() = %q, want #f0b429", got)
	}
	if got := (Color{}).Hex(); got != "" {
		t.Errorf("zero Color Hex() = %q, want empty", got)
	}
}

func TestSnapshotDoesNotPanic(t *testing.T) {
	_ = Snapshot()
}
