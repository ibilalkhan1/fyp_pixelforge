package assetlibrary_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/assetlibrary"
)

func TestLibrary_NewAndCacheRoot(t *testing.T) {
	lib := assetlibrary.NewLibrary("/some/cache")
	assert.Equal(t, "/some/cache", lib.CacheRoot())
	assert.Empty(t, lib.Installed())
	assert.Nil(t, lib.Manifest())
}

func TestLibrary_MarkInstalledAndIsInstalled(t *testing.T) {
	lib := assetlibrary.NewLibrary(t.TempDir())
	pack := assetlibrary.Pack{ID: "asteroids-starter", Title: "Asteroids", Game: "asteroids"}
	lib.MarkInstalled(pack)
	assert.True(t, lib.IsInstalled("asteroids-starter"))
	assert.False(t, lib.IsInstalled("missing"))
	require.Len(t, lib.Installed(), 1)
	assert.Equal(t, "asteroids-starter", lib.Installed()[0].ID)
}

func TestLibrary_MarkInstalledIgnoresEmptyID(t *testing.T) {
	lib := assetlibrary.NewLibrary(t.TempDir())
	lib.MarkInstalled(assetlibrary.Pack{ID: ""})
	assert.Empty(t, lib.Installed())
}

func TestLibrary_PacksForGameFilters(t *testing.T) {
	lib := assetlibrary.NewLibrary(t.TempDir())
	lib.MarkInstalled(assetlibrary.Pack{ID: "asteroids", Game: "asteroids"})
	lib.MarkInstalled(assetlibrary.Pack{ID: "mario", Game: "mario"})
	lib.MarkInstalled(assetlibrary.Pack{ID: "shared", Game: ""})

	assert.Len(t, lib.PacksForGame("asteroids"), 1)
	assert.Len(t, lib.PacksForGame("mario"), 1)
	assert.Len(t, lib.PacksForGame(""), 3, "empty game filter returns every installed pack")
	assert.Len(t, lib.PacksForGame("all"), 3, "alias 'all' also returns every pack")
	assert.Empty(t, lib.PacksForGame("custom"), "custom routes to the user-library, not curated packs")
	assert.Empty(t, lib.PacksForGame("nonexistent"))
}

func TestLibrary_LookupAssetResolvesByPackAndPath(t *testing.T) {
	lib := assetlibrary.NewLibrary(t.TempDir())
	lib.MarkInstalled(assetlibrary.Pack{
		ID: "asteroids", Game: "asteroids",
		Assets: []assetlibrary.Asset{
			{Path: "sprites/ship.png", Kind: "sprite", License: "CC0", Author: "Kenney"},
		},
	})
	got := lib.LookupAsset("asteroids", "sprites/ship.png")
	require.NotNil(t, got)
	assert.Equal(t, "CC0", got.License)
	assert.Nil(t, lib.LookupAsset("asteroids", "missing.png"))
	assert.Nil(t, lib.LookupAsset("missing-pack", "sprites/ship.png"))
}

func TestLibrary_FindAssetMatchesByBareName(t *testing.T) {
	lib := assetlibrary.NewLibrary(t.TempDir())
	lib.MarkInstalled(assetlibrary.Pack{
		ID: "asteroids",
		Assets: []assetlibrary.Asset{
			{Path: "sprites/ship.png", Kind: "sprite", License: "CC0"},
			{Path: "audio/blast.wav", Kind: "sfx", License: "CC-BY-4.0", Author: "X"},
		},
	})
	packID, asset, ok := lib.FindAsset("ship")
	require.True(t, ok)
	assert.Equal(t, "asteroids", packID)
	assert.Equal(t, "CC0", asset.License)

	packID, asset, ok = lib.FindAsset("blast")
	require.True(t, ok)
	assert.Equal(t, "CC-BY-4.0", asset.License)

	_, _, ok = lib.FindAsset("missing")
	assert.False(t, ok)
}

func TestLibrary_SetManifestRecords(t *testing.T) {
	lib := assetlibrary.NewLibrary(t.TempDir())
	m := &assetlibrary.Manifest{SchemaVersion: "1"}
	lib.SetManifest(m)
	assert.Same(t, m, lib.Manifest())
}

func TestLibrary_NilLibraryIsSafe(t *testing.T) {
	var lib *assetlibrary.Library
	assert.False(t, lib.IsInstalled("x"))
	assert.Empty(t, lib.Installed())
	assert.Nil(t, lib.LookupAsset("x", "y"))
	assert.Empty(t, lib.CacheRoot())
	assert.Nil(t, lib.Pack("x"))
	assert.Empty(t, lib.Packs())
	assert.False(t, lib.IsEmbedded("x"))
	src, ok := lib.EmbeddedFS("x")
	assert.Nil(t, src)
	assert.False(t, ok)
}

// Plan-009 U20: the Library now distinguishes between embedded
// packs (always-available, byte-content via fs.FS) and downloaded
// packs (registered via the bootstrap, byte-content on disk).

func TestLibrary_MarkEmbeddedRegistersPackAndFS(t *testing.T) {
	lib := assetlibrary.NewLibrary(t.TempDir())
	starter := assetlibrary.Pack{
		ID:      "starter",
		Version: "1.0.0",
		Title:   "Starter Pack",
		Assets: []assetlibrary.Asset{
			{Path: "sprites/hero.png", Kind: "sprite", License: "CC0"},
		},
	}
	src := fstest.MapFS{
		"sprites/hero.png": &fstest.MapFile{Data: []byte("png-bytes")},
	}
	lib.MarkEmbedded(starter, src)

	// Pack metadata is present in the unified packs list.
	assert.True(t, lib.IsInstalled("starter"))
	assert.True(t, lib.IsEmbedded("starter"))
	require.Len(t, lib.Packs(), 1)
	assert.Equal(t, "starter", lib.Packs()[0].ID)

	// And the fs.FS is reachable so callers can hand back bytes.
	got, ok := lib.EmbeddedFS("starter")
	require.True(t, ok)
	b, err := fs.ReadFile(got, "sprites/hero.png")
	require.NoError(t, err)
	assert.Equal(t, []byte("png-bytes"), b)
}

func TestLibrary_PackReturnsDefensiveCopy(t *testing.T) {
	lib := assetlibrary.NewLibrary(t.TempDir())
	lib.MarkInstalled(assetlibrary.Pack{
		ID:    "p",
		Title: "Original",
		Assets: []assetlibrary.Asset{
			{Path: "a.png", Kind: "sprite", License: "CC0", Author: "kenney"},
		},
	})
	got := lib.Pack("p")
	require.NotNil(t, got)
	got.Title = "Tampered"
	got.Assets[0].License = "Tampered"

	// Internal state must not have moved.
	fresh := lib.Pack("p")
	assert.Equal(t, "Original", fresh.Title)
	assert.Equal(t, "CC0", fresh.Assets[0].License)
	assert.Nil(t, lib.Pack("missing"))
}

func TestLibrary_PacksAliasesInstalled(t *testing.T) {
	lib := assetlibrary.NewLibrary(t.TempDir())
	lib.MarkInstalled(assetlibrary.Pack{ID: "a"})
	lib.MarkEmbedded(assetlibrary.Pack{ID: "starter"}, nil)
	assert.ElementsMatch(t,
		[]string{"a", "starter"},
		[]string{lib.Packs()[0].ID, lib.Packs()[1].ID})
}

func TestLibrary_MarkEmbeddedNilSourceStillRegistersMetadata(t *testing.T) {
	// Tests can register metadata without bytes; EmbeddedFS then
	// reports "no FS available" while the pack still shows up in
	// Packs(). Useful for tests that don't need to read bytes.
	lib := assetlibrary.NewLibrary(t.TempDir())
	lib.MarkEmbedded(assetlibrary.Pack{ID: "meta-only"}, nil)
	assert.True(t, lib.IsInstalled("meta-only"))
	assert.False(t, lib.IsEmbedded("meta-only"), "nil FS means EmbeddedFS reports absent")
	_, ok := lib.EmbeddedFS("meta-only")
	assert.False(t, ok)
}

func TestLibrary_MarkEmbeddedIgnoresEmptyID(t *testing.T) {
	lib := assetlibrary.NewLibrary(t.TempDir())
	lib.MarkEmbedded(assetlibrary.Pack{ID: ""}, fstest.MapFS{})
	assert.Empty(t, lib.Packs())
}
