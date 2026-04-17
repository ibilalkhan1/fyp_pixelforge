package pixelforge_ring_test

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge/internal/pixelforge_ring"
)

func BenchmarkBuffer_NextWritePointer(b *testing.B) {
	b.ReportAllocs()
	// allocation takes place only here
	buffer := piring.NewBuffer[someStruct](4)
	for i := 0; i < buffer.Cap(); i++ {
		buffer.NextWritePointer().slice = make([]int, 128)
	}
	for b.Loop() {
		pointer := buffer.NextWritePointer() // no allocation here
		pointer.slice[0] = 1
	}
}

func BenchmarkBuffer_PointerTo(b *testing.B) {
	b.ReportAllocs()
	buffer := piring.NewBuffer[someStruct](4)
	buffer.NextWritePointer().slice = make([]int, 128)
	z := 0
	for b.Loop() {
		z += buffer.PointerTo(0).slice[0] // no allocations here
	}
	_ = z
}

type someStruct struct {
	slice []int
}
