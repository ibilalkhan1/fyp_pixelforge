package pixelforge_gui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
)

// labelTestSetup configures a fresh 128x64 canvas so each test starts
// from a known-empty surface.
func labelTestSetup(t *testing.T) pixelforge.Canvas {
	t.Helper()
	pixelforge.SetScreenSize(128, 64)
	pixelforge.Cls()
	return pixelforge.Screen()
}

func TestDefaultFontMeasure(t *testing.T) {
	font := pgui.DefaultFont()
	t.Run("empty string measures zero width with full line height", func(t *testing.T) {
		w, h := font.Measure("")
		assert.Equal(t, 0, w)
		assert.Equal(t, pixelforge_cofont.Sheet.Height, h)
	})
	t.Run("ascii text measures the cofont sheet widths", func(t *testing.T) {
		w, h := font.Measure("READY")
		// Each ASCII glyph is 4px wide in the cofont sheet.
		expectedW := 0
		for _, r := range "READY" {
			expectedW += pixelforge_cofont.Sheet.Chars[r].W
		}
		assert.Equal(t, expectedW, w)
		assert.Equal(t, pixelforge_cofont.Sheet.Height, h)
	})
	t.Run("multi-line text reports the widest line and stacked heights", func(t *testing.T) {
		w, h := font.Measure("AB\nCDE")
		// Two lines stacked.
		assert.Equal(t, 2*pixelforge_cofont.Sheet.Height, h)
		// Width is the wider of "AB" and "CDE".
		require.Greater(t, w, 0)
	})
	t.Run("line height matches the cofont sheet height", func(t *testing.T) {
		assert.Equal(t, pixelforge_cofont.Sheet.Height, font.LineHeight())
	})
}

// hasNonZeroPixels reports whether the canvas contains any non-zero
// pixel inside the rect (cleared canvases are all-zero so any rendered
// text shows up as non-zero).
func hasNonZeroPixels(c pixelforge.Canvas, x, y, w, h int) bool {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			if c.Get(i, j) != 0 {
				return true
			}
		}
	}
	return false
}

func TestLabel_Draw(t *testing.T) {
	t.Run("renders non-empty text at the configured position", func(t *testing.T) {
		canvas := labelTestSetup(t)

		root := pgui.New()
		lbl := pgui.NewLabel(10, 10, 60, 16, pgui.LabelOptions{
			Text:    "READY",
			FgColor: 7,
		})
		root.Attach(lbl.Element)

		root.Draw()

		// Cofont glyphs are 4x8; "READY" fits inside x=10..30, y=10..18.
		assert.True(t, hasNonZeroPixels(canvas, 10, 10, 30, 8),
			"expected READY text to leave non-zero pixels inside the label area")
	})

	t.Run("empty text is a no-op", func(t *testing.T) {
		canvas := labelTestSetup(t)

		root := pgui.New()
		lbl := pgui.NewLabel(0, 0, 60, 16, pgui.LabelOptions{Text: ""})
		root.Attach(lbl.Element)

		root.Draw()

		assert.False(t, hasNonZeroPixels(canvas, 0, 0, 60, 16),
			"empty label must not write any pixels")
	})

	t.Run("text outside the parent area is clipped", func(t *testing.T) {
		canvas := labelTestSetup(t)

		root := pgui.New()
		// Parent area is only 8 px wide; "WIDETEXT" should be clipped.
		parent := pgui.Attach(root, 5, 5, 8, 16)
		lbl := pgui.NewLabel(0, 0, 8, 16, pgui.LabelOptions{
			Text:    "WIDETEXT",
			FgColor: 7,
		})
		parent.Attach(lbl.Element)

		root.Draw()

		// Pixels outside the parent's clip stay zero.
		assert.False(t, hasNonZeroPixels(canvas, 14, 5, 30, 8),
			"text past the parent's right edge must be clipped")
	})

	t.Run("center alignment positions text near the middle of the label", func(t *testing.T) {
		canvas := labelTestSetup(t)

		root := pgui.New()
		lbl := pgui.NewLabel(0, 0, 32, 8, pgui.LabelOptions{
			Text:    "OK",
			FgColor: 7,
			Align:   pgui.AlignCenter,
		})
		root.Attach(lbl.Element)

		root.Draw()

		// Left half (x=0..6) should be empty; the centered "OK" lives
		// near x=12-16.
		assert.False(t, hasNonZeroPixels(canvas, 0, 0, 6, 8),
			"center-aligned text must not write at the left edge")
		assert.True(t, hasNonZeroPixels(canvas, 6, 0, 20, 8),
			"center-aligned text must write near the middle")
	})
}

func TestLabel_SetText(t *testing.T) {
	labelTestSetup(t)
	lbl := pgui.NewLabel(0, 0, 40, 8, pgui.LabelOptions{Text: "OLD"})
	assert.Equal(t, "OLD", lbl.Text)
	lbl.SetText("NEW")
	assert.Equal(t, "NEW", lbl.Text)
}

func TestLabel_OutOfSheetGlyph(t *testing.T) {
	// An out-of-sheet rune should not panic; cofont's char map returns
	// an empty sprite for unknown runes, which is harmless to print.
	labelTestSetup(t)
	root := pgui.New()
	lbl := pgui.NewLabel(0, 0, 60, 8, pgui.LabelOptions{
		Text:    "AB" + string([]rune{0x1F600}) + "CD", // emoji glyph
		FgColor: 7,
	})
	root.Attach(lbl.Element)
	require.NotPanics(t, func() { root.Draw() })
}
