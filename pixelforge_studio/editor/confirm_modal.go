package editor

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// ConfirmDialog is the editor-side confirm-modal. It carries title/
// message text plus an OnConfirm callback fired when the user accepts.
type ConfirmDialog struct {
	visible   bool
	title     string
	message   string
	onConfirm func()
}

// NewConfirmDialog returns a hidden dialog ready to be Show()n.
func NewConfirmDialog() *ConfirmDialog {
	return &ConfirmDialog{}
}

// Visible reports whether the dialog is currently shown.
func (c *ConfirmDialog) Visible() bool { return c.visible }

// Show pops the dialog with the supplied copy and confirmation handler.
// Replaces any prior dialog state.
func (c *ConfirmDialog) Show(title, message string, onConfirm func()) {
	c.title = title
	c.message = message
	c.onConfirm = onConfirm
	c.visible = true
}

// Confirm runs the handler and hides the dialog. Used by tests.
func (c *ConfirmDialog) Confirm() {
	c.visible = false
	if c.onConfirm != nil {
		c.onConfirm()
	}
}

// Cancel hides the dialog without running the handler.
func (c *ConfirmDialog) Cancel() { c.visible = false }

// Update processes input. Returns true while visible.
func (c *ConfirmDialog) Update(windowW, windowH int) bool {
	if !c.visible {
		return false
	}
	body := c.bodyRect(windowW, windowH)
	confirm, cancel := c.buttonRects(body)
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		c.Cancel()
		return true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		c.Confirm()
		return true
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		switch {
		case confirm.Contains(mx, my):
			c.Confirm()
		case cancel.Contains(mx, my):
			c.Cancel()
		case !body.Contains(mx, my):
			c.Cancel()
		}
	}
	return true
}

// Draw renders the dialog over the supplied window.
func (c *ConfirmDialog) Draw(dst *ebiten.Image, windowW, windowH int) {
	body := c.bodyRect(windowW, windowH)
	scrim := widgets.Rect{X: 0, Y: 0, W: windowW, H: windowH}
	vectorFill(dst, scrim, scrimColor)

	vectorFill(dst, body, dialogBg)
	debugPrint(dst, c.title, body.X+12, body.Y+8)
	debugPrint(dst, c.message, body.X+12, body.Y+32)

	confirm, cancel := c.buttonRects(body)
	vectorFill(dst, cancel, dialogCancel)
	debugPrint(dst, "Cancel", cancel.X+16, cancel.Y+3)
	vectorFill(dst, confirm, dialogConfirm)
	debugPrint(dst, "Confirm", confirm.X+12, confirm.Y+3)
}

func (c *ConfirmDialog) bodyRect(windowW, windowH int) widgets.Rect {
	bw := 360
	bh := 120
	if bw > windowW-20 {
		bw = windowW - 20
	}
	if bh > windowH-20 {
		bh = windowH - 20
	}
	return widgets.Rect{X: (windowW - bw) / 2, Y: (windowH - bh) / 2, W: bw, H: bh}
}

func (c *ConfirmDialog) buttonRects(body widgets.Rect) (confirm, cancel widgets.Rect) {
	confirm = widgets.Rect{X: body.X + body.W - 90, Y: body.Y + body.H - 30, W: 80, H: 22}
	cancel = widgets.Rect{X: body.X + body.W - 180, Y: body.Y + body.H - 30, W: 80, H: 22}
	return
}

var (
	scrimColor    = color.RGBA{R: 0, G: 0, B: 0, A: 0xa0}
	dialogBg      = color.RGBA{R: 0x22, G: 0x22, B: 0x2c, A: 0xff}
	dialogConfirm = color.RGBA{R: 0x46, G: 0x86, B: 0xff, A: 0xff}
	dialogCancel  = color.RGBA{R: 0x44, G: 0x44, B: 0x4c, A: 0xff}
)
