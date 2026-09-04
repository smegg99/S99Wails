// platform/accent/example_test.go

package accent_test

import (
	"fmt"
	"image/color"

	"github.com/smegg99/s99wails/v2/platform/accent"
)

func ExampleNormalizeToHex() {
	fmt.Println(accent.NormalizeToHex("rgb(240, 180, 41)"))
	fmt.Println(accent.NormalizeToHex("rgba(59, 111, 160, 0.8)"))
	fmt.Println(accent.NormalizeToHex("#22303c"))
	// Output:
	// #f0b429
	// #3b6fa0
	// #22303c
}

func ExampleColor_Hex() {
	bell := accent.Color{RGBA: color.RGBA{R: 0xF0, G: 0xB4, B: 0x29, A: 0xFF}, OK: true}
	fmt.Println(bell.Hex())
	fmt.Println(accent.Color{}.Hex() == "") // unresolved colors yield ""
	// Output:
	// #f0b429
	// true
}
