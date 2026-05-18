// Package pixelforge_blackboard provides a single, process-wide
// key/value store for shared mutable gameplay state (score, lives,
// health, current scene, player position, etc.).
//
// The blackboard is the canonical home for "global" runtime values
// that multiple entities, verbs, and inspector panels need to read or
// write. Per the v1 entity-verb-sheet milestone, it replaces the
// pattern of each entity inventing its own per-instance state and
// gives the studio's blackboard inspector workspace (U9) a stable
// surface to enumerate.
//
// # Design
//
// A single *Blackboard value is created at init time and registered
// against the pievent registry under the well-known name "state/*".
// Studio surfaces (the blackboard inspector) and runtime code locate
// it via pievent.LookupTarget("state/*") and type-assert back to
// *Blackboard for read/write access.
//
// Internally the map[string]any payload is guarded by a sync.RWMutex
// so concurrent Get/Set calls from goroutines (e.g. routine-based
// verbs) are race-free. Every Set call publishes a BlackboardChange
// event on an internal pievent.Target so inspector panels can observe
// mutations live without polling.
//
// # Canonical keys
//
// canonical_keys.go declares a CanonicalKeys map listing well-known
// keys (score, lives, health, current_scene, player_x, player_y)
// together with their declared CanonicalKeyType and default value.
// The blackboard itself does not enforce these types — designer-added
// keys remain free-form — but the studio inspector and verb-binding
// compiler use the table to pick typed widgets and to validate
// recipe argument shapes.
//
// # What this package does NOT do
//
// It is the primitive only. There is no studio panel here, no save
// serialization, no per-scene scoping. Those concern higher layers
// (see plan units U9 and the idea #6 save UI brainstorm).
package pixelforge_blackboard
