package pixelforge_font_test

import (
	_ "embed"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_test"
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_font"
)

//go:embed internal/test/font.png
var fontPNG []byte

var fontSheet pifont.Sheet

func init() {
	prevPalette := pixelforge.Palette
	defer func() {
		pixelforge.Palette = prevPalette
	}()

	pixelforge.Palette = pixelforge.DecodePalette(fontPNG)
	fontCanvas := pixelforge.DecodeCanvas(fontPNG)
	fontSheet = pifont.Sheet{
		Chars: map[rune]pixelforge.Sprite{
			'S': {
				Area:   pixelforge.Area[int]{X: 0, Y: 0, W: 8, H: 8},
				Source: fontCanvas,
			},
			'T': {
				Area:   pixelforge.Area[int]{X: 8, Y: 0, W: 8, H: 8},
				Source: fontCanvas,
			},
			'⬤': {
				Area:   pixelforge.Area[int]{X: 0, Y: 8, W: 8, H: 8},
				Source: fontCanvas,
			},
			'❤': {
				Area:   pixelforge.Area[int]{X: 8, Y: 8, W: 8, H: 8},
				Source: fontCanvas,
			},
		},
		Height:  8,
		FgColor: 1,
		BgColor: 0,
	}
}

var (
	//go:embed internal/test/text-color-equal-to-bg.png
	textColorEqualToBg []byte
	//go:embed internal/test/text-color-equal-to-fg.png
	textColorEqualToFg []byte
	//go:embed internal/test/text-color-different.png
	textColorDifferent []byte
)

func TestSheet_Print(t *testing.T) {
	t.Run("should print with different colors", func(t *testing.T) {
		tests := map[string]struct {
			bgColor   pixelforge.Color
			textColor pixelforge.Color
			png       []byte
		}{
			"text color different than Bg and Fg": {
				bgColor:   0,
				textColor: 2,
				png:       textColorDifferent,
			},
			"text color equal to Bg": {
				bgColor:   2,
				textColor: 0,
				png:       textColorEqualToBg,
			},
			"text color equal to Fg": {
				bgColor:   0,
				textColor: 1,
				png:       textColorEqualToFg,
			},
		}

		for testName, testCase := range tests {
			t.Run(testName, func(t *testing.T) {
				pixelforge.Palette = pixelforge.DecodePalette(testCase.png)
				expectedCanvas := pixelforge.DecodeCanvas(testCase.png)
				pixelforge.SetScreenSize(8, 8)

				pixelforge.Screen().Clear(testCase.bgColor)
				pixelforge.SetColor(testCase.textColor)
				pixelforge.SetTransparency(testCase.textColor, false)
				// when
				fontSheet.Print("S", 0, 0)
				// then
				pitest.AssertSurfaceEqual(t, expectedCanvas, pixelforge.Screen())
			})
		}
	})
}

func BenchmarkSheet_Print(b *testing.B) {
	sheet := pifont.Sheet{
		Chars: map[rune]pixelforge.Sprite{
			'a': {
				Area:   pixelforge.Area[int]{X: 0, Y: 0, W: 8, H: 8},
				Source: pixelforge.NewCanvas(8, 8),
			},
		},
	}
	b.ReportAllocs()
	for b.Loop() {
		sheet.Print("aaaaaaaaaaaaaaaaaaaa", 100, 100)
	}
}
