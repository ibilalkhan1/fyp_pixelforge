package capsuleruntime

import (
	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// Plan-009 U15 — bomb_timer.go owns the per-tick countdown logic for
// the Bomberman bomb-fuse mechanic. AdvanceTick calls advanceBombTimers
// every tick; the helper scans the current scene's entities, decrements
// each BombTimer component's ticks_remaining, and on the tick the
// timer hits zero publishes the bomb's destruction + the grid_explode
// blast.
//
// Design choice: per-tick rescan rather than a cached list. The
// CurrentScene.Entities slice is small (a Bomberman level rarely has
// more than a handful of live bombs), and the rescan keeps the bomb-
// timer system stateless on the Runtime — spawn/destroy events that
// add or remove bombs land into the canonical entity list without
// any sidecar invalidation. If a fixture ever pushes hundreds of
// simultaneous bombs we can revisit; for U15 the simplicity wins.
//
// Cancellation: a recipe that wants to defuse a bomb without firing
// the blast strips the BombTimer component (via
// pixelforge_project.RemoveBombTimerFor) or destroys the entity
// outright (spawn/destroy_other). Either path stops the timer
// because the scan skips entities that don't carry the component.

// advanceBombTimers is called once per AdvanceTick. It scans the
// current scene's entities, decrements every BombTimer it finds, and
// publishes the explosion cascade for any timer that reaches zero on
// this tick.
//
// Bomb-detonation cascade (in publish order):
//  1. spawn/destroy_other — the bomb entity is removed first so the
//     blast doesn't damage itself + so observers see the entity gone
//     before the blast events arrive.
//  2. damage/grid_explode — the blast fires from the bomb's stored
//     grid origin with its stored range; the damage sink iterates
//     cardinal rays and publishes damage/take_damage per affected
//     entity.
//
// The function collects detonations into a slice first and then
// publishes them outside the iteration loop. This keeps the scene-
// entity slice stable during decrement (publishing destroy_other
// mutates CurrentScene.Entities) and is also a safety guard against
// spawn-during-iteration ordering surprises.
func advanceBombTimers(rt *Runtime) {
	if rt == nil || rt.CurrentScene == nil {
		return
	}

	// Two-pass design:
	//   pass 1: decrement every BombTimer and collect detonations
	//   pass 2: publish each detonation through the bus
	// The collect-then-publish split keeps the scene slice stable
	// during pass 1 (the spawn sink mutates CurrentScene.Entities
	// on destroy_other), and the fire order is stable scene-order.
	type detonation struct {
		entityID string
		gridX    int
		gridY    int
		rang     int
	}
	var fires []detonation

	ents := rt.CurrentScene.Entities
	for i := range ents {
		e := &ents[i]
		bt, ok := pixelforge_project.BombTimerFor(e)
		if !ok {
			continue
		}
		// Decrement first, then check for fire. A timer authored
		// with ticks_remaining=1 fires on the next AdvanceTick (it
		// decrements 1 → 0, the equality check fires). A timer
		// authored with ticks_remaining=0 fires on the FIRST
		// AdvanceTick — the recipe author wanted an "instant" bomb.
		bt.TicksRemaining--
		if bt.TicksRemaining < 0 {
			bt.TicksRemaining = 0
		}
		pixelforge_project.SetBombTimerFor(e, bt)
		if bt.TicksRemaining == 0 {
			fires = append(fires, detonation{
				entityID: e.ID,
				gridX:    bt.GridX,
				gridY:    bt.GridY,
				rang:     bt.Range,
			})
		}
	}

	// Pass 2: publish each detonation. destroy_other first (so the
	// bomb entity is gone before the blast iterates), then
	// grid_explode (which the damage sink consumes to fan out the
	// per-tile damage events).
	for _, fire := range fires {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "spawn/destroy_other",
			Args:  map[string]any{"entity": fire.entityID},
		})
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "damage/grid_explode",
			Args: map[string]any{
				"entity_id": fire.entityID,
				"origin":    map[string]any{"gx": fire.gridX, "gy": fire.gridY},
				"range":     float64(fire.rang),
				"damage":    float64(1),
			},
		})
	}
}
