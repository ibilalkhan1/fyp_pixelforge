---
date: 2026-05-18
topic: nes-palette-art-director-v1
origin: docs/ideation/2026-05-18-pixelforge-nes-class-no-code-ideation.md (idea #3)
---

# NES Palette as Art-Director + Central Quantizer + Soft-Warn Constraints — v1

## Summary

v1 ships an NES sub-palette overlay on top of the existing 64-color engine palette (8 named sub-palettes — 4 background + 4 sprite — each referencing 4 slots from `Base[64]`); every sprite and tile binds to one sub-palette so changing a color restyles the whole game live in the preview. The existing engine-side `QuantizeRGBA` wires into every image-import path with a ghosted before/after diff offering Accept / Re-quantize / Reject. Two NES-authenticity overlays — 8-sprites-per-scanline and 2×2 background-tile palette-block — ship as toggleable visualizations in the studio. Soft-warn, never gates, never present in the shipped game.

---

## Problem Frame

Pixelforge's engine ships a 64-color indexed palette and 4 ColorTables that already give it the bones of a retro-look game framework. The studio's chrome migration completed in 2026-05-18, the inspector renders any `pfcomponent` field through reflection, and scene-as-texture preview already exists. But the visual identity story is incoherent:

- `PaletteData.Base [64]string` is freely editable to 64 arbitrary RGBs. There's no NES sub-palette concept — designers can use any color anywhere, on anything.
- `pixelforge_studio/palette/quantize.go` ships a working RGB-ΔE quantizer with `fast/balanced/quality` presets, but **no studio code calls it during asset import**. A designer dragging in a 24-bit Photoshop mockup gets... a 24-bit Photoshop mockup, not a quantized NES-styled sprite.
- The NES had famous, distinctive constraints that produced its visual identity: 8 sprites per scanline (more causes flicker), 2×2 background-tile blocks share one palette quadrant (the "attribute table" rule — most NES games' instantly-recognizable look comes from this constraint). Pixelforge enforces neither, visualizes neither, mentions neither.

The result: "NES-class" is currently a vibe, not a constraint. A designer with no NES experience produces art that looks vaguely retro but not authentically NES. A designer with NES experience who *wants* the constraint can't get it. Pico-8's reputation — and the discipline of its community — comes from constraints that are visible everywhere; Pixelforge has the engine bones for the same discipline but no studio surface that engages them.

This brainstorm scopes v1 of the visual-identity layer: the smallest cut that makes the palette an art-director (one color changes the whole game), makes asset import safe-by-construction (any PNG becomes NES-styled), and makes NES authenticity visible without gating.

---

## Actors

- **A1. Designer.** Imports PNGs, paints sprites and tiles, picks palette colors. Comfortable creating visual assets. Not pre-trained on NES hardware constraints — learns them through the studio's overlays.
- **A2. Pixelforge Studio.** Surfaces the palette workspace (where designer manages sub-palettes), invokes the quantizer on imports, drives the sub-palette assignment UI in the sprite + TileAtlas inspectors, renders the constraint overlays in the Scene + Sprite editors.
- **A3. Existing engine palette pipeline.** `pixelforge_project.PaletteData` (64-color base + 4 ColorTables), `pixelforge_studio/palette/quantize.go` (RGB-ΔE quantizer, unchanged in v1).

---

## Key Flows

- **F1. Designer changes a palette color and the whole game restyles**
  - **Trigger:** Designer opens the Palette workspace and edits one of the 64 base colors (e.g., changes brown #8B4513 to grey #808080)
  - **Actors:** A1, A2, A3
  - **Steps:** (1) Designer picks a slot in the palette grid and opens the color picker; (2) drags to a new color; (3) every sprite and tile that uses any sub-palette referencing that slot updates in the live preview within one frame; (4) save persists the change to `Base[]`
  - **Outcome:** The whole game's visual identity shifts in real time — designer sees the impact of palette choices immediately.
  - **Covered by:** R1, R5, R8

- **F2. Designer drops a 24-bit PNG into the studio**
  - **Trigger:** Designer drags a `hero.png` (24-bit color) into the asset browser
  - **Actors:** A1, A2, A3
  - **Steps:** (1) Studio detects the import is a non-quantized image; (2) auto-picks the best-matching sub-palette (or uses the designer's pre-selected target); (3) runs `QuantizeRGBA` against that sub-palette; (4) shows a ghosted before/after diff in a modal with three buttons — Accept / Re-quantize / Reject; (5) if mean RGB ΔE exceeds a threshold, a warning banner appears above the diff ("significant color shift — consider a different sub-palette"); (6) designer chooses an action
  - **Outcome:** The imported sprite is NES-styled (Accept) or replaced with a different quantization (Re-quantize) or skipped (Reject).
  - **Covered by:** R2, R3, R4

- **F3. Designer places too many sprites on one row and sees the flicker warning**
  - **Trigger:** Designer authors a level scene with 9+ entities on the same horizontal slice (one scanline's worth of pixels)
  - **Actors:** A1, A2
  - **Steps:** (1) Designer places the 9th entity; (2) the Scene workspace's overlay flags the affected scanline with a colored highlight (e.g., red horizontal band); (3) tooltip on hover explains "NES would flicker — more than 8 sprites on this scanline"; (4) designer can leave it (warning is soft) or rearrange
  - **Outcome:** The designer sees the violation before shipping; they decide whether to fix it or leave it (flicker was historically a deliberate technique).
  - **Covered by:** R6, R7

- **F4. Designer paints adjacent BG tiles using incompatible palette quadrants**
  - **Trigger:** Designer paints tile A with sub-palette `bg_0` next to tile B with sub-palette `bg_1`, violating the 2×2 BG palette-block rule
  - **Actors:** A1, A2
  - **Steps:** (1) Designer paints; (2) Scene workspace overlay flags the affected 2×2 block with a colored outline; (3) tooltip explains "real NES requires every 2×2 background block share one palette quadrant"; (4) designer can fix or ignore
  - **Outcome:** The constraint is visible; the designer chooses authenticity vs flexibility per-tile.
  - **Covered by:** R6, R7

---

## Requirements

**Sub-palette structure**

- R1. The project schema adds an **NES sub-palette overlay** on top of `PaletteData.Base [64]string`: **8 named sub-palettes** (4 background — `bg_0..bg_3`; 4 sprite — `sprite_0..sprite_3`), each referencing **4 of the 64 base slots** by index. The engine's existing 64-color base palette is **unchanged**; sub-palettes are an additive layer.
- R2. Every `SpriteAsset` carries a **sub-palette assignment field** (`omitempty`) naming one of the four sprite sub-palettes. Default when unset: `sprite_0`.
- R3. Every `TileAtlas` (from idea #2) carries a **per-tile sub-palette assignment** (`omitempty`). The granularity follows the NES attribute-table constraint: assignment is per-2×2-BG-tile-block by default, with per-tile override possible. Default when unset: `bg_0`.
- R4. The schema additions are `omitempty` so projects saved before v1 load cleanly with default assignments applied at load time.

**Live restyle**

- R5. Changing a color in `Base[]` (via the Palette workspace) immediately updates every sprite, tile, and UI element in the live preview that uses any sub-palette referencing that slot — within one frame, no manual refresh required.
- R6. The Palette workspace shows the 64-color base palette grid **plus** an 8-sub-palette overlay (each sub-palette displayed as 4 swatches with names). Designer edits which 4 base slots each sub-palette references by dragging swatches or picking via dropdown.

**Central quantizer + diff overlay**

- R7. The existing `QuantizeRGBA` (in `pixelforge_studio/palette/quantize.go`) is **wired into every image-import path** in the studio (PNG drag-drop, sprite-sheet imports, asset-browser additions). Audio imports are out of scope (idea #4).
- R8. When a non-quantized image is imported, the studio shows a **diff overlay modal**: ghosted before/after comparison of original RGBA vs quantized output, with three actions — **Accept** (commit), **Re-quantize** (open a sub-menu to pick a different sub-palette or quantizer preset and retry), **Reject** (cancel import).
- R9. If the designer hasn't pre-selected a target sub-palette, the quantizer **auto-picks** the best-matching sub-palette and labels it in the diff ("Quantized against sprite_0"). Designer can change via Re-quantize.
- R10. If mean RGB ΔE between original and quantized exceeds a threshold (planning resolves the exact value; "noticeable color shift" is the target feel), a **warning banner** appears above the diff: *"Significant color shift — consider a different sub-palette."* Designer can still Accept.

**Soft-warn constraint overlays**

- R11. The studio renders two NES-authenticity overlays as **toggleable visualizations** in the Scene workspace, Sprite editor, and Palette workspace:
  - **8-sprites-per-scanline**: highlights horizontal slices where >8 sprites overlap (red horizontal band)
  - **2×2 BG palette-block**: outlines 2×2 background-tile blocks where adjacent tiles use incompatible palette quadrants
- R12. Both overlays are **toggleable via the studio's View menu**, default **ON**. Settings persist per-project.
- R13. Overlays are **studio-side only** — they do NOT render in the shipped game runtime. The exported binary contains no overlay code.
- R14. Overlays **never gate save or export**. A project with constraint violations ships normally; violations are visible only inside the studio.
- R15. Each overlay surfaces a **tooltip on hover** explaining the constraint in plain language (one sentence; not jargon). Designer learns the constraint by reading the tooltip when they see the highlight.

---

## Acceptance Examples

- **AE1. Covers R1, R5.** Given a project with two sprites — Hero assigned to `sprite_0` (slots 5, 12, 18, 25) and Goomba assigned to `sprite_1` (slots 5, 30, 41, 55) — when the designer changes the color at slot 5 from brown to grey, both Hero and Goomba update in the live preview within one frame (because both reference slot 5 via their respective sub-palettes).
- **AE2. Covers R7, R8, R9.** Given the designer drags a 24-bit `hero.png` into the asset browser, when the import begins, a modal appears showing the original 24-bit image side-by-side with a quantized version (labeled "Quantized against sprite_0 — auto-picked"). Three buttons appear: Accept, Re-quantize, Reject.
- **AE3. Covers R8.** Given the diff modal is open, when the designer clicks Re-quantize, a sub-menu appears letting them pick a different target sub-palette and/or quantizer preset (`fast/balanced/quality`). On confirmation the quantizer runs again and the diff updates.
- **AE4. Covers R10.** Given the designer imports a high-color photograph whose mean RGB ΔE against the quantized output exceeds the threshold, when the diff appears, a warning banner reads "Significant color shift — consider a different sub-palette" above the before/after view. The Accept button remains enabled.
- **AE5. Covers R11, R12, R15.** Given the Scene workspace is open with sprite-scanline overlay ON, when the designer places a 9th entity on a horizontal slice already containing 8 entities, a red horizontal band appears across the affected scanline. Hovering shows: "NES would flicker — more than 8 sprites on this scanline."
- **AE6. Covers R12, R14.** Given the designer toggles both overlays OFF via the View menu, when they save the project and reopen later, both overlays are still OFF (settings persisted). Save and export work normally regardless of any constraint violations elsewhere.
- **AE7. Covers R13.** Given the designer ships a binary via idea #7's build pipeline, when a classmate runs the game, no scanline highlights, no palette-block outlines, no warning banners appear during gameplay. The runtime is unaware of the studio's overlays.

---

## Success Criteria

- **Identity outcome:** A first-time designer who has never used Pixelforge can drop a Photoshop mockup, see it become an NES-styled sprite via the quantizer's diff modal, accept it, and place it in a level — without learning what "indexed color" is.
- **Live-restyle outcome:** A designer mid-project changes the base color of slot 5 in the Palette workspace and sees their entire game's grey-toned enemies become brown in the preview, within the same second they made the change. Palette becomes felt as art-direction, not a constraint imposed afterwards.
- **Authenticity outcome:** A designer who paints a level violating the 2×2 palette-block rule sees the violation marked in red in the Scene preview, hovers to learn what's wrong, and decides whether the violation is intentional. They never have to read NESdev forums to know what they're doing.
- **Non-gating outcome:** A designer who explicitly wants to violate NES constraints (e.g., uses flicker as a technique, or wants 32-color sprites that exceed NES-strict 25) ships their game without the studio blocking them.
- **Downstream handoff outcome:** Planning consumes this doc and does not need to invent sub-palette structure, quantizer wiring shape, or overlay UX. Only implementation specifics (exact field names, exact ΔE threshold value, exact ImGui modal pattern) are open for planning.

---

## Scope Boundaries

- **Engine palette reduction to NES-strict 32 colors** — out (rejected via the palette-shape question). The 64-color base palette stays; sub-palettes layer on top.
- **Per-project mode toggle (strict vs extended)** — out (rejected via the palette-shape question). One shape: 64-color base + 8 sub-palettes.
- **Additional NES constraint overlays beyond scanline + 2×2 palette-block** — out. No attribute-table over-budget warning, no 256-tiles-per-CHR-bank warning, no sprite-0-hit visualization, no PPU timing visualization. Two constraints in v1; more can layer later.
- **Export-time constraint gating.** No "your game has palette violations, fix before shipping" block. The shipped game can have any violation the designer left in.
- **Runtime overlay visualization.** Overlays render only in the studio; never in the shipped binary.
- **Custom user-defined constraints.** The two soft-warn constraints are baked in for v1; designer cannot author new ones.
- **Audio quantizer.** Audio import handling is idea #4's brainstorm.
- **Aseprite import with native palette inheritance.** v1 quantizer operates on RGBA pixels. Aseprite's own indexed palette files could short-circuit quantization in a future asset-pipeline release.
- **AI-assisted palette suggestions** ("here's a palette that fits your art").
- **Palette swap animations** (authoring UI for time-varying color-table animations). The engine `ColorTables` system supports this; the studio surface for authoring is a separate concern.
- **Per-frame palette changes mid-scanline** (the NES sky-gradient technique). Defers to a much later release.
- **Per-sprite color override** (a sprite that uses colors outside its sub-palette). Sub-palette assignment is binding in v1; if the designer needs more colors per sprite, they reassign the sub-palette.

---

## Key Decisions

- **Sub-palette overlay on Base[64], not engine reduction.** Adds NES discipline without breaking existing 64-color fixtures (engine examples, test data, `editor.pforge`). Non-breaking is more important than maximal authenticity for v1; the soft-warn overlays from sub-deliverable (c) still surface when designers exceed NES-strict 25-color usage.
- **Quantizer is wired, not rewritten.** `QuantizeRGBA` already works. v1 only adds the studio's import-time invocation + the diff overlay UX. Reduces risk; preserves the existing engine-side palette-quantization work documented in `docs/solutions/palette-quantization-metric.md`.
- **Diff overlay over silent quantize.** Silent quantize would surprise designers ("why does my PNG look different?"). The diff modal teaches what the quantizer did and gives consent. Pairs with the audience constraint (designer not pre-trained on indexed-color workflows).
- **Auto-pick sub-palette unless designer pre-selects.** Reduces friction in the common case; designer can override via Re-quantize. Matches the "designer paints, system learns" pattern from idea #2's emergent rules.
- **Soft-warn, never gate.** Flicker and 2×2 block violations are historically used as deliberate techniques in NES games. Hard gates would force false authenticity; soft warnings inform the designer and let them choose.
- **Overlays studio-side only.** The shipped binary has no overlay code or runtime dependency on the studio. Keeps the cart shippable as a standalone artifact.
- **Two constraints in v1, more later.** Scanline + 2×2 block are the two most-distinctive NES visual signatures (per the NESdev research). Other constraints (attribute-table budgeting, CHR-bank limits) are real but less load-bearing for the visual identity goal.
- **Sub-palette assignment per-sprite + per-tile (per-2×2-block).** Matches NES hardware (sprites pick one sprite palette; BG attribute table is per-2×2-block). Designers don't need to know this mapping — the inspector just exposes "Palette" as a dropdown on sprites and a per-tile setting on the TileAtlas.

---

## Dependencies / Assumptions

- **Depends on idea #2's TileAtlas component** for per-tile sub-palette assignment (R3). If idea #2 doesn't land in the same release, idea #3's TileAtlas-side work degrades to scene-level `Tilemaps` assignment (less granular).
- **Depends on the existing `pixelforge_studio/palette/quantize.go`** continuing to work as it does today. v1 doesn't modify the algorithm; only wires it in.
- **Depends on the engine's indexed-palette rendering** — palette mutation already cascades to all rendered output because the engine renders against indices, not RGB. Live restyle (R5) leverages this; no engine change required.
- **Depends on the studio's reflection inspector** (U4 of the ImGui migration) for the per-sprite sub-palette dropdown in the SpriteAsset inspector.
- **Assumes ImGui's modal-popup pattern** can render a side-by-side image diff with three action buttons and an optional warning banner. Verified plausible from the cimgui-go bindings; planning confirms exact API.
- **Assumes overlay rendering** can be done as a translucent layer on top of the Scene workspace's existing scene-as-texture preview without altering the texture itself. Planning confirms ImGui DrawList supports this.

---

## Outstanding Questions

### Resolve Before Planning

- *(none — scope is fully resolved at the brainstorm level)*

### Deferred to Planning

- **[Affects R10] [Technical]** Exact mean RGB ΔE threshold above which the "significant color shift" warning fires. "Noticeable to the eye" is the target; planning picks a value through trial against a handful of sample imports.
- **[Affects R3] [Technical]** Exact schema field for per-tile vs per-2×2-block sub-palette assignment on `TileAtlas`. Candidates: per-tile field (most general but redundant for NES-strict), per-2×2-block (NES-authentic but adds a grouping concept), or per-tile with auto-derived-2×2-warning. Planning picks the cleanest fit.
- **[Affects R2] [Technical]** Whether `SpriteAsset.sub_palette` is a string field (`"sprite_0"`) or an integer index (0..3). Either round-trips the same observable behavior.
- **[Affects R1] [Technical]** How the 8 sub-palettes are stored on `PaletteData`. Candidates: 8 fixed-name fields (`bg_0`..`sprite_3`), one array `[]SubPalette` with name + 4 slot indices, or one map `map[string][4]int`. Planning picks the shape that round-trips through `omitempty` cleanly and reads natively in the reflection inspector.
- **[Affects R11] [Needs research]** Whether scanline-overlap detection in the Scene preview can be computed cheaply per-frame from the current entity layout (counting overlapping sprites by Y-coordinate banding) or whether it needs a more sophisticated PPU emulation. v1 wants the cheap version; planning verifies it covers the common case.
- **[Affects R7] [Needs research]** Whether the existing PNG-import path is already centralized in the studio (one entry point for all PNG additions) or scattered across multiple workspaces. If scattered, planning needs to define the import-pipeline seam first.
