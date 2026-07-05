// darkmode_other.go
//go:build !(linux || windows || darwin)

package accent

func detectScheme() (Scheme, bool) {
	return SchemeNoPreference, false
}
