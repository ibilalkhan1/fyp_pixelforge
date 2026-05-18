package pixelforge_project

// world_reserved.go declares the structural types for the schema
// fields v1 of idea #1 (ScreenRoom Mario-strip primitive) reserves
// but does not consume. They land here so:
//
//   1. Forward-compat .pforge files that carry World / Zones / Warps /
//      CameraMode keys deserialise into typed values rather than being
//      silently dropped.
//   2. Future genres (Zelda-style rooms, painted zones, scene-to-scene
//      warps, per-scene camera-mode override) can ship their consumers
//      without a breaking schema change.
//
// All types are zero-value friendly and tagged omitempty so a project
// authored under v1 round-trips without any of these keys appearing in
// the JSON output. None of these types are consumed by the v1 runtime
// or editor; tests for the consumers ship in the plans that activate
// each genre.

// Rect is the integer bounding-box primitive shared by World rooms and
// Zones. Coordinates are in scene-space pixels.
type Rect struct {
	X int `json:"x,omitempty"`
	Y int `json:"y,omitempty"`
	W int `json:"w,omitempty"`
	H int `json:"h,omitempty"`
}

// World groups one or more rooms inside a single Scene. Reserved for
// the Zelda-class scenes deferred from v1. Mode names the world's
// navigation flavour (e.g. "scroll", "screen_flip"); Rooms is the
// per-room layout.
type World struct {
	Mode  string `json:"mode,omitempty"`
	Rooms []Room `json:"rooms,omitempty"`
}

// Room is one bounded sub-region inside a World. Reserved for v1; the
// shape mirrors LDtk's Level structure (ID + Name + Bounds) so a
// future importer can map directly.
type Room struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Bounds Rect   `json:"bounds,omitempty"`
}

// Zone is a painted region inside a Scene that triggers an effect
// when the Player enters it: music swap, save point, scene
// transition, scroll-mode change, etc. Kind names the trigger
// category; v1 reserves the type without naming the kinds.
type Zone struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Kind   string `json:"kind,omitempty"`
	Bounds Rect   `json:"bounds,omitempty"`
}

// Warp is a scene-to-scene transition link. From / To name Scene IDs;
// FromTile / ToTile name the tile cells the Player exits and enters
// at. Reserved for the multi-scene navigation feature deferred from
// v1; v1 projects are single-scene.
type Warp struct {
	ID       string    `json:"id,omitempty"`
	From     string    `json:"from,omitempty"`
	To       string    `json:"to,omitempty"`
	FromTile SpawnCell `json:"from_tile,omitempty"`
	ToTile   SpawnCell `json:"to_tile,omitempty"`
}
