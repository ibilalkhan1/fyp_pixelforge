package pixelforge_metrics_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_metrics"
)

func TestRenderMode_Bitmask(t *testing.T) {
	t.Run("RenderAll includes every panel except heat map", func(t *testing.T) {
		all := pixelforge_metrics.RenderAll
		assert.NotZero(t, all&pixelforge_metrics.RenderTextMetrics)
		assert.NotZero(t, all&pixelforge_metrics.RenderInputs)
		assert.NotZero(t, all&pixelforge_metrics.RenderBudget)
		assert.NotZero(t, all&pixelforge_metrics.RenderAudio)
		assert.NotZero(t, all&pixelforge_metrics.RenderEventBus)
		assert.NotZero(t, all&pixelforge_metrics.RenderColorTable)
	})

	t.Run("Mode = 0 disables every panel", func(t *testing.T) {
		var disabled pixelforge_metrics.RenderMode
		assert.Zero(t, disabled&pixelforge_metrics.RenderTextMetrics)
		assert.Zero(t, disabled&pixelforge_metrics.RenderBudget)
		assert.Zero(t, disabled&pixelforge_metrics.RenderColorTable)
	})
}

func TestEngineCounters_Reset(t *testing.T) {
	pixelforge.PixelsWrittenThisFrame = 42
	pixelforge.ColorTableAccesses[0][1][2] = 99
	pixelforge.ResetFrameCounters()
	assert.Equal(t, uint64(0), pixelforge.PixelsWrittenThisFrame)
	assert.Equal(t, uint64(0), pixelforge.ColorTableAccesses[0][1][2])
}

func TestHeatMap_Decay(t *testing.T) {
	pixelforge.HeatMapBuffer = []uint16{0, 1, 2, 100, 0xffff}
	pixelforge.DecayHeatMap()
	assert.Equal(t, []uint16{0, 0, 1, 50, 0x7fff}, pixelforge.HeatMapBuffer)
	pixelforge.HeatMapBuffer = nil
}

func TestEngineCounters_TrackDraws(t *testing.T) {
	pixelforge.SetScreenSize(16, 16)
	pixelforge.SetDrawTarget(pixelforge.Screen())
	pixelforge.ResetColorTables()
	pixelforge.ResetFrameCounters()

	pixelforge.SetColor(7)
	pixelforge.RectFill(0, 0, 7, 7) // 64 pixels

	assert.Equal(t, uint64(64), pixelforge.PixelsWrittenThisFrame,
		"RectFill should increment the per-frame pixel counter")

	pixelforge.ResetFrameCounters()
	assert.Zero(t, pixelforge.PixelsWrittenThisFrame)
}

func TestHeatMap_TracksScreenWrites(t *testing.T) {
	pixelforge.SetScreenSize(8, 4)
	pixelforge.SetDrawTarget(pixelforge.Screen())
	pixelforge.ResetColorTables()
	pixelforge.HeatMapBuffer = make([]uint16, 8*4)
	defer func() { pixelforge.HeatMapBuffer = nil }()

	pixelforge.SetColor(7)
	pixelforge.SetPixel(2, 1)
	pixelforge.SetPixel(2, 1)
	pixelforge.SetPixel(2, 1)

	idx := 1*8 + 2
	assert.Equal(t, uint16(3), pixelforge.HeatMapBuffer[idx],
		"repeated SetPixel should accumulate heat map count")
}

func TestHeatMap_Saturation(t *testing.T) {
	pixelforge.SetScreenSize(2, 1)
	pixelforge.SetDrawTarget(pixelforge.Screen())
	pixelforge.HeatMapBuffer = []uint16{0xfffe, 0xffff}
	defer func() { pixelforge.HeatMapBuffer = nil }()

	pixelforge.SetColor(7)
	pixelforge.SetPixel(0, 0)
	pixelforge.SetPixel(0, 0) // would overflow without saturation
	pixelforge.SetPixel(1, 0) // already saturated, must stay capped

	assert.Equal(t, uint16(0xffff), pixelforge.HeatMapBuffer[0])
	assert.Equal(t, uint16(0xffff), pixelforge.HeatMapBuffer[1])
}

func TestFramePhaseDurations_RoundTrip(t *testing.T) {
	defer pixelforge.SetFramePhaseDurations(0, 0, 0, 0)

	pixelforge.SetFramePhaseDurations(1, 2, 3, 6)
	in, up, dr, tot := pixelforge.FramePhaseDurations()
	assert.EqualValues(t, 1, in)
	assert.EqualValues(t, 2, up)
	assert.EqualValues(t, 3, dr)
	assert.EqualValues(t, 6, tot)
}
