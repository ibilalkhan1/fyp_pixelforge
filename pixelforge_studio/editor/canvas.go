package editor

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_entity"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// Canvas is the central scene viewport. It renders entities at their
// schema-declared positions and dispatches mouse input to the active
// tool.
type Canvas struct {
	// dragging holds the entity ID actively being moved by the Select
	// tool. Empty when no drag is in progress.
	dragging   string
	dragOffX   float64
	dragOffY   float64

	// nextEntitySeq counts entities created in this editor session so
	// the Place tool can mint stable, unique IDs without a UUID dep.
	nextEntitySeq int
}

// NewCanvas returns a fresh canvas.
func NewCanvas() *Canvas { return &Canvas{} }

// Draw paints the canvas into area. project + scene supply the entities
// to render; entity selection is read from the editor.
func (c *Canvas) Draw(dst *ebiten.Image, area widgets.Rect, e *Editor) {
	if area.W <= 0 || area.H <= 0 {
		return
	}
	vectorFill(dst, area, colCanvasBgInner)
	scene := e.activeScene()
	if scene == nil {
		debugPrint(dst, "(no scene)", area.X+8, area.Y+8)
		return
	}
	viewBox := c.viewBox(area, e.project)
	vectorFill(dst, viewBox, colCanvasViewBox)

	scaleX, scaleY := c.scale(viewBox, e.project)

	for i := range scene.Entities {
		ent := &scene.Entities[i]
		marker := c.entityMarkerRect(viewBox, ent, scaleX, scaleY)
		vectorFill(dst, marker, colCanvasEntity)
		if ent.ID == e.selectedEntityID {
			// 1px white outline around selection.
			strokeWidgetsRect(dst, widgets.Rect{
				X: marker.X - 1, Y: marker.Y - 1,
				W: marker.W + 2, H: marker.H + 2,
			}, color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
		}
	}
}

// Update handles mouse input using the global ebiten cursor. Kept
// for backwards compatibility; new callers should use UpdateAt to
// supply pre-mapped mouse coords (the U5 scene-as-texture flow needs
// to translate window-pixel coords into texture-pixel coords before
// dispatch).
func (c *Canvas) Update(area widgets.Rect, e *Editor) {
	mx, my := ebiten.CursorPosition()
	c.UpdateAt(area, mx, my, e)
}

// UpdateAt is the explicit-mouse-coord entry point Canvas.Update
// wraps. SceneWorkspace.Update calls this with mouse coords mapped
// through mapMouseToSceneTexture so the canvas's existing viewBox
// math operates on texture-space pixels — the same space the scene
// is rendered into.
//
// Idea #1 v1 adds two dispatch branches:
//   - ToolPaint routes LMB events through the per-stroke painter
//     state machine (brush start/paint/end; bucket single-shot;
//     rectangle anchor/fill).
//   - RMB on any tool routes into HandleRightClick with a
//     RightClickContext describing what was hit (entity vs empty
//     cell). v1 only acts on empty-cell clicks ("Set spawn here");
//     U7 will add entity actions to the same dispatch.
func (c *Canvas) UpdateAt(area widgets.Rect, mx, my int, e *Editor) {
	if area.W <= 0 || area.H <= 0 {
		return
	}
	scene := e.activeScene()
	if scene == nil {
		return
	}
	viewBox := c.viewBox(area, e.project)

	// Right-click is independent of the active tool — designers
	// expect "Set spawn here" to work whether they're in Select,
	// Place, Delete, or Paint mode.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) && viewBox.Contains(mx, my) {
		c.handleRightClick(scene, viewBox, e, mx, my)
	}

	if e.tool == ToolSelect {
		c.updateSelectTool(scene, viewBox, e, mx, my)
		return
	}

	if e.tool == ToolPaint {
		c.updatePaintTool(scene, viewBox, e, mx, my)
		return
	}

	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	if !viewBox.Contains(mx, my) {
		return
	}
	sceneX, sceneY := c.windowToScene(viewBox, e.project, mx, my)

	switch e.tool {
	case ToolPlace:
		c.handlePlace(scene, e, sceneX, sceneY)
	case ToolDelete:
		c.handleDelete(scene, viewBox, e, mx, my)
	}
}

// updatePaintTool is the Paint tool's per-frame input handler. It
// dispatches based on the painter's active sub-mode:
//
//   - Brush: LMB-down opens a stroke; held LMB paints the hovered
//     cell into the stroke (the dedup gate inside the painter drops
//     repeats); LMB-up commits the stroke onto the undo stack.
//   - Bucket: LMB-down does a single-shot flood-fill and commits
//     atomically.
//   - Rectangle: LMB-down sets the anchor; LMB-up fills the rect
//     and commits.
//
// All three sub-modes mark the editor dirty whenever the stroke
// actually changed at least one cell (the StrokeCommand return from
// the painter is nil for no-op strokes).
func (c *Canvas) updatePaintTool(scene *pixelforge_project.Scene, viewBox widgets.Rect, e *Editor, mx, my int) {
	painter := e.Painter()
	if painter == nil {
		return
	}
	// The painter writes against the first tilemap layer in the
	// scene (palette's "active layer" concept is a future unit; v1
	// always targets index 0). If the scene has no tilemaps yet we
	// can't paint — silently no-op so a mistakenly Paint-active
	// click on an empty scene doesn't crash.
	if len(scene.TileAtlases) == 0 {
		return
	}
	layer := &scene.TileAtlases[0]
	stack := e.UndoStack()

	insideViewbox := viewBox.Contains(mx, my)
	sceneX, sceneY := c.windowToScene(viewBox, e.project, mx, my)
	col, row := scenePxToCell(sceneX, sceneY, layer)

	switch painter.SubMode {
	case PaintBrush:
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && insideViewbox {
			painter.BrushStartStroke(layer)
			if painter.BrushPaint(col, row, painter.ActiveTileID) {
				e.MarkDirty()
			}
		} else if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && insideViewbox {
			if painter.BrushPaint(col, row, painter.ActiveTileID) {
				e.MarkDirty()
			}
		}
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
			if cmd := painter.BrushEndStroke(stack); cmd != nil {
				e.recordStrokeAndQueuePromotions(layer, cmd)
			}
		}
	case PaintBucket:
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && insideViewbox {
			if cmd := painter.BucketFill(layer, col, row, painter.ActiveTileID, stack); cmd != nil {
				e.MarkDirty()
				e.recordStrokeAndQueuePromotions(layer, cmd)
			}
		}
	case PaintRectangle:
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && insideViewbox {
			painter.RectangleAnchor(col, row)
		} else if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && painter.RectangleAnchored() {
			if cmd := painter.RectangleFill(layer, col, row, painter.ActiveTileID, stack); cmd != nil {
				e.MarkDirty()
				e.recordStrokeAndQueuePromotions(layer, cmd)
			}
		}
	}
}

// handleRightClick builds a RightClickContext for the (mx, my)
// position and dispatches into HandleRightClick. v1 always populates
// the Cell field; U7 will hit-test against scene.Entities first and
// set EntityID when the click lands on an existing entity marker.
//
// The function is the single seam tests use to drive the right-click
// flow; the live ImGui popup that surfaces "Set spawn here" as a
// clickable menu item will (in a future UI polish unit) invoke the
// same HandleRightClick with the same context, so test and
// production paths converge on one handler.
func (c *Canvas) handleRightClick(scene *pixelforge_project.Scene, viewBox widgets.Rect, e *Editor, mx, my int) {
	if scene == nil || e == nil {
		return
	}
	// First, hit-test entities so a future U7 extension can populate
	// EntityID — v1 leaves the entity branch unhandled but the
	// dispatch shape is structurally complete.
	entityID := c.PickEntityAt(scene, viewBox, e.project, mx, my)
	if entityID != "" {
		HandleRightClick(e, RightClickContext{
			Scene:    scene,
			EntityID: entityID,
		})
		return
	}
	sceneX, sceneY := c.windowToScene(viewBox, e.project, mx, my)
	col, row := scenePxToCellScene(sceneX, sceneY, scene)
	HandleRightClick(e, RightClickContext{
		Scene: scene,
		Cell:  &RightClickCell{Col: col, Row: row},
	})
}

// scenePxToCell maps a scene-space pixel (sceneX, sceneY) into the
// (col, row) of layer's tile grid. Layer's TileW/TileH are the
// authoritative tile size; falls back to 8 (the v1 default) when
// the layer hasn't been initialized.
func scenePxToCell(sceneX, sceneY int, layer *pixelforge_project.TileAtlas) (int, int) {
	tw, th := 8, 8
	if layer != nil {
		if layer.TileW > 0 {
			tw = layer.TileW
		}
		if layer.TileH > 0 {
			th = layer.TileH
		}
	}
	col := sceneX / tw
	row := sceneY / th
	if col < 0 {
		col = 0
	}
	if row < 0 {
		row = 0
	}
	return col, row
}

// scenePxToCellScene is the right-click variant — when the scene has
// no tilemaps yet (so no TileW/TileH is available), fall back to the
// v1 default 8×8 tile size. The right-click "Set spawn here" action
// is defined in cell coords, so we need a cell mapping even when the
// scene's tilemaps are empty.
func scenePxToCellScene(sceneX, sceneY int, scene *pixelforge_project.Scene) (int, int) {
	if scene == nil || len(scene.TileAtlases) == 0 {
		return scenePxToCell(sceneX, sceneY, nil)
	}
	return scenePxToCell(sceneX, sceneY, &scene.TileAtlases[0])
}

// PlaceEntity creates a new entity at the supplied scene-space
// coordinates. Public so tests can drive placement without simulating
// mouse input.
//
// Idea #1 v1 (U7) changes the placement contract: the supplied
// (sceneX, sceneY) pixel coords are snapped to the nearest tile cell
// using pixelforge_entity.TileW / TileH (8 px each — the Mario-strip
// convention), then written to the entity's TileX / TileY fields.
// The legacy float64 Position.X / Y is also populated to the cell's
// top-left pixel so the editor's existing entity-marker hit-test +
// drag tooling (which still reads Position) continues to function for
// freshly-placed entities. The cell coords are clamped to the level's
// addressable tile range so a click past the level's right or bottom
// edge resolves to the last valid cell.
func (c *Canvas) PlaceEntity(scene *pixelforge_project.Scene, e *Editor, sceneX, sceneY int) *pixelforge_project.Entity {
	sprite := e.SelectedSpriteName()
	if sprite == "" {
		e.SetStatusMessage("Select a sprite first")
		return nil
	}
	tileX, tileY := clampTileCellToLevel(scene, e.project, sceneX/pixelforge_entity.TileW, sceneY/pixelforge_entity.TileH)
	c.nextEntitySeq++
	id := fmt.Sprintf("ent-%d", c.nextEntitySeq)
	ent := pixelforge_project.Entity{
		ID:    id,
		Name:  sprite,
		TileX: tileX,
		TileY: tileY,
		Position: pixelforge_project.EntityPosition{
			X: float64(tileX * pixelforge_entity.TileW),
			Y: float64(tileY * pixelforge_entity.TileH),
		},
		Components: []pixelforge_project.EntityComponent{{
			Type: "Sprite",
			Values: map[string]any{
				"sprite": sprite,
			},
		}},
	}
	scene.Entities = append(scene.Entities, ent)
	e.SelectEntity(id)
	e.MarkDirty()
	return &scene.Entities[len(scene.Entities)-1]
}

// clampTileCellToLevel clamps (col, row) into the level's addressable
// tile range. The level is GridWidthScreens × 32 tiles wide and
// GridHeightScreens × 30 tiles tall (32×30 tiles per 256×240 screen).
// Pre-defaults scenes (GridWidthScreens / GridHeightScreens == 0) fall
// back to a single-screen bound so the legacy single-screen behavior
// survives the schema migration — same fallback shape used by
// sceneDragBounds.
func clampTileCellToLevel(scene *pixelforge_project.Scene, project *pixelforge_project.Project, col, row int) (int, int) {
	const tilesPerScreenX, tilesPerScreenY = 32, 30
	widthScreens, heightScreens := 1, 1
	if scene != nil {
		if scene.GridWidthScreens > 0 {
			widthScreens = scene.GridWidthScreens
		}
		if scene.GridHeightScreens > 0 {
			heightScreens = scene.GridHeightScreens
		}
	}
	maxCol := widthScreens*tilesPerScreenX - 1
	maxRow := heightScreens*tilesPerScreenY - 1
	// Belt-and-braces for very small (or zero-width) project overrides:
	// keep at least one addressable cell so a click on a fresh canvas
	// never returns a negative max.
	if maxCol < 0 {
		maxCol = 0
	}
	if maxRow < 0 {
		maxRow = 0
	}
	_ = project // reserved for future per-project tile-size overrides.
	if col < 0 {
		col = 0
	}
	if col > maxCol {
		col = maxCol
	}
	if row < 0 {
		row = 0
	}
	if row > maxRow {
		row = maxRow
	}
	return col, row
}

// DeleteEntityAt removes the topmost entity whose marker contains the
// supplied window-space coordinates. Returns the deleted ID or "".
func (c *Canvas) DeleteEntityAt(scene *pixelforge_project.Scene, viewBox widgets.Rect, project *pixelforge_project.Project, mx, my int) string {
	scaleX, scaleY := c.scale(viewBox, project)
	for i := len(scene.Entities) - 1; i >= 0; i-- {
		marker := c.entityMarkerRect(viewBox, &scene.Entities[i], scaleX, scaleY)
		if marker.Contains(mx, my) {
			id := scene.Entities[i].ID
			scene.Entities = append(scene.Entities[:i], scene.Entities[i+1:]...)
			return id
		}
	}
	return ""
}

// PickEntityAt returns the topmost entity ID whose marker contains
// (mx, my). Returns "" when nothing is hit.
func (c *Canvas) PickEntityAt(scene *pixelforge_project.Scene, viewBox widgets.Rect, project *pixelforge_project.Project, mx, my int) string {
	scaleX, scaleY := c.scale(viewBox, project)
	for i := len(scene.Entities) - 1; i >= 0; i-- {
		marker := c.entityMarkerRect(viewBox, &scene.Entities[i], scaleX, scaleY)
		if marker.Contains(mx, my) {
			return scene.Entities[i].ID
		}
	}
	return ""
}

func (c *Canvas) updateSelectTool(scene *pixelforge_project.Scene, viewBox widgets.Rect, e *Editor, mx, my int) {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		id := c.PickEntityAt(scene, viewBox, e.project, mx, my)
		e.SelectEntity(id)
		if id != "" {
			c.dragging = id
			ent := e.selectedEntity()
			if ent != nil {
				scaleX, scaleY := c.scale(viewBox, e.project)
				cx := float64(viewBox.X) + ent.Position.X*scaleX
				cy := float64(viewBox.Y) + ent.Position.Y*scaleY
				c.dragOffX = float64(mx) - cx
				c.dragOffY = float64(my) - cy
			}
		} else {
			c.dragging = ""
		}
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		c.dragging = ""
		return
	}
	if c.dragging == "" {
		return
	}
	ent := e.selectedEntity()
	if ent == nil || ent.ID != c.dragging {
		return
	}
	scaleX, scaleY := c.scale(viewBox, e.project)
	if scaleX == 0 || scaleY == 0 {
		return
	}
	newX := (float64(mx) - c.dragOffX - float64(viewBox.X)) / scaleX
	newY := (float64(my) - c.dragOffY - float64(viewBox.Y)) / scaleY
	// Idea #1 v1 widens the drag clamp from the single-screen
	// assumption [0, ScreenWidth] to the full level bounds
	// [0, GridWidthScreens * ScreenWidth]. Pre-v1 scenes (those whose
	// applyDefaults hasn't run, e.g. fresh-from-NewProject) fall back
	// to the project's ScreenWidth/Height so the legacy clamp behavior
	// survives the schema migration.
	maxX, maxY := sceneDragBounds(scene, e.project)
	newX = clampF(newX, 0, maxX)
	newY = clampF(newY, 0, maxY)
	if newX != ent.Position.X || newY != ent.Position.Y {
		ent.Position.X = newX
		ent.Position.Y = newY
		e.MarkDirty()
	}
}

func (c *Canvas) handlePlace(scene *pixelforge_project.Scene, e *Editor, sceneX, sceneY int) {
	c.PlaceEntity(scene, e, sceneX, sceneY)
}

func (c *Canvas) handleDelete(scene *pixelforge_project.Scene, viewBox widgets.Rect, e *Editor, mx, my int) {
	deleted := c.DeleteEntityAt(scene, viewBox, e.project, mx, my)
	if deleted == "" {
		return
	}
	if e.selectedEntityID == deleted {
		e.SelectEntity("")
	}
	e.MarkDirty()
}

// viewBox returns the letterboxed area for the project's screen.
//
// Idea #1 v1 shifts the box by the engine-wide pixelforge.Camera
// offset so the editor's window→scene coordinate conversion stays
// consistent with what sceneGame.Draw renders. Without this shift,
// once the camera scrolls past the first screen the mouse-pick math
// would mis-target — clicks would land on the wrong cell because
// the painted texture has moved underneath the static viewBox but
// the coordinate conversion does not know it.
//
// The shift is in scene-space pixels (scale-corrected): camera at
// X=100 in scene-space moves the viewBox left by 100 scene-pixels,
// which is 100*scaleX window-pixels. windowToScene then adds the
// camera back so clicks resolve to absolute scene-space coords.
func (c *Canvas) viewBox(area widgets.Rect, project *pixelforge_project.Project) widgets.Rect {
	if project == nil || project.ScreenWidth <= 0 || project.ScreenHeight <= 0 {
		return area
	}
	sw := float64(project.ScreenWidth)
	sh := float64(project.ScreenHeight)
	aspect := sw / sh
	availW := float64(area.W)
	availH := float64(area.H)

	var base widgets.Rect
	if availW/availH > aspect {
		// Wider than the scene: pillarbox.
		w := availH * aspect
		x := float64(area.X) + (availW-w)/2
		base = widgets.Rect{X: int(x), Y: area.Y, W: int(w), H: area.H}
	} else {
		h := availW / aspect
		y := float64(area.Y) + (availH-h)/2
		base = widgets.Rect{X: area.X, Y: int(y), W: area.W, H: int(h)}
	}

	camX, camY := pixelforge.Camera.X, pixelforge.Camera.Y
	if camX == 0 && camY == 0 {
		return base
	}
	scaleX := float64(base.W) / sw
	scaleY := float64(base.H) / sh
	base.X -= int(float64(camX) * scaleX)
	base.Y -= int(float64(camY) * scaleY)
	return base
}

// sceneDragBounds returns the maximum (X, Y) scene-space coordinate
// an entity can be dragged to. Idea #1 v1 widens these from the
// pre-v1 single-screen assumption to the full level bounds derived
// from Scene.GridWidthScreens / GridHeightScreens. Pre-v1 scenes
// whose grid fields haven't been defaulted (e.g. fresh-from-
// NewProject in tests) fall back to the project's ScreenWidth/Height
// so the legacy behavior survives the schema migration.
func sceneDragBounds(scene *pixelforge_project.Scene, project *pixelforge_project.Project) (float64, float64) {
	const screenW, screenH = 256, 240
	maxX := float64(project.ScreenWidth)
	maxY := float64(project.ScreenHeight)
	if scene == nil {
		return maxX, maxY
	}
	if scene.GridWidthScreens > 0 {
		maxX = float64(scene.GridWidthScreens * screenW)
	}
	if scene.GridHeightScreens > 0 {
		maxY = float64(scene.GridHeightScreens * screenH)
	}
	return maxX, maxY
}

// scale returns the pixels-per-scene-unit factor in both axes.
func (c *Canvas) scale(viewBox widgets.Rect, project *pixelforge_project.Project) (float64, float64) {
	if project == nil || project.ScreenWidth <= 0 || project.ScreenHeight <= 0 {
		return 1, 1
	}
	return float64(viewBox.W) / float64(project.ScreenWidth),
		float64(viewBox.H) / float64(project.ScreenHeight)
}

// windowToScene converts a window-space coordinate inside the view box
// to scene-space integers (clamped to scene bounds).
func (c *Canvas) windowToScene(viewBox widgets.Rect, project *pixelforge_project.Project, mx, my int) (int, int) {
	sx, sy := c.scale(viewBox, project)
	if sx == 0 {
		sx = 1
	}
	if sy == 0 {
		sy = 1
	}
	x := int((float64(mx-viewBox.X)) / sx)
	y := int((float64(my-viewBox.Y)) / sy)
	if project != nil {
		if x < 0 {
			x = 0
		}
		if x > project.ScreenWidth {
			x = project.ScreenWidth
		}
		if y < 0 {
			y = 0
		}
		if y > project.ScreenHeight {
			y = project.ScreenHeight
		}
	}
	return x, y
}

// entityMarkerRect returns the window-space rect used to render entity.
// The marker is a fixed 12-pixel square centered on the entity.
func (c *Canvas) entityMarkerRect(viewBox widgets.Rect, ent *pixelforge_project.Entity, scaleX, scaleY float64) widgets.Rect {
	cx := float64(viewBox.X) + ent.Position.X*scaleX
	cy := float64(viewBox.Y) + ent.Position.Y*scaleY
	const size = 12
	return widgets.Rect{
		X: int(cx) - size/2,
		Y: int(cy) - size/2,
		W: size,
		H: size,
	}
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func strokeWidgetsRect(dst *ebiten.Image, r widgets.Rect, c color.RGBA) {
	// reuse vector.StrokeRect via the inspector helper signature
	strokeRectAt(dst, r, c)
}

var (
	colCanvasBgInner = color.RGBA{R: 0x0a, G: 0x0a, B: 0x12, A: 0xff}
	colCanvasViewBox = color.RGBA{R: 0x18, G: 0x18, B: 0x24, A: 0xff}
	colCanvasEntity  = color.RGBA{R: 0xdd, G: 0xb6, B: 0x4a, A: 0xff}
)

// (U9 deleted the cart-resident DrawCanvas + tool-indicator path —
// the Scene preview renders exclusively through the U5 texture
// pipeline now. The pixelforge engine import survives because
// Canvas.Draw still paints into the off-screen scene texture.)
