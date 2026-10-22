package windows

// Strip describes untransformed tab positions relative to the webview. Pinned
// tabs occupy slots but cannot be displaced by an incoming tab.
type Strip struct {
	Bounds Rect      `json:"bounds"`
	Tabs   []TabRect `json:"tabs"`
}
type TabRect struct {
	ID     string `json:"id"`
	X      int    `json:"x"`
	Width  int    `json:"width"`
	Pinned bool   `json:"pinned"`
}
type Drop struct {
	ID     string `json:"id"`
	Tab    string `json:"tab"`
	Target string `json:"target"`
	Slot   int    `json:"slot"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	// Face is what the source page said the tab looks like, so the window it
	// hovers can draw it in the slot it opens.
	Face string `json:"face,omitempty"`
}

// Hit returns an insertion slot, excluding the source tab, or -1 outside.
func (s Strip) Hit(x, y int, source string) int {
	b := s.Bounds
	if x < b.X || y < b.Y || x >= b.X+b.Width || y >= b.Y+b.Height {
		return -1
	}
	slot := 0
	for _, tab := range s.Tabs {
		if tab.ID == source {
			continue
		}
		if !tab.Pinned && x < tab.X+tab.Width/2 {
			return slot
		}
		slot++
	}
	return slot
}

// DropSlot appends a tab dropped over another workspace, preserving source tear-off.
func (s Strip) DropSlot(x, y int, source string) int {
	slot := s.Hit(x, y, source)
	if slot < 0 && source == "" && s.Bounds.Width > 0 && s.Bounds.Height > 0 {
		return len(s.Tabs)
	}
	return slot
}
