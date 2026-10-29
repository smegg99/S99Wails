// shell/windows/geometry.go
package windows

// Rect is a desktop work area or a window rectangle, in logical pixels.
type Rect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}
type Monitor struct {
	ID      string
	Work    Rect
	Primary bool
}

// Place restores a window to its monitor, or the primary when that one is gone, inside the work area.
func Place(saved Rect, monitor string, monitors []Monitor, minWidth, minHeight int) Rect {
	if len(monitors) == 0 {
		return saved
	}
	chosen := monitors[0]
	for _, m := range monitors {
		if m.Primary {
			chosen = m
			break
		}
	}
	found := false
	for _, m := range monitors {
		if m.ID == monitor {
			chosen = m
			found = true
			break
		}
	}
	area := chosen.Work
	saved.Width = min(max(saved.Width, minWidth), max(area.Width, 1))
	saved.Height = min(max(saved.Height, minHeight), max(area.Height, 1))
	if !found {
		saved.X = area.X + (area.Width-saved.Width)/2
		saved.Y = area.Y + (area.Height-saved.Height)/2
	}
	saved.X = min(max(saved.X, area.X), area.X+area.Width-saved.Width)
	saved.Y = min(max(saved.Y, area.Y), area.Y+area.Height-saved.Height)
	return saved
}
