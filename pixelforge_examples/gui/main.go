// This example demonstrates building a simple GUI hierarchy with pixelforge_gui.
// It shows:
//   - A panel (container) with a local coordinate system
//   - Three buttons arranged vertically inside the panel
//   - Clicking a button logs its label
//
// The layout is recalculated relative to the panel's position,
// showing pixelforge_gui's tree-structure approach.
package main

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
	"log"
)

// colors used in this example (default Pixelforge palette):
const (
	lightBlue = 28
	white     = 7
	darkBlue  = 1
	lightGray = 6
	blue      = 12
)

func main() {
	pixelforge.SetScreenSize(128, 128)
	// create the root of the entire GUI element tree
	root := pixelforge_gui.New()
	// add a panel (container) at global coordinates
	panel := attachPanel(root, 32, 32, 63, 63)
	// add buttons to the panel using its local coordinate system
	attachButton(panel, 10, 9, 44, 14, "BUTTON 1")
	attachButton(panel, 10, 25, 44, 14, "BUTTON 2")
	// add a button with a callback that runs when the user clicks
	// and releases the left mouse button while staying inside its area
	btn3 := attachButton(panel, 10, 41, 44, 14, "BUTTON 3")
	btn3.OnTap = func(event pixelforge_gui.Event) {
		log.Println("Button 3 was tapped")
	}

	pixelforge.Update = func() {
		// root.Update() must be called in the game loop
		root.Update()
	}

	pixelforge.Draw = func() {
		pixelforge.Cls()
		// root.Draw() must be called in the game loop
		root.Draw()
	}

	pixelforge_ebiten.Run()
}

func attachPanel(parent *pixelforge_gui.Element, x, y, w, h int) *pixelforge_gui.Element {
	panel := pixelforge_gui.Attach(parent, x, y, w, h)
	panel.OnDraw = func(event pixelforge_gui.DrawEvent) {
		pixelforge.SetColor(lightBlue)
		pixelforge.Rect(0, 0, panel.W-1, panel.H-1)
		pixelforge.SetColor(darkBlue)
		pixelforge.RectFill(1, 1, panel.W-2, panel.H-2)
	}
	return panel
}

func attachButton(parent *pixelforge_gui.Element, x, y, w, h int, label string) *pixelforge_gui.Element {
	btn := pixelforge_gui.Attach(parent, x, y, w, h)
	btn.OnDraw = func(event pixelforge_gui.DrawEvent) {
		var frame, bg, text pixelforge.Color = lightGray, blue, white
		if event.HasPointer {
			frame, bg, text = lightGray, lightBlue, white
		}

		if event.Pressed {
			pixelforge.Camera.Y -= 1 // the camera is automatically reset after drawing the element
			bg = blue
		}

		pixelforge.SetColor(frame)
		pixelforge.Rect(0, 0, w-2, h-2)

		pixelforge.SetColor(bg)
		pixelforge.RectFill(1, 1, w-3, h-3)

		pixelforge.SetColor(text)
		pixelforge_cofont.Print(label, 6, 4)
	}
	return btn
}
