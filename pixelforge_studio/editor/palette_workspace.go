// palette_workspace.go owns idea #3 v1 U5's NES-style palette
// workspace. The workspace renders two clusters in the editor's
// docked layout:
//
//  1. An 8x8 grid of the project's 64 base palette colors. Click
//     any slot to open ColorEdit3; the edit mutates
//     Palette.Base[i] and the engine's indexed renderer cascades
//     the change into the scene preview within one frame (R5).
//  2. Eight rows for the BG + Sprite sub-palettes. Each row shows
//     the editable name + four slot swatches; designers re-bind
//     swatches by entering a slot index (drag-drop is a future v2
//     UX polish).
//
// State-mutating helpers (SetBaseColor, RenameSubPalette,
// AssignSubPaletteSlot) live as standalone functions so tests can
// exercise the mutation contract without driving cimgui-go. Render
// is thin: it walks the project, paints widgets, and dispatches
// clicks back through the helpers.
package editor

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor/widgets"
)

// paletteWorkspaceColumns is the per-row count of base-slot
// buttons. 8 columns over 64 slots gives the canonical 8x8 grid.
const paletteWorkspaceColumns = 8

// PaletteWorkspace renders the NES-style palette editor. Implements
// the Workspace interface so the docked layout treats it like the
// Scene / Inspector / Assets panels.
type PaletteWorkspace struct {
	// editingSlot tracks which base slot's ColorEdit3 popup is
	// open; -1 means none. Persistent across frames so the popup
	// survives ImGui's stateless dispatch.
	editingSlot int
}

// NewPaletteWorkspace returns a fresh workspace bound for use with
// editor.RegisterWorkspace.
func NewPaletteWorkspace() *PaletteWorkspace {
	return &PaletteWorkspace{editingSlot: -1}
}

// Name implements editor.Workspace.
func (w *PaletteWorkspace) Name() string { return "nes_palette" }

// DisplayName implements editor.Workspace — the docked tab label.
func (w *PaletteWorkspace) DisplayName() string { return "NES Palette" }

// Render emits the workspace inside the current ImGui frame.
// Skipped when no live ImGui backend is attached so unit tests that
// build a bare Editor don't crash on cgo dispatch.
func (w *PaletteWorkspace) Render(e *Editor) {
	if e == nil || e.imgui == nil || !e.imgui.live {
		return
	}
	if !imgui.Begin(w.DisplayName()) {
		imgui.End()
		return
	}
	defer imgui.End()
	w.renderBaseGrid(e)
	imgui.Separator()
	w.renderSubPaletteRows(e)
}

func (w *PaletteWorkspace) renderBaseGrid(e *Editor) {
	imgui.TextUnformatted("Base palette (64 slots)")
	for i := 0; i < pixelforge_project.MaxColors; i++ {
		if i%paletteWorkspaceColumns != 0 {
			imgui.SameLine()
		}
		label := fmt.Sprintf("##base_%d", i)
		col := paletteColorToVec4(e.project.Palette.Base[i])
		if imgui.ColorButtonV(label, col, 0, imgui.Vec2{X: 20, Y: 20}) {
			w.editingSlot = i
			imgui.OpenPopupStr("PaletteSlotEditor")
		}
	}
	if w.editingSlot >= 0 && imgui.BeginPopup("PaletteSlotEditor") {
		current := paletteColorToArr3(e.project.Palette.Base[w.editingSlot])
		if imgui.ColorEdit3V(fmt.Sprintf("Slot %d", w.editingSlot), &current, 0) {
			SetBaseColor(e.project, w.editingSlot, arr3ToHex(current))
			e.MarkDirty()
		}
		imgui.EndPopup()
	}
}

func (w *PaletteWorkspace) renderSubPaletteRows(e *Editor) {
	imgui.TextUnformatted("BG sub-palettes")
	for i := range e.project.Palette.BGSubPalettes {
		w.renderSubPaletteRow(e, &e.project.Palette.BGSubPalettes[i], "bg", i)
	}
	imgui.Separator()
	imgui.TextUnformatted("Sprite sub-palettes")
	for i := range e.project.Palette.SpriteSubPalettes {
		w.renderSubPaletteRow(e, &e.project.Palette.SpriteSubPalettes[i], "sprite", i)
	}
}

func (w *PaletteWorkspace) renderSubPaletteRow(e *Editor, sp *pixelforge_project.SubPalette, family string, idx int) {
	name := sp.Name
	if imgui.InputTextWithHint(fmt.Sprintf("##name_%s_%d", family, idx), "name", &name, 0, nil) {
		sp.Name = name
		e.MarkDirty()
	}
	for j := range sp.Slots {
		imgui.SameLine()
		col := paletteColorToVec4(e.project.Palette.Base[sp.Slots[j]])
		label := fmt.Sprintf("##swatch_%s_%d_%d", family, idx, j)
		if imgui.ColorButtonV(label, col, 0, imgui.Vec2{X: 18, Y: 18}) {
			// Cycle the slot index forward as a simple v1 picker —
			// the planned drag-drop binding lands in v2.
			next := (sp.Slots[j] + 1) % pixelforge_project.MaxColors
			AssignSubPaletteSlot(e.project, family, idx, j, next)
			e.MarkDirty()
		}
	}
	imgui.SameLine()
	imgui.TextDisabled(fmt.Sprintf("[%d %d %d %d]", sp.Slots[0], sp.Slots[1], sp.Slots[2], sp.Slots[3]))
}

// SetBaseColor mutates Palette.Base[slot] to the supplied "#RRGGBB"
// string. Returns true when the value changed so callers can decide
// to MarkDirty. Out-of-range slots reject silently.
func SetBaseColor(p *pixelforge_project.Project, slot int, hex string) bool {
	if p == nil || slot < 0 || slot >= pixelforge_project.MaxColors {
		return false
	}
	if p.Palette.Base[slot] == hex {
		return false
	}
	p.Palette.Base[slot] = hex
	return true
}

// RenameSubPalette mutates the Name of the family's idx-th sub-
// palette. family must be "bg" or "sprite"; out-of-range family or
// idx rejects silently. Returns true on change.
func RenameSubPalette(p *pixelforge_project.Project, family string, idx int, newName string) bool {
	target := selectSubPaletteSlice(p, family)
	if target == nil || idx < 0 || idx >= len(target) {
		return false
	}
	if target[idx].Name == newName {
		return false
	}
	target[idx].Name = newName
	return true
}

// AssignSubPaletteSlot writes slotIdx into family[paletteIdx].Slots[swatchIdx].
// Clamps slotIdx into [0, MaxColors) silently. Returns true on
// change.
func AssignSubPaletteSlot(p *pixelforge_project.Project, family string, paletteIdx, swatchIdx, slotIdx int) bool {
	target := selectSubPaletteSlice(p, family)
	if target == nil || paletteIdx < 0 || paletteIdx >= len(target) {
		return false
	}
	if swatchIdx < 0 || swatchIdx >= 4 {
		return false
	}
	if slotIdx < 0 {
		slotIdx = 0
	}
	if slotIdx >= pixelforge_project.MaxColors {
		slotIdx = pixelforge_project.MaxColors - 1
	}
	if target[paletteIdx].Slots[swatchIdx] == slotIdx {
		return false
	}
	target[paletteIdx].Slots[swatchIdx] = slotIdx
	return true
}

// selectSubPaletteSlice returns the (mutable) BG or Sprite array
// from the project's palette. Returns nil when family is unknown,
// which the caller treats as a reject.
func selectSubPaletteSlice(p *pixelforge_project.Project, family string) []pixelforge_project.SubPalette {
	if p == nil {
		return nil
	}
	switch family {
	case "bg":
		return p.Palette.BGSubPalettes[:]
	case "sprite":
		return p.Palette.SpriteSubPalettes[:]
	}
	return nil
}

// paletteColorToArr3 / paletteColorToVec4 parse "#RRGGBB" into the
// ImGui [3]float32 / Vec4 the ColorEdit / ColorButton widgets
// accept. Falls back to (0,0,0) on parse failure so a malformed
// slot never crashes the workspace.
func paletteColorToArr3(hex string) [3]float32 {
	r, g, b := parseHexRGB(hex)
	return [3]float32{
		float32(r) / 255,
		float32(g) / 255,
		float32(b) / 255,
	}
}

func paletteColorToVec4(hex string) imgui.Vec4 {
	a := paletteColorToArr3(hex)
	return imgui.Vec4{X: a[0], Y: a[1], Z: a[2], W: 1.0}
}

// arr3ToHex renders an ImGui [3]float32 (0..1 floats) back to
// "#RRGGBB" for storage in Palette.Base.
func arr3ToHex(v [3]float32) string {
	clamp := func(f float32) byte {
		if f < 0 {
			return 0
		}
		if f > 1 {
			return 255
		}
		return byte(f*255 + 0.5)
	}
	r := clamp(v[0])
	g := clamp(v[1])
	b := clamp(v[2])
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// parseHexRGB pulls 3 byte values out of "#RRGGBB". Returns
// (0,0,0) on parse failure so the workspace renders something
// rather than crashing.
func parseHexRGB(s string) (byte, byte, byte) {
	if len(s) != 7 || s[0] != '#' {
		return 0, 0, 0
	}
	hexByte := func(a, b byte) (byte, bool) {
		hv := func(c byte) (byte, bool) {
			switch {
			case c >= '0' && c <= '9':
				return c - '0', true
			case c >= 'a' && c <= 'f':
				return 10 + c - 'a', true
			case c >= 'A' && c <= 'F':
				return 10 + c - 'A', true
			}
			return 0, false
		}
		hi, ok := hv(a)
		if !ok {
			return 0, false
		}
		lo, ok := hv(b)
		if !ok {
			return 0, false
		}
		return hi<<4 | lo, true
	}
	r, okR := hexByte(s[1], s[2])
	g, okG := hexByte(s[3], s[4])
	b, okB := hexByte(s[5], s[6])
	if !okR || !okG || !okB {
		return 0, 0, 0
	}
	return r, g, b
}

// RegisterPaletteWorkspaceWith installs the NES palette workspace
// onto the editor. Mirrors palette.RegisterWith's convention. Kept
// as a function (not editor.init() registration) so the workspace
// only loads when the studio's main.go opts in.
func RegisterPaletteWorkspaceWith(e *Editor) *PaletteWorkspace {
	w := NewPaletteWorkspace()
	e.RegisterWorkspace(w)
	return w
}

// _ = widgets.Context{} keeps the import alive; the workspace
// doesn't read from widgets directly but the file's neighbours do.
var _ = widgets.Context{}
