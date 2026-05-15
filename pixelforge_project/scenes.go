package pixelforge_project

import "encoding/json"

// Scene is a self-contained collection of entities. A project has at
// least one scene; M5 visual scripting can switch between them.
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
}

// TilemapLayer is one paint-tool surface inside a scene. Values are
// arbitrary integers: in pixel mode the value is a palette index, in
// tile mode the value is an offset into the project's sprite catalog.
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
type Entity struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	Position EntityPosition `json:"position"`

	Components []EntityComponent `json:"components"`
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
