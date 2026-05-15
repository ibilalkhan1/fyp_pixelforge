package pixelforge_routine

import pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"

// Publish creates a Routine step that publishes `event` on `target`
// once on its first tick, then advances. Typed publishes (other than
// string) go through a Custom step.
func Publish[T comparable](target pievent.Target[T], event T) Step {
	return func() bool {
		if target != nil {
			target.Publish(event)
		}
		return true
	}
}
