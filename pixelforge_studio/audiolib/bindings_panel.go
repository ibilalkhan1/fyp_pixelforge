// bindings_panel.go is idea #4 v1 U7's right-side panel of the
// Audio workspace. Walks Project.Bindings as a table — one row per
// AudioBinding — with Topic dropdown, sound picker, scene/condition
// inputs, and a delete button. Add Binding button appends an empty
// row.
//
// State-mutating operations (AddBinding, DeleteBinding,
// SetBindingTopic, etc.) live as standalone methods so tests can
// exercise the mutation contract without driving cimgui-go. The
// Render method is a thin imgui pass.
package audiolib

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
)

// BindingsPanel renders the bindings table. Carries one piece of
// transient state: which row's "Pick sound..." overlay is open.
// -1 means none open.
type BindingsPanel struct {
	editor      *editor.Editor
	pickerOpen  int
}

// NewBindingsPanel returns a panel bound to editor.
func NewBindingsPanel(e *editor.Editor) *BindingsPanel {
	return &BindingsPanel{editor: e, pickerOpen: -1}
}

// AddBinding appends an empty AudioBinding row. Designer fills in
// fields via the table. Returns the index of the new row so the
// picker (when called from the library Bind path) can scroll to it.
func (b *BindingsPanel) AddBinding() int {
	if b == nil || b.editor == nil || b.editor.Project() == nil {
		return -1
	}
	p := b.editor.Project()
	p.Bindings = append(p.Bindings, pixelforge_project.AudioBinding{})
	b.editor.MarkDirty()
	return len(p.Bindings) - 1
}

// DeleteBinding removes the row at idx. Out-of-range idx is a no-op.
func (b *BindingsPanel) DeleteBinding(idx int) {
	if b == nil || b.editor == nil || b.editor.Project() == nil {
		return
	}
	p := b.editor.Project()
	if idx < 0 || idx >= len(p.Bindings) {
		return
	}
	p.Bindings = append(p.Bindings[:idx], p.Bindings[idx+1:]...)
	b.editor.MarkDirty()
}

// SetBindingTopic updates row idx's Topic. Marks dirty when the
// value actually changed.
func (b *BindingsPanel) SetBindingTopic(idx int, topic string) {
	if !b.indexValid(idx) {
		return
	}
	p := b.editor.Project()
	if p.Bindings[idx].Topic == topic {
		return
	}
	p.Bindings[idx].Topic = topic
	b.editor.MarkDirty()
}

// SetBindingSample updates row idx's SampleName (typically called
// from the sound picker overlay's selection callback).
func (b *BindingsPanel) SetBindingSample(idx int, sampleName string) {
	if !b.indexValid(idx) {
		return
	}
	p := b.editor.Project()
	if p.Bindings[idx].SampleName == sampleName {
		return
	}
	p.Bindings[idx].SampleName = sampleName
	b.editor.MarkDirty()
}

// SetBindingSceneID updates row idx's SceneID restriction.
func (b *BindingsPanel) SetBindingSceneID(idx int, sceneID string) {
	if !b.indexValid(idx) {
		return
	}
	p := b.editor.Project()
	if p.Bindings[idx].SceneID == sceneID {
		return
	}
	p.Bindings[idx].SceneID = sceneID
	b.editor.MarkDirty()
}

// SetBindingCondition updates row idx's TriggerCondition.
func (b *BindingsPanel) SetBindingCondition(idx int, cond string) {
	if !b.indexValid(idx) {
		return
	}
	p := b.editor.Project()
	if p.Bindings[idx].TriggerCondition == cond {
		return
	}
	p.Bindings[idx].TriggerCondition = cond
	b.editor.MarkDirty()
}

// SoundPickerOpen reports whether the sound picker overlay is
// currently open for a row.
func (b *BindingsPanel) SoundPickerOpen() bool { return b != nil && b.pickerOpen >= 0 }

// SoundPickerRow returns the row index the picker is open for, or
// -1 when closed.
func (b *BindingsPanel) SoundPickerRow() int {
	if b == nil {
		return -1
	}
	return b.pickerOpen
}

// OpenSoundPicker shows the sound picker for binding row idx.
func (b *BindingsPanel) OpenSoundPicker(idx int) {
	if !b.indexValid(idx) {
		return
	}
	b.pickerOpen = idx
}

// CloseSoundPicker dismisses the overlay without picking.
func (b *BindingsPanel) CloseSoundPicker() {
	if b == nil {
		return
	}
	b.pickerOpen = -1
}

// PickSampleForOpenRow handles the sound picker's selection: writes
// sampleName to the open row, closes the overlay.
func (b *BindingsPanel) PickSampleForOpenRow(sampleName string) {
	if b == nil || b.pickerOpen < 0 {
		return
	}
	b.SetBindingSample(b.pickerOpen, sampleName)
	b.pickerOpen = -1
}

// AvailableSampleNames returns the names of every AudioSample in
// the project, so the sound picker overlay can render them in a
// flat list with no distinction between library-sourced and user-
// imported (R9).
func (b *BindingsPanel) AvailableSampleNames() []string {
	if b == nil || b.editor == nil || b.editor.Project() == nil {
		return nil
	}
	out := make([]string, 0, len(b.editor.Project().Audio))
	for _, a := range b.editor.Project().Audio {
		out = append(out, a.Name)
	}
	return out
}

// indexValid is the bounds check every Set/Delete uses.
func (b *BindingsPanel) indexValid(idx int) bool {
	if b == nil || b.editor == nil || b.editor.Project() == nil {
		return false
	}
	return idx >= 0 && idx < len(b.editor.Project().Bindings)
}

// Render emits the bindings table inside the current ImGui child.
// The header line carries the row count + the Add Binding button;
// each row gets a Topic / Sound / SceneID / Condition / Delete
// strip.
func (b *BindingsPanel) Render() {
	if !editorImguiLive(b.editor) {
		return
	}
	p := b.editor.Project()
	if p == nil {
		imgui.TextDisabled("(no project)")
		return
	}
	imgui.TextUnformatted(fmt.Sprintf("Bindings (%d)", len(p.Bindings)))
	imgui.SameLine()
	if imgui.Button("Add Binding") {
		b.AddBinding()
	}
	imgui.Separator()
	for i := range p.Bindings {
		b.renderRow(i)
	}
	if b.SoundPickerOpen() {
		b.renderSoundPicker()
	}
}

func (b *BindingsPanel) renderRow(idx int) {
	p := b.editor.Project()
	binding := &p.Bindings[idx]
	imgui.PushIDInt(int32(idx))
	defer imgui.PopID()

	// Topic input — combo would be richer (sources from pievent +
	// project topics) but a free-text input keeps the UI honest
	// without bringing in the pievent dependency from this file.
	// Future polish: swap to BeginCombo per the plan's U7 vision.
	topic := binding.Topic
	if imgui.InputTextWithHint("Topic", "(none)", &topic, 0, nil) {
		b.SetBindingTopic(idx, topic)
	}
	// Sound picker
	sound := binding.SampleName
	if sound == "" {
		sound = "(pick sound)"
	}
	imgui.SameLine()
	if imgui.Button(sound + "##sound") {
		b.OpenSoundPicker(idx)
	}
	// SceneID restriction
	sceneID := binding.SceneID
	imgui.SameLine()
	if imgui.InputTextWithHint("Scene", "(all)", &sceneID, 0, nil) {
		b.SetBindingSceneID(idx, sceneID)
	}
	// Condition
	cond := binding.TriggerCondition
	imgui.SameLine()
	if imgui.InputTextWithHint("Cond", "(always)", &cond, 0, nil) {
		b.SetBindingCondition(idx, cond)
	}
	imgui.SameLine()
	if imgui.Button("Delete") {
		b.DeleteBinding(idx)
	}
}

func (b *BindingsPanel) renderSoundPicker() {
	popupID := "##audiolib_sound_picker"
	imgui.OpenPopupStr(popupID)
	if !imgui.BeginPopup(popupID) {
		return
	}
	defer imgui.EndPopup()
	imgui.TextUnformatted("Pick sound for binding")
	imgui.Separator()
	for _, name := range b.AvailableSampleNames() {
		if imgui.SelectableBool(name) {
			b.PickSampleForOpenRow(name)
			return
		}
	}
	imgui.Separator()
	if imgui.Button("Cancel") {
		b.CloseSoundPicker()
	}
}
