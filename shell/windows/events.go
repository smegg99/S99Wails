// shell/windows/events.go
package windows

// The events the window service sends to its pages; frontendgen's names.ts carries the same values.
const (
	eventWindowsChanged = "windows:changed"
	eventTabArrive      = "tab:arrive"
	eventTabRelease     = "tab:release"
	eventTabDone        = "tab:done"
	eventTabCancel      = "tab:cancel"
	eventTabCancelled   = "tab:cancelled"
	eventTabHover       = "tab:hover"
	eventTabTrack       = "tab:track"
	eventTabDragEnd     = "tab:drag-end"
	eventTabDrop        = "tab:drop"
	eventWindowBlocked  = "window:blocked"
	eventWindowResume   = "window:resume"
	eventWindowClose    = "window:close"
)
