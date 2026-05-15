package pixelforge_routine

import (
	"log"
	"sync"
)

// Easing names accepted by Tween. Unknown names fall back to linear
// with a one-time warning.
const (
	EaseLinear     = "linear"
	EaseIn         = "ease-in"
	EaseOut        = "ease-out"
	EaseInOut      = "ease-in-out"
)

var (
	easingWarnOnce sync.Map // map[string]struct{} — fired at most once per unknown name
)

// Tween creates a Routine step that interpolates *target from `from`
// to `to` over `ticks` ticks using the named easing.
//
// `ticks == 0` jumps to `to` immediately. `from == to` is a no-op.
// Unknown easing names fall back to linear with a one-time log warning.
func Tween(target *float64, from, to float64, ticks int, ease string) Step {
	if target == nil {
		return func() bool { return true }
	}
	if ticks <= 0 || from == to {
		return func() bool {
			*target = to
			return true
		}
	}
	easeFn := easingFunc(ease)
	elapsed := 0
	return func() bool {
		elapsed++
		if elapsed >= ticks {
			*target = to
			return true
		}
		t := float64(elapsed) / float64(ticks)
		*target = from + (to-from)*easeFn(t)
		return false
	}
}

func easingFunc(name string) func(float64) float64 {
	switch name {
	case "", EaseLinear:
		return easeLinear
	case EaseIn:
		return easeIn
	case EaseOut:
		return easeOut
	case EaseInOut:
		return easeInOut
	default:
		if _, loaded := easingWarnOnce.LoadOrStore(name, struct{}{}); !loaded {
			log.Printf("[piroutine] unknown easing %q; falling back to linear", name)
		}
		return easeLinear
	}
}

func easeLinear(t float64) float64 { return t }
func easeIn(t float64) float64     { return t * t }
func easeOut(t float64) float64    { return 1 - (1-t)*(1-t) }
func easeInOut(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	x := -2*t + 2
	return 1 - x*x/2
}
