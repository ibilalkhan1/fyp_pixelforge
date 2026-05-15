package editor

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
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
	defaultTitleBarH  = 40
	defaultLeftPanelW = 220
	defaultRightPanelW = 260
	defaultStatusBarH = 24

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

	TitleBarH  int
	LeftPanelW int
	RightPanelW int
	StatusBarH int

	TitleBar   rect
	LeftPanel  rect
	RightPanel rect
	Canvas     rect
	StatusBar  rect
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

	// Clamp horizontal panels so a minimum canvas survives. We shrink
	// left and right symmetrically when needed.
	titleH := clampMax(l.TitleBarH, minTitleBarH, h/3)
	statusH := clampMax(l.StatusBarH, minStatusBarH, h/4)
	if titleH+statusH+minCanvasH > h {
		// canvas height vanishes; let chrome carve up the whole window
		titleH = h / 2
		statusH = h - titleH
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
		// canvas width vanishes; split horizontal space evenly
		leftW = w / 2
		rightW = w - leftW
		if leftW < 0 {
			leftW = 0
		}
		if rightW < 0 {
			rightW = 0
		}
	}

	l.TitleBar = rect{X: 0, Y: 0, W: w, H: titleH}

	bodyY := titleH
	bodyH := h - titleH - statusH
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
func (l *chromeLayout) draw(dst *ebiten.Image, _ *Editor) {
	drawTitleBar(dst, l)
	drawLeftPanel(dst, l)
	drawRightPanel(dst, l)
	drawCenterCanvas(dst, l)
	drawStatusBar(dst, l)
}

func drawTitleBar(dst *ebiten.Image, l *chromeLayout) {
	r := l.TitleBar
	if r.W <= 0 || r.H <= 0 {
		return
	}
	fillRect(dst, r, colTitleBar)
	// Accent stripe across the bottom edge of the title bar.
	vector.DrawFilledRect(dst, float32(r.X), float32(r.Y+r.H-2), float32(r.W), 2, colTitleAccent, false)
	if r.H >= debugLineHeight {
		printAt(dst, "Pixelforge Studio  v"+Version, r.X+12, r.Y+(r.H-debugLineHeight)/2)
	}
}

func drawLeftPanel(dst *ebiten.Image, l *chromeLayout) {
	r := l.LeftPanel
	if r.W <= 0 || r.H <= 0 {
		return
	}
	fillRect(dst, r, colPanel)
	drawPanelHeader(dst, r, "ASSET BROWSER")
	drawDividerRight(dst, r)
}

func drawRightPanel(dst *ebiten.Image, l *chromeLayout) {
	r := l.RightPanel
	if r.W <= 0 || r.H <= 0 {
		return
	}
	fillRect(dst, r, colPanel)
	drawPanelHeader(dst, r, "INSPECTOR")
	drawDividerLeft(dst, r)
}

func drawCenterCanvas(dst *ebiten.Image, l *chromeLayout) {
	r := l.Canvas
	if r.W <= 0 || r.H <= 0 {
		return
	}
	fillRect(dst, r, colCanvasBg)
	if r.H >= debugLineHeight*2 {
		// Center the placeholder text vertically; horizontal nudge is
		// approximate (debug font is roughly 6px per glyph).
		label := "CANVAS"
		x := r.X + (r.W-len(label)*6)/2
		y := r.Y + (r.H-debugLineHeight)/2
		printAtColored(dst, label, x, y, colTextDim)
	}
}

func drawStatusBar(dst *ebiten.Image, l *chromeLayout) {
	r := l.StatusBar
	if r.W <= 0 || r.H <= 0 {
		return
	}
	fillRect(dst, r, colStatusBar)
	if r.H >= debugLineHeight {
		printAtColored(dst, "READY", r.X+8, r.Y+(r.H-debugLineHeight)/2, colTextDim)
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

// printAtColored draws coloured debug text. ebitenutil.DebugPrintAt only
// supports a single colour, so to colourise we render to a tiny scratch
// image and tint it via DrawImage's ColorScale.
func printAtColored(dst *ebiten.Image, s string, x, y int, c color.RGBA) {
	if s == "" {
		return
	}
	// Approximate text width (6 px per glyph) so the scratch image is
	// just big enough. DebugPrintAt is the only safe call inside the
	// loop — Ebitengine reuses a cached glyph atlas, so this stays fast.
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
