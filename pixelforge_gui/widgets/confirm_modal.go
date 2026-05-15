package widgets

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
)

// ConfirmModalOptions configures a ConfirmModal.
type ConfirmModalOptions struct {
	TitleColor    pixelforge.Color
	BodyColor     pixelforge.Color
	BackdropColor pixelforge.Color
	OKBgColor     pixelforge.Color
	CancelBgColor pixelforge.Color
	TextColor     pixelforge.Color
}

// ConfirmModal is the canvas-resident equivalent of the native confirm
// dialog. Mirrors the API: Show(title, msg, onOK), Hide(), Visible().
type ConfirmModal struct {
	X, Y, W, H int

	Title   string
	Message string

	OnOK     func()
	OnCancel func()

	visible bool

	titleColor    pixelforge.Color
	bodyColor     pixelforge.Color
	backdropColor pixelforge.Color
	okBgColor     pixelforge.Color
	cancelBgColor pixelforge.Color
	textColor     pixelforge.Color

	okRect, cancelRect IntRect
}

// NewConfirmModal constructs a confirm modal at (x, y) with the given
// dimensions.
func NewConfirmModal(x, y, w, h int, opts ConfirmModalOptions) *ConfirmModal {
	if opts.TitleColor == 0 {
		opts.TitleColor = 7
	}
	if opts.BodyColor == 0 {
		opts.BodyColor = 2
	}
	if opts.BackdropColor == 0 {
		opts.BackdropColor = 0
	}
	if opts.OKBgColor == 0 {
		opts.OKBgColor = 12
	}
	if opts.CancelBgColor == 0 {
		opts.CancelBgColor = 5
	}
	if opts.TextColor == 0 {
		opts.TextColor = 7
	}
	return &ConfirmModal{
		X: x, Y: y, W: w, H: h,
		titleColor: opts.TitleColor, bodyColor: opts.BodyColor,
		backdropColor: opts.BackdropColor, okBgColor: opts.OKBgColor,
		cancelBgColor: opts.CancelBgColor, textColor: opts.TextColor,
	}
}

// Show makes the modal visible with the given title + message and
// records the callbacks.
func (c *ConfirmModal) Show(title, message string, onOK, onCancel func()) {
	c.Title = title
	c.Message = message
	c.OnOK = onOK
	c.OnCancel = onCancel
	c.visible = true
}

// Hide dismisses the modal without firing either callback.
func (c *ConfirmModal) Hide() { c.visible = false }

// Visible reports whether the modal is currently displayed.
func (c *ConfirmModal) Visible() bool { return c.visible }

// HandleClick processes a click. Returns true when consumed.
func (c *ConfirmModal) HandleClick(px, py int) bool {
	if !c.visible {
		return false
	}
	if c.okRect.Contains(px, py) {
		c.visible = false
		if c.OnOK != nil {
			c.OnOK()
		}
		return true
	}
	if c.cancelRect.Contains(px, py) {
		c.visible = false
		if c.OnCancel != nil {
			c.OnCancel()
		}
		return true
	}
	return true // consume any other click while visible (modal precedence)
}

// HandleEscape mirrors HandleClick on the cancel button.
func (c *ConfirmModal) HandleEscape() bool {
	if !c.visible {
		return false
	}
	c.visible = false
	if c.OnCancel != nil {
		c.OnCancel()
	}
	return true
}

// Draw paints the modal via engine primitives.
func (c *ConfirmModal) Draw() {
	if !c.visible {
		return
	}
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	// Modal body — centred 60% of the available area, capped.
	bodyW := c.W * 3 / 4
	bodyH := 80
	bodyX := c.X + (c.W-bodyW)/2
	bodyY := c.Y + (c.H-bodyH)/2

	// Backdrop dim.
	pixelforge.SetColor(c.backdropColor)
	pixelforge.RectFill(c.X, c.Y, c.X+c.W-1, c.Y+c.H-1)

	// Body fill + border.
	pixelforge.SetColor(c.bodyColor)
	pixelforge.RectFill(bodyX, bodyY, bodyX+bodyW-1, bodyY+bodyH-1)
	pixelforge.SetColor(c.textColor)
	pixelforge.Rect(bodyX, bodyY, bodyX+bodyW-1, bodyY+bodyH-1)

	// Title + message.
	pixelforge.SetColor(c.titleColor)
	pixelforge_cofont.Print(c.Title, bodyX+8, bodyY+8)
	pixelforge.SetColor(c.textColor)
	pixelforge_cofont.Print(c.Message, bodyX+8, bodyY+24)

	// Buttons (OK on the right, Cancel on the left of it).
	btnW := 60
	btnH := 18
	gap := 8
	okX := bodyX + bodyW - btnW - 8
	cancelX := okX - btnW - gap
	btnY := bodyY + bodyH - btnH - 8
	c.okRect = IntRect{X: okX, Y: btnY, W: btnW, H: btnH}
	c.cancelRect = IntRect{X: cancelX, Y: btnY, W: btnW, H: btnH}

	pixelforge.SetColor(c.okBgColor)
	pixelforge.RectFill(okX, btnY, okX+btnW-1, btnY+btnH-1)
	pixelforge.SetColor(c.cancelBgColor)
	pixelforge.RectFill(cancelX, btnY, cancelX+btnW-1, btnY+btnH-1)
	pixelforge.SetColor(c.textColor)
	pixelforge_cofont.Print("OK", okX+btnW/2-4, btnY+5)
	pixelforge_cofont.Print("Cancel", cancelX+8, btnY+5)
}

// SetBounds repositions the modal.
func (c *ConfirmModal) SetBounds(x, y, w, h int) {
	c.X, c.Y, c.W, c.H = x, y, w, h
}
