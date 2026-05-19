package buildpipeline

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/png"
	"sort"
	"strings"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// IconResult bundles the per-platform artifacts the icon pipeline
// produces. Format-specific bytes (Windows .ico/.syso, macOS .icns)
// are nil-able — when the external icon-format libs aren't wired
// in this build, the platform-specific entries stay nil and the
// builder writes a no-icon binary (still functional, just without
// a custom icon resource).
type IconResult struct {
	// Sprite is the resolved sprite the icon was rendered from
	// (or nil when fallback ran and no project sprite was
	// suitable). Useful for the UI to display "icon: hero" next
	// to the build.
	Sprite *pixelforge_project.SpriteAsset

	// FaviconBase64 is the 32x32 PNG, base64-encoded, suitable for
	// the WASM bundler's <link rel="icon"> data URL. Always
	// populated (uses the fallback icon if no sprite resolved).
	FaviconBase64 string

	// Note carries an informational message for the build status
	// pill — e.g. "auto-picked: hero" or "fallback icon used".
	Note string
}

// ResolveIconSprite returns the SpriteAsset the icon pipeline
// should render. Priority:
//
//  1. p.IconSpriteName (designer-marked) — look up by name; if
//     found, return.
//  2. Auto-pick by reference count across scenes' entities, items,
//     and bindings. Tie-break by sprite Name (alphabetical).
//     Prefer 16x16 sprites when reference counts tie.
//  3. Return nil (caller falls back to the embedded default).
func ResolveIconSprite(p *pixelforge_project.Project) *pixelforge_project.SpriteAsset {
	if p == nil || len(p.Sprites) == 0 {
		return nil
	}
	// Designer-marked path.
	if p.IconSpriteName != "" {
		for i := range p.Sprites {
			if p.Sprites[i].Name == p.IconSpriteName {
				return &p.Sprites[i]
			}
		}
	}
	// Auto-pick by reference count.
	counts := referenceCounts(p)

	type scored struct {
		sprite *pixelforge_project.SpriteAsset
		count  int
		is16   bool
	}
	var ranked []scored
	for i := range p.Sprites {
		s := &p.Sprites[i]
		ranked = append(ranked, scored{
			sprite: s,
			count:  counts[s.Name],
			is16:   s.FrameW == 16 && s.FrameH == 16,
		})
	}
	sort.Slice(ranked, func(a, b int) bool {
		if ranked[a].count != ranked[b].count {
			return ranked[a].count > ranked[b].count
		}
		if ranked[a].is16 != ranked[b].is16 {
			return ranked[a].is16
		}
		return ranked[a].sprite.Name < ranked[b].sprite.Name
	})
	if len(ranked) == 0 {
		return nil
	}
	return ranked[0].sprite
}

// referenceCounts walks the project's scenes / items / bindings /
// dialogues looking for sprite-name references. Each occurrence
// increments the per-sprite count.
func referenceCounts(p *pixelforge_project.Project) map[string]int {
	counts := map[string]int{}
	if p == nil {
		return counts
	}
	for _, scene := range p.Scenes {
		for _, ent := range scene.Entities {
			for _, comp := range ent.Components {
				for _, v := range comp.Values {
					if s, ok := v.(string); ok {
						counts[s]++
					}
				}
			}
		}
	}
	for _, it := range p.Items {
		if it.Icon != "" {
			counts[it.Icon]++
		}
	}
	return counts
}

// GenerateFavicon returns the rasterised logo as base64 PNG —
// every shipped game (Host + WASM) wears the same Pixelforge
// brand mark. Plan-008 U5 replaces the per-sprite hash-colour
// fallback with this real raster; the legacy per-sprite helper
// lives on as generateLegacyFavicon for callers transitioning off
// the old signature.
func GenerateFavicon() (string, error) {
	return GenerateLogoFavicon()
}

// generateLegacyFavicon is the pre-U5 per-sprite favicon path.
// Kept private one release so callers that still pass a sprite
// don't fail to compile while their migration to the no-arg
// GenerateFavicon lands.
func generateLegacyFavicon(sprite *pixelforge_project.SpriteAsset) (string, error) {
	const size = 32
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	c := iconColorFor(sprite)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, c)
		}
	}
	// Border so the favicon isn't a flat coloured square.
	border := color.RGBA{R: 0, G: 0, B: 0, A: 255}
	for i := 0; i < size; i++ {
		img.Set(i, 0, border)
		img.Set(i, size-1, border)
		img.Set(0, i, border)
		img.Set(size-1, i, border)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// iconColorFor derives a stable colour from the sprite's name so
// each project gets a recognisable favicon even without real
// sprite bytes wired in. Empty / nil sprite returns Pixelforge
// orange.
func iconColorFor(sprite *pixelforge_project.SpriteAsset) color.RGBA {
	if sprite == nil || sprite.Name == "" {
		return color.RGBA{R: 0xff, G: 0x88, B: 0x33, A: 0xff}
	}
	h := uint32(0)
	for _, r := range sprite.Name {
		h = h*31 + uint32(r)
	}
	return color.RGBA{
		R: uint8((h >> 16) & 0xff),
		G: uint8((h >> 8) & 0xff),
		B: uint8(h & 0xff),
		A: 0xff,
	}
}

// GenerateIconResult composes the IconResult for the supplied
// project. Plan-008 U5 collapses the per-project favicon to the
// brand logo so every shipped game wears the same icon; Sprite
// is still resolved for designer visibility ("auto-picked icon:
// hero" survives as a Note) but never drives the rendered bytes.
//
// Platform-specific bytes (.ico, .icns, .syso) are produced by
// BuildLogoICO / BuildLogoICNS / BuildWindowsSyso in icon_logo.go
// and consumed by the per-target builders directly; this function
// only produces the WASM favicon string.
func GenerateIconResult(p *pixelforge_project.Project) (*IconResult, error) {
	sprite := ResolveIconSprite(p)
	favicon, err := GenerateFavicon()
	if err != nil {
		return nil, err
	}
	return &IconResult{
		Sprite:        sprite,
		FaviconBase64: favicon,
		Note:          "logo.svg",
	}, nil
}

// iconNote composes the status-pill text the Build workspace
// renders alongside the build status: "icon: hero" / "auto-picked
// icon: hero" / "fallback icon (no project sprites)".
func iconNote(p *pixelforge_project.Project, sprite *pixelforge_project.SpriteAsset) string {
	if sprite == nil {
		return "fallback icon (no project sprites)"
	}
	if p != nil && p.IconSpriteName == sprite.Name {
		return "icon: " + sprite.Name
	}
	return "auto-picked icon: " + sprite.Name
}

// errIconUnsupported is the legacy sentinel callers errors.Is'd
// against while per-platform icon generation was stubbed. U5 ships
// real bytes from oksvg/rasterx; the sentinel + IsIconUnsupported
// stay one release for source-compat with downstream callers.
//
// Deprecated: U5 wires real .ico/.icns generation. New callers
// should branch on the concrete error, not this sentinel.
var errIconUnsupported = errors.New("buildpipeline: per-platform icon generation pending external libs (goversioninfo, icns)")

// GenerateWindowsIcoStub returns the rasterised logo as Windows
// .ico bytes. The sprite arg is ignored — every shipped game gets
// the brand logo per U5; the signature is preserved to keep
// downstream callers compiling.
func GenerateWindowsIcoStub(sprite *pixelforge_project.SpriteAsset) ([]byte, error) {
	return BuildLogoICO()
}

// GenerateMacIcnsStub returns the rasterised logo as macOS .icns
// bytes. Same brand-logo contract as GenerateWindowsIcoStub.
func GenerateMacIcnsStub(sprite *pixelforge_project.SpriteAsset) ([]byte, error) {
	return BuildLogoICNS()
}

// IsIconUnsupported reports whether the supplied error is the
// legacy per-platform icon stub's sentinel. After U5 the real
// paths never return it; legacy callers continue to receive false
// from successful generation.
//
// Deprecated: U5 wires real icon generation; this sentinel
// survives one release for source compat.
func IsIconUnsupported(err error) bool {
	return errors.Is(err, errIconUnsupported)
}

// sanitizeForFilename produces a filesystem-safe variant of the
// game name (used by the icon-resource Filename field). Mirrors
// pixelforge_save.Sanitize's discipline so save dirs + icon
// resource names share the same identifier space.
func sanitizeForFilename(s string) string {
	if s == "" {
		return "untitled"
	}
	var sb strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z':
			sb.WriteRune(r)
		case r >= '0' && r <= '9':
			sb.WriteRune(r)
		case r == '-' || r == '_':
			sb.WriteRune(r)
		case r == ' ':
			sb.WriteRune('_')
		}
	}
	out := sb.String()
	if out == "" {
		return "untitled"
	}
	return out
}
