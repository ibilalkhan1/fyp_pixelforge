package capsuleruntime

import (
	"log"

	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// cardinalDirs is the (dx, dy) tuple set the grid_explode handler
// iterates: east, west, south, north (in that order, so blast events
// reach the bus in a stable, replay-able sequence). Order is fixed
// to keep CI parity tests deterministic across replays.
var cardinalDirs = [4][2]int{
	{1, 0},  // east
	{-1, 0}, // west
	{0, 1},  // south
	{0, -1}, // north
}

// damageSink is the concrete DamageSink plan-009 U9 lands. It handles
// the four damage primitives: die, hurt_player, take_damage, and
// explode_radius (point-blast variant — radial AoE that damages every
// entity within a radius of a blast center).
//
// The sink's defining design choice is that every cascade — "hp ≤ 0
// triggers die", "blast hits N entities" — is dispatched through
// pixelforge_loop's verbs.bus rather than via direct method calls.
// This preserves the observable event sequence for CI parity tests
// (U11+): a recorder subscribing to the bus sees the exact same
// sequence of damage/take_damage + damage/die + visual/flash +
// audio/play_sound events that the cart would emit at runtime, in
// the same order. Direct method calls would skip the bus and leave
// CI without a stable hook to assert on.
//
// All methods are safe to call on a nil sink or a runtime with no
// scenes; the verb-recipe surface contract is "never crash a shipped
// game", and missing-entity / no-health lookups log a warning and
// no-op rather than panic.
type damageSink struct {
	rt *Runtime
}

// newDamageSink constructs the production damageSink. Exposed as a
// constructor so fillDefaults stays the only place the rt-pointer
// wiring happens.
func newDamageSink(rt *Runtime) *damageSink {
	return &damageSink{rt: rt}
}

// Die marks the named entity as dead by publishing
// spawn/destroy_other on verbs.bus — the spawn sink consumes that
// event on the same dispatch tick and removes the entity from the
// current scene. The sink also zeroes the entity's health component
// (when present) so any per-tick observer (a renderer overlay, a
// debug HUD) sees the entity as 0 HP before the destroy lands.
//
// Empty id is a no-op (the catalog occasionally fires die with no
// target during exploratory authoring; dropping silently keeps that
// safe).
func (d *damageSink) Die(entityID string) {
	if d == nil || d.rt == nil {
		return
	}
	if entityID == "" {
		debugDrop("damage/die", "missing entity", nil)
		return
	}
	// Best-effort zero the HP component so any subscriber that
	// re-reads the entity before the destroy_other lands sees the
	// "dead" state. Entities without a health component are silently
	// fine — the destroy still fires.
	if ent := d.findEntity(entityID); ent != nil {
		pixelforge_project.SetHealthFor(ent, 0)
	}
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "spawn/destroy_other",
		Args:  map[string]any{"entity": entityID},
	})
}

// HurtPlayer decrements rt.Globals["player_hp"] by amount. When the
// resulting HP drops to ≤ 0, a chained damage/die is published on
// verbs.bus targeting the entity ID stored in
// rt.Globals["player_entity"] (when set). The cascade through the
// bus rather than a direct Die call preserves the observable event
// sequence for CI parity tests.
//
// Missing or non-numeric player_hp starts from 0 — the verb-recipe
// surface contract requires graceful degradation rather than a
// panic on a malformed Globals state. Negative amount (a heal) adds
// to player_hp without firing die.
func (d *damageSink) HurtPlayer(amount int) {
	if d == nil || d.rt == nil {
		return
	}
	if d.rt.Globals == nil {
		d.rt.Globals = map[string]any{}
	}
	hp := readGlobalInt(d.rt.Globals, "player_hp")
	hp -= amount
	if hp < 0 {
		hp = 0
	}
	d.rt.Globals["player_hp"] = hp
	if hp <= 0 && amount > 0 {
		// Cascade to die through the bus. The targeted entity is
		// the runtime's configured "player_entity" Globals key;
		// when unset (no canonical player entity), publish with an
		// empty entity arg so the bus event is still visible to
		// recorders but the spawn sink no-ops on the empty ID.
		playerEntity, _ := d.rt.Globals["player_entity"].(string)
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "damage/die",
			Args:  map[string]any{"entity": playerEntity},
		})
	}
}

// TakeDamage decrements the named entity's HP via the health
// component. When the resulting HP drops to ≤ 0, a chained damage/die
// is published on verbs.bus.
//
// Entities without a health component log a warning and no-op — the
// recipe author may have targeted a non-damageable entity (a wall, a
// marker), and the runtime degrades gracefully rather than panicking.
func (d *damageSink) TakeDamage(entityID string, amount int) {
	if d == nil || d.rt == nil {
		return
	}
	if entityID == "" {
		debugDrop("damage/take_damage", "missing entity", nil)
		return
	}
	ent := d.findEntity(entityID)
	if ent == nil {
		log.Printf("[capsuleruntime] damage/take_damage: unknown entity %q", entityID)
		return
	}
	hp, _, ok := pixelforge_project.HealthFor(ent)
	if !ok {
		log.Printf("[capsuleruntime] damage/take_damage: entity %q has no health component", entityID)
		return
	}
	hp -= amount
	if hp < 0 {
		hp = 0
	}
	pixelforge_project.SetHealthFor(ent, hp)
	if hp <= 0 && amount > 0 {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "damage/die",
			Args:  map[string]any{"entity": entityID},
		})
	}
}

// ExplodeRadius applies a point-blast AoE: every entity within
// `radius` pixels of (center_x, center_y) takes `damage` damage via
// a chained damage/take_damage event on verbs.bus. The sink also
// emits a visual/flash on a synthetic "_blast_center" entity ID
// (the renderer treats this as a transient blast effect rather than
// an authored entity) and an audio/play_sound for the explosion SFX
// (only when "explosion_sfx" is registered in the project's audio
// list — missing-SFX projects silently skip the audio event).
//
// Distance comparison uses squared-distance (dx² + dy² ≤ radius²)
// to avoid sqrt — keeps the math fully deterministic across
// platforms and avoids the float-rounding wiggle a sqrt would
// introduce.
//
// Args:
//   - center_x: float pixels (blast origin X)
//   - center_y: float pixels (blast origin Y)
//   - radius:   float pixels (inclusive radius)
//   - damage:   int (damage applied to each within-radius entity)
func (d *damageSink) ExplodeRadius(args map[string]any) {
	if d == nil || d.rt == nil || d.rt.CurrentScene == nil {
		return
	}
	cx := argFloatOr(args, "center_x", 0)
	cy := argFloatOr(args, "center_y", 0)
	radius := argFloatOr(args, "radius", 0)
	damage := int(argFloatOr(args, "damage", 0))
	if radius <= 0 {
		debugDrop("damage/explode_radius", "non-positive radius", args)
		return
	}
	radius2 := radius * radius

	// First, publish the visual+audio so recorders see the blast
	// effects before the per-entity damage events — matches the
	// recipe's authorial intent ("flash + boom, then hurt").
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "visual/flash",
		Args: map[string]any{
			"entity": "_blast_center",
			"color":  "#ffffff",
			"ms":     float64(200),
		},
	})
	if d.hasAudioSample("explosion_sfx") {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "audio/play_sound",
			Args:  map[string]any{"name": "explosion_sfx"},
		})
	}

	// Iterate the live scene entities. We deliberately do not
	// short-circuit on the first hit — the blast damages every
	// entity in range, and each damage event must reach the bus
	// for CI parity recorders. Iteration order is the authored
	// order in CurrentScene.Entities; this is stable so the
	// recorded event sequence is reproducible.
	ents := d.rt.CurrentScene.Entities
	for i := range ents {
		e := &ents[i]
		dx := e.Position.X - cx
		dy := e.Position.Y - cy
		if dx*dx+dy*dy > radius2 {
			continue
		}
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "damage/take_damage",
			Args: map[string]any{
				"entity": e.ID,
				"amount": float64(damage),
			},
		})
	}
}

// GridExplode applies a grid-cardinal blast: from origin (gx, gy) the
// blast walks up to `range` tiles outward in each of the 4 cardinal
// directions, stopping at the first solid tile encountered along each
// ray. Every entity whose Body overlaps an affected tile takes
// `damage` damage via a chained damage/take_damage event on
// verbs.bus. Used by Bomberman's bomb fuse (after the BombTimer fires
// damage/grid_explode at the bomb's grid position).
//
// The handler also emits one visual/flash for the blast center
// (entity "_blast_center", matching ExplodeRadius's convention so
// renderers can share the flash sprite) and an audio/play_sound for
// the explosion SFX (only when "explosion_sfx" is registered in the
// project's audio list — missing-SFX projects silently skip the
// audio event).
//
// Args:
//   - origin: map[string]any{"gx": int, "gy": int} — blast origin in
//     grid coordinates. The center tile itself is NOT damaged (the
//     bomb that authored the blast was destroyed before the explode
//     fired — see BombTimer in runtime.go).
//   - range:  int (tiles outward in each cardinal direction)
//   - damage: int (damage applied to each entity on an affected tile)
//
// Solid-tile convention matches U13's resolveTilemap: any non-zero
// TileID in the scene's first TileAtlas blocks the blast. Scenes
// without a TileAtlas fall back to 16×16 cells with no solid tiles,
// so the blast walks the full range in every direction.
func (d *damageSink) GridExplode(args map[string]any) {
	if d == nil || d.rt == nil || d.rt.CurrentScene == nil {
		return
	}
	rng := int(argFloatOr(args, "range", 0))
	if rng <= 0 {
		debugDrop("damage/grid_explode", "non-positive range", args)
		return
	}
	originGX, originGY, okOrigin := readGridOrigin(args)
	if !okOrigin {
		debugDrop("damage/grid_explode", "missing origin", args)
		return
	}
	damage := int(argFloatOr(args, "damage", 0))

	// Visual + audio fire first so recorders see the blast effects
	// before the per-entity damage events, matching the
	// ExplodeRadius cascade order.
	piloop.VerbsBus().Publish(&piloop.VerbEvent{
		Topic: "visual/flash",
		Args: map[string]any{
			"entity": "_blast_center",
			"color":  "#ffffff",
			"ms":     float64(200),
		},
	})
	if d.hasAudioSample("bomb_explode") {
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "audio/play_sound",
			Args:  map[string]any{"name": "bomb_explode"},
		})
	}

	tileW, tileH := d.gridTileSize()
	atlas := d.solidAtlas()

	// Walk each cardinal ray. Order is fixed (east/west/south/north)
	// so the publish sequence is deterministic for CI parity.
	for _, dir := range cardinalDirs {
		dx, dy := dir[0], dir[1]
		for step := 1; step <= rng; step++ {
			tx := originGX + dx*step
			ty := originGY + dy*step
			if isSolidCell(atlas, tx, ty) {
				// Blast is blocked by the solid tile. The convention
				// here is "solid tile stops the blast at that cell";
				// the solid itself is not damaged (walls don't have
				// HP in this Bomberman cut). U16's breakable-tile
				// logic will overlay a wall-damage path on top.
				break
			}
			d.publishDamageOnTile(tx, ty, tileW, tileH, damage)
		}
	}
}

// publishDamageOnTile fires damage/take_damage at every entity whose
// body overlaps the (tx, ty) tile. The tile bounds are
// [tx*tileW, (tx+1)*tileW) × [ty*tileH, (ty+1)*tileH) in pixel space;
// an entity's AABB overlaps the tile iff its position bounds touch
// the tile bounds. Iteration order is the authored entity order in
// CurrentScene.Entities so the recorded event sequence is
// reproducible.
func (d *damageSink) publishDamageOnTile(tx, ty, tileW, tileH, damage int) {
	tileMinX := float64(tx * tileW)
	tileMaxX := float64((tx + 1) * tileW)
	tileMinY := float64(ty * tileH)
	tileMaxY := float64((ty + 1) * tileH)

	ents := d.rt.CurrentScene.Entities
	for i := range ents {
		e := &ents[i]
		// Entity AABB defaults to 16×16 (matching Boot's body
		// defaults). The collision test uses the entity's *body*
		// dimensions when available so the blast catches narrower
		// hitboxes correctly; the body is the source of truth for
		// physics in this runtime.
		w, h := entityAABB(d.rt, e)
		ex := e.Position.X
		ey := e.Position.Y
		if ex+w <= tileMinX || ex >= tileMaxX {
			continue
		}
		if ey+h <= tileMinY || ey >= tileMaxY {
			continue
		}
		piloop.VerbsBus().Publish(&piloop.VerbEvent{
			Topic: "damage/take_damage",
			Args: map[string]any{
				"entity": e.ID,
				"amount": float64(damage),
			},
		})
	}
}

// gridTileSize sources (tileW, tileH) from the current scene's first
// TileAtlas; falls back to 16×16 when no atlas is present (the
// Bomberman classic cell size).
func (d *damageSink) gridTileSize() (int, int) {
	if d.rt != nil && d.rt.CurrentScene != nil && len(d.rt.CurrentScene.TileAtlases) > 0 {
		atlas := &d.rt.CurrentScene.TileAtlases[0]
		if atlas.TileW > 0 && atlas.TileH > 0 {
			return atlas.TileW, atlas.TileH
		}
	}
	return 16, 16
}

// solidAtlas returns the current scene's first TileAtlas (or nil when
// the scene has none). The blast checks isSolidCell against this
// atlas to decide whether each step in a cardinal ray is blocked.
func (d *damageSink) solidAtlas() *pixelforge_project.TileAtlas {
	if d.rt != nil && d.rt.CurrentScene != nil && len(d.rt.CurrentScene.TileAtlases) > 0 {
		return &d.rt.CurrentScene.TileAtlases[0]
	}
	return nil
}

// isSolidCell reports whether the (col, row) cell in atlas carries a
// non-zero tile value (matching U13's "any non-zero TileID is solid"
// convention). Out-of-range coords are treated as solid so the blast
// stops at the level boundary rather than walking infinitely.
func isSolidCell(atlas *pixelforge_project.TileAtlas, col, row int) bool {
	if atlas == nil {
		// No atlas painted → no solids → blast walks the full
		// range. Boundary is treated as open (callers cap by range).
		return false
	}
	if row < 0 || row >= len(atlas.Grid) {
		return true // out-of-range vertically: stop the blast
	}
	cells := atlas.Grid[row]
	if col < 0 || col >= len(cells) {
		return true // out-of-range horizontally: stop the blast
	}
	return cells[col] != 0
}

// readGridOrigin extracts gx + gy from args. Accepts both the
// nested-map form (Args{"origin": {"gx": int, "gy": int}}, the form
// the bomb timer publishes) and the flat form (Args{"origin_gx": int,
// "origin_gy": int}, useful for hand-authored recipes). Returns
// (0, 0, false) when neither form is present.
func readGridOrigin(args map[string]any) (int, int, bool) {
	if args == nil {
		return 0, 0, false
	}
	if raw, ok := args["origin"]; ok && raw != nil {
		if m, mok := raw.(map[string]any); mok {
			gx := int(argFloatOr(m, "gx", 0))
			gy := int(argFloatOr(m, "gy", 0))
			_, hasX := m["gx"]
			_, hasY := m["gy"]
			if hasX || hasY {
				return gx, gy, true
			}
		}
	}
	_, hasX := args["origin_gx"]
	_, hasY := args["origin_gy"]
	if hasX || hasY {
		return int(argFloatOr(args, "origin_gx", 0)), int(argFloatOr(args, "origin_gy", 0)), true
	}
	return 0, 0, false
}

// entityAABB returns the (width, height) the blast overlap test uses
// for entity e. Prefers the live physics body's dimensions (the
// source of truth for collision); falls back to 16×16 when the body
// is absent (e.g. entity was authored but the runtime hasn't seen a
// motion verb on it yet).
func entityAABB(rt *Runtime, e *pixelforge_project.Entity) (float64, float64) {
	if rt != nil && rt.Bodies != nil {
		if body, ok := rt.Bodies[e.ID]; ok && body != nil {
			return body.Width.Float(), body.Height.Float()
		}
	}
	return 16, 16
}

// findEntity returns a pointer to the entity in the current scene
// with the matching ID, or nil when absent. The pointer is into the
// scene's live slice so callers can mutate component values directly
// (the spawnSink's destroy path uses the same pattern).
func (d *damageSink) findEntity(id string) *pixelforge_project.Entity {
	if d.rt == nil || d.rt.CurrentScene == nil {
		return nil
	}
	ents := d.rt.CurrentScene.Entities
	for i := range ents {
		if ents[i].ID == id {
			return &ents[i]
		}
	}
	return nil
}

// hasAudioSample returns true when the project's Audio list contains
// an entry with the requested name. Used by ExplodeRadius to skip
// the audio/play_sound publish when the project hasn't authored an
// "explosion_sfx" sample — keeps the bus event-stream clean and
// avoids a flood of "unknown sample" warnings from the audio sink.
func (d *damageSink) hasAudioSample(name string) bool {
	if d.rt == nil || d.rt.Project == nil {
		return false
	}
	for i := range d.rt.Project.Audio {
		if d.rt.Project.Audio[i].Name == name {
			return true
		}
	}
	return false
}

// readGlobalInt extracts an int from rt.Globals. JSON-style float64,
// programmatic int, and a few int sibling types all coerce; absent
// or wrong-type keys default to 0 (matching the broader sink "missing
// arg means zero" posture).
func readGlobalInt(globals map[string]any, key string) int {
	if globals == nil {
		return 0
	}
	raw, ok := globals[key]
	if !ok {
		return 0
	}
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case int32:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	}
	return 0
}
