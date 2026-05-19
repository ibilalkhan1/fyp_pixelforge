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
const CurrentSchemaVersion = 1

// Snapshot is the JSON-serializable save unit. Carries everything a
// load must restore so the player can resume their session bit-for-
// bit. The shape mirrors the brainstorm's R3 contract.
type Snapshot struct {
	SchemaVersion  int              `json:"schema_version"`
	GameTitle      string           `json:"game_title,omitempty"`
	SavedAt        time.Time        `json:"saved_at"`
	Blackboard     map[string]any   `json:"blackboard,omitempty"`
	CurrentSceneID string           `json:"current_scene_id,omitempty"`
	PlayerPos      PlayerPosition   `json:"player_pos,omitempty"`
	SceneEntities  []EntitySnapshot `json:"scene_entities,omitempty"`
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
type EntitySnapshot struct {
	ID     string         `json:"id"`
	TileX  int            `json:"tile_x,omitempty"`
	TileY  int            `json:"tile_y,omitempty"`
	Values map[string]any `json:"values,omitempty"`
}

// MarshalJSON wraps stdlib JSON with indent + escape-HTML=false to
// match pixelforge_project/saver.go's git-diff discipline. Two
// saves of the same Snapshot produce byte-identical bytes.
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
// log a warning and best-effort load.
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

// migrate is the per-version snapshot-migration hook. v1 is the
// only version today so this is a no-op; future versions add
// per-bump steps here.
func migrate(s *Snapshot) {
	_ = s
}
