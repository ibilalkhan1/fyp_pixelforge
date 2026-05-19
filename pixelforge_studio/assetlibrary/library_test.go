package assetlibrary_test

import (
	"testing"

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
}
