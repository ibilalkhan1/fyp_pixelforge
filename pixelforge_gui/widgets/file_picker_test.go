package widgets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilePicker_OpenSetsCurrent(t *testing.T) {
	tmp := t.TempDir()
	p := NewFilePicker()
	p.SetBounds(0, 0, 320, 180)
	p.Open(FilePickerOptions{StartPath: tmp})
	assert.True(t, p.Visible())
	assert.Equal(t, tmp, p.Current())
}

func TestFilePicker_OpenFallsBackToHome(t *testing.T) {
	p := NewFilePicker()
	p.SetBounds(0, 0, 320, 180)
	p.Open(FilePickerOptions{StartPath: "/no/such/path/anywhere"})
	home, _ := os.UserHomeDir()
	assert.Equal(t, home, p.Current())
}

func TestFilePicker_EnterDir(t *testing.T) {
	tmp := t.TempDir()
	sub := filepath.Join(tmp, "sub")
	require.NoError(t, os.Mkdir(sub, 0o755))
	p := NewFilePicker()
	p.SetBounds(0, 0, 320, 180)
	p.Open(FilePickerOptions{StartPath: tmp})
	// First (and only) entry is "sub/".
	require.Len(t, p.entries, 1)
	res := p.Enter(0)
	assert.Empty(t, res)
	assert.Equal(t, sub, p.Current())
}

func TestFilePicker_EnterFileReturnsAbs(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "file.txt")
	require.NoError(t, os.WriteFile(f, []byte("hi"), 0o644))
	p := NewFilePicker()
	p.SetBounds(0, 0, 320, 180)
	p.Open(FilePickerOptions{StartPath: tmp})
	res := p.Enter(0)
	assert.Equal(t, f, res)
}

func TestFilePicker_SaveModeConfirmDoesNothingWithoutName(t *testing.T) {
	pixelforge.SetScreenSize(320, 180)
	tmp := t.TempDir()
	fired := false
	p := NewFilePicker()
	p.SetBounds(0, 0, 320, 180)
	p.Open(FilePickerOptions{StartPath: tmp, Mode: PickSave, Extensions: []string{".pforge"}, OnConfirm: func(string) { fired = true }})
	p.Confirm()
	assert.False(t, fired)
}

func TestFilePicker_SaveModeAppendsExt(t *testing.T) {
	pixelforge.SetScreenSize(320, 180)
	tmp := t.TempDir()
	var got string
	p := NewFilePicker()
	p.SetBounds(0, 0, 320, 180)
	p.Open(FilePickerOptions{StartPath: tmp, Mode: PickSave, Extensions: []string{".pforge"}, OnConfirm: func(s string) { got = s }})
	p.SetSaveName("game")
	p.Confirm()
	assert.Equal(t, filepath.Join(tmp, "game.pforge"), got)
	assert.False(t, p.Visible())
}

func TestFilePicker_CancelClosesAndFiresCallback(t *testing.T) {
	tmp := t.TempDir()
	called := false
	p := NewFilePicker()
	p.SetBounds(0, 0, 320, 180)
	p.Open(FilePickerOptions{StartPath: tmp, OnCancel: func() { called = true }})
	p.Cancel()
	assert.False(t, p.Visible())
	assert.True(t, called)
}

func TestFilePicker_FilterByExtension(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "a.pforge"), []byte{}, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "b.txt"), []byte{}, 0o644))
	p := NewFilePicker()
	p.SetBounds(0, 0, 320, 180)
	p.Open(FilePickerOptions{StartPath: tmp, Extensions: []string{".pforge"}})
	require.Len(t, p.entries, 1)
	assert.Equal(t, "a.pforge", p.entries[0].Name)
}

func TestFilePicker_EscapeDismisses(t *testing.T) {
	tmp := t.TempDir()
	p := NewFilePicker()
	p.SetBounds(0, 0, 320, 180)
	p.Open(FilePickerOptions{StartPath: tmp})
	assert.True(t, p.HandleEscape())
	assert.False(t, p.Visible())
}

func TestFilePicker_DirModeReturnsCurrentDir(t *testing.T) {
	tmp := t.TempDir()
	var got string
	p := NewFilePicker()
	p.SetBounds(0, 0, 320, 180)
	p.Open(FilePickerOptions{StartPath: tmp, Mode: PickDir, OnConfirm: func(s string) { got = s }})
	p.Confirm()
	assert.Equal(t, tmp, got)
}
