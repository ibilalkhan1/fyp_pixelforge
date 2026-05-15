package widgets

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
	pimouse "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_mouse"
)

// pguiPointerLocal returns the mouse pointer position in element-local
// coordinates, consistent with how pgui.Element computes hasPointer in
// Update / Draw. Callers are responsible for ensuring this is invoked
// from within a pgui callback (camera has already been shifted).
func pguiPointerLocal(_ *pgui.Element) (int, int) {
	p := pimouse.Position.Add(pixelforge.Camera)
	return p.X, p.Y
}
