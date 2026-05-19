package dialogue_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/dialogue"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/editor"
)

func newDialogueEditor(t *testing.T) (*editor.Editor, *dialogue.Workspace) {
	t.Helper()
	e := editor.New()
	w := dialogue.RegisterWith(e)
	return e, w
}

func TestWorkspace_RegistersWithEditor(t *testing.T) {
	e, w := newDialogueEditor(t)
	assert.Equal(t, "dialogue", w.Name())
	assert.Equal(t, "Dialogue", w.DisplayName())
	found := false
	for _, ws := range e.Workspaces() {
		if ws.Name() == "dialogue" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestWorkspace_NewTreeAppendsAndMarksDirty(t *testing.T) {
	e, w := newDialogueEditor(t)
	require.True(t, w.NewTree("intro"))
	assert.Contains(t, e.Project().Dialogues, "intro")
	assert.True(t, e.IsDirty())
}

func TestWorkspace_NewTreeDuplicateReturnsFalse(t *testing.T) {
	_, w := newDialogueEditor(t)
	w.NewTree("intro")
	assert.False(t, w.NewTree("intro"))
}

func TestWorkspace_NewTreeEmptyNameReturnsFalse(t *testing.T) {
	_, w := newDialogueEditor(t)
	assert.False(t, w.NewTree(""))
}

func TestWorkspace_SetScriptUpdatesProjectAndParses(t *testing.T) {
	e, w := newDialogueEditor(t)
	w.NewTree("greeting")
	errs := w.SetScript("greeting", "HERO: hello")
	assert.Empty(t, errs)
	assert.Contains(t, e.Project().Dialogues["greeting"].Script, "HERO: hello")
}

func TestWorkspace_SetScriptSurfacesParseErrors(t *testing.T) {
	_, w := newDialogueEditor(t)
	w.NewTree("broken")
	errs := w.SetScript("broken", "not_a_valid_line")
	assert.NotEmpty(t, errs)
}

func TestWorkspace_ParseErrorsCachedPerTree(t *testing.T) {
	_, w := newDialogueEditor(t)
	w.NewTree("a")
	w.NewTree("b")
	_ = w.SetScript("a", "broken line")
	_ = w.SetScript("b", "HERO: ok")
	assert.NotEmpty(t, w.ParseErrors("a"))
	assert.Empty(t, w.ParseErrors("b"))
}

func TestWorkspace_DeleteTreeRemovesEntry(t *testing.T) {
	e, w := newDialogueEditor(t)
	w.NewTree("doomed")
	require.True(t, w.DeleteTree("doomed"))
	assert.NotContains(t, e.Project().Dialogues, "doomed")
}

func TestWorkspace_DeleteTreeClearsActiveSelection(t *testing.T) {
	_, w := newDialogueEditor(t)
	w.NewTree("active")
	w.SetActiveTree("active")
	w.DeleteTree("active")
	assert.Empty(t, w.ActiveTree())
}

func TestWorkspace_TreeNamesReturnsSorted(t *testing.T) {
	_, w := newDialogueEditor(t)
	w.NewTree("zebra")
	w.NewTree("alpha")
	w.NewTree("mango")
	names := w.TreeNames()
	assert.Equal(t, []string{"alpha", "mango", "zebra"}, names)
}

func TestWorkspace_SetActiveTreeAndGet(t *testing.T) {
	_, w := newDialogueEditor(t)
	w.NewTree("a")
	w.SetActiveTree("a")
	assert.Equal(t, "a", w.ActiveTree())
}
