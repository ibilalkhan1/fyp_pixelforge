package assetlibrary_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/assetlibrary"
)

// buildLibWithPack returns a Library carrying the supplied pack
// so credits tests can assemble against a known asset set.
func buildLibWithPack(t *testing.T, pack assetlibrary.Pack) *assetlibrary.Library {
	t.Helper()
	lib := assetlibrary.NewLibrary(t.TempDir())
	lib.MarkInstalled(pack)
	return lib
}

func TestAssembleCredits_CCBYIncluded(t *testing.T) {
	lib := buildLibWithPack(t, assetlibrary.Pack{
		ID: "asteroids",
		Assets: []assetlibrary.Asset{
			{Path: "audio/blast.wav", Kind: "sfx", License: "CC-BY-4.0", Author: "freesound user X", SourceURL: "https://freesound.org/x"},
		},
	})
	p := pixelforge_project.NewProject("t")
	p.Audio = []pixelforge_project.AudioSample{
		{Name: "blast", RelativePath: "audio/blast.wav"},
	}
	got := assetlibrary.AssembleCredits(p, lib)
	require.Len(t, got, 1)
	assert.Equal(t, "blast", got[0].Name)
	assert.Equal(t, "CC-BY-4.0", got[0].License)
	assert.Equal(t, "freesound user X", got[0].Author)
	assert.Equal(t, "https://freesound.org/x", got[0].SourceURL)
}

func TestAssembleCredits_CC0Excluded(t *testing.T) {
	lib := buildLibWithPack(t, assetlibrary.Pack{
		ID: "asteroids",
		Assets: []assetlibrary.Asset{
			{Path: "sprites/ship.png", Kind: "sprite", License: "CC0", Author: "Kenney"},
		},
	})
	p := pixelforge_project.NewProject("t")
	p.Sprites = []pixelforge_project.SpriteAsset{
		{Name: "ship", RelativePath: "sprites/ship.png"},
	}
	got := assetlibrary.AssembleCredits(p, lib)
	assert.Empty(t, got, "CC0 assets must be excluded from the credits page")
}

func TestAssembleCredits_MixedLicensesOnlyCCBYIncluded(t *testing.T) {
	lib := buildLibWithPack(t, assetlibrary.Pack{
		ID: "mix",
		Assets: []assetlibrary.Asset{
			{Path: "sprites/ship.png", Kind: "sprite", License: "CC0", Author: "K"},
			{Path: "audio/blast.wav", Kind: "sfx", License: "CC-BY-4.0", Author: "X"},
			{Path: "audio/music.ogg", Kind: "bgm", License: "CC-BY-SA-4.0", Author: "Y"},
		},
	})
	p := pixelforge_project.NewProject("t")
	p.Sprites = []pixelforge_project.SpriteAsset{{Name: "ship", RelativePath: "sprites/ship.png"}}
	p.Audio = []pixelforge_project.AudioSample{
		{Name: "blast", RelativePath: "audio/blast.wav"},
		{Name: "music", RelativePath: "audio/music.ogg"},
	}
	got := assetlibrary.AssembleCredits(p, lib)
	require.Len(t, got, 2)
	names := []string{got[0].Name, got[1].Name}
	assert.Contains(t, names, "blast")
	assert.Contains(t, names, "music")
	assert.NotContains(t, names, "ship", "CC0 sprite must be excluded")
}

func TestAssembleCredits_OrphanedAssetRecordedAsUnknown(t *testing.T) {
	lib := assetlibrary.NewLibrary(t.TempDir())
	// No packs installed → every reference is orphaned.
	p := pixelforge_project.NewProject("t")
	p.Sprites = []pixelforge_project.SpriteAsset{{Name: "userdrop", RelativePath: "custom/userdrop.png"}}
	got := assetlibrary.AssembleCredits(p, lib)
	require.Len(t, got, 1)
	assert.Equal(t, "userdrop", got[0].Name)
	assert.Equal(t, "Unknown", got[0].Author)
}

func TestAssembleCredits_DeduplicatesByName(t *testing.T) {
	lib := buildLibWithPack(t, assetlibrary.Pack{
		ID: "p",
		Assets: []assetlibrary.Asset{
			{Path: "audio/blast.wav", Kind: "sfx", License: "CC-BY-4.0", Author: "X"},
		},
	})
	p := pixelforge_project.NewProject("t")
	// Two entries with the same name (unusual but possible after
	// schema migrations).
	p.Audio = []pixelforge_project.AudioSample{
		{Name: "blast", RelativePath: "audio/blast.wav"},
		{Name: "blast", RelativePath: "audio/blast2.wav"},
	}
	got := assetlibrary.AssembleCredits(p, lib)
	assert.Len(t, got, 1, "duplicate names dedupe to a single credit entry")
}

func TestAssembleCredits_NilInputsReturnNil(t *testing.T) {
	assert.Nil(t, assetlibrary.AssembleCredits(nil, nil))
	assert.Nil(t, assetlibrary.AssembleCredits(nil, assetlibrary.NewLibrary(t.TempDir())))
	assert.Nil(t, assetlibrary.AssembleCredits(pixelforge_project.NewProject("t"), nil))
}

func TestAssembleCredits_EmptyProjectReturnsEmpty(t *testing.T) {
	lib := buildLibWithPack(t, assetlibrary.Pack{ID: "p", Assets: nil})
	p := pixelforge_project.NewProject("t")
	got := assetlibrary.AssembleCredits(p, lib)
	assert.Empty(t, got)
}

func TestAssembleCredits_DeterministicOrder(t *testing.T) {
	lib := buildLibWithPack(t, assetlibrary.Pack{
		ID: "p",
		Assets: []assetlibrary.Asset{
			{Path: "z.png", Kind: "sprite", License: "CC-BY-4.0", Author: "X"},
			{Path: "a.png", Kind: "sprite", License: "CC-BY-4.0", Author: "X"},
		},
	})
	p := pixelforge_project.NewProject("t")
	p.Sprites = []pixelforge_project.SpriteAsset{
		{Name: "z", RelativePath: "z.png"},
		{Name: "a", RelativePath: "a.png"},
	}
	got := assetlibrary.AssembleCredits(p, lib)
	require.Len(t, got, 2)
	assert.Equal(t, "a", got[0].Name)
	assert.Equal(t, "z", got[1].Name)
}
