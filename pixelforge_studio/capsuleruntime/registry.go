// Package capsuleruntime owns the capsule-side wiring that turns
// an embedded .pforge project + assets/ FS into populated engine
// registries. The codegen-emitted capsule.go calls Boot between
// project-load and pixelforge_ebiten.Run.
//
// Three concerns live here:
//
//   - Loaders that decode sprites / audio / dialogue from the
//     embed.FS the capsule template carries.
//   - Capsule-side registries for sprites, scenes, and items —
//     state that the engine packages don't own but the subscriber
//     layer needs (entity render lookups, scene/change handlers,
//     inventory give/take handlers).
//   - The Boot/Runtime seam that wires the verb-recipe event bus
//     subscribers (installed by InstallSubscribers in subscribers.go,
//     U2) to the populated registries.
//
// The split keeps engine packages (pixelforge_audio, pixelforge_
// dialogue, pixelforge_menus) unaware of the verb-recipe topic
// surface — they expose Register/Lookup APIs, and the capsule wires
// subscribers that read payloads and call into those APIs. Same
// cycle-break shape as the PNG/WAV import-runner injection pattern
// from plan-005.
package capsuleruntime

import (
	"sort"
	"sync"

	"github.com/ibilalkhan1/fyp_pixelforge"
	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_project"
)

// SpriteEntry pairs a decoded canvas with its source metadata so
// callers (entity renderer, library workspace) can both blit pixels
// and reason about frame layout without a second lookup.
type SpriteEntry struct {
	Asset  pixelforge_project.SpriteAsset
	Canvas pixelforge.Canvas
}

// spriteRegistry, sceneRegistry, itemRegistry are the three capsule-
// side registries Boot populates. Sprite data lives here (not in
// engine packages) because the canvas-decoding step needs the
// capsule's embed.FS and the asset metadata; engine code is asset-
// agnostic by design.
//
// Scenes and items live here because the subscriber layer (U2) needs
// name→object lookup for scene/change and inventory give/take
// handlers, and shoving these into pixelforge_project would invert
// the dependency direction (project is the schema; capsuleruntime is
// the loader that materialises it).
var (
	spriteRegistryMu sync.RWMutex
	spriteRegistry   = map[string]SpriteEntry{}

	sceneRegistryMu sync.RWMutex
	sceneRegistry   = map[string]pixelforge_project.Scene{}

	itemRegistryMu sync.RWMutex
	itemRegistry   = map[string]pixelforge_project.ItemDefinition{}
)

// RegisterSprite stores entry under name. Empty names are ignored;
// re-registration replaces. Idempotency mirrors the audio/dialogue
// registries — Boot may run twice across studio-preview reloads.
func RegisterSprite(name string, entry SpriteEntry) {
	if name == "" {
		return
	}
	spriteRegistryMu.Lock()
	defer spriteRegistryMu.Unlock()
	spriteRegistry[name] = entry
}

// LookupSprite returns the registered sprite entry plus ok.
func LookupSprite(name string) (SpriteEntry, bool) {
	spriteRegistryMu.RLock()
	defer spriteRegistryMu.RUnlock()
	e, ok := spriteRegistry[name]
	return e, ok
}

// AllSpriteNames returns every registered sprite name sorted.
func AllSpriteNames() []string {
	spriteRegistryMu.RLock()
	defer spriteRegistryMu.RUnlock()
	out := make([]string, 0, len(spriteRegistry))
	for name := range spriteRegistry {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// RegisterScene stores scene under its ID. Empty IDs are ignored.
func RegisterScene(scene pixelforge_project.Scene) {
	if scene.ID == "" {
		return
	}
	sceneRegistryMu.Lock()
	defer sceneRegistryMu.Unlock()
	sceneRegistry[scene.ID] = scene
}

// LookupScene returns the registered scene plus ok.
func LookupScene(id string) (pixelforge_project.Scene, bool) {
	sceneRegistryMu.RLock()
	defer sceneRegistryMu.RUnlock()
	s, ok := sceneRegistry[id]
	return s, ok
}

// AllSceneIDs returns every registered scene ID sorted.
func AllSceneIDs() []string {
	sceneRegistryMu.RLock()
	defer sceneRegistryMu.RUnlock()
	out := make([]string, 0, len(sceneRegistry))
	for id := range sceneRegistry {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// RegisterItem stores item under its ID. Empty IDs are ignored.
func RegisterItem(item pixelforge_project.ItemDefinition) {
	if item.ID == "" {
		return
	}
	itemRegistryMu.Lock()
	defer itemRegistryMu.Unlock()
	itemRegistry[item.ID] = item
}

// LookupItem returns the registered item plus ok.
func LookupItem(id string) (pixelforge_project.ItemDefinition, bool) {
	itemRegistryMu.RLock()
	defer itemRegistryMu.RUnlock()
	it, ok := itemRegistry[id]
	return it, ok
}

// AllItemIDs returns every registered item ID sorted.
func AllItemIDs() []string {
	itemRegistryMu.RLock()
	defer itemRegistryMu.RUnlock()
	out := make([]string, 0, len(itemRegistry))
	for id := range itemRegistry {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// ResetRegistriesForTest clears all three capsule-side registries
// and the audio + dialogue registries the loaders populate. Test-
// only — callers exercising Boot in isolation use this to keep
// per-test state local.
func ResetRegistriesForTest() {
	spriteRegistryMu.Lock()
	spriteRegistry = map[string]SpriteEntry{}
	spriteRegistryMu.Unlock()

	sceneRegistryMu.Lock()
	sceneRegistry = map[string]pixelforge_project.Scene{}
	sceneRegistryMu.Unlock()

	itemRegistryMu.Lock()
	itemRegistry = map[string]pixelforge_project.ItemDefinition{}
	itemRegistryMu.Unlock()
}
