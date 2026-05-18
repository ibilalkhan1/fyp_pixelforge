package palette

import (
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

// RegisterWith installs the palette workspace on the editor.
func RegisterWith(e *editor.Editor) *Workspace {
	w := NewWorkspace()
	e.RegisterWorkspace(w)
	return w
}
