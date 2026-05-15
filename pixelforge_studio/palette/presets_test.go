package palette

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// Add appends a preset and marks it active.
func TestPresetStack_AddMakesActive(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	s := NewPresetStack()
	idx := s.Add(p, "Dawn")
	require.Equal(t, 0, idx)
	assert.True(t, s.IsActive(idx))
	assert.Equal(t, "Dawn", p.Palette.Presets[idx].Name)
}

// Toggle flips the active flag.
func TestPresetStack_Toggle(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	s := NewPresetStack()
	idx := s.Add(p, "Dawn")
	assert.True(t, s.IsActive(idx))
	s.Toggle(idx)
	assert.False(t, s.IsActive(idx))
	s.Toggle(idx)
	assert.True(t, s.IsActive(idx))
}

// Compose with an active preset overrides the matching slot.
func TestPresetStack_ComposeAppliesOverrides(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	s := NewPresetStack()
	idx := s.Add(p, "Dawn")
	p.Palette.Presets[idx].PaletteOverrides[8] = "#ff0000"

	out := s.Compose(p)
	assert.Equal(t, "#ff0000", out.Palette.Base[8])
	// Base palette unchanged.
	assert.NotEqual(t, "#ff0000", p.Palette.Base[8])
}

// Toggling an active preset off restores the prior state.
func TestPresetStack_ToggleRestoresBase(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	s := NewPresetStack()
	idx := s.Add(p, "Dawn")
	p.Palette.Presets[idx].PaletteOverrides[8] = "#ff0000"

	s.Toggle(idx)
	out := s.Compose(p)
	assert.Equal(t, p.Palette.Base[8], out.Palette.Base[8],
		"inactive preset leaves base palette intact")
}

// Two presets active in order [A, B]: B wins on overlapping slots.
func TestPresetStack_LaterPresetWins(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	s := NewPresetStack()
	a := s.Add(p, "A")
	b := s.Add(p, "B")
	p.Palette.Presets[a].PaletteOverrides[8] = "#ff0000"
	p.Palette.Presets[b].PaletteOverrides[8] = "#00ff00"

	out := s.Compose(p)
	assert.Equal(t, "#00ff00", out.Palette.Base[8])
}

// Out-of-range slot indices are skipped silently; the renderer surfaces
// the warning marker via presetHasWarning.
func TestPresetStack_OutOfRangeSlotSkipped(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	s := NewPresetStack()
	idx := s.Add(p, "Bad")
	p.Palette.Presets[idx].PaletteOverrides[99] = "#ff0000"

	// Compose does not panic.
	out := s.Compose(p)
	assert.NotNil(t, out)
	assert.True(t, s.presetHasWarning(p.Palette.Presets[idx]))
}

// Remove drops the preset and re-keys the active map.
func TestPresetStack_Remove(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	s := NewPresetStack()
	a := s.Add(p, "A")
	b := s.Add(p, "B")
	_ = a

	s.Remove(p, a)
	require.Len(t, p.Palette.Presets, 1)
	assert.Equal(t, "B", p.Palette.Presets[0].Name)
	// What was preset b is now at index 0 and active.
	assert.True(t, s.IsActive(0))
	_ = b
}

// Empty preset is a no-op when active.
func TestPresetStack_EmptyPresetNoOp(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	original := p.Palette.Base[8]
	s := NewPresetStack()
	s.Add(p, "Empty")
	out := s.Compose(p)
	assert.Equal(t, original, out.Palette.Base[8])
}

// Rename mutates the preset's name.
func TestPresetStack_Rename(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	s := NewPresetStack()
	idx := s.Add(p, "Old")
	s.Rename(p, idx, "New")
	assert.Equal(t, "New", p.Palette.Presets[idx].Name)
}
