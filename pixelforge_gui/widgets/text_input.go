package widgets

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
)

// TextInputOptions configures a TextInput.
type TextInputOptions struct {
	Initial     string
	MaxRunes    int // 0 means unbounded
	BgColor     pixelforge.Color
	BorderColor pixelforge.Color
	FocusColor  pixelforge.Color
	FgColor     pixelforge.Color
	CursorColor pixelforge.Color
	OnSubmit    func(value string)
	OnChange    func(value string)
}

// TextInput is a single-line input. The widget owns a rune buffer and a
// cursor index; callers drive the buffer via the AppendRune /
// Backspace / MoveCursor / Submit methods. Wrapping these methods around
// the editor's Ebitengine input pump keeps pixelforge_gui free of the
// Ebitengine dependency.
type TextInput struct {
	*pgui.Element

	BgColor     pixelforge.Color
	BorderColor pixelforge.Color
	FocusColor  pixelforge.Color
	FgColor     pixelforge.Color
	CursorColor pixelforge.Color

	MaxRunes int

	OnSubmit func(value string)
	OnChange func(value string)

	buffer []rune
	cursor int

	focus *pgui.FocusManager
}

// NewTextInput constructs a TextInput rooted at (x, y, w, h).
//
// When focus is non-nil, the input registers itself for Tab traversal
// and renders a focus ring when it owns focus.
func NewTextInput(x, y, w, h int, focus *pgui.FocusManager, opts TextInputOptions) *TextInput {
	if opts.BgColor == 0 {
		opts.BgColor = 1
	}
	if opts.BorderColor == 0 {
		opts.BorderColor = 6
	}
	if opts.FocusColor == 0 {
		opts.FocusColor = 12
	}
	if opts.FgColor == 0 {
		opts.FgColor = 7
	}
	if opts.CursorColor == 0 {
		opts.CursorColor = 10
	}
	ti := &TextInput{
		Element: &pgui.Element{
			Area: pixelforge.IntArea{X: x, Y: y, W: w, H: h},
		},
		BgColor:     opts.BgColor,
		BorderColor: opts.BorderColor,
		FocusColor:  opts.FocusColor,
		FgColor:     opts.FgColor,
		CursorColor: opts.CursorColor,
		MaxRunes:    opts.MaxRunes,
		OnSubmit:    opts.OnSubmit,
		OnChange:    opts.OnChange,
		buffer:      []rune(opts.Initial),
		cursor:      len([]rune(opts.Initial)),
		focus:       focus,
	}
	if focus != nil {
		focus.Register(ti.Element)
	}
	ti.Element.OnDraw = func(ev pgui.DrawEvent) {
		ti.draw()
	}
	ti.Element.OnTap = func(ev pgui.Event) {
		if focus != nil {
			focus.Focus(ti.Element)
		}
	}
	return ti
}

// Value returns the current buffer as a string.
func (t *TextInput) Value() string { return string(t.buffer) }

// SetValue replaces the buffer and clamps the cursor.
func (t *TextInput) SetValue(s string) {
	t.buffer = []rune(s)
	if t.cursor > len(t.buffer) {
		t.cursor = len(t.buffer)
	}
	t.fireChange()
}

// Cursor returns the current cursor index.
func (t *TextInput) Cursor() int { return t.cursor }

// Focused reports whether this input owns keyboard focus.
func (t *TextInput) Focused() bool {
	if t.focus == nil {
		return false
	}
	return t.focus.Focused() == t.Element
}

// AppendRune inserts r at the cursor position when there is room.
func (t *TextInput) AppendRune(r rune) {
	if t.MaxRunes > 0 && len(t.buffer) >= t.MaxRunes {
		return
	}
	// Insert at cursor position.
	t.buffer = append(t.buffer, 0)
	copy(t.buffer[t.cursor+1:], t.buffer[t.cursor:])
	t.buffer[t.cursor] = r
	t.cursor++
	t.fireChange()
}

// AppendRunes is the bulk form of AppendRune.
func (t *TextInput) AppendRunes(rs []rune) {
	for _, r := range rs {
		t.AppendRune(r)
	}
}

// Backspace removes the rune before the cursor, if any.
func (t *TextInput) Backspace() {
	if t.cursor == 0 {
		return
	}
	t.buffer = append(t.buffer[:t.cursor-1], t.buffer[t.cursor:]...)
	t.cursor--
	t.fireChange()
}

// DeleteForward removes the rune at the cursor, if any.
func (t *TextInput) DeleteForward() {
	if t.cursor >= len(t.buffer) {
		return
	}
	t.buffer = append(t.buffer[:t.cursor], t.buffer[t.cursor+1:]...)
	t.fireChange()
}

// MoveCursor adjusts the cursor index by delta (clamped to [0, len]).
func (t *TextInput) MoveCursor(delta int) {
	t.cursor += delta
	if t.cursor < 0 {
		t.cursor = 0
	}
	if t.cursor > len(t.buffer) {
		t.cursor = len(t.buffer)
	}
}

// CursorHome moves the cursor to the start.
func (t *TextInput) CursorHome() { t.cursor = 0 }

// CursorEnd moves the cursor to the end.
func (t *TextInput) CursorEnd() { t.cursor = len(t.buffer) }

// Submit fires the OnSubmit callback with the current value.
func (t *TextInput) Submit() {
	if t.OnSubmit != nil {
		t.OnSubmit(string(t.buffer))
	}
}

func (t *TextInput) fireChange() {
	if t.OnChange != nil {
		t.OnChange(string(t.buffer))
	}
}

func (t *TextInput) draw() {
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	pixelforge.SetColor(t.BgColor)
	pixelforge.RectFill(0, 0, t.W-1, t.H-1)

	border := t.BorderColor
	if t.Focused() {
		border = t.FocusColor
	}
	pixelforge.SetColor(border)
	pixelforge.Rect(0, 0, t.W-1, t.H-1)

	pixelforge.SetColor(t.FgColor)
	font := pgui.DefaultFont()
	textY := (t.H - font.LineHeight()) / 2
	str := string(t.buffer)
	_, _ = font.Print(str, 4, textY)

	if t.Focused() {
		// Cursor: render a 1×lineHeight vertical bar at the cursor
		// glyph position.
		pixelforge.SetColor(t.CursorColor)
		prefixW, _ := font.Measure(string(t.buffer[:t.cursor]))
		cx := 4 + prefixW
		pixelforge.RectFill(cx, textY, cx, textY+font.LineHeight()-1)
	}
}
