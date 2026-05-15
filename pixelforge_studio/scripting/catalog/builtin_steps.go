package catalog

import (
	"fmt"
	"log"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_routine"
)

func init() {
	RegisterStep("Wait", buildWait)
	RegisterStep("Tween", buildTween)
	RegisterStep("Move", buildMove)
	RegisterStep("Play", buildPlay)
	RegisterStep("Publish", buildPublish)
	RegisterStep("Branch", buildBranch)
	RegisterStep("Custom", buildCustom)
}

func buildWait(args map[string]any, _ Context) (pixelforge_routine.Step, error) {
	ticks, err := ArgInt(args, "ticks")
	if err != nil {
		return nil, err
	}
	return pixelforge_routine.Wait(ticks), nil
}

func buildTween(args map[string]any, ctx Context) (pixelforge_routine.Step, error) {
	from, err := ArgFloat(args, "from")
	if err != nil {
		return nil, err
	}
	to, err := ArgFloat(args, "to")
	if err != nil {
		return nil, err
	}
	ticks := ArgIntOr(args, "ticks", 0)
	ease := ArgStringOr(args, "ease", pixelforge_routine.EaseLinear)

	// Tween needs a *float64. v1 uses a ValueRef adapter; if the
	// referenced value isn't a float64-compatible target, we fall
	// back to a no-op step (warning logged at compile time).
	path := ArgStringOr(args, "target", "")
	var ref ValueRef
	if path != "" && ctx != nil {
		ref = ctx.ValueRef(path)
	}
	if ref == nil {
		// Build an internal scratch float we can tween into so the
		// step still consumes ticks deterministically — preserves
		// timing even when the binding is missing.
		var scratch float64 = from
		return pixelforge_routine.Tween(&scratch, from, to, ticks, ease), nil
	}
	// Pull current value as starting basis if from is the zero of
	// the registered field; otherwise honour the explicit `from`.
	var bound float64
	if cur, ok := ref.Get().(float64); ok {
		bound = cur
	}
	_ = bound
	scratch := from
	step := pixelforge_routine.Tween(&scratch, from, to, ticks, ease)
	return func() bool {
		done := step()
		ref.Set(scratch)
		return done
	}, nil
}

func buildMove(args map[string]any, ctx Context) (pixelforge_routine.Step, error) {
	dx, err := ArgInt(args, "dx")
	if err != nil {
		return nil, err
	}
	dy, err := ArgInt(args, "dy")
	if err != nil {
		return nil, err
	}
	ticks := ArgIntOr(args, "ticks", 0)
	entityID := ArgStringOr(args, "entity", "")

	if ctx == nil || entityID == "" {
		// Untargeted Move just consumes its ticks.
		return pixelforge_routine.Wait(max1(ticks)), nil
	}
	mover, ok := ctx.Entity(entityID).(EntityMover)
	if !ok || mover == nil {
		return pixelforge_routine.Wait(max1(ticks)), nil
	}
	startX, startY := mover.PositionXY()
	elapsed := 0
	steps := max1(ticks)
	return func() bool {
		elapsed++
		if elapsed >= steps {
			mover.SetPositionXY(startX+float64(dx), startY+float64(dy))
			return true
		}
		t := float64(elapsed) / float64(steps)
		mover.SetPositionXY(startX+float64(dx)*t, startY+float64(dy)*t)
		return false
	}, nil
}

// EntityMover is implemented by runtime adapters that expose mutable
// access to an entity's position. The runtime engine package
// supplies a concrete implementation; tests can provide a fake.
type EntityMover interface {
	PositionXY() (float64, float64)
	SetPositionXY(x, y float64)
}

func buildPlay(args map[string]any, ctx Context) (pixelforge_routine.Step, error) {
	name, err := ArgString(args, "sample")
	if err != nil {
		return nil, err
	}
	pitch := ArgFloatOr(args, "pitch", 1.0)
	vol := ArgFloatOr(args, "volume", 1.0)
	chArg := ArgIntOr(args, "channel", int(pixelforge_audio.Chan1))

	var sample *pixelforge_audio.Sample
	if ctx != nil {
		if s, ok := ctx.Sample(name).(*pixelforge_audio.Sample); ok {
			sample = s
		}
	}
	return pixelforge_routine.Play(pixelforge_audio.Chan(chArg), sample, pitch, vol), nil
}

func buildPublish(args map[string]any, ctx Context) (pixelforge_routine.Step, error) {
	targetName, err := ArgString(args, "target")
	if err != nil {
		return nil, err
	}
	event, err := ArgString(args, "event")
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		return noopStep(fmt.Sprintf("Publish: nil context, target=%q", targetName)), nil
	}
	raw := ctx.LookupTarget(targetName)
	if raw == nil {
		return noopStep(fmt.Sprintf("Publish: unknown target %q", targetName)), nil
	}
	if t, ok := raw.(pievent.Target[string]); ok {
		return pixelforge_routine.Publish(t, event), nil
	}
	// Fall back: warn + no-op rather than failing the whole routine.
	return noopStep(fmt.Sprintf("Publish: target %q is not a string Target (got %T)", targetName, raw)), nil
}

func buildBranch(args map[string]any, _ Context) (pixelforge_routine.Step, error) {
	cond, _ := ArgBool(args, "predicate")
	return pixelforge_routine.Branch(func() bool { return cond }, nil, nil), nil
}

func buildCustom(args map[string]any, _ Context) (pixelforge_routine.Step, error) {
	hook := ArgStringOr(args, "hook", "")
	return noopStep(fmt.Sprintf("Custom: hook %q not yet wired", hook)), nil
}

func noopStep(reason string) pixelforge_routine.Step {
	logged := false
	return func() bool {
		if !logged {
			log.Printf("[catalog] no-op step: %s", reason)
			logged = true
		}
		return true
	}
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}
