package editor

// right_click_menu.go owns idea #1 v1's right-click dispatch on the
// Scene preview. The canvas's UpdateAt switch routes RMB events into
// HandleRightClick with a RightClickContext describing what was
// clicked; this file decides which actions are offered and applies
// them on selection.
//
// v1 ships exactly one action ("Set spawn here" for empty-cell
// clicks), but the dispatch is structured so U7 can add entity-
// specific actions ("Set as Player" / "Unmark as Player") without
// reshaping the call site:
//
//   - RightClickContext carries both Cell and Entity fields. Exactly
//     one is meaningful per click; handlers dispatch by checking
//     which is non-nil/empty.
//   - HandleRightClick is the single seam the canvas invokes. It
//     mutates state directly (and MarkDirty) when the action is
//     unambiguous; the live ImGui popup path is a thin wrapper that
//     this same function powers (the popup writes the same handler
//     for its menu item callback).
//
// Tests drive HandleRightClick + the per-action helpers directly so
// the right-click flow is verifiable without standing up an ImGui
// context.

import (
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// RightClickContext describes what the user right-clicked on. v1
// recognizes two cases:
//
//   - Cell != nil: the click landed on an empty tile cell. The "Set
//     spawn here" action is offered (and is the only v1 action).
//   - EntityID != "": the click hit an existing entity. U7 will
//     extend with "Set as Player" / "Unmark as Player"; v1 leaves
//     the case wired but with no actions to keep the dispatch
//     structurally complete.
//
// Exactly one field should be set per click. The picking logic in
// canvas.go enforces this (entity hit-test runs first, fall through
// to cell on miss).
type RightClickContext struct {
	Scene    *pixelforge_project.Scene
	Cell     *RightClickCell
	EntityID string
}

// RightClickCell carries the (Col, Row) of the cell that was
// right-clicked. Held by pointer so RightClickContext.Cell == nil
// unambiguously means "this click did not target a cell".
type RightClickCell struct {
	Col int
	Row int
}

// HandleRightClick is the single entry point the canvas dispatch
// invokes. For v1 it implements one path:
//
//   - Cell click → mutates ctx.Scene.SpawnTile to the clicked cell
//     and marks the editor dirty.
//
// Returns true when an action fired (used by the canvas to suppress
// the live ImGui popup when there's nothing to offer). When U7 adds
// entity actions the entity branch will return true for handled
// clicks; for v1 it returns false because the menu is empty.
//
// The function intentionally does not open an ImGui popup itself —
// the popup is a workspace-rendered concern (it needs the screen-
// space position of the cursor at click time, which only the
// workspace knows). Instead, HandleRightClick is the action-layer
// the popup's menu items invoke; tests bypass the popup and call
// HandleRightClick directly.
func HandleRightClick(e *Editor, ctx RightClickContext) bool {
	if e == nil || ctx.Scene == nil {
		return false
	}
	switch {
	case ctx.EntityID != "":
		// U7 adds the Player-tagging actions on entity right-click.
		// "Set as Player" applies mutual-exclusivity (only one Player
		// per scene); "Unmark as Player" reverts the tag to World.
		// The two menu items are mutually exclusive by current state
		// — the workspace popup picks which one to surface based on
		// the clicked entity's current Archetype, and the action
		// helpers below are the single seam that mutates state.
		ent := findEntityByID(ctx.Scene, ctx.EntityID)
		if ent == nil {
			return false
		}
		if ent.Archetype == pixelforge_project.ArchetypePlayer {
			return unmarkAsPlayer(e, ent)
		}
		return setAsPlayer(e, ctx.Scene, ent)
	case ctx.Cell != nil:
		return setSpawnHere(e, ctx.Scene, ctx.Cell.Col, ctx.Cell.Row)
	}
	return false
}

// findEntityByID returns the entity in scene with the supplied ID, or
// nil when no match. Kept package-private; the right-click dispatch is
// the only call site.
func findEntityByID(scene *pixelforge_project.Scene, id string) *pixelforge_project.Entity {
	if scene == nil || id == "" {
		return nil
	}
	for i := range scene.Entities {
		if scene.Entities[i].ID == id {
			return &scene.Entities[i]
		}
	}
	return nil
}

// setAsPlayer marks ent as the scene's Player. The invariant: exactly
// one entity per scene carries Archetype == ArchetypePlayer at any
// time. Before flipping ent, every other entity currently tagged
// Player has its Archetype demoted to ArchetypeWorld. Returns true and
// marks the editor dirty on success (which is unconditional — an
// already-Player ent reaches unmarkAsPlayer above, never this path).
//
// Only the Archetype field is mutated. The entity's Components,
// VerbBindings, TileX / TileY, Position, and any other fields are
// preserved — designers expect "Set as Player" to be a tag operation,
// not a destructive reset.
func setAsPlayer(e *Editor, scene *pixelforge_project.Scene, ent *pixelforge_project.Entity) bool {
	if e == nil || scene == nil || ent == nil {
		return false
	}
	for i := range scene.Entities {
		if &scene.Entities[i] == ent {
			continue
		}
		if scene.Entities[i].Archetype == pixelforge_project.ArchetypePlayer {
			scene.Entities[i].Archetype = pixelforge_project.ArchetypeWorld
		}
	}
	ent.Archetype = pixelforge_project.ArchetypePlayer
	e.MarkDirty()
	e.SetStatusMessage("Player set")
	return true
}

// unmarkAsPlayer reverts ent's Archetype to ArchetypeWorld. Used when
// the designer right-clicks the currently-tagged Player and picks
// "Unmark as Player". Other entities are untouched; after this the
// scene has zero Players (a legal state — the preview's camera
// follower silently no-ops when no Player is present).
func unmarkAsPlayer(e *Editor, ent *pixelforge_project.Entity) bool {
	if e == nil || ent == nil {
		return false
	}
	ent.Archetype = pixelforge_project.ArchetypeWorld
	e.MarkDirty()
	e.SetStatusMessage("Player cleared")
	return true
}

// setSpawnHere is the cell-click action handler. Writes SpawnTile,
// marks dirty, posts a status message so the designer sees the
// effect (the spawn tile has no visual representation in v1).
// Exported indirectly via HandleRightClick; kept package-private so
// the right-click dispatch is the single seam.
func setSpawnHere(e *Editor, scene *pixelforge_project.Scene, col, row int) bool {
	if e == nil || scene == nil {
		return false
	}
	// Clamp to non-negative; the workspace already clamps to the
	// grid's upper bound via canvas.windowToScene, so we only need
	// to guard the negative-edge case here.
	if col < 0 {
		col = 0
	}
	if row < 0 {
		row = 0
	}
	scene.SpawnTile = pixelforge_project.SpawnCell{Col: col, Row: row}
	e.MarkDirty()
	e.SetStatusMessage("Spawn set")
	return true
}
