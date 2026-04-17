package bench_test

import (
	"math/rand"
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge"
)

func BenchmarkDrawSprite(b *testing.B) {
	pixelforge.SetScreenSize(256, 256)
	canvas := pixelforge.NewCanvas(256, 256)
	for i := 0; i < canvas.W()*canvas.H(); i++ {
		canvas.Data()[i] = pixelforge.Color(rand.Intn(256))
	}

	sprite := pixelforge.SpriteFrom(canvas, 128, 128, 16, 16) //  396.6 (1 color table), 510 (4), vs 559 (ReadMask and TargetMask)

	for b.Loop() {
		pixelforge.DrawSprite(sprite, 10, 10)
	}
}

func BenchmarkLine(b *testing.B) {
	pixelforge.SetScreenSize(256, 256)
	for b.Loop() {
		pixelforge.Line(64, 64, 30, 30) // 94
	}
}

func BenchmarkRect(b *testing.B) {
	pixelforge.SetScreenSize(256, 256)
	b.ReportAllocs()
	for b.Loop() {
		pixelforge.RectFill(64, 64, 128, 128) // 2771 (one color table) vs 4337 (4 color tables) vs 4833 (Write and ReadMask)
	}
}
