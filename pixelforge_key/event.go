package pixelforge_key

import pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"

// Event is published when the player presses or releases a key.
//
// It may be published more than once during a single game tick.
type Event struct {
	Type EventType
	Key  Key
}

type EventType string

const (
	EventUp   EventType = "up"
	EventDown EventType = "down"
)

func Target() pievent.Target[Event] {
	return target
}

// events are published all the time - even when game is paused.
func DebugTarget() pievent.Target[Event] {
	return debugTarget
}
