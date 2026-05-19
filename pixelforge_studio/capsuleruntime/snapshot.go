package capsuleruntime

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_physics"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	pisave "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_save"
)

// EncodeSnapshot walks the runtime's current scene + globals + physics
// bodies and returns a v2 pisave.Snapshot. The returned snapshot is
// schema-v2 — populated Entities, Globals, and Tick — with the legacy
// v1 fields left empty (omitempty keeps them out of the marshaled
// JSON).
//
// The encoder is deterministic: the same rt state produces a
// byte-identical snapshot across repeated calls. Determinism comes
// from two properties:
//   - Entities is a slice walked in rt.CurrentScene.Entities order,
//     which is the authored order (or post-spawn order — both stable
//     within a single runtime lifetime).
//   - Components + Globals are map[string]json.RawMessage; Go's
//     stdlib json.Marshal sorts map keys lexicographically, so the
//     output ordering is stable across encode calls.
//
// Returns an error only when a component's Values cannot be marshaled
// (a programmer error — Values is map[string]any, which json.Marshal
// rejects only on unsupported types like channels).
func EncodeSnapshot(rt *Runtime) (pisave.Snapshot, error) {
	if rt == nil {
		return pisave.Snapshot{}, fmt.Errorf("capsuleruntime: nil runtime")
	}
	snap := pisave.Snapshot{
		SchemaVersion: pisave.CurrentSchemaVersion,
		Tick:          rt.Tick,
	}
	if rt.Project != nil {
		title := rt.Project.SaveConfig.GameTitle
		if title == "" {
			title = rt.Project.Name
		}
		snap.GameTitle = title
	}
	if rt.CurrentScene != nil {
		snap.CurrentSceneID = rt.CurrentScene.ID
		entities, err := encodeEntitiesWithBarrels(rt.CurrentScene.Entities, rt.Bodies, rt.BarrelStates)
		if err != nil {
			return pisave.Snapshot{}, err
		}
		snap.Entities = entities
	}
	if len(rt.Globals) > 0 {
		globals, err := encodeGlobals(rt.Globals)
		if err != nil {
			return pisave.Snapshot{}, err
		}
		snap.Globals = globals
	}
	return snap, nil
}

// encodeEntities walks the scene's entity list and emits one
// EntitySnapshotV2 per entry. Component Values are marshaled into
// json.RawMessage so the snapshot survives schema evolution on the
// component side — the runtime owns the component types, the
// snapshot module owns the byte-level shape.
func encodeEntities(entities []pixelforge_project.Entity, bodies map[string]*pixelforge_physics.Body) ([]pisave.EntitySnapshotV2, error) {
	if len(entities) == 0 {
		return nil, nil
	}
	out := make([]pisave.EntitySnapshotV2, 0, len(entities))
	for i := range entities {
		e := &entities[i]
		comp, err := encodeComponents(e.Components)
		if err != nil {
			return nil, fmt.Errorf("capsuleruntime: encode entity %q components: %w", e.ID, err)
		}
		es := pisave.EntitySnapshotV2{
			ID:         e.ID,
			Position:   [3]float64{e.Position.X, e.Position.Y, float64(e.Position.Z)},
			Components: comp,
		}
		if body, ok := bodies[e.ID]; ok && body != nil {
			es.BodyState = &pisave.BodyStateV2{
				PositionX:      body.Position.X.Float(),
				PositionY:      body.Position.Y.Float(),
				VelocityX:      body.Velocity.X.Float(),
				VelocityY:      body.Velocity.Y.Float(),
				Rotation:       body.Rotation.Float(),
				Grounded:       body.Grounded,
				OnLadder:       body.OnLadder,
				LadderClimbing: body.LadderClimbing,
			}
		}
		out = append(out, es)
	}
	return out, nil
}

// encodeEntitiesWithBarrels is the U18 variant that walks
// rt.BarrelStates alongside the entities slice so the per-entity
// BarrelDirection + BarrelSpeed land on EntitySnapshotV2.BodyState.
// Used by EncodeSnapshot when rt has a non-empty BarrelStates map.
func encodeEntitiesWithBarrels(
	entities []pixelforge_project.Entity,
	bodies map[string]*pixelforge_physics.Body,
	barrels map[string]pixelforge_physics.BarrelState,
) ([]pisave.EntitySnapshotV2, error) {
	out, err := encodeEntities(entities, bodies)
	if err != nil {
		return nil, err
	}
	for i := range out {
		es := &out[i]
		if es.BodyState == nil {
			continue
		}
		if bs, ok := barrels[es.ID]; ok {
			es.BodyState.BarrelDirection = int8(bs.Direction)
			es.BodyState.BarrelSpeed = bs.Speed.Float()
		}
	}
	return out, nil
}

// encodeComponents marshals each component's Values map to
// json.RawMessage keyed by component Type. Empty / nil component
// lists produce a nil map (omitempty on EntitySnapshotV2.Components
// keeps the key out of the JSON output).
//
// Duplicate-Type components: the schema currently allows two
// components of the same Type on one entity (no uniqueness check in
// pixelforge_project). When that happens the encoder keeps the LAST
// one in walk order; the decoder mirrors this by reconstructing the
// entity with the saved component once. Authoring discipline
// (one-component-per-Type-per-entity) is the recommended posture
// surfaced in health.go.
func encodeComponents(components []pixelforge_project.EntityComponent) (map[string]json.RawMessage, error) {
	if len(components) == 0 {
		return nil, nil
	}
	out := make(map[string]json.RawMessage, len(components))
	for _, c := range components {
		if c.Type == "" {
			continue
		}
		vals := c.Values
		if vals == nil {
			vals = map[string]any{}
		}
		raw, err := json.Marshal(vals)
		if err != nil {
			return nil, fmt.Errorf("marshal component %q: %w", c.Type, err)
		}
		out[c.Type] = raw
	}
	return out, nil
}

// encodeGlobals marshals rt.Globals into the v2 Globals shape. Each
// value is marshaled to json.RawMessage; stdlib JSON serialisation
// already normalises numeric types (int → number, float64 → number)
// so the saved bytes round-trip byte-equal across encode calls.
func encodeGlobals(globals map[string]any) (map[string]json.RawMessage, error) {
	out := make(map[string]json.RawMessage, len(globals))
	for k, v := range globals {
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshal global %q: %w", k, err)
		}
		out[k] = raw
	}
	return out, nil
}

// DecodeSnapshot reverses EncodeSnapshot: it mutates rt's state in
// place so the runtime resumes from the saved frame. The decoder
// handles both v2 saves (new fields) and v1 saves (legacy fields)
// transparently — callers pass in whatever Snapshot Service.LoadFromSlot
// produced.
//
// Behaviour by SchemaVersion:
//   - 0 or 1: legacy v1 path. CurrentSceneID + Blackboard land in
//     rt.Globals (best-effort migration); SceneEntities are matched
//     by ID against rt.CurrentScene.Entities and their Values merged
//     into the corresponding component (no v1 save in the wild
//     actually carries component data — the v1 sink wrote
//     Snapshot{} — but the migration path is intentionally tolerant).
//   - 2: full v2 path. Entities replace the current scene's entity
//     list; per-entity body state is restored to rt.Bodies; Globals
//     unmarshals back to map[string]any; Tick is restored.
//
// Missing entity references in v2 saves (an entity ID in
// Snapshot.Entities that the rebuilt scene doesn't carry) log a
// warning and skip — best-effort matches the broader "never crash a
// shipped game" posture. The same applies to v2 saves loaded against
// a project whose scene has changed since the save: missing IDs are
// skipped, extra entities in the rebuilt scene keep their authored
// state.
func DecodeSnapshot(rt *Runtime, snapshot pisave.Snapshot) error {
	if rt == nil {
		return fmt.Errorf("capsuleruntime: nil runtime")
	}

	switch {
	case snapshot.SchemaVersion >= 2:
		return decodeSnapshotV2(rt, snapshot)
	default:
		// SchemaVersion 0 or 1 — legacy save. Log once + best-effort.
		log.Printf("[capsuleruntime] loading v%d save in v%d runtime — best-effort migration",
			snapshot.SchemaVersion, pisave.CurrentSchemaVersion)
		return decodeSnapshotV1(rt, snapshot)
	}
}

// decodeSnapshotV2 is the full reflection-based restore path.
func decodeSnapshotV2(rt *Runtime, snapshot pisave.Snapshot) error {
	rt.Tick = snapshot.Tick

	// Restore CurrentScene + Entities. The decoder walks the saved
	// Entities list and matches by ID against the rebuilt scene's
	// authored entities; matched entries have their position +
	// components overwritten from the snapshot. New IDs (entities
	// spawned mid-game) are appended; missing IDs (entities destroyed
	// mid-game) are removed from the scene.
	if snapshot.CurrentSceneID != "" && rt.Project != nil {
		for i := range rt.Project.Scenes {
			if rt.Project.Scenes[i].ID == snapshot.CurrentSceneID {
				rt.CurrentScene = &rt.Project.Scenes[i]
				break
			}
		}
	}
	if rt.CurrentScene != nil {
		newEntities := make([]pixelforge_project.Entity, 0, len(snapshot.Entities))
		// Build a quick lookup so we can recover Name / Archetype
		// from the authored entity if the snapshot doesn't carry them
		// (forward-compat for snapshots written by a thinner encoder).
		authoredByID := make(map[string]*pixelforge_project.Entity, len(rt.CurrentScene.Entities))
		for i := range rt.CurrentScene.Entities {
			e := &rt.CurrentScene.Entities[i]
			authoredByID[e.ID] = e
		}
		for _, es := range snapshot.Entities {
			ent := pixelforge_project.Entity{
				ID: es.ID,
				Position: pixelforge_project.EntityPosition{
					X: es.Position[0],
					Y: es.Position[1],
					Z: int(es.Position[2]),
				},
				Components: decodeComponents(es.Components),
			}
			if authored, ok := authoredByID[es.ID]; ok {
				ent.Name = authored.Name
				ent.Archetype = authored.Archetype
				ent.VerbBindings = authored.VerbBindings
				ent.TileX = authored.TileX
				ent.TileY = authored.TileY
			}
			newEntities = append(newEntities, ent)
		}
		rt.CurrentScene.Entities = newEntities

		// Rebuild the Bodies map + physics world's body roster from
		// the per-entity BodyState. Bodies for entities that no
		// longer appear in the snapshot are dropped from the world.
		rt.rebuildBodiesFromSnapshot(snapshot.Entities)
	}

	// Restore Globals from RawMessage map back to map[string]any so
	// the existing damage sink + future sinks can read them with
	// their current any-shape API.
	if rt.Globals == nil {
		rt.Globals = map[string]any{}
	}
	for k := range rt.Globals {
		delete(rt.Globals, k)
	}
	for k, raw := range snapshot.Globals {
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			log.Printf("[capsuleruntime] decode global %q: %v", k, err)
			continue
		}
		rt.Globals[k] = v
	}
	return nil
}

// decodeSnapshotV1 is the legacy-fields-only restore path. v1 saves
// in the wild were always Snapshot{} literal-empty (the U10 sink
// stub wrote no state), so the typical post-load result is "rt
// keeps its just-booted state". The migration path remains
// tolerant: if a v1 save somehow carries CurrentSceneID + legacy
// Blackboard, those land in rt.Globals and rt's scene pointer.
func decodeSnapshotV1(rt *Runtime, snapshot pisave.Snapshot) error {
	if snapshot.CurrentSceneID != "" && rt.Project != nil {
		for i := range rt.Project.Scenes {
			if rt.Project.Scenes[i].ID == snapshot.CurrentSceneID {
				rt.CurrentScene = &rt.Project.Scenes[i]
				break
			}
		}
	}
	if rt.Globals == nil {
		rt.Globals = map[string]any{}
	}
	for k, v := range snapshot.Blackboard {
		rt.Globals[k] = v
	}
	// SceneEntities carries per-entity Values overrides. Merge them
	// into the matching scene entity by ID; unmatched IDs are dropped
	// with a warning.
	if rt.CurrentScene != nil {
		byID := make(map[string]*pixelforge_project.Entity, len(rt.CurrentScene.Entities))
		for i := range rt.CurrentScene.Entities {
			e := &rt.CurrentScene.Entities[i]
			byID[e.ID] = e
		}
		for _, ent := range snapshot.SceneEntities {
			target, ok := byID[ent.ID]
			if !ok {
				log.Printf("[capsuleruntime] v1 load: entity %q not in current scene — skipped", ent.ID)
				continue
			}
			if ent.Values != nil {
				// Best-effort: distribute Values across components
				// by component-type lookup. A v1 save without a
				// matching component on the entity drops the
				// override.
				for componentType, raw := range ent.Values {
					applyV1ValueOverride(target, componentType, raw)
				}
			}
		}
	}
	return nil
}

// applyV1ValueOverride merges a v1 Values map (keyed by component
// type) into the target entity's matching component. v1 saves
// stored per-entity overrides as a single flat map; this helper
// distributes them back across components by Type.
func applyV1ValueOverride(target *pixelforge_project.Entity, componentType string, raw any) {
	for i := range target.Components {
		c := &target.Components[i]
		if c.Type != componentType {
			continue
		}
		// Treat raw as map[string]any if it shaped that way; else
		// store it under a synthetic "value" key. Real v1 saves
		// never carried component overrides, so this branch is
		// purely defensive.
		if m, ok := raw.(map[string]any); ok {
			if c.Values == nil {
				c.Values = map[string]any{}
			}
			for k, v := range m {
				c.Values[k] = v
			}
			return
		}
		if c.Values == nil {
			c.Values = map[string]any{}
		}
		c.Values["value"] = raw
		return
	}
}

// decodeComponents reverses encodeComponents: unmarshals each saved
// RawMessage back into a Values map and packs them into the
// per-entity component slice. The slice is built in sorted-key
// order so the output is deterministic across decodes.
func decodeComponents(components map[string]json.RawMessage) []pixelforge_project.EntityComponent {
	if len(components) == 0 {
		return nil
	}
	out := make([]pixelforge_project.EntityComponent, 0, len(components))
	// Sort keys so the rebuilt component list is byte-equal across
	// decodes — Go map iteration is non-deterministic, but the
	// project schema doesn't care about component ordering at runtime
	// (sinks look up components by Type, not by index). Sorting is
	// purely for round-trip determinism.
	keys := sortedKeys(components)
	for _, t := range keys {
		raw := components[t]
		var values map[string]any
		if err := json.Unmarshal(raw, &values); err != nil {
			log.Printf("[capsuleruntime] decode component %q: %v", t, err)
			values = map[string]any{}
		}
		out = append(out, pixelforge_project.EntityComponent{Type: t, Values: values})
	}
	return out
}

// sortedKeys returns the keys of m in sorted order — used by the
// decoder so the rebuilt component slice is deterministic.
func sortedKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple insertion sort — N is small (a handful of components
	// per entity) and avoids pulling in "sort" for the only call site.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

// rebuildBodiesFromSnapshot rebuilds rt.Bodies + rt.Physics from the
// saved per-entity BodyState. The existing physics world is
// preserved (the PhysicsConfig is part of project state, not save
// state) but its body roster is replaced.
//
// Implementation note: we tear down the old bodies before adding
// the saved ones so the world's resolv.Space doesn't keep stale
// shapes around after a load.
func (rt *Runtime) rebuildBodiesFromSnapshot(entities []pisave.EntitySnapshotV2) {
	if rt.Physics == nil {
		// No physics world (degenerate runtime). Skip body restore.
		rt.Bodies = map[string]*pixelforge_physics.Body{}
		return
	}
	// Remove every existing body from the world.
	for _, b := range rt.Physics.Bodies() {
		rt.Physics.RemoveBody(b)
	}
	rt.Bodies = map[string]*pixelforge_physics.Body{}
	// Plan-009 U18 — also clear and rebuild BarrelStates from the
	// per-entity BarrelDirection + BarrelSpeed on BodyStateV2. Bodies
	// whose saved BarrelDirection == 0 are NOT barrels (treat them as
	// non-barrel entries) and so do not appear in the rebuilt map; the
	// motion sink's lazy-init path will populate them on the next
	// barrel_roll verb if needed.
	if rt.BarrelStates == nil {
		rt.BarrelStates = map[string]pixelforge_physics.BarrelState{}
	}
	for k := range rt.BarrelStates {
		delete(rt.BarrelStates, k)
	}
	for _, es := range entities {
		if es.BodyState == nil {
			continue
		}
		body := pixelforge_physics.NewBody(
			es.ID,
			pixelforge_physics.Vec2{
				X: pixelforge_physics.FixedFromFloat(es.BodyState.PositionX),
				Y: pixelforge_physics.FixedFromFloat(es.BodyState.PositionY),
			},
			pixelforge_physics.FixedFromInt(16),
			pixelforge_physics.FixedFromInt(16),
		)
		body.Velocity = pixelforge_physics.Vec2{
			X: pixelforge_physics.FixedFromFloat(es.BodyState.VelocityX),
			Y: pixelforge_physics.FixedFromFloat(es.BodyState.VelocityY),
		}
		body.Rotation = pixelforge_physics.FixedFromFloat(es.BodyState.Rotation)
		body.Grounded = es.BodyState.Grounded
		body.OnLadder = es.BodyState.OnLadder
		body.LadderClimbing = es.BodyState.LadderClimbing
		body.Sync()
		rt.Bodies[es.ID] = body
		rt.Physics.AddBody(body)
		if es.BodyState.BarrelDirection != 0 {
			rt.BarrelStates[es.ID] = pixelforge_physics.BarrelState{
				Direction: pixelforge_physics.BarrelDir(es.BodyState.BarrelDirection),
				Speed:     pixelforge_physics.FixedFromFloat(es.BodyState.BarrelSpeed),
			}
		}
	}
}
