package ingest_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/ingest"
)

// fakeDropFS wraps fstest.MapFS so DragDrop's Poll receives a
// drop-shaped fs.FS without needing an Ebitengine window.
func fakeDropFS(files map[string][]byte) fs.FS {
	out := fstest.MapFS{}
	for name, data := range files {
		out[name] = &fstest.MapFile{Data: data}
	}
	return out
}

func TestDragDrop_SinglePNGDispatches(t *testing.T) {
	disp := ingest.NewDispatcher()
	runner := &syncRunner{}
	disp.SetSpriteRunner(runner)
	staging := t.TempDir()
	dropFS := fakeDropFS(map[string][]byte{"ship.png": []byte("png-bytes")})

	dd := ingest.NewDragDrop(disp, staging, func() fs.FS { return dropFS })
	count := dd.Poll()
	assert.Equal(t, 1, count)
	require.Len(t, runner.Calls(), 1)
	assert.Equal(t, filepath.Join(staging, "ship.png"), runner.Calls()[0])

	// Materialised file exists on disk.
	data, err := os.ReadFile(filepath.Join(staging, "ship.png"))
	require.NoError(t, err)
	assert.Equal(t, "png-bytes", string(data))
}

func TestDragDrop_MixedKindsAllDispatch(t *testing.T) {
	disp := ingest.NewDispatcher()
	sprite := &syncRunner{}
	sfx := &syncRunner{}
	bgm := &syncRunner{}
	disp.SetSpriteRunner(sprite)
	disp.SetSFXRunner(sfx)
	disp.SetBGMRunner(bgm)
	staging := t.TempDir()
	dropFS := fakeDropFS(map[string][]byte{
		"ship.png":   []byte("p"),
		"blast.wav":  []byte("w"),
		"level1.ogg": []byte("o"),
	})

	dd := ingest.NewDragDrop(disp, staging, func() fs.FS { return dropFS })
	count := dd.Poll()
	assert.Equal(t, 3, count)
	assert.Len(t, sprite.Calls(), 1)
	assert.Len(t, sfx.Calls(), 1)
	assert.Len(t, bgm.Calls(), 1)
}

func TestDragDrop_UnknownExtensionIgnored(t *testing.T) {
	disp := ingest.NewDispatcher()
	runner := &syncRunner{}
	disp.SetSpriteRunner(runner)
	staging := t.TempDir()
	dropFS := fakeDropFS(map[string][]byte{
		"notes.txt": []byte("ignore me"),
		"ship.png":  []byte("keep me"),
	})
	dd := ingest.NewDragDrop(disp, staging, func() fs.FS { return dropFS })
	assert.Equal(t, 1, dd.Poll(), "only the PNG should dispatch")
	assert.Len(t, runner.Calls(), 1)
}

func TestDragDrop_EmptySourceReturnsZero(t *testing.T) {
	disp := ingest.NewDispatcher()
	dd := ingest.NewDragDrop(disp, t.TempDir(), func() fs.FS { return nil })
	assert.Equal(t, 0, dd.Poll())
}

func TestDragDrop_NestedManifestsFlatten(t *testing.T) {
	disp := ingest.NewDispatcher()
	runner := &syncRunner{}
	disp.SetSpriteRunner(runner)
	staging := t.TempDir()
	// Nested directory — drop a folder containing assets.
	dropFS := fakeDropFS(map[string][]byte{
		"my-pack/ship.png": []byte("png-bytes"),
	})
	dd := ingest.NewDragDrop(disp, staging, func() fs.FS { return dropFS })
	assert.Equal(t, 1, dd.Poll())
	// Materialised file collapses to staging root (basename only).
	_, err := os.Stat(filepath.Join(staging, "ship.png"))
	assert.NoError(t, err)
}
