package palette

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// writePNG persists img to disk as a PNG.
func writePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	require.NoError(t, png.Encode(f, img))
}

// Import of a 4-frame strip auto-detects 8×8 frames.
func TestImport_DetectsFramesAndRegisters(t *testing.T) {
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "snake.png")
	img := stripImage(8, 8, 4, 1) // from frame_strip_test.go
	writePNG(t, pngPath, img)

	p := pixelforge_project.NewProject("t")
	res, err := Import(pngPath, p, "")
	require.NoError(t, err)
	assert.Equal(t, "snake", res.SpriteName)
	assert.Equal(t, 8, res.FrameW)
	assert.Equal(t, 8, res.FrameH)
	require.Len(t, p.Sprites, 1)
	assert.Equal(t, "snake", p.Sprites[0].Name)
	assert.Equal(t, "sprites/snake.png", p.Sprites[0].RelativePath)
}

// Sidecar override beats auto-detection.
func TestImport_SidecarOverridesFrameSize(t *testing.T) {
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "fig.png")
	img := solidImage(32, 32)
	writePNG(t, pngPath, img)
	require.NoError(t, os.WriteFile(pngPath+".meta", []byte(`{"frame_w":16,"frame_h":16}`), 0o644))

	p := pixelforge_project.NewProject("t")
	res, err := Import(pngPath, p, "")
	require.NoError(t, err)
	assert.Equal(t, 16, res.FrameW)
	assert.Equal(t, 16, res.FrameH)
	assert.True(t, res.UsedSidecar)
}

// CollisionMask bits match opaque pixel positions.
func TestImport_CollisionMaskMatchesOpaquePixels(t *testing.T) {
	// 4×4 image with a diagonal of opaque pixels.
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for i := 0; i < 4; i++ {
		img.Set(i, i, color.RGBA{R: 255, A: 255})
	}
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "x.png")
	writePNG(t, pngPath, img)

	p := pixelforge_project.NewProject("t")
	_, err := Import(pngPath, p, "")
	require.NoError(t, err)
	mask := p.Sprites[0].CollisionMask
	for i := 0; i < 4; i++ {
		bit := i*4 + i
		assert.Equal(t, uint8(1<<uint(bit%8)), mask[bit/8]&(1<<uint(bit%8)),
			"diagonal bit %d should be set", bit)
	}
}

// Malformed PNG aborts the import atomically — project is unchanged.
func TestImport_AtomicOnDecodeFailure(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.png")
	require.NoError(t, os.WriteFile(bad, []byte("not a png"), 0o644))

	p := pixelforge_project.NewProject("t")
	_, err := Import(bad, p, "")
	assert.Error(t, err)
	assert.Empty(t, p.Sprites)
}

// Atomic-on-failure for the nil project guard.
func TestImport_NilProjectReturnsError(t *testing.T) {
	_, err := Import("/some/path.png", nil, "")
	assert.Error(t, err)
}

// When projectSourcePath is set, the asset file is copied into
// *-assets/sprites/.
func TestImport_CopiesAssetWhenSourcePathSet(t *testing.T) {
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "frog.png")
	writePNG(t, pngPath, solidImage(4, 4))

	projectDir := t.TempDir()
	projectPath := filepath.Join(projectDir, "game.pforge")

	p := pixelforge_project.NewProject("t")
	_, err := Import(pngPath, p, projectPath)
	require.NoError(t, err)

	want := filepath.Join(pixelforge_project.AssetsDir(projectPath), "sprites", "frog.png")
	info, err := os.Stat(want)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}
