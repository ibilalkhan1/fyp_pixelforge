// rpg_schema.go owns idea #6 v1 U4's schema additions: Dialogues,
// Menus, Items, and SaveConfig. Each is additive omitempty so pre-v1
// projects round-trip cleanly (AE7); applyDefaults backfills empty
// maps/slices and a default SaveConfig so the runtime never sees a
// nil reference.
//
// Types stay declarative — DialogueScript carries the raw script
// string, MenuConfig carries template name + parameter map,
// ItemDefinition carries id/name/icon/description/effect/category.
// The runtime engines (pixelforge_dialogue, pixelforge_menus) read
// these structs and bring them to life.
package pixelforge_project

// DialogueScript is one named dialogue tree's authoring source. The
// runtime parser (pixelforge_dialogue) converts the raw string into
// a node tree on load + on edit; the script string is what the
// designer types in the Dialogue workspace.
type DialogueScript struct {
	// Name is the project-scoped identifier the verb-recipe
	// "open_dialogue" references. Unique within Project.Dialogues.
	Name string `json:"name"`

	// Script is the multi-line authoring source. Screenplay-style
	// (SPEAKER: text), Twine labels (:: label), choices
	// ([[text -> label | if cond]]), {state.key} interpolation,
	// and stage directions (walk_left N, pause N, etc.).
	Script string `json:"script"`
}

// MenuConfig is one named menu instance bound to one of the nine
// canonical templates. Parameters are a free-form key/value map so
// the workspace + runtime stay decoupled from template-specific
// schemas.
type MenuConfig struct {
	// Name is the project-scoped identifier the verb-recipe
	// "open_menu" references.
	Name string `json:"name"`

	// Template names one of the registered templates (title,
	// game_over, high_score, pause, save_game, load_game,
	// inventory, status, stage_select). Unknown template names
	// fall back to a "(unknown template)" placeholder renderer.
	Template string `json:"template"`

	// Parameters is the template-specific configuration the
	// designer authors in the workspace's parameter editor. e.g.
	// title template carries {"game_name": "...", "subtitle": "..."}.
	// Empty / missing keys default per-template.
	Parameters map[string]any `json:"parameters,omitempty"`
}

// ItemDefinition is one entry in the project's flat item database
// (Project.Items). The blackboard's "inventory" key holds a slice
// of InventoryEntry (item_id, count) that references these by ID.
type ItemDefinition struct {
	// ID is the stable identifier inventory + verb-recipes reference.
	ID string `json:"id"`

	// Name is the human-readable label (rendered in the inventory
	// menu and item-picker dropdowns).
	Name string `json:"name"`

	// Icon is the sprite name (sprite asset's Name field). The
	// SpriteThumbnailWidget renders a small preview using this.
	Icon string `json:"icon,omitempty"`

	// Description is the long-form text rendered in the inventory
	// menu's detail panel.
	Description string `json:"description,omitempty"`

	// EffectVerb names the verb-recipe invoked when the player
	// uses the item (e.g. "restore_health(50)"). Empty = passive
	// item (key, plot flag, etc.).
	EffectVerb string `json:"effect_verb,omitempty"`

	// Category is a free-form classifier the inventory menu can
	// filter on (potion, weapon, key, plot, etc.).
	Category string `json:"category,omitempty"`
}

// SaveConfig is the per-project save-pipeline knob. v1 carries the
// game-title override (used by the save path sanitiser) and the
// autosave-enabled flag; future versions add slot count / format /
// custom paths.
type SaveConfig struct {
	// GameTitle overrides the project's Name for save-path
	// derivation. Useful when the designer wants to ship under a
	// different brand than the project name. Empty falls back to
	// Project.Name.
	GameTitle string `json:"game_title,omitempty"`

	// AutosaveEnabled gates the runtime autosave path. Default
	// true (set by applyDefaults via the Set sentinel pattern).
	AutosaveEnabled bool `json:"autosave_enabled,omitempty"`

	// Set carries the "designer has authored this" intent so the
	// {false} state survives the omitempty boundary on save / load.
	// Mirrors EditorOverlays.Set (idea #3 v1 U1).
	Set bool `json:"set,omitempty"`
}

// DefaultSaveConfig returns the canonical SaveConfig new projects
// receive on first load (autosave enabled; no title override).
func DefaultSaveConfig() SaveConfig {
	return SaveConfig{
		AutosaveEnabled: true,
		Set:             true,
	}
}

// applyDefaults backfills new projects' save config so the runtime
// never sees a zero-value Set bit.
func (s *SaveConfig) applyDefaults() {
	if s == nil {
		return
	}
	if !s.Set {
		*s = DefaultSaveConfig()
	}
}
