package palette

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// ImportResult summarises the outcome of an Import call. Used by the
// editor's status messages and the M2 import status modal.
//
// Idea #3 v1 U3 added the optional Diff field so the editor's diff
// modal can render before/after thumbnails + the warning banner.
// Existing callers that ignore Diff stay compatible.
type ImportResult struct {
	SpriteName     string
	RegisteredIdx  int
	FrameW, FrameH int
	Width, Height  int
	CollisionBits  int
	UsedSidecar    bool

	// Diff is populated when the import was run via the diff-aware
	// entry point (ImportWithDiff or ImportPNGForDiff). Nil for
	// callers using the original Import signature.
	Diff *ImportDiff
}

// ImportDiff carries the side-by-side data the U4 diff modal needs:
// the original RGBA, the quantizer's index output, a reconstructed
// preview image (indices mapped back through the palette), the
// auto- or manually-chosen sub-palette, and the mean per-pixel
// distance score that drives the warning banner.
type ImportDiff struct {
	OriginalImage          image.Image
	QuantizedIndices       []byte
	QuantizedReconstructed image.Image
	ChosenSubPalette       string
	AutoPicked             bool
	MeanDeltaE             float64
	Width                  int
	Height                 int
}

// Import runs the palette-aware PNG import pipeline:
//
//   1. decode the PNG,
//   2. detect frame size via alpha-gutter analysis,
//   3. quantize each pixel to the nearest palette slot,
//   4. derive a 1-bit collision mask from the first frame's opaque pixels,
//   5. parse a `<pngPath>.meta` sidecar if it exists,
//   6. copy the asset into the project's *-assets/sprites/ directory,
//   7. register a SpriteAsset entry on the project.
//
// Errors at any step return without mutating the project — the import
// is atomic.
func Import(pngPath string, p *pixelforge_project.Project, projectSourcePath string) (ImportResult, error) {
	if p == nil {
		return ImportResult{}, errors.New("import: nil project")
	}
	if pngPath == "" {
		return ImportResult{}, errors.New("import: empty path")
	}

	img, err := decodePNGFile(pngPath)
	if err != nil {
		return ImportResult{}, err
	}
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	sidecar, sidecarErr := LoadSidecar(pngPath)
	_ = sidecarErr // surfaced via UsedSidecar = false; caller logs

	frameW, frameH := DetectFrames(img)
	if sidecar.FrameW > 0 {
		frameW = sidecar.FrameW
	}
	if sidecar.FrameH > 0 {
		frameH = sidecar.FrameH
	}

	quantized := quantizeImage(p, img)
	collision := deriveCollisionMask(img, frameW, frameH)

	name := spriteNameFromPath(pngPath)
	relPath := filepath.ToSlash(filepath.Join("sprites", filepath.Base(pngPath)))
	if projectSourcePath != "" {
		if err := copyAssetFile(pngPath, projectSourcePath, relPath); err != nil {
			return ImportResult{}, err
		}
	}

	sprite := pixelforge_project.SpriteAsset{
		Name:          name,
		RelativePath:  relPath,
		Width:         w,
		Height:        h,
		FrameW:        frameW,
		FrameH:        frameH,
		CollisionMask: collision,
		Animations:    sidecarAnimations(sidecar),
	}
	// Register on the project.
	p.Sprites = append(p.Sprites, sprite)
	idx := len(p.Sprites) - 1

	// quantized is captured for completeness but not stored on the
	// schema yet — the engine reads the PNG bytes directly. The slice
	// is kept around so future schemas can persist the quantized form.
	_ = quantized

	return ImportResult{
		SpriteName:    name,
		RegisteredIdx: idx,
		FrameW:        frameW,
		FrameH:        frameH,
		Width:         w,
		Height:        h,
		CollisionBits: bitsSet(collision),
		UsedSidecar:   sidecarErr == nil && (sidecar.FrameW > 0 || sidecar.FrameH > 0 || len(sidecar.AnimationClips) > 0),
	}, nil
}

// ImportWithDiff is the diff-aware sibling of Import. It runs the
// same pipeline but additionally builds the ImportDiff payload the
// editor's U4 diff modal needs. When targetSubPalette is empty, the
// pipeline calls PickBestSubPalette to choose one and records the
// auto-pick decision in ImportDiff.AutoPicked.
//
// Sets the new SpriteAsset's SubPalette field to the chosen value so
// the imported sprite is bound to a sub-palette from the moment it
// lands in the project.
func ImportWithDiff(
	pngPath string,
	p *pixelforge_project.Project,
	projectSourcePath string,
	targetSubPalette string,
) (ImportResult, error) {
	if p == nil {
		return ImportResult{}, errors.New("import: nil project")
	}
	if pngPath == "" {
		return ImportResult{}, errors.New("import: empty path")
	}

	img, err := decodePNGFile(pngPath)
	if err != nil {
		return ImportResult{}, err
	}
	return importImageWithDiff(img, pngPath, p, projectSourcePath, targetSubPalette)
}

// ImportImageWithDiff is the in-memory variant of ImportWithDiff —
// the import_handler tests pass synthetic images directly without
// touching the filesystem. pngPath is recorded on the SpriteAsset's
// RelativePath; copyAssetFile is skipped when projectSourcePath is
// empty (the test path) so no disk write happens.
func ImportImageWithDiff(
	img image.Image,
	pngPath string,
	p *pixelforge_project.Project,
	projectSourcePath string,
	targetSubPalette string,
) (ImportResult, error) {
	if p == nil {
		return ImportResult{}, errors.New("import: nil project")
	}
	return importImageWithDiff(img, pngPath, p, projectSourcePath, targetSubPalette)
}

func importImageWithDiff(
	img image.Image,
	pngPath string,
	p *pixelforge_project.Project,
	projectSourcePath string,
	targetSubPalette string,
) (ImportResult, error) {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	autoPicked := false
	if targetSubPalette == "" {
		targetSubPalette, _ = PickBestSubPalette(img, p, FamilySprite)
		autoPicked = true
	}

	sidecar, sidecarErr := LoadSidecar(pngPath)
	frameW, frameH := DetectFrames(img)
	if sidecar.FrameW > 0 {
		frameW = sidecar.FrameW
	}
	if sidecar.FrameH > 0 {
		frameH = sidecar.FrameH
	}

	quantized := quantizeImage(p, img)
	reconstructed := reconstructFromIndices(quantized, w, h, p)
	collision := deriveCollisionMask(img, frameW, frameH)

	name := spriteNameFromPath(pngPath)
	relPath := filepath.ToSlash(filepath.Join("sprites", filepath.Base(pngPath)))
	if projectSourcePath != "" && pngPath != "" {
		if err := copyAssetFile(pngPath, projectSourcePath, relPath); err != nil {
			return ImportResult{}, err
		}
	}

	sprite := pixelforge_project.SpriteAsset{
		Name:          name,
		RelativePath:  relPath,
		Width:         w,
		Height:        h,
		FrameW:        frameW,
		FrameH:        frameH,
		CollisionMask: collision,
		Animations:    sidecarAnimations(sidecar),
		SubPalette:    targetSubPalette,
	}
	p.Sprites = append(p.Sprites, sprite)
	idx := len(p.Sprites) - 1

	deltaE := MeanDistanceToSubPalette(img, p, targetSubPalette)
	diff := &ImportDiff{
		OriginalImage:          img,
		QuantizedIndices:       quantized,
		QuantizedReconstructed: reconstructed,
		ChosenSubPalette:       targetSubPalette,
		AutoPicked:             autoPicked,
		MeanDeltaE:             deltaE,
		Width:                  w,
		Height:                 h,
	}

	return ImportResult{
		SpriteName:    name,
		RegisteredIdx: idx,
		FrameW:        frameW,
		FrameH:        frameH,
		Width:         w,
		Height:        h,
		CollisionBits: bitsSet(collision),
		UsedSidecar:   sidecarErr == nil && (sidecar.FrameW > 0 || sidecar.FrameH > 0 || len(sidecar.AnimationClips) > 0),
		Diff:          diff,
	}, nil
}

// reconstructFromIndices renders the post-quantize palette indices
// back to an RGBA image by walking PaletteData.Base. Used by the
// diff modal as the "after" panel.
func reconstructFromIndices(indices []byte, w, h int, p *pixelforge_project.Project) image.Image {
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := indices[y*w+x]
			if int(idx) >= pixelforge_project.MaxColors {
				idx = 0
			}
			c, ok := parseHexColor(p.Palette.Base[idx])
			if !ok {
				c = color.RGBA{}
			}
			if idx == 0 {
				// Slot 0 is transparent per palette conventions; the
				// reconstructed preview keeps that transparency so
				// the diff modal shows the cutout correctly.
				c.A = 0
			} else {
				c.A = 0xff
			}
			out.Set(x, y, c)
		}
	}
	return out
}

// quantizeImage runs the palette quantizer over the full image.
func quantizeImage(p *pixelforge_project.Project, img image.Image) []byte {
	q := NewQuantizer()
	q.Bind(p)
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	out := make([]byte, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r16, g16, b16, a16 := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			rgba := color.RGBA{
				R: uint8(r16 >> 8),
				G: uint8(g16 >> 8),
				B: uint8(b16 >> 8),
				A: uint8(a16 >> 8),
			}
			out[y*w+x] = q.QuantizePixel(rgba)
		}
	}
	return out
}

// deriveCollisionMask builds a 1-bit-per-pixel mask of the first frame's
// opaque pixels, row-major.
func deriveCollisionMask(img image.Image, frameW, frameH int) []uint8 {
	bounds := img.Bounds()
	if frameW <= 0 || frameH <= 0 {
		return nil
	}
	if frameW > bounds.Dx() {
		frameW = bounds.Dx()
	}
	if frameH > bounds.Dy() {
		frameH = bounds.Dy()
	}
	totalBits := frameW * frameH
	out := make([]uint8, (totalBits+7)/8)
	for y := 0; y < frameH; y++ {
		for x := 0; x < frameW; x++ {
			_, _, _, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			if a > 0 {
				bit := y*frameW + x
				out[bit/8] |= 1 << uint(bit%8)
			}
		}
	}
	return out
}

// bitsSet counts the 1-bits in mask.
func bitsSet(mask []uint8) int {
	n := 0
	for _, b := range mask {
		for b != 0 {
			n += int(b & 1)
			b >>= 1
		}
	}
	return n
}

// sidecarAnimations converts the sidecar clip list to schema entries.
func sidecarAnimations(sc Sidecar) []pixelforge_project.AnimationClip {
	out := make([]pixelforge_project.AnimationClip, 0, len(sc.AnimationClips))
	for _, c := range sc.AnimationClips {
		out = append(out, pixelforge_project.AnimationClip{
			Name:     c.Name,
			Frames:   c.Frames,
			FPS:      c.FPS,
			LoopMode: c.LoopMode,
		})
	}
	return out
}

// spriteNameFromPath strips the extension and directory prefix.
func spriteNameFromPath(p string) string {
	base := filepath.Base(p)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return name
}

// copyAssetFile copies pngPath into the project's *-assets/relPath.
func copyAssetFile(pngPath, projectSourcePath, relPath string) error {
	dir := pixelforge_project.AssetsDir(projectSourcePath)
	dst := filepath.Join(dir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	src, err := os.Open(pngPath)
	if err != nil {
		return err
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, src); err != nil {
		return err
	}
	return nil
}

// decodePNGFile decodes a PNG from disk into an image.Image.
func decodePNGFile(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return img, nil
}
