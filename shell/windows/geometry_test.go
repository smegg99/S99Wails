// shell/windows/geometry_test.go

package windows

import "testing"

func TestMonitorRestorationAndMissingMonitor(t *testing.T) {
	monitors := []Monitor{
		{ID: "left", Work: Rect{X: -1920, Y: 0, Width: 1920, Height: 1080}},
		{ID: "main", Primary: true, Work: Rect{X: 0, Y: 28, Width: 2560, Height: 1412}},
	}
	saved := Rect{X: -1800, Y: 100, Width: 1280, Height: 820}
	if got := Place(saved, "left", monitors, 1024, 700); got != saved {
		t.Fatalf("valid monitor moved: %+v", got)
	}
	got := Place(saved, "unplugged", monitors, 1024, 700)
	if got.X < 0 || got.Y < 28 || got.X+got.Width > 2560 || got.Y+got.Height > 1440 {
		t.Fatalf("invisible after unplug: %+v", got)
	}
	off := Place(Rect{X: 9999, Y: -999, Width: 4000, Height: 2500}, "left", monitors, 1024, 700)
	if off.X != -1920 || off.Y != 0 || off.Width != 1920 || off.Height != 1080 {
		t.Fatalf("oversized: %+v", off)
	}
}
