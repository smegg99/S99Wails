//go:build !(linux || windows || darwin)

// accent/darkmode_other.go

package accent

func detectScheme() (Scheme, bool) {
	return SchemeNoPreference, false
}
