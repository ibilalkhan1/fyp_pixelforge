package internal

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/internal/pixelforge_ring"
)

var theScreenRecorder = newScreenRecorder(128)

// SetScreenSnapshotHistorySize sets the new number of screen snapshots held in memory
// Calling this method clears the screen snapshots history
func SetScreenSnapshotHistorySize(historySize int) {
	theScreenRecorder = newScreenRecorder(historySize)
}

func newScreenRecorder(historySize int) *screenRecorder {
	buffer := piring.NewBuffer[screenSnapshot](historySize)

	return &screenRecorder{snapshots: buffer}
}

type screenRecorder struct {
	snapshots *piring.Buffer[screenSnapshot]
	shift     int // which element from the end is currently selected
}

type screenSnapshot struct {
	canvas         pixelforge.Canvas
	paletteMapping pixelforge.PaletteMap
	palette        pixelforge.PaletteArray
}

func (s *screenRecorder) Save() {
	snapshot := s.snapshots.NextWritePointer()
	screen := pixelforge.Screen()
	// reuse canvas if possible
	if snapshot.canvas.W() == screen.W() && snapshot.canvas.H() == screen.H() {
		snapshot.canvas.SetData(screen.Data())
	} else {
		snapshot.canvas = screen.Clone()
	}
	snapshot.palette = pixelforge.Palette
	snapshot.paletteMapping = pixelforge.PaletteMapping

	s.shift = 0
}

func (s *screenRecorder) HasPrev() bool {
	return -s.shift+1 <= s.snapshots.Len()
}

func (s *screenRecorder) ShowPrev() bool {
	if -s.shift+1 > s.snapshots.Len() {
		return false
	}
	s.shift -= 1
	s.showCurrent()
	return true
}

func (s *screenRecorder) ShowNext() bool {
	if s.shift >= -1 {
		return false
	}
	s.shift += 1
	s.showCurrent()
	return true
}

func (s *screenRecorder) showCurrent() {
	snapshot := s.snapshots.PointerTo(s.snapshots.Len() + s.shift)
	pixelforge.Screen().SetData(snapshot.canvas.Data())
	pixelforge.Palette = snapshot.palette
	pixelforge.PaletteMapping = snapshot.paletteMapping
}

func (s *screenRecorder) Reset() {
	s.snapshots.Reset()
	s.shift = 0
}

func (s *screenRecorder) GoToLast() {
	s.shift = 0
}
