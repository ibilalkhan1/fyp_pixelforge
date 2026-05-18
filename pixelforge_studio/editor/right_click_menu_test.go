// right_click_menu_test.go covers U7's entity-branch additions to the
// right-click dispatch — "Set as Player" / "Unmark as Player" — and
// the mutual-exclusivity invariant they enforce ("exactly one Player
// per scene"). Tests drive HandleRightClick directly to bypass the
// live ImGui popup and exercise the action-layer in isolation.
package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// TestSetAsPlayer_MarksArchetype — an entity whose Archetype is
// something other than Player gets flipped to Player; the editor's
// dirty flag fires; the action returns true.
func TestSetAsPlayer_MarksArchetype(t *testing.T) {
	e := New()
	scene := e.activeScene()
	require.NotNil(t, scene)
	scene.Entities = []pixelforge_project.Entity{{
		ID:        "goomba",
		Archetype: pixelforge_project.ArchetypeEnemy,
		TileX:     10, TileY: 24,
	}}
	e.ClearDirty()

	handled := HandleRightClick(e, RightClickContext{
		Scene:    scene,
		EntityID: "goomba",
	})

	assert.True(t, handled)
	assert.Equal(t, pixelforge_project.ArchetypePlayer, scene.Entities[0].Archetype)
	assert.True(t, e.IsDirty(), "Set as Player must mark editor dirty")
}

// TestSetAsPlayer_MutualExclusivity — at most one entity carries the
// Player tag. When a non-Player entity is promoted, every other
// previously-Player entity is demoted to World in the same call.
func TestSetAsPlayer_MutualExclusivity(t *testing.T) {
	e := New()
	scene := e.activeScene()
	require.NotNil(t, scene)
	scene.Entities = []pixelforge_project.Entity{
		{ID: "mario", Archetype: pixelforge_project.ArchetypePlayer, TileX: 4, TileY: 24},
		{ID: "goomba", Archetype: pixelforge_project.ArchetypeEnemy, TileX: 10, TileY: 24},
	}

	handled := HandleRightClick(e, RightClickContext{
		Scene:    scene,
		EntityID: "goomba",
	})

	assert.True(t, handled)
	// Goomba is now the Player; Mario was demoted to World.
	assert.Equal(t, pixelforge_project.ArchetypeWorld, scene.Entities[0].Archetype, "previous Player demoted to World")
	assert.Equal(t, pixelforge_project.ArchetypePlayer, scene.Entities[1].Archetype, "clicked entity becomes Player")

	// Belt-and-braces: assert exactly one Player exists.
	playerCount := 0
	for _, ent := range scene.Entities {
		if ent.Archetype == pixelforge_project.ArchetypePlayer {
			playerCount++
		}
	}
	assert.Equal(t, 1, playerCount, "exactly one Player per scene by construction")
}

// TestSetAsPlayer_UnmarkRevertsToWorld — right-clicking the currently
// tagged Player invokes the "Unmark as Player" branch and reverts the
// Archetype to World. The dirty flag still fires.
func TestSetAsPlayer_UnmarkRevertsToWorld(t *testing.T) {
	e := New()
	scene := e.activeScene()
	require.NotNil(t, scene)
	scene.Entities = []pixelforge_project.Entity{{
		ID:        "mario",
		Archetype: pixelforge_project.ArchetypePlayer,
		TileX:     4, TileY: 24,
	}}
	e.ClearDirty()

	handled := HandleRightClick(e, RightClickContext{
		Scene:    scene,
		EntityID: "mario",
	})

	assert.True(t, handled)
	assert.Equal(t, pixelforge_project.ArchetypeWorld, scene.Entities[0].Archetype)
	assert.True(t, e.IsDirty())
}

// TestSetAsPlayer_PreservesExistingComponents — "Set as Player" is a
// pure tag operation: only Archetype mutates. Components, position,
// tile coords, and name survive untouched. Belt-and-braces against a
// regression where a future refactor might re-template the entity.
func TestSetAsPlayer_PreservesExistingComponents(t *testing.T) {
	e := New()
	scene := e.activeScene()
	require.NotNil(t, scene)
	scene.Entities = []pixelforge_project.Entity{{
		ID:        "goomba",
		Name:      "Goomba",
		Archetype: pixelforge_project.ArchetypeEnemy,
		TileX:     10, TileY: 24,
		Position: pixelforge_project.EntityPosition{X: 80, Y: 192, Z: 1},
		Components: []pixelforge_project.EntityComponent{
			{Type: "Sprite", Values: map[string]any{"sprite": "goomba"}},
			{Type: "Walker", Values: map[string]any{"speed": 1.0}},
		},
	}}

	handled := HandleRightClick(e, RightClickContext{
		Scene:    scene,
		EntityID: "goomba",
	})

	require.True(t, handled)
	ent := scene.Entities[0]
	assert.Equal(t, pixelforge_project.ArchetypePlayer, ent.Archetype)
	assert.Equal(t, "Goomba", ent.Name)
	assert.Equal(t, 10, ent.TileX)
	assert.Equal(t, 24, ent.TileY)
	assert.Equal(t, float64(80), ent.Position.X)
	assert.Equal(t, float64(192), ent.Position.Y)
	assert.Equal(t, 1, ent.Position.Z)
	require.Len(t, ent.Components, 2)
	assert.Equal(t, "Sprite", ent.Components[0].Type)
	assert.Equal(t, "Walker", ent.Components[1].Type)
	assert.Equal(t, 1.0, ent.Components[1].Values["speed"])
}

// TestHandleRightClick_UnknownEntityIDIsNoOp — an EntityID that does
// not match any entity in the scene returns false and mutates
// nothing. Defensive against stale IDs in the dispatch context.
func TestHandleRightClick_UnknownEntityIDIsNoOp(t *testing.T) {
	e := New()
	scene := e.activeScene()
	require.NotNil(t, scene)
	scene.Entities = []pixelforge_project.Entity{{
		ID: "mario", Archetype: pixelforge_project.ArchetypePlayer,
	}}
	e.ClearDirty()

	handled := HandleRightClick(e, RightClickContext{
		Scene:    scene,
		EntityID: "nonexistent",
	})

	assert.False(t, handled)
	assert.Equal(t, pixelforge_project.ArchetypePlayer, scene.Entities[0].Archetype)
	assert.False(t, e.IsDirty())
}
