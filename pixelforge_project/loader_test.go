package pixelforge_project

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Happy path: save then load returns an equal project.
func TestLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snake.pforge")

	original := snakeShapedProject(t)
	require.NoError(t, original.Save(path))

	// Drop the missing sprite asset onto disk so validateAssets passes.
	writeFile(t, filepath.Join(AssetsDir(path), "sprites", "sprites.png"), []byte("fakepng"))

	loaded, err := Load(path)
	require.NoError(t, err)

	assert.Equal(t, original.Name, loaded.Name)
	assert.Equal(t, original.ScreenWidth, loaded.ScreenWidth)
	assert.Equal(t, len(original.Sprites), len(loaded.Sprites))
	assert.Equal(t, len(original.Scenes[0].Entities), len(loaded.Scenes[0].Entities))
}

// Saving twice produces byte-identical files. The minimal-diff
// invariant our git-merge story depends on.
func TestSave_DeterministicBytes(t *testing.T) {
	dir := t.TempDir()
	path1 := filepath.Join(dir, "a.pforge")
	path2 := filepath.Join(dir, "b.pforge")

	// Pin nowFunc so Save's touch() doesn't introduce per-call drift.
	withFrozenClock(t)

	p := snakeShapedProject(t)
	require.NoError(t, p.Save(path1))
	on1, err := os.ReadFile(path1)
	require.NoError(t, err)

	require.NoError(t, p.Save(path2))
	on2, err := os.ReadFile(path2)
	require.NoError(t, err)

	assert.Equal(t, on1, on2, "two saves of the same project must produce byte-identical files")
}

// withFrozenClock pins pixelforge_project's now-source to a fixed
// instant for the duration of the test. Restores the previous value
// via t.Cleanup so concurrent or subsequent tests are unaffected.
func withFrozenClock(t *testing.T) {
	t.Helper()
	frozen := pinClock()
	t.Cleanup(frozen)
}

// Save creates the *-assets directory if missing.
func TestSave_CreatesAssetsDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fresh.pforge")
	p := NewProject("fresh")
	require.NoError(t, p.Save(path))

	stat, err := os.Stat(AssetsDir(path))
	require.NoError(t, err)
	assert.True(t, stat.IsDir())
}

// Missing sprite asset → ErrMissingAsset with the absolute path.
func TestLoad_MissingSpriteAsset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.pforge")
	p := NewProject("missing")
	p.Sprites = append(p.Sprites, SpriteAsset{
		Name:         "ghost",
		RelativePath: "sprites/ghost.png",
	})
	require.NoError(t, p.Save(path))

	// We saved without the asset file; loading must reject it.
	_, err := Load(path)
	require.Error(t, err)

	var missing *ErrMissingAsset
	require.True(t, errors.As(err, &missing))
	assert.Contains(t, missing.AbsPath, "ghost.png")
}

// A JSON file with no schema_version → ErrNotAPforgeFile.
func TestLoad_NotAPforgeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rando.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"name":"abc"}`), 0o644))

	_, err := Load(path)
	require.Error(t, err)
	var nf *ErrNotAPforgeFile
	require.True(t, errors.As(err, &nf))
	assert.Equal(t, path, nf.Path)
}

// Future-version schema → ErrUnsupportedSchemaVersion.
func TestLoad_FutureSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "future.pforge")
	require.NoError(t, os.WriteFile(path, []byte(`{"schema_version":999}`), 0o644))

	_, err := Load(path)
	require.Error(t, err)
	var uv *ErrUnsupportedSchemaVersion
	require.True(t, errors.As(err, &uv))
	assert.Equal(t, 999, uv.Got)
}

// Modifying a single entity position, saving, re-loading, re-saving
// should leave a one-field diff — no field reordering, no whitespace
// churn elsewhere. We assert this by comparing the two file outputs and
// confirming only the relevant byte range differs.
func TestSave_MinimalDiffOnSmallEdit(t *testing.T) {
	withFrozenClock(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "edit.pforge")

	p := snakeShapedProject(t)
	require.NoError(t, p.Save(path))

	loaded, err := loadWithoutAssetCheck(path)
	require.NoError(t, err)
	loaded.Scenes[0].Entities[0].Position.X = 99
	path2 := filepath.Join(dir, "edit2.pforge")
	require.NoError(t, loaded.Save(path2))

	b1, _ := os.ReadFile(path)
	b2, _ := os.ReadFile(path2)
	// Files differ
	require.NotEqual(t, b1, b2)
	// But they share most lines — at least 95% of bytes are common.
	common := commonByteCount(b1, b2)
	total := len(b1)
	if len(b2) > total {
		total = len(b2)
	}
	require.Greater(t, total, 0)
	ratio := float64(common) / float64(total)
	assert.Greater(t, ratio, 0.95, "single-field edit should leave most bytes unchanged (got %.2f)", ratio)
}

// LoadReader handles in-memory projects without asset validation.
func TestLoadReader(t *testing.T) {
	p := NewProject("ram")
	data, err := encodeProject(p)
	require.NoError(t, err)

	loaded, err := LoadReader(bytesReader(data))
	require.NoError(t, err)
	assert.Equal(t, "ram", loaded.Name)
}

// loadWithoutAssetCheck reads a project but accepts missing assets.
// Used by the round-trip-with-edit test to avoid round-tripping the
// fake-png writes.
func loadWithoutAssetCheck(path string) (*Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	p := &Project{}
	if err := jsonUnmarshal(data, p); err != nil {
		return nil, err
	}
	p.normalizeSlices()
	return p, nil
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, data, 0o644))
}

func commonByteCount(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	c := 0
	for i := 0; i < n; i++ {
		if a[i] == b[i] {
			c++
		}
	}
	return c
}
