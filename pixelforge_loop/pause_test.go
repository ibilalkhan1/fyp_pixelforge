package pixelforge_loop_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
)

// pause_test.go covers idea #6 v1 U3's gate primitive. The gate is
// package-scoped so each test must Resume() in cleanup to avoid
// leaking pause state into the next test.

func resetPause(t *testing.T) {
	t.Helper()
	t.Cleanup(piloop.Resume)
	piloop.Resume()
}

func TestPause_IdempotentMultipleCalls(t *testing.T) {
	resetPause(t)
	piloop.Pause()
	piloop.Pause()
	assert.True(t, piloop.IsPaused())
}

func TestResume_IdempotentMultipleCalls(t *testing.T) {
	resetPause(t)
	piloop.Pause()
	piloop.Resume()
	piloop.Resume()
	assert.False(t, piloop.IsPaused())
}

func TestIsPaused_DefaultFalse(t *testing.T) {
	resetPause(t)
	assert.False(t, piloop.IsPaused(),
		"package gate starts in the running state")
}

func TestSuppressedWhilePaused_GateClosed(t *testing.T) {
	resetPause(t)
	assert.False(t, piloop.SuppressedWhilePaused(piloop.EventUpdate))
	assert.False(t, piloop.SuppressedWhilePaused(piloop.EventLateUpdate))
	assert.False(t, piloop.SuppressedWhilePaused(piloop.EventLateDraw))
}

func TestSuppressedWhilePaused_GateOpenBlocksUpdates(t *testing.T) {
	resetPause(t)
	piloop.Pause()
	assert.True(t, piloop.SuppressedWhilePaused(piloop.EventUpdate),
		"EventUpdate suppressed while paused")
	assert.True(t, piloop.SuppressedWhilePaused(piloop.EventLateUpdate),
		"EventLateUpdate suppressed while paused")
}

func TestSuppressedWhilePaused_AllowsDrawAndInputWhilePaused(t *testing.T) {
	resetPause(t)
	piloop.Pause()
	assert.False(t, piloop.SuppressedWhilePaused(piloop.EventDraw),
		"EventDraw must pass through while paused (overlays render)")
	assert.False(t, piloop.SuppressedWhilePaused(piloop.EventLateDraw),
		"EventLateDraw must pass through while paused")
	assert.False(t, piloop.SuppressedWhilePaused(piloop.EventFrameStart),
		"EventFrameStart passes through")
	assert.False(t, piloop.SuppressedWhilePaused(piloop.EventInit),
		"EventInit passes through")
}

func TestPauseResume_RestartsUpdateFlow(t *testing.T) {
	resetPause(t)
	piloop.Pause()
	assert.True(t, piloop.SuppressedWhilePaused(piloop.EventUpdate))
	piloop.Resume()
	assert.False(t, piloop.SuppressedWhilePaused(piloop.EventUpdate),
		"Resume restores EventUpdate dispatch")
}
