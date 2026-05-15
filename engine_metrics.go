package pixelforge

import "time"

// Engine-internals counters and probes for visualization overlays
// (e.g. pixelforge_metrics dashboard panels). All values are read-only
// for inspection; modifying them externally is unsupported.

// PixelsWrittenThisFrame is incremented by every successful pixel write
// (after clip checks pass). Visualization overlays reset it at the start
// of each frame via ResetFrameCounters.
var PixelsWrittenThisFrame uint64

// ColorTableAccesses tracks how often each entry of ColorTables is read
// during a frame. The first index is the table index (0..3), the second
// is the draw color (0..MaxColors-1), and the third is the target color.
// Visualization overlays reset it at the start of each frame via ResetFrameCounters.
var ColorTableAccesses [4][MaxColors][MaxColors]uint64

// HeatMapBuffer is an optional per-pixel write density buffer sized to
// the current screen. When non-nil and matching the screen's pixel count,
// every successful pixel write to the screen increments the matching
// entry (saturating at 65535).
//
// Allocated lazily by visualization overlays; the engine only writes to it.
var HeatMapBuffer []uint16

// ResetFrameCounters zeroes the per-frame pixel and color-table counters.
// Visualization overlays call this from EventFrameStart so the values
// reflect a single frame of activity.
func ResetFrameCounters() {
	PixelsWrittenThisFrame = 0
	ColorTableAccesses = [4][MaxColors][MaxColors]uint64{}
}

// DecayHeatMap halves every entry in HeatMapBuffer.
// Visualization overlays call this at frame start so recent writes
// remain visible while old activity fades.
func DecayHeatMap() {
	for i := range HeatMapBuffer {
		HeatMapBuffer[i] >>= 1
	}
}

// drawingToScreen reports whether the current draw target is the screen.
// Used by per-pixel instrumentation so the heat map only reflects screen writes.
func drawingToScreen() bool {
	if len(drawTarget.data) == 0 || len(screen.data) == 0 {
		return false
	}
	return &drawTarget.data[0] == &screen.data[0]
}

// recordHeatMap increments HeatMapBuffer[idx] with saturation,
// when the buffer is non-nil and matches the screen size.
func recordHeatMap(idx int) {
	if HeatMapBuffer == nil {
		return
	}
	if !drawingToScreen() {
		return
	}
	if idx < 0 || idx >= len(HeatMapBuffer) {
		return
	}
	if HeatMapBuffer[idx] < 0xffff {
		HeatMapBuffer[idx]++
	}
}

var (
	inputDur, updateDur, drawDur, totalDur time.Duration
)

// SetFramePhaseDurations records the per-phase wall-clock durations for the
// most recent frame. Backends (e.g. pixelforge_ebiten) call this once per
// tick; overlays read the values via FramePhaseDurations.
func SetFramePhaseDurations(input, update, draw, total time.Duration) {
	inputDur = input
	updateDur = update
	drawDur = draw
	totalDur = total
}

// FramePhaseDurations returns the per-phase wall-clock durations for the
// most recent frame: input handling, pixelforge.Update, pixelforge.Draw,
// and the total tick duration.
func FramePhaseDurations() (input, update, draw, total time.Duration) {
	return inputDur, updateDur, drawDur, totalDur
}
