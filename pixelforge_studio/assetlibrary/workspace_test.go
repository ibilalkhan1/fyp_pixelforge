package assetlibrary_test

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/assetlibrary"
)

// fakeEditor satisfies assetlibrary.EditorBinding so workspace
// tests don't drag in the editor package.
type fakeEditor struct {
	project     *pixelforge_project.Project
	projectPath string
	dirty       bool
}

func (f *fakeEditor) Project() *pixelforge_project.Project { return f.project }
func (f *fakeEditor) CurrentProjectPath() string           { return f.projectPath }
func (f *fakeEditor) MarkDirty()                           { f.dirty = true }

// stagePack writes a fake pack to disk under cacheRoot, populates
// the supplied library, and returns the manifest entry the
// workspace will see.
func stagePack(t *testing.T, cacheRoot string, lib *assetlibrary.Library, pack assetlibrary.Pack, files map[string][]byte) {
	t.Helper()
	packDir := assetlibrary.PackDir(cacheRoot, pack.ID)
	require.NoError(t, os.MkdirAll(packDir, 0o755))
	for relPath, data := range files {
		dst := filepath.Join(packDir, relPath)
		require.NoError(t, os.MkdirAll(filepath.Dir(dst), 0o755))
		require.NoError(t, os.WriteFile(dst, data, 0o644))
	}
	lib.MarkInstalled(pack)
}

func setupWorkspace(t *testing.T) (*assetlibrary.Workspace, *fakeEditor, *assetlibrary.Library, string) {
	t.Helper()
	cacheRoot := t.TempDir()
	projectDir := t.TempDir()
	projectPath := filepath.Join(projectDir, "game.pforge")
	require.NoError(t, os.WriteFile(projectPath, []byte("{}"), 0o644))

	lib := assetlibrary.NewLibrary(cacheRoot)
	editor := &fakeEditor{
		project:     pixelforge_project.NewProject("test"),
		projectPath: projectPath,
	}
	ws := assetlibrary.NewWorkspace(editor, lib)
	return ws, editor, lib, cacheRoot
}

func TestWorkspace_NameAndDisplayName(t *testing.T) {
	ws, _, _, _ := setupWorkspace(t)
	assert.Equal(t, "library", ws.Name())
	assert.Equal(t, "Library", ws.DisplayName())
}

func TestWorkspace_DefaultActiveTabIsAll(t *testing.T) {
	ws, _, _, _ := setupWorkspace(t)
	assert.Equal(t, "all", ws.ActiveTab())
}

func TestWorkspace_SetActiveTabKnownValuesPersist(t *testing.T) {
	ws, _, _, _ := setupWorkspace(t)
	ws.SetActiveTab("mario")
	assert.Equal(t, "mario", ws.ActiveTab())
	ws.SetActiveTab("custom")
	assert.Equal(t, "custom", ws.ActiveTab())
}

func TestWorkspace_SetActiveTabUnknownValueIgnored(t *testing.T) {
	ws, _, _, _ := setupWorkspace(t)
	ws.SetActiveTab("mario")
	ws.SetActiveTab("not-a-real-tab")
	assert.Equal(t, "mario", ws.ActiveTab(), "unknown tabs must not displace the prior selection")
}

func TestWorkspace_PacksForActiveTabFiltersByGame(t *testing.T) {
	ws, _, lib, cacheRoot := setupWorkspace(t)
	stagePack(t, cacheRoot, lib, assetlibrary.Pack{ID: "asteroids", Game: "asteroids"}, nil)
	stagePack(t, cacheRoot, lib, assetlibrary.Pack{ID: "mario", Game: "mario"}, nil)

	ws.SetActiveTab("asteroids")
	got := ws.PacksForActiveTab()
	require.Len(t, got, 1)
	assert.Equal(t, "asteroids", got[0].ID)

	ws.SetActiveTab("all")
	assert.Len(t, ws.PacksForActiveTab(), 2)
}

func TestWorkspace_AddToProjectCopiesSpriteAndRecordsEntry(t *testing.T) {
	ws, ed, lib, cacheRoot := setupWorkspace(t)
	stagePack(t, cacheRoot, lib, assetlibrary.Pack{
		ID: "asteroids", Game: "asteroids",
		Assets: []assetlibrary.Asset{
			{Path: "sprites/ship.png", Kind: "sprite", License: "CC0", Author: "Kenney"},
		},
	}, map[string][]byte{"sprites/ship.png": []byte("png-bytes")})

	require.NoError(t, ws.AddToProject("asteroids", "sprites/ship.png"))
	// Project sprite recorded.
	require.Len(t, ed.project.Sprites, 1)
	assert.Equal(t, "ship", ed.project.Sprites[0].Name)
	assert.Equal(t, "sprites/ship.png", ed.project.Sprites[0].RelativePath)
	assert.True(t, ed.dirty, "AddToProject must mark the editor dirty")
	// File copied into project's *-assets/ dir.
	assetsDir := pixelforge_project.AssetsDir(ed.projectPath)
	data, err := os.ReadFile(filepath.Join(assetsDir, "sprites", "ship.png"))
	require.NoError(t, err)
	assert.Equal(t, "png-bytes", string(data))
}

func TestWorkspace_AddToProjectIsIdempotent(t *testing.T) {
	ws, ed, lib, cacheRoot := setupWorkspace(t)
	stagePack(t, cacheRoot, lib, assetlibrary.Pack{
		ID: "p",
		Assets: []assetlibrary.Asset{
			{Path: "sprites/ship.png", Kind: "sprite", License: "CC0"},
		},
	}, map[string][]byte{"sprites/ship.png": []byte("x")})

	require.NoError(t, ws.AddToProject("p", "sprites/ship.png"))
	require.NoError(t, ws.AddToProject("p", "sprites/ship.png"))
	assert.Len(t, ed.project.Sprites, 1,
		"second AddToProject must not duplicate the SpriteAsset record")
}

func TestWorkspace_AddToProjectAudioRecordsAudioSample(t *testing.T) {
	ws, ed, lib, cacheRoot := setupWorkspace(t)
	stagePack(t, cacheRoot, lib, assetlibrary.Pack{
		ID: "p",
		Assets: []assetlibrary.Asset{
			{Path: "audio/blast.wav", Kind: "sfx", License: "CC-BY-4.0", Author: "X"},
			{Path: "audio/music.ogg", Kind: "bgm", License: "CC-BY-4.0", Author: "Y"},
		},
	}, map[string][]byte{
		"audio/blast.wav": []byte("wav"),
		"audio/music.ogg": []byte("ogg"),
	})

	require.NoError(t, ws.AddToProject("p", "audio/blast.wav"))
	require.NoError(t, ws.AddToProject("p", "audio/music.ogg"))
	require.Len(t, ed.project.Audio, 2)
	names := map[string]string{}
	for _, a := range ed.project.Audio {
		names[a.Name] = a.SuggestedChannelPriority
	}
	assert.Equal(t, "sfx", names["blast"])
	assert.Equal(t, "bgm", names["music"])
}

func TestWorkspace_AddToProjectUnknownAssetErrors(t *testing.T) {
	ws, _, _, _ := setupWorkspace(t)
	err := ws.AddToProject("nope", "missing.png")
	require.Error(t, err)
	assert.True(t, errors.Is(err, assetlibrary.ErrUnknownAsset))
}

func TestWorkspace_AddToProjectMissingEditorErrors(t *testing.T) {
	lib := assetlibrary.NewLibrary(t.TempDir())
	ws := assetlibrary.NewWorkspace(nil, lib)
	err := ws.AddToProject("any", "sprites/x.png")
	require.Error(t, err)
	assert.True(t, errors.Is(err, assetlibrary.ErrNoEditor))
}

func TestWorkspace_CustomAssetsReadsUserLibraryDir(t *testing.T) {
	ws, _, lib, cacheRoot := setupWorkspace(t)
	userDir := assetlibrary.UserLibraryDir(cacheRoot)
	require.NoError(t, os.MkdirAll(userDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(userDir, "drop1.png"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(userDir, "drop2.wav"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(userDir, "ignored.txt"), []byte("x"), 0o644))
	_ = lib

	got := ws.CustomAssets()
	require.Len(t, got, 2)
	names := []string{got[0].Path, got[1].Path}
	assert.Contains(t, names, "drop1.png")
	assert.Contains(t, names, "drop2.wav")
}

func TestWorkspace_CustomAssetsEmptyDirReturnsNil(t *testing.T) {
	ws, _, _, _ := setupWorkspace(t)
	assert.Empty(t, ws.CustomAssets(),
		"missing user-library dir returns nil (not an error)")
}

func TestLicenseBadge(t *testing.T) {
	assert.Equal(t, "CC0", assetlibrary.LicenseBadge("CC0"))
	assert.Equal(t, "CC-BY-4.0", assetlibrary.LicenseBadge("CC-BY-4.0"))
	assert.Equal(t, "?", assetlibrary.LicenseBadge(""))
}

func TestCanonicalGameTabs_Order(t *testing.T) {
	want := []string{"all", "asteroids", "bomberman", "mario", "donkey_kong", "custom"}
	assert.Equal(t, want, assetlibrary.CanonicalGameTabs)
}

func TestTabDisplayName(t *testing.T) {
	assert.Equal(t, "All", assetlibrary.TabDisplayName("all"))
	assert.Equal(t, "Mario", assetlibrary.TabDisplayName("mario"))
	assert.Equal(t, "Donkey Kong", assetlibrary.TabDisplayName("donkey_kong"))
	assert.Equal(t, "Custom", assetlibrary.TabDisplayName("custom"))
	assert.Equal(t, "unknown", assetlibrary.TabDisplayName("unknown"))
}

func TestPreviewSpriteThumb_DecodesPNG(t *testing.T) {
	cacheRoot := t.TempDir()
	packDir := assetlibrary.PackDir(cacheRoot, "p")
	require.NoError(t, os.MkdirAll(filepath.Join(packDir, "sprites"), 0o755))

	// Build a real 4x4 PNG with image/png so the test isn't
	// hostage to a hand-crafted byte sequence.
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	src.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, src))
	require.NoError(t, os.WriteFile(filepath.Join(packDir, "sprites", "x.png"), buf.Bytes(), 0o644))

	img, err := assetlibrary.PreviewSpriteThumb(cacheRoot, "p", "sprites/x.png")
	require.NoError(t, err)
	assert.Equal(t, 4, img.Bounds().Dx())
	assert.Equal(t, 4, img.Bounds().Dy())
}

func TestPreviewSpriteThumb_MissingFileErrors(t *testing.T) {
	_, err := assetlibrary.PreviewSpriteThumb(t.TempDir(), "missing", "x.png")
	require.Error(t, err)
}
