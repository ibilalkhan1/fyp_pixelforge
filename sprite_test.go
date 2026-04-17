package pixelforge_test

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge"
)

func TestStretch(t *testing.T) {
	// temporary test
	dst := pixelforge.NewCanvas(16, 16)
	pixelforge.SetDrawTarget(dst)

	src := pixelforge.NewCanvas(8, 8)
	src.Clear(7)

	spr := pixelforge.CanvasSprite(src)

	pixelforge.Stretch(spr, 0, 0, 8, 8)
	pixelforge.Stretch(spr, -1, 0, 8, 8)
	pixelforge.Stretch(spr, 0, -1, 8, 8)
	pixelforge.Stretch(spr, 16, 0, 8, 8)
	pixelforge.Stretch(spr, 0, 16, 8, 8)

	pixelforge.Stretch(spr.WithFlipX(true), 0, 0, 8, 8)
	pixelforge.Stretch(spr.WithFlipY(true), 0, 0, 8, 8)

	pixelforge.Stretch(spr.WithSize(0, 0), 0, 0, 8, 8)
}
