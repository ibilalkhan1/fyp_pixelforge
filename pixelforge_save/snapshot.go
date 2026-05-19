package pixelforge_save

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"
)

// CurrentSchemaVersion is the version stamp every Snapshot the
// pipeline writes today carries. Bumped when the snapshot shape
// changes in a way old loaders can't best-effort handle; migrate()
// is the per-bump hook.
//
// Plan-009 U10 bumps this from 1 → 2. v1 saves continue to load
// (UnmarshalSnapshot reads the legacy typed fields and leaves the
// new v2 fields empty); v2 saves carry the entity-component
// reflection payload the capsuleruntime snapshot encoder emits.
const CurrentSchemaVersion = 2

// ErrFutureSchema is returned by UnmarshalSnapshot when the on-disk
// SchemaVersion is strictly greater than CurrentSchemaVersion AND the
// caller has opted into the strict-future check via the
// UnmarshalStrict helper. The default UnmarshalSnapshot path keeps
// the long-standing "log a warning + best-effort load" posture so
// older callers (the snapshot-version test, the service tests) keep
// their behaviour. Higher-level callers (capsuleruntime.DecodeSnapshot)
// route through UnmarshalStrict and refuse to load future-versioned
// snapshots so a forward-incompatible save doesn't silently corrupt
// a younger runtime's state.
var ErrFutureSchema = errors.New("pixelforge_save: snapshot from future schema version")

// Snapshot is the JSON-serializable save unit. Carries everything a
// load must restore so the player can resume their session bit-for-
// bit. The shape mirrors the brainstorm's R3 contract.
//
// Plan-009 U10 schema-v2 changes (additive only — v1 fields are
// retained with omitempty so v1 saves on disk continue to load):
//   - Entities: entity-component reflection payload; one entry per
//     entity in the current scene at save time.
//   - Globals: opaque runtime-level scratchpad (e.g. lives, score)
//     stored as raw JSON so the decoder can re-marshal into any
//     concrete type the runtime expects.
//   - Tick: the runtime's frame counter, restored on load so timed
//     effects continue from the saved frame.
//
// v1 fields (Blackboard, CurrentSceneID, PlayerPos, SceneEntities)
// remain so legacy save files on disk keep loading; the v2 encoder
// stops writing them (omitempty leaves them out of the output JSON),
// the decoder uses them only when SchemaVersion <= 1.
type Snapshot struct {
	SchemaVersion  int              `json:"schema_version"`
	GameTitle      string           `json:"game_title,omitempty"`
	SavedAt        time.Time        `json:"saved_at"`
	Blackboard     map[string]any   `json:"blackboard,omitempty"`
	CurrentSceneID string           `json:"current_scene_id,omitempty"`
	PlayerPos      PlayerPosition   `json:"player_pos,omitempty"`
	SceneEntities  []EntitySnapshot `json:"scene_entities,omitempty"`

	// Plan-009 U10 — v2 entity-component reflection payload.
	// Entities is the per-entity state captured at save time. One
	// entry per entity in the current scene; component values are
	// stored as json.RawMessage so the decoder can re-marshal into
	// whichever concrete shape the runtime uses today without the
	// snapshot package owning the entity-component registry.
	// omitempty keeps v1 saves (no entries) from emitting an empty
	// "entities" key.
	Entities []EntitySnapshotV2 `json:"entities,omitempty"`

	// Globals is the runtime's cross-sink scratchpad (lives, score,
	// player HP, current player entity ID). Stored as
	// map[string]json.RawMessage so the encoder preserves byte-equal
	// numbers (1 vs 1.0) across save → load → re-save without the
	// snapshot package needing to know the concrete types each game
	// uses.
	Globals map[string]json.RawMessage `json:"globals,omitempty"`

	// Tick is the runtime's frame counter at save time. Restored so
	// timed effects (visual/flash duration, scene/wait countdown,
	// damage-fuse expiry) continue from the saved frame.
	Tick uint64 `json:"tick,omitempty"`
}

// EntitySnapshotV2 is the per-entity payload v2 saves carry. Each
// entry mirrors one pixelforge_project.Entity (ID + position +
// components) plus the entity's physics body state at save time
// (Position, Velocity, Rotation, Grounded, OnLadder, LadderClimbing)
// when one exists.
//
// Components is map[string]json.RawMessage keyed by component Type
// (e.g. "health", "inventory"). The encoder marshals the entity's
// component Values into the RawMessage. Sorted-key marshaling is
// guaranteed by stdlib json.Marshal on map[string]json.RawMessage so
// the same state produces byte-identical output across saves.
//
// BodyState is a pointer so the omitempty discipline emits no
// "body_state" key for entities without a physics body (e.g. a
// dialogue marker or a UI-only entity).
type EntitySnapshotV2 struct {
	ID         string                     `json:"id"`
	Position   [3]float64                 `json:"position"`
	Components map[string]json.RawMessage `json:"components,omitempty"`
	BodyState  *BodyStateV2               `json:"body_state,omitempty"`
}

// BodyStateV2 carries the per-entity physics body state v2 saves
// preserve. Position + Velocity are stored as float64 pairs (the
// decoder converts back to Fixed32 via FixedFromFloat); Rotation is
// stored as a raw float64 (radians). The Fixed32 round-trip is
// loss-less within the representable range (Fixed32 has 16
// fractional bits; float64 has 52 — the conversion is exact for any
// value the substrate can hold).
type BodyStateV2 struct {
	PositionX      float64 `json:"px"`
	PositionY      float64 `json:"py"`
	VelocityX      float64 `json:"vx"`
	VelocityY      float64 `json:"vy"`
	Rotation       float64 `json:"rot,omitempty"`
	Grounded       bool    `json:"grounded,omitempty"`
	OnLadder       bool    `json:"on_ladder,omitempty"`
	LadderClimbing bool    `json:"ladder_climbing,omitempty"`
	// Plan-009 U18 — BarrelDirection captures the per-tick rolling
	// direction of a DK-style barrel entity (+1 right, -1 left, 0 not
	// a barrel / inert). Additive against v2: existing saves without a
	// "barrel_dir" key decode as 0 (the "not a barrel" sentinel) and
	// keep round-trip semantics for non-barrel bodies. Stored as int8
	// to match pixelforge_physics.BarrelDir.
	BarrelDirection int8 `json:"barrel_dir,omitempty"`
	// BarrelSpeed is the per-tick horizontal sweep speed for the
	// barrel rolling pattern, stored as a float64 (Fixed32 round-trips
	// losslessly through float64 within the substrate's range — see
	// EntitySnapshotV2 docstring). Zero for non-barrel bodies.
	BarrelSpeed float64 `json:"barrel_speed,omitempty"`
}

// PlayerPosition is the player entity's tile-cell location when the
// snapshot was taken. Restored on load so the player respawns at
// the saved cell.
type PlayerPosition struct {
	TileX int `json:"tile_x"`
	TileY int `json:"tile_y"`
}

// EntitySnapshot carries per-entity state that needs to survive a
// save/load cycle (e.g. NPC dialogue progress, picked-up flags).
// Stored as a free-form values map so the snapshot stays decoupled
// from per-entity schema evolution.
//
// This is the v1 per-entity payload. v2 saves carry the richer
// EntitySnapshotV2 in Snapshot.Entities instead; the field is kept
// here (with omitempty on Snapshot.SceneEntities) so v1 saves on disk
// continue to load.
type EntitySnapshot struct {
	ID     string         `json:"id"`
	TileX  int            `json:"tile_x,omitempty"`
	TileY  int            `json:"tile_y,omitempty"`
	Values map[string]any `json:"values,omitempty"`
}

// MarshalJSON wraps stdlib JSON with indent + escape-HTML=false to
// match pixelforge_project/saver.go's git-diff discipline. Two
// saves of the same Snapshot produce byte-identical bytes.
//
// Determinism note: Go's stdlib json.Marshal sorts map[string]*
// keys lexicographically (since Go 1.12), so map[string]json.RawMessage
// (Globals + per-entity Components) round-trips byte-equal across
// saves of the same state without any custom sort.
func (s Snapshot) MarshalToJSON() ([]byte, error) {
	type wire Snapshot
	w := wire(s)
	if w.SchemaVersion == 0 {
		w.SchemaVersion = CurrentSchemaVersion
	}
	buf, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("save: marshal snapshot: %w", err)
	}
	return buf, nil
}

// UnmarshalSnapshot parses JSON bytes into a Snapshot. Missing
// keys get zero values (forward-compat); future schema versions
// log a warning and best-effort load (callers wanting a strict
// future-check use UnmarshalStrict).
func UnmarshalSnapshot(data []byte) (Snapshot, error) {
	if len(data) == 0 {
		return Snapshot{}, errors.New("save: empty snapshot bytes")
	}
	var s Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return Snapshot{}, fmt.Errorf("save: parse snapshot: %w", err)
	}
	if s.SchemaVersion == 0 {
		// Older snapshots without the version field — treat as v1.
		s.SchemaVersion = 1
	}
	if s.SchemaVersion > CurrentSchemaVersion {
		log.Printf("pixelforge_save: snapshot from future version %d "+
			"(current = %d); best-effort load",
			s.SchemaVersion, CurrentSchemaVersion)
	} else if s.SchemaVersion < CurrentSchemaVersion {
		migrate(&s)
	}
	return s, nil
}

// UnmarshalStrict is the future-incompatible-aware variant of
// UnmarshalSnapshot. Returns ErrFutureSchema (wrapped, so
// errors.Is(err, ErrFutureSchema) reports true) when the on-disk
// SchemaVersion exceeds CurrentSchemaVersion. Otherwise behaves
// exactly like UnmarshalSnapshot. The capsuleruntime snapshot
// decoder routes through this so a forward-incompatible save can't
// silently corrupt a younger runtime's state.
func UnmarshalStrict(data []byte) (Snapshot, error) {
	if len(data) == 0 {
		return Snapshot{}, errors.New("save: empty snapshot bytes")
	}
	var s Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return Snapshot{}, fmt.Errorf("save: parse snapshot: %w", err)
	}
	if s.SchemaVersion == 0 {
		s.SchemaVersion = 1
	}
	if s.SchemaVersion > CurrentSchemaVersion {
		return Snapshot{}, fmt.Errorf("%w: %d (current = %d)",
			ErrFutureSchema, s.SchemaVersion, CurrentSchemaVersion)
	}
	if s.SchemaVersion < CurrentSchemaVersion {
		migrate(&s)
	}
	return s, nil
}

// migrate is the per-version snapshot-migration hook. v1 → v2 is a
// no-op at the struct level: v1's typed fields (Blackboard,
// CurrentSceneID, PlayerPos, SceneEntities) remain on Snapshot with
// omitempty, and the capsuleruntime decoder reads them directly when
// SchemaVersion <= 1. We leave SchemaVersion at its on-disk value so
// downstream code can detect "this was a v1 save" and route through
// the legacy fields without inferring it from a sentinel.
func migrate(s *Snapshot) {
	_ = s
}
