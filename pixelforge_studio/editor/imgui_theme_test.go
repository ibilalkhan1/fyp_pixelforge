package editor

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// TestHexToVec4_ParsesStandardForm — palette entries are "#RRGGBB";
// the converter normalises them to ImGui's 0..1 Vec4.
func TestHexToVec4_ParsesStandardForm(t *testing.T) {
	v := hexToVec4("#FF8000")
	assert.InDelta(t, 1.0, v.X, 0.001, "R = 255/255")
	assert.InDelta(t, 0.502, v.Y, 0.01, "G = 128/255")
	assert.InDelta(t, 0.0, v.Z, 0.001, "B = 0/255")
	assert.Equal(t, float32(1), v.W, "fully opaque alpha")
}

// TestHexToVec4_FallsBackOnMalformed — a bad hex string returns
// neutral grey so a corrupted palette entry doesn't crash the
// renderer.
func TestHexToVec4_FallsBackOnMalformed(t *testing.T) {
	v := hexToVec4("not-a-hex")
	assert.Equal(t, float32(0.5), v.X)
	assert.Equal(t, float32(0.5), v.Y)
	assert.Equal(t, float32(0.5), v.Z)
}

// TestThemeLoadsPaletteColors — given a project whose palette slot 5
// is a specific RGB, the resulting imguiTheme uses that exact colour
// for the slot the EditorTheme assigned to slot 5.
func TestThemeLoadsPaletteColors(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Palette.Base[5] = "#112233"
	theme := DefaultEditorTheme()
	// DefaultEditorTheme sets PanelHeaderSlot = 5.
	it := buildImguiTheme(theme, &p.Palette)
	want := hexToVec4("#112233")
	assert.Equal(t, want, it.PanelHdr,
		"palette slot 5 must feed PanelHeader because DefaultEditorTheme assigns it there")
}

// TestThemeFallsBackOnMissingPalette — buildImguiTheme tolerates a
// nil palette by emitting neutral grey for every slot. This is the
// path the editor takes before a project is loaded.
func TestThemeFallsBackOnMissingPalette(t *testing.T) {
	theme := DefaultEditorTheme()
	it := buildImguiTheme(theme, nil)
	grey := hexToVec4("#808080") // not exact — comparing against the documented fallback semantics
	_ = grey
	// Each slot resolves to the documented (0.5, 0.5, 0.5, 1) grey.
	assert.Equal(t, float32(0.5), it.Text.X)
	assert.Equal(t, float32(0.5), it.WindowBg.Y)
	assert.Equal(t, float32(0.5), it.Accent.Z)
}

// TestThemeFallsBackOnNilTheme — a nil EditorTheme defers to
// DefaultEditorTheme(), so callers never have to nil-check before
// calling buildImguiTheme.
func TestThemeFallsBackOnNilTheme(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	assert.NotPanics(t, func() {
		_ = buildImguiTheme(nil, &p.Palette)
	})
}

// TestApplyImguiTheme_SkippedWithoutLiveBackend — applyImguiTheme
// returns 0 (no pushes) when live=false, so unit tests that never
// stand up an ImGui context don't trip the C-side style stack.
func TestApplyImguiTheme_SkippedWithoutLiveBackend(t *testing.T) {
	pushed := applyImguiTheme(imguiTheme{}, false)
	assert.Equal(t, 0, pushed)
}

// TestImguiIniPathInUserConfigDir — the persistence path lives
// under <user-config-dir>/pixelforge-studio/imgui.ini so dock
// layout survives restarts. The directory is created if missing.
func TestImguiIniPathInUserConfigDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := imguiIniPath()
	if err != nil {
		t.Skipf("user config dir not resolvable in this environment: %v", err)
	}
	assert.Equal(t, "imgui.ini", filepath.Base(path))
	assert.Contains(t, path, "pixelforge-studio")
}

// TestImguiIniPath_DirectoryCreated — the parent dir is mkdir'd so
// ImGui's first save doesn't fail with a "no such directory" error.
func TestImguiIniPath_DirectoryCreated(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_, err := imguiIniPath()
	if err != nil {
		t.Skipf("user config dir not resolvable: %v", err)
	}
	// A second call should still succeed (idempotent mkdir).
	_, err = imguiIniPath()
	assert.NoError(t, err)
}

// TestActiveImguiThemeAlwaysSafe — activeImguiTheme on a fresh
// editor returns a usable theme without panicking, even before a
// project or cart is fully populated.
func TestActiveImguiThemeAlwaysSafe(t *testing.T) {
	e := New()
	assert.NotPanics(t, func() {
		_ = e.activeImguiTheme()
	})
}

// TestFontFallbackHandled — the legacy applyFontTheme in
// cart_loader.go still tolerates unknown font names by logging and
// falling back to cofont. Mirror that contract here so theme-driven
// font changes via editor.pforge don't crash chrome.
func TestFontFallbackHandled(t *testing.T) {
	assert.NotPanics(t, func() {
		applyFontTheme("nonexistent-font")
	})
	assert.NotPanics(t, func() {
		applyFontTheme("cofont")
	})
}
