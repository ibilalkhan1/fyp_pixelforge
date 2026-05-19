package pixelforge_project

// Plan-009 U15 — BombTimer is the canonical component type for the
// Bomberman bomb-fuse mechanic. An entity carrying a "BombTimer"
// component ticks down each AdvanceTick; when ticks_remaining hits
// zero the runtime publishes spawn/destroy_other for the entity and
// damage/grid_explode from the entity's stored grid coordinates.
//
// The component lives on Entity.Components alongside health, with
// Type=="BombTimer" and Values keyed by "ticks_remaining" + "range" +
// "grid_x" + "grid_y" (all stored as float64 because JSON unmarshalling
// normalises every number that way). The helpers in this file insulate
// the bomb-timer sink from the schema-level encoding so future authoring
// surfaces (the inspector, the prefab editor) can swap the storage
// shape without touching the sinks.
//
// Entities without a "BombTimer" component report ok=false from
// BombTimerFor; the runtime's per-tick scan skips them. Removing the
// timer (a player picked up the bomb, the bomb teleported, etc.) is
// a SetBombTimerFor with ticks_remaining=0 — the next tick fires the
// explosion. To cancel cleanly without firing, callers strip the
// component (RemoveBombTimerFor below).
const BombTimerComponentType = "BombTimer"

// BombTimer is the strongly-typed view of the schema-side
// EntityComponent{Type:"BombTimer", Values:{...}}. The runtime + sinks
// read this struct rather than touching the raw map.
type BombTimer struct {
	// TicksRemaining is the fuse countdown. The runtime decrements
	// this once per AdvanceTick; explosion fires on the tick the
	// value hits zero.
	TicksRemaining int

	// Range is the cardinal-blast reach in tiles. The grid_explode
	// handler walks up to Range tiles outward in each of the 4
	// cardinal directions from (GridX, GridY).
	Range int

	// GridX / GridY are the bomb's tile-coordinate origin. The
	// runtime uses these as the blast origin when the timer expires
	// (so a bomb that was moved or never had its position synced
	// via place_on_grid still explodes from a well-defined cell).
	GridX int
	GridY int
}

// BombTimerFor returns the BombTimer values for entity, plus an ok
// flag that is true only when the entity actually carries a
// "BombTimer" component. Missing / nil entity returns (zero, false).
//
// The lookup walks Components in order; the first entry whose Type
// matches BombTimerComponentType wins. Designers must not author two
// BombTimer components on the same entity (the schema currently has
// no uniqueness check); if they do, only the first is read.
func BombTimerFor(entity *Entity) (BombTimer, bool) {
	if entity == nil {
		return BombTimer{}, false
	}
	for i := range entity.Components {
		c := &entity.Components[i]
		if c.Type != BombTimerComponentType {
			continue
		}
		return BombTimer{
			TicksRemaining: readIntValue(c.Values, "ticks_remaining"),
			Range:          readIntValue(c.Values, "range"),
			GridX:          readIntValue(c.Values, "grid_x"),
			GridY:          readIntValue(c.Values, "grid_y"),
		}, true
	}
	return BombTimer{}, false
}

// SetBombTimerFor writes the BombTimer values into the entity's
// existing component. Returns true when the write landed; false when
// the entity has no BombTimer component (caller should treat this as
// "ignore — entity not a bomb"). The helper is the canonical update
// path the runtime calls to decrement TicksRemaining each tick.
//
// Negative TicksRemaining clamps to 0 so a runaway decrement can't
// drive the timer below zero; the next-tick fire still cascades from
// the zero value.
func SetBombTimerFor(entity *Entity, bt BombTimer) bool {
	if entity == nil {
		return false
	}
	if bt.TicksRemaining < 0 {
		bt.TicksRemaining = 0
	}
	for i := range entity.Components {
		c := &entity.Components[i]
		if c.Type != BombTimerComponentType {
			continue
		}
		if c.Values == nil {
			c.Values = map[string]any{}
		}
		c.Values["ticks_remaining"] = float64(bt.TicksRemaining)
		c.Values["range"] = float64(bt.Range)
		c.Values["grid_x"] = float64(bt.GridX)
		c.Values["grid_y"] = float64(bt.GridY)
		return true
	}
	return false
}

// RemoveBombTimerFor strips the BombTimer component from the entity's
// component list. Used by the runtime to cancel a timer cleanly (e.g.
// the bomb was picked up, defused, or teleported off-screen) so the
// next AdvanceTick doesn't fire the blast. Returns true when a
// component was removed; false when the entity carried no BombTimer.
func RemoveBombTimerFor(entity *Entity) bool {
	if entity == nil {
		return false
	}
	for i := range entity.Components {
		if entity.Components[i].Type != BombTimerComponentType {
			continue
		}
		entity.Components = append(entity.Components[:i], entity.Components[i+1:]...)
		return true
	}
	return false
}
