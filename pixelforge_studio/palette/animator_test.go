package palette

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// OpenForSlot makes the animator visible and targets the slot.
func TestAnimator_OpenForSlot(t *testing.T) {
	a := NewAnimator()
	assert.False(t, a.Visible())
	a.OpenForSlot(8)
	assert.True(t, a.Visible())
	assert.Equal(t, 8, a.Slot())

	// Out-of-range slot does not change state.
	a.OpenForSlot(-1)
	assert.Equal(t, 8, a.Slot())
}

// AddKeyframe creates an animation entry the first time and inserts
// keyframes in time-sorted order.
func TestAnimator_AddKeyframeCreatesAnimation(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	a := NewAnimator()
	a.OpenForSlot(8)

	a.AddKeyframe(p, 1.0, "#00ff00")
	a.AddKeyframe(p, 0.0, "#ff0000")
	a.AddKeyframe(p, 0.5, "#0000ff")

	anim := a.Animation(p)
	require.NotNil(t, anim)
	require.Len(t, anim.Keyframes, 3)
	assert.Equal(t, 0.0, anim.Keyframes[0].Time)
	assert.Equal(t, 0.5, anim.Keyframes[1].Time)
	assert.Equal(t, 1.0, anim.Keyframes[2].Time)
}

// PreviewAt interpolates between keyframes by default.
func TestAnimator_PreviewAtLinearInterp(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	a := NewAnimator()
	a.OpenForSlot(8)
	a.AddKeyframe(p, 0.0, "#ff0000")
	a.AddKeyframe(p, 1.0, "#00ff00")

	// At t=0.5 the linear interpolation lands at #808000 (approx).
	got := a.PreviewAt(p, 0.5)
	c, ok := parseHexColor(got)
	require.True(t, ok)
	assert.InDelta(t, 127, int(c.R), 2)
	assert.InDelta(t, 127, int(c.G), 2)
	assert.Equal(t, uint8(0), c.B)
}

// step easing produces a discrete jump.
func TestAnimator_PreviewAtStepEasing(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	a := NewAnimator()
	a.OpenForSlot(8)
	a.AddKeyframe(p, 0.0, "#ff0000")
	a.AddKeyframe(p, 1.0, "#00ff00")
	a.SetEasing(p, "step")

	// Step easing keeps the lower keyframe's color across the interval —
	// the jump happens at the next keyframe's exact timestamp.
	assert.Equal(t, "#ff0000", a.PreviewAt(p, 0.49))
	assert.Equal(t, "#ff0000", a.PreviewAt(p, 0.99))
	// At or above the final keyframe, the clamp returns it.
	assert.Equal(t, "#00ff00", a.PreviewAt(p, 1.0))
	assert.Equal(t, "#00ff00", a.PreviewAt(p, 1.5))
}

// Empty animation returns the base slot color.
func TestAnimator_PreviewAtEmptyAnimation(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	a := NewAnimator()
	a.OpenForSlot(8)
	got := a.PreviewAt(p, 0.5)
	assert.Equal(t, p.Palette.Base[8], got)
}

// ClipLength setter validates positive values.
func TestAnimator_SetClipLength(t *testing.T) {
	a := NewAnimator()
	a.SetClipLength(5.0)
	assert.Equal(t, 5.0, a.ClipLength())
	a.SetClipLength(-1) // ignored
	assert.Equal(t, 5.0, a.ClipLength())
}

// applyEasing curves behave as documented at the boundaries.
func TestAnimator_EasingCurves(t *testing.T) {
	for _, easing := range []string{"linear", "ease_in", "ease_out", "ease_in_out", "step"} {
		assert.InDelta(t, 0, applyEasing(easing, 0), 0.001, easing)
	}
	// linear, ease_in, ease_out, ease_in_out all land at 1 at x=1.
	for _, easing := range []string{"linear", "ease_in", "ease_out", "ease_in_out"} {
		assert.InDelta(t, 1, applyEasing(easing, 1), 0.001, easing)
	}
	// Step stays at 0 across the interior.
	assert.Equal(t, 0.0, applyEasing("step", 0.5))
}
