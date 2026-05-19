package menus_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/menus"
)

func newMenusEditor(t *testing.T) (*editor.Editor, *menus.Workspace) {
	t.Helper()
	e := editor.New()
	w := menus.RegisterWith(e)
	return e, w
}

func TestWorkspace_RegistersWithEditor(t *testing.T) {
	e, w := newMenusEditor(t)
	assert.Equal(t, "menus", w.Name())
	assert.Equal(t, "Menus", w.DisplayName())
	found := false
	for _, ws := range e.Workspaces() {
		if ws.Name() == "menus" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestWorkspace_NewMenuWithValidTemplate(t *testing.T) {
	e, w := newMenusEditor(t)
	require.True(t, w.NewMenu("title_screen", "title"))
	got := e.Project().Menus["title_screen"]
	assert.Equal(t, "title", got.Template)
	assert.True(t, e.IsDirty())
}

func TestWorkspace_NewMenuUnknownTemplateRejects(t *testing.T) {
	_, w := newMenusEditor(t)
	assert.False(t, w.NewMenu("oops", "not_a_template"))
}

func TestWorkspace_NewMenuEmptyNameRejects(t *testing.T) {
	_, w := newMenusEditor(t)
	assert.False(t, w.NewMenu("", "title"))
}

func TestWorkspace_NewMenuDuplicateRejects(t *testing.T) {
	_, w := newMenusEditor(t)
	w.NewMenu("title_screen", "title")
	assert.False(t, w.NewMenu("title_screen", "title"))
}

func TestWorkspace_SetMenuTemplateUpdates(t *testing.T) {
	_, w := newMenusEditor(t)
	w.NewMenu("m", "title")
	assert.True(t, w.SetMenuTemplate("m", "pause"))
}

func TestWorkspace_SetMenuTemplateUnknownRejects(t *testing.T) {
	_, w := newMenusEditor(t)
	w.NewMenu("m", "title")
	assert.False(t, w.SetMenuTemplate("m", "no_such"))
}

func TestWorkspace_SetMenuParameterWritesValue(t *testing.T) {
	e, w := newMenusEditor(t)
	w.NewMenu("m", "title")
	require.True(t, w.SetMenuParameter("m", "game_name", "Adventure"))
	assert.Equal(t, "Adventure", e.Project().Menus["m"].Parameters["game_name"])
}

func TestWorkspace_SetMenuParameterIdempotentReturnsFalse(t *testing.T) {
	_, w := newMenusEditor(t)
	w.NewMenu("m", "title")
	w.SetMenuParameter("m", "k", "v")
	assert.False(t, w.SetMenuParameter("m", "k", "v"))
}

func TestWorkspace_DeleteMenu(t *testing.T) {
	e, w := newMenusEditor(t)
	w.NewMenu("doomed", "title")
	require.True(t, w.DeleteMenu("doomed"))
	assert.NotContains(t, e.Project().Menus, "doomed")
}

func TestWorkspace_AvailableTemplatesIncludesNineCanonical(t *testing.T) {
	_, w := newMenusEditor(t)
	got := w.AvailableTemplates()
	assert.Len(t, got, 9)
	assert.Contains(t, got, "title")
	assert.Contains(t, got, "inventory")
}

func TestWorkspace_MenuNamesSorted(t *testing.T) {
	_, w := newMenusEditor(t)
	w.NewMenu("zebra", "title")
	w.NewMenu("alpha", "pause")
	assert.Equal(t, []string{"alpha", "zebra"}, w.MenuNames())
}
