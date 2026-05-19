package pixelforge_project

// Plan-009 U9 — Health is the canonical component type for entity hit
// points. Damage-system verbs (damage/take_damage, damage/die) read
// the entity's HP via HealthFor and mutate it via SetHealthFor.
//
// Health lives on Entity.Components as a regular EntityComponent with
// Type=="health" and Values keyed by "hp" + "max_hp" (both stored as
// float64 because JSON unmarshalling normalises every number that
// way). The helpers in this file insulate the damage sink from the
// schema-level encoding so future authoring surfaces (the inspector,
// the prefab editor) can swap the storage shape without touching the
// sinks.
//
// Entities without a "health" component report ok=false from
// HealthFor; the damage sink treats this as "no HP tracked" and
// gracefully no-ops on take_damage rather than crashing — recipes
// targeting an unbreakable wall or a marker entity must remain safe.
const HealthComponentType = "health"

// HealthFor returns the current and max HP for entity, plus an ok
// flag that is true only when the entity actually carries a "health"
// component. Missing / nil entity returns (0, 0, false).
//
// The lookup walks Components in order; the first entry whose Type
// matches HealthComponentType wins. Designers must not author two
// health components on the same entity (the schema currently has no
// uniqueness check); if they do, only the first is read.
func HealthFor(entity *Entity) (hp, maxHP int, ok bool) {
	if entity == nil {
		return 0, 0, false
	}
	for i := range entity.Components {
		c := &entity.Components[i]
		if c.Type != HealthComponentType {
			continue
		}
		hp = readIntValue(c.Values, "hp")
		maxHP = readIntValue(c.Values, "max_hp")
		return hp, maxHP, true
	}
	return 0, 0, false
}

// SetHealthFor writes hp into the entity's health component. Negative
// values clamp to 0 so a damage verb that overshoots can't drive hp
// below zero (the sink reports "dead" via a downstream damage/die
// rather than via a negative number). Returns true when the write
// landed; false when the entity has no health component (caller
// should treat this as "ignore — entity not damageable").
//
// max_hp is preserved as-is — the helper never invents a default
// max_hp because the .pforge schema's authored value is canonical.
func SetHealthFor(entity *Entity, hp int) bool {
	if entity == nil {
		return false
	}
	if hp < 0 {
		hp = 0
	}
	for i := range entity.Components {
		c := &entity.Components[i]
		if c.Type != HealthComponentType {
			continue
		}
		if c.Values == nil {
			c.Values = map[string]any{}
		}
		c.Values["hp"] = float64(hp)
		return true
	}
	return false
}

// readIntValue extracts an int from a Values map. JSON unmarshalling
// always lands numbers as float64; programmatic tests sometimes pass
// int directly. Other shapes default to 0 — matching the sink's
// "missing-arg means zero" posture so a malformed component doesn't
// crash a shipped game.
func readIntValue(values map[string]any, key string) int {
	if values == nil {
		return 0
	}
	raw, ok := values[key]
	if !ok {
		return 0
	}
	switch v := raw.(type) {
	case float64:
		return int(v)
	case float32:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case int32:
		return int(v)
	}
	return 0
}
