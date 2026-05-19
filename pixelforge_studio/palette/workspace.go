package palette

import (
	"image"
	"image/color"

	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// Workspace is the M2 palette workspace. Implements editor.Workspace.
type Workspace struct {
	grid      *Grid
	matrix    *Matrix
	presets   *PresetStack
	animator  *Animator
	abWipe    bool
}

// NewWorkspace constructs the palette workspace fresh.
func NewWorkspace() *Workspace {
	w := &Workspace{
		grid:     NewGrid(),
		matrix:   NewMatrix(),
		presets:  NewPresetStack(),
		animator: NewAnimator(),
	}
	w.grid.OnAnimate(func(slot int) { w.animator.OpenForSlot(slot) })
	return w
}

// Name is the stable workspace identifier the editor switches on.
func (w *Workspace) Name() string { return "palette" }

// DisplayName is the dock window title.
func (w *Workspace) DisplayName() string { return "Palette" }

// Render registers the Palette ImGui window inside the dockspace and
// captures its inner content rect so the native palette grid / matrix /
// presets sub-panels keep rendering through the editor's draw path.
// A full ImGui rewrite of the palette workspace is out of scope for
// the U3 plan; U7/U8-style rebuilds for palette can come later.
func (w *Workspace) Render(e *editor.Editor) {
	if e == nil {
		return
	}
	flags := imgui.WindowFlagsNoBackground | imgui.WindowFlagsNoScrollbar
	if !imgui.BeginV(w.DisplayName(), nil, flags) {
		imgui.End()
		e.SetPanelRect(w.DisplayName(), widgets.Rect{})
		return
	}
	defer imgui.End()
	e.CaptureCurrentWindowRect(w.DisplayName())
}

// Grid exposes the swatch grid for tests.
func (w *Workspace) Grid() *Grid { return w.grid }

// Matrix exposes the ColorTable matrix view.
func (w *Workspace) Matrix() *Matrix { return w.matrix }

// Presets exposes the preset stack.
func (w *Workspace) Presets() *PresetStack { return w.presets }

// Animator exposes the animation timeline popover.
func (w *Workspace) Animator() *Animator { return w.animator }

// Update routes input to the active sub-panel.
func (w *Workspace) Update(e *editor.Editor) {
	// Spacebar = A/B wipe.
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		w.abWipe = true
		e.SetStatusMessage("A/B WIPE")
	}
	if inpututil.IsKeyJustReleased(ebiten.KeySpace) {
		w.abWipe = false
	}

	if e.Project() == nil {
		return
	}
	// Palette renders inside its own dockable window now (U3); use the
	// captured panel rect for the workspace's own DisplayName rather
	// than the scene canvas rect.
	regions := w.layout(e.PanelRect(w.DisplayName()))
	if w.animator.Visible() {
		w.animator.Update(regions.animator, e.Project(), e)
		return
	}
	mx, my := mouseXY()
	if w.grid.PickerOpen() || regions.grid.Contains(mx, my) {
		if w.grid.Update(regions.grid, e.Project(), e) {
			return
		}
	}
	w.matrix.Update(regions.matrix, e.Project(), e)
	w.presets.Update(regions.presets, e.Project(), e)
}

// Draw paints all sub-panels.
func (w *Workspace) Draw(dst *ebiten.Image, area widgets.Rect, e *editor.Editor) {
	if area.W <= 0 || area.H <= 0 {
		return
	}
	p := e.Project()
	if p == nil {
		ebitenutilPrint(dst, "(no project)", area.X+8, area.Y+8)
		return
	}
	regions := w.layout(area)
	vector.DrawFilledRect(dst, float32(area.X), float32(area.Y), float32(area.W), float32(area.H), colWorkspaceBg, false)

	// Section labels.
	ebitenutilPrint(dst, "PALETTE", regions.grid.X+4, regions.grid.Y-12)
	ebitenutilPrint(dst, "COLOR TABLES", regions.matrix.X+4, regions.matrix.Y-12)
	ebitenutilPrint(dst, "PRESETS", regions.presets.X+4, regions.presets.Y-12)

	if w.abWipe {
		// Render the base palette/colortables (no preset composition).
		baseP := *p
		w.grid.Draw(dst, regions.grid, &baseP)
		w.matrix.Draw(dst, regions.matrix, &baseP)
	} else {
		composed := w.presets.Compose(p)
		w.grid.Draw(dst, regions.grid, composed)
		w.matrix.Draw(dst, regions.matrix, composed)
	}
	w.presets.Draw(dst, regions.presets, p)
	if w.animator.Visible() {
		w.animator.Draw(dst, regions.animator, p)
	}

	// Footer / reset button.
	reset := widgets.Rect{X: area.X + area.W - 140, Y: area.Y + area.H - 28, W: 130, H: 22}
	vector.DrawFilledRect(dst, float32(reset.X), float32(reset.Y), float32(reset.W), float32(reset.H), colResetButton, false)
	ebitenutilPrint(dst, "Reset to defaults", reset.X+8, reset.Y+3)
	mx, my := ebiten.CursorPosition()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && reset.Contains(mx, my) {
		e.PromptIfDirty("Reset palette?", "This restores the default palette and identity color tables.", func() {
			p.Palette = pixelforge_project.DefaultPalette()
			e.MarkDirty()
		})
	}
}

// workspaceRegions enumerates the sub-rects sub-panels paint into.
type workspaceRegions struct {
	grid     widgets.Rect
	matrix   widgets.Rect
	presets  widgets.Rect
	animator widgets.Rect
}

func (w *Workspace) layout(area widgets.Rect) workspaceRegions {
	pad := 12
	headerH := 20
	footerH := 36

	// Left two-thirds: grid on top, matrix below.
	leftW := area.W * 2 / 3
	rightW := area.W - leftW - pad
	gridH := (area.H - headerH*2 - footerH - pad*2) * 4 / 10
	matrixH := area.H - headerH*2 - footerH - pad*2 - gridH

	grid := widgets.Rect{X: area.X + pad, Y: area.Y + headerH, W: leftW - pad, H: gridH}
	matrix := widgets.Rect{X: area.X + pad, Y: grid.Y + grid.H + headerH + pad, W: leftW - pad, H: matrixH}
	presets := widgets.Rect{X: area.X + leftW + pad, Y: area.Y + headerH, W: rightW - pad, H: area.H - headerH - footerH - pad}
	animator := widgets.Rect{X: area.X + 32, Y: area.Y + 32, W: area.W - 64, H: area.H - 64}

	return workspaceRegions{grid: grid, matrix: matrix, presets: presets, animator: animator}
}

// mouseXYInRect helper.
func mouseXY() (int, int) { return ebiten.CursorPosition() }

var (
	colWorkspaceBg = color.RGBA{R: 0x0c, G: 0x0c, B: 0x14, A: 0xff}
	colResetButton = color.RGBA{R: 0x44, G: 0x44, B: 0x4c, A: 0xff}
)

// RegisterWith installs the palette workspace on the editor, wires
// the auto-tile rule synth as the AutoTileObserver (idea #2 v1),
// and installs the PNG-import runner (idea #3 v1) so File → Import
// PNG flows through palette.ImportWithDiff. Two interfaces, one
// registration site — the editor stays at the bottom of the import
// graph.
func RegisterWith(e *editor.Editor) *Workspace {
	w := NewWorkspace()
	e.RegisterWorkspace(w)
	e.SetAutoTileObserver(newAutoTileObserverAdapter())
	if h := e.ImportHandler(); h != nil {
		h.SetRunner(newPNGImportRunner(e))
	}
	return w
}

// pngImportRunner bridges editor.ImportHandler to
// palette.ImportWithDiff. Holds a reference to the editor so it can
// reach the active project + ProjectSourcePath at call time
// (designers can save-as mid-session; the runner must re-read each
// call rather than caching).
type pngImportRunner struct {
	editor *editor.Editor
}

func newPNGImportRunner(e *editor.Editor) *pngImportRunner {
	return &pngImportRunner{editor: e}
}

// ImportPath runs the diff-aware pipeline against a PNG on disk.
func (r *pngImportRunner) ImportPath(pngPath string) (*editor.PNGImportResult, error) {
	res, err := ImportWithDiff(
		pngPath,
		r.editor.Project(),
		r.editor.CurrentProjectPath(),
		"",
	)
	if err != nil {
		return nil, err
	}
	return adaptImportResult(&res), nil
}

// ImportImage runs the in-memory pipeline. Used by tests + the
// future drag-drop handler.
func (r *pngImportRunner) ImportImage(img image.Image, pngPath string) (*editor.PNGImportResult, error) {
	res, err := ImportImageWithDiff(
		img,
		pngPath,
		r.editor.Project(),
		r.editor.CurrentProjectPath(),
		"",
	)
	if err != nil {
		return nil, err
	}
	return adaptImportResult(&res), nil
}

// Requantize re-runs the pipeline with a different target sub-
// palette using the cached original image.
func (r *pngImportRunner) Requantize(img image.Image, pngPath, targetSubPalette string) (*editor.PNGImportResult, error) {
	res, err := ImportImageWithDiff(
		img,
		pngPath,
		r.editor.Project(),
		r.editor.CurrentProjectPath(),
		targetSubPalette,
	)
	if err != nil {
		return nil, err
	}
	return adaptImportResult(&res), nil
}

// RollbackLastImport truncates the freshly-appended sprite. The
// editor calls this when the designer picks Reject or Re-quantize.
func (r *pngImportRunner) RollbackLastImport(idx int) {
	p := r.editor.Project()
	if p == nil || idx < 0 {
		return
	}
	if idx >= len(p.Sprites) {
		return
	}
	p.Sprites = p.Sprites[:idx]
}

// adaptImportResult converts a palette.ImportResult to the editor-
// boundary shape. Drops the QuantizedIndices payload because the
// modal only renders the reconstructed preview; tests that need
// indices use the palette package directly.
func adaptImportResult(res *ImportResult) *editor.PNGImportResult {
	out := &editor.PNGImportResult{
		SpriteName:    res.SpriteName,
		RegisteredIdx: res.RegisteredIdx,
		Width:         res.Width,
		Height:        res.Height,
	}
	if res.Diff != nil {
		out.Diff = &editor.PNGImportDiff{
			OriginalImage:          res.Diff.OriginalImage,
			QuantizedReconstructed: res.Diff.QuantizedReconstructed,
			ChosenSubPalette:       res.Diff.ChosenSubPalette,
			AutoPicked:             res.Diff.AutoPicked,
			MeanDeltaE:             res.Diff.MeanDeltaE,
			Width:                  res.Diff.Width,
			Height:                 res.Diff.Height,
		}
	}
	return out
}

// autoTileObserverAdapter wraps palette.AutoTileRuleSynth in the
// editor.AutoTileObserver interface so the canvas dispatch can call
// into the synth without the editor package importing palette
// (which would create a cycle, since palette already imports editor).
type autoTileObserverAdapter struct {
	synth *AutoTileRuleSynth
}

func newAutoTileObserverAdapter() *autoTileObserverAdapter {
	return &autoTileObserverAdapter{synth: NewAutoTileRuleSynth()}
}

// RecordStroke translates the editor's AutoTileCell into the synth's
// PaintCell, runs the observation, and translates promotions back
// across the boundary.
func (a *autoTileObserverAdapter) RecordStroke(
	layer *pixelforge_project.TileAtlas,
	cells []editor.AutoTileCell,
) []editor.AutoTilePromotion {
	if a == nil || a.synth == nil || layer == nil || len(cells) == 0 {
		return nil
	}
	stroke := make([]PaintCell, 0, len(cells))
	for _, c := range cells {
		stroke = append(stroke, PaintCell{X: c.X, Y: c.Y, Value: c.Value})
	}
	raw := a.synth.RecordStrokeWithPromotions(layer, stroke)
	if len(raw) == 0 {
		return nil
	}
	out := make([]editor.AutoTilePromotion, 0, len(raw))
	for _, r := range raw {
		out = append(out, editor.AutoTilePromotion{
			RuleIndex: r.RuleIndex,
			Pattern:   r.Pattern,
			Output:    r.Output,
		})
	}
	return out
}
