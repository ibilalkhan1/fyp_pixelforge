package pixelforge_pad

import "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"

// EventButton is published when the player presses or releases a gamepad button.
//
// It may be published more than once during a single game tick.
type EventButton struct {
	Type   EventButtonType
	Button Button
	Player int
}

type EventButtonType string

const (
	EventUp   EventButtonType = "up"
	EventDown EventButtonType = "down"
)

type EventConnection struct {
	Type   EventConnectionType
	Player int
}

type EventConnectionType string

const (
	EventConnect    EventConnectionType = "connect"
	EventDisconnect EventConnectionType = "disconnect"
)

var buttonTarget = pievent.NewTarget[EventButton]()

func ButtonTarget() pievent.Target[EventButton] {
	return buttonTarget
}

var target = pievent.NewTarget[EventConnection]()

func ConnectionTarget() pievent.Target[EventConnection] {
	return target
}
