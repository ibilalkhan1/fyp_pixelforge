package capsuleruntime

import (
	"log"
	"math"

	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_physics"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// motionSink is the concrete MotionSink plan-009 U8/U13/U15/U17 lands.
// It handles the motion primitives Asteroids needs (apply_thrust,
// rotate_entity, screen_wrap), the platformer essentials Mario needs
// (jump, apply_gravity, collision/solid_collide), the grid-bomber
// placement Bomberman needs (place_on_grid), and the Donkey-Kong
// climber + barrel verbs U17 lands (ladder_climb + barrel_roll), all
// backed by pixelforge_physics. Remaining motion/* topics
// (move_pattern, bounce, move_with_intent) stay routed through the
// no-op debugDrop fallback — they are explicitly out of the v2
// shipping surface (see plan-009 Scope Boundaries) and will land
// against a future authoring iteration.
//
// All methods are safe to call on a nil sink or a runtime with a nil
// Physics world; the verb-recipe surface contract is "never crash a
// shipped game", and missing-entity lookups log a warning and no-op
// rather than panic.
type motionSink struct {
	rt *Runtime
}

// newMotionSink constructs the production motionSink. Exposed as a
// constructor so fillDefaults stays the only place the rt-pointer
// wiring happens.
func newMotionSink(rt *Runtime) *motionSink {
	return &motionSink{rt: rt}
}

// Apply dispatches one motion verb to the matching concrete handler.
// Topics not yet handled (move_pattern/bounce/intent — explicitly
// out of scope for v2 per plan-009's Scope Boundaries) drop through
// debugDrop so authored recipes still dispatch cleanly without
// crashing.
func (m *motionSink) Apply(topic string, args map[string]any) {
	if m == nil || m.rt == nil {
		return
	}
	switch topic {
	case "motion/apply_thrust":
		m.applyThrust(args)
	case "motion/rotate_entity":
		m.rotateEntity(args)
	case "motion/screen_wrap":
		m.screenWrap(args)
	case "motion/jump":
		m.jump(args)
	case "motion/apply_gravity":
		m.applyGravity(args)
	case "motion/place_on_grid":
		m.placeOnGrid(args)
	case "motion/ladder_climb":
		m.ladderClimb(args)
	case "motion/barrel_roll":
		m.barrelRoll(args)
	case "collision/solid_collide":
		m.solidCollide(args)
	default:
		// Deferred topics — move_pattern / bounce / move_with_intent
		// are out of v2 scope. Drop silently rather than log every
		// tick so an authored recipe doesn't drown the log in
		// expected misses.
		debugDrop(topic, "motion handler not yet implemented", args)
	}
}

// applyThrust adds an impulse along the requested direction (degrees)
// to the named entity's velocity vector.
//
// Args:
//   - entity:    string entity ID
//   - direction: float degrees (0 = +X axis, 90 = +Y axis, mod 360)
//   - force:     float magnitude in pixels-per-tick
//
// Computation uses the deterministic SinDeg256 / CosDeg256 LUT trig so
// the same recipe produces bit-identical velocity deltas across
// platforms.
func (m *motionSink) applyThrust(args map[string]any) {
	id, ok := argString(args, "entity")
	if !ok || id == "" {
		debugDrop("motion/apply_thrust", "missing entity", args)
		return
	}
	body := m.lookupBody(id, "motion/apply_thrust", args)
	if body == nil {
		return
	}
	direction := argFloatOr(args, "direction", 0)
	force := argFloatOr(args, "force", 0)

	// Fold direction into [0, 360) to keep degree256 mapping stable
	// across negative and over-360 inputs.
	direction = math.Mod(direction, 360)
	if direction < 0 {
		direction += 360
	}
	degree256 := uint16(direction * 256)

	forceFix := pixelforge_physics.FixedFromFloat(force)
	dx := pixelforge_physics.CosDeg256(degree256).Mul(forceFix)
	dy := pixelforge_physics.SinDeg256(degree256).Mul(forceFix)
	pixelforge_physics.ApplyImpulse(body, dx, dy)
}

// rotateEntity increments the entity's rotation by `delta` degrees,
// folded into [0, 360) so the value never drifts unbounded across a
// long session.
//
// Args:
//   - entity: string entity ID
//   - delta:  float degrees (signed; negative rotates counter-clockwise)
//
// Rotation is stored as a Fixed32 angle measured in DEGREES (not
// radians) — the rotate_entity verb is the only place rotation is
// authored today, and the renderer in a later unit will convert to
// the engine's preferred angle units when it consumes Body.Rotation.
func (m *motionSink) rotateEntity(args map[string]any) {
	id, ok := argString(args, "entity")
	if !ok || id == "" {
		debugDrop("motion/rotate_entity", "missing entity", args)
		return
	}
	body := m.lookupBody(id, "motion/rotate_entity", args)
	if body == nil {
		return
	}
	delta := argFloatOr(args, "delta", 0)

	deltaFix := pixelforge_physics.FixedFromFloat(delta)
	const fixed360 = pixelforge_physics.Fixed32(360 << pixelforge_physics.FixedShift)
	next := body.Rotation.Add(deltaFix)
	// Fold into [0, 360). Use a loop rather than modulo so the
	// Fixed32 semantics stay explicit (no surprises around the
	// integer-division truncation).
	for next < 0 {
		next = next.Add(fixed360)
	}
	for next >= fixed360 {
		next = next.Sub(fixed360)
	}
	body.Rotation = next
}

// screenWrap wraps the named entity's position into the screen bounds
// when the world's PhysicsConfig has ScreenWrap enabled. Off otherwise.
//
// Args:
//   - entity: string entity ID
//
// The wrap is authoritative on the World's config, not on the args —
// passing screen_wrap to an entity in a non-wrap world is a no-op
// rather than an error, because the recipe author wrote one Asteroids
// recipe and the same recipe rides into a Mario cart as a misconfigure
// (the runtime degrades gracefully).
func (m *motionSink) screenWrap(args map[string]any) {
	id, ok := argString(args, "entity")
	if !ok || id == "" {
		debugDrop("motion/screen_wrap", "missing entity", args)
		return
	}
	body := m.lookupBody(id, "motion/screen_wrap", args)
	if body == nil {
		return
	}
	if m.rt.Physics == nil {
		return
	}
	cfg := m.rt.Physics.Config()
	if !cfg.ScreenWrap {
		// Recipe fired in a non-wrap world — silent no-op (see
		// docstring; misconfigured wraps degrade gracefully).
		return
	}
	body.Position = pixelforge_physics.WrapPosition(body.Position, cfg.ScreenWidth, cfg.ScreenHeight)
	body.Sync()
}

// lookupBody resolves an entity ID to its physics body in rt.Bodies.
// Missing entries log a warning and return nil so the calling
// handler can no-op. Centralised here so the unknown-entity log
// message is consistent across all three handlers.
func (m *motionSink) lookupBody(id, topic string, args map[string]any) *pixelforge_physics.Body {
	if m.rt.Bodies == nil {
		log.Printf("[capsuleruntime] %s: bodies map is nil (Boot did not initialise physics)", topic)
		return nil
	}
	body, ok := m.rt.Bodies[id]
	if !ok || body == nil {
		log.Printf("[capsuleruntime] %s: unknown entity %q", topic, id)
		_ = args
		return nil
	}
	return body
}

// jump applies an upward impulse to a grounded body. Variable-jump-
// height (hold-to-rise) is explicitly out of v2 — `motion/jump` is a
// single-shot impulse, gated by Body.Grounded so a recipe spamming
// the verb mid-air can't produce a double-jump.
//
// Args:
//   - entity:   string entity ID
//   - strength: float magnitude in Fixed32 units (positive — direction
//     is implicit "up"; negative values are clamped to zero so a
//     mis-authored "strength: -5" doesn't yank the player downward).
//
// Behaviour:
//   - Body.Grounded == true: ApplyJump sets Velocity.Y = -strength and
//     clears Grounded (the body is now airborne).
//   - Body.Grounded == false: no-op silently (no double-jump, no log
//     every airborne tick).
func (m *motionSink) jump(args map[string]any) {
	id, ok := argString(args, "entity")
	if !ok || id == "" {
		debugDrop("motion/jump", "missing entity", args)
		return
	}
	body := m.lookupBody(id, "motion/jump", args)
	if body == nil {
		return
	}
	if !body.Grounded {
		// Airborne — silently ignore. Double-jump is a separate
		// mechanic gated by an explicit verb (out of scope for v2).
		return
	}
	strength := argFloatOr(args, "strength", 0)
	if strength <= 0 {
		// Negative / zero magnitude is rejected: jump direction is
		// implicit-up; a negative strength would yank the body the
		// wrong way and is almost certainly a recipe bug.
		return
	}
	pixelforge_physics.ApplyJump(body, pixelforge_physics.FixedFromFloat(strength))
}

// applyGravity advances a single body one tick through the physics
// integrator (which accumulates `Gravity * dt` into the body's velocity
// and integrates velocity into position). If the project's physics
// preset enables tile-AABB collision, the body is then resolved
// against the current scene's solid tilemap so a falling body lands
// on the floor (and Grounded is set on the way down).
//
// Args:
//   - entity: string entity ID
//
// dt is fixed at 1/TPS — the integrator's gravity is stored in
// per-second² units, so dt is the standard 1/60 for a 60-TPS preset.
// Recipe authors don't pass dt explicitly; the runtime owns the tick
// cadence.
func (m *motionSink) applyGravity(args map[string]any) {
	id, ok := argString(args, "entity")
	if !ok || id == "" {
		debugDrop("motion/apply_gravity", "missing entity", args)
		return
	}
	body := m.lookupBody(id, "motion/apply_gravity", args)
	if body == nil {
		return
	}
	if m.rt.Physics == nil {
		return
	}
	cfg := m.rt.Physics.Config()
	tps := cfg.TPS
	if tps <= 0 {
		tps = 60
	}
	dt := pixelforge_physics.FixedOne.Div(pixelforge_physics.FixedFromInt(tps))
	pixelforge_physics.Integrate(body, m.rt.Physics, dt)
	// Resolve against any tilemap colliders the scene exposes. This
	// is the "gravity zeroed by ground resolution" path: a grounded
	// body whose Integrate step added downward velocity gets snapped
	// back to the tile-top + Velocity.Y zeroed by ResolveTilemap-
	// Collision, leaving Grounded true.
	m.resolveTilemap(body)
}

// solidCollide is the explicit "check this entity against the scene's
// solid tiles right now" verb. The integrator's per-tick collision
// resolution covers the common case (gravity-driven landings); this
// verb exists for recipes that move an entity via a teleport / move-
// pattern path and want to enforce solidity afterward.
//
// Args:
//   - entity: string entity ID
func (m *motionSink) solidCollide(args map[string]any) {
	id, ok := argString(args, "entity")
	if !ok || id == "" {
		debugDrop("collision/solid_collide", "missing entity", args)
		return
	}
	body := m.lookupBody(id, "collision/solid_collide", args)
	if body == nil {
		return
	}
	m.resolveTilemap(body)
}

// placeOnGrid snaps the named entity to a tile-cell origin: pixel
// position = (grid_x * tileW, grid_y * tileH). Bomberman recipes call
// this to spawn a bomb on the player's current cell or to teleport
// the hero to a specific cell on level start. Velocity is NOT cleared
// — recipes that want a hard "stop and stick" follow place_on_grid
// with their own velocity reset; the verb is a fire-and-forget
// positioning primitive.
//
// Args:
//   - entity: string entity ID
//   - grid_x: float column index (0-based)
//   - grid_y: float row index (0-based)
//
// Tile dimensions are sourced from the current scene's first
// TileAtlas (matching the U13 "first atlas wins" convention). Scenes
// with no atlas fall back to 16×16 — the Bomberman classic cell
// size — so a recipe author can fire place_on_grid before painting
// a tile layer.
func (m *motionSink) placeOnGrid(args map[string]any) {
	id, ok := argString(args, "entity")
	if !ok || id == "" {
		debugDrop("motion/place_on_grid", "missing entity", args)
		return
	}
	body := m.lookupBody(id, "motion/place_on_grid", args)
	if body == nil {
		return
	}
	gx := int(argFloatOr(args, "grid_x", 0))
	gy := int(argFloatOr(args, "grid_y", 0))
	tileW, tileH := m.gridTileSize()
	body.Position = pixelforge_physics.Vec2{
		X: pixelforge_physics.FixedFromInt(gx * tileW),
		Y: pixelforge_physics.FixedFromInt(gy * tileH),
	}
	body.Sync()
}

// gridTileSize returns the (tileW, tileH) the place_on_grid +
// grid_explode handlers use to map grid coordinates to pixel space.
// Reads the current scene's first TileAtlas; falls back to 16×16
// (Bomberman classic cell size) when no atlas is present.
func (m *motionSink) gridTileSize() (int, int) {
	if m.rt != nil && m.rt.CurrentScene != nil && len(m.rt.CurrentScene.TileAtlases) > 0 {
		atlas := &m.rt.CurrentScene.TileAtlases[0]
		if atlas.TileW > 0 && atlas.TileH > 0 {
			return atlas.TileW, atlas.TileH
		}
	}
	return 16, 16
}

// resolveTilemap walks the current scene's first TileAtlas (multi-
// atlas support is U14+) and resolves the body against it via the
// substrate's AABB sweeper. No-op when the world's collision mode
// isn't tile-AABB, when the scene has no atlases, or when the atlas
// grid is empty. Solid-tile convention: any non-zero TileID is solid.
//
// The "any non-zero ID" rule is the simplest convention that lets a
// designer paint solid platforms without an extra schema surface. v2
// will add a per-tile-class solidity flag (Scene.TileAtlas.SlopeFlags
// is the reserved field); until then a recipe that needs decorative
// non-solid tiles must keep them on a separate atlas (also U14+).
func (m *motionSink) resolveTilemap(body *pixelforge_physics.Body) {
	if m.rt.Physics == nil || m.rt.CurrentScene == nil {
		return
	}
	cfg := m.rt.Physics.Config()
	if cfg.CollisionMode != pixelforge_physics.CollisionModeTileAABB {
		return
	}
	if len(m.rt.CurrentScene.TileAtlases) == 0 {
		return
	}
	atlas := &m.rt.CurrentScene.TileAtlases[0]
	if atlas.TileW <= 0 || atlas.TileH <= 0 || len(atlas.Grid) == 0 {
		return
	}
	grid := tileAtlasGrid{atlas: atlas}
	solids := solidIDsFromAtlas(atlas)
	// Filter out the ladder marker ID so the resolver treats ladder
	// cells as pass-through (a DK-style climber must be able to walk
	// the body's AABB through ladder tiles). The ladder-overlap
	// helper detects ladders on the side-channel.
	if cfg.LadderAware {
		solids = filterID(solids, pixelforge_physics.DefaultLadderTileID)
	}
	if len(solids) == 0 {
		return
	}
	pixelforge_physics.ResolveTilemapCollision(body, m.rt.Physics, grid, solids, atlas.TileW, atlas.TileH)
}

// filterID returns ids with every occurrence of skip removed. Used
// by resolveTilemap to drop the ladder marker from the solid set
// without mutating the atlas itself.
func filterID(ids []int, skip int) []int {
	out := ids[:0:0]
	for _, id := range ids {
		if id == skip {
			continue
		}
		out = append(out, id)
	}
	return out
}

// tileAtlasGrid adapts *pixelforge_project.TileAtlas to the
// pixelforge_physics.TileGrid contract. Kept in capsuleruntime (not
// in pixelforge_project) so the substrate package stays free of an
// import on the project schema — the project type is the input, the
// physics type is the output, and the adapter lives at the seam.
type tileAtlasGrid struct {
	atlas *pixelforge_project.TileAtlas
}

func (g tileAtlasGrid) GridWidth() int {
	if g.atlas == nil || len(g.atlas.Grid) == 0 {
		return 0
	}
	// Grid is row-major: width is the length of the first row. Rows
	// of uneven width are treated as zero-padded by TileID below.
	return len(g.atlas.Grid[0])
}

func (g tileAtlasGrid) GridHeight() int {
	if g.atlas == nil {
		return 0
	}
	return len(g.atlas.Grid)
}

func (g tileAtlasGrid) TileID(col, row int) int {
	if g.atlas == nil || row < 0 || row >= len(g.atlas.Grid) {
		return 0
	}
	rowCells := g.atlas.Grid[row]
	if col < 0 || col >= len(rowCells) {
		return 0
	}
	return rowCells[col]
}

// solidIDsFromAtlas collects every distinct non-zero tile value the
// atlas's grid uses. The substrate's ResolveTilemapCollision takes
// the set as a slice (the helper builds a map internally), so we
// deduplicate here to keep the slice tight on a sparsely-painted
// atlas. Order is insertion order (matches first-seen sweep);
// substrate semantics are set-equivalent so the order doesn't affect
// resolution.
func solidIDsFromAtlas(atlas *pixelforge_project.TileAtlas) []int {
	if atlas == nil {
		return nil
	}
	seen := make(map[int]bool, 4)
	out := make([]int, 0, 4)
	for _, row := range atlas.Grid {
		for _, id := range row {
			if id == 0 || seen[id] {
				continue
			}
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// ---- U17: ladder_climb + barrel_roll ----------------------------

// dkLadderClimbSpeed is the per-tick ascent/descent speed used by
// motion/ladder_climb when no explicit "speed" arg is passed. DK's
// classic Climber climbs ~1 px/tick at 60 TPS; recipe authors who
// want a faster or slower ascent override via the verb arg.
const dkLadderClimbSpeed = 1.0

// dkBarrelRollSpeed is the per-tick horizontal sweep used by
// motion/barrel_roll when no explicit "speed" arg is passed. Tuned
// so a barrel rolls visibly across a 256-px screen in ~2 seconds at
// 60 TPS (~2 px/tick).
const dkBarrelRollSpeed = 2.0

// ladderClimb implements the motion/ladder_climb verb. Donkey-Kong
// climbers press up/down on a ladder to ascend/descend with gravity
// suppressed; pressing "off" releases the climb and gravity resumes.
//
// Args:
//   - entity:    string entity ID (required)
//   - direction: "up" | "down" | "off" (required)
//   - speed:     float per-tick climb speed (optional; defaults to
//     dkLadderClimbSpeed)
//
// Side-channel state on body:
//   - OnLadder is recomputed from the current scene's ladder tiles.
//     Climbers that have stepped off a ladder column have OnLadder=false
//     after the verb runs.
//   - LadderClimbing is set true while ascending/descending; false
//     after "off" or once the climber reaches the top of the ladder.
//
// Bus side-effects:
//   - First time a climb starts on a ladder, publishes motion/ladder_entered.
//   - On "off" or auto-top-reach, publishes motion/ladder_exited.
//   These observable events let CI replay tests assert the climb
//   transitioned correctly without snapshotting raw body state.
func (m *motionSink) ladderClimb(args map[string]any) {
	id, ok := argString(args, "entity")
	if !ok || id == "" {
		debugDrop("motion/ladder_climb", "missing entity", args)
		return
	}
	body := m.lookupBody(id, "motion/ladder_climb", args)
	if body == nil {
		return
	}
	direction, _ := argString(args, "direction")
	if direction == "" {
		debugDrop("motion/ladder_climb", "missing direction", args)
		return
	}

	// Refresh OnLadder by checking the current scene's tilemap.
	// Done every tick so a climber who has fallen off the column
	// can't climb mid-air.
	body.OnLadder = m.bodyOverlapsLadder(body)

	switch direction {
	case "off":
		if body.LadderClimbing {
			body.LadderClimbing = false
			publishVerb("motion/ladder_exited", map[string]any{"entity": id})
		}
		return
	case "up", "down":
		// Falls through to climb-step below.
	default:
		log.Printf("[capsuleruntime] motion/ladder_climb: unknown direction %q", direction)
		return
	}

	if !body.OnLadder {
		log.Printf("[capsuleruntime] motion/ladder_climb: entity %q not on a ladder — climb ignored", id)
		return
	}

	// Configure climb state for this tick.
	wasClimbing := body.LadderClimbing
	body.LadderClimbing = true
	if !wasClimbing {
		publishVerb("motion/ladder_entered", map[string]any{"entity": id})
	}

	climbSpeed := argFloatOr(args, "speed", dkLadderClimbSpeed)
	dir := pixelforge_physics.LadderUp
	if direction == "down" {
		dir = pixelforge_physics.LadderDown
	}
	state := pixelforge_physics.LadderState{Climbing: true, Direction: dir}
	pixelforge_physics.ApplyLadderMovement(body, state, pixelforge_physics.FixedFromFloat(climbSpeed))

	// Auto-release at the top: a climber who has cleared the ladder's
	// topmost tile transitions to Grounded so they step naturally onto
	// the platform above.
	if direction == "up" && m.ladderTopReached(body) {
		body.LadderClimbing = false
		body.Grounded = true
		publishVerb("motion/ladder_exited", map[string]any{"entity": id, "reason": "top_reached"})
	}
}

// barrelRoll implements the motion/barrel_roll verb. A barrel
// applies a per-tick horizontal sweep (direction tracked in the
// runtime's per-entity BarrelStates) plus gravity (via Integrate),
// then collision-resolves against the scene's solid tiles. If the
// resolution zeroes the horizontal velocity (i.e. the barrel hit a
// wall), the direction flips for the next tick.
//
// Args:
//   - entity: string entity ID (required)
//   - speed:  float per-tick horizontal sweep speed (optional;
//     defaults to dkBarrelRollSpeed)
//   - initial_direction: "left" | "right" (optional; defaults to
//     "right" on the first call; ignored on subsequent calls
//     because the runtime tracks state per-entity)
//
// This verb is fired each tick by the recipe author — there is no
// implicit "barrels keep rolling forever" loop in the substrate.
// The state map persists across calls so the direction the barrel
// is heading after a wall-bounce is remembered.
func (m *motionSink) barrelRoll(args map[string]any) {
	id, ok := argString(args, "entity")
	if !ok || id == "" {
		debugDrop("motion/barrel_roll", "missing entity", args)
		return
	}
	body := m.lookupBody(id, "motion/barrel_roll", args)
	if body == nil {
		return
	}
	if m.rt.Physics == nil {
		return
	}
	if m.rt.BarrelStates == nil {
		m.rt.BarrelStates = map[string]pixelforge_physics.BarrelState{}
	}
	state, exists := m.rt.BarrelStates[id]
	if !exists {
		speed := argFloatOr(args, "speed", dkBarrelRollSpeed)
		dirArg, _ := argString(args, "initial_direction")
		dir := pixelforge_physics.BarrelRight
		if dirArg == "left" {
			dir = pixelforge_physics.BarrelLeft
		}
		state = pixelforge_physics.NewBarrelState(dir, pixelforge_physics.FixedFromFloat(speed))
	}

	cfg := m.rt.Physics.Config()
	tps := cfg.TPS
	if tps <= 0 {
		tps = 60
	}
	dt := pixelforge_physics.FixedOne.Div(pixelforge_physics.FixedFromInt(tps))

	// Two-pass integrate so the tilemap resolver picks the right MTV
	// axis on each pass. Single-pass produces "diagonal collision
	// pushes the barrel sideways out of the floor" artefacts (the
	// substrate's tie-break only picks Y when X-velocity is zero).
	//
	// Pass 1: vertical only — clear vx, run Integrate so gravity adds
	// to vy, then ResolveTilemapCollision picks the Y axis cleanly
	// (vx=0 ⇒ preferY=true on equal-overlap ties).
	body.Velocity.X = pixelforge_physics.FixedZero
	pixelforge_physics.Integrate(body, m.rt.Physics, dt)
	m.resolveTilemap(body)

	// Pass 2: horizontal sweep — write the barrel's per-tick velocity
	// onto vx, integrate just the X axis (substrate has no axis-
	// scoped Integrate; we do the equivalent inline so gravity
	// doesn't get applied twice), then resolve. The resolver picks
	// the X axis cleanly because vy is whatever the floor-resolution
	// just zeroed it to (0 when grounded) and overlapY is now zero.
	state = pixelforge_physics.StepBarrelTick(body, state)
	body.Position.X = body.Position.X.Add(body.Velocity.X)
	body.Sync()
	m.resolveTilemap(body)

	// Post-resolve: if X-velocity zeroed (wall collision), flip dir.
	state = pixelforge_physics.ObserveBarrelCollision(body, state)
	m.rt.BarrelStates[id] = state
}

// bodyOverlapsLadder reports whether body's AABB intersects any
// ladder tile in the current scene's first TileAtlas. Returns false
// when the scene has no atlas, when the world's physics preset
// isn't LadderAware, or when the body doesn't overlap a ladder cell.
func (m *motionSink) bodyOverlapsLadder(body *pixelforge_physics.Body) bool {
	if m.rt.Physics == nil {
		return false
	}
	cfg := m.rt.Physics.Config()
	if !cfg.LadderAware {
		return false
	}
	if m.rt.CurrentScene == nil || len(m.rt.CurrentScene.TileAtlases) == 0 {
		return false
	}
	atlas := &m.rt.CurrentScene.TileAtlases[0]
	if atlas.TileW <= 0 || atlas.TileH <= 0 || len(atlas.Grid) == 0 {
		return false
	}
	grid := tileAtlasGrid{atlas: atlas}
	return pixelforge_physics.BodyOverlapsLadderTile(
		body, grid, pixelforge_physics.DefaultLadderTileID, atlas.TileW, atlas.TileH,
	)
}

// ladderTopReached mirrors bodyOverlapsLadder but delegates to the
// substrate's LadderTopReached helper for the top-of-ladder detection.
func (m *motionSink) ladderTopReached(body *pixelforge_physics.Body) bool {
	if m.rt.Physics == nil {
		return false
	}
	cfg := m.rt.Physics.Config()
	if !cfg.LadderAware {
		return false
	}
	if m.rt.CurrentScene == nil || len(m.rt.CurrentScene.TileAtlases) == 0 {
		return false
	}
	atlas := &m.rt.CurrentScene.TileAtlases[0]
	if atlas.TileW <= 0 || atlas.TileH <= 0 || len(atlas.Grid) == 0 {
		return false
	}
	grid := tileAtlasGrid{atlas: atlas}
	return pixelforge_physics.LadderTopReached(
		body, grid, pixelforge_physics.DefaultLadderTileID, atlas.TileW, atlas.TileH,
	)
}

// publishVerb is a thin wrapper that publishes a VerbEvent onto the
// global verbs bus. Used by ladderClimb to emit the ladder_entered /
// ladder_exited cascade events the CI replay tests assert on.
//
// Lives in motion_sink.go (not subscribers.go) because the only
// motion-side cascade publishes for U17 are these two events; the
// damage sink emits its own cascades via the same indirect path.
func publishVerb(topic string, args map[string]any) {
	piloop.VerbsBus().Publish(&piloop.VerbEvent{Topic: topic, Args: args})
}
