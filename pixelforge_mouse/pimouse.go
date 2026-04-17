package pixelforge_mouse

import (
	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/internal/input"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
)

var (
	Position      pixelforge.Position
	MovementDelta pixelforge.Position // mouse movement delta since the last frame
)

func Duration(b Button) int {
	return buttonState.Duration(b)
}

type Button string

const (
	Left  Button = "Left"
	Right Button = "Right"
)

var buttonTarget = pievent.NewTarget[EventButton]()
var buttonDebugTarget = pievent.NewTarget[EventButton]()
var moveTarget = pievent.NewTarget[EventMove]()
var moveDebugTarget = pievent.NewTarget[EventMove]()

var buttonState input.State[Button]

func init() {
	onButton := func(event EventButton, _ pievent.Handler) {
		switch event.Type {
		case EventButtonDown:
			buttonState.SetDownFrame(event.Button, pixelforge.Frame)
		case EventButtonUp:
			buttonState.SetUpFrame(event.Button, pixelforge.Frame)
		}
	}
	buttonTarget.SubscribeAll(onButton)
	buttonDebugTarget.SubscribeAll(onButton)

	onMove := func(event EventMove, _ pievent.Handler) {
		Position = event.Position
		MovementDelta = event.Position.Subtract(event.Previous)
	}
	moveTarget.SubscribeAll(onMove)
	moveDebugTarget.SubscribeAll(onMove)
}
