package audiolib

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_audio"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
)

// workspace_test.go covers U6's picker logic (filter, Play, Bind)
// + U7's bindings panel state mutations. The imgui-driven Render
// path is exercised structurally via TestWorkspace_RegisteredOnEditor;
// the state contracts are tested without driving cimgui-go.

func newAudioEditor(t *testing.T) *editor.Editor {
	t.Helper()
	e := editor.New()
	require.NotNil(t, e)
	return e
}

func TestWorkspace_RegisteredOnEditor(t *testing.T) {
	resetForTest()
	e := newAudioEditor(t)
	w := RegisterWith(e)
	require.NotNil(t, w)
	assert.Equal(t, "audio", w.Name())
	assert.Equal(t, "Audio", w.DisplayName())
	// Editor's workspace registry now contains it.
	found := false
	for _, ws := range e.Workspaces() {
		if ws.Name() == "audio" {
			found = true
			break
		}
	}
	assert.True(t, found, "Audio workspace registered with the editor")
}

func TestPickerPanel_FilterByCategorySubstring(t *testing.T) {
	resetForTest()
	e := newAudioEditor(t)
	w := RegisterWith(e)
	p := w.Picker()
	p.SetFilter("jump")
	got := p.FilteredPatches()
	require.NotEmpty(t, got)
	for _, patch := range got {
		assert.Contains(t, patch.Name+patch.Category, "jump")
	}
}

func TestPickerPanel_FilterEmptyShowsAll(t *testing.T) {
	resetForTest()
	e := newAudioEditor(t)
	w := RegisterWith(e)
	p := w.Picker()
	all, _ := LoadCatalog()
	assert.Equal(t, len(all), len(p.FilteredPatches()))
}

func TestPickerPanel_FilterBGMShowsBGMPatches(t *testing.T) {
	resetForTest()
	e := newAudioEditor(t)
	w := RegisterWith(e)
	p := w.Picker()
	p.SetFilter("bgm")
	got := p.FilteredPatches()
	require.NotEmpty(t, got)
	for _, patch := range got {
		assert.True(t, patch.IsBGM,
			"bgm filter returns only BGM patches: %s", patch.Name)
	}
}

func TestPickerPanel_HandlePlayInvokesAudition(t *testing.T) {
	withRecordingBackend(t)
	resetForTest()
	e := newAudioEditor(t)
	w := NewWorkspace(e)
	w.audition = NewAudition(pixelforge_audio.NewAllocator())
	w.picker = NewPickerPanel(e, w.audition)

	patches, _ := LoadCatalog()
	require.NotEmpty(t, patches)
	ch, err := w.picker.HandlePlay(patches[0])
	require.NoError(t, err)
	assert.NotEqual(t, pixelforge_audio.Chan(0), ch)
	assert.True(t, w.audition.IsActive(patches[0].Name))
}

func TestPickerPanel_HandlePlayTwiceStops(t *testing.T) {
	withRecordingBackend(t)
	resetForTest()
	e := newAudioEditor(t)
	w := NewWorkspace(e)
	w.audition = NewAudition(pixelforge_audio.NewAllocator())
	w.picker = NewPickerPanel(e, w.audition)

	patches, _ := LoadCatalog()
	_, _ = w.picker.HandlePlay(patches[0])
	require.True(t, w.audition.IsActive(patches[0].Name))
	_, _ = w.picker.HandlePlay(patches[0])
	assert.False(t, w.audition.IsActive(patches[0].Name),
		"second Play on same patch stops it")
}

func TestPickerPanel_HandleBindImportsAndAppendsBindingRow(t *testing.T) {
	resetForTest()
	e := newAudioEditor(t)
	w := RegisterWith(e)
	p := w.Picker()

	patches, _ := LoadCatalog()
	require.NotEmpty(t, patches)
	require.NoError(t, p.HandleBind(patches[0]))

	proj := e.Project()
	require.Len(t, proj.Audio, 1, "AudioSample appended on Bind")
	require.Len(t, proj.Bindings, 1, "fresh AudioBinding row appended on Bind")
	assert.Equal(t, proj.Audio[0].Name, proj.Bindings[0].SampleName)
	assert.Empty(t, proj.Bindings[0].Topic,
		"new binding has empty Topic — designer fills it in next")
	assert.True(t, e.IsDirty())
}

func TestPickerPanel_HandleBindReturnsErrorOnImportFailure(t *testing.T) {
	resetForTest()
	p := NewPickerPanel(nil, nil)
	err := p.HandleBind(LibraryPatch{Name: "nope"})
	require.Error(t, err)
}

func TestBindingsPanel_AddBindingAppendsEmptyRow(t *testing.T) {
	resetForTest()
	e := newAudioEditor(t)
	w := RegisterWith(e)
	b := w.Bindings()
	pre := len(e.Project().Bindings)
	idx := b.AddBinding()
	assert.Equal(t, pre, idx)
	assert.Len(t, e.Project().Bindings, pre+1)
	assert.True(t, e.IsDirty())
}

func TestBindingsPanel_DeleteBindingRemovesRow(t *testing.T) {
	resetForTest()
	e := newAudioEditor(t)
	w := RegisterWith(e)
	b := w.Bindings()
	b.AddBinding()
	b.AddBinding()
	b.AddBinding()
	b.DeleteBinding(1)
	assert.Len(t, e.Project().Bindings, 2)
}

func TestBindingsPanel_DeleteOutOfRangeIsNoOp(t *testing.T) {
	resetForTest()
	e := newAudioEditor(t)
	w := RegisterWith(e)
	b := w.Bindings()
	b.AddBinding()
	b.DeleteBinding(99)
	b.DeleteBinding(-1)
	assert.Len(t, e.Project().Bindings, 1)
}

func TestBindingsPanel_SetBindingFieldsMutateAndMarkDirty(t *testing.T) {
	resetForTest()
	e := newAudioEditor(t)
	w := RegisterWith(e)
	b := w.Bindings()
	idx := b.AddBinding()
	e.ClearDirty()

	b.SetBindingTopic(idx, "game/PlayerJumped")
	b.SetBindingSample(idx, "jump_spring")
	b.SetBindingSceneID(idx, "level1")
	b.SetBindingCondition(idx, "PlayerSpeed > 0")

	binding := e.Project().Bindings[idx]
	assert.Equal(t, "game/PlayerJumped", binding.Topic)
	assert.Equal(t, "jump_spring", binding.SampleName)
	assert.Equal(t, "level1", binding.SceneID)
	assert.Equal(t, "PlayerSpeed > 0", binding.TriggerCondition)
	assert.True(t, e.IsDirty())
}

func TestBindingsPanel_SetSameValueIsNoOp(t *testing.T) {
	resetForTest()
	e := newAudioEditor(t)
	w := RegisterWith(e)
	b := w.Bindings()
	idx := b.AddBinding()
	b.SetBindingTopic(idx, "x")
	e.ClearDirty()
	b.SetBindingTopic(idx, "x")
	assert.False(t, e.IsDirty(), "setting same value does not mark dirty")
}

func TestBindingsPanel_SoundPickerOpenCloseFlow(t *testing.T) {
	resetForTest()
	e := newAudioEditor(t)
	w := RegisterWith(e)
	b := w.Bindings()
	idx := b.AddBinding()

	assert.False(t, b.SoundPickerOpen())
	b.OpenSoundPicker(idx)
	assert.True(t, b.SoundPickerOpen())
	assert.Equal(t, idx, b.SoundPickerRow())

	b.CloseSoundPicker()
	assert.False(t, b.SoundPickerOpen())
}

func TestBindingsPanel_PickSampleForOpenRowSetsAndCloses(t *testing.T) {
	resetForTest()
	e := newAudioEditor(t)
	w := RegisterWith(e)
	b := w.Bindings()
	idx := b.AddBinding()
	b.OpenSoundPicker(idx)
	b.PickSampleForOpenRow("jump_spring")
	assert.Equal(t, "jump_spring", e.Project().Bindings[idx].SampleName)
	assert.False(t, b.SoundPickerOpen(), "picker closes after selection")
}

func TestBindingsPanel_AvailableSampleNamesEnumeratesProjectAudio(t *testing.T) {
	resetForTest()
	e := newAudioEditor(t)
	w := RegisterWith(e)
	b := w.Bindings()

	// Import two patches so the project has two AudioSamples.
	_, _ = ImportFromLibrary("jump/spring", e.Project(), "")
	_, _ = ImportFromLibrary("bgm/title/intro_loop", e.Project(), "")

	got := b.AvailableSampleNames()
	assert.ElementsMatch(t, []string{"jump_spring", "bgm_title_intro_loop"}, got)
}
