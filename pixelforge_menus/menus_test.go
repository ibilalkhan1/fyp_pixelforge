package pixelforge_menus_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	piloop "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_loop"
	pimenus "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_menus"
)

func resetPauseAndMenus(t *testing.T) {
	t.Helper()
	piloop.Resume()
	pimenus.ResetForTest()
	t.Cleanup(func() {
		piloop.Resume()
		pimenus.ResetForTest()
	})
}

// ===== Registry tests =====

func TestRegistry_AllCanonicalTemplatesPresent(t *testing.T) {
	resetPauseAndMenus(t)
	for _, name := range pimenus.CanonicalTemplateNames {
		_, ok := pimenus.LookupTemplate(name)
		assert.True(t, ok, "canonical template %s registered", name)
	}
}

func TestRegistry_LookupUnknownReturnsFalse(t *testing.T) {
	resetPauseAndMenus(t)
	_, ok := pimenus.LookupTemplate("not_a_real_template")
	assert.False(t, ok)
}

func TestRegistry_AllTemplatesReturnsSortedNames(t *testing.T) {
	resetPauseAndMenus(t)
	got := pimenus.AllTemplates()
	require.Len(t, got, len(pimenus.CanonicalTemplateNames))
	// Sorted alphabetically.
	for i := 1; i < len(got); i++ {
		assert.Less(t, got[i-1], got[i])
	}
}

func TestRegistry_DuplicateRegisterPanics(t *testing.T) {
	resetPauseAndMenus(t)
	assert.Panics(t, func() {
		pimenus.RegisterTemplate(pimenus.Template{Name: "title"})
	})
}

func TestRegistry_EmptyNameRegisterPanics(t *testing.T) {
	resetPauseAndMenus(t)
	assert.Panics(t, func() {
		pimenus.RegisterTemplate(pimenus.Template{})
	})
}

// ===== Stack tests =====

func TestStack_PushAndPop(t *testing.T) {
	resetPauseAndMenus(t)
	s := pimenus.NewMenuStack()
	assert.Equal(t, 0, s.Depth())
	s.Push("title_menu", "title", nil)
	assert.Equal(t, 1, s.Depth())
	popped, ok := s.Pop()
	require.True(t, ok)
	assert.Equal(t, "title_menu", popped.MenuName)
	assert.Equal(t, 0, s.Depth())
}

func TestStack_PopOnEmptyReturnsFalse(t *testing.T) {
	resetPauseAndMenus(t)
	s := pimenus.NewMenuStack()
	_, ok := s.Pop()
	assert.False(t, ok)
}

func TestStack_TopReturnsLatestEntry(t *testing.T) {
	resetPauseAndMenus(t)
	s := pimenus.NewMenuStack()
	s.Push("a", "title", nil)
	s.Push("b", "pause", nil)
	top, ok := s.Top()
	require.True(t, ok)
	assert.Equal(t, "b", top.MenuName)
}

func TestStack_OverlayPushTriggersPause(t *testing.T) {
	resetPauseAndMenus(t)
	require.False(t, piloop.IsPaused())
	s := pimenus.NewMenuStack()
	s.Push("pause_menu", "pause", nil) // pause = overlay
	assert.True(t, piloop.IsPaused(),
		"overlay menu push triggers piloop.Pause")
}

func TestStack_OverlayPopRestoresResume(t *testing.T) {
	resetPauseAndMenus(t)
	s := pimenus.NewMenuStack()
	s.Push("pause_menu", "pause", nil)
	_, _ = s.Pop()
	assert.False(t, piloop.IsPaused(),
		"closing the last overlay resumes the scene")
}

func TestStack_TwoOverlaysSingleResumeOnFinalPop(t *testing.T) {
	resetPauseAndMenus(t)
	s := pimenus.NewMenuStack()
	s.Push("pause_menu", "pause", nil)
	s.Push("inv_menu", "inventory", nil)
	assert.True(t, piloop.IsPaused())
	_, _ = s.Pop()
	assert.True(t, piloop.IsPaused(),
		"intermediate pop keeps the scene paused (one overlay still up)")
	_, _ = s.Pop()
	assert.False(t, piloop.IsPaused(),
		"last overlay pop resumes the scene")
}

func TestStack_FullScreenMenuDoesNotPause(t *testing.T) {
	resetPauseAndMenus(t)
	s := pimenus.NewMenuStack()
	s.Push("title_menu", "title", nil) // title = full_screen
	assert.False(t, piloop.IsPaused(),
		"full-screen menus do not pause the scene")
}

func TestStack_ClearPopsEverythingAndResumes(t *testing.T) {
	resetPauseAndMenus(t)
	s := pimenus.NewMenuStack()
	s.Push("a", "pause", nil)
	s.Push("b", "inventory", nil)
	require.True(t, piloop.IsPaused())
	s.Clear()
	assert.Equal(t, 0, s.Depth())
	assert.False(t, piloop.IsPaused())
}

// ===== Nav tests =====

func TestNav_OnDpadDownIncrementsCursor(t *testing.T) {
	resetPauseAndMenus(t)
	s := pimenus.NewMenuStack()
	s.Push("title_menu", "title", nil)
	assert.Equal(t, 1, s.OnDpadDown())
	assert.Equal(t, 2, s.OnDpadDown())
}

func TestNav_OnDpadDownWrapsAtMax(t *testing.T) {
	resetPauseAndMenus(t)
	s := pimenus.NewMenuStack()
	s.Push("title_menu", "title", nil) // 3 options default
	s.OnDpadDown()
	s.OnDpadDown()
	assert.Equal(t, 0, s.OnDpadDown(), "wraps from 2 → 0")
}

func TestNav_OnDpadUpDecrementsAndWraps(t *testing.T) {
	resetPauseAndMenus(t)
	s := pimenus.NewMenuStack()
	s.Push("title_menu", "title", nil)
	assert.Equal(t, 2, s.OnDpadUp(), "wraps from 0 → 2")
}

func TestNav_OnUseInvokesApply(t *testing.T) {
	resetPauseAndMenus(t)
	s := pimenus.NewMenuStack()
	s.Push("title_menu", "title", nil)
	verb, args, ok := s.OnUse()
	require.True(t, ok)
	assert.Equal(t, "scene_change", verb)
	assert.Equal(t, "level1", args["target"])
}

func TestNav_OnUseSaveGameDispatchesSlot(t *testing.T) {
	resetPauseAndMenus(t)
	s := pimenus.NewMenuStack()
	s.Push("save_menu", "save_game", nil)
	s.OnDpadDown() // cursor 1
	verb, args, ok := s.OnUse()
	require.True(t, ok)
	assert.Equal(t, "save_now", verb)
	assert.Equal(t, "slot2", args["slot"])
}

func TestNav_InventoryEmptyHasNoOptions(t *testing.T) {
	resetPauseAndMenus(t)
	s := pimenus.NewMenuStack()
	s.Push("inv_menu", "inventory", nil)
	// SelectionCount returns 0; cursor stays at 0.
	assert.Equal(t, 0, s.OnDpadDown())
}

func TestNav_InventoryWithItemsCountsCorrectly(t *testing.T) {
	resetPauseAndMenus(t)
	s := pimenus.NewMenuStack()
	s.Push("inv_menu", "inventory", map[string]any{
		"items": []any{"potion", "sword", "shield"},
	})
	// 3 items → cursor cycles 1, 2, 0.
	assert.Equal(t, 1, s.OnDpadDown())
	assert.Equal(t, 2, s.OnDpadDown())
	assert.Equal(t, 0, s.OnDpadDown())
}
