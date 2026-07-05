// accent/accent_other.go
//go:build !(linux || windows || darwin)

package accent

func getAccentColor() Color {
	return Color{OK: false}
}
