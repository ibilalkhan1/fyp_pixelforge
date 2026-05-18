package pixelforge_project

import (
	"bytes"
	"encoding/json"
)

// Scene is a self-contained collection of entities. A project has at
// least one scene; M5 visual scripting can switch between them.
//
// v1 of idea #1 (ScreenRoom Mario-strip primitive) adds a cluster of
// additive omitempty fields that promote a Scene from a flat entity
// bag into a bounded 2D world primitive: GridWidthScreens /
// GridHeightScreens describe the level's bounds in 256x240 screens
// (each screen is 32x30 tiles of 8x8), SpawnTile names the cell the
// Player teleports to on scene start, DefaultTileID is the value used
// for new cells on grid grow, and Camera carries the smooth-follow
// configuration the engine's pixelforge_camera package consumes.
// World/Zones/Warps/CameraMode are reserved for future genres
// (Zelda-style rooms, painted music/save zones, scene-to-scene warps,
// per-scene camera-mode override) and serialise as omitempty so pre-v1
// projects round-trip without spurious keys.
type Scene struct {
	// ID is a stable string used by EventSubscriptions and behaviors
	// to reference the scene. Unique within Project.Scenes.
	ID string `json:"id"`

	// Name is the human-readable label shown in the editor.
	Name string `json:"name"`

	// Entities are the placed objects in this scene. Order is
	// preserved across save / load and matters only for tie-breaks
	// in mouse-pick selection.
	Entities []Entity `json:"entities"`

	// Tilemaps are the paint-tool layers authored in M2. Each layer
	// is a flat grid of integer tile values. Backwards-compatible:
	// older .pforge files without this field round-trip as an empty
	// slice.
	Tilemaps []TilemapLayer `json:"tilemaps"`

	// GridWidthScreens is the level's horizontal size, measured in
	// 256-px screens. Defaults to 16 (a 16-screen-wide Mario-class
	// strip) via applyDefaults. omitempty so pre-v1 files round-trip.
	GridWidthScreens int `json:"grid_width_screens,omitempty"`

	// GridHeightScreens is the level's vertical size in 240-px
	// screens. Defaults to 1 (single-screen tall, Mario-strip
	// shape). omitempty so pre-v1 files round-trip.
	GridHeightScreens int `json:"grid_height_screens,omitempty"`

	// SpawnTile is the tile cell the Player entity teleports to on
	// scene start. Defaults to {0, 0}. Right-click "Set spawn here"
	// in the editor writes this. applyDefaults clamps it to within
	// the grid bounds.
	SpawnTile SpawnCell `json:"spawn_tile,omitempty"`

	// DefaultTileID is the value used for newly-grown grid cells on
	// resize, and the visual "empty" cell in the painter. Defaults
	// to 0.
	DefaultTileID int `json:"default_tile_id,omitempty"`

	// Camera is the per-scene smooth-follow configuration the
	// pixelforge_camera Follower consumes. Zero-value individual
	// fields are populated to their NES-class defaults by
	// applyDefaults so partial overrides still produce a working
	// camera. omitempty on the struct keeps pre-v1 files clean.
	Camera SceneCameraConfig `json:"camera,omitempty"`

	// World is reserved for future Zelda-style multi-room scenes.
	// Unused in v1; zero-value omitempty so pre-v1 files round-trip
	// and forward-compat files preserve unknown contents.
	World World `json:"world,omitempty"`

	// Zones is reserved for painted regions that trigger music
	// swaps, save points, transitions, or scroll-mode changes.
	// Unused in v1.
	Zones []Zone `json:"zones,omitempty"`

	// Warps is reserved for scene-to-scene transitions. Unused in
	// v1 (single-scene projects only).
	Warps []Warp `json:"warps,omitempty"`

	// CameraMode is reserved for per-scene override of the camera
	// algorithm (e.g. "smooth_follow" vs "snap_per_screen"). v1
	// always uses smooth-follow; this field is omitempty and
	// preserved across round-trips for forward-compat.
	CameraMode string `json:"camera_mode,omitempty"`
}

// SpawnCell names a tile cell by (Col, Row) — column-major addressing
// to match the engine's (X, Y) sprite-blit semantics. Both fields are
// omitempty so the {0, 0} default does not pollute pre-v1 JSON.
type SpawnCell struct {
	Col int `json:"col,omitempty"`
	Row int `json:"row,omitempty"`
}

// SceneCameraConfig carries the smooth-follow tunables the
// pixelforge_camera Follower reads each tick. Defaults match Phase 1
// research convergence (Mark Brown / GMTK + Cave Story patterns):
// 30% horizontal dead-zone (77 px of 256), 40% vertical dead-zone
// (96 px of 240), 32-px velocity-based look-ahead, framerate-
// independent smoothing constants 0.20 / 0.40. applyDefaults fills
// each field individually if zero so designers can override one knob
// without losing the others.
type SceneCameraConfig struct {
	DeadZoneW  float64 `json:"dead_zone_w,omitempty"`
	DeadZoneH  float64 `json:"dead_zone_h,omitempty"`
	LookAheadX float64 `json:"look_ahead_x,omitempty"`
	SmoothingX float64 `json:"smoothing_x,omitempty"`
	SmoothingY float64 `json:"smoothing_y,omitempty"`
}

// Scene grid + camera defaults. Exposed as exported constants so the
// editor's resize UI and the pixelforge_camera package can reference
// the same source of truth.
const (
	DefaultGridWidthScreens  = 16
	DefaultGridHeightScreens = 1

	// ScreenTilesWide / ScreenTilesTall describe how many 8x8 tiles
	// fit in one 256x240 screen. Used by applyDefaults to compute
	// SpawnTile clamp bounds: GridWidthScreens * ScreenTilesWide.
	ScreenTilesWide = 32
	ScreenTilesTall = 30

	DefaultCameraDeadZoneW  = 77.0
	DefaultCameraDeadZoneH  = 96.0
	DefaultCameraLookAheadX = 32.0
	DefaultCameraSmoothingX = 0.20
	DefaultCameraSmoothingY = 0.40
)

// MarshalJSON gives Scene struct-aware omitempty discipline. Stdlib
// encoding/json never suppresses non-pointer struct values even when
// tagged omitempty, but pre-v1 .pforge files need to round-trip
// without spurious "spawn_tile", "camera", "world", "zones", "warps",
// "camera_mode" keys appearing. The hand-rolled marshaler emits each
// optional struct field only when at least one inner field is
// non-zero (or the slice is non-empty), keeping the omitempty
// contract that the field tags advertise.
//
// The hand-rolled emission is also the canonical key order: existing
// non-optional keys first (id, name, entities, tilemaps), then the
// optional world-primitive additions in declaration order. Two saves
// of the same Scene produce byte-identical JSON, preserving the
// git-merge-friendly invariant the project's saver relies on.
func (s Scene) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')

	writeKey := func(first *bool, key string) {
		if !*first {
			buf.WriteByte(',')
		}
		*first = false
		buf.WriteByte('"')
		buf.WriteString(key)
		buf.WriteString(`":`)
	}

	writeValue := func(v any) error {
		raw, err := json.Marshal(v)
		if err != nil {
			return err
		}
		buf.Write(raw)
		return nil
	}

	first := true

	// Always-emitted core fields. Match the on-disk shape of the
	// existing fixtures: id, name, entities, tilemaps.
	writeKey(&first, "id")
	if err := writeValue(s.ID); err != nil {
		return nil, err
	}
	writeKey(&first, "name")
	if err := writeValue(s.Name); err != nil {
		return nil, err
	}
	writeKey(&first, "entities")
	ents := s.Entities
	if ents == nil {
		ents = []Entity{}
	}
	if err := writeValue(ents); err != nil {
		return nil, err
	}
	writeKey(&first, "tilemaps")
	tms := s.Tilemaps
	if tms == nil {
		tms = []TilemapLayer{}
	}
	if err := writeValue(tms); err != nil {
		return nil, err
	}

	// Optional world-primitive fields. Emit only when non-zero so
	// pre-v1 files (and v1 scenes that never touched a field) stay
	// byte-stable through round-trip.
	if s.GridWidthScreens != 0 {
		writeKey(&first, "grid_width_screens")
		if err := writeValue(s.GridWidthScreens); err != nil {
			return nil, err
		}
	}
	if s.GridHeightScreens != 0 {
		writeKey(&first, "grid_height_screens")
		if err := writeValue(s.GridHeightScreens); err != nil {
			return nil, err
		}
	}
	if s.SpawnTile != (SpawnCell{}) {
		writeKey(&first, "spawn_tile")
		if err := writeValue(s.SpawnTile); err != nil {
			return nil, err
		}
	}
	if s.DefaultTileID != 0 {
		writeKey(&first, "default_tile_id")
		if err := writeValue(s.DefaultTileID); err != nil {
			return nil, err
		}
	}
	if s.Camera != (SceneCameraConfig{}) {
		writeKey(&first, "camera")
		if err := writeValue(s.Camera); err != nil {
			return nil, err
		}
	}
	if s.World.Mode != "" || len(s.World.Rooms) > 0 {
		writeKey(&first, "world")
		if err := writeValue(s.World); err != nil {
			return nil, err
		}
	}
	if len(s.Zones) > 0 {
		writeKey(&first, "zones")
		if err := writeValue(s.Zones); err != nil {
			return nil, err
		}
	}
	if len(s.Warps) > 0 {
		writeKey(&first, "warps")
		if err := writeValue(s.Warps); err != nil {
			return nil, err
		}
	}
	if s.CameraMode != "" {
		writeKey(&first, "camera_mode")
		if err := writeValue(s.CameraMode); err != nil {
			return nil, err
		}
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// TilemapLayer is one paint-tool surface inside a scene. Values are
// arbitrary integers: in pixel mode the value is a palette index, in
// tile mode the value is an offset into the project's sprite catalog.
//
// v1 of idea #1 adds SpriteSheetRef, naming the SpriteAsset in the
// project whose frames the layer's Grid values index into. Empty
// SpriteSheetRef means the layer renders nothing (the renderer skips
// unbound layers); omitempty so pre-v1 files round-trip.
type TilemapLayer struct {
	// Name is the editor-displayed layer label.
	Name string `json:"name"`

	// TileW / TileH are the tile dimensions in scene-space pixels.
	TileW int `json:"tile_w"`
	TileH int `json:"tile_h"`

	// Grid is a row-major matrix of tile values. Out-of-range coords
	// are no-ops at paint time.
	Grid [][]int `json:"grid"`

	// AutoTileRules captures user-painted neighbor patterns that the
	// editor synthesizes into transition rules. The runtime treats
	// rules as a hint — the on-disk Grid is the source of truth.
	AutoTileRules []AutoTileRule `json:"auto_tile_rules"`

	// SpriteSheetRef names a SpriteAsset in the project whose frames
	// this layer's Grid values index into (cell ID N -> frame N-1 of
	// the bound sheet; ID 0 = empty). Empty when unbound; omitempty
	// so pre-v1 files round-trip without spurious keys.
	SpriteSheetRef string `json:"sprite_sheet_ref,omitempty"`
}

// AutoTileRule binds a 3×3 neighbor pattern to an output tile value.
// The middle cell (index 4) of Pattern is the cell being painted; the
// surrounding cells are the read context. -1 cells are wildcards.
type AutoTileRule struct {
	Pattern [9]int `json:"pattern"`
	Output  int    `json:"output"`
	Count   int    `json:"count"`
}

// Entity is one placed object in a scene. Entities have a stable string
// ID (so event subscriptions can target them across edits), a position,
// and a flat list of components. Components hold all per-entity state.
//
// v1 of idea #5 adds two additive omitempty fields: Archetype names one
// of the six character-sheet roles (Player/Enemy/Pickup/Hazard/NPC/
// World); VerbBindings holds the per-trigger-slot verb assignments the
// runtime compiles into EventSheetRule entries. Both are absent in pre-
// v1 projects; the loader's applyDefaults sanitises empty Archetype to
// ArchetypeWorld and leaves VerbBindings as a nil/empty slice.
//
// v1 of idea #1 (ScreenRoom Mario-strip primitive) adds TileX/TileY
// as the canonical tile-cell coordinates for entity placement going
// forward. Plan reconciliation: idea #1's plan U1 originally said
// "no new field on Entity in this unit" but U4 (entity renderer) and
// U7 (tile-cell placement) both assume TileX/TileY exist. The plan's
// "Key Technical Decisions" section is authoritative ("Entity
// positions: dual representation. New TileX/TileY int fields
// (canonical going forward); existing float64 Position.X/Y derived"),
// so TileX/TileY land here in U1. Existing pixel-coord Position stays
// for backwards compatibility; idea #1's U7 will derive TileX/TileY
// from Position at load time for pre-v1 entities. omitempty on both
// keeps the zero-value {0, 0} from polluting pre-v1 JSON.
type Entity struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	Position EntityPosition `json:"position"`

	Components []EntityComponent `json:"components"`

	// Archetype names the entity's character-sheet role. Sanitised to
	// ArchetypeWorld on load if empty. omitempty so pre-v1 entities
	// round-trip without spurious new keys when their archetype was
	// never explicitly set (defaulting happens at load, not save).
	Archetype string `json:"archetype,omitempty"`

	// VerbBindings list the per-trigger-slot verb-recipe assignments
	// for the character-sheet inspector. Each entry compiles to one
	// EventSheetRule at load time. Empty for pre-v1 entities and for
	// designer-untouched entities of any archetype.
	VerbBindings []VerbBinding `json:"verb_bindings,omitempty"`

	// TileX / TileY are the canonical tile-cell coordinates idea #1
	// v1 introduces. The pixelforge_entity renderer reads these to
	// blit each entity at (TileX*TileW, TileY*TileH) in scene-space.
	// omitempty so pre-v1 entities (which carry only Position) and
	// entities authored at cell (0,0) round-trip without a phantom
	// key.
	TileX int `json:"tile_x,omitempty"`
	TileY int `json:"tile_y,omitempty"`
}

// EntityPosition is the entity's transform in scene-space pixels. Z is
// the layer used for paint order; higher Z draws on top.
type EntityPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z int     `json:"z"`
}

// EntityComponent is one typed value attached to an entity. The
// reflection-driven registry in pfcomponent resolves the Type string to
// a concrete Go struct, then unmarshals Values into it. The schema
// itself stores Values as a free-form JSON map so the editor never has
// to migrate when a component grows fields.
type EntityComponent struct {
	Type   string         `json:"type"`
	Values map[string]any `json:"values"`
}

// MarshalJSON serialises Values with sorted keys so two equivalent
// projects produce byte-identical output. Go's stdlib json.Marshal
// already sorts map keys (since Go 1.12), so this is technically only
// belt-and-braces — but spelling it out keeps the determinism contract
// explicit, and the loader/saver tests assert it.
func (c EntityComponent) MarshalJSON() ([]byte, error) {
	// json.Marshal on map[string]any sorts keys lexicographically.
	// Use an explicit struct so the field order ("type" before
	// "values") is also deterministic.
	type wire struct {
		Type   string          `json:"type"`
		Values json.RawMessage `json:"values"`
	}
	vals, err := json.Marshal(c.Values)
	if err != nil {
		return nil, err
	}
	// If Values was nil, emit {} rather than null. Loaders treat the
	// two equivalently, but {} round-trips cleanly through the
	// MarshalIndent pipeline used by Save.
	if string(vals) == "null" {
		vals = []byte(`{}`)
	}
	return json.Marshal(wire{Type: c.Type, Values: vals})
}
