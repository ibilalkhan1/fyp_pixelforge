package capsuleruntime

// visualSink is the concrete VisualSink plan-009 U7 lands. It keeps
// a per-entity render-override table on the Runtime (rt.VisualEffects)
// that the renderer consults each frame to decide whether to hide,
// recolor, or sprite-swap an entity. Flash uses a tick-based
// countdown — engine drivers call rt.AdvanceTick() once per game
// tick, which expires the flash when rt.Tick reaches the recorded
// ExpireTick.
//
// All methods are no-ops when the entity arg is missing/empty;
// recipes that fire with no entity target match the broader
// "never crash a shipped game" posture by silently dropping.
type visualSink struct {
	rt *Runtime
}

// newVisualSink constructs the production visualSink.
func newVisualSink(rt *Runtime) *visualSink {
	return &visualSink{rt: rt}
}

// effectFor returns the per-entity effect record, creating it
// lazily. Caller must hold the entity ID (caller-checked for empty).
func (v *visualSink) effectFor(id string) *VisualEffect {
	if v.rt.VisualEffects == nil {
		v.rt.VisualEffects = map[string]*VisualEffect{}
	}
	eff, ok := v.rt.VisualEffects[id]
	if !ok {
		eff = &VisualEffect{}
		v.rt.VisualEffects[id] = eff
	}
	return eff
}

// Hide flips the per-entity Hidden bit. The renderer skips entities
// whose effect record has Hidden=true; absent entries render
// normally.
func (v *visualSink) Hide(args map[string]any) {
	if v == nil || v.rt == nil {
		return
	}
	id, ok := argString(args, "entity_id")
	if !ok {
		id, ok = argString(args, "entity")
	}
	if !ok || id == "" {
		debugDrop("visual/hide", "missing entity", args)
		return
	}
	v.effectFor(id).Hidden = true
}

// Show clears the per-entity Hidden bit. If the entity had no
// other active overrides, the effect record is removed entirely so
// the map doesn't accumulate empty entries.
func (v *visualSink) Show(args map[string]any) {
	if v == nil || v.rt == nil {
		return
	}
	id, ok := argString(args, "entity_id")
	if !ok {
		id, ok = argString(args, "entity")
	}
	if !ok || id == "" {
		debugDrop("visual/show", "missing entity", args)
		return
	}
	if v.rt.VisualEffects == nil {
		return
	}
	eff, ok := v.rt.VisualEffects[id]
	if !ok {
		return
	}
	eff.Hidden = false
	if eff.FlashColor == "" && eff.SpriteSwap == "" {
		delete(v.rt.VisualEffects, id)
	}
}

// Flash queues a per-entity palette override that auto-expires
// after ms milliseconds. Conversion to a tick countdown uses
// Project.TPS so the same recipe fires identically across 30-TPS
// and 60-TPS projects. A flash of zero/negative duration is a no-op
// (the recipe was malformed; never block on a 0-tick countdown).
func (v *visualSink) Flash(args map[string]any) {
	if v == nil || v.rt == nil {
		return
	}
	id, ok := argString(args, "entity_id")
	if !ok {
		id, ok = argString(args, "entity")
	}
	if !ok || id == "" {
		debugDrop("visual/flash", "missing entity", args)
		return
	}
	ms := int(argFloatOr(args, "ms", 0))
	if ms <= 0 {
		ms = int(argFloatOr(args, "duration_ms", 0))
	}
	if ms <= 0 {
		debugDrop("visual/flash", "non-positive duration", args)
		return
	}
	color, _ := argString(args, "color")
	if color == "" {
		color = "#ffffff"
	}
	tps := 60
	if v.rt.Project != nil && v.rt.Project.TPS > 0 {
		tps = v.rt.Project.TPS
	}
	// Round up so a 100ms flash at 60 TPS gets 6 ticks (100/1000*60=6.0).
	// Use ceiling so very short ms windows still produce at least one tick.
	ticks := (ms*tps + 999) / 1000
	if ticks < 1 {
		ticks = 1
	}
	eff := v.effectFor(id)
	eff.FlashColor = color
	eff.ExpireTick = v.rt.Tick + uint64(ticks)
}

// SwapSprite installs a sprite-swap override the renderer reads in
// place of the entity's authored sprite. Empty name clears the swap.
func (v *visualSink) SwapSprite(args map[string]any) {
	if v == nil || v.rt == nil {
		return
	}
	id, ok := argString(args, "entity_id")
	if !ok {
		id, ok = argString(args, "entity")
	}
	if !ok || id == "" {
		debugDrop("visual/swap_sprite", "missing entity", args)
		return
	}
	name, _ := argString(args, "name")
	if name == "" {
		name, _ = argString(args, "sprite")
	}
	eff := v.effectFor(id)
	eff.SpriteSwap = name
	if name == "" && !eff.Hidden && eff.FlashColor == "" {
		delete(v.rt.VisualEffects, id)
	}
}
