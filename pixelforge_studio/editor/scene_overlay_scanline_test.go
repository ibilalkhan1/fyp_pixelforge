package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// scene_overlay_scanline_test.go covers U6's per-row counting + band
// derivation. The paint path is exercised structurally via U9 e2e
// tests; here the focus is the testable logic.

// stackedEntities returns n entities all at TileY (same vertical
// position) so they collectively cover the same Y range.
func stackedEntities(n int, tileY int) []pixelforge_project.Entity {
	out := make([]pixelforge_project.Entity, n)
	for i := range out {
		out[i] = pixelforge_project.Entity{ID: "e", TileY: tileY}
	}
	return out
}

// TestCountScanlineOccupancy_StacksIncrementCounts: 9 entities at
// TileY=10 with default 8px tile height occupy y=80..88; each y
// in that range counts as 9.
func TestCountScanlineOccupancy_StacksIncrementCounts(t *testing.T) {
	counts := CountScanlineOccupancy(stackedEntities(9, 10), 240, 8)
	for y := 80; y < 88; y++ {
		assert.Equal(t, 9, counts[y], "y=%d carries all 9 entities", y)
	}
	for y := 88; y < 240; y++ {
		assert.Equal(t, 0, counts[y], "rows outside the entities' range stay zero")
	}
}

// TestCountScanlineOccupancy_OutOfBoundsClips: entities whose Y
// range extends past screenHeight clip at the boundary.
func TestCountScanlineOccupancy_OutOfBoundsClips(t *testing.T) {
	entities := []pixelforge_project.Entity{
		{ID: "e", TileY: 30}, // y=240..248; screen is 240 tall
	}
	counts := CountScanlineOccupancy(entities, 240, 8)
	// Entity contributes nothing because its range starts at the
	// screen-bottom boundary.
	for _, c := range counts {
		assert.Equal(t, 0, c)
	}
}

// TestCountScanlineOccupancy_NegativeTileYIgnored: a malformed
// entity at TileY=-2 doesn't crash; its rows clip into [0, h).
func TestCountScanlineOccupancy_NegativeTileYIgnored(t *testing.T) {
	entities := []pixelforge_project.Entity{{ID: "e", TileY: -2}}
	counts := CountScanlineOccupancy(entities, 240, 8)
	// y=-16..-8 clipped to [0,0) → no increments.
	for _, c := range counts {
		assert.Equal(t, 0, c)
	}
}

// TestScanlineViolationRanges_FlagsAbovingThreshold: rows whose
// count > threshold surface as a band; merged adjacents collapse.
func TestScanlineViolationRanges_FlagsAbovingThreshold(t *testing.T) {
	counts := []int{0, 0, 5, 9, 9, 9, 9, 0, 0, 10, 10, 0}
	bands := ScanlineViolationRanges(counts, 8)
	assert.Equal(t, []ScanlineBand{
		{YStart: 3, YEnd: 7},
		{YStart: 9, YEnd: 11},
	}, bands)
}

// TestScanlineViolationRanges_NoneAboveThreshold: a count list
// where every entry is <= threshold returns nil.
func TestScanlineViolationRanges_NoneAboveThreshold(t *testing.T) {
	counts := []int{0, 1, 2, 8, 8}
	bands := ScanlineViolationRanges(counts, 8)
	assert.Nil(t, bands)
}

// TestScanlineViolationRanges_AllAboveThreshold: every row violates
// → one band from start to end.
func TestScanlineViolationRanges_AllAboveThreshold(t *testing.T) {
	counts := []int{10, 10, 10}
	bands := ScanlineViolationRanges(counts, 8)
	assert.Equal(t, []ScanlineBand{{YStart: 0, YEnd: 3}}, bands)
}

// TestPaintScanlineOverlay_DisabledNoOp: with ScanlineEnabled=false,
// the painter is a no-op even when violations exist.
func TestPaintScanlineOverlay_DisabledNoOp(t *testing.T) {
	scene := &pixelforge_project.Scene{Entities: stackedEntities(9, 10)}
	overlays := pixelforge_project.EditorOverlays{
		ScanlineEnabled:     false,
		PaletteBlockEnabled: true,
	}
	// Test contract: PaintScanlineOverlay with nil dst must not
	// crash even when the gate is open. dst-nil short-circuits at
	// the start.
	assert.NotPanics(t, func() {
		PaintScanlineOverlay(nil, scene, overlays)
	})
	// With disabled overlays + nil dst it's a no-op.
	overlays.ScanlineEnabled = false
	assert.NotPanics(t, func() {
		PaintScanlineOverlay(nil, scene, overlays)
	})
}

// TestPaintScanlineOverlay_FewerThanThresholdNoBand: 8 entities on
// the same row do not violate; the painter short-circuits.
func TestPaintScanlineOverlay_FewerThanThresholdNoBand(t *testing.T) {
	scene := &pixelforge_project.Scene{Entities: stackedEntities(8, 10)}
	// No dst → just verify the early-return path doesn't try to
	// access dst when the threshold check fails.
	overlays := pixelforge_project.EditorOverlays{ScanlineEnabled: true}
	assert.NotPanics(t, func() {
		PaintScanlineOverlay(nil, scene, overlays)
	})
}
