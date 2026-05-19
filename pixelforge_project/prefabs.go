// prefabs.go owns plan-009 U7's schema addition: Prefabs — a flat,
// name-keyed registry of entity templates the spawn/entity verb-
// recipe instantiates at runtime. Additive omitempty so pre-v1
// projects round-trip cleanly; applyDefaults backfills a nil slice
// to an empty one so spawnSink never has to nil-check.
//
// A Prefab is a minimal entity template — a Name + a component
// list — because that's all the spawn sink needs to materialise a
// new Entity at (x, y) inside the current scene. Future versions
// can add per-prefab tile-cell defaults, archetype hints, or
// designer-set verb bindings; the schema reservation is the leverage
// move so adding those is a pure additive change.
package pixelforge_project

// Prefab is one named entity template. The spawn/entity sink looks
// up the prefab by Name, then constructs a new Entity in the
// current scene cloning the prefab's Components.
type Prefab struct {
	// Name is the project-scoped identifier the spawn/entity
	// recipe references. Unique within Project.Prefabs.
	Name string `json:"name"`

	// Components is the flat list cloned into each newly-spawned
	// entity. The runtime deep-copies each component's Values map
	// so two spawned-from-the-same-prefab entities have independent
	// mutable state.
	Components []EntityComponent `json:"components,omitempty"`
}
