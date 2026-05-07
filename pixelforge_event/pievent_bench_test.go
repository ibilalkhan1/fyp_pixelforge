package pixelforge_event_test

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
)

func init() {
	pixelforge_event.GlobalTracingOff = true
}

func BenchmarkPublish(b *testing.B) {
	b.ReportAllocs()
	target := pixelforge_event.NewTarget[someEvent]()
	event := someEvent{a: "1"} // event should not be empty for get meaningful results
	target.SubscribeAll(func(someEvent, pixelforge_event.Handler) {})

	for b.Loop() {
		target.Publish(event) // zero alokacji! LOVE IT
	}
}

func BenchmarkSubscribe(b *testing.B) {
	b.ReportAllocs()
	target := pixelforge_event.NewTarget[someEvent]()
	listener := func(someEvent, pixelforge_event.Handler) {}

	for b.Loop() {
		// 3 allocs, because stack trace is analyzed - only for debugging
		// 0 allocs for production code
		target.SubscribeAll(listener)
	}
}

func BenchmarkSubscribeEvent(b *testing.B) {
	b.ReportAllocs()
	target := pixelforge_event.NewTarget[someEvent]()
	listener := func(someEvent, pixelforge_event.Handler) {}

	for b.Loop() {
		// 3 allocs, because stack trace is analyzed - only for debugging
		// 0 allocs for production code
		target.Subscribe(someEvent{a: "a"}, listener)
	}
}

type someEvent struct {
	a string
}
