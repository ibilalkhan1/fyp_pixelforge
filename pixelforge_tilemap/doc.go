// Package pixelforge_tilemap renders a pixelforge_project.TilemapLayer to
// the active pixelforge draw target as a grid of 8x8 sprite slices read
// from a horizontal sprite-sheet strip.
//
// Role in the engine: the editor's scene-preview Draw shim AND idea #7's
// shipped-runtime Capsule both call Render every frame to blit the level
// geometry behind the entity sprites. Both paths share this single
// renderer by construction so "what the designer sees in preview" equals
// "what the shipped binary draws" without divergence.
//
// Convention: the bound sheet is a horizontal strip of TileW x TileH
// tiles. Cell ID 0 means empty (skipped), ID N (1..) sources tile at
// sheet X = (N-1) * TileW. Out-of-range IDs (past the sheet's right
// edge) are skipped silently — the renderer never panics on bad data
// per docs/solutions/editor-pforge-schema-shape.md's "malformed entries
// log + fall through to defaults, never fatal" discipline.
//
// Camera handling: Render passes scene-space destination coordinates
// to pixelforge.DrawSprite, which subtracts pixelforge.Camera internally
// (see pixelforge.Stretch). The renderer never reads or writes the
// Camera global itself; pixelforge_camera owns those writes.
//
// Test seam: the engine's DrawSprite writes through several package
// globals (drawTarget, Camera, clip, color tables) that make it
// awkward to assert call shapes from a unit test. To keep tests
// hermetic the package depends on the unexported Drawer interface and
// exposes RenderWith for injecting a recorder. The exported Render is
// a thin wrapper that binds the engine-backed drawer. Production code
// should always call Render; tests should always call RenderWith.
package pixelforge_tilemap
