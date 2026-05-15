package widgets_test

import (
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui/widgets"
	"github.com/stretchr/testify/assert"
)

func TestStepCard_DefaultDimensions(t *testing.T) {
	card := widgets.NewStepCard(0, 0, 0, 0, widgets.StepCardOptions{
		Kind:  "Wait",
		Label: "30 ticks",
	})
	assert.Equal(t, 64, card.W)
	assert.Equal(t, 56, card.H)
	assert.Equal(t, "Wait", card.Kind)
	assert.Equal(t, "30 ticks", card.Label)
}

func TestStepCard_ClickFiresOnSelect(t *testing.T) {
	clicked := 0
	card := widgets.NewStepCard(0, 0, 64, 56, widgets.StepCardOptions{
		Kind:     "Wait",
		OnSelect: func() { clicked++ },
	})
	assert.True(t, card.Press(10, 10))
	wasClick := card.Release()
	assert.True(t, wasClick)
	assert.Equal(t, 1, clicked)
}

func TestStepCard_DragFiresOnDragMove(t *testing.T) {
	deltas := []struct{ dx, dy int }{}
	card := widgets.NewStepCard(0, 0, 64, 56, widgets.StepCardOptions{
		Kind:       "Move",
		OnDragMove: func(dx, dy int) { deltas = append(deltas, struct{ dx, dy int }{dx, dy}) },
	})
	assert.True(t, card.Press(10, 10))
	card.Move(20, 12)
	card.Move(30, 14)
	assert.Equal(t, 20, card.CumulativeDX())
	assert.Equal(t, 4, card.CumulativeDY())
	assert.False(t, card.Release(), "drag with movement should not register as click")
	assert.Len(t, deltas, 2)
}

func TestStepCard_PressOutsideIgnored(t *testing.T) {
	card := widgets.NewStepCard(100, 100, 64, 56, widgets.StepCardOptions{})
	assert.False(t, card.Press(0, 0))
}

func TestStepCard_DragThenReleaseToZeroStillNotClick(t *testing.T) {
	clicked := 0
	card := widgets.NewStepCard(0, 0, 64, 56, widgets.StepCardOptions{
		OnSelect: func() { clicked++ },
	})
	card.Press(10, 10)
	card.Move(20, 10) // dx = +10
	card.Move(10, 10) // dx = -10 (net 0)
	assert.False(t, card.Release(), "movement-then-undo should still be a drag, not a click")
	assert.Equal(t, 0, clicked)
}

func TestStepCard_ActiveAndSelectedFlags(t *testing.T) {
	card := widgets.NewStepCard(0, 0, 64, 56, widgets.StepCardOptions{
		Kind:     "Wait",
		IsActive: true,
		Selected: true,
	})
	assert.True(t, card.IsActive)
	assert.True(t, card.Selected)
}
