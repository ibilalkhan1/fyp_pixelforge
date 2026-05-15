package editor

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_ebiten"
	pgui "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_gui"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// EditorCanvasW, EditorCanvasH are the logical pixel dimensions of the
// editor's Pixelforge canvas. Picked to match the M0 default window so
// the chrome layout math is integer-pixel arithmetic.
const (
	EditorCanvasW = 1280
	EditorCanvasH = 800
)

// editorCart owns the editor's logical Pixelforge canvas, a pgui root
// element used by canvas-resident chrome, and the keyboard focus
// manager. Editor.Draw routes the workspace region through it; the
// surrounding native chrome continues to paint via the existing
// ebitenutil/vector path.
type editorCart struct {
	canvas pixelforge.Canvas

	// root covers the entire logical canvas. Workspaces attach their
	// canvas-resident chrome here.
	root *pgui.Element

	// focus tracks the canvas-resident widget that owns keyboard input.
	focus *pgui.FocusManager

	// workspaceRoot is a child of root that workspaces draw their
	// per-workspace chrome into. It is recreated on workspace switch
	// so each workspace gets a clean element tree.
	workspaceRoot *pgui.Element

	// theme is the editor's loaded theme. nil means use M0-M2 defaults.
	theme *EditorTheme

	// hostImage is the reusable Ebitengine image we rasterise the
	// Pixelforge canvas into before blitting to the screen.
	hostImage *ebiten.Image
}

// newEditorCart constructs an editorCart with a freshly allocated
// canvas and an empty pgui root. The caller is expected to load
// editor.pforge afterwards via loadEditorTheme.
func newEditorCart() *editorCart {
	c := &editorCart{
		canvas: pixelforge.NewCanvas(EditorCanvasW, EditorCanvasH),
		focus:  pgui.NewFocusManager(),
	}
	c.root = &pgui.Element{
		Area: pixelforge.IntArea{W: EditorCanvasW, H: EditorCanvasH},
	}
	c.workspaceRoot = pgui.Attach(c.root, 0, 0, EditorCanvasW, EditorCanvasH)
	return c
}

// Canvas returns the cart's logical canvas (for tests / advanced uses).
func (c *editorCart) Canvas() pixelforge.Canvas { return c.canvas }

// Root returns the pgui root for the editor canvas.
func (c *editorCart) Root() *pgui.Element { return c.root }

// WorkspaceRoot returns the pgui element workspaces attach their
// chrome to. Switching workspaces should detach the previous tree.
func (c *editorCart) WorkspaceRoot() *pgui.Element { return c.workspaceRoot }

// FocusManager returns the cart's focus manager.
func (c *editorCart) FocusManager() *pgui.FocusManager { return c.focus }

// Theme returns the editor's loaded theme (or nil if not yet loaded).
func (c *editorCart) Theme() *EditorTheme { return c.theme }

// SetTheme stores the loaded editor theme.
func (c *editorCart) SetTheme(t *EditorTheme) { c.theme = t }

// ResetWorkspaceRoot detaches the existing workspace root tree and
// returns a fresh element scoped to the workspace region (relative to
// the canvas).
func (c *editorCart) ResetWorkspaceRoot(rel widgets.Rect) *pgui.Element {
	if c.workspaceRoot != nil {
		c.root.Detach(c.workspaceRoot)
	}
	c.workspaceRoot = pgui.Attach(c.root, rel.X, rel.Y, rel.W, rel.H)
	return c.workspaceRoot
}

// renderInto paints the editor canvas. It sets the draw target to the
// cart's canvas, clears with the theme background, paints the
// canvas-resident workspace chrome (left panel / centre canvas /
// right panel), runs the active workspace's canvas-resident Draw, then
// restores the prior draw target.
//
// Layout inside the canvas:
//
//	┌──────────────┬────────────────────────┬──────────────┐
//	│ left panel   │   workspace viewport   │ right panel  │
//	│ (asset bro.) │                        │ (inspector)  │
//	└──────────────┴────────────────────────┴──────────────┘
//
// Panel widths are derived from the M0-M2 chrome defaults scaled to
// the editor canvas width so the layout stays predictable.
func (c *editorCart) renderInto(e *Editor) {
	prevTarget := pixelforge.SetDrawTarget(c.canvas)
	defer pixelforge.SetDrawTarget(prevTarget)

	theme := DefaultEditorTheme()
	if c.theme != nil {
		theme = c.theme
	}
	prevColor := pixelforge.SetColor(theme.BackgroundSlot)
	defer pixelforge.SetColor(prevColor)
	c.canvas.Clear(theme.BackgroundSlot)

	if e == nil {
		c.root.Draw()
		return
	}

	leftW, rightW := c.panelWidths()
	canvasW := c.canvas.W()
	canvasH := c.canvas.H()

	// U33: when chrome is hidden, the entire canvas is the workspace
	// region — no side panels, full-bleed game canvas behind the
	// workspace.
	if e.ChromeHidden() {
		ws := e.activeWorkspaceImpl()
		if cw, ok := ws.(CanvasWorkspace); ok && cw != nil {
			cw.DrawCanvas(widgets.Rect{X: 0, Y: 0, W: canvasW, H: canvasH}, e)
		}
		c.root.Draw()
		return
	}

	leftRect := widgets.Rect{X: 0, Y: 0, W: leftW, H: canvasH}
	rightRect := widgets.Rect{X: canvasW - rightW, Y: 0, W: rightW, H: canvasH}
	workspaceRect := widgets.Rect{X: leftW, Y: 0, W: canvasW - leftW - rightW, H: canvasH}

	// Active workspace paints its chrome into the workspace region.
	ws := e.activeWorkspaceImpl()
	if cw, ok := ws.(CanvasWorkspace); ok && cw != nil {
		cw.DrawCanvas(workspaceRect, e)
	}

	// Asset browser left panel.
	if e.assetBrowser != nil {
		e.assetBrowser.DrawCanvas(leftRect, e)
	}
	// Inspector right panel.
	if e.inspector != nil {
		e.inspector.DrawCanvas(rightRect, e.project, e.selectedEntity(), theme)
	}

	c.root.Draw()
}

// panelWidths returns the left/right panel widths in canvas pixels.
// Derived from the M0-M2 chrome defaults so muscle memory carries.
func (c *editorCart) panelWidths() (left, right int) {
	left = c.canvas.W() / 6
	right = c.canvas.W() / 5
	if left < 180 {
		left = 180
	}
	if right < 220 {
		right = 220
	}
	return left, right
}

// blitToScreen copies the editor canvas into the given workspace rect on
// the supplied Ebitengine image. The blit uses integer-pixel scaling
// that fits the canvas inside the rect (letterbox/pillarbox preserved).
func (c *editorCart) blitToScreen(screen *ebiten.Image, workspaceRect widgets.Rect) {
	if workspaceRect.W <= 0 || workspaceRect.H <= 0 {
		return
	}
	sub := screen.SubImage(image.Rect(
		workspaceRect.X, workspaceRect.Y,
		workspaceRect.X+workspaceRect.W, workspaceRect.Y+workspaceRect.H,
	)).(*ebiten.Image)

	if c.hostImage == nil ||
		c.hostImage.Bounds().Dx() != c.canvas.W() ||
		c.hostImage.Bounds().Dy() != c.canvas.H() {
		c.hostImage = ebiten.NewImage(c.canvas.W(), c.canvas.H())
	}
	pixelforge_ebiten.CopyCanvasToEbitenImage(c.canvas, c.hostImage)

	scaleX := float64(workspaceRect.W) / float64(c.canvas.W())
	scaleY := float64(workspaceRect.H) / float64(c.canvas.H())
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}
	scaledW := float64(c.canvas.W()) * scale
	scaledH := float64(c.canvas.H()) * scale
	offX := (float64(workspaceRect.W) - scaledW) / 2
	offY := (float64(workspaceRect.H) - scaledH) / 2

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(workspaceRect.X)+offX, float64(workspaceRect.Y)+offY)
	sub.DrawImage(c.hostImage, op)
}

// CanvasRect returns the workspace-rect-relative position of (mx, my)
// in window-pixel space, translated to canvas-local coordinates. The
// caller passes the workspace rect (where the canvas blits) and the
// global pointer; the result is canvas-local (X, Y) coordinates.
//
// When the pointer lies outside the canvas blit region, the returned
// inside flag is false.
func (c *editorCart) WindowToCanvas(workspaceRect widgets.Rect, mx, my int) (cx, cy int, inside bool) {
	if workspaceRect.W <= 0 || workspaceRect.H <= 0 {
		return 0, 0, false
	}
	scaleX := float64(workspaceRect.W) / float64(c.canvas.W())
	scaleY := float64(workspaceRect.H) / float64(c.canvas.H())
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}
	scaledW := int(float64(c.canvas.W()) * scale)
	scaledH := int(float64(c.canvas.H()) * scale)
	offX := workspaceRect.X + (workspaceRect.W-scaledW)/2
	offY := workspaceRect.Y + (workspaceRect.H-scaledH)/2

	if mx < offX || my < offY || mx >= offX+scaledW || my >= offY+scaledH {
		return 0, 0, false
	}
	if scale == 0 {
		return 0, 0, false
	}
	cx = int(float64(mx-offX) / scale)
	cy = int(float64(my-offY) / scale)
	return cx, cy, true
}

