// scene_settings_test.go pins idea #1 v1 U8's contract: ResizeGrid's
// grow-preserves-content, shrink-truncates-with-flagging,
// shrink-clamps-spawn, and dirty-flag semantics. Plus the inspector
// dispatch path (scene selection routes to RenderSceneSettings without
// disturbing the entity selection branch).
//
// The ImGui-side rendering itself is exercised by the live editor —
// the tests here drive the seams that survive without a live frame.
package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// makeScene builds a small scene fixture with one tilemap. Callers
// can mutate the grid before invoking ResizeGrid to test grow/shrink
// behaviours.
func makeScene(widthScreens, heightScreens int) *pixelforge_project.Scene {
	cols := widthScreens * pixelforge_project.ScreenTilesWide
	rows := heightScreens * pixelforge_project.ScreenTilesTall
	grid := make([][]int, rows)
	for r := 0; r < rows; r++ {
		grid[r] = make([]int, cols)
	}
	return &pixelforge_project.Scene{
		ID:                "main",
		Name:              "Main",
		GridWidthScreens:  widthScreens,
		GridHeightScreens: heightScreens,
		TileAtlases: []pixelforge_project.TileAtlas{
			{
				Name:  "ground",
				TileW: 8, TileH: 8,
				Grid: grid,
			},
		},
	}
}

// TestResize_Grow_PreservesContent: painted cells survive a width
// expansion at their original (row, col) coordinates; the new area
// fills with DefaultTileID (0 when unset).
func TestResize_Grow_PreservesContent(t *testing.T) {
	scene := makeScene(16, 1)
	scene.TileAtlases[0].Grid[10][5] = 3 // paint a marker cell

	flagged, err := ResizeGrid(scene, 24, 1, false)
	require.NoError(t, err)
	assert.Empty(t, flagged, "grow has no out-of-bounds entities")

	assert.Equal(t, 24, scene.GridWidthScreens)
	require.Equal(t, 24*pixelforge_project.ScreenTilesWide,
		len(scene.TileAtlases[0].Grid[10]),
		"row 10 widened to the new column count")
	assert.Equal(t, 3, scene.TileAtlases[0].Grid[10][5],
		"painted cell at (5, 10) preserved across grow")
	assert.Equal(t, 0, scene.TileAtlases[0].Grid[10][600],
		"new area defaults to DefaultTileID=0 when unset")
}

// TestResize_Grow_NewAreaDefaultTile: with DefaultTileID=2, new
// columns added by a grow fill with 2.
func TestResize_Grow_NewAreaDefaultTile(t *testing.T) {
	scene := makeScene(16, 1)
	scene.DefaultTileID = 2

	_, err := ResizeGrid(scene, 24, 1, false)
	require.NoError(t, err)

	// Every cell of column 600 (well past the original 16*32=512
	// boundary) should equal the default tile ID.
	for r := 0; r < pixelforge_project.ScreenTilesTall; r++ {
		assert.Equal(t, 2, scene.TileAtlases[0].Grid[r][600],
			"row %d new column 600 fills with DefaultTileID", r)
	}
	// Pre-existing cells (column 5) retain their zero baseline.
	assert.Equal(t, 0, scene.TileAtlases[0].Grid[0][5],
		"existing-area cells stay at their original value")
}

// TestResize_Shrink_TruncatesGrid: a grid extending to col 511
// (16 screens × 32 = 512) shrinks cleanly to col 255 (8 screens × 32
// = 256) without panic. Asserts the column count is the new bound.
func TestResize_Shrink_TruncatesGrid(t *testing.T) {
	scene := makeScene(16, 1)
	scene.TileAtlases[0].Grid[0][511] = 9 // marker at the far right

	_, err := ResizeGrid(scene, 8, 1, false)
	require.NoError(t, err)

	assert.Equal(t, 8, scene.GridWidthScreens)
	assert.Equal(t, 8*pixelforge_project.ScreenTilesWide,
		len(scene.TileAtlases[0].Grid[0]),
		"row width truncated to 8 screens worth of columns")
}

// TestResize_Shrink_OutOfBoundsEntitiesFlagged: shrinking past an
// entity's tile coord returns the entity in the flagged list AND does
// NOT mutate the scene (force=false dry-run).
func TestResize_Shrink_OutOfBoundsEntitiesFlagged(t *testing.T) {
	scene := makeScene(16, 1)
	scene.Entities = []pixelforge_project.Entity{
		{ID: "hero", Name: "Hero", TileX: 4, TileY: 24},
		{ID: "far", Name: "Far", TileX: 400, TileY: 24},
	}

	flagged, err := ResizeGrid(scene, 8, 1, false)
	require.NoError(t, err)
	require.Len(t, flagged, 1, "exactly one entity is out of bounds")
	assert.Equal(t, "far", flagged[0].ID)
	assert.Equal(t, "Far", flagged[0].Name)

	// Force=false with flagged entities must leave the scene untouched.
	assert.Equal(t, 16, scene.GridWidthScreens,
		"dry-run shrink with flagged entities must not mutate scene")
	assert.Equal(t, 400, scene.Entities[1].TileX,
		"flagged entity tile coord preserved on dry-run")
	assert.Equal(t, 16*pixelforge_project.ScreenTilesWide,
		len(scene.TileAtlases[0].Grid[0]),
		"grid not truncated on dry-run")
}

// TestResize_Shrink_ForceClampsEntities: force=true commits the
// shrink and collapses each out-of-bounds entity to the last valid
// cell. Designer can re-place after the modal confirms.
func TestResize_Shrink_ForceClampsEntities(t *testing.T) {
	scene := makeScene(16, 1)
	scene.Entities = []pixelforge_project.Entity{
		{ID: "hero", Name: "Hero", TileX: 4, TileY: 24},
		{ID: "far", Name: "Far", TileX: 400, TileY: 25},
		{ID: "ok", Name: "OK", TileX: 200, TileY: 24},
	}

	flagged, err := ResizeGrid(scene, 8, 1, true)
	require.NoError(t, err)
	require.Len(t, flagged, 1, "only 'far' was out of bounds")

	assert.Equal(t, 8, scene.GridWidthScreens)
	assert.Equal(t, 4, scene.Entities[0].TileX, "in-bounds 'hero' untouched")
	maxCol := 8*pixelforge_project.ScreenTilesWide - 1
	assert.Equal(t, maxCol, scene.Entities[1].TileX,
		"'far' clamped to last valid column")
	assert.Equal(t, 25, scene.Entities[1].TileY,
		"'far' row preserved (within bounds)")
	assert.Equal(t, 200, scene.Entities[2].TileX,
		"in-bounds 'ok' untouched")
}

// TestResize_Shrink_ClampsSpawnTile: SpawnTile beyond the new
// horizontal bound clamps to the last valid column regardless of
// force. Spawn is metadata, not an entity.
func TestResize_Shrink_ClampsSpawnTile(t *testing.T) {
	scene := makeScene(16, 1)
	scene.SpawnTile = pixelforge_project.SpawnCell{Col: 300, Row: 0}

	_, err := ResizeGrid(scene, 8, 1, false)
	require.NoError(t, err)

	maxCol := 8*pixelforge_project.ScreenTilesWide - 1
	assert.Equal(t, maxCol, scene.SpawnTile.Col,
		"spawn col clamped to last valid index on shrink")
	assert.Equal(t, 0, scene.SpawnTile.Row)
}

// TestResize_MarksDirty: a successful synchronous resize (grow,
// or shrink with no flagged entities) flips the editor's dirty flag.
func TestResize_MarksDirty(t *testing.T) {
	e := New()
	p := pixelforge_project.NewProject("t")
	p.Scenes[0] = *makeScene(16, 1)
	e.SetProject(p)
	assert.False(t, e.IsDirty(), "fresh project is clean")

	scene := &e.project.Scenes[0]
	ok := applyResize(scene, e, 24, 1)
	assert.True(t, ok, "grow commits synchronously")
	assert.True(t, e.IsDirty(), "grow flips dirty flag")
	assert.Equal(t, 24, scene.GridWidthScreens)
}

// TestResize_ShrinkWithFlaggedEntities_DeferredViaModal: when the
// shrink would displace entities, applyResize raises the confirm
// modal rather than mutating immediately; on Confirm the resize
// completes with force=true and dirty flips.
func TestResize_ShrinkWithFlaggedEntities_DeferredViaModal(t *testing.T) {
	e := New()
	p := pixelforge_project.NewProject("t")
	p.Scenes[0] = *makeScene(16, 1)
	p.Scenes[0].Entities = []pixelforge_project.Entity{
		{ID: "far", Name: "Far", TileX: 400, TileY: 24},
	}
	e.SetProject(p)

	scene := &e.project.Scenes[0]
	committed := applyResize(scene, e, 8, 1)
	assert.False(t, committed, "shrink with flagged entities defers behind modal")
	assert.False(t, e.IsDirty(), "deferred resize must not flip dirty yet")
	assert.True(t, e.confirmDialog.Visible(),
		"confirm modal raised for the shrink-with-entities path")
	assert.Equal(t, 16, scene.GridWidthScreens,
		"scene unchanged until designer confirms")

	// Simulate designer pressing Confirm.
	e.confirmDialog.Confirm()
	assert.Equal(t, 8, scene.GridWidthScreens,
		"confirm commits the shrink with force=true")
	maxCol := 8*pixelforge_project.ScreenTilesWide - 1
	assert.Equal(t, maxCol, scene.Entities[0].TileX,
		"flagged entity clamped on commit")
	assert.True(t, e.IsDirty(), "confirm flips dirty flag")
}

// TestResize_ShrinkCancelLeavesSceneUntouched: Cancel on the confirm
// modal aborts the resize entirely — scene + dirty flag both unchanged.
func TestResize_ShrinkCancelLeavesSceneUntouched(t *testing.T) {
	e := New()
	p := pixelforge_project.NewProject("t")
	p.Scenes[0] = *makeScene(16, 1)
	p.Scenes[0].Entities = []pixelforge_project.Entity{
		{ID: "far", Name: "Far", TileX: 400, TileY: 24},
	}
	e.SetProject(p)

	scene := &e.project.Scenes[0]
	applyResize(scene, e, 8, 1)
	require.True(t, e.confirmDialog.Visible())

	e.confirmDialog.Cancel()
	assert.Equal(t, 16, scene.GridWidthScreens,
		"cancel leaves the scene unchanged")
	assert.False(t, e.IsDirty(),
		"cancel does not flip the dirty flag")
	assert.Equal(t, 400, scene.Entities[0].TileX,
		"flagged entity untouched on cancel")
}

// TestResize_NilSceneReturnsError: defensive guard — ResizeGrid on a
// nil scene returns an error rather than panicking.
func TestResize_NilSceneReturnsError(t *testing.T) {
	_, err := ResizeGrid(nil, 8, 1, false)
	assert.Error(t, err)
}

// TestResize_InvalidDimensionsReturnError: width or height < 1
// rejects the resize without mutating any state.
func TestResize_InvalidDimensionsReturnError(t *testing.T) {
	scene := makeScene(16, 1)
	_, err := ResizeGrid(scene, 0, 1, false)
	assert.Error(t, err)
	_, err = ResizeGrid(scene, 16, -1, false)
	assert.Error(t, err)
	assert.Equal(t, 16, scene.GridWidthScreens, "scene untouched on invalid input")
}

// TestSceneSettings_RendersOnSceneSelection: with no entity selected
// and a scene-selection set, the editor.selectedScene() resolves to
// the named scene; with both cleared it returns nil. This is the
// selection-axis seam the inspector's dispatch reads.
//
// (The live ImGui rendering itself is covered by the manual smoke
// verification in the plan; here we pin the selection model that
// drives whether RenderSceneSettings is called at all.)
func TestSceneSettings_RendersOnSceneSelection(t *testing.T) {
	e := New()
	p := pixelforge_project.NewProject("t")
	p.Scenes[0].ID = "main"
	p.Scenes[0].Name = "Main"
	e.SetProject(p)

	// Default: no scene selection, no entity selection.
	assert.Nil(t, e.selectedScene(), "no scene selected by default")
	assert.Nil(t, e.selectedEntity(), "no entity selected by default")

	// Selecting a scene resolves it.
	e.SelectScene("main")
	got := e.selectedScene()
	require.NotNil(t, got, "selectedScene resolves the selected ID")
	assert.Equal(t, "main", got.ID)

	// Clearing the selection collapses back to nil.
	e.SelectScene("")
	assert.Nil(t, e.selectedScene(),
		"empty selection returns nil — inspector falls back to '(no selection)'")

	// Unknown ID returns nil.
	e.SelectScene("ghost")
	assert.Nil(t, e.selectedScene(),
		"unknown scene id returns nil")
}

// TestSceneSettings_EntitySelectionTakesPrecedence: when both an
// entity AND a scene are selected, the inspector's dispatch must
// prefer the entity (character-sheet wins; scene settings stay
// hidden until designer deselects the entity).
func TestSceneSettings_EntitySelectionTakesPrecedence(t *testing.T) {
	e := New()
	p := pixelforge_project.NewProject("t")
	p.Scenes[0].ID = "main"
	p.Scenes[0].Entities = []pixelforge_project.Entity{
		{ID: "hero", Name: "Hero"},
	}
	e.SetProject(p)

	e.SelectScene("main")
	e.SelectEntity("hero")

	require.NotNil(t, e.selectedEntity(),
		"entity selection is the inspector's preferred branch")
	require.NotNil(t, e.selectedScene(),
		"scene selection coexists but the inspector dispatches to entity first")
}

// TestSceneSettings_SetProjectClearsSceneSelection: replacing the
// project clears the scene selection (parallel to entity selection)
// so the inspector doesn't render a stale pointer.
func TestSceneSettings_SetProjectClearsSceneSelection(t *testing.T) {
	e := New()
	p1 := pixelforge_project.NewProject("p1")
	p1.Scenes[0].ID = "main"
	e.SetProject(p1)
	e.SelectScene("main")
	require.NotNil(t, e.selectedScene())

	p2 := pixelforge_project.NewProject("p2")
	e.SetProject(p2)
	assert.Empty(t, e.SelectedSceneID(),
		"SetProject clears scene selection")
	assert.Nil(t, e.selectedScene())
}

// TestResizeLayerGrid_RaggedSourceTolerated: pure helper guard — a
// ragged input grid (rows of differing widths, mimicking a malformed
// pre-v1 file) resizes cleanly without panic. Each row is padded /
// truncated independently to newCols.
func TestResizeLayerGrid_RaggedSourceTolerated(t *testing.T) {
	src := [][]int{
		{1, 2, 3},
		{4, 5}, // short row
		{6, 7, 8, 9, 10}, // long row
	}
	out := resizeLayerGrid(src, 4, 3, 0)
	require.Len(t, out, 3)
	for _, row := range out {
		assert.Len(t, row, 4, "every row padded/truncated to newCols")
	}
	assert.Equal(t, 1, out[0][0])
	assert.Equal(t, 0, out[0][3], "short source padded with default")
	assert.Equal(t, 4, out[1][0])
	assert.Equal(t, 0, out[1][2], "short source row padded with default")
	assert.Equal(t, 6, out[2][0])
	assert.Equal(t, 9, out[2][3])
	// long row's index 4 should be truncated away.
}

// TestResizeLayerGrid_FillsNewRowsWithDefault: rows added on a height
// grow fill entirely with defaultTileID.
func TestResizeLayerGrid_FillsNewRowsWithDefault(t *testing.T) {
	src := [][]int{
		{1, 1, 1},
		{1, 1, 1},
	}
	out := resizeLayerGrid(src, 3, 4, 7)
	require.Len(t, out, 4)
	for c := 0; c < 3; c++ {
		assert.Equal(t, 7, out[2][c], "new row %d fills with default 7", 2)
		assert.Equal(t, 7, out[3][c], "new row %d fills with default 7", 3)
		assert.Equal(t, 1, out[0][c], "existing row preserved")
	}
}
