package capsuleruntime

import (
	"fmt"
	"sync/atomic"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// spawnSink is the concrete SpawnSink plan-009 U7 lands. It looks up
// a prefab by name in rt.Project.Prefabs, clones its component list,
// and appends a new entity to the current scene at the requested
// position. DestroySelf / DestroyOther remove the named entity from
// the current scene by ID.
//
// Spawn-id uniqueness: each spawn generates a synthetic ID of the
// form "<prefab>_<seq>" where seq is a per-runtime monotonic counter
// — designers can target the spawned entity by ID for follow-up
// verbs (visual/flash, damage/take_damage). The counter survives the
// runtime's lifetime so two spawns from the same prefab never
// collide.
type spawnSink struct {
	rt *Runtime
	// spawnSeq is the monotonic counter that disambiguates
	// synthetic spawn IDs. uint64 so a 60-TPS game spawning one
	// entity per frame would take ~10 thousand years to wrap.
	spawnSeq uint64
}

// newSpawnSink constructs the production spawnSink.
func newSpawnSink(rt *Runtime) *spawnSink {
	return &spawnSink{rt: rt}
}

// nextSpawnID returns the next synthetic entity ID for prefab. The
// per-sink counter is atomic so concurrent spawns from different
// goroutines (e.g. CI replay threads) don't produce dupes.
func (s *spawnSink) nextSpawnID(prefab string) string {
	n := atomic.AddUint64(&s.spawnSeq, 1)
	return fmt.Sprintf("%s_%d", prefab, n)
}

// Spawn materialises one new entity in rt.CurrentScene cloned from
// the named prefab's component list. Unknown prefabs log + no-op
// (graceful degrade — a recipe targeting a removed prefab must not
// crash a shipped game).
//
// The cloned components carry independent maps so two entities
// spawned from the same prefab can mutate state independently
// (component values are otherwise shared map references).
func (s *spawnSink) Spawn(args map[string]any) {
	if s == nil || s.rt == nil || s.rt.CurrentScene == nil {
		return
	}
	name, ok := argString(args, "prefab")
	if !ok {
		name, ok = argString(args, "name")
	}
	if !ok || name == "" {
		debugDrop("spawn/entity", "missing prefab", args)
		return
	}
	var template *pixelforge_project.Prefab
	for i := range s.rt.Project.Prefabs {
		if s.rt.Project.Prefabs[i].Name == name {
			template = &s.rt.Project.Prefabs[i]
			break
		}
	}
	if template == nil {
		debugDrop("spawn/entity", "unknown prefab", args)
		return
	}
	x := argFloatOr(args, "x", 0)
	y := argFloatOr(args, "y", 0)
	id := s.nextSpawnID(name)
	clones := cloneComponents(template.Components)
	entity := pixelforge_project.Entity{
		ID:         id,
		Name:       name,
		Position:   pixelforge_project.EntityPosition{X: x, Y: y},
		Components: clones,
	}
	s.rt.CurrentScene.Entities = append(s.rt.CurrentScene.Entities, entity)
}

// DestroySelf removes the entity named by args["entity"] (or
// args["entity_id"]) from the current scene. Synonym to
// DestroyOther in this implementation — the recipe-level
// distinction between "self" and "other" matters for the trigger
// surface (who fired the recipe), not for the destroy mechanic.
func (s *spawnSink) DestroySelf(args map[string]any) {
	s.destroy(args, "spawn/destroy_self")
}

// DestroyOther removes the named entity from the current scene.
// See DestroySelf for the recipe-level distinction.
func (s *spawnSink) DestroyOther(args map[string]any) {
	s.destroy(args, "spawn/destroy_other")
}

// destroy is the shared remove-from-scene helper. Removes the first
// entity in CurrentScene.Entities matching the args ID. No-op when
// the ID is missing or unknown.
func (s *spawnSink) destroy(args map[string]any, topic string) {
	if s == nil || s.rt == nil || s.rt.CurrentScene == nil {
		return
	}
	id, ok := argString(args, "entity_id")
	if !ok {
		id, ok = argString(args, "entity")
	}
	if !ok || id == "" {
		debugDrop(topic, "missing entity", args)
		return
	}
	ents := s.rt.CurrentScene.Entities
	for i := range ents {
		if ents[i].ID == id {
			s.rt.CurrentScene.Entities = append(ents[:i], ents[i+1:]...)
			delete(s.rt.VisualEffects, id)
			return
		}
	}
}

// cloneComponents deep-copies a prefab's component list so the
// spawned entity owns mutable Values independent of the prefab
// template and of any other entity spawned from it.
func cloneComponents(src []pixelforge_project.EntityComponent) []pixelforge_project.EntityComponent {
	if len(src) == 0 {
		return []pixelforge_project.EntityComponent{}
	}
	out := make([]pixelforge_project.EntityComponent, len(src))
	for i, c := range src {
		var vals map[string]any
		if c.Values != nil {
			vals = make(map[string]any, len(c.Values))
			for k, v := range c.Values {
				vals[k] = v
			}
		}
		out[i] = pixelforge_project.EntityComponent{Type: c.Type, Values: vals}
	}
	return out
}
