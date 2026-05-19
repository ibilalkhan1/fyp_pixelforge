package catalog

import (
	"log"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	pievent "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_event"
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
)

func init() {
	RegisterAction("play_sample", buildPlaySample)
	RegisterAction("set_value", buildSetValue)
	RegisterAction("publish_event", buildPublishEvent)
	RegisterAction("move_entity", buildMoveEntity)
	RegisterAction("branch", buildBranchAction)
}

func buildPlaySample(args map[string]any, ctx Context) (Effect, error) {
	name, err := ArgString(args, "name")
	if err != nil {
		return nil, err
	}
	pitch := ArgFloatOr(args, "pitch", 1.0)
	vol := ArgFloatOr(args, "volume", 1.0)
	chArg := ArgIntOr(args, "channel", int(pixelforge_audio.Chan1))
	return func(_ any) {
		if ctx == nil {
			return
		}
		sample, ok := ctx.Sample(name).(*pixelforge_audio.Sample)
		if !ok || sample == nil {
			return
		}
		pixelforge_audio.Play(pixelforge_audio.Chan(chArg), sample, pitch, vol)
	}, nil
}

func buildSetValue(args map[string]any, ctx Context) (Effect, error) {
	path, err := ArgString(args, "target")
	if err != nil {
		return nil, err
	}
	value, ok := args["value"]
	if !ok {
		return func(_ any) {}, nil
	}
	return func(_ any) {
		if ctx == nil {
			return
		}
		if ref := ctx.ValueRef(path); ref != nil {
			ref.Set(value)
		}
	}, nil
}

func buildPublishEvent(args map[string]any, ctx Context) (Effect, error) {
	targetName, err := ArgString(args, "target")
	if err != nil {
		return nil, err
	}
	event, err := ArgString(args, "event")
	if err != nil {
		return nil, err
	}
	// Capture every arg except target/event as the payload. The
	// recipe's authored arg map (sample name, item id, fuse ticks,
	// etc.) survives to the subscriber so payload-bearing verbs
	// don't need a blackboard round-trip. Shallow copy keeps the
	// builder safe to call multiple times against the same args
	// without mutating the catalog's defaults.
	payload := make(map[string]any, len(args))
	for k, v := range args {
		if k == "target" || k == "event" {
			continue
		}
		payload[k] = v
	}
	return func(_ any) {
		if ctx == nil {
			return
		}
		raw := ctx.LookupTarget(targetName)
		if raw == nil {
			return
		}
		if t, ok := raw.(pievent.Target[*piloop.VerbEvent]); ok {
			t.Publish(&piloop.VerbEvent{Topic: event, Args: payload})
			return
		}
		// Back-compat: the legacy "loop.main" Target[string] path
		// stays addressable so a recipe explicitly targeting an
		// existing string-typed target still publishes its event
		// name. Verb-recipe subscribers live exclusively on
		// "verbs.bus" (Target[*VerbEvent]) so this fallback never
		// fires for the catalog's own builtins.
		if t, ok := raw.(pievent.Target[string]); ok {
			t.Publish(event)
			return
		}
		log.Printf("[catalog] publish_event: target %q is neither a VerbEvent Target nor a string Target", targetName)
	}, nil
}

func buildMoveEntity(args map[string]any, ctx Context) (Effect, error) {
	entityID, err := ArgString(args, "entity")
	if err != nil {
		return nil, err
	}
	dx, err := ArgFloat(args, "dx")
	if err != nil {
		return nil, err
	}
	dy, err := ArgFloat(args, "dy")
	if err != nil {
		return nil, err
	}
	return func(_ any) {
		if ctx == nil {
			return
		}
		mover, ok := ctx.Entity(entityID).(EntityMover)
		if !ok || mover == nil {
			return
		}
		x, y := mover.PositionXY()
		mover.SetPositionXY(x+dx, y+dy)
	}, nil
}

// buildBranchAction is the action-form Branch — when the predicate
// is true it runs `then` actions, otherwise `else_`. Because actions
// can't carry nested actions through the v1 schema, the action-form
// branch in this catalog just toggles a passthrough effect; the
// recursive-rule form is the real branching primitive.
func buildBranchAction(args map[string]any, _ Context) (Effect, error) {
	_, _ = ArgBool(args, "predicate")
	return func(_ any) {}, nil
}
