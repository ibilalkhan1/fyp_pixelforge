// Package pixelforge_project defines the declarative .pforge schema and
// its in-memory representation. A Project is the single source of truth
// for everything the editor authors: screen size, palette and color
// tables, sprite + audio assets, scenes and their entities, behavior
// graphs, and event subscriptions. The runtime loads it via the loader
// in this package; code-gen embeds it into exported binaries.
//
// JSON is the v1 wire format. The schema is intentionally exhaustive at
// M1 even though the editor surfaces (palette UI, audio editor, scripting
// graphs) land in later milestones — reserving the fields now prevents
// breaking-change migrations later.
package pixelforge_project

import "time"

// Project is the root of a .pforge document. Field order in source = key
// order in JSON output; do not reorder without a deliberate diff.
type Project struct {
	SchemaVersion int    `json:"schema_version"`
	Name          string `json:"name"`

	ScreenWidth  int `json:"screen_width"`
	ScreenHeight int `json:"screen_height"`
	TPS          int `json:"tps"`

	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`

	Palette PaletteData `json:"palette"`

	// Theme is the editor's chrome theme. Optional in M0-M2 saves;
	// missing on load falls back to DefaultTheme so older projects open
	// cleanly. M3 introduces this field for the editor-as-cart fixture.
	Theme Theme `json:"theme"`

	Sprites  []SpriteAsset `json:"sprites"`
	Audio    []AudioSample `json:"audio"`
	Scenes   []Scene       `json:"scenes"`
	Behaviors []BehaviorGraph `json:"behaviors"`

	Bindings           []AudioBinding      `json:"bindings"`
	EventSubscriptions []EventSubscription `json:"event_subscriptions"`

	// InputMap binds named intents (e.g. "input/jump") to physical
	// keyboard keys + gamepad buttons. Optional in saves: missing on
	// load falls back to DefaultInputMap() so pre-v1 .pforge files open
	// with the standard NES-style bindings. v1 introduces this field as
	// part of idea #5 (entity verb-sheet + input intents + blackboard).
	InputMap []InputBinding `json:"input_map,omitempty"`

	// ExtensionHooks names code-extension slots a generated game can
	// wire to user-supplied Go functions. Empty in the M1 happy path.
	ExtensionHooks []ExtensionHook `json:"extension_hooks"`
}

// EventSubscription wires an entity (or a global handler) to a topic
// published on the runtime's pievent bus. Populated by M5 visual
// scripting; reserved at M1 so the schema does not break later.
type EventSubscription struct {
	// Topic is the pievent.Target name plus an event identifier, e.g.
	// "game/EventUpdate". Empty in M1 default projects.
	Topic string `json:"topic"`

	// EntityID, when non-empty, scopes the subscription to a single
	// entity in a scene. Empty means a project-global subscription.
	EntityID string `json:"entity_id"`

	// BehaviorRef is the name of a BehaviorGraph that handles this
	// event. Resolved at runtime; the schema never inlines logic.
	BehaviorRef string `json:"behavior_ref"`
}

// ExtensionHook names a callable point where the user can plug in a Go
// function. The generated main.go wires these; the editor surfaces them
// as "[code extension: X]" placeholders.
type ExtensionHook struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// NewProject returns a minimal valid Project. The defaults match the
// engine defaults (320x180, 30 TPS) so newly created projects produce
// runnable games immediately.
func NewProject(name string) *Project {
	now := time.Now().UTC()
	return &Project{
		SchemaVersion: SchemaVersion,
		Name:          name,
		ScreenWidth:   320,
		ScreenHeight:  180,
		TPS:           30,
		CreatedAt:     now,
		ModifiedAt:    now,
		Palette:       DefaultPalette(),
		Theme:         DefaultTheme(),
		Sprites:       []SpriteAsset{},
		Audio:         []AudioSample{},
		Scenes: []Scene{{
			ID:       "main",
			Name:     "Main",
			Entities: []Entity{},
			Tilemaps: []TilemapLayer{},
		}},
		Behaviors:          []BehaviorGraph{},
		Bindings:           []AudioBinding{},
		EventSubscriptions: []EventSubscription{},
		ExtensionHooks:     []ExtensionHook{},
	}
}

// applyDefaults fills in additive schema fields that older projects may
// be missing on load:
//   - Theme falls back to DefaultTheme() if zero-value (M0-M2 projects).
//   - InputMap falls back to DefaultInputMap() if empty (pre-v1 of idea #5).
//   - Each Entity.Archetype falls back to ArchetypeWorld if empty (pre-v1).
//   - Each Scene's world-primitive fields (GridWidthScreens,
//     GridHeightScreens, Camera config) get NES-class defaults if zero,
//     and SpawnTile is clamped into the resulting grid bounds (pre-v1
//     of idea #1).
//
// applyDefaults is intentionally non-destructive: a non-empty InputMap
// or a non-empty Archetype is left untouched, even if the values are
// unknown to this build (forward-compat). Camera defaults populate at
// the field granularity so a designer who overrides DeadZoneW alone
// still gets the standard LookAheadX/SmoothingX/SmoothingY defaults.
func (p *Project) applyDefaults() {
	zero := Theme{}
	if p.Theme == zero {
		p.Theme = DefaultTheme()
	}
	p.Theme.SanitizeSlots()

	if len(p.InputMap) == 0 {
		p.InputMap = DefaultInputMap()
	}

	for i := range p.Scenes {
		for j := range p.Scenes[i].Entities {
			if p.Scenes[i].Entities[j].Archetype == "" {
				p.Scenes[i].Entities[j].Archetype = DefaultArchetype
			}
		}
		p.Scenes[i].applyWorldDefaults()
	}
}

// applyWorldDefaults populates the additive world-primitive fields
// idea #1 introduces. Mutating method so it composes cleanly with the
// per-scene loop in applyDefaults and stays close to the field
// declarations it defaults.
//
// Per-field defaulting (not "the whole struct is zero, replace it")
// is the load-time contract callers depend on: a designer who saves
// a scene with a custom DeadZoneW=120 and nothing else must keep
// DeadZoneW=120 while still picking up the default LookAheadX,
// SmoothingX, SmoothingY. SpawnTile clamps into the post-default
// grid bounds so a shrink-then-save-then-load cycle never produces
// an out-of-bounds spawn.
func (s *Scene) applyWorldDefaults() {
	if s.GridWidthScreens == 0 {
		s.GridWidthScreens = DefaultGridWidthScreens
	}
	if s.GridHeightScreens == 0 {
		s.GridHeightScreens = DefaultGridHeightScreens
	}

	if s.Camera.DeadZoneW == 0 {
		s.Camera.DeadZoneW = DefaultCameraDeadZoneW
	}
	if s.Camera.DeadZoneH == 0 {
		s.Camera.DeadZoneH = DefaultCameraDeadZoneH
	}
	if s.Camera.LookAheadX == 0 {
		s.Camera.LookAheadX = DefaultCameraLookAheadX
	}
	if s.Camera.SmoothingX == 0 {
		s.Camera.SmoothingX = DefaultCameraSmoothingX
	}
	if s.Camera.SmoothingY == 0 {
		s.Camera.SmoothingY = DefaultCameraSmoothingY
	}

	// Clamp SpawnTile to the (cols, rows) grid the defaulted screen
	// dimensions imply. Clamp at max-index (cols-1) so the spawn
	// always names a valid cell. Negative values clamp to 0 so a
	// malformed pre-v1 file doesn't blow up the renderer.
	maxCol := s.GridWidthScreens*ScreenTilesWide - 1
	maxRow := s.GridHeightScreens*ScreenTilesTall - 1
	if s.SpawnTile.Col < 0 {
		s.SpawnTile.Col = 0
	}
	if s.SpawnTile.Col > maxCol {
		s.SpawnTile.Col = maxCol
	}
	if s.SpawnTile.Row < 0 {
		s.SpawnTile.Row = 0
	}
	if s.SpawnTile.Row > maxRow {
		s.SpawnTile.Row = maxRow
	}
}

// normalizeSlices replaces every nil slice with an empty one. Keeps the
// JSON output free of "null" values, which matters for git-diff
// friendliness and for downstream loaders that don't handle null
// gracefully.
func (p *Project) normalizeSlices() {
	if p.Sprites == nil {
		p.Sprites = []SpriteAsset{}
	}
	if p.Audio == nil {
		p.Audio = []AudioSample{}
	}
	if p.Scenes == nil {
		p.Scenes = []Scene{}
	}
	if p.Behaviors == nil {
		p.Behaviors = []BehaviorGraph{}
	}
	if p.Bindings == nil {
		p.Bindings = []AudioBinding{}
	}
	if p.EventSubscriptions == nil {
		p.EventSubscriptions = []EventSubscription{}
	}
	if p.ExtensionHooks == nil {
		p.ExtensionHooks = []ExtensionHook{}
	}
	for i := range p.Scenes {
		if p.Scenes[i].Entities == nil {
			p.Scenes[i].Entities = []Entity{}
		}
		if p.Scenes[i].Tilemaps == nil {
			p.Scenes[i].Tilemaps = []TilemapLayer{}
		}
		for j := range p.Scenes[i].Tilemaps {
			if p.Scenes[i].Tilemaps[j].AutoTileRules == nil {
				p.Scenes[i].Tilemaps[j].AutoTileRules = []AutoTileRule{}
			}
		}
		for j := range p.Scenes[i].Entities {
			if p.Scenes[i].Entities[j].Components == nil {
				p.Scenes[i].Entities[j].Components = []EntityComponent{}
			}
		}
	}
}
