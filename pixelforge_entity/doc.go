// Package pixelforge_entity provides a small renderer that walks a
// scene's entity slice and blits each entity's sprite at its tile-cell
// position. It is the entity-side counterpart to pixelforge_tilemap:
// both packages iterate Scene-level data and call pixelforge.DrawSprite,
// and both rely on the engine's existing pixelforge.Camera offset for
// scroll behaviour.
//
// # Coordinate convention
//
// Entities authored in the v1 ScreenRoom world primitive carry
// canonical tile-cell coordinates in Entity.TileX / Entity.TileY (see
// pixelforge_project). RenderAll multiplies those by the fixed
// TileW / TileH constants (8 px each — the Mario-strip convention from
// the plan) to obtain a scene-space pixel destination, then hands that
// destination to pixelforge.DrawSprite. DrawSprite subtracts
// pixelforge.Camera itself, so RenderAll never touches the camera
// directly; it only emits scene-space coordinates.
//
// # Sprite-component lookup
//
// Each entity may carry zero or more EntityComponents. The renderer
// looks for the first component whose Type == "Sprite" and reads the
// sprite name from Values["sprite"] (the key the editor's PlaceEntity
// flow writes — see pixelforge_studio/editor/canvas.go). Entities
// without a Sprite component are skipped silently; entities that name
// a SpriteAsset not present in the supplied catalog log a warning via
// the standard log package and are also skipped.
//
// # Z-ordering
//
// Entities are drawn in ascending Z order (EntityPosition.Z). The
// lowest Z renders first (behind), the highest last (in front). The
// input slice is never mutated; RenderAll iterates a sorted index
// vector instead. Ties in Z preserve the original slice order so the
// result is stable and reproducible.
//
// # Testability
//
// All DrawSprite calls flow through the package-level DrawSpriteFn
// variable, which defaults to pixelforge.DrawSprite. Tests swap it
// for a recorder to verify call arguments without needing a live
// Ebitengine render context. Production code never touches the
// variable.
//
// # Thread-safety
//
// Like the rest of the pixelforge_* packages this package is not
// thread-safe; RenderAll must be called from the same goroutine that
// drives the engine game loop.
package pixelforge_entity
