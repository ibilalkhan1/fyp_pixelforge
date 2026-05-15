package capture

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeProjectWithSprite(t *testing.T) (string, *pixelforge_project.Project) {
	t.Helper()
	tmp := t.TempDir()
	pforge := filepath.Join(tmp, "p.pforge")
	spritesDir := filepath.Join(pixelforge_project.AssetsDir(pforge), "sprites")
	require.NoError(t, os.MkdirAll(spritesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(spritesDir, "hero.png"), []byte("fakepng"), 0o644))
	p := pixelforge_project.NewProject("test")
	p.Sprites = append(p.Sprites, pixelforge_project.SpriteAsset{
		Name:         "hero",
		RelativePath: "sprites/hero.png",
		Width:        16,
		Height:       16,
		FrameW:       16,
		FrameH:       16,
	})
	return pforge, p
}

func TestPromoteRangeToClip_WritesStripAndAppendsClip(t *testing.T) {
	pixelforge.SetScreenSize(16, 16)
	rec := New(8)
	for i := 0; i < 5; i++ {
		pixelforge.SetColor(pixelforge.Color(i + 1))
		pixelforge.RectFill(0, 0, 15, 15)
		rec.SaveFrame()
	}
	pforge, project := makeProjectWithSprite(t)
	abs, err := PromoteRangeToClip(rec, 1, 4, pforge, "hero", "walk", project)
	require.NoError(t, err)

	// Strip on disk.
	_, err = os.Stat(abs)
	require.NoError(t, err)
	f, err := os.Open(abs)
	require.NoError(t, err)
	defer f.Close()
	img, err := png.Decode(f)
	require.NoError(t, err)
	// 4 frames × 16 wide = 64; 16 tall.
	assert.Equal(t, image.Rect(0, 0, 64, 16), img.Bounds())

	// AnimationClip appended.
	require.Len(t, project.Sprites[0].Animations, 1)
	clip := project.Sprites[0].Animations[0]
	assert.Equal(t, "walk", clip.Name)
	assert.Equal(t, []int{0, 1, 2, 3}, clip.Frames)
	assert.Equal(t, "hero/walk.png", clip.ClipPath)
}

func TestPromoteRangeToClip_OverwritesExisting(t *testing.T) {
	pixelforge.SetScreenSize(16, 16)
	rec := New(8)
	for i := 0; i < 3; i++ {
		rec.SaveFrame()
	}
	pforge, project := makeProjectWithSprite(t)

	_, err := PromoteRangeToClip(rec, 0, 1, pforge, "hero", "walk", project)
	require.NoError(t, err)
	_, err = PromoteRangeToClip(rec, 0, 2, pforge, "hero", "walk", project)
	require.NoError(t, err)
	// Still one clip — the second call replaced it.
	require.Len(t, project.Sprites[0].Animations, 1)
	assert.Equal(t, 3, len(project.Sprites[0].Animations[0].Frames))
}

func TestPromoteRangeToClip_SwapsOutOfOrderRange(t *testing.T) {
	pixelforge.SetScreenSize(16, 16)
	rec := New(8)
	for i := 0; i < 5; i++ {
		rec.SaveFrame()
	}
	pforge, project := makeProjectWithSprite(t)
	_, err := PromoteRangeToClip(rec, 3, 1, pforge, "hero", "walk", project)
	require.NoError(t, err)
	assert.Equal(t, 3, len(project.Sprites[0].Animations[0].Frames))
}

func TestPromoteRangeToClip_RejectsSpriteWithoutFrameDims(t *testing.T) {
	pixelforge.SetScreenSize(16, 16)
	rec := New(8)
	rec.SaveFrame()
	rec.SaveFrame()
	pforge, project := makeProjectWithSprite(t)
	project.Sprites[0].FrameW = 0
	_, err := PromoteRangeToClip(rec, 0, 1, pforge, "hero", "walk", project)
	assert.ErrorIs(t, err, ErrSpriteFrameSize)
}

func TestPromoteRangeToClip_RejectsEmptyRange(t *testing.T) {
	pixelforge.SetScreenSize(16, 16)
	rec := New(8)
	rec.SaveFrame()
	pforge, project := makeProjectWithSprite(t)
	_, err := PromoteRangeToClip(rec, 0, 0, pforge, "hero", "walk", project)
	assert.Error(t, err)
}

func TestPromoteRangeToClip_RejectsUnknownSprite(t *testing.T) {
	pixelforge.SetScreenSize(16, 16)
	rec := New(8)
	rec.SaveFrame()
	rec.SaveFrame()
	pforge, project := makeProjectWithSprite(t)
	_, err := PromoteRangeToClip(rec, 0, 1, pforge, "missing-sprite", "walk", project)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing-sprite")
}

func TestPromoteRangeToClip_RoundTripsThroughLoader(t *testing.T) {
	pixelforge.SetScreenSize(16, 16)
	rec := New(8)
	for i := 0; i < 4; i++ {
		rec.SaveFrame()
	}
	pforge, project := makeProjectWithSprite(t)
	_, err := PromoteRangeToClip(rec, 0, 2, pforge, "hero", "walk", project)
	require.NoError(t, err)
	require.NoError(t, project.Save(pforge))

	loaded, err := pixelforge_project.Load(pforge)
	require.NoError(t, err)
	require.Len(t, loaded.Sprites[0].Animations, 1)
	assert.Equal(t, "walk", loaded.Sprites[0].Animations[0].Name)
	assert.Equal(t, "hero/walk.png", loaded.Sprites[0].Animations[0].ClipPath)
}
