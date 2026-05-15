package editor

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// debugLineHeight matches the metrics overlay so chrome rows line up.
const debugLineHeight = 16

// printAt wraps ebitenutil.DebugPrintAt; kept as a single call site so
// future themed text rendering can swap implementations without churn.
func printAt(dst *ebiten.Image, s string, x, y int) {
	ebitenutil.DebugPrintAt(dst, s, x, y)
}

// Default chrome dimensions in window pixels. Picked to match the
// dimensions of the legacy studio so muscle memory still works, and to
// give panels enough room for the M1+ inspector content.
const (
	defaultTitleBarH   = 40
	defaultLeftPanelW  = 220
	defaultRightPanelW = 260
	defaultStatusBarH  = 24

	// Minimum dimensions the chrome refuses to shrink below; the canvas
	// is the slack space, so we always preserve at least this much for it.
	minCanvasW = 200
	minCanvasH = 80

	// Below this minimum panels are clamped so the canvas stays usable
	// on small windows. Panels never grow above the defaults — extra
	// window space goes to the canvas.
	minLeftPanelW  = 80
	minRightPanelW = 80
	minTitleBarH   = 24
	minStatusBarH  = 16
)

// chromeLayout describes where every chrome region sits inside the window
// after the latest call to recompute. Coordinates are window-pixel space.
type chromeLayout struct {
	WindowW, WindowH int

	TitleBarH   int
	LeftPanelW  int
	RightPanelW int
	StatusBarH  int

	TitleBar   rect
	LeftPanel  rect
	RightPanel rect
	Canvas     rect
	StatusBar  rect

	// MenuBar is the strip above TitleBar.
	MenuBar rect

	// TabStrip is the strip between menu bar and the body region.
	TabStrip rect
}

// rect is a plain rectangle in window-pixel space.
type rect struct {
	X, Y, W, H int
}

// defaultChromeLayout returns the chrome with default sizes, not yet
// computed for any window. recompute fills in the per-region rects.
func defaultChromeLayout() *chromeLayout {
	return &chromeLayout{
		TitleBarH:   defaultTitleBarH,
		LeftPanelW:  defaultLeftPanelW,
		RightPanelW: defaultRightPanelW,
		StatusBarH:  defaultStatusBarH,
	}
}

// computeChromeLayout returns the chrome rects for a w×h window using the
// supplied chrome target dimensions. Exposed for tests so they can assert
// exact pixel positions without spinning up a real Editor.
func computeChromeLayout(w, h, titleH, leftW, rightW, statusH int) *chromeLayout {
	l := &chromeLayout{
		TitleBarH:   titleH,
		LeftPanelW:  leftW,
		RightPanelW: rightW,
		StatusBarH:  statusH,
	}
	l.recompute(w, h)
	return l
}

// LeftGutterRect returns the 4px-wide drag gutter between the left
// panel and the canvas, in window-pixel space. Empty when the canvas
// has collapsed.
func (l *chromeLayout) LeftGutterRect() rect {
	if l.Canvas.W <= 0 {
		return rect{}
	}
	return rect{X: l.Canvas.X - 2, Y: l.Canvas.Y, W: 4, H: l.Canvas.H}
}

// RightGutterRect returns the gutter between the canvas and the right
// panel.
func (l *chromeLayout) RightGutterRect() rect {
	if l.Canvas.W <= 0 {
		return rect{}
	}
	return rect{X: l.Canvas.X + l.Canvas.W - 2, Y: l.Canvas.Y, W: 4, H: l.Canvas.H}
}

// ApplyLeftPanelDelta widens (positive) or narrows (negative) the
// left panel by delta pixels, clamped to (minLeftPanelW, max). The
// canvas absorbs the difference.
func (l *chromeLayout) ApplyLeftPanelDelta(delta int) {
	want := l.LeftPanelW + delta
	if want < minLeftPanelW {
		want = minLeftPanelW
	}
	maxLeft := (l.WindowW - minCanvasW - l.RightPanelW)
	if want > maxLeft {
		want = maxLeft
	}
	l.LeftPanelW = want
	l.recompute(l.WindowW, l.WindowH)
}

// ApplyRightPanelDelta widens / narrows the right panel symmetrically.
func (l *chromeLayout) ApplyRightPanelDelta(delta int) {
	want := l.RightPanelW + delta
	if want < minRightPanelW {
		want = minRightPanelW
	}
	maxRight := (l.WindowW - minCanvasW - l.LeftPanelW)
	if want > maxRight {
		want = maxRight
	}
	l.RightPanelW = want
	l.recompute(l.WindowW, l.WindowH)
}

// recompute lays the chrome out for a given window size. Panel widths and
// title/status heights clamp down so the canvas always retains at least
// minCanvasW × minCanvasH pixels; if the window is smaller than the chrome
// itself, the canvas shrinks to 0 but the regions stay non-overlapping.
func (l *chromeLayout) recompute(w, h int) {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	l.WindowW, l.WindowH = w, h

	menuH := widgets.MenuBarHeight
	stripH := tabStripH

	// Clamp horizontal panels so a minimum canvas survives. We shrink
	// left and right symmetrically when needed.
	titleH := clampMax(l.TitleBarH, minTitleBarH, h/3)
	statusH := clampMax(l.StatusBarH, minStatusBarH, h/4)
	chromeFixed := menuH + stripH + titleH + statusH
	if chromeFixed+minCanvasH > h {
		titleH = h / 4
		statusH = h / 4
		if titleH < 0 {
			titleH = 0
		}
		if statusH < 0 {
			statusH = 0
		}
	}

	leftW := clampMax(l.LeftPanelW, minLeftPanelW, (w-minCanvasW)/2)
	rightW := clampMax(l.RightPanelW, minRightPanelW, (w-minCanvasW)/2)
	if leftW+rightW+minCanvasW > w {
		leftW = w / 2
		rightW = w - leftW
		if leftW < 0 {
			leftW = 0
		}
		if rightW < 0 {
			rightW = 0
		}
	}

	// Layout vertical regions top → bottom.
	l.MenuBar = rect{X: 0, Y: 0, W: w, H: menuH}
	l.TitleBar = rect{X: 0, Y: menuH, W: w, H: titleH}
	l.TabStrip = rect{X: 0, Y: menuH + titleH, W: w, H: stripH}

	bodyY := menuH + titleH + stripH
	bodyH := h - bodyY - statusH
	if bodyH < 0 {
		bodyH = 0
	}

	l.LeftPanel = rect{X: 0, Y: bodyY, W: leftW, H: bodyH}
	l.RightPanel = rect{X: w - rightW, Y: bodyY, W: rightW, H: bodyH}

	canvasX := leftW
	canvasW := w - leftW - rightW
	if canvasW < 0 {
		canvasW = 0
	}
	l.Canvas = rect{X: canvasX, Y: bodyY, W: canvasW, H: bodyH}

	l.StatusBar = rect{X: 0, Y: h - statusH, W: w, H: statusH}
}

// clampMax returns target if it fits between minV and maxV, otherwise
// the appropriate bound. maxV is allowed to be smaller than minV — in
// that case the result is minV (we never go below the floor).
func clampMax(target, minV, maxV int) int {
	if target < minV {
		target = minV
	}
	if maxV < minV {
		return minV
	}
	if target > maxV {
		target = maxV
	}
	return target
}

// canvasRectWidgets exposes the Canvas region as a widgets.Rect.
func (l *chromeLayout) canvasRectWidgets() widgets.Rect {
	return widgets.Rect{X: l.Canvas.X, Y: l.Canvas.Y, W: l.Canvas.W, H: l.Canvas.H}
}

// leftPanelRectWidgets exposes the LeftPanel region as a widgets.Rect.
func (l *chromeLayout) leftPanelRectWidgets() widgets.Rect {
	return widgets.Rect{X: l.LeftPanel.X, Y: l.LeftPanel.Y, W: l.LeftPanel.W, H: l.LeftPanel.H}
}

// Chrome color palette. Picked once here so the editor's visual identity
// lives in a single place; future theme support reads from this struct.
var (
	colTitleBar    = color.RGBA{R: 0x28, G: 0x28, B: 0x34, A: 0xff}
	colTitleAccent = color.RGBA{R: 0x46, G: 0x86, B: 0xff, A: 0xff}
	colPanel       = color.RGBA{R: 0x1f, G: 0x1f, B: 0x29, A: 0xff}
	colPanelHeader = color.RGBA{R: 0x2a, G: 0x2a, B: 0x36, A: 0xff}
	colCanvasBg    = color.RGBA{R: 0x0d, G: 0x0d, B: 0x14, A: 0xff}
	colDivider     = color.RGBA{R: 0x0a, G: 0x0a, B: 0x10, A: 0xff}
	colStatusBar   = color.RGBA{R: 0x14, G: 0x14, B: 0x1b, A: 0xff}
	colText        = color.RGBA{R: 0xea, G: 0xea, B: 0xf0, A: 0xff}
	colTextDim     = color.RGBA{R: 0x88, G: 0x88, B: 0x95, A: 0xff}
)

// draw paints the chrome onto the supplied target image. The Editor
// passes itself so future panels can read project state directly.
func (l *chromeLayout) draw(dst *ebiten.Image, e *Editor) {
	drawTitleBar(dst, l, e)
	drawLeftPanel(dst, l, e)
	drawRightPanel(dst, l, e)
	drawCenterCanvas(dst, l, e)
	drawTabStrip(dst, l, e)
	drawStatusBar(dst, l, e)

	// Inspector content overlays the right panel header.
	if entity := e.selectedEntity(); entity != nil {
		area := widgets.Rect{
			X: l.RightPanel.X,
			Y: l.RightPanel.Y + 20,
			W: l.RightPanel.W,
			H: l.RightPanel.H - 20,
		}
		if e.inspector.Draw(dst, area, e.project, entity) {
			e.MarkDirty()
		}
	}

	// Asset browser receives input + paints over the left panel body.
	if e.assetBrowser != nil {
		area := widgets.Rect{
			X: l.LeftPanel.X,
			Y: l.LeftPanel.Y + 20,
			W: l.LeftPanel.W,
			H: l.LeftPanel.H - 20,
		}
		e.assetBrowser.Draw(dst, area, e)
		e.assetBrowser.Update(area, e)
	}
}

func drawTitleBar(dst *ebiten.Image, l *chromeLayout, e *Editor) {
	r := l.TitleBar
	if r.W <= 0 || r.H <= 0 {
		return
	}
	fillRect(dst, r, colTitleBar)
	vector.DrawFilledRect(dst, float32(r.X), float32(r.Y+r.H-2), float32(r.W), 2, colTitleAccent, false)
	if r.H >= debugLineHeight {
		title := "Pixelforge Studio  v" + Version
		if e != nil && e.project != nil {
			title += " — " + e.project.Name
			if e.IsDirty() {
				title += " *"
			}
		}
		printAt(dst, title, r.X+12, r.Y+(r.H-debugLineHeight)/2)
	}
}

func drawLeftPanel(dst *ebiten.Image, l *chromeLayout, _ *Editor) {
	r := l.LeftPanel
	if r.W <= 0 || r.H <= 0 {
		return
	}
	fillRect(dst, r, colPanel)
	drawPanelHeader(dst, r, "ASSET BROWSER")
	drawDividerRight(dst, r)
}

func drawRightPanel(dst *ebiten.Image, l *chromeLayout, _ *Editor) {
	r := l.RightPanel
	if r.W <= 0 || r.H <= 0 {
		return
	}
	fillRect(dst, r, colPanel)
	drawPanelHeader(dst, r, "INSPECTOR")
	drawDividerLeft(dst, r)
}

func drawCenterCanvas(dst *ebiten.Image, l *chromeLayout, e *Editor) {
	r := l.Canvas
	if r.W <= 0 || r.H <= 0 {
		return
	}
	fillRect(dst, r, colCanvasBg)
	if e == nil {
		return
	}
	// Delegate to the active workspace; if none registered, render the
	// placeholder text the M0 chrome used.
	if w := e.activeWorkspaceImpl(); w != nil {
		w.Draw(dst, widgets.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H}, e)
		return
	}
	if r.H >= debugLineHeight*2 {
		label := "CANVAS"
		x := r.X + (r.W-len(label)*6)/2
		y := r.Y + (r.H-debugLineHeight)/2
		printAtColored(dst, label, x, y, colTextDim)
	}
}

func drawStatusBar(dst *ebiten.Image, l *chromeLayout, e *Editor) {
	r := l.StatusBar
	if r.W <= 0 || r.H <= 0 {
		return
	}
	fillRect(dst, r, colStatusBar)
	if r.H < debugLineHeight {
		return
	}
	y := r.Y + (r.H-debugLineHeight)/2
	left := "READY"
	if e != nil {
		entityCount := 0
		if s := e.activeScene(); s != nil {
			entityCount = len(s.Entities)
		}
		spriteLabel := e.SelectedSpriteName()
		if spriteLabel == "" {
			spriteLabel = "(none)"
		}
		left = "tool=" + e.Tool().String() + "  sprite=" + spriteLabel + "  entities=" + itoa(entityCount)
		if e.StatusMessage() != "" {
			left += "  | " + e.StatusMessage()
		}
	}
	printAtColored(dst, left, r.X+8, y, colTextDim)

	if e != nil && e.IsDirty() {
		dirty := "* unsaved"
		x := r.X + r.W - len(dirty)*6 - 8
		printAtColored(dst, dirty, x, y, colWarning)
	}

	// U33: chrome-hidden hint in the right portion of the status bar.
	if e != nil && e.ChromeHidden() {
		hint := "chrome hidden - press Esc to restore"
		printAtColored(dst, hint, r.X+r.W/2-len(hint)*3, y, colWarning)
	}
}

func drawTabStrip(dst *ebiten.Image, l *chromeLayout, e *Editor) {
	r := l.TabStrip
	if r.W <= 0 || r.H <= 0 {
		return
	}
	if e == nil {
		return
	}
	area := widgets.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H}
	e.drawTabStrip(dst, area)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		e.handleTabStripClick(area, mx, my)
	}
}

// drawPanelHeader paints a small header strip at the top of a panel.
func drawPanelHeader(dst *ebiten.Image, r rect, label string) {
	if r.H < debugLineHeight+4 {
		return
	}
	header := rect{X: r.X, Y: r.Y, W: r.W, H: debugLineHeight + 4}
	fillRect(dst, header, colPanelHeader)
	printAtColored(dst, label, r.X+8, r.Y+2, colText)
}

func drawDividerLeft(dst *ebiten.Image, r rect) {
	vector.DrawFilledRect(dst, float32(r.X), float32(r.Y), 1, float32(r.H), colDivider, false)
}

func drawDividerRight(dst *ebiten.Image, r rect) {
	vector.DrawFilledRect(dst, float32(r.X+r.W-1), float32(r.Y), 1, float32(r.H), colDivider, false)
}

// fillRect is a thin convenience wrapper around vector.DrawFilledRect.
func fillRect(dst *ebiten.Image, r rect, c color.RGBA) {
	vector.DrawFilledRect(dst, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), c, false)
}

// strokeRectAt strokes a 1px outline around the supplied widgets.Rect.
func strokeRectAt(dst *ebiten.Image, r widgets.Rect, c color.RGBA) {
	vector.StrokeRect(dst, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), 1, c, false)
}

// colWarning is the orange dirty-state color (shared with widgets).
var colWarning = color.RGBA{R: 0xff, G: 0x9d, B: 0x40, A: 0xff}

// printAtColored draws coloured debug text. ebitenutil.DebugPrintAt only
// supports a single colour, so to colourise we render to a tiny scratch
// image and tint it via DrawImage's ColorScale.
func printAtColored(dst *ebiten.Image, s string, x, y int, c color.RGBA) {
	if s == "" {
		return
	}
	w := len(s) * 6
	if w < 1 {
		w = 1
	}
	tmp := ebiten.NewImage(w, debugLineHeight)
	printAt(tmp, s, 0, 0)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(c)
	dst.DrawImage(tmp, op)
}
