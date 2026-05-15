package pixelforge_gui

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
)

// Font is the chokepoint that canvas-resident chrome uses for text. A
// single method set lets the editor swap to TTF later without touching
// any Label call site.
type Font interface {
	// Print draws text starting at (x, y) using the current draw color.
	// Returns the position where the caller can continue writing.
	Print(text string, x, y int) (endX, endY int)
	// Measure returns the rendered width and height of text without
	// drawing it. Multi-line input is honoured.
	Measure(text string) (w, h int)
	// LineHeight returns the vertical advance between two lines.
	LineHeight() int
}

// DefaultFont returns a Font backed by pixelforge_cofont.Sheet (the
// PICO-8 4x8 font). The cofont package's init has already populated
// Sheet.Chars; this wrapper exposes it through the Font interface.
func DefaultFont() Font {
	return cofontFont{}
}

type cofontFont struct{}

func (cofontFont) Print(text string, x, y int) (int, int) {
	return pixelforge_cofont.Print(text, x, y)
}

func (cofontFont) Measure(text string) (int, int) {
	if text == "" {
		return 0, pixelforge_cofont.Sheet.Height
	}
	maxW := 0
	curW := 0
	lines := 1
	for _, r := range text {
		if r == '\n' {
			if curW > maxW {
				maxW = curW
			}
			curW = 0
			lines++
			continue
		}
		sprite, ok := pixelforge_cofont.Sheet.Chars[r]
		if !ok {
			// Fall back to '?' glyph width (4px for ASCII).
			sprite = pixelforge_cofont.Sheet.Chars['?']
		}
		curW += sprite.W
	}
	if curW > maxW {
		maxW = curW
	}
	return maxW, lines * pixelforge_cofont.Sheet.Height
}

func (cofontFont) LineHeight() int {
	return pixelforge_cofont.Sheet.Height
}

// Align controls horizontal text alignment inside a Label's area.
type Align int

const (
	AlignLeft Align = iota
	AlignCenter
	AlignRight
)

// LabelOptions configures a Label. Zero values mean: left-align, default
// font, default text color (slot 7), no padding.
type LabelOptions struct {
	Text     string
	FgColor  pixelforge.Color
	Font     Font
	Align    Align
	PaddingX int
	PaddingY int
}

// Label is a small, non-interactive widget that draws text inside its
// Area. It uses the default font when none is supplied.
type Label struct {
	*Element

	Text     string
	FgColor  pixelforge.Color
	Font     Font
	Align    Align
	PaddingX int
	PaddingY int
}

// NewLabel constructs a Label with the given options. The returned Label
// already has its Element wired with an OnDraw callback; callers attach
// it to a parent via parent.Attach(label.Element).
func NewLabel(x, y, w, h int, opts LabelOptions) *Label {
	if opts.Font == nil {
		opts.Font = DefaultFont()
	}
	if opts.FgColor == 0 {
		// Slot 7 is the canonical M0-M2 chrome text color.
		opts.FgColor = 7
	}
	lbl := &Label{
		Element: &Element{
			Area: pixelforge.IntArea{X: x, Y: y, W: w, H: h},
		},
		Text:     opts.Text,
		FgColor:  opts.FgColor,
		Font:     opts.Font,
		Align:    opts.Align,
		PaddingX: opts.PaddingX,
		PaddingY: opts.PaddingY,
	}
	lbl.Element.OnDraw = func(ev DrawEvent) {
		lbl.draw()
	}
	return lbl
}

// SetText updates the label's text. The next draw reflects it.
func (l *Label) SetText(s string) {
	l.Text = s
}

// SetFgColor updates the label's foreground color.
func (l *Label) SetFgColor(c pixelforge.Color) {
	l.FgColor = c
}

// draw paints the label inside its area. Coordinates are element-local:
// pixelforge_gui.Element.Draw has already shifted the camera and clip so
// (0, 0) is the label's top-left.
func (l *Label) draw() {
	if l.Text == "" {
		return
	}
	prev := pixelforge.SetColor(l.FgColor)
	defer pixelforge.SetColor(prev)

	textW, _ := l.Font.Measure(l.Text)
	x := l.PaddingX
	switch l.Align {
	case AlignCenter:
		x = (l.W - textW) / 2
	case AlignRight:
		x = l.W - textW - l.PaddingX
	}
	y := l.PaddingY
	l.Font.Print(l.Text, x, y)
}
