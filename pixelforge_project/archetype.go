package pixelforge_project

// Archetype constants name the six v1 entity character-sheet roles. An
// entity's Archetype drives the inspector's per-archetype trigger-slot
// layout (see pixelforge_studio/scripting/archetype). Unset Archetype
// is sanitized to ArchetypeWorld at load via Project.applyDefaults so
// pre-v1 .pforge files have a meaningful default.
//
// The six values are closed in v1; expansion (custom archetypes) is
// deferred. Unknown archetype values on load are preserved as-is so
// forward-compat files round-trip; the inspector falls back to the
// World template when it can't resolve a registered archetype.
const (
	ArchetypePlayer  = "Player"
	ArchetypeEnemy   = "Enemy"
	ArchetypePickup  = "Pickup"
	ArchetypeHazard  = "Hazard"
	ArchetypeNPC     = "NPC"
	ArchetypeWorld   = "World"
)

// DefaultArchetype is the sanitized fallback for any entity whose
// Archetype field is empty on load. World is the minimal template
// (when_scene_starts + every_tick only) so defaulting an old entity
// to World cannot accidentally activate behavior the designer didn't
// author.
const DefaultArchetype = ArchetypeWorld

// AllArchetypes lists the six v1 archetypes in display order. Used by
// the inspector's archetype dropdown and by tests asserting the closed
// set.
func AllArchetypes() []string {
	return []string{
		ArchetypePlayer,
		ArchetypeEnemy,
		ArchetypePickup,
		ArchetypeHazard,
		ArchetypeNPC,
		ArchetypeWorld,
	}
}
