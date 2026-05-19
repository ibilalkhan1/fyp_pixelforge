//go:build !js

package pixelforge_save_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pisave "github.com/ibilalkhan1/fyp_pixelforge/pixelforge_save"
)

func newTempBackend(t *testing.T) *pisave.BackendNative {
	t.Helper()
	dir := t.TempDir()
	return pisave.NewBackendNativeAtPath(dir)
}

func TestBackendNative_WriteThenRead(t *testing.T) {
	b := newTempBackend(t)
	require.NoError(t, b.Write("slot1", []byte("hello")))
	data, err := b.Read("slot1")
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

func TestBackendNative_ReadMissingReturnsError(t *testing.T) {
	b := newTempBackend(t)
	_, err := b.Read("missing")
	require.Error(t, err)
}

func TestBackendNative_DeleteRemovesFile(t *testing.T) {
	b := newTempBackend(t)
	require.NoError(t, b.Write("slot1", []byte("hi")))
	require.NoError(t, b.Delete("slot1"))
	_, err := b.Read("slot1")
	require.Error(t, err)
}

func TestBackendNative_DeleteMissingIsNoOp(t *testing.T) {
	b := newTempBackend(t)
	require.NoError(t, b.Delete("never_existed"))
}

func TestBackendNative_ListReturnsWrittenSlots(t *testing.T) {
	b := newTempBackend(t)
	require.NoError(t, b.Write("slot1", []byte("a")))
	require.NoError(t, b.Write(pisave.AutosaveSlotName, []byte("b")))
	got, err := b.List()
	require.NoError(t, err)
	names := []string{}
	for _, m := range got {
		names = append(names, m.Name)
	}
	assert.ElementsMatch(t, []string{"slot1", "autosave"}, names)
}

func TestBackendNative_ListMissingDirReturnsNil(t *testing.T) {
	b := pisave.NewBackendNativeAtPath(filepath.Join(t.TempDir(), "does-not-exist"))
	got, err := b.List()
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestBackendNative_WriteCreatesMissingDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), "fresh")
	b := pisave.NewBackendNativeAtPath(root)
	require.NoError(t, b.Write("slot1", []byte("hi")))
	info, err := os.Stat(root)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestBackendNative_NewBackendNativeFromTitleNonEmpty(t *testing.T) {
	b, err := pisave.NewBackendNative("My Game")
	require.NoError(t, err)
	assert.NotEmpty(t, b.Dir())
}
