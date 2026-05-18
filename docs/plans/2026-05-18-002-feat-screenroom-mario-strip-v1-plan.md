---
title: "feat: ScreenRoom Mario-strip primitive — world authoring + tile painter + camera follow (idea #1 v1)"
type: feat
status: active
date: 2026-05-18
depth: deep
origin: docs/brainstorms/2026-05-18-screenroom-mario-strip-v1-requirements.md
ideation: docs/ideation/2026-05-18-pixelforge-nes-class-no-code-ideation.md (idea #1)
related_plans:
  - docs/plans/2026-05-18-001-feat-entity-verb-sheet-v1-plan.md (idea #5 — Player Archetype is the Player tag)
---

# feat: ScreenRoom Mario-strip primitive (idea #1 v1)

## Summary

v1 ships the editor + engine substrate for authoring a Mario-class side-scrolling level: a bounded 2D tile grid per scene (default 16 screens × 1 tall, 8×8 tiles at 256×240 per screen), a tile painter with three tools (brush, paint-bucket, rectangle-fill) wired onto the existing-but-stub `Paint` tool, tile-cell entity placement with right-click "Set as Player" (mutates `Entity.Archetype = "Player"` from idea #5) and "Set spawn here" (Scene-level property), a new engine-side `pixelforge_camera` package that drives Mario-style smooth-follow with dead-zone + look-ahead, and engine renderers (`pixelforge_tilemap`, entity sprite iterator) that the editor's existing scene-as-texture preview already consumes. Schema reserves `World{Mode, Rooms[]}`, `Zones[]`, `Warps[]`, `CameraMode` as `omitempty` for future genres. Zero external dependencies — the only candidate (`lafriks/go-tiled`) is irrelevant in v1 because Tiled import is out of scope. The "ship to a classmate" loop depends on idea #7's plan landing the Capsule + Build pipeline; v1 of idea #1 ships the editor preview + engine renderers and validates the model there.

---

## Leverage Doctrine (applied)

Per `docs/plans/2026-05-18-001-feat-entity-verb-sheet-v1-plan.md`'s Leverage Doctrine appendix: evaluate before depending; port a pattern instead of wrapping a library when wrapping exceeds the from-scratch cost; build native when in-repo primitives already cover 80%.

**Candidates evaluated for idea #1:**

| Candidate | Status | Verdict |
|---|---|---|
| `lafriks/go-tiled` (Tiled TMX/JSON reader) | Active, MIT | **Skip** — v1 doesn't import Tiled files (deferred per origin); native authoring only |
| Ebitengine camera/scrolling libraries | None mature; `sedyh/awesome-ebitengine` lists `willow` but it's a general display tree | **Build native** — Mark Brown / GMTK pattern, ~80 LOC custom (per Phase 1 external research) |
| Tile-painter libraries (Go) | None | **Build native on ImGui** — standard 4-tool pattern (brush/bucket/rect/optional-line) is well-documented |
| Undo/redo libraries | None worth depending on for per-stroke tile painting | **Build native** — command-pattern stack with cell-diff payloads, ~60 LOC |

Total custom: ~80 LOC camera follower + ~150 LOC tile painter UI + ~80 LOC tilemap renderer + ~50 LOC entity renderer + ~60 LOC undo + ~120 LOC scene resize + supporting tests. Well below the wrapping cost of any candidate.

**Reference reading (no imports):** Mark Brown's "How to Make a Good 2D Camera" + Itay Keren's "Scroll Back" GDC talk for camera; Tiled / LDtk docs for painter UX.

---

## Problem Frame

Pixelforge's engine has a working palette + sprite drawing pipeline, and the editor has — post-ImGui migration — a scene-as-texture live preview, a Scene workspace with a tool palette (`Select`/`Place`/`Delete`/`Paint`), and a project schema with `Scene.Tilemaps []TilemapLayer` and `Scene.Entities []Entity`. What it lacks:

- **No world primitive.** `Scene.Tilemaps` is a flat grid per scene with no relationship to anything else. There's no concept of "level width in screens," no spawn point at the scene level, no camera config. Grep for `camera|scroll|parallax|viewport` in `pixelforge_project/` and `pixelforge_studio/` returns only mouse-wheel asset-browser scrolling and ImGui dockspace viewport — no game camera anywhere.
- **No tilemap renderer.** `TilemapLayer.Grid [][]int` is data with no consumer. The editor's preview shows entity markers as 12-px squares; tiles are invisible. The shipped game's codegen template (`pixelforge_studio/codegen/templates.go`) emits only `SetScreenSize + SetTPS + pixelforge_ebiten.Run()` — zero rendering.
- **No camera follow.** The engine has a global `pixelforge.Camera Position{X, Y}` (already applied in `setPixelWithColor` and sprite blit). But nothing writes to it; nothing computes "where should the camera be given the player's position + level bounds + dead-zone."
- **The `Paint` tool is a stub.** Defined in `pixelforge_studio/editor/tools.go:11` but `canvas.go`'s switch only handles `Place` / `Delete` / `Select`. The toolbar shows the radio button; clicking it does nothing.
- **Entity placement is pixel-space float64.** `EntityPosition.X/Y` are floats; `PlaceEntity` writes raw `windowToScene` pixel coords. No tile-cell snapping. Drag-clamp at `canvas.go:200-201` caps movement at `ScreenWidth × ScreenHeight` — a single-screen assumption that breaks for Mario-class wide levels.
- **No right-click handling.** "Set as Player" + "Set spawn here" — both need new right-click dispatch through `canvas.UpdateAt`.

The brainstorm's bet: a designer in your community paints a 16-screens-wide Mario-class level using brush + bucket + rectangle tools, drops a Player + a handful of enemies + pickups on tile cells, right-clicks one cell to set spawn, watches the camera smooth-scroll across the level in the preview as Mario walks right, and (when idea #7's Capsule + Build pipeline ships) hands a single-file binary to a classmate who plays the identical experience.

This plan ships the editor + engine substrate for that bet. The "ship a binary" half of Success Criteria belongs to idea #7's plan, which the brainstorm explicitly acknowledged. v1 of idea #1 proves the model works end-to-end in the editor preview; idea #7 carries it to a shipped artifact.

---

## Carried Forward from Origin

All 13 requirements (R1-R13), all 5 acceptance examples (AE1-AE5), and all 3 actors (A1-A3) from `docs/brainstorms/2026-05-18-screenroom-mario-strip-v1-requirements.md` are in scope.

| Origin | Scope summary | Plan unit(s) |
|---|---|---|
| R1, R2, R3, R4 | Single bounded grid per scene; 256×240 screen = 32×30 tiles of 8×8; resize-preserves-content; integer cell coords | U1, U8 |
| R5, R6 | Three tools (brush, bucket, rect); explicit tile selection (no auto-rules in v1; idea #2 deferred) | U6 |
| R7, R8, R9 | Tile-cell entity placement; Player marking (idea #5 dep); spawn cell as scene property | U7, U4 (spawn scene-level field) |
| R10, R11 | Smooth-follow camera with dead-zone + look-ahead; hard-stop at level bounds | U2, U3 (camera + engine integration), U5 |
| R12 | Build to current-platform native binary (idea #7's plan owns this) | Dependency only — not built here |
| R13 | Schema reservations (`World{Mode, Rooms[]}`, `Zones[]`, `Warps[]`, `CameraMode`) as `omitempty` | U1 |
| AE1–AE5 | All five acceptance examples covered as integration tests | U9 |
| A1–A3 | Designer, Studio, Shipped game — all referenced in unit flows | All units |

Origin's "Deferred to Planning" section: 4 technical questions resolved in Phase 2 (see Key Technical Decisions). No blocking product questions.

---

## High-Level Technical Design

How the pieces fit together in the editor-preview AND shipped-runtime paths (note that idea #7's Capsule provides the shipped-runtime wiring):

```
                 SCHEMA (per-Scene, .pforge)
                 ═════════════════════════════════════
   ┌──────────────────────────────────────────────┐
   │ Scene                                        │
   │ ─ GridWidthScreens   int (default 16)        │  U1
   │ ─ GridHeightScreens  int (default 1)         │
   │ ─ SpawnTile          {Col, Row}              │
   │ ─ DefaultTileID      int                     │
   │ ─ Camera             {DZW, DZH, LA, SmoothX} │
   │ ─ Tilemaps  []TilemapLayer                   │
   │ ─ Entities  []Entity (Archetype, TileX/Y)    │
   │                                              │
   │ TilemapLayer                                 │
   │ ─ SpriteSheetRef string (NEW — sheet binding)│
   │ ─ TileW, TileH   int    (existing)           │
   │ ─ Grid           [][]int (existing)          │
   └──────────────┬───────────────────────────────┘
                  │ loaded by both paths
       ┌──────────┴────────────┐
       ▼                       ▼
 ┌──────────────────┐   ┌──────────────────┐
 │ EDITOR (live     │   │ SHIPPED RUNTIME  │
 │ preview)         │   │ (via idea #7's   │
 │                  │   │ Capsule)         │
 │ sceneGame.Update │   │                  │
 │  └─ camera.Update│   │ capsule.Run loop │
 │  └─ tilemap.Draw │   │  └─ camera.Update│
 │  └─ entity.Draw  │   │  └─ tilemap.Draw │
 └──────────────────┘   │  └─ entity.Draw  │
                        └──────────────────┘
                  │                       │
                  └───────┬───────────────┘
                          │ all writes flow through
                          ▼
                ┌──────────────────────────────┐
                │ ENGINE PRIMITIVES (new in    │
                │ this plan)                   │
                │                              │
                │ pixelforge_camera (U3)       │
                │  └─ Follower struct          │
                │  └─ Update(player, bounds)   │
                │     → writes pixelforge.     │
                │       Camera global          │
                │                              │
                │ pixelforge_tilemap (U2)      │
                │  └─ Render(layer, sheet)     │
                │     → iterates Grid +        │
                │       DrawSprite per cell    │
                │     → respects               │
                │       pixelforge.Camera      │
                │                              │
                │ entity render (U4) — small   │
                │  └─ helper in pixelforge or  │
                │    new pixelforge_entity     │
                │  └─ iterates Scene.Entities  │
                │  └─ DrawSprite at TileX/Y    │
                └──────────────────────────────┘
                          │
                          ▼
                ┌──────────────────────────────┐
                │ pixelforge.Camera Position   │
                │ (already exists; applied in  │
                │  setPixelWithColor + sprite  │
                │  blit per pixelforge.go:49,  │
                │  115, 157; sprite.go:43)     │
                └──────────────────────────────┘
```

*This illustrates the intended approach and is directional guidance for review, not implementation specification. The implementing agent should treat it as context, not code to reproduce.*

The structural insight: **single rendering pipeline for editor and shipped game.** The editor's sceneGame.Draw already renders through the engine's drawing functions; adding tilemap + entity rendering means the editor preview and the shipped binary observe identical output by construction. No "works in editor but not shipped" divergence.

---

## Output Structure

This plan adds three new engine packages, one scene-level config struct, and several files inside the existing `pixelforge_studio/editor/` package:

```
pixelforge_camera/                       (NEW package, U3)
├── camera.go                            — Follower struct, Update, defaults
├── camera_test.go
└── doc.go

pixelforge_tilemap/                      (NEW package, U2)
├── tilemap.go                           — Render function over Grid
├── tilemap_test.go
└── doc.go

pixelforge_entity/                       (NEW package, U4)
├── render.go                            — RenderAll(entities, sprites)
├── render_test.go
└── doc.go

pixelforge_project/
├── scenes.go                            (MODIFY, U1) — Scene grid/spawn/camera fields
└── world_reserved.go                    (NEW, U1) — World/Zones/Warps/CameraMode reserved

pixelforge_studio/editor/
├── canvas.go                            (MODIFY, U7) — tile-cell entity place; level-bounds drag
├── canvas_input.go                      (MODIFY, U5) — preview drives camera update
├── tile_painter.go                      (NEW, U6) — Paint tool brush/bucket/rect handlers
├── tile_painter_test.go
├── tile_palette.go                      (NEW, U6) — palette panel UI
├── undo_stroke.go                       (NEW, U6) — command stack with cell-diff
├── undo_stroke_test.go
├── scene_settings.go                    (NEW, U8) — resize spinners in inspector
├── scene_settings_test.go
├── right_click_menu.go                  (NEW, U7) — "Set as Player" / "Set spawn here"
└── workspaces.go                        (MODIFY, U6/U7) — extended toolbar + dispatch

pixelforge_studio/integration_test/      (existing from idea #5; appended)
├── mario_strip_e2e_test.go              (NEW, U9)
└── fixtures/mario_strip_scene.pforge    (NEW, U9)
```

This is a scope declaration. The implementer may consolidate or split files if implementation reveals a better layout — per-unit `**Files:**` sections remain authoritative.

---

## Implementation Units

Dependency-ordered. Each unit ships a complete, testable capability — "one-shot complete features per unit" per the user's directive.

### U1. Schema additions for world primitive

**Goal:** All Scene-level + per-layer schema fields land as additive `omitempty`. Pre-v1 `.pforge` files load cleanly with default values; round-trip preserves new fields. Future-genre fields are reserved but unused.

**Requirements:** R1, R3 (grid resize semantics), R9 (spawn-cell scene-level property), R13 (forward-compat schema reservation), AE6 (pre-v1 file loads cleanly).

**Dependencies:** none (foundational).

**Files:**
- `pixelforge_project/scenes.go` (MODIFY — extend `Scene` struct)
- `pixelforge_project/world_reserved.go` (NEW — `World{Mode, Rooms[]}`, `Zones[]`, `Warps[]`, `CameraMode` types reserved as zero-valued)
- `pixelforge_project/scenes_test.go` (NEW or extend existing)
- `pixelforge_project/project.go` (MODIFY — extend `applyDefaults` / `normalizeSlices` per project.go:107-159)

**Approach:**
- `Scene` gains: `GridWidthScreens int` (default 16), `GridHeightScreens int` (default 1), `SpawnTile struct{Col, Row int}` (default {0, 0}), `DefaultTileID int` (default 0), `Camera SceneCameraConfig{DeadZoneW, DeadZoneH, LookAheadX, SmoothX, SmoothY float64}` (defaults: 30% × 256, 40% × 240, 32, 0.20, 0.40).
- `TilemapLayer` gains: `SpriteSheetRef string` (omitempty — names a SpriteAsset in the project that the layer's `Grid` values index into).
- `world_reserved.go` declares `World`, `Zone`, `Warp` types with their fields, but `Scene` only holds them as `omitempty` slice/struct fields with no runtime consumption in v1.
- `Entity` already has `Archetype` from idea #5's plan U2; no new field on Entity in this unit.
- `applyDefaults` (project.go:107-113) extends to: populate Scene defaults if zero, validate SpawnTile is inside grid bounds (clamp if outside), default DefaultTileID to 0.
- Per `docs/solutions/editor-pforge-schema-shape.md`: malformed entries log + fall through to defaults, never fatal.

**Patterns to follow:** existing `Theme` field on Project (project.go:31-33) — the model for "additive optional struct with defaults"; `[]EventSubscription` slice convention (project.go:52-64); existing `TilemapLayer` struct (scenes.go:30-46) as the template for adding a single new string field.

**Test scenarios:**
- `TestScene_GridDefaults`: load a project JSON with no `grid_width_screens` key; after `applyDefaults`, value is 16. Same for height (1).
- `TestScene_SpawnTileDefaults`: pre-v1 scene → `SpawnTile == {0, 0}` post-load.
- `TestScene_CameraConfigDefaults`: pre-v1 scene → Camera config equals the documented defaults (30% / 40% / 32 / 0.20 / 0.40).
- `TestScene_ResizePreservesContent`: programmatically create a scene with painted cells at (5, 10) and (15, 25); change GridWidthScreens from 16 to 24; assert painted cells preserved at original coords; new area (cells 512+) returns 0 (DefaultTileID).
- `TestScene_ResizeClampsSpawn`: scene with SpawnTile={20, 0}; shrink GridWidthScreens to 8 (8 screens × 32 tiles = 256 tiles wide); spawn clamped to {255, 0}.
- `TestTilemapLayer_SpriteSheetRefOmitempty`: marshal layer without SpriteSheetRef → no `sprite_sheet_ref` key in JSON.
- `TestProject_PreV1FixtureRoundtrip`: load `pixelforge_studio/editor/cart_assets/editor.pforge`; assert no errors; re-save; verify no spurious new keys appear in the JSON diff.
- `TestProject_ReservedFieldsOmitemptyOnLoad`: load a project without `world`, `zones`, `warps`, `camera_mode` keys → fields are zero-valued; round-trip omits them.
- Covers AE6 (pre-v1 file loads cleanly with defaults).

**Verification:** `go test ./pixelforge_project/...` passes; round-trip of existing `cart_assets/editor.pforge` produces no key churn beyond intended new fields.

---

### U2. Engine package: pixelforge_tilemap (tilemap renderer)

**Goal:** A new engine-side package that walks a `TilemapLayer.Grid` and renders each cell as a sprite slice from the layer's bound `SpriteSheetRef`. Respects `pixelforge.Camera` for scrolling. Editor preview AND shipped game share this renderer.

**Requirements:** R1, R2 (256×240 tile rendering at viewport scale), R3 (rendering matches grid dimensions).

**Dependencies:** U1 (schema fields).

**Files:**
- `pixelforge_tilemap/tilemap.go` (NEW)
- `pixelforge_tilemap/tilemap_test.go` (NEW)
- `pixelforge_tilemap/doc.go` (NEW)

**Approach:**
- `Render(layer *pixelforge_project.TilemapLayer, sheet *pixelforge.Surface)` — the entry point. Iterates `Grid[row][col]`; for each non-zero cell, computes source rect from `cell ID → sheet position` (sheet is a horizontal strip of 8×8 tiles, indexed by ID); destination is `(col*TileW, row*TileH)` in scene-space; calls existing `pixelforge.DrawSprite` (which already respects `pixelforge.Camera`).
- Cell ID 0 = empty (no draw); cell ID 1..N = tile index in the sheet.
- Resolves `SpriteSheetRef` via the project's sprite list (passed in by caller — keeps the renderer decoupled from `pixelforge_project.Project`).
- Optional second mode: render only cells inside a viewport rect (culling). v1 ships unculled first; culling layered if perf demands. Cull benchmark is in tests.
- **No new external dependencies.** Iteration + existing `DrawSprite`.

**Patterns to follow:** existing `pixelforge.DrawSprite` signature + camera offset semantics (pixelforge.go:115, sprite.go:43-44) — the renderer is a thin loop over those calls. Hello-world examples in `pixelforge_examples/hello/` show the existing drawing pattern.

**Test scenarios:**
- `TestRender_EmptyGrid`: layer with all-zero grid → no DrawSprite calls (verify via mock or test surface).
- `TestRender_SingleTile`: 1×1 grid with cell ID=1 → one DrawSprite at (0, 0) sourcing tile 1 from the sheet.
- `TestRender_3x3Grid`: 3×3 grid with mixed IDs → exactly 9 cells iterated; non-zero cells drawn at (col*8, row*8) in scene-space.
- `TestRender_RespectsCameraOffset`: set `pixelforge.Camera = {X: 100, Y: 0}`; render at cell (15, 0) (scene-space 120, 0); assert resulting sprite-blit destination is screen-space 20 (= 120 - 100).
- `TestRender_BoundsClamp`: render a layer wider than the camera viewport; assert no out-of-bounds DrawSprite calls (or, if culling is implemented, no draws for cells fully off-screen).
- `TestRender_CellIDZeroSkipped`: grid with mixed 0 and 1 cells; only non-zero cells produce DrawSprite calls.
- `TestRender_PerformanceBudget`: render a 512×30 grid (16 screens × 30 tiles tall) in under 8 ms on the reference machine; flags if perf degrades and a culling pass is needed (this is the perf gate for the v1 default level size).

**Verification:** `go test ./pixelforge_tilemap/...` passes; benchmark confirms <8 ms for default level render.

---

### U3. Engine package: pixelforge_camera (smooth-follow with dead-zone + look-ahead)

**Goal:** A `Follower` struct that, given a player position + level bounds + config (dead-zone, look-ahead, smoothing), computes the desired camera offset each tick and writes to `pixelforge.Camera`. Mario-style: velocity-based look-ahead, dead-zone before camera moves, hard-clamp at level bounds.

**Requirements:** R10 (smooth follow with dead-zone + look-ahead), R11 (hard-stop at level bounds).

**Dependencies:** none (parallel with U1, U2).

**Files:**
- `pixelforge_camera/camera.go` (NEW)
- `pixelforge_camera/camera_test.go` (NEW)
- `pixelforge_camera/doc.go` (NEW)

**Approach:**
- `Follower{Config FollowerConfig; lastVelX, lastVelY float64; ...}` — stateful per-scene.
- `FollowerConfig{DeadZoneW, DeadZoneH float64; LookAheadX, LookAheadY float64; SmoothingX, SmoothingY float64; BoundsW, BoundsH int}` — populated from Scene.Camera config + Scene grid dimensions.
- `Update(playerX, playerY float64, dt float64)`:
  1. Compute player velocity from current vs. last position (used for look-ahead direction).
  2. Compute desired offset: target = clamp(player - camera, ±dead-zone/2). Look-ahead adds `sign(vel.X) * LookAheadX` (velocity-based per Phase 1 research; NOT facing-based).
  3. Smooth toward target: `camera += (target - camera) * (1 - exp(-k * dt))` where k is per-axis smoothing constant. Framerate-independent.
  4. **Clamp final camera position to level bounds** (not the target — clamping target breaks look-ahead near edges per research).
  5. Write to `pixelforge.Camera.X = int(camX); pixelforge.Camera.Y = int(camY)`.
- v1 ships horizontal-only camera follow (Mario-strip is 1 screen tall). Vertical follow exists in code but produces no movement when GridHeightScreens=1.
- Defaults from research: DeadZoneW=77 (30% of 256), DeadZoneH=96 (40% of 240), LookAheadX=32 px, SmoothingX=0.20, SmoothingY=0.40.
- **Edge case:** when player is *outside* the dead-zone, camera moves to keep them inside the dead-zone (not toward center). This is the correct Mario behavior and the source of the most common camera bug — explicit test scenario.

**Patterns to follow:** existing `pixelforge.Camera` global (`pixelforge.go:49-50`) — write target. Mark Brown / GMTK pattern + Cave Story velocity-based look-ahead (Phase 1 research).

**Test scenarios:**
- `TestFollower_PlayerInDeadZone_CameraStill`: player at (128, 120); camera at (128, 120); player moves to (130, 120) (still inside dead-zone); camera unchanged.
- `TestFollower_PlayerExitsDeadZone_CameraFollows`: player moves to (200, 120) (outside 77-px dead-zone); camera target = player.X - dead-zone-edge; after several Update ticks at 60fps, camera converges to target within 1 px.
- `TestFollower_LookAheadVelocityBased`: player moving right at +5 px/tick; computed target includes +32 px look-ahead in +X direction. Player decelerates and moves left; target's look-ahead component flips sign (smoothly over 0.3-0.5s per research).
- `TestFollower_HardClampAtLevelEdges`: player walking right toward edge; player.X > BoundsW - viewport/2; camera.X clamps to (BoundsW - viewport). Player can continue moving toward edge (player not clamped); camera stays put.
- `TestFollower_NoSnapOnFacingFlip`: player rapidly oscillates left-right (simulates a tap); camera does NOT snap or jitter. Asserts velocity-based look-ahead avoids facing-based snap.
- `TestFollower_FramerateIndependent`: same scene, simulate at 30fps and 120fps for 1 second of game time; final camera position within 1 px between the two runs (exp-based smoothing is framerate-independent).
- `TestFollower_VerticalNoMovementWhenSingleScreenTall`: GridHeightScreens=1, BoundsH=240; player at y=120, vertical camera offset never changes.
- `TestFollower_WritesPixelforgeCamera`: after Update, `pixelforge.Camera.X` and `.Y` equal int(camX) / int(camY) respectively.
- Covers AE4 (camera follows Player, stops at edge).

**Verification:** `go test ./pixelforge_camera/...` passes; race detector clean; manual smoke: load a Mario-strip fixture in the editor, simulate player movement in the preview, observe smooth scrolling with no jitter.

---

### U4. Engine package: pixelforge_entity (entity sprite renderer)

**Goal:** A small helper that iterates `Scene.Entities` and renders each entity's sprite at its tile-cell position. Respects `pixelforge.Camera`. Skips entities without a Sprite component.

**Requirements:** R4 (integer cell positions; renderer reads them), R7 (entity sprite renders at cell position).

**Dependencies:** U1 (schema), U2 (DrawSprite pattern; the entity renderer is structurally the same as the tilemap renderer but iterates Scene.Entities instead of TilemapLayer.Grid).

**Files:**
- `pixelforge_entity/render.go` (NEW)
- `pixelforge_entity/render_test.go` (NEW)
- `pixelforge_entity/doc.go` (NEW)

**Approach:**
- `RenderAll(entities []pixelforge_project.Entity, sprites []pixelforge_project.SpriteAsset)` — iterates entities; for each, looks up the Sprite component (existing `Entity.Components` slice with `Type: "Sprite"`); finds the named sprite asset; calls `pixelforge.DrawSprite` at `(entity.TileX * TileW, entity.TileY * TileH)`.
- **TileX / TileY are the new fields, not the float64 Position.** v1 entity placement (U7) writes these directly. Migration of existing pixel-coord entities at load time: round to nearest tile (handled in U1's `applyDefaults`).
- Entities without a Sprite component are skipped (silent, not an error — some entities are logic-only).
- Entities whose Sprite component references an unknown sprite name log a warning and skip (don't crash).
- Z-ordering: existing `EntityPosition.Z` (paint order) preserved as the iteration ordering hint.
- Camera offset applied automatically via `pixelforge.DrawSprite`'s existing `Camera` respect.

**Patterns to follow:** existing `pixelforge.DrawSprite` (pixelforge.go:115 for camera offset semantics); existing Entity + EntityComponent shape (`pixelforge_project/scenes.go:60-67, 82-85`).

**Test scenarios:**
- `TestRenderAll_EmptyEntities`: empty slice → no DrawSprite calls.
- `TestRenderAll_SingleEntityWithSprite`: one entity at TileX=10, TileY=24, Sprite component referencing "hero"; DrawSprite called once at scene-coord (80, 192) with "hero" sprite.
- `TestRenderAll_EntityWithoutSprite_Skipped`: entity has no Sprite component; no DrawSprite call for it; no error.
- `TestRenderAll_UnknownSpriteName_LogsWarning`: entity references "nonexistent" sprite; warning logged; no DrawSprite call; render continues for other entities.
- `TestRenderAll_RespectsCameraOffset`: pixelforge.Camera={X: 100, Y: 0}; entity at TileX=20 (scene-coord 160); resulting blit destination is screen-coord 60.
- `TestRenderAll_ZOrderingHonored`: three entities with Z = 1, 0, 2; rendered in order 1 (lowest Z first), 0, 2 (assertion via call order in mock).
- `TestRenderAll_MultiEntityScene`: 10 entities, mixed types (some with sprites, some without); only entities with sprites produce DrawSprite calls; ordering by Z.

**Verification:** `go test ./pixelforge_entity/...` passes; integration smoke: render a fixture scene; visually confirm entities appear at their cell positions, scrolled by the camera.

---

### U5. Editor preview drives the camera + renderers

**Goal:** Wire U2 (tilemap renderer), U3 (camera follower), U4 (entity renderer) into the existing editor preview pipeline. The Scene workspace's preview shows the rendered tilemap + entities + smooth scrolling without designer interaction needed beyond marking a Player.

**Requirements:** R10, R11 (camera follows in preview), the preview half of R5/R6 (designer sees painted tiles), AE4 (camera follow observable in preview).

**Dependencies:** U1 (schema), U2 (tilemap renderer), U3 (camera follower), U4 (entity renderer).

**Files:**
- `pixelforge_studio/editor/canvas_input.go` (MODIFY — sceneGame.Draw + sceneGame.Update extensions)
- `pixelforge_studio/editor/canvas.go` (MODIFY — `viewBox` honors camera offset; drag clamp widens to level bounds)
- `pixelforge_studio/editor/canvas_input_test.go` (MODIFY or NEW)
- `pixelforge_studio/editor/canvas_preview_test.go` (NEW)

**Approach:**
- `sceneGame.Update` (current shim from U5 of ImGui migration) extends to:
  1. Find the Player entity (`Entity.Archetype == "Player"` — idea #5 dependency).
  2. Compute player pixel position from `(TileX * TileW, TileY * TileH)`.
  3. Read scene's `Camera` config + grid bounds.
  4. Call `pixelforge_camera.Update(playerX, playerY, dt)` — this writes to `pixelforge.Camera`.
- `sceneGame.Draw` (current shim) extends to:
  1. For each `Scene.Tilemaps[i]`: call `pixelforge_tilemap.Render(layer, sheet)`.
  2. Call `pixelforge_entity.RenderAll(scene.Entities, project.Sprites)`.
  3. Existing entity-marker / selection-outline overlay continues to render *on top* of the new sprite-rendered entities (so designer can still see the selection box in the preview during authoring).
- `canvas.go` `viewBox` (canvas.go:225-243) extends to subtract `pixelforge.Camera.X` and `.Y` so the editor's coordinate conversion accounts for camera scroll. Without this, clicks on cells past the first screen would mis-target.
- `canvas.go:200-201` drag clamp widens: instead of `[0, ScreenWidth]`, clamp to `[0, GridWidthScreens * ScreenWidth]` and `[0, GridHeightScreens * ScreenHeight]`.

**Patterns to follow:** existing `sceneGame.Update` + `sceneGame.Draw` in `pixelforge_studio/editor/canvas_input.go`; existing `canvas.UpdateAt` mouse-coord mapping.

**Test scenarios:**
- `TestPreview_TilemapRenders`: load a fixture with a non-empty tilemap; simulate one preview frame; assert tilemap renderer called with the layer (verify via spy on DrawSprite call counts).
- `TestPreview_EntitiesRender`: load fixture with 3 entities (Player + 2 enemies); simulate frame; assert RenderAll called; all 3 entities produced DrawSprite.
- `TestPreview_CameraTracksPlayer`: fixture with Player at TileX=4; simulate Player moving to TileX=20 over several preview frames; assert `pixelforge.Camera.X` advances accordingly (passes through camera follower's smoothing).
- `TestPreview_CameraStopsAtLevelEdge`: Player at TileX=510 (near right edge of 16-screen level = 512 tiles); camera position clamped to (BoundsW - viewport).
- `TestPreview_NoPlayer_CameraStill`: fixture with no Player entity; sceneGame.Update doesn't update camera; pixelforge.Camera stays at (0, 0).
- `TestPreview_ViewBoxAccountsForCameraOffset`: with Camera.X=100, click at screen-coord (50, 0) → scene-coord (150, 0) per canvas.windowToScene mapping (camera offset added back).
- `TestPreview_DragClampWidensToLevelBounds`: drag entity past first screen (X=256); position clamps to GridWidthScreens*ScreenWidth-1, not ScreenWidth-1.
- Covers AE4 (camera follow observable in preview).

**Verification:** `go test ./pixelforge_studio/editor/...` passes; manual smoke: open studio with mario-strip fixture, move Player entity, observe camera scroll; click cell past first screen, place entity at expected coord.

---

### U6. Tile painter UI (Paint tool brush/bucket/rect + palette + undo)

**Goal:** Wire the existing-but-stub `Paint` tool with three sub-modes (brush, paint-bucket, rectangle-fill). Add a tile palette panel showing tiles from the active TilemapLayer's bound SpriteSheetRef. Per-stroke undo via command-pattern with cell-diff lists. Ghost-preview before commit. Right-click cell → "Set spawn here" context menu.

**Requirements:** R5 (three tools), R6 (explicit tile selection), R9 (spawn cell set via right-click).

**Dependencies:** U1 (schema fields), U5 (canvas.UpdateAt routes painter input). Parallel with U7.

**Files:**
- `pixelforge_studio/editor/tile_painter.go` (NEW — Paint tool handler with three sub-modes)
- `pixelforge_studio/editor/tile_palette.go` (NEW — palette panel showing tiles from bound sheet)
- `pixelforge_studio/editor/undo_stroke.go` (NEW — command stack with cell-diff payloads)
- `pixelforge_studio/editor/right_click_menu.go` (NEW — context menu dispatch)
- `pixelforge_studio/editor/tile_painter_test.go` (NEW)
- `pixelforge_studio/editor/undo_stroke_test.go` (NEW)
- `pixelforge_studio/editor/workspaces.go` (MODIFY — toolbar adds brush/bucket/rect sub-mode radio buttons under the existing Paint tool; renders palette panel when Paint tool active)
- `pixelforge_studio/editor/canvas.go` (MODIFY — UpdateAt dispatches to tile_painter when Tool == Paint; right-click dispatch)

**Approach:**
- `ToolPaint` (already in tools.go:11) gains three sub-modes via a new `PaintSubMode` enum: `Brush`, `Bucket`, `Rectangle`. Toolbar shows the sub-mode picker when Paint is active.
- Tile palette panel: ImGui child window showing tiles from the bound `TilemapLayer.SpriteSheetRef` as an `ImageButton` grid. Designer clicks a tile to select it as the active tile for brush/bucket/rect.
- **Brush sub-mode**: LMB-down paints; LMB-drag paints continuously (cell-by-cell, deduped per cell within a stroke). Ghost tile shown at hover cell.
- **Bucket sub-mode**: LMB click does 4-connected flood-fill, replacing all contiguously-matching cells starting from the clicked cell with the active tile ID. Per Phase 1 research, 4-connected is canonical (Tiled / LDtk / Aseprite).
- **Rectangle sub-mode**: LMB-down marks anchor; drag shows ghost rectangle preview; LMB-up fills the rect with the active tile.
- **Undo/redo per-stroke**: each tool action (full brush stroke, one bucket fire, one rectangle fill) is one command on the undo stack. Command payload: `[]CellDiff{Col, Row, OldID, NewID}`. Stack capacity 100 (per research).
- **Right-click handling** (greenfield — canvas has no right-click dispatch today): on right-click in the Scene preview, open an ImGui popup menu with "Set spawn here" (only when right-clicking an empty cell or a cell without an entity). Mutates `Scene.SpawnTile`, marks dirty.
- All mutations route through `MarkDirty()` per `docs/solutions/dirty-state-ux.md`.

**Patterns to follow:** existing toolbar pattern in `pixelforge_studio/editor/workspaces.go:200-208` (radio buttons); existing `comboField` for sub-mode picker (could reuse — though radio buttons match the existing tool palette aesthetic); Tiled / LDtk tile palette panel UX (Phase 1 research).

**Test scenarios:**
- `TestBrush_SingleClickPaintsOneCell`: LMB click at cell (5, 10) with brush tool + active tile ID=3; `Grid[10][5] == 3` after click.
- `TestBrush_DragPaintsMultipleCells`: LMB-down at (5, 10), drag through (5, 11), (5, 12), LMB-up; cells (5, 10), (5, 11), (5, 12) all = 3.
- `TestBrush_DedupesRepeatedCellsInStroke`: drag goes through (5, 10) → (6, 10) → (5, 10) again (jittered mouse); resulting cell-diff list has only two entries (5,10) and (6,10), not three.
- `TestBucket_FourConnectedFill`: grid `[[0,0,1],[0,0,1],[1,1,1]]`; bucket-click at (0, 0) with active tile=5; cells (0,0), (0,1), (1,0), (1,1) become 5; cells with original value 1 unchanged.
- `TestBucket_DoesNotFloodAcrossDifferentValues`: grid `[[0,1,0]]`; bucket-click at (0, 0); cell (0, 0) → 5; (0, 1) unchanged (different value), (0, 2) unchanged (not connected via 4-neighbors after the 1 blocks).
- `TestRectangle_DragPreviewThenCommit`: anchor at (5, 10), drag to (8, 12), commit; cells in rect (5,10)-(8,12) inclusive all become the active tile.
- `TestUndo_BrushStrokeReversed`: brush a 5-cell stroke; undo; all 5 cells revert to their previous values.
- `TestUndo_BucketFireReversed`: bucket fills 20 cells; undo; all 20 cells revert.
- `TestUndo_RedoAfterUndo`: undo a brush stroke; redo; cells restored to post-stroke values.
- `TestUndo_StackCapped`: perform 105 strokes; assert stack capacity ≤ 100; oldest strokes evicted; recent 100 strokes undoable.
- `TestRightClick_SetSpawnHere`: right-click on cell (8, 24); pick "Set spawn here"; `Scene.SpawnTile == {Col: 8, Row: 24}`; dirty flag set.
- `TestTilePalette_RendersTilesFromBoundSheet`: layer with SpriteSheetRef = "tiles_overworld"; palette panel renders the sheet sliced into tiles (verify via mock — actual rendering depends on a live ImGui context but the panel's selected-tile state is testable).
- `TestPainter_NoActiveTileSelected_PaintsZero`: brush with no tile selected (selected tile ID = 0) → paint produces clears, not no-op. (Design decision: tile 0 = clear; verify behavior.)
- Covers AE1 (R5, R6 — brush + bucket painting), part of AE3 (spawn set via right-click; preview-time effect verified in U7).

**Verification:** `go test ./pixelforge_studio/editor/...` passes; manual smoke: open studio with a sprite-sheet imported, select Paint tool, pick a tile from the palette, paint a 5-cell stroke; Ctrl+Z reverts; right-click a cell, set spawn, observe SpawnTile change in scene settings.

---

### U7. Entity placement on tile cells + Player marking + spawn flow

**Goal:** Modify the existing `Place` tool to snap to nearest tile cell. Right-click on an entity opens a context menu with "Set as Player" — toggling it mutates `Entity.Archetype = "Player"` (mutual-exclusivity: previous Player loses tag). On scene start (preview), Player entity teleports to `Scene.SpawnTile`.

**Requirements:** R7 (tile-cell placement), R8 (Player marking via right-click), R9 (spawn-cell honored at start), AE2 (Player tag mutual exclusivity), AE3 (spawn cell respected).

**Dependencies:** U1 (TileX/TileY fields), U5 (canvas dispatch); cross-reference idea #5's plan U2 (Archetype field on Entity).

**Files:**
- `pixelforge_studio/editor/canvas.go` (MODIFY — `PlaceEntity` snaps to tile cells)
- `pixelforge_studio/editor/right_click_menu.go` (extend from U6 — add "Set as Player" entry on entity right-click)
- `pixelforge_studio/editor/canvas_input.go` (MODIFY — sceneGame.Start hook teleports Player to SpawnTile)
- `pixelforge_studio/editor/canvas_test.go` (MODIFY)
- `pixelforge_studio/editor/right_click_menu_test.go` (NEW or extend from U6)

**Approach:**
- `PlaceEntity` (canvas.go:112-135) extension: instead of writing `Position.X = float64(sceneX); Position.Y = float64(sceneY)`, compute `TileX = sceneX / TileW`, `TileY = sceneY / TileH`. Write to new `Entity.TileX` + `TileY` fields (added in U1).
- The existing `Position` float field stays for backwards compatibility but is derived from TileX/TileY at load time (snap-to-nearest-tile). Code reads `TileX/TileY` going forward.
- Right-click on an entity (different from right-click on an empty cell — extend U6's right-click menu to detect what was clicked): show menu with "Set as Player" if entity is not currently Player; "Unmark as Player" if it is.
- **Mutual exclusivity**: when designer picks "Set as Player", iterate `Scene.Entities`; any entity with `Archetype == "Player"` has its Archetype cleared (set to ArchetypeWorld); the clicked entity's Archetype is set to "Player". Mark dirty. Only one Player per scene by construction.
- **Spawn flow**: when sceneGame starts a fresh preview run (scene reset, e.g., user clicks "Restart Preview"), find the Player entity; if Scene.SpawnTile is non-default, teleport Player to (SpawnTile.Col, SpawnTile.Row). If SpawnTile is unset (default 0,0), Player stays at its authored position.
- Drag clamp (already widened in U5) ensures dragged entities can move across the full level width.

**Patterns to follow:** existing `PlaceEntity` (canvas.go:112-135); existing dirty-state pattern (`docs/solutions/dirty-state-ux.md`); idea #5's Archetype field definition (cross-reference its plan U2).

**Test scenarios:**
- `TestPlaceEntity_SnapsToTileCell`: click at scene-coord (47, 193) with TileW=8 → entity placed at TileX=5, TileY=24 (since 47/8=5, 193/8=24).
- `TestSetAsPlayer_MarksArchetype`: entity has Archetype=Enemy; right-click → "Set as Player"; entity's Archetype becomes "Player"; dirty flag set.
- `TestSetAsPlayer_MutualExclusivity`: scene has entity A with Archetype=Player and entity B with Archetype=Enemy; right-click B → "Set as Player"; A's Archetype becomes World; B's Archetype becomes Player. Exactly one Player exists after.
- `TestSetAsPlayer_UnmarkRevertsToWorld`: entity has Archetype=Player; right-click → "Unmark as Player"; Archetype becomes World.
- `TestSpawn_PlayerTeleportsOnStart`: scene has Player at (TileX=2, TileY=24); SpawnTile=(8, 24); start preview; first frame after start, Player position is (8, 24).
- `TestSpawn_DefaultSpawnTileNoTeleport`: SpawnTile=(0, 0) (default); Player starts at authored position; no teleport.
- `TestPlaceEntity_OutsideLevelBoundsClamped`: attempt to place at scene-coord that exceeds GridWidthScreens*ScreenWidth; placement coords clamp to the last valid cell.
- `TestPlaceEntity_PreservesExistingComponents`: existing entity has Sprite + a behavior component; right-click "Set as Player"; only Archetype mutates; other components untouched.
- Covers AE2 (mutual exclusivity), AE3 (spawn cell respected).

**Verification:** `go test ./pixelforge_studio/editor/...` passes; manual smoke: place entity by clicking; verify TileX/TileY snap; right-click entity, set as Player; restart preview; observe teleport to spawn cell.

---

### U8. Scene resize UI (inspector spinners)

**Goal:** Scene-level inspector panel adds two spinners (grid width in screens, grid height in screens). Resize preserves existing painted content + entity positions. New cells default to `DefaultTileID` (visually empty).

**Requirements:** R3 (resize-preserves-content), F3 (designer expands level beyond default), AE5 (resize preserves coords).

**Dependencies:** U1 (schema fields).

**Files:**
- `pixelforge_studio/editor/scene_settings.go` (NEW — inspector panel for scene-level settings)
- `pixelforge_studio/editor/scene_settings_test.go` (NEW)
- `pixelforge_studio/editor/inspector.go` (MODIFY — render scene settings panel when scene is selected)

**Approach:**
- Scene settings panel: opens when designer selects the scene itself (not an entity within it). Shows: scene name, `GridWidthScreens` spinner, `GridHeightScreens` spinner, default tile picker (uses U6's palette panel for the picker UX), Camera config sub-panel (collapsed by default — advanced setting).
- Resize behavior:
  - **Grow**: extend `Tilemaps[i].Grid` with new cells defaulting to `DefaultTileID`. Existing content stays at its current coords.
  - **Shrink**: truncate `Tilemaps[i].Grid` at the new bounds. Entities with TileX/TileY outside the new bounds are flagged for the designer (toast: "N entities will be outside the new level bounds"); designer confirms via modal before commit. Spawn cell auto-clamps to new bounds.
- All mutations route through `MarkDirty()`.

**Patterns to follow:** existing inspector dispatch (`pixelforge_studio/editor/inspector.go`); existing dirty-state pattern; existing `confirm_modal.go` for the shrink-confirm.

**Test scenarios:**
- `TestResize_Grow_PreservesContent`: scene with Grid[10][5]=3, GridWidthScreens=16; resize to 24 screens; assert Grid[10][5] still 3; Grid[10][600] = DefaultTileID (new area).
- `TestResize_Grow_NewAreaDefaultTile`: scene with DefaultTileID=2; resize to add 10 columns; new columns are all 2.
- `TestResize_Shrink_TruncatesGrid`: scene with Grid extending to col 511; resize to 8 screens (256 cols); Grid truncated to col 255; assert no panic, no out-of-bounds.
- `TestResize_Shrink_OutOfBoundsEntitiesFlagged`: scene with entity at TileX=400; resize to 8 screens (256 cols max); confirm modal triggered listing 1 entity; on confirm, entity is moved to closest valid tile (or designer chooses delete — exact UX is a planning sub-decision but flagging happens).
- `TestResize_Shrink_ClampsSpawnTile`: SpawnTile=(300, 0); shrink to 8 screens; SpawnTile clamped to (255, 0).
- `TestResize_MarksDirty`: any successful resize sets the dirty flag.
- `TestSceneSettings_RendersOnSceneSelection`: select scene (not entity); inspector shows the settings panel; deselect; panel hidden.
- Covers AE5 (resize preserves coords).

**Verification:** `go test ./pixelforge_studio/editor/...` passes; manual smoke: open studio, select scene, change grid width from 16 to 24, paint a tile in the new area, save, reopen, painted tile persists.

---

### U9. End-to-end Mario-strip acceptance tests

**Goal:** Integration test that ties together U1-U8. Load a fixture Mario-strip project, simulate preview frames, assert all 5 acceptance examples from the origin are observable.

**Requirements:** R1-R13 covered transitively via AE1-AE5.

**Dependencies:** U1-U8 all merged.

**Files:**
- `pixelforge_studio/integration_test/mario_strip_e2e_test.go` (NEW)
- `pixelforge_studio/integration_test/fixtures/mario_strip_scene.pforge` (NEW)

**Approach:**
- Fixture: 16×1 grid scene; 1 sprite sheet imported (a few placeholder 8×8 tiles); Player entity at TileX=4, TileY=24 marked Archetype=Player; 2 Enemy entities at (TileX=10, TileY=24) and (TileX=20, TileY=24); SpawnTile=(4, 24); DefaultTileID=0; a horizontal strip of "ground" tiles painted at row 25.
- Test harness loads fixture; constructs sceneGame; simulates preview frames; asserts model state and camera position.
- **No headless rendering**; tests assert on `pixelforge.Camera.X/Y`, entity positions, tile values, scene settings — not pixels.

**Test scenarios** (one per AE):
- `TestE2E_AE1_BrushPaintsBucketFills`: from a fresh fixture, simulate brush stroke + bucket fill via the `TilePainter` API (not ImGui — direct API drive); assert resulting Grid cells.
- `TestE2E_AE2_PlayerTagMutualExclusivity`: load fixture; assert exactly one Player (Mario). Programmatically call "Set as Player" on Enemy A; verify Mario un-marked, Enemy A marked.
- `TestE2E_AE3_SpawnCellRespected`: load fixture; start preview; Player position becomes (4, 24) per SpawnTile, not its authored position (which we set deliberately different).
- `TestE2E_AE4_CameraFollowsAndStops`: load fixture; simulate Player walking right 30 cells (one cell per tick); assert camera.X advances after Player exits dead-zone; when Player approaches right edge (TileX > GridWidthScreens*32 - viewport/8), camera.X clamps to (GridWidthScreens*ScreenWidth - ScreenWidth).
- `TestE2E_AE5_ResizePreservesCoords`: load fixture; programmatically resize to 24 screens; assert painted ground row still present at row 25; assert all 3 entities still at their original TileX/TileY.
- `TestE2E_PreV1FileLoadsCleanly`: load a fixture saved without Scene grid fields, Camera config, SpawnTile — values default per U1's applyDefaults; preview runs without error.
- `TestE2E_FullPreviewLoop_NoPanic`: simulate 600 preview ticks (10 seconds at 60fps); no panics; final state consistent (camera position deterministic given Player movement).

**Verification:** `go test ./pixelforge_studio/integration_test/...` passes; all 5 AEs green; manual smoke: launch studio with `mario_strip_scene.pforge`, walk through the level using arrow keys, observe scrolling.

---

## Scope Boundaries

### Deferred to Follow-Up Work

- **Multi-scene navigation, scene-to-scene warps.** v1 = one scene per project; `Warps[]` schema reserved but unused.
- **Snap-per-screen camera (Zelda)**. `CameraMode` schema reserved; v1 uses smooth-follow only.
- **Painted zones** for music-swap / save-point / transition / scroll-mode-change. `Zones[]` reserved.
- **Vertical multi-screen levels (Metroid up/down)**. v1 grid height defaults to 1; designer can set higher but camera only follows horizontally.
- **RPG-class systems** (menus, dialogue, inventory, save state). Idea #6's plan.
- **Audio editor surface**. Idea #4's plan.
- **Sprite-sheet slicer + Aseprite/Tiled/LDtk importers**. v1 assumes the designer has a sprite-sheet imported with frames known. A slicer is a parallel concern (ideation surfaced it as a separate deliverable).
- **WASM / cross-platform / auto-icon ship**. Idea #7's plan.
- **Tilemap auto-rules (LDtk IntGrid + Auto-layer)**. Idea #2's plan ships the upgrade path.
- **AI-assisted level generation, parametric sprites, cellular-automata room generation**. Out — likely out of product identity too.
- **Mobile / touch authoring; browser-based studio**. Out of product identity.
- **Tile-painter Line tool** (Bresenham between anchor and cursor). Phase 1 research called it out as the "inferred 4th"; deferred — three tools cover ~95% of authoring.
- **Multi-tile stamp selection** (drag a rect in the palette to select a multi-tile stamp the brush then uses). v1 ships single-tile selection; multi-tile stamp is a v2 painter polish.
- **Right-click cell deletion** (delete-tool alternative via context menu). v1 ships Delete tool + brush-with-0; right-click delete deferred.
- **Tile-painter shortcut keys** (B/G/R for brush/bucket/rect; bracket keys for tile cycling). v1 ships ImGui radio buttons; hotkeys later.

### Outside this product's identity

- Multiplayer / network-synchronized scenes.
- Open-world streaming (chunk-load levels too large to fit in memory).
- AI-assisted tile generation (LLM proposes tile placements).
- Cloud-hosted level sharing built into the studio.

---

## Key Technical Decisions

- **Zero external dependencies.** Phase 1 research evaluated `lafriks/go-tiled` (Tiled importer — not needed in v1), Ebitengine camera libraries (`willow` is general-purpose; ~80 LOC custom is cleaner), and tile-painter libraries (none). All rejected per the leverage doctrine. Total custom code: ~540 LOC across U2/U3/U4/U6 (engine + painter), well under the wrap cost of any candidate. Port the Mark Brown / GMTK camera pattern and Tiled palette UX as design reference only.
- **Single rendering pipeline for editor and shipped game.** The engine's `pixelforge_tilemap`, `pixelforge_entity`, and `pixelforge_camera` packages are called from both the editor's `sceneGame.Update`/`Draw` shim AND the shipped runtime (via idea #7's Capsule). This is the structural guarantee for "what the designer sees in preview matches what ships."
- **Camera = velocity-based look-ahead, not facing-based.** Per Phase 1 research — facing-based look-ahead causes nauseating snaps on tap; velocity-based naturally damps. Cave Story / SMW pattern.
- **Camera dead-zone defaults: 30% horizontal, 40% vertical** of the 256×240 viewport. Per Phase 1 research convergence. Designers can override per-scene in the Camera config but the defaults work for Mario-class.
- **Camera bounds clamping at the camera, not the target.** Clamping the target breaks look-ahead near edges (Mario keeps moving right but camera stops; if target is also clamped, look-ahead pulls the camera back into the level). Per Phase 1 research.
- **Player tag = `Entity.Archetype == "Player"`** (idea #5 dependency, not new in this plan). Mutual exclusivity enforced in the right-click "Set as Player" handler (U7); only one Player per scene.
- **Spawn cell as scene-level property, not a painted zone.** Zones deferred; spawn needs to exist anyway. Right-click cell → "Set spawn here" + `Scene.SpawnTile` field gives the designer the same UX without introducing the zone concept.
- **Schema reservation for future-genre fields now.** `World{Mode, Rooms[]}`, `Zones[]`, `Warps[]`, `CameraMode` land as `omitempty` fields in v1 so v1 projects stay forward-compatible. Honors the project's established additive-only schema discipline.
- **`Paint` tool is expanded, not replaced.** The enum value exists in `tools.go:11` already; the toolbar shows the radio button; clicking does nothing. v1 wires it up rather than minting parallel tools — keeps the tool palette stable.
- **Per-stroke undo with cell-diff lists, not per-cell undo or full-grid snapshots.** Per Phase 1 research — per-cell is universally rejected as miserable UX; full snapshots are wasteful for large grids. Command stack with `[]CellDiff{Col, Row, OldID, NewID}` is the Tiled / LDtk / Aseprite convention.
- **Tile size 8×8 base, fixed in v1 by editor convention.** `TilemapLayer.TileW/TileH` already in schema (per-layer); v1 doesn't add project-level tile-size config. 16×16 metatile grouping is a v2 polish.
- **Entity positions: dual representation.** New `TileX`/`TileY` int fields (canonical going forward); existing float64 `Position.X/Y` derived (snap-to-nearest-tile at load time). Backwards compatibility via the load-time derivation.
- **Codegen integration deferred to idea #7's plan.** The engine renderers ship here (U2/U3/U4); the shipped runtime wiring is idea #7's Capsule. v1 of this plan proves the model works in the editor preview; idea #7 carries it to the ship loop.

---

## Dependencies / Assumptions

- **Existing engine** (`pixelforge`, `pixelforge_ebiten`) continues to expose `DrawSprite`, `SetScreenSize`, `Camera Position` global. v1 of this plan adds three new packages but modifies no existing engine code.
- **Existing studio Scene workspace + canvas + sceneGame texture pipeline** (post-ImGui-migration U5) continues to work. This plan modifies `canvas.go`, `canvas_input.go`, `workspaces.go` to integrate the renderers + camera + painter.
- **Existing project schema patterns** (additive `omitempty`, `applyDefaults`, `normalizeSlices`) handle the v1 schema additions per `docs/solutions/editor-pforge-schema-shape.md`.
- **Idea #5's Archetype field on Entity** (idea #5's plan U2). Without it, the Player tag mechanism in U7 doesn't work. **Strict dependency:** idea #5's plan U2 must land before this plan's U7.
- **Idea #7's plan** owns the shipped-binary half of the bet. R12 is satisfied transitively via idea #7's Capsule + Build pipeline. v1 of this plan proves the model works in the editor preview.
- **Existing `pixelforge.Camera`** global (pixelforge.go:49-50) is the camera-write target. No engine change required to add the camera primitive; the new `pixelforge_camera` package writes to the existing global.
- **Existing `confirm_modal.go`** for the shrink-confirm dialog (U8). Verified to exist.

---

## Risk Analysis & Mitigation

| Risk | Severity | Mitigation |
|---|---|---|
| Tilemap renderer perf degrades for 16-screen levels (16 × 30 tiles × 30 rows = 14,400 cells) | Medium | U2 includes a perf benchmark (`TestRender_PerformanceBudget`) gated at 8 ms. If perf misses budget, add viewport culling (cell-skip outside camera + margin). |
| Camera follower jitters or snaps on rapid input changes | Medium | U3 includes `TestFollower_NoSnapOnFacingFlip` + `TestFollower_FramerateIndependent`. Velocity-based look-ahead avoids the snap class. Smoothing constants are config-tunable per-scene. |
| Tile-cell entity coords lose precision vs existing pixel-coord entities | Low | U1 migration snaps existing pixel coords to nearest tile at load time. Tests verify no entity position changes by more than TileW/2 (the snap distance). |
| Mutual-exclusivity bug — designer ends up with multiple Players | Medium | U7's `TestSetAsPlayer_MutualExclusivity` explicit. Right-click handler iterates ALL entities, not just the selection, before marking the new Player. Defensive sanity check in sceneGame.Update (warns if >1 Player found). |
| Right-click context menu interferes with existing left-click selection | Low | ImGui's BeginPopupContextItem pattern is well-isolated; popup closes on click outside. Existing left-click flow untouched. |
| Schema migration of `cart_assets/editor.pforge` introduces unintended changes | Low | U1's `TestProject_PreV1FixtureRoundtrip` explicit. The fixture has empty Entities + Tilemaps; the only change should be additive default values, which are `omitempty` and shouldn't appear in JSON if equal to defaults. |
| `pixelforge.Camera` global state causes thread-safety issues if editor preview and main loop write concurrently | Low | Engine is single-threaded per Ebitengine convention; `Camera` writes are tick-bound. Document the contract in the new `pixelforge_camera` package. |
| The drag-clamp widening (U5) breaks existing single-screen examples | Low | Pre-v1 scenes have GridWidthScreens=0 by default; `applyDefaults` (U1) sets it to 16 (default level width). Behavior change is opt-in via the schema default. Examples that explicitly want single-screen behavior can set GridWidthScreens=1 (256 px), preserving prior clamp. |
| Coupling to idea #5's plan blocks shipping if idea #5 slips | High | Document strict dependency. Idea #5 should ship before or alongside idea #1. If idea #5 slips, fall back: implement a temporary `Entity.IsPlayer bool` field in U7 instead of using Archetype; remove when idea #5 lands. Migration: convert `IsPlayer: true` to `Archetype: "Player"` at load time. |

---

## System-Wide Impact

**New packages introduced:** `pixelforge_camera`, `pixelforge_tilemap`, `pixelforge_entity`. Three new package roots in the engine; all follow `pixelforge_*` naming. None pull external dependencies. None modify existing engine code.

**Modified packages:**
- `pixelforge_project` — schema additions on `Scene` + `TilemapLayer`; new types in `world_reserved.go`; extended `applyDefaults` / `normalizeSlices`.
- `pixelforge_studio/editor` — `canvas.go`, `canvas_input.go`, `workspaces.go`, `inspector.go` extended; six new files (`tile_painter.go`, `tile_palette.go`, `undo_stroke.go`, `right_click_menu.go`, `scene_settings.go`, plus tests).

**Affected workflows:**
- **Designer authoring** — the primary workflow target. New: tile painter, right-click context menus, scene-resize inspector, scrolling preview.
- **Engine renderers** — three new engine packages that the editor preview consumes today and idea #7's Capsule consumes at ship time. The renderers are pure additions; no existing code paths change semantics.
- **Existing project examples** — `cart_assets/editor.pforge` (the only existing fixture) round-trips cleanly because new fields are `omitempty` defaults. No active migration needed.
- **Game runtime (post-idea-#7)** — idea #7's Capsule will call the same `pixelforge_tilemap.Render` / `pixelforge_camera.Update` / `pixelforge_entity.RenderAll` calls in its emitted `main.go`. v1 of this plan does NOT modify codegen; that's idea #7's plan.

**Documentation impact:** None at v1 — designers discover the new tools in the toolbar; the schema fields are additive. Post-v1 `/ce-compound` should capture (1) the engine camera-follow + dead-zone pattern, (2) the per-stroke undo command pattern, (3) the dual-tile-coord-with-load-time-snap pattern as `docs/solutions/` entries.

**Operational / rollout:** Standard release. No data migration required for the single existing fixture. Designers opt into the new world primitive by editing scene settings; old projects load with default Mario-strip-sized scenes (16 × 1) — those defaults may look surprising for a designer who had a single-screen game, but the only existing fixture is the editor cart itself which has no entities or tiles so the change is invisible. If a real pre-v1 single-screen project surfaces during rollout, the designer can override `GridWidthScreens` to 1 in the scene settings.
