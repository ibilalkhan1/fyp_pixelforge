package pixelforge_mouse_test

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_mouse"
	"github.com/stretchr/testify/assert"
)

func TestDuration(t *testing.T) {
	t.Run("should return 0 by default", func(t *testing.T) {
		assert.Equal(t, 0, pimouse.Duration(pimouse.Left))
	})

	pimouse.ButtonTarget().Publish(pimouse.EventButton{
		Button: pimouse.Left,
		Type:   pimouse.EventButtonDown,
	})

	t.Run("should return 1 when button was down in the current frame", func(t *testing.T) {
		assert.Equal(t, 1, pimouse.Duration(pimouse.Left))
	})

	pixelforge.Frame++

	t.Run("should return 2 when button has been down since the previous frame", func(t *testing.T) {
		assert.Equal(t, 2, pimouse.Duration(pimouse.Left))
	})

	pimouse.ButtonTarget().Publish(pimouse.EventButton{
		Button: pimouse.Left,
		Type:   pimouse.EventButtonUp,
	})

	t.Run("should return 0 when button is up", func(t *testing.T) {
		assert.Equal(t, 0, pimouse.Duration(pimouse.Left))
	})
}

func TestPosition(t *testing.T) {
	expected := pixelforge.Position{X: 1, Y: 2}
	event := pimouse.EventMove{
		Position: expected,
		Previous: pixelforge.Position{X: 3, Y: 5},
	}
	// when
	pimouse.MoveTarget().Publish(event)
	// then
	assert.Equal(t, expected, pimouse.Position)
	assert.Equal(t, pixelforge.Position{X: -2, Y: -3}, pimouse.MovementDelta)
}
