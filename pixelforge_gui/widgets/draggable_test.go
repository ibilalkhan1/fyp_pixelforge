package widgets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDraggable_PressInsideRegionStartsDrag(t *testing.T) {
	started := false
	d := NewDraggable(IntRect{X: 10, Y: 10, W: 20, H: 20})
	d.OnDragStart = func() { started = true }
	assert.True(t, d.Press(15, 15))
	assert.True(t, started)
	assert.True(t, d.Pressed())
}

func TestDraggable_PressOutsideRegionIsNoOp(t *testing.T) {
	d := NewDraggable(IntRect{X: 10, Y: 10, W: 20, H: 20})
	assert.False(t, d.Press(0, 0))
	assert.False(t, d.Pressed())
}

func TestDraggable_MoveEmitsDelta(t *testing.T) {
	var dx, dy int
	d := NewDraggable(IntRect{X: 0, Y: 0, W: 100, H: 100})
	d.OnDrag = func(ddx, ddy int) { dx += ddx; dy += ddy }
	d.Press(10, 10)
	d.Move(20, 15)
	assert.Equal(t, 10, dx)
	assert.Equal(t, 5, dy)
	d.Move(25, 17)
	assert.Equal(t, 15, dx)
	assert.Equal(t, 7, dy)
}

func TestDraggable_ReleaseClearsPressedAndFiresEnd(t *testing.T) {
	ended := false
	d := NewDraggable(IntRect{X: 0, Y: 0, W: 100, H: 100})
	d.OnDragEnd = func() { ended = true }
	d.Press(10, 10)
	assert.True(t, d.Release())
	assert.True(t, ended)
	assert.False(t, d.Pressed())

	// Second Release is a no-op.
	assert.False(t, d.Release())
}

func TestDraggable_MoveWithoutPressIgnored(t *testing.T) {
	calls := 0
	d := NewDraggable(IntRect{X: 0, Y: 0, W: 100, H: 100})
	d.OnDrag = func(int, int) { calls++ }
	d.Move(10, 10)
	assert.Equal(t, 0, calls)
}

func TestDraggable_CumulativeDelta(t *testing.T) {
	d := NewDraggable(IntRect{X: 0, Y: 0, W: 100, H: 100})
	d.Press(10, 10)
	d.Move(15, 12)
	d.Move(20, 15)
	assert.Equal(t, 10, d.CumulativeDX())
	assert.Equal(t, 5, d.CumulativeDY())
	d.Release()
	assert.Equal(t, 0, d.CumulativeDX(), "cumulative resets after release")
}

func TestDraggable_SetRegionUpdatesHitTest(t *testing.T) {
	d := NewDraggable(IntRect{X: 0, Y: 0, W: 10, H: 10})
	d.SetRegion(IntRect{X: 100, Y: 100, W: 10, H: 10})
	assert.False(t, d.Press(5, 5))
	assert.True(t, d.Press(105, 105))
}
