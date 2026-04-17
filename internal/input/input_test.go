package input_test

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/internal/input"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestState_Duration(t *testing.T) {
	const btn = "btn"

	t.Run("should return 0 when input was never pressed", func(t *testing.T) {
		var i input.State[string]
		assert.Equal(t, 0, i.Duration(btn))
	})

	t.Run("should return 1 when input was pressed and released this frame", func(t *testing.T) {
		var i input.State[string]
		pixelforge.Frame = 1
		i.SetDownFrame(btn, pixelforge.Frame)
		i.SetUpFrame(btn, pixelforge.Frame)
		assert.Equal(t, 1, i.Duration(btn))
	})

	t.Run("should return 0 when input was pressed previous frame and released this frame", func(t *testing.T) {
		var i input.State[string]
		pixelforge.Frame = 0
		i.SetDownFrame(btn, pixelforge.Frame)
		pixelforge.Frame++
		i.SetUpFrame(btn, pixelforge.Frame)
		assert.Equal(t, 0, i.Duration(btn))
	})

	t.Run("should return 2 when input was pressed previous frame but not released this frame", func(t *testing.T) {
		var i input.State[string]
		pixelforge.Frame = 0
		i.SetDownFrame(btn, pixelforge.Frame)
		pixelforge.Frame++
		assert.Equal(t, 2, i.Duration(btn))
	})

	t.Run("should return 1 when input was pressed, released and pressed again this frame", func(t *testing.T) {
		var i input.State[string]
		pixelforge.Frame = 0
		i.SetDownFrame(btn, pixelforge.Frame)
		i.SetUpFrame(btn, pixelforge.Frame)
		i.SetDownFrame(btn, pixelforge.Frame)
		assert.Equal(t, 1, i.Duration(btn))
	})

	t.Run("should return 1 when input was pressed and released previous frame and pressed this frame", func(t *testing.T) {
		var i input.State[string]
		pixelforge.Frame = 0
		i.SetDownFrame(btn, pixelforge.Frame)
		i.SetUpFrame(btn, pixelforge.Frame)
		pixelforge.Frame++
		i.SetDownFrame(btn, pixelforge.Frame)
		assert.Equal(t, 1, i.Duration(btn))
	})
}
