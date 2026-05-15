package pixelforge_routine_test

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge"
	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_routine"
	"github.com/stretchr/testify/assert"
)

func TestTween(t *testing.T) {
	t.Run("linear interpolation over n ticks", func(t *testing.T) {
		var v float64
		step := pixelforge_routine.Tween(&v, 0, 10, 5, pixelforge_routine.EaseLinear)
		assert.False(t, step()) // tick 1
		assert.InDelta(t, 2.0, v, 0.001)
		assert.False(t, step()) // tick 2
		assert.InDelta(t, 4.0, v, 0.001)
		assert.False(t, step()) // tick 3
		assert.InDelta(t, 6.0, v, 0.001)
		assert.False(t, step()) // tick 4
		assert.InDelta(t, 8.0, v, 0.001)
		assert.True(t, step()) // tick 5 done
		assert.InDelta(t, 10.0, v, 0.001)
	})

	t.Run("ticks=0 jumps to to immediately", func(t *testing.T) {
		v := 1.0
		step := pixelforge_routine.Tween(&v, 0, 5, 0, pixelforge_routine.EaseLinear)
		assert.True(t, step())
		assert.InDelta(t, 5.0, v, 0.001)
	})

	t.Run("from == to is a no-op", func(t *testing.T) {
		v := 3.0
		step := pixelforge_routine.Tween(&v, 3, 3, 10, pixelforge_routine.EaseLinear)
		assert.True(t, step())
		assert.InDelta(t, 3.0, v, 0.001)
	})

	t.Run("unknown easing falls back to linear with warning", func(t *testing.T) {
		var v float64
		step := pixelforge_routine.Tween(&v, 0, 10, 2, "bouncy-cubic-not-real")
		assert.False(t, step())
		assert.InDelta(t, 5.0, v, 0.001)
		assert.True(t, step())
		assert.InDelta(t, 10.0, v, 0.001)
	})

	t.Run("nil target advances immediately", func(t *testing.T) {
		step := pixelforge_routine.Tween(nil, 0, 10, 5, pixelforge_routine.EaseLinear)
		assert.True(t, step())
	})

	t.Run("ease-in arc grows slower then faster", func(t *testing.T) {
		var v float64
		step := pixelforge_routine.Tween(&v, 0, 100, 4, pixelforge_routine.EaseIn)
		step() // 0.25 -> 0.0625 * 100 = 6.25
		assert.InDelta(t, 6.25, v, 0.01)
	})
}

func TestMove(t *testing.T) {
	t.Run("advances by dx, dy over n ticks", func(t *testing.T) {
		pos := &pixelforge.Position{X: 10, Y: 20}
		step := pixelforge_routine.Move(pos, 10, -5, 5)
		for i := 0; i < 4; i++ {
			assert.False(t, step())
		}
		assert.True(t, step())
		assert.Equal(t, pixelforge.Position{X: 20, Y: 15}, *pos)
	})

	t.Run("zero ticks jumps immediately", func(t *testing.T) {
		pos := &pixelforge.Position{X: 1, Y: 2}
		step := pixelforge_routine.Move(pos, 4, 3, 0)
		assert.True(t, step())
		assert.Equal(t, pixelforge.Position{X: 5, Y: 5}, *pos)
	})

	t.Run("zero delta is no-op", func(t *testing.T) {
		pos := &pixelforge.Position{X: 7, Y: 8}
		step := pixelforge_routine.Move(pos, 0, 0, 10)
		assert.True(t, step())
		assert.Equal(t, pixelforge.Position{X: 7, Y: 8}, *pos)
	})

	t.Run("nil position advances immediately", func(t *testing.T) {
		step := pixelforge_routine.Move(nil, 5, 5, 10)
		assert.True(t, step())
	})
}

func TestPublish(t *testing.T) {
	t.Run("fires event exactly once on first tick", func(t *testing.T) {
		target := pievent.NewTarget[string]()
		received := []string{}
		target.SubscribeAll(func(e string, _ pievent.Handler) {
			received = append(received, e)
		})

		step := pixelforge_routine.Publish(target, "ping")
		assert.True(t, step())
		assert.Equal(t, []string{"ping"}, received)
	})

	t.Run("nil target is a no-op", func(t *testing.T) {
		var target pievent.Target[string]
		step := pixelforge_routine.Publish(target, "ping")
		assert.True(t, step())
	})
}

func TestBranch(t *testing.T) {
	t.Run("true predicate runs ifTrue", func(t *testing.T) {
		var ranTrue, ranFalse int
		ifTrue := []pixelforge_routine.Step{
			pixelforge_routine.Call(func() { ranTrue++ }),
		}
		ifFalse := []pixelforge_routine.Step{
			pixelforge_routine.Call(func() { ranFalse++ }),
		}
		step := pixelforge_routine.Branch(func() bool { return true }, ifTrue, ifFalse)
		assert.True(t, step())
		assert.Equal(t, 1, ranTrue)
		assert.Equal(t, 0, ranFalse)
	})

	t.Run("false predicate runs ifFalse", func(t *testing.T) {
		var ranTrue, ranFalse int
		ifTrue := []pixelforge_routine.Step{
			pixelforge_routine.Call(func() { ranTrue++ }),
		}
		ifFalse := []pixelforge_routine.Step{
			pixelforge_routine.Call(func() { ranFalse++ }),
		}
		step := pixelforge_routine.Branch(func() bool { return false }, ifTrue, ifFalse)
		assert.True(t, step())
		assert.Equal(t, 0, ranTrue)
		assert.Equal(t, 1, ranFalse)
	})

	t.Run("empty selected branch completes immediately", func(t *testing.T) {
		step := pixelforge_routine.Branch(func() bool { return true }, nil, nil)
		assert.True(t, step())
	})

	t.Run("multi-step ifTrue waits across ticks", func(t *testing.T) {
		var ran int
		ifTrue := []pixelforge_routine.Step{
			pixelforge_routine.Wait(2),
			pixelforge_routine.Call(func() { ran++ }),
		}
		step := pixelforge_routine.Branch(func() bool { return true }, ifTrue, nil)
		assert.False(t, step())
		assert.False(t, step())
		assert.True(t, step())
		assert.Equal(t, 1, ran)
	})

	t.Run("nil predicate falls through to ifFalse", func(t *testing.T) {
		var ranFalse int
		ifFalse := []pixelforge_routine.Step{
			pixelforge_routine.Call(func() { ranFalse++ }),
		}
		step := pixelforge_routine.Branch(nil, nil, ifFalse)
		assert.True(t, step())
		assert.Equal(t, 1, ranFalse)
	})
}

func TestComposedRoutine_Integration(t *testing.T) {
	target := pievent.NewTarget[string]()
	received := []string{}
	target.SubscribeAll(func(e string, _ pievent.Handler) {
		received = append(received, e)
	})

	var v float64
	pos := &pixelforge.Position{X: 0, Y: 0}
	var ranBranch int

	r := pixelforge_routine.New(
		pixelforge_routine.Wait(2),
		pixelforge_routine.Tween(&v, 0, 4, 4, pixelforge_routine.EaseLinear),
		pixelforge_routine.Publish(target, "checkpoint"),
		pixelforge_routine.Move(pos, 6, 0, 3),
		pixelforge_routine.Branch(func() bool { return true },
			[]pixelforge_routine.Step{
				pixelforge_routine.Call(func() { ranBranch++ }),
			}, nil),
	)
	for r.Resume() {
	}
	assert.Equal(t, []string{"checkpoint"}, received)
	assert.InDelta(t, 4.0, v, 0.001)
	assert.Equal(t, pixelforge.Position{X: 6, Y: 0}, *pos)
	assert.Equal(t, 1, ranBranch)
}
