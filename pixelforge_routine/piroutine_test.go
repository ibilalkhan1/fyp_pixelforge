package pixelforge_routine_test

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_routine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPiroutine(t *testing.T) {
	r := pixelforge_routine.New(
		pixelforge_routine.Wait(2),
		pixelforge_routine.Printf("abc"),
		pixelforge_routine.Wait(5),
	)
	r.SetTracing(true)
	for r.Resume() {
	}
}

func TestRoutine_Resume(t *testing.T) {
	t.Run("empty routine", func(t *testing.T) {
		routine := pixelforge_routine.New()
		assert.False(t, routine.Resume())
	})

	t.Run("single step", func(t *testing.T) {
		var stepExecuted bool
		step := func() bool {
			stepExecuted = true
			return true // finish immediately
		}
		routine := pixelforge_routine.New(step)
		assert.False(t, routine.Resume()) // finished
		assert.True(t, stepExecuted)
	})

	t.Run("wait step", func(t *testing.T) {
		routine := pixelforge_routine.New(pixelforge_routine.Wait(1))
		assert.True(t, routine.Resume())  // not yet finished
		assert.False(t, routine.Resume()) // finished
		assert.False(t, routine.Resume()) // nothing changed
	})
}

func TestCall(t *testing.T) {
	t.Run("should call callback", func(t *testing.T) {
		var executed = false
		step := pixelforge_routine.Call(func() {
			executed = true
		})
		// when
		result := step()
		// then
		assert.True(t, executed)
		assert.True(t, result)
	})
}

func TestSlowDown(t *testing.T) {
	t.Run("should wait n updates before running callback", func(t *testing.T) {
		executionCount := 0
		step := pixelforge_routine.SlowDown(2, func() bool {
			executionCount++
			return true
		})
		assert.False(t, step())
		assert.Equal(t, 0, executionCount)
		assert.False(t, step())
		assert.Equal(t, 0, executionCount)
		assert.True(t, step())
		assert.Equal(t, 1, executionCount)
	})

	t.Run("should immediately run callback", func(t *testing.T) {
		executionCount := 0
		step := pixelforge_routine.SlowDown(0, func() bool {
			executionCount++
			return true
		})
		assert.True(t, step())
		assert.Equal(t, 1, executionCount)
	})

	t.Run("should wait another n updates after callback returned false", func(t *testing.T) {
		executionCount := 0
		step := pixelforge_routine.SlowDown(3, func() bool {
			executionCount++
			return executionCount%2 == 0
		})
		for range 3 {
			assert.False(t, step()) // wait
		}
		assert.False(t, step()) // callback returns false
		for range 3 {
			assert.False(t, step()) // wait
		}
		assert.True(t, step()) // callback returns true this time
		assert.Equal(t, 2, executionCount)
	})
}

func TestRoutine_ScheduleOn(t *testing.T) {
	t.Run("should run callback on event", func(t *testing.T) {
		executionCount := 0
		step := func() bool {
			executionCount++
			return true
		}
		routine := pixelforge_routine.New(step)
		// when
		handler := routine.ScheduleOn(pixelforge_loop.EventDraw)
		// then
		require.True(t, pixelforge_loop.Target().IsSubscribed(handler))
		pixelforge_loop.Target().Publish(pixelforge_loop.EventDraw) // runs callback and unsubscribes handlers
		assert.Equal(t, 1, executionCount)
		assert.False(t, pixelforge_loop.Target().IsSubscribed(handler))
	})
}
