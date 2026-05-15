package editor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// Sprites/Audio expose the project's catalogs in declaration order.
func TestAssetBrowser_SpritesAndAudio(t *testing.T) {
	b := NewAssetBrowser()
	p := pixelforge_project.NewProject("t")
	p.Sprites = []pixelforge_project.SpriteAsset{
		{Name: "snake", Width: 16, Height: 16, FrameW: 8, FrameH: 8},
		{Name: "fruit", Width: 8, Height: 8, FrameW: 8, FrameH: 8},
	}
	p.Audio = []pixelforge_project.AudioSample{
		{Name: "eat"},
	}
	assert.Equal(t, []string{"snake", "fruit"}, spriteNames(b.Sprites(p)))
	assert.Len(t, b.Audio(p), 1)
}

// Hit-test helpers return the correct asset names.
func TestAssetBrowser_HitTest(t *testing.T) {
	b := NewAssetBrowser()
	p := pixelforge_project.NewProject("t")
	p.Sprites = []pixelforge_project.SpriteAsset{{Name: "a"}, {Name: "b"}}
	p.Audio = []pixelforge_project.AudioSample{{Name: "x"}}

	assert.Equal(t, "a", b.HitTestSprite(p, 0))
	assert.Equal(t, "b", b.HitTestSprite(p, 1))
	assert.Equal(t, "", b.HitTestSprite(p, 2))
	assert.Equal(t, "x", b.HitTestAudio(p, 0))
	assert.Equal(t, "", b.HitTestAudio(p, 99))
}

// Setting a project clears the thumbnail cache.
func TestAssetBrowser_InvalidateCacheOnSetProject(t *testing.T) {
	e := New()
	e.assetBrowser.thumbnails["foo"] = nil
	require.Len(t, e.assetBrowser.thumbnails, 1)

	e.SetProject(pixelforge_project.NewProject("fresh"))
	assert.Len(t, e.assetBrowser.thumbnails, 0)
}

// SetSelectedSpriteName + getter round-trip through the editor and
// the asset browser drives the Place tool's source sprite.
func TestEditor_SelectedSpriteNameRoundTrip(t *testing.T) {
	e := New()
	assert.Equal(t, "", e.SelectedSpriteName())
	e.SetSelectedSpriteName("fruit")
	assert.Equal(t, "fruit", e.SelectedSpriteName())
}

func spriteNames(sprites []pixelforge_project.SpriteAsset) []string {
	out := make([]string, len(sprites))
	for i, s := range sprites {
		out[i] = s.Name
	}
	return out
}
