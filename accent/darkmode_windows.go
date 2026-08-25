//go:build windows

// accent/darkmode_windows.go

package accent

import "golang.org/x/sys/windows/registry"

func detectScheme() (Scheme, bool) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		return SchemeNoPreference, false
	}
	defer k.Close()

	v, _, err := k.GetIntegerValue("AppsUseLightTheme")
	if err != nil {
		return SchemeNoPreference, false
	}
	if v == 0 {
		return SchemeDark, true
	}
	return SchemeLight, true
}
