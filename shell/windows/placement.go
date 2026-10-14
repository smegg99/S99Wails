package windows

import "github.com/wailsapp/wails/v3/pkg/application"

type Geometry struct {
	X, Y, Width, Height int
	Maximised           bool
	Monitor             string
}

// PlaceWindow clamps a saved rectangle against the monitors currently present.
// Call only once the GUI is running.
func PlaceWindow(win *application.WebviewWindow, geometry *Geometry, minWidth, minHeight int) {
	monitors := []Monitor{}
	for _, screen := range application.Get().Screen.GetAll() {
		area := screen.WorkArea
		monitors = append(monitors, Monitor{ID: screen.ID, Primary: screen.IsPrimary,
			Work: Rect{X: area.X, Y: area.Y, Width: area.Width, Height: area.Height}})
	}
	placed := Place(Rect{X: int(geometry.X), Y: int(geometry.Y), Width: int(geometry.Width), Height: int(geometry.Height)}, geometry.Monitor, monitors, minWidth, minHeight)
	win.UnMaximise()
	win.SetSize(placed.Width, placed.Height)
	win.SetPosition(placed.X, placed.Y)
	if geometry.Maximised {
		win.Maximise()
	}
}
