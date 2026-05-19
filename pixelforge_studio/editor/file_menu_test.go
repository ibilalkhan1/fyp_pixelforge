package editor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// FileMenu.OpenPath loads the project at path, clears the selection,
// and pushes the path onto recent projects.
func TestFileMenu_OpenPathLoadsProject(t *testing.T) {
	dir := t.TempDir()
	pforgePath := filepath.Join(dir, "sample.pforge")
	original := pixelforge_project.NewProject("sample")
	require.NoError(t, original.Save(pforgePath))

	e := New()
	e.SelectEntity("stale") // confirm selection clears

	require.NoError(t, e.fileMenu.OpenPath(pforgePath))
	assert.Equal(t, "sample", e.Project().Name)
	assert.Equal(t, pforgePath, e.CurrentProjectPath())
	assert.Empty(t, e.SelectedEntityID())
	assert.Contains(t, e.settings.RecentProjects, pforgePath)
	assert.False(t, e.IsDirty())
}

// FileMenu.Save with a current path writes to that path and clears dirty.
func TestFileMenu_SaveWritesAndClearsDirty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "g.pforge")

	e := New()
	e.SetCurrentProjectPath(path)
	e.MarkDirty()

	require.NoError(t, e.fileMenu.Save())
	assert.False(t, e.IsDirty())
	_, err := os.Stat(path)
	assert.NoError(t, err)
}

// FileMenu.Save with no current path routes to SaveAs (which opens the picker).
func TestFileMenu_SaveOnUntitledOpensPicker(t *testing.T) {
	e := New()
	e.MarkDirty()
	assert.False(t, e.filePicker.Visible())
	require.NoError(t, e.fileMenu.Save())
	assert.True(t, e.filePicker.Visible())
}

// FileMenu.SaveTo writes to a custom path and updates the current path.
func TestFileMenu_SaveToUpdatesCurrentPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.pforge")
	e := New()
	require.NoError(t, e.fileMenu.SaveTo(path))
	assert.Equal(t, path, e.CurrentProjectPath())
}

// FileMenu.New on a dirty project opens the confirm modal.
func TestFileMenu_NewOnDirtyOpensConfirm(t *testing.T) {
	e := New()
	e.MarkDirty()
	e.fileMenu.New()
	assert.True(t, e.confirmDialog.Visible(), "dirty project → confirm modal")

	// Confirm replaces the project.
	old := e.Project()
	e.confirmDialog.Confirm()
	assert.NotSame(t, old, e.Project())
	assert.False(t, e.IsDirty())
}

// FileMenu.New on a clean project skips the modal.
func TestFileMenu_NewOnCleanReplacesImmediately(t *testing.T) {
	e := New()
	old := e.Project()
	e.fileMenu.New()
	assert.False(t, e.confirmDialog.Visible())
	assert.NotSame(t, old, e.Project())
}

// Open → Save As → Open: the round-trip produces an equivalent project.
func TestFileMenu_OpenSaveAsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.pforge")
	dstPath := filepath.Join(dir, "dst.pforge")

	src := pixelforge_project.NewProject("rt")
	src.ScreenWidth = 640
	src.ScreenHeight = 480
	require.NoError(t, src.Save(srcPath))

	e := New()
	require.NoError(t, e.fileMenu.OpenPath(srcPath))
	require.NoError(t, e.fileMenu.SaveTo(dstPath))

	e2 := New()
	require.NoError(t, e2.fileMenu.OpenPath(dstPath))
	assert.Equal(t, 640, e2.Project().ScreenWidth)
	assert.Equal(t, 480, e2.Project().ScreenHeight)
	assert.Equal(t, "rt", e2.Project().Name)
}

// OpenPath on a malformed file surfaces the error and leaves the
// existing project loaded.
func TestFileMenu_OpenPathOnMalformedFile(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.pforge")
	require.NoError(t, os.WriteFile(bad, []byte("not valid"), 0o644))

	e := New()
	original := e.Project()
	err := e.fileMenu.OpenPath(bad)
	assert.Error(t, err)
	assert.Same(t, original, e.Project(), "project unchanged on failure")
}

// Quit on a dirty project shows the confirm; clean Quit terminates.
func TestFileMenu_QuitDirtyShowsConfirm(t *testing.T) {
	e := New()
	e.MarkDirty()
	e.fileMenu.Quit()
	assert.True(t, e.confirmDialog.Visible())
	assert.False(t, e.terminate)
	e.confirmDialog.Confirm()
	assert.True(t, e.terminate)
}

func TestFileMenu_QuitCleanTerminates(t *testing.T) {
	e := New()
	e.fileMenu.Quit()
	assert.True(t, e.terminate)
}

// Plan-009 U22 — File → New → Blank Platformer creates a project
// pre-wired with the Mario-class preset. The handler runs through
// PromptIfDirty (so a dirty project is preserved) and replaces the
// editor's in-memory project on the clean / confirmed path.
func TestFileMenu_NewFromTemplate_BlankPlatformer(t *testing.T) {
	e := New()
	e.fileMenu.NewFromTemplate("Blank Platformer")
	assert.Equal(t, "mario", e.Project().PhysicsPreset, "platformer template flips physics_preset")
	assert.False(t, e.IsDirty(), "new-from-template should not flip dirty bit")
	assert.Empty(t, e.CurrentProjectPath(), "template-spawned project has no on-disk source")
}

// Plan-009 U22 — File → New → submenu must enumerate exactly the four
// generic genre starters (AE10).
func TestFileMenu_NewFromTemplateMenu_HasFourGenericRows(t *testing.T) {
	e := New()
	items := e.NewFromTemplateMenu()
	require.Len(t, items, 4, "exactly four genre starters")
	labels := map[string]bool{}
	for _, it := range items {
		labels[it.Label] = true
	}
	for _, want := range []string{
		"Blank Platformer",
		"Blank Arcade Shooter",
		"Blank Grid Game",
		"Blank Ladder Platformer",
	} {
		assert.True(t, labels[want], "expected submenu label %q", want)
	}
	// AE10 negative: trademarked names must not appear.
	for _, banned := range []string{"Mario", "Asteroids", "Bomberman", "Donkey Kong", "DK"} {
		assert.False(t, labels[banned], "label %q must not appear (AE10)", banned)
	}
}

// Plan-009 U22 — Unknown template name surfaces a status message
// without mutating the project.
func TestFileMenu_NewFromTemplate_UnknownNameIsNoOp(t *testing.T) {
	e := New()
	old := e.Project()
	e.fileMenu.NewFromTemplate("Mario")
	assert.Same(t, old, e.Project(), "unknown template must not replace project")
	assert.Contains(t, e.StatusMessage(), "unknown template")
}

// Plan-009 U22 — Dirty-state prompt fires on NewFromTemplate just
// like the legacy New handler.
func TestFileMenu_NewFromTemplate_DirtyOpensConfirm(t *testing.T) {
	e := New()
	e.MarkDirty()
	e.fileMenu.NewFromTemplate("Blank Platformer")
	assert.True(t, e.confirmDialog.Visible(), "dirty project → confirm modal before template swap")

	e.confirmDialog.Confirm()
	assert.Equal(t, "mario", e.Project().PhysicsPreset)
}
