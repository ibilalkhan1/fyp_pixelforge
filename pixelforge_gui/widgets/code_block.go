package widgets

import (
	"strings"

	"github.com/ibilalkhan1/fyp_pixelforge"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
)

// CodeBlockOptions configures a CodeBlock widget.
type CodeBlockOptions struct {
	Text       string
	FgColor    pixelforge.Color
	BgColor    pixelforge.Color
	BorderColor pixelforge.Color
}

// CodeBlock is a read-only multi-line text panel used inside a
// Modal body to display generated source. v1 doesn't syntax-colour
// — the host's theme provides foreground/background colours and the
// widget renders one line per text line.
type CodeBlock struct {
	*pgui.Element

	Text        string
	FgColor     pixelforge.Color
	BgColor     pixelforge.Color
	BorderColor pixelforge.Color
}

// NewCodeBlock constructs a CodeBlock rooted at (x, y, w, h).
func NewCodeBlock(x, y, w, h int, opts CodeBlockOptions) *CodeBlock {
	if opts.FgColor == 0 {
		opts.FgColor = 7
	}
	if opts.BgColor == 0 {
		opts.BgColor = 1
	}
	if opts.BorderColor == 0 {
		opts.BorderColor = 6
	}
	c := &CodeBlock{
		Element: &pgui.Element{
			Area: pixelforge.IntArea{X: x, Y: y, W: w, H: h},
		},
		Text:        opts.Text,
		FgColor:     opts.FgColor,
		BgColor:     opts.BgColor,
		BorderColor: opts.BorderColor,
	}
	c.Element.OnDraw = func(_ pgui.DrawEvent) { c.draw() }
	return c
}

// SetText replaces the rendered text.
func (c *CodeBlock) SetText(s string) {
	if c == nil {
		return
	}
	c.Text = s
}

// LineCount returns the number of lines in Text (split on '\n').
func (c *CodeBlock) LineCount() int {
	if c == nil || c.Text == "" {
		return 0
	}
	return strings.Count(c.Text, "\n") + 1
}

func (c *CodeBlock) draw() {
	prev := pixelforge.GetColor()
	defer pixelforge.SetColor(prev)

	pixelforge.SetColor(c.BgColor)
	pixelforge.RectFill(0, 0, c.W-1, c.H-1)
	pixelforge.SetColor(c.BorderColor)
	pixelforge.Rect(0, 0, c.W-1, c.H-1)

	font := pgui.DefaultFont()
	_, lineH := font.Measure("Ag")
	if lineH <= 0 {
		lineH = 8
	}
	pixelforge.SetColor(c.FgColor)
	y := 4
	for _, line := range strings.Split(c.Text, "\n") {
		if y > c.H-lineH {
			break
		}
		_, _ = font.Print(line, 4, y)
		y += lineH + 1
	}
}
