package editor

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/ibilalkhan1/fyp_pixelforge"
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

// Update handles mouse input. e is the editor; area is the canvas rect.
func (c *Canvas) Update(area widgets.Rect, e *Editor) {
	if area.W <= 0 || area.H <= 0 {
		return
	}
	scene := e.activeScene()
	if scene == nil {
		return
	}
	mx, my := ebiten.CursorPosition()
	viewBox := c.viewBox(area, e.project)

	if e.tool == ToolSelect {
		c.updateSelectTool(scene, viewBox, e, mx, my)
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

// PlaceEntity creates a new entity at the supplied scene-space
// coordinates. Public so tests can drive placement without simulating
// mouse input.
func (c *Canvas) PlaceEntity(scene *pixelforge_project.Scene, e *Editor, sceneX, sceneY int) *pixelforge_project.Entity {
	sprite := e.SelectedSpriteName()
	if sprite == "" {
		e.SetStatusMessage("Select a sprite first")
		return nil
	}
	c.nextEntitySeq++
	id := fmt.Sprintf("ent-%d", c.nextEntitySeq)
	ent := pixelforge_project.Entity{
		ID:       id,
		Name:     sprite,
		Position: pixelforge_project.EntityPosition{X: float64(sceneX), Y: float64(sceneY)},
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
	newX = clampF(newX, 0, float64(e.project.ScreenWidth))
	newY = clampF(newY, 0, float64(e.project.ScreenHeight))
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
func (c *Canvas) viewBox(area widgets.Rect, project *pixelforge_project.Project) widgets.Rect {
	if project == nil || project.ScreenWidth <= 0 || project.ScreenHeight <= 0 {
		return area
	}
	sw := float64(project.ScreenWidth)
	sh := float64(project.ScreenHeight)
	aspect := sw / sh
	availW := float64(area.W)
	availH := float64(area.H)
	if availW/availH > aspect {
		// Wider than the scene: pillarbox.
		w := availH * aspect
		x := float64(area.X) + (availW-w)/2
		return widgets.Rect{X: int(x), Y: area.Y, W: int(w), H: area.H}
	}
	h := availW / aspect
	y := float64(area.Y) + (availH-h)/2
	return widgets.Rect{X: area.X, Y: int(y), W: area.W, H: int(h)}
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

// DrawCanvas renders the Scene viewport into the editor cart's
// Pixelforge canvas using engine primitives. rel is the workspace
// region inside the editor canvas (top-left at 0,0, sized to the
// canvas dimensions). The viewport letterbox + entity markers + the
// selection outline are all painted via pixelforge.RectFill / Rect.
func (c *Canvas) DrawCanvas(rel widgets.Rect, e *Editor) {
	if rel.W <= 0 || rel.H <= 0 {
		return
	}
	theme := DefaultEditorTheme()
	if e.cart != nil && e.cart.Theme() != nil {
		theme = e.cart.Theme()
	}

	prevColor := pixelforge.GetColor()
	defer pixelforge.SetColor(prevColor)

	// Workspace area background (canvas viewport bg).
	pixelforge.SetColor(theme.BackgroundSlot)
	pixelforge.RectFill(rel.X, rel.Y, rel.X+rel.W-1, rel.Y+rel.H-1)

	scene := e.activeScene()
	if scene == nil {
		return
	}

	viewBox := c.viewBox(rel, e.project)
	pixelforge.SetColor(theme.PanelSlot)
	pixelforge.RectFill(viewBox.X, viewBox.Y,
		viewBox.X+viewBox.W-1, viewBox.Y+viewBox.H-1)

	scaleX, scaleY := c.scale(viewBox, e.project)
	// U33: when chrome is hidden, the Scene viewport shows only the
	// game canvas — markers + selection outline + tool indicator are
	// part of the editor chrome and must hide too.
	if e.ChromeHidden() {
		return
	}
	for i := range scene.Entities {
		ent := &scene.Entities[i]
		marker := c.entityMarkerRect(viewBox, ent, scaleX, scaleY)
		pixelforge.SetColor(theme.AccentSlot)
		pixelforge.RectFill(marker.X, marker.Y,
			marker.X+marker.W-1, marker.Y+marker.H-1)
		if ent.ID == e.selectedEntityID {
			pixelforge.SetColor(theme.TextSlot)
			pixelforge.Rect(marker.X-1, marker.Y-1,
				marker.X+marker.W, marker.Y+marker.H)
		}
	}

	// U29: tool indicator using cofont (no ebitenutil reach-out).
	c.drawToolIndicator(rel, e, theme)
}

// drawToolIndicator prints the active tool name + selected sprite via
// pixelforge_cofont in the top-left of the workspace region. The
// indicator is hidden when no project is loaded.
func (c *Canvas) drawToolIndicator(rel widgets.Rect, e *Editor, theme *EditorTheme) {
	if e == nil || e.project == nil || e.activeScene() == nil {
		return
	}
	label := e.Tool().String()
	sprite := e.SelectedSpriteName()
	if e.Tool() == ToolPlace {
		if sprite == "" {
			label += " — (no sprite)"
		} else {
			label += " — " + sprite
		}
	}
	prevColor := pixelforge.SetColor(theme.TextSlot)
	defer pixelforge.SetColor(prevColor)
	pcofont(label, rel.X+8, rel.Y+8)
}
