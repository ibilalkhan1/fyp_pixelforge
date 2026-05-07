package pixelforge_key_test

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_key"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	aDownEvent    = pixelforge_key.Event{Type: pixelforge_key.EventDown, Key: pixelforge_key.A}
	aUpEvent      = pixelforge_key.Event{Type: pixelforge_key.EventUp, Key: pixelforge_key.A}
	ctrlDownEvent = pixelforge_key.Event{Type: pixelforge_key.EventDown, Key: pixelforge_key.Ctrl}
)

func TestRegisterShortcut(t *testing.T) {
	t.Run("single key", func(t *testing.T) {
		executionTimes := 0
		shortcut := pixelforge_key.RegisterShortcut(func() {
			executionTimes++
		}, pixelforge_key.A)
		require.NotNil(t, shortcut)

		pixelforge_key.Target().Publish(aDownEvent)
		pixelforge_loop.Target().Publish(pixelforge_loop.EventLateUpdate)

		t.Run("should execute callback when key is pressed", func(t *testing.T) {
			assert.Equal(t, 1, executionTimes)
		})

		executionTimes = 0

		pixelforge_loop.Target().Publish(pixelforge_loop.EventLateUpdate)

		t.Run("should not execute callback again on next frame when key is still pressed", func(t *testing.T) {
			assert.Equal(t, 0, executionTimes)
		})

		shortcut.Unregister()

		pixelforge_key.Target().Publish(aDownEvent)
		pixelforge_loop.Target().Publish(pixelforge_loop.EventLateUpdate)

		t.Run("should not execute callback after shortcut is unregistered", func(t *testing.T) {
			assert.Equal(t, 0, executionTimes)
		})
	})

	t.Run("multiple keys", func(t *testing.T) {
		executionTimes := 0
		shortcut := pixelforge_key.RegisterShortcut(func() {
			executionTimes++
		}, pixelforge_key.A, pixelforge_key.Ctrl)
		require.NotNil(t, shortcut)
		// when
		pixelforge_key.Target().Publish(aDownEvent)
		pixelforge_key.Target().Publish(ctrlDownEvent)
		pixelforge_loop.Target().Publish(pixelforge_loop.EventLateUpdate)
		// then
		assert.Equal(t, 1, executionTimes)
	})

	t.Run("should not run callback when all keys are not pressed simultaneously, but was down before", func(t *testing.T) {
		executionTimes := 0
		shortcut := pixelforge_key.RegisterShortcut(func() {
			executionTimes++
		}, pixelforge_key.A, pixelforge_key.Ctrl)
		require.NotNil(t, shortcut)

		pixelforge_key.Target().Publish(aDownEvent)
		pixelforge_loop.Target().Publish(pixelforge_loop.EventLateUpdate)
		require.Equal(t, 0, executionTimes)
		// when
		pixelforge_key.Target().Publish(ctrlDownEvent)
		pixelforge_key.Target().Publish(aUpEvent) // "A" no longer down
		pixelforge_loop.Target().Publish(pixelforge_loop.EventLateUpdate)
		// then
		assert.Equal(t, 0, executionTimes)
	})
}
