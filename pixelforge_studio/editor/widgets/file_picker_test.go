package widgets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readDirEntries returns directories first, then files, both alpha-asc.
func TestReadDirEntries_SortOrder(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "zebra.txt"), []byte("z"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "alpha.txt"), []byte("a"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "beta"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "Apple"), 0o755))

	entries, err := readDirEntries(dir, FilePickerOptions{})
	require.NoError(t, err)
	require.Len(t, entries, 4)
	// Dirs first.
	assert.True(t, entries[0].IsDir)
	assert.True(t, entries[1].IsDir)
	// Alpha order is case-insensitive.
	assert.Equal(t, "Apple", entries[0].Name)
	assert.Equal(t, "beta", entries[1].Name)
	assert.Equal(t, "alpha.txt", entries[2].Name)
	assert.Equal(t, "zebra.txt", entries[3].Name)
}

// Extension filter hides files outside the allowed list but keeps dirs.
func TestReadDirEntries_ExtensionFilter(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scene.pforge"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sprite.png"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("x"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))

	entries, err := readDirEntries(dir, FilePickerOptions{Extensions: []string{".pforge"}})
	require.NoError(t, err)
	names := entryNames(entries)
	assert.Equal(t, []string{"sub", "scene.pforge"}, names)
}

// Hidden files are excluded unless ShowHidden is set.
func TestReadDirEntries_HiddenFilesHiddenByDefault(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".hidden"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "visible.txt"), []byte("x"), 0o644))

	entries, err := readDirEntries(dir, FilePickerOptions{})
	require.NoError(t, err)
	assert.Equal(t, []string{"visible.txt"}, entryNames(entries))

	entries, err = readDirEntries(dir, FilePickerOptions{ShowHidden: true})
	require.NoError(t, err)
	assert.Contains(t, entryNames(entries), ".hidden")
}

// PickDir mode hides regular files from the listing.
func TestReadDirEntries_PickDirHidesFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))

	entries, err := readDirEntries(dir, FilePickerOptions{Mode: PickDir})
	require.NoError(t, err)
	assert.Equal(t, []string{"sub"}, entryNames(entries))
}

// extensionAllowed honours empty allow-list (no filter) and ignores case.
func TestExtensionAllowed(t *testing.T) {
	assert.True(t, extensionAllowed("game.pforge", nil))
	assert.True(t, extensionAllowed("game.pforge", []string{".pforge"}))
	assert.True(t, extensionAllowed("GAME.PFORGE", []string{".pforge"}))
	assert.False(t, extensionAllowed("art.png", []string{".pforge"}))
}

// ensureExtension appends the first allowed extension when missing.
func TestEnsureExtension(t *testing.T) {
	assert.Equal(t, "game.pforge", ensureExtension("game", []string{".pforge"}))
	assert.Equal(t, "game.pforge", ensureExtension("game.pforge", []string{".pforge"}))
	assert.Equal(t, "game.txt.pforge", ensureExtension("game.txt", []string{".pforge"}))
	// No allow-list = pass through.
	assert.Equal(t, "raw", ensureExtension("raw", nil))
}

// Open populates the entries from the starting directory and seeds the
// modal Visible flag.
func TestFilePicker_OpenListsStartDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("x"), 0o644))

	p := NewFilePicker()
	p.Open(FilePickerOptions{StartPath: dir, Mode: PickOpen})

	assert.True(t, p.Visible())
	assert.Equal(t, filepath.Clean(dir), p.CurrentPath())
	assert.Equal(t, []string{"a.txt", "b.txt"}, entryNames(p.Entries()))
}

// Confirm in PickOpen mode returns the absolute path of the selected
// file and fires OnConfirm exactly once.
func TestFilePicker_ConfirmOpen(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scene.pforge"), []byte("{}"), 0o644))

	var gotPath string
	calls := 0
	p := NewFilePicker()
	p.Open(FilePickerOptions{
		StartPath:  dir,
		Mode:       PickOpen,
		Extensions: []string{".pforge"},
		OnConfirm:  func(path string) { gotPath = path; calls++ },
	})
	require.Equal(t, []string{"scene.pforge"}, entryNames(p.Entries()))

	got := p.Confirm()
	want := filepath.Join(filepath.Clean(dir), "scene.pforge")
	assert.Equal(t, want, got)
	assert.Equal(t, want, gotPath)
	assert.Equal(t, 1, calls)
	assert.False(t, p.Visible(), "picker dismisses after Confirm")
}

// Confirm in PickSave mode appends the first allowed extension.
func TestFilePicker_ConfirmSaveAppendsExtension(t *testing.T) {
	dir := t.TempDir()
	var gotPath string
	p := NewFilePicker()
	p.Open(FilePickerOptions{
		StartPath:  dir,
		Mode:       PickSave,
		Extensions: []string{".pforge"},
		OnConfirm:  func(path string) { gotPath = path },
	})
	p.SetSaveName("newgame")
	got := p.Confirm()
	want := filepath.Join(filepath.Clean(dir), "newgame.pforge")
	assert.Equal(t, want, got)
	assert.Equal(t, want, gotPath)
}

// Confirm in PickSave with an empty filename is a no-op.
func TestFilePicker_ConfirmSaveEmptyNameNoOps(t *testing.T) {
	dir := t.TempDir()
	calls := 0
	p := NewFilePicker()
	p.Open(FilePickerOptions{
		StartPath: dir,
		Mode:      PickSave,
		OnConfirm: func(path string) { calls++ },
	})
	assert.Equal(t, "", p.Confirm())
	assert.Equal(t, 0, calls)
	assert.True(t, p.Visible(), "no-op leaves picker visible")
}

// NavigateUp at the filesystem root is a no-op (does not panic).
func TestFilePicker_NavigateUpAtRootNoOps(t *testing.T) {
	p := NewFilePicker()
	p.Open(FilePickerOptions{StartPath: string(filepath.Separator)})
	before := p.CurrentPath()
	assert.NotPanics(t, func() { p.NavigateUp() })
	assert.Equal(t, before, p.CurrentPath())
}

// Unreadable path surfaces an error instead of panicking; picker stays
// usable so the user can navigate elsewhere.
func TestFilePicker_OpenOnUnreadablePathSetsError(t *testing.T) {
	p := NewFilePicker()
	p.Open(FilePickerOptions{StartPath: "/this/does/not/exist/anywhere"})
	// Falls back to home dir → no error. We instead force the error
	// branch by calling setCurrent on a bogus path post-open.
	p.setCurrent("/this/does/not/exist/anywhere")
	assert.NotNil(t, p.listErr)
}

// Navigate into a child directory updates the listing.
func TestFilePicker_NavigateIntoSubdir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	require.NoError(t, os.Mkdir(sub, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "leaf.txt"), []byte("x"), 0o644))

	p := NewFilePicker()
	p.Open(FilePickerOptions{StartPath: dir})
	p.Navigate("sub")

	assert.Equal(t, filepath.Clean(sub), p.CurrentPath())
	assert.Equal(t, []string{"leaf.txt"}, entryNames(p.Entries()))
}

// PickDir confirm returns the current directory when nothing is
// selected, and the joined subdir path when a directory is selected.
func TestFilePicker_ConfirmPickDir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	require.NoError(t, os.Mkdir(sub, 0o755))

	p := NewFilePicker()
	p.Open(FilePickerOptions{StartPath: dir, Mode: PickDir})
	// No entries are selected by default; selecting index 0 (the only
	// entry in PickDir mode) targets "sub".
	require.Len(t, p.Entries(), 1)
	got := p.Confirm()
	assert.Equal(t, filepath.Clean(sub), got)
}

// Dismiss flips Visible and fires OnCancel.
func TestFilePicker_DismissFiresOnCancel(t *testing.T) {
	calls := 0
	p := NewFilePicker()
	p.Open(FilePickerOptions{
		StartPath: t.TempDir(),
		OnCancel:  func() { calls++ },
	})
	p.Dismiss()
	assert.False(t, p.Visible())
	assert.Equal(t, 1, calls)
}

func entryNames(entries []fileEntry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.Name
	}
	return out
}
