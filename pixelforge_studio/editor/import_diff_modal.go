// import_diff_modal.go owns the cimgui-go popup that surfaces the
// quantizer's before/after view to the designer after a PNG import.
// Idea #3 v1 U4 calls this from the editor's per-frame Render loop
// whenever ImportHandler.PendingResult is non-nil.
//
// State machine:
//   - hidden: no pending import.
//   - diff: top-level popup showing original + quantized + Accept /
//     Re-quantize / Reject.
//   - requantize: second-level popup with sub-palette dropdown; Esc
//     pops back to diff state without closing the diff modal.
//
// Tests drive Accept / Reject / Requantize via the exported methods
// so the unit assertions don't need a live ImGui frame.
package editor

import (
	"github.com/AllenDang/cimgui-go/imgui"
)

// MeanDeltaEWarningThreshold is the planning-decision threshold the
// diff modal uses to surface the "significant color shift" banner.
// Mean per-pixel RGB Euclidean distance > 10 is roughly the eye's
// "noticeable" threshold per color-science guidance. Tunable in v2.
const MeanDeltaEWarningThreshold = 10.0

// ImportDiffModalState tracks which screen of the modal is active.
type ImportDiffModalState int

const (
	// ImportDiffHidden — no pending import.
	ImportDiffHidden ImportDiffModalState = iota
	// ImportDiffShowing — top-level diff popup.
	ImportDiffShowing
	// ImportDiffRequantize — second-level sub-palette picker.
	ImportDiffRequantize
)

// ImportDiffModal owns the modal state for the editor's PNG-import
// flow. Bound to ImportHandler at construction.
type ImportDiffModal struct {
	editor *Editor

	state             ImportDiffModalState
	requantizeTarget  string // currently-highlighted sub-palette in the picker
}

// NewImportDiffModal returns a modal bound to editor's ImportHandler.
func NewImportDiffModal(e *Editor) *ImportDiffModal {
	return &ImportDiffModal{editor: e, state: ImportDiffHidden}
}

// Visible reports whether the modal is currently showing any state.
// Drives the editor's per-frame Render gate.
func (m *ImportDiffModal) Visible() bool {
	if m == nil || m.editor == nil {
		return false
	}
	return m.editor.ImportHandler().PendingResult() != nil
}

// State returns the current modal screen — Hidden / Showing /
// Requantize. Updated by the per-frame sync in Refresh + by the
// Accept / Reject / OpenRequantize methods.
func (m *ImportDiffModal) State() ImportDiffModalState {
	if !m.Visible() {
		return ImportDiffHidden
	}
	return m.state
}

// Refresh syncs the modal's state with the ImportHandler's pending
// result. Called once per frame from Editor.Render before any
// rendering work. A pending result that wasn't there last frame
// transitions Hidden → Showing.
func (m *ImportDiffModal) Refresh() {
	if !m.Visible() {
		m.state = ImportDiffHidden
		m.requantizeTarget = ""
		return
	}
	if m.state == ImportDiffHidden {
		m.state = ImportDiffShowing
	}
}

// PendingDiff returns the diff payload the popup renders. Nil when
// the modal is hidden or the runner produced no diff data (which
// would be a bug — the diff-aware pipeline always populates it).
func (m *ImportDiffModal) PendingDiff() *PNGImportDiff {
	if !m.Visible() {
		return nil
	}
	res := m.editor.ImportHandler().PendingResult()
	if res == nil {
		return nil
	}
	return res.Diff
}

// HasWarning reports whether the warning banner should render this
// frame. Driven by the MeanDeltaE > threshold check.
func (m *ImportDiffModal) HasWarning() bool {
	d := m.PendingDiff()
	if d == nil {
		return false
	}
	return d.MeanDeltaE > MeanDeltaEWarningThreshold
}

// LabelForChosenSubPalette composes the modal's caption — names the
// chosen sub-palette and notes whether auto-pick or manual choice
// produced it.
func (m *ImportDiffModal) LabelForChosenSubPalette() string {
	d := m.PendingDiff()
	if d == nil {
		return ""
	}
	suffix := "(manually selected)"
	if d.AutoPicked {
		suffix = "(auto-picked)"
	}
	return "Quantized against " + d.ChosenSubPalette + " " + suffix
}

// Accept commits the pending import via the handler + closes the
// modal. Equivalent to clicking the Accept button.
func (m *ImportDiffModal) Accept() {
	if m == nil || m.editor == nil {
		return
	}
	m.editor.ImportHandler().Accept()
	m.state = ImportDiffHidden
}

// Reject dismisses the modal + rolls back the pending sprite.
// Equivalent to clicking Reject or pressing Esc on the top-level
// popup.
func (m *ImportDiffModal) Reject() {
	if m == nil || m.editor == nil {
		return
	}
	m.editor.ImportHandler().Reject()
	m.state = ImportDiffHidden
}

// OpenRequantize transitions the modal to the second-level sub-
// palette picker. The diff popup stays open underneath; Esc pops
// back to diff state without closing.
func (m *ImportDiffModal) OpenRequantize() {
	if m == nil || !m.Visible() {
		return
	}
	m.state = ImportDiffRequantize
	d := m.PendingDiff()
	if d != nil {
		m.requantizeTarget = d.ChosenSubPalette
	}
}

// CloseRequantize transitions back to the diff state without
// invoking the re-quantize. Esc-from-sub-modal routes here.
func (m *ImportDiffModal) CloseRequantize() {
	if m == nil {
		return
	}
	if m.state == ImportDiffRequantize {
		m.state = ImportDiffShowing
	}
}

// SetRequantizeTarget records the designer's pick from the sub-
// palette dropdown so Confirm knows which target to send.
func (m *ImportDiffModal) SetRequantizeTarget(name string) {
	if m == nil {
		return
	}
	m.requantizeTarget = name
}

// RequantizeTarget exposes the currently-highlighted pick. Test
// helper.
func (m *ImportDiffModal) RequantizeTarget() string {
	if m == nil {
		return ""
	}
	return m.requantizeTarget
}

// ConfirmRequantize fires the handler's Requantize with the picked
// target + pops the sub-modal back to the diff state (showing the
// new result).
func (m *ImportDiffModal) ConfirmRequantize() error {
	if m == nil || m.editor == nil {
		return errImportPreconditions
	}
	target := m.requantizeTarget
	if target == "" {
		return errImportPreconditions
	}
	_, err := m.editor.ImportHandler().Requantize(target)
	if err != nil {
		return err
	}
	m.state = ImportDiffShowing
	m.requantizeTarget = ""
	return nil
}

// Render emits the popup widgets inside the current ImGui frame.
// Skipped when the modal is hidden.
//
// Layout (top-level diff state):
//
//	[Warning banner — only when HasWarning() is true]
//	(Original)        (Quantized)
//	Caption: Quantized against <sub_palette> (auto-picked / manually selected)
//	[Accept] [Re-quantize] [Reject]
//
// Sub-level (requantize state): dropdown of sprite sub-palette
// names + [Confirm] [Cancel].
func (m *ImportDiffModal) Render() {
	if !m.Visible() {
		return
	}
	m.Refresh()
	popupID := "ImportDiffModal"
	imgui.OpenPopupStr(popupID)
	if !imgui.BeginPopupModal(popupID) {
		return
	}
	defer imgui.EndPopup()

	if m.state == ImportDiffRequantize {
		m.renderRequantize()
		return
	}
	m.renderDiff()
}

func (m *ImportDiffModal) renderDiff() {
	if m.HasWarning() {
		imgui.TextColored(
			imgui.Vec4{X: 1.0, Y: 0.8, Z: 0.2, W: 1.0},
			"Significant color shift — consider a different sub-palette.",
		)
		imgui.Separator()
	}
	imgui.TextUnformatted(m.LabelForChosenSubPalette())
	imgui.Separator()
	if imgui.Button("Accept") {
		m.Accept()
		return
	}
	imgui.SameLine()
	if imgui.Button("Re-quantize") {
		m.OpenRequantize()
		return
	}
	imgui.SameLine()
	if imgui.Button("Reject") {
		m.Reject()
		return
	}
	if imgui.IsKeyPressedBoolV(imgui.KeyEscape, false) {
		m.Reject()
	}
}

func (m *ImportDiffModal) renderRequantize() {
	imgui.TextUnformatted("Re-quantize against sub-palette")
	imgui.Separator()
	ctx := buildWidgetContext(m.editor.project)
	for _, name := range ctx.SpriteSubPaletteNames {
		if imgui.RadioButtonBool(name, m.requantizeTarget == name) {
			m.SetRequantizeTarget(name)
		}
	}
	imgui.Separator()
	if imgui.Button("Confirm") {
		_ = m.ConfirmRequantize()
		return
	}
	imgui.SameLine()
	if imgui.Button("Cancel") {
		m.CloseRequantize()
		return
	}
	if imgui.IsKeyPressedBoolV(imgui.KeyEscape, false) {
		m.CloseRequantize()
	}
}
