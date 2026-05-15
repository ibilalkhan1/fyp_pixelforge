package capture

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// ErrSpriteFrameSize is returned when the target sprite hasn't set
// FrameW/FrameH; the cliplet promoter relies on those to size each
// frame in the strip.
var ErrSpriteFrameSize = errors.New("sprite has no FrameW/FrameH; set them before promoting a cliplet")

// PromoteRangeToClip captures the frames in [start, end] (inclusive) and
// writes them as a horizontal strip PNG into the project's *-assets/
// directory, then appends an AnimationClip to the named sprite.
//
//   - projectPath is the .pforge path on disk; the strip lands at
//     <projectAssets>/<sprite>/<clipName>.png
//   - spriteName is the SpriteAsset whose .Animations field gains the
//     new clip
//   - clipName is the user-supplied clip identifier; existing clips with
//     the same name are overwritten in place
//
// Returns the absolute path of the strip PNG on success.
func PromoteRangeToClip(rec *Recorder, start, end int, projectPath, spriteName, clipName string, project *pixelforge_project.Project) (string, error) {
	if rec == nil {
		return "", fmt.Errorf("recorder is nil")
	}
	if project == nil {
		return "", fmt.Errorf("project is nil")
	}
	if clipName == "" {
		return "", fmt.Errorf("clipName must not be empty")
	}
	if start > end {
		start, end = end, start
	}
	if start == end {
		return "", fmt.Errorf("empty range: start == end (%d)", start)
	}
	if start < 0 {
		start = 0
	}
	frames := rec.Frames()
	if end >= len(frames) {
		end = len(frames) - 1
	}
	if end < start {
		return "", fmt.Errorf("no frames in range [%d, %d]", start, end)
	}

	// Find the sprite. Return a clear name-mismatch error otherwise.
	spriteIdx := -1
	for i, s := range project.Sprites {
		if s.Name == spriteName {
			spriteIdx = i
			break
		}
	}
	if spriteIdx < 0 {
		return "", fmt.Errorf("sprite %q not found in project", spriteName)
	}
	sprite := &project.Sprites[spriteIdx]
	if sprite.FrameW <= 0 || sprite.FrameH <= 0 {
		return "", ErrSpriteFrameSize
	}

	// Build the strip image. Each captured frame is centred and
	// cropped to FrameW × FrameH; the strip is FrameW * N wide and
	// FrameH tall.
	n := end - start + 1
	stripW := sprite.FrameW * n
	stripH := sprite.FrameH

	palette := make(color.Palette, pixelforge.MaxColors)
	srcPalette := frames[start].Palette
	srcMapping := frames[start].PaletteMapping
	for i := range palette {
		mapped := srcMapping[i] & (pixelforge.MaxColors - 1)
		rgb := srcPalette[mapped]
		r, g, b := rgb.RGB()
		palette[i] = color.NRGBA{R: r, G: g, B: b, A: 255}
	}

	strip := image.NewPaletted(image.Rect(0, 0, stripW, stripH), palette)
	for i := 0; i < n; i++ {
		f := frames[start+i]
		writeFrameAt(strip, f, i*sprite.FrameW, 0, sprite.FrameW, sprite.FrameH)
	}

	// Write the strip PNG.
	assetsDir := pixelforge_project.AssetsDir(projectPath)
	clipDir := filepath.Join(assetsDir, sprite.Name)
	if err := os.MkdirAll(clipDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir cliplet dir: %w", err)
	}
	absPath := filepath.Join(clipDir, clipName+".png")
	f, err := os.Create(absPath)
	if err != nil {
		return "", fmt.Errorf("create cliplet PNG: %w", err)
	}
	defer f.Close()
	if err := png.Encode(f, strip); err != nil {
		return "", fmt.Errorf("encode cliplet PNG: %w", err)
	}

	// Build the AnimationClip and append or replace by name.
	indices := make([]int, n)
	durations := make([]int, n)
	for i := 0; i < n; i++ {
		indices[i] = i
		durations[i] = 1
	}
	relPath := filepath.ToSlash(filepath.Join(sprite.Name, clipName+".png"))
	clip := pixelforge_project.AnimationClip{
		Name:      clipName,
		Frames:    indices,
		FPS:       float64(pixelforge.TPS()),
		LoopMode:  "loop",
		ClipPath:  relPath,
		Durations: durations,
	}
	replaced := false
	for i := range sprite.Animations {
		if sprite.Animations[i].Name == clipName {
			sprite.Animations[i] = clip
			replaced = true
			break
		}
	}
	if !replaced {
		sprite.Animations = append(sprite.Animations, clip)
	}
	return absPath, nil
}

// writeFrameAt copies a centered FrameW × FrameH region of frame.Canvas
// into the strip at (x, y). Frames smaller than the requested crop are
// padded with index 0 (treat-as-transparent slot).
func writeFrameAt(dst *image.Paletted, frame *Frame, x, y, frameW, frameH int) {
	srcW := frame.Canvas.W()
	srcH := frame.Canvas.H()
	// Source crop top-left.
	srcX0 := (srcW - frameW) / 2
	srcY0 := (srcH - frameH) / 2
	if srcX0 < 0 {
		srcX0 = 0
	}
	if srcY0 < 0 {
		srcY0 = 0
	}
	data := frame.Canvas.Data()
	for fy := 0; fy < frameH; fy++ {
		for fx := 0; fx < frameW; fx++ {
			sx := srcX0 + fx
			sy := srcY0 + fy
			var c pixelforge.Color
			if sx >= 0 && sx < srcW && sy >= 0 && sy < srcH {
				c = data[sy*srcW+sx]
			}
			dst.Pix[(y+fy)*dst.Stride+(x+fx)] = uint8(c)
		}
	}
}
