package widgets

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Modal is a thin scrim drawn over the full window. A widget rendered
// on top of a modal owns its own input handling; Modal itself only
// paints the backdrop and reports dismiss requests.
//
// Multiple modals stack via a small slice on the editor; only the
// topmost handles input.
type Modal struct {
	Visible bool

	// Body is the rectangle the child widget claims. A click outside
	// Body (but inside the window) is treated as a dismiss request.
	Body Rect

	// OnDismiss fires when the user presses Esc or clicks the backdrop.
	// Nil callbacks are safe — Visible is still flipped to false.
	OnDismiss func()
}

// DrawBackdrop fills the window with a semi-transparent layer. Callers
// invoke this before drawing their modal body.
func (m *Modal) DrawBackdrop(dst *ebiten.Image) {
	if !m.Visible {
		return
	}
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	fillRect(dst, Rect{X: 0, Y: 0, W: w, H: h}, modalScrim)
}

// HandleDismiss checks for Esc / click-outside dismissal. Returns true
// if the modal was dismissed this frame.
func (m *Modal) HandleDismiss(mx, my int) bool {
	if !m.Visible {
		return false
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		m.dismiss()
		return true
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if !m.Body.Contains(mx, my) {
			m.dismiss()
			return true
		}
	}
	return false
}

func (m *Modal) dismiss() {
	m.Visible = false
	if m.OnDismiss != nil {
		m.OnDismiss()
	}
}

var modalScrim = color.RGBA{R: 0, G: 0, B: 0, A: 0xa0}
