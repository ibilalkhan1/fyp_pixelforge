// scene_settings.go owns the scene-level inspector panel for idea #1
// v1 U8 — the Scene workspace's "the scene itself is the selection
// target" surface. When the inspector dispatches to a scene rather
// than an entity, RenderSceneSettings emits the panel: scene name
// header, grid width / height spinners, default-tile picker (numeric
// for v1; the U6 palette UX can layer on later), camera config sub-
// panel, and read-only spawn-cell display.
//
// ResizeGrid is the pure-data helper the panel calls when the
// designer bumps GridWidthScreens / GridHeightScreens: it grows or
// shrinks each TilemapLayer.Grid while preserving existing content
// and flags entities that fall outside the new bounds. The "force"
// flag separates the dry-run-with-flagged-entities pass from the
// commit-after-confirm pass, so the shrink-confirm modal (driven by
// the existing confirm_modal.go) can sit between the two and let the
// designer cancel without mutating the scene.
package editor

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// EntityRef points to a single entity inside a scene, identified by
// its stable ID + the human-readable name the modal copy displays.
// ResizeGrid returns EntityRefs (rather than *Entity pointers) so the
// caller can show the list in a confirm modal without holding live
// scene-state pointers across the confirm/cancel window.
type EntityRef struct {
	ID   string
	Name string
}

// ResizeGrid grows or shrinks every Tilemap in scene to the new
// (newWidthScreens, newHeightScreens) bounds.
//
// Grow: each Tilemap.Grid is extended with new cells defaulting to
// scene.DefaultTileID. Existing painted cells stay at their original
// (row, col) coordinates so resize-preserves-content semantics hold
// (R3, AE5).
//
// Shrink: cells past the new bounds are truncated. Entities whose
// TileX / TileY exceed the new tile bounds are gathered into the
// returned outOfBoundsEntities list.
//
//   - When force is false AND any entity is out of bounds, ResizeGrid
//     returns the list WITHOUT mutating the scene. The caller raises
//     a confirm modal; on confirm, the caller re-invokes with
//     force=true.
//   - When force is true, out-of-bounds entities are clamped to the
//     last valid cell ((newCols-1, newRows-1)) and the grid is
//     truncated.
//
// Spawn cell always auto-clamps into the new bounds, regardless of
// force — the spawn is metadata, not a placed entity, and the loader
// already enforces this contract at load time (project.go's
// applyWorldDefaults).
//
// ResizeGrid does NOT call MarkDirty itself; callers do that after
// confirming the resize completed. Keeps the resize logic side-
// effect-free against the Editor singleton, which makes it trivially
// testable from a pure unit test.
func ResizeGrid(scene *pixelforge_project.Scene, newWidthScreens, newHeightScreens int, force bool) (outOfBoundsEntities []EntityRef, err error) {
	if scene == nil {
		return nil, fmt.Errorf("ResizeGrid: nil scene")
	}
	if newWidthScreens < 1 {
		return nil, fmt.Errorf("ResizeGrid: width must be >= 1, got %d", newWidthScreens)
	}
	if newHeightScreens < 1 {
		return nil, fmt.Errorf("ResizeGrid: height must be >= 1, got %d", newHeightScreens)
	}

	newCols := newWidthScreens * pixelforge_project.ScreenTilesWide
	newRows := newHeightScreens * pixelforge_project.ScreenTilesTall

	// Identify entities that would fall outside the new bounds.
	for i := range scene.Entities {
		ent := &scene.Entities[i]
		if ent.TileX >= newCols || ent.TileY >= newRows {
			outOfBoundsEntities = append(outOfBoundsEntities, EntityRef{
				ID:   ent.ID,
				Name: ent.Name,
			})
		}
	}

	// Shrink-without-force gate: caller must confirm before we mutate.
	if !force && len(outOfBoundsEntities) > 0 {
		return outOfBoundsEntities, nil
	}

	// Apply the new dimensions.
	scene.GridWidthScreens = newWidthScreens
	scene.GridHeightScreens = newHeightScreens

	// Resize each tilemap. Both grow and shrink go through the same
	// pure helper so the row/column conventions match exactly.
	for li := range scene.Tilemaps {
		layer := &scene.Tilemaps[li]
		layer.Grid = resizeLayerGrid(layer.Grid, newCols, newRows, scene.DefaultTileID)
	}

	// Clamp out-of-bounds entities to the last valid cell. Per the
	// plan: shrink with force collapses the out-of-bounds entities to
	// (newCols-1, newRows-1) rather than deleting them — designer can
	// re-place after the confirm.
	if force && len(outOfBoundsEntities) > 0 {
		maxCol := newCols - 1
		maxRow := newRows - 1
		for i := range scene.Entities {
			ent := &scene.Entities[i]
			if ent.TileX > maxCol {
				ent.TileX = maxCol
			}
			if ent.TileY > maxRow {
				ent.TileY = maxRow
			}
		}
	}

	// Spawn cell always clamps into the new bounds.
	maxCol := newCols - 1
	maxRow := newRows - 1
	if scene.SpawnTile.Col > maxCol {
		scene.SpawnTile.Col = maxCol
	}
	if scene.SpawnTile.Row > maxRow {
		scene.SpawnTile.Row = maxRow
	}
	if scene.SpawnTile.Col < 0 {
		scene.SpawnTile.Col = 0
	}
	if scene.SpawnTile.Row < 0 {
		scene.SpawnTile.Row = 0
	}

	return outOfBoundsEntities, nil
}

// resizeLayerGrid returns a fresh [][]int sized (newRows × newCols)
// with existing values copied from src at their original coords. New
// cells default to defaultTileID. Handles both grow (newRows/newCols
// >= original) and shrink (truncate) symmetrically; ragged source
// grids are tolerated.
//
// Allocating a fresh outer slice keeps the resize append-free at the
// call site and avoids accidental aliasing between the resized layer
// and any test fixture that still holds the original.
func resizeLayerGrid(src [][]int, newCols, newRows, defaultTileID int) [][]int {
	out := make([][]int, newRows)
	for r := 0; r < newRows; r++ {
		row := make([]int, newCols)
		if defaultTileID != 0 {
			for c := range row {
				row[c] = defaultTileID
			}
		}
		if r < len(src) {
			oldRow := src[r]
			cap := newCols
			if len(oldRow) < cap {
				cap = len(oldRow)
			}
			copy(row[:cap], oldRow[:cap])
		}
		out[r] = row
	}
	return out
}

// RenderSceneSettings emits the inspector panel for a scene that is
// the current selection target. Returns true when any widget mutated
// scene state this frame (caller flips the dirty flag).
//
// Live ImGui frame required: the function no-ops cleanly when the
// editor's imgui backend is absent / test-only so unit tests can
// drive the resize logic via ResizeGrid directly without standing up
// a live context.
//
// The shrink-with-out-of-bounds-entities flow routes through the
// editor's existing confirm_modal.go: when the spinner edit would
// truncate entities, RenderSceneSettings stages the desired (w, h) on
// the editor and pops the confirm modal; on confirm the resize re-
// runs with force=true and the dirty flag flips. This keeps the
// scene mutation atomic — designer cancels => zero churn.
func RenderSceneSettings(scene *pixelforge_project.Scene, e *Editor) bool {
	if scene == nil || e == nil {
		return false
	}
	if e.imgui == nil || !e.imgui.live {
		return false
	}
	mutated := false

	imgui.Text("Scene: " + scene.Name)
	imgui.Separator()

	// ---- Grid size spinners ----------------------------------------
	w := int32(scene.GridWidthScreens)
	if imgui.InputIntV("Grid Width (screens)", &w, 1, 1, 0) {
		newW := int(w)
		if newW < 1 {
			newW = 1
		}
		if newW != scene.GridWidthScreens {
			if applyResize(scene, e, newW, scene.GridHeightScreens) {
				mutated = true
			}
		}
	}

	h := int32(scene.GridHeightScreens)
	if imgui.InputIntV("Grid Height (screens)", &h, 1, 1, 0) {
		newH := int(h)
		if newH < 1 {
			newH = 1
		}
		if newH != scene.GridHeightScreens {
			if applyResize(scene, e, scene.GridWidthScreens, newH) {
				mutated = true
			}
		}
	}

	// ---- Default tile picker --------------------------------------
	// v1 ships the simpler numeric picker. The U6 palette UX (which
	// shows sprite-sheet thumbnails) could be wired here later — the
	// plan flags either as acceptable; numeric is the minimum that
	// proves the schema field round-trips and the new-cell-fill works.
	dt := int32(scene.DefaultTileID)
	if imgui.InputIntV("Default Tile ID", &dt, 1, 1, 0) {
		newDT := int(dt)
		if newDT < 0 {
			newDT = 0
		}
		if newDT != scene.DefaultTileID {
			scene.DefaultTileID = newDT
			e.MarkDirty()
			mutated = true
		}
	}

	// ---- Camera config sub-panel (collapsed by default) -----------
	if imgui.CollapsingHeaderTreeNodeFlagsV("Camera", 0) {
		if renderCameraConfig(&scene.Camera) {
			e.MarkDirty()
			mutated = true
		}
	}

	// ---- Spawn cell (read-only — set via right-click in U6) -------
	imgui.Separator()
	imgui.Text(fmt.Sprintf("Spawn tile: col %d, row %d (set via right-click in scene)",
		scene.SpawnTile.Col, scene.SpawnTile.Row))

	return mutated
}

// applyResize is the shared spinner-commit path. It runs ResizeGrid
// in dry-run mode first; if entities would fall out of bounds, it
// raises the confirm modal whose OnConfirm closure re-runs the
// resize with force=true. Returns true when the resize completed
// synchronously (grow paths and shrink paths with zero out-of-bounds
// entities); false when the resize was deferred behind the modal.
//
// Either way the caller's dirty flag flips: synchronous mutations
// flip immediately, deferred mutations flip inside the modal's
// OnConfirm. Keeps "did anything change this frame" honest.
func applyResize(scene *pixelforge_project.Scene, e *Editor, newW, newH int) bool {
	flagged, err := ResizeGrid(scene, newW, newH, false)
	if err != nil {
		return false
	}
	if len(flagged) == 0 {
		// Grow path, or shrink with no flagged entities — resize
		// already committed.
		e.MarkDirty()
		return true
	}
	// Shrink path with entities to displace. Defer the actual mutation
	// behind the confirm modal so the designer can cancel.
	if e.confirmDialog == nil {
		// No modal available — fail safe by not mutating.
		return false
	}
	msg := fmt.Sprintf("%d entit%s will be moved into the new bounds.",
		len(flagged), plural(len(flagged)))
	e.confirmDialog.Show("Shrink scene?", msg, func() {
		if _, ferr := ResizeGrid(scene, newW, newH, true); ferr == nil {
			e.MarkDirty()
		}
	})
	return false
}

// plural returns "y" / "ies" suffix selector for the modal copy.
func plural(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

// renderCameraConfig emits the five SceneCameraConfig knobs as
// InputFloat rows. Returns true when any value changed this frame.
// Designers tweak these to override the NES-class defaults the loader
// applies (per-field granularity is preserved — zero-value fields
// re-default to the documented constants on next load).
func renderCameraConfig(cam *pixelforge_project.SceneCameraConfig) bool {
	mutated := false

	dzw := float32(cam.DeadZoneW)
	if imgui.InputFloatV("Dead Zone W", &dzw, 0, 0, "%.2f", 0) {
		cam.DeadZoneW = float64(dzw)
		mutated = true
	}
	dzh := float32(cam.DeadZoneH)
	if imgui.InputFloatV("Dead Zone H", &dzh, 0, 0, "%.2f", 0) {
		cam.DeadZoneH = float64(dzh)
		mutated = true
	}
	la := float32(cam.LookAheadX)
	if imgui.InputFloatV("Look Ahead X", &la, 0, 0, "%.2f", 0) {
		cam.LookAheadX = float64(la)
		mutated = true
	}
	sx := float32(cam.SmoothingX)
	if imgui.InputFloatV("Smoothing X", &sx, 0, 0, "%.4f", 0) {
		cam.SmoothingX = float64(sx)
		mutated = true
	}
	sy := float32(cam.SmoothingY)
	if imgui.InputFloatV("Smoothing Y", &sy, 0, 0, "%.4f", 0) {
		cam.SmoothingY = float64(sy)
		mutated = true
	}
	return mutated
}
