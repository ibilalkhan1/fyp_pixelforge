package widgets

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
)

// ModalOptions configures a Modal.
type ModalOptions struct {
	// BodyW, BodyH are the body rectangle dimensions; the body is
	// centred in the modal area.
	BodyW, BodyH int
	BackdropColor pixelforge.Color
	BodyBgColor   pixelforge.Color
	BodyBorder    pixelforge.Color
	OnDismiss     func()
}

// Modal is a full-area backdrop + centred body. The body is exposed as
// Modal.Body so callers can attach child widgets (Label, Button, etc.)
// in body-local coordinates.
type Modal struct {
	*pgui.Element

	BackdropColor pixelforge.Color
	BodyBgColor   pixelforge.Color
	BodyBorder    pixelforge.Color
	OnDismiss     func()

	BodyW, BodyH int
	BodyX, BodyY int

	Body *pgui.Element

	visible bool
}

// NewModal constructs a Modal whose backdrop covers (x, y, w, h). The
// body is centred and exposes Body for child attachment.
func NewModal(x, y, w, h int, opts ModalOptions) *Modal {
	if opts.BackdropColor == 0 {
		opts.BackdropColor = 0
	}
	if opts.BodyBgColor == 0 {
		opts.BodyBgColor = 1
	}
	if opts.BodyBorder == 0 {
		opts.BodyBorder = 6
	}
	if opts.BodyW == 0 {
		opts.BodyW = w * 2 / 3
	}
	if opts.BodyH == 0 {
		opts.BodyH = h * 2 / 3
	}
	m := &Modal{
		Element: &pgui.Element{
			Area: pixelforge.IntArea{X: x, Y: y, W: w, H: h},
		},
		BackdropColor: opts.BackdropColor,
		BodyBgColor:   opts.BodyBgColor,
		BodyBorder:    opts.BodyBorder,
		OnDismiss:     opts.OnDismiss,
		BodyW:         opts.BodyW,
		BodyH:         opts.BodyH,
		BodyX:         (w - opts.BodyW) / 2,
		BodyY:         (h - opts.BodyH) / 2,
	}
	m.Body = pgui.Attach(m.Element, m.BodyX, m.BodyY, m.BodyW, m.BodyH)
	m.Element.OnDraw = func(ev pgui.DrawEvent) {
		if !m.visible {
			return
		}
		m.drawBackdrop()
		m.drawBody()
	}
	// Clicking the backdrop (outside the body) dismisses. The Body's
	// own OnTap handlers handle clicks inside.
	m.Element.OnTap = func(ev pgui.Event) {
		mx, my := pguiPointerLocal(m.Element)
		if mx >= m.BodyX && mx < m.BodyX+m.BodyW &&
			my >= m.BodyY && my < m.BodyY+m.BodyH {
			return
		}
		m.Dismiss()
	}
	return m
}

// Show makes the modal visible.
func (m *Modal) Show() { m.visible = true }

// Dismiss closes the modal and fires OnDismiss.
func (m *Modal) Dismiss() {
	if !m.visible {
		return
	}
	m.visible = false
	if m.OnDismiss != nil {
		m.OnDismiss()
	}
}

// Visible reports whether the modal is currently showing.
func (m *Modal) Visible() bool { return m.visible }

// HandleEscape dismisses the modal if it's open. Returns true when it
// owned the Esc event so the caller can prevent further routing.
func (m *Modal) HandleEscape() bool {
	if !m.visible {
		return false
	}
	m.Dismiss()
	return true
}

func (m *Modal) drawBackdrop() {
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	pixelforge.SetColor(m.BackdropColor)
	pixelforge.RectFill(0, 0, m.W-1, m.H-1)
}

func (m *Modal) drawBody() {
	// The Body element has its own OnDraw delegated to children, so the
	// engine handles its rendering recursively. Here we paint the body
	// background + border once. Because Body is a child of m.Element,
	// pgui has already pushed the camera/clip; we paint relative to the
	// modal's coordinate space.
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	pixelforge.SetColor(m.BodyBgColor)
	pixelforge.RectFill(m.BodyX, m.BodyY, m.BodyX+m.BodyW-1, m.BodyY+m.BodyH-1)
	pixelforge.SetColor(m.BodyBorder)
	pixelforge.Rect(m.BodyX, m.BodyY, m.BodyX+m.BodyW-1, m.BodyY+m.BodyH-1)
}

// ModalStack manages a stack of modals so the topmost one consumes
// dismiss events (Esc, click-outside) without affecting the layers
// underneath.
type ModalStack struct {
	stack []*Modal
}

// NewModalStack creates an empty ModalStack.
func NewModalStack() *ModalStack { return &ModalStack{} }

// Push adds m to the top of the stack and shows it.
func (s *ModalStack) Push(m *Modal) {
	s.stack = append(s.stack, m)
	m.Show()
}

// Top returns the topmost visible modal, or nil.
func (s *ModalStack) Top() *Modal {
	for i := len(s.stack) - 1; i >= 0; i-- {
		if s.stack[i].Visible() {
			return s.stack[i]
		}
	}
	return nil
}

// HandleEscape routes Esc to the top modal only. Returns true when a
// modal consumed the event.
func (s *ModalStack) HandleEscape() bool {
	if top := s.Top(); top != nil {
		return top.HandleEscape()
	}
	return false
}

// Cleanup removes dismissed modals from the stack to free memory.
func (s *ModalStack) Cleanup() {
	out := s.stack[:0]
	for _, m := range s.stack {
		if m.Visible() {
			out = append(out, m)
		}
	}
	s.stack = out
}
