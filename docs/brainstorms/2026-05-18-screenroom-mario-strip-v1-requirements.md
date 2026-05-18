---
date: 2026-05-18
topic: screenroom-mario-strip-v1
origin: docs/ideation/2026-05-18-pixelforge-nes-class-no-code-ideation.md (idea #1)
---

# ScreenRoom — Mario-Strip v1

## Summary

v1 of the world-authoring primitive is the **Mario-strip**: one scene per project, a bounded NES-native tile grid (default ~16 screens wide × 1 tall, 8×8 tiles at 256×240 pixels per screen), smooth-scroll camera following a designated Player entity, and tile painting + entity placement. Deliberately excludes Zelda / Metroid / Final Fantasy / Tetris / Punch-Out / Excitebike — those genres land later via extensions of the same primitive with schema reserved now.

---

## Problem Frame

Pixelforge Studio just completed its ImGui chrome migration (2026-05-18). The studio has dockable workspaces, a reflection-driven inspector, scene-as-texture live preview, and a codegen pipeline that produces a thin Go shim + vendored engine. What it does *not* have is any concept of a *world* the designer authors.

Today, `Scene.Tilemaps` is a single flat grid per scene with no relationship to anything else; there is zero matches for `camera`, `scroll`, or `parallax` in `pixelforge_project/` or `pixelforge_studio/`. Entities exist as flat positioned-by-pixel records. The studio can place a sprite on a single static screen — but it cannot author a level that scrolls, much less an entire NES-class game.

The designer audience (friends / classmates community, not pre-trained on game-editor metaphors like GDevelop's event sheets or RPG Maker's map tree) needs a world-authoring primitive that feels intuitive *without* prior game-editor knowledge. The historical Nintendo workflow — designers drew levels on graph paper, engineers transcribed — points at the right metaphor: the designer paints freely on a tile grid, the system handles screen-quantum and scrolling behind the scenes. Anything more abstract leaks engineering concepts (rooms, nametables, attribute tables) into the designer's mental model.

This brainstorm scopes the smallest version of that primitive — enough to prove a designer can paint a Mario-class side-scrolling level and ship it, before committing to Zelda's snap-per-screen rooms, Metroid's connected world map, or Final Fantasy's scene-to-scene warps.

---

## Actors

- **A1. Designer (Player-1).** The friend / classmate authoring a game. Comfortable making sprites and audio. Not pre-trained on game-editor metaphors. Sits down expecting to paint a level visually and see it run.
- **A2. Pixelforge Studio.** The editor application. Reads the project file, surfaces the painting + entity-placement workspaces, drives the always-on scene preview, and writes the project file on save.
- **A3. The shipped game.** The single-file binary the designer hands to a classmate. Runs the project file via the runtime capsule (covered separately under idea #7 of the ideation).

---

## Key Flows

- **F1. First-time author paints a Mario-class level**
  - **Trigger:** Designer opens a new project, picks a "Mario-strip" template (or default), and lands in the Scene workspace
  - **Actors:** A1, A2
  - **Steps:** (1) Designer sees a bounded tile grid (~16 screens wide × 1 tall = 512 × 30 tiles), pre-filled with a default sky/ground or empty; (2) selects a tile from a sprite-sheet palette and paints cells with brush / paint-bucket / rectangle tools; (3) drops a Player entity at a tile cell and marks it as the spawn point; (4) drops enemies / pickups on other cells; (5) clicks Play in the scene preview and the camera follows the Player smoothly across the level
  - **Outcome:** A playable scrolling level exists inside the studio preview
  - **Covered by:** R1, R2, R3, R4, R5, R7, R8

- **F2. Ship the level as a single-file binary**
  - **Trigger:** Designer confirms the preview plays correctly and clicks the ship/build action (full UX defined under idea #7)
  - **Actors:** A1, A2, A3
  - **Steps:** (1) Studio invokes the build pipeline for the current host platform (e.g., Linux binary); (2) project file + assets are embedded; (3) binary appears in a known directory the designer can drag from
  - **Outcome:** A single executable file exists that the designer can hand to a classmate on the same OS
  - **Covered by:** R10, R11 (these define the *requirement* on the ship path; the full ship pipeline is idea #7's brainstorm)

- **F3. Designer expands the level beyond the default width**
  - **Trigger:** Designer fills the default 16-screen grid and wants more space
  - **Actors:** A1, A2
  - **Steps:** (1) Designer opens scene settings; (2) edits the grid width in screens; (3) studio extends the painted area with empty tiles on the right (or left, if extending backward)
  - **Outcome:** The grid is resized; existing painted content is preserved at its current coordinates
  - **Covered by:** R3, R6

---

## Requirements

**Scene & world shape**

- R1. The scene contains exactly **one bounded 2D tile grid** of fixed width × height measured in screens. The default is 16 screens wide × 1 screen tall.
- R2. A "screen" is exactly **256 × 240 pixels** (32 × 30 tiles of 8 × 8). The camera viewport at any moment shows exactly one screen.
- R3. The designer can resize the grid (in scene settings) at any time. Existing painted content stays at its grid coordinates; new area appears empty.
- R4. The grid has integer coordinates; entities and tiles live on integer cell positions (not free-float pixel coordinates).

**Painting**

- R5. The painter offers three tools selectable from the Scene workspace toolbar: **single-tile brush, paint-bucket flood-fill, rectangle-fill**.
- R6. The designer picks tiles from a sprite-sheet palette and paints them explicitly into cells. (Auto-rules and semantic-IntGrid-style painting are out of v1 — see idea #2 of the ideation.)

**Entities & spawn**

- R7. The designer places entities on grid cells. Each entity is positioned at an integer (col, row).
- R8. Exactly one entity per scene is marked as the **Player**. The designer marks it by right-clicking the entity (or its cell) and choosing "Set as Player". The Player tag is mutually exclusive — marking a new entity un-marks the prior one.
- R9. The scene stores a **spawn cell** as a scene-level property (cell coordinates, not a painted region). The designer sets it by right-clicking any cell and choosing "Set spawn here". On level start, the Player entity teleports to this cell.

**Camera**

- R10. The camera **smoothly follows** the Player entity using a **dead-zone with slight look-ahead** in the player's facing direction. (Specific dead-zone and look-ahead distances are tuning parameters resolved during planning; "Mario-like" is the target feel.)
- R11. The camera **hard-stops at level bounds** — it never reveals area outside the painted grid. If the Player approaches an edge, the camera stops scrolling while the Player can continue moving toward the edge.

**Shipping**

- R12. The bet (designer ships a binary to a classmate) requires a working ship pipeline. The minimum required from idea #7 to validate v1 is: **build to a current-platform native binary** (the OS the designer is running). Cross-platform, WASM, and auto-icon are covered by idea #7's own brainstorm; v1 of ScreenRoom only requires that *some* binary exists.

**Schema forward-compat**

- R13. The project schema reserves the following fields as `omitempty` now, even though no v1 UI walks them: `World{Mode, Rooms[]}` (for future GridVania / Metroid layouts), `Zones[]` (for future music-swap / save / transition zones), `Warps[]` (for future multi-scene navigation), `CameraMode` (for future snap-per-screen / locked-per-zone modes). This honors the project's established schema-first / additive-only discipline so v1 projects load cleanly in v2+ builds.

---

## Acceptance Examples

- **AE1. Covers R5, R6.** Given an empty scene with the brush tool active, when the designer left-clicks a cell, the cell takes the currently-selected tile's pixels. When the designer clicks the paint-bucket and clicks a cell, every contiguously-empty cell connected to the clicked cell fills with the selected tile.
- **AE2. Covers R8.** Given a scene with three entities (Player, GoombaA, GoombaB) and the Player tag on the Player entity, when the designer right-clicks GoombaA and picks "Set as Player", the Player entity loses its tag and GoombaA gains it. The scene now has exactly one Player.
- **AE3. Covers R9.** Given a scene with the Player entity placed at cell (3, 24) and the spawn cell set to (5, 24), when the designer clicks Play in the preview, the Player entity appears at cell (5, 24) — not at (3, 24).
- **AE4. Covers R10, R11.** Given a 16-screens-wide level with the Player at cell (10, 24), when the designer presses Right in the preview, the Player walks right; once the Player crosses the dead-zone, the camera begins scrolling smoothly. When the Player reaches the rightmost screen, the camera stops at the level bound, but the Player can still walk to the edge.
- **AE5. Covers R3.** Given a scene authored at the default 16 × 1 grid with painted content at cells (200, 15) through (400, 25), when the designer changes the grid width to 24 screens, the painted content remains at its existing coordinates and the area from cell (512, 0) to (768, 30) appears empty.

---

## Success Criteria

- **Designer outcome:** A first-time designer with no prior game-editor experience can open Pixelforge Studio, paint a wide tile-grid using the brush + bucket + rectangle tools, drop a Player + a handful of enemies and pickups, set the spawn point, and watch a complete Mario-class scrolling level play through in the preview — without consulting documentation.
- **Ship-loop outcome:** The same designer can produce a single-file native binary they hand to a classmate (via idea #7's build pipeline), and the classmate double-clicks the file and plays the level identical to what the designer saw in preview.
- **Downstream handoff outcome:** The planning agent (`/ce-plan`) consuming this doc does not need to invent any user-facing behavior. Every requirement is observable; every tool and field name is decided at the scope-level; only implementation specifics (which Go types, which file paths, which functions) are open for planning to resolve.

---

## Scope Boundaries

- **Multi-scene navigation** — out of v1. One scene per project. Scene-to-scene warps land later (see idea #6 dependencies and the `Warps[]` schema reservation).
- **Snap-per-screen camera (Zelda)** — out of v1. The `CameraMode` schema field is reserved so the snap mode lands additively.
- **Painted zones** — out of v1. No music-swap / save-point / transition / scroll-mode-change zones. The `Zones[]` schema is reserved.
- **Vertical multi-screen levels (Metroid up/down sections)** — out of v1. Grid height stays at 1 screen for the default template; designers can resize but the camera-following code only handles horizontal scroll in v1.
- **RPG-class systems** (menus, dialogue, inventory, save state) — out of v1; covered by idea #6 of the ideation.
- **Audio editor** — out of v1; covered by idea #4 of the ideation. (Audio playback at runtime still works through engine primitives, but the studio surface for composing/binding audio is separate.)
- **Sprite-sheet slicer + Aseprite/Tiled/LDtk importers** — out of v1 for this brainstorm specifically. Tile painting in v1 assumes the designer already has a sprite-sheet imported with frames known. Slicing UX is a parallel concern (the ideation surfaced it as a discrete deliverable).
- **WASM / cross-platform / auto-icon ship** — out of v1; idea #7's brainstorm. v1 requires only "build to current host platform."
- **Tilemap auto-rules (LDtk IntGrid + Auto-layer)** — explicitly out of v1; covered by idea #2 of the ideation. v1 is explicit tile selection.
- **AI-assisted level generation, parametric sprites, cellular-automata room generation** — out of v1 (and likely out of the product entirely per ideation rejections).
- **Mobile / touch authoring; browser-based studio** — out of product identity. The studio stays native desktop.

---

## Key Decisions

- **Graph-paper-world mental model.** Designer paints freely on one large bounded grid per scene. "Room" and "screen" are NOT authoring concepts — they only surface for camera behavior and (later) zone-based transitions. Picked over the LDtk-GridVania rooms-on-a-grid alternative because the designer audience is not pre-trained on game-editor metaphors and historical Nintendo workflow (graph paper) used the unbounded-grid mental model.
- **NES native 256 × 240 screen.** Picked over the Pixelforge engine's existing 320 × 180 default to maximize the "looks like a real NES cart" identity. Picked over PICO-8-style 128 × 128 because the user's reference set (Mario, Zelda, etc.) is unambiguously NES-class, not Pico-8-class.
- **Mario-strip v1 cut.** Smallest version that still proves the primitive: one scene, bounded ~16 screens × 1 tall, smooth scroll, no zones, no warps. Excludes Zelda / Metroid / FF deliberately; those genres prove their own bets in later units.
- **Explicit tile selection in v1; auto-rules deferred.** Idea #2 of the ideation specifies LDtk-style semantic-brush + auto-rule painting as a more ambitious painter. v1 ships the explicit-selection version because (a) it covers Mario-class authoring fully and (b) auto-rules require a tileset-author-time investment that this brainstorm's scope doesn't engage.
- **Player as a tag, not a hard-coded entity.** Designer marks any entity as Player; the camera and spawn logic key off the tag. Picked over a hardcoded "Player" entity type so future games can use the same primitive for different player concepts (Mario, Mega Man, the chosen one in an RPG).
- **Spawn as a scene-level cell property, not a painted zone.** Zones are out of v1; spawn needs to exist anyway. A right-click-cell-set-spawn mechanism gives the designer the same UX without introducing the zone concept.
- **Schema reservation for future-genre fields now.** `World{Mode, Rooms[]}`, `Zones[]`, `Warps[]`, `CameraMode` land as `omitempty` fields in v1 so v1 projects load cleanly in v2+ builds. Honors the project's established schema-first discipline.

---

## Dependencies / Assumptions

- **Depends on idea #7 (Project Capsule + Build pane)** shipping at least a "current-platform native binary" path. Without it, R12 isn't satisfied and the bet isn't proven. Cross-platform / WASM / auto-icon are idea #7's own brainstorm; this brainstorm doesn't define them.
- **Depends on existing studio primitives** — the dockable Scene workspace (post-ImGui-migration), the reflection-driven inspector, scene-as-texture preview, and codegen vendored shim are all assumed to be the substrate v1 builds on. No regression-risk to those.
- **Assumes sprite-sheet import exists** in some form (manual file copy + JSON entry is acceptable for v1). The slicer UX from the ideation is a parallel deliverable; v1 of ScreenRoom doesn't define it.
- **Assumes the engine's existing `pixelforge_routine` Step sequencer + `pievent` pub/sub** are the substrate for entity behaviors. Entity behaviors themselves (verb-sheets, intent layer, blackboard) are idea #5's brainstorm.

---

## Outstanding Questions

### Resolve Before Planning

- *(none — scope is fully resolved at the brainstorm level)*

### Deferred to Planning

- **[Affects R10] [Technical]** Exact dead-zone size + look-ahead distance for the camera follow shape. "Mario-like" is the target feel; specific pixel values are tuning numbers planning resolves with reference to the canonical Mario implementations.
- **[Affects R7, R8] [Technical]** How the Player tag is represented in the project schema — a boolean field on `Entity`, a separate `PlayerEntityID` scene-level field, or a `pfcomponent.Register("Player")` tag-only component. Each round-trips the same observable behavior.
- **[Affects R13] [Technical]** Whether the v1 codegen reads the reserved-but-unused `World`/`Zones`/`Warps`/`CameraMode` fields (and ignores them) or whether the codegen skips them entirely until v2 UI walks them. Either preserves the additive-schema discipline; planning picks the cleaner shape.
- **[Affects R12] [Needs research]** What "current-platform native binary" means concretely under Ebitengine + the existing codegen — does `go build ./...` from the export directory already produce a runnable binary on the host platform, or does idea #7's brainstorm need to define it? (Likely already works; planning verifies.)
