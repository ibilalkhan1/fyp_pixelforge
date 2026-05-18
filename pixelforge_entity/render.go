package pixelforge_entity

import (
	"log"
	"sort"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// Tile dimensions in scene-space pixels. The v1 ScreenRoom plan fixes
// the world-primitive convention at 8x8; per-layer TilemapLayer.TileW /
// TileH exists in the schema but entity placement always uses the
// canonical 8x8 cells (per the plan's "Tile size 8x8 base, fixed in v1
// by editor convention" decision). When a future genre needs a
// different cell size this constant pair grows into an argument; for
// now hard-coding keeps the renderer dependency-free.
const (
	TileW = 8
	TileH = 8
)

// SpriteComponentType is the EntityComponent.Type string the renderer
// looks for. Matches the value the editor's PlaceEntity flow writes in
// pixelforge_studio/editor/canvas.go.
const SpriteComponentType = "Sprite"

// SpriteValueKey is the key inside an EntityComponent's Values map
// that names the SpriteAsset to render. Matches the writer in
// canvas.PlaceEntity ("sprite"). Documented here so future component
// authors (or a migration to a typed component registry via
// pfcomponent) keep the same convention.
const SpriteValueKey = "sprite"

// DrawSpriteFn is the indirection point for pixelforge.DrawSprite.
// Production calls flow through the engine's real DrawSprite (which
// also subtracts pixelforge.Camera); tests swap this variable for a
// recorder to capture (sprite, dx, dy) tuples without needing a live
// Ebitengine context. The variable is process-global like other
// pixelforge_* state; the package documents the single-goroutine
// contract.
var DrawSpriteFn = pixelforge.DrawSprite

// RenderAll iterates entities in ascending Z order (EntityPosition.Z)
// and draws each one's bound sprite at (TileX*TileW, TileY*TileH) in
// scene-space. The input slice is never mutated; a sorted index
// vector drives iteration so the caller's ordering is preserved.
//
// Entities without a Sprite component are skipped silently (some
// entities are logic-only). Entities whose Sprite component names a
// SpriteAsset that is absent from the supplied sprites slice log a
// warning via the standard log package and are skipped; rendering
// continues for the remaining entities.
//
// The camera offset is applied automatically by DrawSpriteFn (the
// real pixelforge.DrawSprite subtracts pixelforge.Camera internally),
// so RenderAll always emits raw scene-space destinations.
func RenderAll(entities []pixelforge_project.Entity, sprites []pixelforge_project.SpriteAsset) {
	if len(entities) == 0 {
		return
	}

	// Build a sorted-index vector so the caller's slice stays intact.
	// sort.SliceStable preserves the original order for ties in Z, so
	// two entities at the same Z draw in their authored sequence.
	order := make([]int, len(entities))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		return entities[order[i]].Position.Z < entities[order[j]].Position.Z
	})

	// Sprite-name lookup. The catalog is typically small (handfuls of
	// sprites per project), so a linear scan inside the loop is cheap
	// but a one-shot map removes the n*m worst case for entity-heavy
	// scenes without changing observable behaviour.
	spriteIdx := make(map[string]int, len(sprites))
	for i := range sprites {
		spriteIdx[sprites[i].Name] = i
	}

	for _, i := range order {
		ent := &entities[i]
		spriteName, ok := spriteComponentName(ent)
		if !ok {
			// No Sprite component — logic-only entity, skip silently.
			continue
		}
		assetIdx, found := spriteIdx[spriteName]
		if !found {
			log.Printf("pixelforge_entity: entity %q references unknown sprite %q; skipping", ent.ID, spriteName)
			continue
		}
		asset := &sprites[assetIdx]

		// v1 entity sprites use the asset's full canvas as the
		// destination region. Anim-frame slicing is a separate
		// concern (handled by pfcomponent / future work); the
		// renderer's job here is just to place the entity at its
		// tile-cell. Build a Sprite that names the source rect using
		// the asset's frame dimensions when present so single-frame
		// sheets and multi-frame sheets both Just Work.
		s := buildSprite(asset)

		dx := ent.TileX * TileW
		dy := ent.TileY * TileH
		DrawSpriteFn(s, dx, dy)
	}
}

// spriteComponentName extracts the sprite asset name from an entity's
// first "Sprite" component. Returns ("", false) when no Sprite
// component is present or when the component's Values map is missing
// the "sprite" key (or stores a non-string value, which would be a
// schema violation but we treat it defensively as "no sprite" to keep
// the renderer crash-free).
func spriteComponentName(ent *pixelforge_project.Entity) (string, bool) {
	for i := range ent.Components {
		comp := &ent.Components[i]
		if comp.Type != SpriteComponentType {
			continue
		}
		raw, exists := comp.Values[SpriteValueKey]
		if !exists {
			return "", false
		}
		name, ok := raw.(string)
		if !ok || name == "" {
			return "", false
		}
		return name, true
	}
	return "", false
}

// buildSprite produces a pixelforge.Sprite from a SpriteAsset. The
// asset stores its dimensions but not a Source canvas; the runtime is
// responsible for resolving the actual pixel data through whatever
// asset pipeline owns it. For the v1 renderer we construct a Sprite
// whose Source is a zero-valued Canvas; the real DrawSprite tolerates
// this only when the test recorder is in place, so production
// integrations (the editor preview + idea #7's Capsule) will wrap
// this call with their own asset resolution. The renderer's contract
// is "I will call DrawSprite with the right destination coordinates
// for each entity" — sprite data plumbing belongs to the caller.
//
// FrameW / FrameH are preferred over Width / Height when non-zero so
// multi-frame sheets blit a single frame rather than the entire strip.
func buildSprite(asset *pixelforge_project.SpriteAsset) pixelforge.Sprite {
	w := asset.FrameW
	h := asset.FrameH
	if w == 0 {
		w = asset.Width
	}
	if h == 0 {
		h = asset.Height
	}
	return pixelforge.Sprite{
		Area: pixelforge.IntArea{X: 0, Y: 0, W: w, H: h},
	}
}
