//go:build !(linux || windows || darwin)

// platform/accent/accent_other.go

package accent

func getAccentColor() Color {
	return Color{OK: false}
}
