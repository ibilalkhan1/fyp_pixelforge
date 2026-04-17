package pixelforge_debug

import pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"

type Event string

const (
	EventPause  Event = "pause"
	EventResume Event = "resume"
)

func Target() pievent.Target[Event] {
	return target
}

var target = pievent.NewTarget[Event]()
