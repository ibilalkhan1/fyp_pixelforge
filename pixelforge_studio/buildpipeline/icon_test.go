package buildpipeline_test

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/buildpipeline"
)

func TestResolveIconSprite_DesignerMarkedWins(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Sprites = []pixelforge_project.SpriteAsset{
		{Name: "hero", FrameW: 16, FrameH: 16},
		{Name: "enemy", FrameW: 16, FrameH: 16},
	}
	p.IconSpriteName = "enemy"
	got := buildpipeline.ResolveIconSprite(p)
	require.NotNil(t, got)
	assert.Equal(t, "enemy", got.Name)
}

func TestResolveIconSprite_AutoPickByReferenceCount(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Sprites = []pixelforge_project.SpriteAsset{
		{Name: "hero"}, {Name: "rare"}, {Name: "common"},
	}
	p.Items = []pixelforge_project.ItemDefinition{
		{ID: "potion", Icon: "common"}, {ID: "elixir", Icon: "common"},
		{ID: "sword", Icon: "common"},
	}
	got := buildpipeline.ResolveIconSprite(p)
	require.NotNil(t, got)
	assert.Equal(t, "common", got.Name)
}

func TestResolveIconSprite_TiebreakByAlphabetical(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Sprites = []pixelforge_project.SpriteAsset{
		{Name: "zebra"}, {Name: "apple"}, {Name: "mango"},
	}
	got := buildpipeline.ResolveIconSprite(p)
	require.NotNil(t, got)
	assert.Equal(t, "apple", got.Name)
}

func TestResolveIconSprite_Prefers16x16OverLarger(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Sprites = []pixelforge_project.SpriteAsset{
		{Name: "big", FrameW: 32, FrameH: 32},
		{Name: "small", FrameW: 16, FrameH: 16},
	}
	got := buildpipeline.ResolveIconSprite(p)
	require.NotNil(t, got)
	assert.Equal(t, "small", got.Name)
}

func TestResolveIconSprite_EmptyProjectReturnsNil(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Sprites = nil
	assert.Nil(t, buildpipeline.ResolveIconSprite(p))
}

func TestResolveIconSprite_DesignerMarkedNonExistentFallsThrough(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Sprites = []pixelforge_project.SpriteAsset{{Name: "hero"}}
	p.IconSpriteName = "ghost"
	got := buildpipeline.ResolveIconSprite(p)
	require.NotNil(t, got)
	assert.Equal(t, "hero", got.Name,
		"dangling IconSpriteName falls through to auto-pick")
}

// TestGenerateFavicon_ReturnsBase64PNG covers the U5 invariant —
// the no-arg favicon is the rasterised brand logo, always 32×32,
// always a valid base64 PNG.
func TestGenerateFavicon_ReturnsBase64PNG(t *testing.T) {
	got, err := buildpipeline.GenerateFavicon()
	require.NoError(t, err)
	assert.NotEmpty(t, got)
	raw, err := base64.StdEncoding.DecodeString(got)
	require.NoError(t, err)
	img, err := png.Decode(strings.NewReader(string(raw)))
	require.NoError(t, err)
	assert.Equal(t, 32, img.Bounds().Dx())
	assert.Equal(t, 32, img.Bounds().Dy())
}

func TestGenerateIconResult_PopulatesAllFields(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Sprites = []pixelforge_project.SpriteAsset{{Name: "hero", FrameW: 16, FrameH: 16}}
	p.IconSpriteName = "hero"
	res, err := buildpipeline.GenerateIconResult(p)
	require.NoError(t, err)
	require.NotNil(t, res.Sprite)
	assert.Equal(t, "hero", res.Sprite.Name)
	assert.NotEmpty(t, res.FaviconBase64)
	assert.Equal(t, "logo.svg", res.Note,
		"U5 sets Note to logo.svg for every project — brand mark is global")
}

func TestGenerateIconResult_NoSpritesStillReturnsFavicon(t *testing.T) {
	p := pixelforge_project.NewProject("t")
	p.Sprites = nil
	res, err := buildpipeline.GenerateIconResult(p)
	require.NoError(t, err)
	assert.Nil(t, res.Sprite, "no sprites = no resolved Sprite, but logo favicon still ships")
	assert.NotEmpty(t, res.FaviconBase64)
}

// TestRasterLogoPNG_DecodesAtRequestedSize covers the per-size
// raster contract — the favicon path (32) plus a couple container
// sizes (16, 256) all produce a valid PNG of the requested size.
func TestRasterLogoPNG_DecodesAtRequestedSize(t *testing.T) {
	buildpipeline.ResetLogoCacheForTest()
	for _, size := range []int{16, 32, 256} {
		raw, err := buildpipeline.RasterLogoPNG(size)
		require.NoError(t, err, "size %d", size)
		img, err := png.Decode(bytes.NewReader(raw))
		require.NoError(t, err, "size %d decode", size)
		assert.Equal(t, size, img.Bounds().Dx(), "raster width @ %d", size)
		assert.Equal(t, size, img.Bounds().Dy(), "raster height @ %d", size)
	}
}

// TestRasterLogoPNG_ZeroSizeReturnsError guards the precondition.
func TestRasterLogoPNG_ZeroSizeReturnsError(t *testing.T) {
	_, err := buildpipeline.RasterLogoPNG(0)
	require.Error(t, err)
	_, err = buildpipeline.RasterLogoPNG(-5)
	require.Error(t, err)
}

// TestRasterLogoPNG_CacheReturnsSameBytes asserts the per-size
// cache returns byte-equal output across calls (idempotency the
// build pipeline relies on — re-rasterising would burn cycles on
// every build click).
func TestRasterLogoPNG_CacheReturnsSameBytes(t *testing.T) {
	buildpipeline.ResetLogoCacheForTest()
	first, err := buildpipeline.RasterLogoPNG(32)
	require.NoError(t, err)
	second, err := buildpipeline.RasterLogoPNG(32)
	require.NoError(t, err)
	assert.True(t, bytes.Equal(first, second), "cached rasters must be byte-identical")
}

// TestBuildLogoICO_ReturnsNonEmptyBytes asserts the .ico path
// produces non-empty output. A deeper test (multi-size container
// inspection) is out of scope until the Windows host builder
// integration test in U3 runs end-to-end.
func TestBuildLogoICO_ReturnsNonEmptyBytes(t *testing.T) {
	got, err := buildpipeline.BuildLogoICO()
	require.NoError(t, err)
	assert.NotEmpty(t, got)
}

// TestBuildLogoICNS_ReturnsNonEmptyBytes mirrors the .ico path
// for macOS.
func TestBuildLogoICNS_ReturnsNonEmptyBytes(t *testing.T) {
	got, err := buildpipeline.BuildLogoICNS()
	require.NoError(t, err)
	assert.NotEmpty(t, got)
}

// TestBuildWindowsSyso_WritesSysoAndIco covers the Windows
// host-builder seam: BuildWindowsSyso drops both rsrc_windows_<arch>.syso
// and the source .ico next to it. The Windows Go linker picks
// the .syso up automatically when `go build` runs in the dir.
func TestBuildWindowsSyso_WritesSysoAndIco(t *testing.T) {
	dir := t.TempDir()
	err := buildpipeline.BuildWindowsSyso(dir, "Hero's Quest", "1.0.0", "amd64")
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, "rsrc_windows_amd64.syso"))
	require.NoError(t, err, "rsrc_windows_amd64.syso must exist")
	_, err = os.Stat(filepath.Join(dir, "heros_quest.ico"))
	require.NoError(t, err, "<gameName>.ico must exist next to the .syso")
}

// TestBuildWindowsSyso_EmptyOutDirErrors guards the precondition.
func TestBuildWindowsSyso_EmptyOutDirErrors(t *testing.T) {
	err := buildpipeline.BuildWindowsSyso("", "Game", "1.0", "amd64")
	require.Error(t, err)
}

// TestWindowsIcoStub_ReturnsRealBytesAfterU5 — after plan-008 U5
// the stub returns real .ico bytes; IsIconUnsupported must report
// false for successful generation.
func TestWindowsIcoStub_ReturnsRealBytesAfterU5(t *testing.T) {
	got, err := buildpipeline.GenerateWindowsIcoStub(&pixelforge_project.SpriteAsset{})
	require.NoError(t, err)
	assert.NotEmpty(t, got)
	assert.False(t, buildpipeline.IsIconUnsupported(err),
		"after U5 the .ico path no longer returns the legacy sentinel")
}

// TestMacIcnsStub_ReturnsRealBytesAfterU5 mirrors the Windows path.
func TestMacIcnsStub_ReturnsRealBytesAfterU5(t *testing.T) {
	got, err := buildpipeline.GenerateMacIcnsStub(&pixelforge_project.SpriteAsset{})
	require.NoError(t, err)
	assert.NotEmpty(t, got)
	assert.False(t, buildpipeline.IsIconUnsupported(err))
}
