// platform/accent/watch.go

package accent

// Watch delivers a fresh Snapshot on every debounced accent or color-scheme change.
func Watch(onChange func(Info)) (stop func()) {
	return watchAccentChanges(func() {
		onChange(Snapshot())
	})
}
