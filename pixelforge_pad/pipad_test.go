package pixelforge_pad_test

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_pad"
	"github.com/stretchr/testify/assert"
)

var (
	player0connected    = pixelforge_pad.EventConnection{Type: pixelforge_pad.EventConnect, Player: 0}
	player1connected    = pixelforge_pad.EventConnection{Type: pixelforge_pad.EventConnect, Player: 1}
	player0disconnected = pixelforge_pad.EventConnection{Type: pixelforge_pad.EventDisconnect, Player: 0}
	player1disconnected = pixelforge_pad.EventConnection{Type: pixelforge_pad.EventDisconnect, Player: 1}
)

func TestPlayerCount(t *testing.T) {
	assert.Equal(t, 0, pixelforge_pad.PlayerCount())
	pixelforge_pad.ConnectionTarget().Publish(player0connected)
	pixelforge_pad.ConnectionTarget().Publish(player1connected)
	assert.Equal(t, 2, pixelforge_pad.PlayerCount())
	pixelforge_pad.ConnectionTarget().Publish(player0disconnected)
	assert.Equal(t, 1, pixelforge_pad.PlayerCount())
	pixelforge_pad.ConnectionTarget().Publish(player1disconnected)
}

func TestDuration(t *testing.T) {
	{
		pixelforge_pad.ConnectionTarget().Publish(player0connected)
		pixelforge_pad.ButtonTarget().Publish(
			pixelforge_pad.EventButton{Type: pixelforge_pad.EventDown, Button: pixelforge_pad.A, Player: 0},
		)

		t.Run("should return duration when button was pressed", func(t *testing.T) {
			duration := pixelforge_pad.Duration(pixelforge_pad.A)
			assert.Equal(t, 1, duration)

			playerDuration := pixelforge_pad.PlayerDuration(pixelforge_pad.A, 0)
			assert.Equal(t, 1, playerDuration)
		})

		pixelforge.Frame++

		t.Run("should take into account how many frames passed", func(t *testing.T) {
			assert.Equal(t, 2, pixelforge_pad.Duration(pixelforge_pad.A))
			assert.Equal(t, 2, pixelforge_pad.PlayerDuration(pixelforge_pad.A, 0))
		})

		pixelforge_pad.ConnectionTarget().Publish(player0disconnected)
	}

	t.Run("should return 0 after controller was disconnected", func(t *testing.T) {
		assert.Equal(t, 0, pixelforge_pad.Duration(pixelforge_pad.A))
		assert.Equal(t, 0, pixelforge_pad.PlayerDuration(pixelforge_pad.A, 0))
	})

	t.Run("should return the longest duration when two controllers are pressed simultaneously", func(t *testing.T) {
		pixelforge_pad.ConnectionTarget().Publish(player0connected)
		defer pixelforge_pad.ConnectionTarget().Publish(player0disconnected)
		pixelforge_pad.ConnectionTarget().Publish(player1connected)
		defer pixelforge_pad.ConnectionTarget().Publish(player1disconnected)

		pixelforge_pad.ButtonTarget().Publish(
			pixelforge_pad.EventButton{Type: pixelforge_pad.EventDown, Button: pixelforge_pad.A, Player: 0},
		)
		pixelforge.Frame++
		pixelforge_pad.ButtonTarget().Publish(
			pixelforge_pad.EventButton{Type: pixelforge_pad.EventUp, Button: pixelforge_pad.A, Player: 1},
		)
		assert.Equal(t, 2, pixelforge_pad.Duration(pixelforge_pad.A))
		assert.Equal(t, 2, pixelforge_pad.PlayerDuration(pixelforge_pad.A, 0))
	})

	t.Run("should return 0 when player was never connected", func(t *testing.T) {
		assert.Equal(t, 0, pixelforge_pad.PlayerDuration(pixelforge_pad.A, 2))
	})
}
