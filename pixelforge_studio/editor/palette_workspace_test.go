package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// palette_workspace_test.go covers the U5 state-mutating helpers
// (SetBaseColor, RenameSubPalette, AssignSubPaletteSlot) + the hex
// parse / format round-trip. The imgui-driven Render path is
// exercised structurally by ensuring the workspace registers and
// reports its name / display name.

// TestSetBaseColor_MutatesAndReportsChange: writing a new hex
// value mutates Palette.Base[slot] + returns true.
func TestSetBaseColor_MutatesAndReportsChange(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	old := p.Palette.Base[5]
	require.NotEqual(t, "#abcdef", old)
	assert.True(t, SetBaseColor(p, 5, "#abcdef"))
	assert.Equal(t, "#abcdef", p.Palette.Base[5])
}

// TestSetBaseColor_NoOpOnIdentical: writing the existing value
// returns false so callers can skip MarkDirty.
func TestSetBaseColor_NoOpOnIdentical(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	current := p.Palette.Base[5]
	assert.False(t, SetBaseColor(p, 5, current))
}

// TestSetBaseColor_RejectsOutOfRangeSlot: slots outside [0, 64)
// reject without mutating the palette.
func TestSetBaseColor_RejectsOutOfRangeSlot(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	assert.False(t, SetBaseColor(p, -1, "#000000"))
	assert.False(t, SetBaseColor(p, 64, "#000000"))
}

// TestRenameSubPalette_BGFamily: renaming a bg sub-palette mutates
// the project's BGSubPalettes[idx].Name.
func TestRenameSubPalette_BGFamily(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Palette.ApplyDefaults()
	assert.True(t, RenameSubPalette(p, "bg", 0, "ground"))
	assert.Equal(t, "ground", p.Palette.BGSubPalettes[0].Name)
}

// TestRenameSubPalette_SpriteFamily: renaming sprite sub-palette.
func TestRenameSubPalette_SpriteFamily(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Palette.ApplyDefaults()
	assert.True(t, RenameSubPalette(p, "sprite", 2, "boss"))
	assert.Equal(t, "boss", p.Palette.SpriteSubPalettes[2].Name)
}

// TestRenameSubPalette_UnknownFamilyReturnsFalse: family token
// outside {bg, sprite} rejects.
func TestRenameSubPalette_UnknownFamilyReturnsFalse(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	assert.False(t, RenameSubPalette(p, "ui", 0, "x"))
}

// TestAssignSubPaletteSlot_WritesAndClamps: assigning a slot index
// inside range writes it; out-of-range clamps into [0, 64).
func TestAssignSubPaletteSlot_WritesAndClamps(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Palette.ApplyDefaults()
	assert.True(t, AssignSubPaletteSlot(p, "bg", 0, 0, 42))
	assert.Equal(t, 42, p.Palette.BGSubPalettes[0].Slots[0])

	// Out-of-range high → clamped to MaxColors-1.
	assert.True(t, AssignSubPaletteSlot(p, "bg", 0, 0, 999))
	assert.Equal(t, pixelforge_project.MaxColors-1,
		p.Palette.BGSubPalettes[0].Slots[0])

	// Out-of-range low → clamped to 0.
	assert.True(t, AssignSubPaletteSlot(p, "bg", 0, 0, -5))
	assert.Equal(t, 0, p.Palette.BGSubPalettes[0].Slots[0])
}

// TestAssignSubPaletteSlot_NoOpOnIdentical: writing the existing
// value returns false.
func TestAssignSubPaletteSlot_NoOpOnIdentical(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Palette.ApplyDefaults()
	current := p.Palette.BGSubPalettes[0].Slots[0]
	assert.False(t, AssignSubPaletteSlot(p, "bg", 0, 0, current))
}

// TestAssignSubPaletteSlot_InvalidSwatchIdx: swatchIdx outside [0,4)
// rejects.
func TestAssignSubPaletteSlot_InvalidSwatchIdx(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Palette.ApplyDefaults()
	assert.False(t, AssignSubPaletteSlot(p, "bg", 0, -1, 5))
	assert.False(t, AssignSubPaletteSlot(p, "bg", 0, 4, 5))
}

// TestPaletteWorkspace_RegistersOnEditor: the workspace registers
// via RegisterPaletteWorkspaceWith.
func TestPaletteWorkspace_RegistersOnEditor(t *testing.T) {
	e := New()
	w := RegisterPaletteWorkspaceWith(e)
	require.NotNil(t, w)
	assert.Equal(t, "nes_palette", w.Name())
	assert.Equal(t, "NES Palette", w.DisplayName())
}

// TestPaletteColorRoundTrip: hex → arr3 → hex preserves the value.
func TestPaletteColorRoundTrip(t *testing.T) {
	for _, hex := range []string{"#000000", "#ffffff", "#8b4513", "#abcdef"} {
		got := arr3ToHex(paletteColorToArr3(hex))
		assert.Equal(t, hex, got)
	}
}

// TestPaletteColorParseFallback: malformed hex parses to (0,0,0)
// without crashing.
func TestPaletteColorParseFallback(t *testing.T) {
	r, g, b := parseHexRGB("not-a-hex")
	assert.Equal(t, byte(0), r)
	assert.Equal(t, byte(0), g)
	assert.Equal(t, byte(0), b)
}
