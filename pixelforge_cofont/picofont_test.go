package pixelforge_cofont_test

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_cofont"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_test"
)

//go:embed "font.png"
var fontPNG []byte

func TestPrint(t *testing.T) {
	t.Run("should print each character", func(t *testing.T) {
		pixelforge.SetScreenSize(128, 128)

		pixelforge.Palette = pixelforge.DecodePalette(fontPNG)
		canvas := pixelforge.DecodeCanvas(fontPNG)

		var table strings.Builder

		// print narrow characters
		for i := 16; i < 128; i++ { // skip escape codes below 16 (such as LF)
			table.WriteRune(rune(i))
			table.WriteByte(' ')
			if i%16 == 15 {
				table.WriteByte('\n')
			}
		}
		// print wide characters
		for i := 128; i < 256; i++ {
			table.WriteRune(rune(i))
			if i%16 == 15 {
				table.WriteByte('\n')
			}
		}
		pixelforge.SetColor(1)
		// when
		picofont.Print(table.String(), 0, 8)
		// then
		pitest.AssertSurfaceEqual(t, canvas, pixelforge.Screen())
	})
}
