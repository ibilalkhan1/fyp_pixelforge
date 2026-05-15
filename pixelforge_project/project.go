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
// be missing on load. Currently scoped to the Theme zero-value fallback
// so M0-M2 projects open with sensible chrome colours.
func (p *Project) applyDefaults() {
	zero := Theme{}
	if p.Theme == zero {
		p.Theme = DefaultTheme()
	}
	p.Theme.SanitizeSlots()
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
