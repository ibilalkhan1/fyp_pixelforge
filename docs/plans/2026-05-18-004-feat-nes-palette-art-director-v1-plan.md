---
title: "feat: NES Palette Art-Director — sub-palettes + quantizer wiring + soft-warn overlays (idea #3 v1)"
type: feat
status: active
date: 2026-05-18
depth: deep
origin: docs/brainstorms/2026-05-18-nes-palette-art-director-v1-requirements.md
ideation: docs/ideation/2026-05-18-pixelforge-nes-class-no-code-ideation.md (idea #3)
ships_with:
  - docs/plans/2026-05-18-002-feat-screenroom-mario-strip-v1-plan.md (idea #1 — Scene workspace overlays attach here)
  - docs/plans/2026-05-18-003-feat-tileatlas-emergent-rules-v1-plan.md (idea #2 — provides TileAtlas + reserved NESPaletteBlock field)
related_plans:
  - docs/plans/2026-05-18-001-feat-entity-verb-sheet-v1-plan.md (idea #5 — Archetype + pfcomponent pattern)
---

# feat: NES Palette Art-Director v1 (idea #3)

## Summary

v1 ships three coupled deliverables on top of the existing 64-color base palette: (1) **NES sub-palette overlay structure** — `PaletteData.BGSubPalettes [4]SubPalette` + `SpriteSubPalettes [4]SubPalette`, where each `SubPalette = {Name, Slots [4]int}` references 4 of the 64 base slots; every `SpriteAsset` carries `SubPalette string` and every TileAtlas (from idea #2) carries `NESPaletteBlock [][]int` per-2×2-block palette indices. (2) **Quantizer wiring + diff modal** — the existing `pixelforge_studio/palette/import_pipeline.go` (which currently has no production caller) becomes the canonical import seam, invoked from a new `File → Import PNG…` menu item; on import, a cimgui-go `BeginPopupModal` shows ghosted before/after with Accept / Re-quantize / Reject actions and a "significant color shift" warning banner (mean per-pixel RGB Euclidean distance > 10.0). (3) **Two soft-warn overlays** — 8-sprites-per-scanline (red horizontal band) + 2×2 BG palette-block consistency (outlined violations) — painted into the scene preview `ebiten.Image` inside `sceneGame.Draw` (per `docs/solutions/always-on-game-embedding.md`), so they're studio-only by construction and never reach the shipped binary. Toggleable via View menu, default ON, persisted per-project via `Project.EditorOverlays`. Zero external dependencies. Critical scope cuts from research: (a) Re-quantize sub-menu in v1 is **sub-palette-only** (the `fast/balanced/quality` presets in `docs/solutions/palette-quantization-metric.md` don't exist in code yet; implementing them is deferred to a quantizer-presets v2 unit); (b) per-tile palette override on TileAtlas is deferred; v1 stores per-2×2-block only via idea #2's `NESPaletteBlock` reserved field.

---

## Leverage Doctrine (applied)

Per `docs/plans/2026-05-18-001-feat-entity-verb-sheet-v1-plan.md`'s Leverage Doctrine appendix.

**Candidates evaluated:**

| Candidate | Status | Verdict |
|---|---|---|
| Color science libraries (`lucasb-eyer/go-colorful`, CIELAB ΔE2000) | Mature, MIT | **Skip** — `palette-quantization-metric.md` already commits to RGB ΔE + slot-dispersion blend. Introducing a second ΔE formula for the warning would disagree with the quantizer's own scoring (per learnings researcher recommendation). The mean-pixel RGB Euclidean distance for the warning is cheap (~5 LOC) and stays consistent. |
| Image quantization libraries (Go bindings to libimagequant, image/draw palette tools) | Available | **Skip** — existing `Quantizer` in `pixelforge_studio/palette/quantize.go` is sub-palette-aware (binds to project palette). Wrapping a generic library would require re-deriving sub-palette constraints. |
| ImGui modal/popup libraries for cimgui-go | None mature | **Build native** — `imgui.BeginPopupModal` is the canonical primitive; ~80 LOC for the diff modal. |
| Image diff / visual comparison libraries | None worth depending on for a 2-button modal | **Build native** — side-by-side `imgui.Image` blits of original RGBA + quantized output (quantized output reconstructed by mapping indices through `Palette.Base`). |

Total custom: ~300 LOC across diff modal, two overlays, sub-palette schema + helpers, auto-pick algorithm, View-menu toggles, palette workspace. Well below wrap costs.

---

## Problem Frame

Pixelforge has the bones of NES authenticity but no studio surface that engages them:

- **No sub-palette concept.** `PaletteData.Base [64]string` (palette.go:9-27) is freely editable to 64 arbitrary RGBs. Designers can use any color anywhere — no constraint, no art-direction. The NES's signature look comes from its **layered** palette: 64 hardware colors, picked 4 at a time into 8 sub-palettes (4 BG + 4 sprite), with sprites + tiles each binding to one sub-palette. Without that layer, palette changes either touch one slot at a time (tedious) or all 64 at once (chaos) — neither makes palette feel like art-direction.
- **Quantizer is unused.** `Quantizer.QuantizeImage(rgba, w, h) []byte` works correctly (tests in `quantize_test.go`), but the only callers are its own tests. `palette.Import(pngPath, *Project, ...)` (the existing pipeline at `import_pipeline.go:222`) is the natural seam, but `asset_browser.go` doesn't invoke it; `file_menu.go` has no Import entry. **A designer dragging a Photoshop mockup into Pixelforge gets nothing — the import doesn't even fire.** Even if it did, they'd get silent quantization (no preview, no consent) and wouldn't learn what the quantizer did to their art.
- **No NES authenticity feedback.** A designer who paints adjacent tiles using incompatible palette quadrants has produced a level that wouldn't render correctly on a real NES, but the studio never flags it. A designer who places 9 entities on one horizontal slice would cause sprite flicker on real hardware; the studio is silent. Pico-8's reputation comes from constraints visible everywhere; Pixelforge has the engine bones for the same discipline but no studio surface engaged with them.

The brainstorm's bet: a designer drags `hero.png` (24-bit Photoshop output) into the studio. A modal pops with `hero.png` on the left, an NES-quantized version on the right, labeled "Quantized against sprite_0 — auto-picked". They click Accept. Place the sprite. Open the Palette workspace, change the brown at slot 5 to grey, watch every enemy that uses any sub-palette referencing slot 5 update in the preview within one frame. Place 9 entities on one row; a red band appears with a tooltip explaining NES would flicker here. They've learned a hardware constraint by seeing it, not by reading NESdev forums.

---

## Carried Forward from Origin

All 15 requirements, 7 acceptance examples, 4 flows, 3 actors from origin are in scope.

| Origin | Scope summary | Plan unit(s) |
|---|---|---|
| R1, R6 | Sub-palette overlay structure + Palette workspace shows base + sub-palettes | U1, U7 |
| R2 | SpriteAsset.SubPalette field (string) | U1, U4 |
| R3 | TileAtlas per-2×2-block palette assignment via `NESPaletteBlock` | U1 (depends on idea #2), U4 |
| R4 | All schema additions `omitempty`; legacy projects load with defaults | U1, U2 |
| R5 | Live restyle in preview within one frame | Transitive — engine already renders against indices; idea #1's `pixelforge_entity` renderer is the studio's preview surface that observes the cascade |
| R7, R8, R9 | Quantizer wired into PNG import; diff modal with Accept/Re-quantize/Reject; auto-pick best sub-palette | U3, U4 |
| R10 | ΔE warning banner — threshold 10.0 mean per-pixel Euclidean RGB (planning resolution) | U4 |
| R11, R12 | Two overlays toggleable via View menu, default ON, persisted per-project | U6, U7, U8 |
| R13 | Overlays studio-only — no runtime code | U6, U7 — by construction, they paint into scene preview `ebiten.Image` only inside `sceneGame.Draw` |
| R14 | Overlays never gate save/export | All units — no enforcement code anywhere |
| R15 | Tooltip on hover explaining each constraint | U6, U7 |
| AE1-AE7, F1-F4 | All seven acceptance examples + four flows | U9 (integration tests) |
| A1-A3 | Designer, Studio, existing palette pipeline | All units |

Origin's "Deferred to Planning" section: all 6 technical questions resolved in Phase 2 (see Key Technical Decisions). One discovered planning issue from research: `palette-quantization-metric.md` is ahead of the code (presets don't exist) — resolved by scoping Re-quantize sub-menu to sub-palette-only in v1, with preset work explicitly deferred.

---

## High-Level Technical Design

How the three deliverables compose:

```
              SCHEMA (per-Project, .pforge)                              U1
              ──────────────────────────────────────────
   ┌──────────────────────────────────────────────────────┐
   │ PaletteData                                          │
   │ ─ Base [64]string (existing)                         │
   │ ─ ColorTables [4][64][64]uint8 (existing)            │
   │ ─ BGSubPalettes     [4]SubPalette   (NEW omitempty)  │
   │ ─ SpriteSubPalettes [4]SubPalette   (NEW omitempty)  │
   │                                                      │
   │ SubPalette { Name string; Slots [4]int }             │
   │                                                      │
   │ SpriteAsset                                          │
   │ ─ ... (existing fields)                              │
   │ ─ SubPalette string (NEW omitempty; "sprite_0" def)  │
   │                                                      │
   │ TileAtlas (from idea #2)                             │
   │ ─ NESPaletteBlock [][]int (RESERVED in idea #2;      │
   │   POPULATED here as per-2×2-block palette index)     │
   │                                                      │
   │ Project.EditorOverlays {                              │
   │   ScanlineEnabled bool                               │
   │   PaletteBlockEnabled bool                           │
   │ } (NEW omitempty; defaults: both true)               │
   └────────────┬─────────────────────────────────────────┘
                │
   ┌────────────┴───────────────────────────────────────┐
   ▼                                                     ▼
┌──────────────────────────┐                  ┌──────────────────────────┐
│ IMPORT PIPELINE (U3)     │                  │ INSPECTORS (U4)          │
│                          │                  │                          │
│ File → Import PNG…  ──┐  │                  │ SpriteAsset:             │
│ (NEW menu item)       │  │                  │  SubPalette dropdown via │
│                       ▼  │                  │  new WidgetSubPalette    │
│ palette.Import(...)      │                  │  kind (mirrors           │
│ (EXISTING — already      │                  │  WidgetSpriteRef         │
│  decodes PNG via         │                  │  precedent)              │
│  png.Decode + calls      │                  │                          │
│  Quantizer; just no UI   │                  │ TileAtlas:               │
│  caller today)           │                  │  Per-2×2-block palette   │
│                          │                  │  picker via tilepainter  │
│                          │                  │  widget extension        │
│ Auto-pick sub-palette ───│  (U2)            │  (extends idea #2 U5)    │
│                          │                  │                          │
└────────┬─────────────────┘                  └──────────────────────────┘
         │ raises diff
         ▼
┌──────────────────────────┐
│ DIFF MODAL (U4)          │
│ cimgui-go                │
│ BeginPopupModal          │
│                          │
│ ┌────────┐  ┌────────┐   │
│ │Original│  │Quantizd│   │
│ │  RGBA  │  │ output │   │
│ └────────┘  └────────┘   │
│                          │
│ Auto-picked: sprite_0    │
│ [⚠ Significant shift]    │
│                          │
│ Accept  Re-quantize ↗    │  (Re-quantize opens
│ Reject                   │   second-level modal:
│                          │   sub-palette picker
│                          │   only in v1; no preset
│                          │   picker)
└──────────────────────────┘

              PALETTE WORKSPACE (U7)
              ────────────────────────────────────
   ┌─────────────────────────────────────────────┐
   │ ┌─────────────────────┐  ┌─────────────┐    │
   │ │  Base [64] grid     │  │ Sub-palette │    │
   │ │  (8×8 swatches)     │  │ overlay     │    │
   │ │  Click → edit color │  │ bg_0  ▣▣▣▣  │    │
   │ │                     │  │ bg_1  ▣▣▣▣  │    │
   │ │                     │  │ bg_2  ▣▣▣▣  │    │
   │ │                     │  │ bg_3  ▣▣▣▣  │    │
   │ │                     │  │ sprt_0▣▣▣▣  │    │
   │ │                     │  │ sprt_1▣▣▣▣  │    │
   │ │                     │  │ sprt_2▣▣▣▣  │    │
   │ │                     │  │ sprt_3▣▣▣▣  │    │
   │ │                     │  │ Drag swatch │    │
   │ │                     │  │ to reassign │    │
   │ └─────────────────────┘  └─────────────┘    │
   └─────────────────────────────────────────────┘

              SCENE PREVIEW + OVERLAYS                          U6, U7
              ──────────────────────────────────────
   ┌─────────────────────────────────────────────┐
   │ sceneGame.Draw(dst ebiten.Image)            │
   │  1. tilemap.Render (from idea #1)           │
   │  2. entity.RenderAll (from idea #1)         │
   │  3. ───────────────────────────────────     │
   │  4. if overlays.ScanlineEnabled:            │
   │       paint red horizontal bands            │  U6
   │       on dst where ≥9 entities overlap Y    │
   │  5. if overlays.PaletteBlockEnabled:        │
   │       paint outlines on dst for 2×2 blocks  │  U7
   │       with mixed palette assignments        │
   │                                             │
   │ Output: ebiten.Image painted only here →    │
   │ scene texture exposed to cimgui-go via      │
   │ imgui.ImageWithBgV (existing).              │
   │                                             │
   │ ► Studio-only by construction (R13):        │
   │   shipped runtime uses pixelforge_ebiten    │
   │   directly, never goes through sceneGame.   │
   └─────────────────────────────────────────────┘
```

*This illustrates the intended approach and is directional guidance for review, not implementation specification.*

The structural insight: **overlays painted into the scene preview `ebiten.Image` satisfy R13 (no runtime overlay code) automatically** — they're inside the studio-only `sceneGame.Draw` path, not in `pixelforge_ebiten`. The shipped binary never calls this code path. No conditional flags, no opt-out config — the boundary is structural.

---

## Output Structure

```
pixelforge_project/
├── palette.go                                  (MODIFY, U1) — BGSubPalettes + SpriteSubPalettes + SubPalette type
├── palette_defaults.go                         (NEW, U1)   — applyDefaults for PaletteData
├── palette_test.go                             (MODIFY, U1)
├── palette_defaults_test.go                    (NEW, U1)
├── sprites.go                                  (MODIFY, U1) — SubPalette string field on SpriteAsset
├── sprites_test.go                             (MODIFY)
├── scenes.go                                   (MODIFY via idea #2)  — TileAtlas.NESPaletteBlock populated here per-2×2-block
└── editor_overlays.go                          (NEW, U1)   — Project.EditorOverlays struct + applyDefaults

pixelforge_studio/palette/
├── subpalette_pick.go                          (NEW, U2)   — auto-pick best-fit sub-palette for image
├── subpalette_pick_test.go                     (NEW, U2)
├── quantize.go                                 (no change in v1) — wiring only
├── import_pipeline.go                          (MODIFY, U3) — accept target sub-palette param; return diff data
└── import_pipeline_test.go                     (MODIFY)

pfcomponent/
└── metadata.go                                 (MODIFY, U4) — add WidgetSubPalette kind

pixelforge_studio/editor/
├── file_menu.go                                (MODIFY, U3) — add File → Import PNG…; View menu overlay toggles
├── import_diff_modal.go                        (NEW, U3)   — cimgui-go BeginPopupModal with diff + 3 actions
├── import_diff_modal_test.go                   (NEW, U3)
├── widgets/context.go                          (MODIFY, U4) — add SubPaletteNames []string
├── inspector.go                                (MODIFY, U4) — WidgetSubPalette dispatch arm
├── tilepainter_widget.go                       (MODIFY via idea #2 U5) — per-2×2-block palette picker added
├── palette_workspace.go                        (NEW, U5)   — 64-slot grid + sub-palette overlay editor
├── palette_workspace_test.go                   (NEW, U5)
├── scene_overlay_scanline.go                   (NEW, U6)   — Y-band entity-overlap detection + paint
├── scene_overlay_scanline_test.go              (NEW, U6)
├── scene_overlay_paletteblock.go               (NEW, U7)   — 2×2 block consistency check + paint
├── scene_overlay_paletteblock_test.go          (NEW, U7)
├── canvas_input.go                             (MODIFY, U6, U7) — sceneGame.Draw calls overlays after entity render
└── editor_overlays_settings.go                 (NEW, U8)   — View menu accessor + per-project persistence

pixelforge_studio/integration_test/
├── nes_palette_e2e_test.go                     (NEW, U9)
└── fixtures/
    ├── hero_24bit.png                          (NEW, U9)   — synthetic 24-bit input for quantizer diff
    ├── photograph_high_chroma.png              (NEW, U9)   — high-ΔE source for warning-banner test
    ├── nes_palette_full_project.pforge         (NEW, U9)   — project with 8 sub-palettes populated
    └── scanline_violation_scene.pforge         (NEW, U9)   — scene with 9 entities on one row
```

Implementer may consolidate or split files; per-unit `**Files:**` sections remain authoritative.

---

## Implementation Units

Dependency-ordered. Foundation (U1-U2) → import flow (U3) → inspectors (U4) → workspace + overlays (U5-U7) → settings + tests (U8-U9).

### U1. Schema additions — sub-palettes + sprite assignment + per-project overlay toggles

**Goal:** Add `PaletteData.BGSubPalettes [4]SubPalette` + `SpriteSubPalettes [4]SubPalette` + `SubPalette {Name, Slots [4]int}`. Add `SpriteAsset.SubPalette string`. Add `Project.EditorOverlays {ScanlineEnabled, PaletteBlockEnabled bool}`. Populate idea #2's `TileAtlas.NESPaletteBlock [][]int` as per-2×2-block palette indices. All `omitempty` with sensible defaults applied on load.

**Requirements:** R1, R2, R3, R4 (schema additions + load-tolerant defaults), R12 (per-project persistence of overlay toggles).

**Dependencies:** none for the PaletteData / SpriteAsset / EditorOverlays parts. The TileAtlas-NESPaletteBlock-population part **depends on idea #2's plan U1** landing first.

**Files:**
- `pixelforge_project/palette.go` (MODIFY — add SubPalette type, BGSubPalettes/SpriteSubPalettes fields)
- `pixelforge_project/palette_defaults.go` (NEW — first-time `applyDefaults` for PaletteData)
- `pixelforge_project/palette_test.go` (MODIFY)
- `pixelforge_project/palette_defaults_test.go` (NEW)
- `pixelforge_project/sprites.go` (MODIFY — add `SubPalette string \`json:"sub_palette,omitempty"\``)
- `pixelforge_project/sprites_test.go` (MODIFY)
- `pixelforge_project/editor_overlays.go` (NEW — `EditorOverlays` struct definition)
- `pixelforge_project/project.go` (MODIFY — call PaletteData.applyDefaults from Project.applyDefaults; add EditorOverlays field with defaults)
- `pixelforge_project/scenes.go` (MODIFY via idea #2's plan U1 — semantics of NESPaletteBlock confirmed as per-2×2-block palette index; populated by U4 and U7 of this plan, not by U1)

**Approach:**
- `SubPalette` struct: `Name string \`json:"name"\`; Slots [4]int \`json:"slots"\``. Slots are indices into `PaletteData.Base[0..63]`.
- `PaletteData.BGSubPalettes [4]SubPalette \`json:"bg_sub_palettes,omitempty"\`` and `SpriteSubPalettes [4]SubPalette \`json:"sprite_sub_palettes,omitempty"\``. Fixed-size arrays (NES has exactly 4 of each). JSON-marshals only when populated; legacy empty `"palette": {}` still loads.
- Default sub-palettes (applied on load when arrays are zero):
  - BG: `bg_0` = {1,2,3,4}; `bg_1` = {5,6,7,8}; `bg_2` = {9,10,11,12}; `bg_3` = {13,14,15,16}.
  - Sprite: `sprite_0` = {17,18,19,20}; `sprite_1` = {21,22,23,24}; `sprite_2` = {25,26,27,28}; `sprite_3` = {29,30,31,32}.
  - (Slot 0 is reserved transparent per `palette.go:89-110`; defaults pull from 1..32 to leave room.)
- `SpriteAsset.SubPalette string \`json:"sub_palette,omitempty"\``: empty string at load defaults to `"sprite_0"` (applied by `applyDefaults`).
- `Project.EditorOverlays struct { ScanlineEnabled bool; PaletteBlockEnabled bool } \`json:"editor_overlays,omitempty"\``: zero struct defaults to both `true` on load (per R12 "default ON").
- `PaletteData.applyDefaults()`: a new function; called from `Project.applyDefaults` (project.go:107-113); backfills sub-palette arrays + sanitizes slot indices (clamps to 0..63; per `editor-pforge-schema-shape.md`'s sanitize discipline — repair, never reject).
- TileAtlas's `NESPaletteBlock [][]int` (reserved in idea #2's plan) is now semantically defined: `NESPaletteBlock[blockRow][blockCol] = subPaletteIndex` where `blockRow = tileRow/2`, `blockCol = tileCol/2`, and `subPaletteIndex` is 0..3 referencing the project's `BGSubPalettes[i]`. This plan claims the field's semantics; idea #2's plan only reserved the field.

**Patterns to follow:** existing additive-omitempty pattern (e.g., `Theme.SanitizeSlots` in `theme.go` called from `Project.applyDefaults` per project.go:107-113); existing `[]EventSubscription` slice with nil backfill in `normalizeSlices` (project.go); existing JSON-tag conventions on `PaletteData`.

**Test scenarios:**
- `TestPaletteData_DefaultsBackfillSubPalettes`: load JSON with `"palette": {}`; after applyDefaults, BGSubPalettes[0].Name == "bg_0", Slots == [1,2,3,4]; same shape for all 8.
- `TestPaletteData_RoundTripWithCustomSubPalettes`: marshal palette with non-default sub-palette assignments; unmarshal; values preserved.
- `TestPaletteData_OmitEmptyWhenZero`: marshal default palette; output JSON does NOT contain `bg_sub_palettes` or `sprite_sub_palettes` keys.
- `TestPaletteData_SanitizeClampsOutOfRangeSlots`: load JSON with `"slots": [70, -3, 64, 100]` (all out of range); after sanitize, all values clamp to 0..63 (e.g., 63, 0, 63, 63).
- `TestPaletteData_LegacyEmptyPaletteRoundTrip`: load `editor.pforge`'s `"palette": {}`; defaults backfill; re-marshal does not introduce keys (because defaults equal the omitempty-zero state for serialization).
- `TestSpriteAsset_SubPaletteOmitEmpty`: marshal sprite without SubPalette; no `sub_palette` key in JSON.
- `TestSpriteAsset_SubPaletteDefaultsToSprite0`: load sprite with no `sub_palette`; after applyDefaults, value is `"sprite_0"`.
- `TestSpriteAsset_SubPaletteRoundTrip`: sprite with `SubPalette: "sprite_2"`; round-trips.
- `TestProjectEditorOverlays_DefaultsToBothEnabled`: load project without `editor_overlays`; after defaults, both flags are `true`.
- `TestProjectEditorOverlays_RoundTrip`: project with `ScanlineEnabled: false, PaletteBlockEnabled: true`; round-trips correctly.
- `TestProjectEditorOverlays_OmitEmptyWhenBothDefault`: NOT — since defaults are `true`, a struct that defaults to both true would be omitted... wait, bool `true` is not the zero value; so the struct WILL serialize. Decision: use `*bool` pointers OR a default-true semantics in applyDefaults. **Implementer decision:** use struct with bool fields; serialize always when at least one is non-default; document that a struct of `{false, false}` is "intent to disable both" and round-trips faithfully.
- Covers AE6 (overlay toggle persistence).

**Verification:** `go test ./pixelforge_project/...` passes; round-trip of `editor.pforge` produces zero key churn; the only existing fixture loads with all 8 sub-palettes auto-populated.

---

### U2. Auto-pick best-fit sub-palette algorithm

**Goal:** Given an `image.Image` and a `*Project`, return the sub-palette (BG or Sprite) whose 4 slots best fit the image's color content. Used by U3's import pipeline when the designer hasn't pre-selected a target.

**Requirements:** R9 (auto-pick when no pre-selection; labels in diff modal).

**Dependencies:** U1 (sub-palettes exist on PaletteData).

**Files:**
- `pixelforge_studio/palette/subpalette_pick.go` (NEW)
- `pixelforge_studio/palette/subpalette_pick_test.go` (NEW)

**Approach:**
- `PickBestSubPalette(img image.Image, p *pixelforge_project.Project, family SubPaletteFamily) (subPaletteName string, score float64)` where `family` is BG or Sprite (caller decides based on import context — sprites import as Sprite, tiles import as BG).
- Algorithm: for each candidate sub-palette in the family (4 candidates):
  1. Convert each of the sub-palette's 4 slot colors from `PaletteData.Base[slot]` (hex string) to RGB.
  2. For each pixel in the image (or a subsample for large images), compute Euclidean distance to the nearest of the 4 sub-palette colors.
  3. Sum per-pixel distances; lower sum = better fit.
- Return the sub-palette with the lowest total distance. Score is the mean per-pixel distance (used by U4's warning banner threshold).
- Subsample for performance: if image > 64×64, sample on a regular 16×16 grid (~256 samples). Standard image-color-analysis trick.
- Default sub-palette family when caller doesn't specify: Sprite (since most imports are sprites).

**Patterns to follow:** existing `Quantizer.QuantizePixel` (quantize.go) for the nearest-color-in-palette pattern; existing test idiom in `quantize_test.go` (use `pixelforge_project.NewProject("t")` + mutate `Palette.Base` slots).

**Test scenarios:**
- `TestPickBestSubPalette_RedImagePicksRedSubPalette`: project with 4 sprite sub-palettes (one mostly-red, others mostly-blue/green/grey); load a solid-red 16×16 image; PickBestSubPalette returns the red sub-palette.
- `TestPickBestSubPalette_TiedSubPalettesReturnsFirst`: image whose colors equidistantly fit two sub-palettes; returns the lower-indexed one (deterministic).
- `TestPickBestSubPalette_GreyscalePicksGreyscalePalette`: greyscale image vs project with one greyscale sub-palette and three vibrant ones; picks greyscale.
- `TestPickBestSubPalette_EmptyImagePicksFirst`: 1×1 image with one pixel; returns the first sub-palette in the family (or matches by closest distance).
- `TestPickBestSubPalette_SubsampleForLargeImage`: 1024×1024 image; algorithm completes in <50ms (perf gate); result is deterministic across runs.
- `TestPickBestSubPalette_BGvsSpriteFamily`: same image; calling with `FamilyBG` vs `FamilySprite` returns from the respective sub-palette set.
- `TestPickBestSubPalette_NoSubPalettesDefined`: project with all-default sub-palettes; returns `"sprite_0"` (or `"bg_0"` depending on family) as graceful fallback.
- `TestPickBestSubPalette_ScoreIsMeanPerPixelDistance`: image with known distances; assert returned `score` matches expected mean.

**Verification:** `go test ./pixelforge_studio/palette/...` passes; perf benchmark for 1024×1024 image completes well under threshold.

---

### U3. Wire palette.Import into File menu + drag-drop

**Goal:** Add a `File → Import PNG…` menu item that opens a file picker, invokes `palette.Import`, and triggers U4's diff modal with the import result. Optionally add drag-drop on the asset browser. Confirm `palette.Import` is the single canonical seam (research finding: no production caller exists today).

**Requirements:** R7 (quantizer wired into every image-import path).

**Dependencies:** U2 (auto-pick available for the import to call when designer hasn't pre-selected).

**Files:**
- `pixelforge_studio/editor/file_menu.go` (MODIFY — add Import PNG menu entry)
- `pixelforge_studio/palette/import_pipeline.go` (MODIFY — extend signature to accept optional target sub-palette + return diff data: original image + quantized indices + score + chosen sub-palette name)
- `pixelforge_studio/palette/import_pipeline_test.go` (MODIFY)
- `pixelforge_studio/editor/asset_browser.go` (OPTIONAL MODIFY — drag-drop handler invokes the import; deferable if first ship cuts it)
- `pixelforge_studio/editor/import_handler.go` (NEW — orchestrates file-picker → import → diff modal → final commit)
- `pixelforge_studio/editor/import_handler_test.go` (NEW)

**Approach:**
- New entry in `Editor.buildMenuDefs()` (file_menu.go) under File menu: `Import PNG…` → opens file picker → on confirm, hands path to `import_handler`.
- `palette.Import(pngPath, *Project, projectSourcePath)` extends:
  - New optional parameter `targetSubPalette string` (empty = use auto-pick from U2).
  - New return type: `ImportResult { OriginalImage image.Image; QuantizedIndices []byte; QuantizedReconstructed image.Image; ChosenSubPalette string; MeanDeltaE float64; Width, Height int }`.
  - `QuantizedReconstructed` is the visual preview: walk `QuantizedIndices`, look up each via `Palette.Base[index]`, write to a fresh `*image.RGBA`. This is what the diff modal displays on the right side.
- `import_handler` orchestrates: receive path → call `palette.Import` → if successful, queue diff modal (via U4); on Accept, commit to project; on Re-quantize, re-invoke palette.Import with a different sub-palette; on Reject, drop.
- Drag-drop on asset browser (optional for v1): cimgui-go exposes drop events via Ebitengine's `DroppedFiles()` per-frame. Plumb through `AssetBrowser.Update` to call the same `import_handler` with each dropped file path.
- Per `docs/solutions/file-picker-design.md`: drive imperatively (`Open`, `Confirm`, `Cancel`); diff modal stacks correctly on top of file picker because both register with FocusManager / ModalStack.

**Patterns to follow:** existing `palette.Import` flow at `import_pipeline.go:222`; existing `file_menu.go:165-204` menu structure; existing `widgets.MenuItem` + `OnSelect` closure pattern; existing file picker pattern at `pixelforge_studio/editor/widgets/file_picker.go`.

**Test scenarios:**
- `TestImportPNG_MenuItemPresentInFileMenu`: after editor construction, menu definition includes `File > Import PNG…`.
- `TestImport_ReturnsImportResultWithAllFields`: import a synthetic 16×16 PNG; ImportResult contains OriginalImage, QuantizedIndices (256 bytes), QuantizedReconstructed (16×16 RGBA), ChosenSubPalette ("sprite_0"), MeanDeltaE (a float).
- `TestImport_TargetSubPaletteOverridesAutoPick`: call import with explicit `targetSubPalette: "sprite_2"`; ChosenSubPalette in result is `"sprite_2"`, regardless of what auto-pick would have chosen.
- `TestImport_AutoPickFiresWhenNoTarget`: call import with empty target; ChosenSubPalette matches what `PickBestSubPalette(family=Sprite)` returns.
- `TestImport_QuantizedReconstructedMatchesIndicesViaPalette`: ImportResult.QuantizedReconstructed at pixel (x,y) equals `Palette.Base[QuantizedIndices[y*W+x]]` parsed to RGB.
- `TestImport_HighDeltaEReportedInResult`: import a high-chroma photograph against a limited sub-palette; MeanDeltaE > 10.0 (the warning threshold from U4); the import still succeeds, the threshold is the modal's UX concern.
- `TestImportHandler_AcceptCommitsToProject`: simulate import + Accept; `Project.Sprites` gains a new SpriteAsset with the imported indices + ChosenSubPalette set.
- `TestImportHandler_RejectDropsImport`: simulate import + Reject; `Project.Sprites` unchanged; no asset added.
- `TestImportHandler_RequantizeUpdatesDiffResult`: simulate import → Re-quantize with different target → second diff modal opens with new ChosenSubPalette + recomputed QuantizedReconstructed.
- `TestImport_PNGFromFilePicker`: integration — open file picker programmatically, confirm a fixture path, import completes, diff modal queued.
- Covers F2 (PNG drag-drop or import → diff appears).

**Verification:** `go test ./pixelforge_studio/...` passes; manual smoke: launch studio, `File > Import PNG…`, pick a test PNG, see diff modal.

---

### U4. Diff modal + inspector sub-palette dropdown + per-tile palette picker

**Goal:** Implement the cimgui-go `BeginPopupModal` diff overlay (Original / Quantized side-by-side, three actions, optional warning banner, Re-quantize sub-modal). Add `pfcomponent.WidgetSubPalette` kind and dispatch arm for the SpriteAsset SubPalette dropdown. Extend idea #2's tilepainter widget (`tilepainter_widget.go`) with a per-2×2-block palette picker that mutates `TileAtlas.NESPaletteBlock`.

**Requirements:** R2 (sprite dropdown), R3 (tile-block picker), R7 (diff modal), R8 (three actions), R9 (auto-pick label), R10 (warning banner at ΔE > 10.0), R15 (tooltip explaining constraint — applies to overlay too but inspector dropdown gets the same tooltip discipline for "what's this?").

**Dependencies:** U1 (schema), U2 (auto-pick result feeds modal label), U3 (import pipeline returns ImportResult feeding modal), idea #2's plan U5 (tilepainter widget exists to extend).

**Files:**
- `pixelforge_studio/editor/import_diff_modal.go` (NEW)
- `pixelforge_studio/editor/import_diff_modal_test.go` (NEW)
- `pfcomponent/metadata.go` (MODIFY — add `WidgetSubPalette WidgetKind` constant; extend `applyPFTag` to recognize `pf:"subpalette,family=bg|sprite"`)
- `pixelforge_studio/editor/widgets/context.go` (MODIFY — add `BGSubPaletteNames []string`, `SpriteSubPaletteNames []string`; populated by Editor on each render)
- `pixelforge_studio/editor/inspector.go` (MODIFY — `case WidgetSubPalette:` arm that routes via `comboField` mirroring WidgetSpriteRef at inspector.go:183-190)
- `pixelforge_studio/editor/tilepainter_widget.go` (MODIFY — add per-2×2-block palette picker; existing widget from idea #2)
- `pixelforge_project/sprites.go` (MODIFY — add `pf:"subpalette,family=sprite"` tag on SpriteAsset.SubPalette)

**Approach:**
- **Diff modal**:
  - cimgui-go `imgui.BeginPopupModal("Import Quantization", &open, ImGuiWindowFlags_AlwaysAutoResize)`.
  - Two `imgui.Image` calls side-by-side, fed by ebitengine textures created from `ImportResult.OriginalImage` and `ImportResult.QuantizedReconstructed`. Reuse the existing scene-as-texture pattern (`backend.CreateTextureFromGame` per `canvas_input.go`).
  - Label: `"Quantized against " + ChosenSubPalette + " (auto-picked)"` or `"(manually selected)"` based on whether designer chose.
  - **Warning banner**: if `ImportResult.MeanDeltaE > 10.0`, render a yellow-tinted bar with text `"⚠ Significant color shift — consider a different sub-palette."` above the two images.
  - Three buttons: `Accept`, `Re-quantize`, `Reject`.
  - `Accept` → invokes `import_handler.Commit(result)` → adds to `Project.Sprites`, marks dirty per `dirty-state-ux.md`.
  - `Re-quantize` → opens **second-level modal** with sub-palette picker (dropdown of all 4 Sprite sub-palette names; no preset picker in v1). On confirm, calls `palette.Import` again with the new target and replaces the diff result inline.
  - `Reject` → cancels import; modal closes; no project mutation.
  - Modal registers with FocusManager / ModalStack per `focus-manager-design.md` so Esc dismisses (= Reject); Re-quantize sub-modal is second-level (Esc pops it but leaves diff modal open).
- **ΔE warning threshold = 10.0** (per planning decision). Tunable constant in the modal; designer-tunable in v2 if needed.
- **WidgetSubPalette dispatch**:
  - Tag syntax: `pf:"subpalette,family=sprite"` or `pf:"subpalette,family=bg"`. Family token determines which list of sub-palette names the inspector pulls from.
  - `applyPFTag` extension: recognize first token `subpalette`; parse `family=X` from remainder; stash family on `Field.Options` (reuse existing `Options []string` field by convention).
  - `inspector.renderField` arm: switch on `f.Family` (or first Option entry), call `comboField(values, key, label, ctx.BGSubPaletteNames | ctx.SpriteSubPaletteNames)`.
  - `widgets.Context` extension: Editor populates `ctx.SpriteSubPaletteNames = [p.Palette.SpriteSubPalettes[i].Name for i in 0..3]` (and similarly for BG) at inspector render time.
- **Per-2×2-block palette picker on TileAtlas widget**:
  - The tilepainter widget (idea #2's U5) gains a new mode: a tool-mode selector entry "Palette Block" that, when active, lets designer click into any 2×2 block in the tilemap and pick which BG sub-palette assigns to it.
  - State stored in `TileAtlas.NESPaletteBlock[blockRow][blockCol] = subPaletteIndex` (0..3).
  - Visual: when picker mode active, the canvas overlays existing 2×2 blocks with their current assignment as a colored tint.
  - On block click, opens a small popup with 4 swatches (4 BG sub-palette names + their slot colors); designer picks; popup closes; assignment persists; MarkDirty.

**Patterns to follow:** existing `WidgetSpriteRef` dispatch at `inspector.go:183-190` for the sub-palette dropdown; existing cimgui-go `imgui.ImageWithBgV` pattern for sprite textures (canvas_input.go); existing scene-as-texture creation via `backend.CreateTextureFromGame`; `focus-manager-design.md` modal stack pattern; `dirty-state-ux.md` for MarkDirty discipline; idea #2's plan U5 tilepainter_widget.go for the canvas-state pub/sub pattern.

**Test scenarios:**
- **Diff modal**:
  - `TestDiffModal_RendersOriginalAndQuantized`: open modal with synthetic ImportResult; two ebitengine textures created; modal flagged "open."
  - `TestDiffModal_WarningBannerShowsAboveThreshold`: ImportResult.MeanDeltaE = 15.0; banner text present in modal output.
  - `TestDiffModal_NoBannerBelowThreshold`: MeanDeltaE = 5.0; no banner text.
  - `TestDiffModal_AcceptCallsHandler`: simulate Accept click; commit handler invoked once with the ImportResult.
  - `TestDiffModal_RejectCallsCancel`: simulate Reject; cancel handler invoked; no commit.
  - `TestDiffModal_RequantizeOpensSecondLevelModal`: simulate Re-quantize; sub-modal flagged open; Esc pops sub-modal, diff modal remains open.
  - `TestDiffModal_RequantizeWithDifferentSubPalette`: in sub-modal pick `"sprite_2"`; `palette.Import` re-invoked with target=`"sprite_2"`; diff result updates; ChosenSubPalette in label now reads `"sprite_2 (manually selected)"`.
  - `TestDiffModal_EscDismisses`: open modal; press Esc; modal closes via FocusManager precedence; equivalent to Reject.
  - `TestDiffModal_ChosenSubPaletteLabelReflectsManualVsAuto`: when imported via auto-pick, label says `"(auto-picked)"`; when imported via Re-quantize manual choice, label says `"(manually selected)"`.
- **WidgetSubPalette dispatch**:
  - `TestApplyPFTag_SubPaletteFamily`: field with `pf:"subpalette,family=sprite"` resolves to WidgetKind=WidgetSubPalette, Family/Options carries "sprite".
  - `TestInspector_SubPaletteDropdownRendersSpriteNames`: SpriteAsset with SubPalette field; inspector renders combo with 4 entries (sprite_0..sprite_3) sourced from `ctx.SpriteSubPaletteNames`.
  - `TestInspector_SubPaletteSelectionUpdatesAsset`: click "sprite_2" in dropdown; `SpriteAsset.SubPalette` becomes "sprite_2"; MarkDirty called.
  - `TestInspector_BGSubPaletteFamilyShowsBGNames`: field with `pf:"subpalette,family=bg"`; renders bg_0..bg_3.
- **Per-2×2-block picker (tilepainter widget)**:
  - `TestTilepainter_PaletteBlockModeAvailable`: tilepainter widget shows "Palette Block" as a mode entry alongside Brush/Bucket/Rect.
  - `TestTilepainter_BlockClickSetsNESPaletteBlock`: TileAtlas with 4×4 grid; click on block at (1,2); `NESPaletteBlock[1][2]` updates to selected sub-palette index; MarkDirty fires.
  - `TestTilepainter_BlockBoundaryConvertsCorrectly`: click on tile-cell (5, 7) in palette-block mode; computes block coords (2, 3) — `NESPaletteBlock[2][3]` updates, not [5][7].
  - `TestTilepainter_NESPaletteBlockMatrixGrowsWithTileAtlas`: TileAtlas grid expanded from 4×4 tiles to 16×16; NESPaletteBlock auto-grows from 2×2 blocks to 8×8 blocks; existing assignments preserved.
- Covers AE1's setup (sprite SubPalette assignment), AE2 (modal appears with Accept/Re-quantize/Reject), AE3 (Re-quantize opens sub-menu), AE4 (warning banner on high-ΔE import), F2 (PNG → diff → action).

**Verification:** `go test ./pfcomponent/... ./pixelforge_studio/editor/...` passes; manual smoke: import a high-color photograph, see warning banner; click Re-quantize, pick a different sub-palette, see updated diff; SpriteAsset inspector shows sub-palette dropdown.

---

### U5. Palette workspace — base grid + sub-palette swatch overlay

**Goal:** New `palette_workspace.go` renders the 64-slot base palette as an 8×8 grid (click any slot → color picker → mutates `PaletteData.Base[i]` → MarkDirty). Renders 8 sub-palette swatch rows below the grid (bg_0..bg_3, sprite_0..sprite_3, each showing 4 swatches with the slot color + slot index). Designer can rename a sub-palette, change a sub-palette's slot assignment (drag swatch from base grid to sub-palette slot OR click sub-palette slot → enter slot index).

**Requirements:** R5 (live restyle when color changes), R6 (Palette workspace shows base + sub-palettes), F1 (designer changes color → whole game restyles).

**Dependencies:** U1 (sub-palette schema).

**Files:**
- `pixelforge_studio/editor/palette_workspace.go` (NEW)
- `pixelforge_studio/editor/palette_workspace_test.go` (NEW)
- `pixelforge_studio/editor/workspaces.go` (MODIFY — register Palette workspace if not already)
- `pixelforge_studio/editor/file_menu.go` (MODIFY — View menu entry to show/hide Palette workspace if not already)

**Approach:**
- Workspace layout: top half = 8×8 grid of base palette colors (rendered as `imgui.ColorButton` per slot); bottom half = 8 rows (4 BG + 4 Sprite), each row showing sub-palette name + 4 swatches.
- Click a base slot → opens `imgui.ColorEdit3` (RGB picker); on change, updates `PaletteData.Base[i]` (parsed hex string) → MarkDirty → live preview reflects the change next frame (engine renders against indices; cascade is automatic per `palettemap.go:8-14`).
- Click a sub-palette swatch → small popup with "slot index: __" input + 4 visible swatches showing all base colors for quick selection.
- Drag-drop within the workspace: drag a base slot onto a sub-palette swatch slot → updates that swatch's slot reference.
- Sub-palette name is editable inline (click → text input).
- Use existing cimgui-go widgets; no custom rendering.
- Per `dirty-state-ux.md`: every mutation through `MarkDirty()`.
- **R5 live restyle**: this works automatically because (a) the engine renders against indices (palettemap.go); (b) idea #1's `pixelforge_entity.RenderAll` (entity sprite renderer) goes through `pixelforge.DrawSprite` which reads `Palette.Base[index]` via `palettemap.PaletteMapping`; (c) the studio preview is the same code path. So mutating `Palette.Base[5]` → next frame's preview shows the new color. **No extra wiring needed in this unit beyond MarkDirty.**

**Patterns to follow:** existing `imgui.ColorButton` / `imgui.ColorEdit3` patterns (search cimgui-go docs); existing workspace registration in `workspaces.go`; existing inspector field-edit MarkDirty discipline.

**Test scenarios:**
- `TestPaletteWorkspace_Renders64SlotGrid`: workspace open; 64 color buttons rendered.
- `TestPaletteWorkspace_ClickSlotOpensColorPicker`: click slot 5; color picker visible.
- `TestPaletteWorkspace_ColorEditUpdatesBase`: change slot 5's color from #8B4513 to #808080; `Palette.Base[5] == "#808080"`; MarkDirty called.
- `TestPaletteWorkspace_RendersSubPaletteRows`: 8 sub-palette rows visible (bg_0..bg_3, sprite_0..sprite_3).
- `TestPaletteWorkspace_SubPaletteSlotShowsCurrentColor`: bg_0 with Slots=[1,2,3,4]; first swatch displays Base[1]'s color.
- `TestPaletteWorkspace_DragBaseSlotToSubPaletteSwatch`: drag base slot 12 onto bg_0's third swatch; `BGSubPalettes[0].Slots[2] == 12`; MarkDirty.
- `TestPaletteWorkspace_RenameSubPalette`: click bg_0 name, type "ground"; `BGSubPalettes[0].Name == "ground"`; MarkDirty.
- `TestPaletteWorkspace_LiveRestyleAfterColorChange`: load project with sprites using bg_0; mutate Base slot that bg_0 references; verify that on next sceneGame.Draw frame, the rendered sprite uses the new color (asserted via pixel-read of the preview ebiten.Image).
- Covers AE1 (color change → restyle), F1.

**Verification:** `go test ./pixelforge_studio/editor/...` passes; manual smoke: open Palette workspace, change a base color, see preview restyle within one frame.

---

### U6. Scanline-overlap overlay — Y-band entity counting

**Goal:** When `EditorOverlays.ScanlineEnabled` is true and the Scene workspace is open, count entities by Y-coordinate band; for any horizontal slice (entity-height-tall band) with ≥9 entities overlapping, paint a red horizontal band into the scene preview `ebiten.Image` inside `sceneGame.Draw`. Hover tooltip explains "NES would flicker — more than 8 sprites on this scanline." Studio-only by construction.

**Requirements:** R11 (overlay), R12 (toggleable), R13 (studio-only), R14 (never gates), R15 (tooltip).

**Dependencies:** U1 (EditorOverlays struct).

**Files:**
- `pixelforge_studio/editor/scene_overlay_scanline.go` (NEW)
- `pixelforge_studio/editor/scene_overlay_scanline_test.go` (NEW)
- `pixelforge_studio/editor/canvas_input.go` (MODIFY — `sceneGame.Draw` calls scanline overlay paint after entity render if enabled)

**Approach:**
- Algorithm per frame (in `sceneGame.Draw`):
  1. If `project.EditorOverlays.ScanlineEnabled` is false, skip.
  2. For each entity in `scene.Entities`, compute its visible Y range: `[entity.TileY*TileH, entity.TileY*TileH + spriteHeight)` (sprite height from the entity's Sprite component or default 8).
  3. Build an array `counts[ScreenHeight]int`. For each entity, increment `counts[y]` for each y in the entity's Y range.
  4. For each y with `counts[y] > 8`, paint a red semi-transparent horizontal band at that y onto the destination `ebiten.Image` via `vector.DrawFilledRect(dst, 0, y, ScreenWidth, 1, redAlpha=128, ...)`.
- For perf: skip if scene has ≤8 entities total (no possible violation). Otherwise the per-frame cost is O(entities × spriteHeight) which is bounded (8×8 sprites, dozens of entities).
- Tooltip: rendered via existing tooltip pattern when the cursor is over a band. Brainstorm specifies the tooltip text verbatim: `"NES would flicker — more than 8 sprites on this scanline."`
- **Why this is studio-only**: the overlay paint happens inside `sceneGame.Draw` (which is the studio's preview compositor), NEVER inside the shipped runtime (which uses `pixelforge_ebiten` directly without going through sceneGame). R13 holds by construction.

**Patterns to follow:** existing `vector.DrawFilledRect` usage in `canvas.go:34-61` (paint patterns); existing tooltip pattern (e.g., asset_browser hover); `docs/solutions/always-on-game-embedding.md` (overlays in the chrome composite pass).

**Test scenarios:**
- `TestScanlineOverlay_DisabledNoPaint`: EditorOverlays.ScanlineEnabled=false; sceneGame.Draw produces output without red bands even when 20 entities overlap one row.
- `TestScanlineOverlay_FewerThan9EntitiesNoBand`: 8 entities on the same Y; no band rendered.
- `TestScanlineOverlay_NineEntitiesPaintsBand`: 9 entities at TileY=10; red band visible at y=80..87 (assuming 8-px-tall sprites: 10×8 = 80).
- `TestScanlineOverlay_TwoSeparateScanlinesBothFlagged`: 9 entities at Y=10 and 9 at Y=20; two red bands.
- `TestScanlineOverlay_PartialOverlapStillCounted`: entities at Y=10 and Y=11 (both 8 px tall, so their Y ranges overlap); both increment counts for y in [88,96); if count > 8, band at that range.
- `TestScanlineOverlay_TooltipTextMatches`: hover cursor over a band; tooltip text contains "NES would flicker".
- `TestScanlineOverlay_PerfBudget_500Entities`: scene with 500 entities; sceneGame.Draw completes in <16ms (60fps budget).
- `TestScanlineOverlay_NoOverlayCodeInShippedBinary`: confirm `sceneGame.Draw` is in the editor package (`pixelforge_studio/editor/canvas_input.go`); confirm the shipped runtime entry point (`pixelforge_ebiten.Run` per the codegen template) does not import the editor package; the overlay code is unreachable from the shipped binary.
- Covers AE5 (9th entity → red band → tooltip), AE7 (shipped binary unaware), F3 (place 9th entity → flicker warning).

**Verification:** `go test ./pixelforge_studio/editor/...` passes; manual smoke: place 9 entities on one Y; observe red band; toggle overlay off; band disappears.

---

### U7. 2×2 BG palette-block consistency overlay

**Goal:** When `EditorOverlays.PaletteBlockEnabled` is true and the Scene workspace is open, iterate the active TileAtlas in 2×2 windows; for each 2×2 block whose 4 tiles reference inconsistent palette assignments (i.e., NESPaletteBlock disagrees with a per-tile expectation OR adjacent blocks conflict), paint a colored outline around the block in the scene preview. Tooltip explains "Real NES requires every 2×2 background block to share one palette quadrant." Studio-only.

**Requirements:** R11 (overlay), R12 (toggleable), R13 (studio-only), R14 (never gates), R15 (tooltip).

**Dependencies:** U1 (EditorOverlays), idea #2 (TileAtlas existence). Note: since v1 stores assignment **per-2×2-block** in `NESPaletteBlock`, the within-block check is trivially satisfied (every block is one quadrant by storage shape). The overlay's real value: when **per-tile override** is added in v2, it would flag tiles within a block that diverge. For v1, the overlay flags **adjacent-block boundary issues** where a designer might inadvertently use sub-palettes whose color clash is visually jarring — though this is more of an aesthetic warning than a hardware constraint.

  **Revised v1 scope:** since per-block storage makes intra-block consistency automatic, the v1 overlay flags **uninitialized blocks** (TileAtlas cells where the tile is painted but `NESPaletteBlock[blockRow][blockCol]` is zero AND the designer hasn't explicitly chosen a sub-palette). These blocks render with bg_0 by default and the overlay outlines them with a yellow border + tooltip: "This 2×2 block has no palette assigned — defaulting to bg_0." Designer can dismiss by explicitly assigning a sub-palette.

  This keeps the overlay meaningful in v1 without per-tile override complexity. When v2 adds per-tile override, the overlay's logic extends to flag genuine intra-block conflicts.

**Files:**
- `pixelforge_studio/editor/scene_overlay_paletteblock.go` (NEW)
- `pixelforge_studio/editor/scene_overlay_paletteblock_test.go` (NEW)
- `pixelforge_studio/editor/canvas_input.go` (MODIFY — sceneGame.Draw calls palette-block overlay paint after scanline overlay if enabled)

**Approach:**
- Algorithm per frame (in `sceneGame.Draw`):
  1. If `project.EditorOverlays.PaletteBlockEnabled` is false, skip.
  2. For the active TileAtlas (one per scene in v1), walk every 2×2 block:
     - `blockRow = row/2`, `blockCol = col/2`.
     - If any tile in the block has `Grid[row][col] != 0` (painted) AND `NESPaletteBlock[blockRow][blockCol] == 0` (unassigned, default-zero meaning bg_0 — note: this is intentionally aliased; designers explicitly assigning bg_0 via U4's picker is allowed; the overlay's heuristic is "block has painted content but no entry was made in NESPaletteBlock matrix").
     - To distinguish "unassigned" from "explicitly-bg_0," use a separate `NESPaletteBlockAssigned [][]bool` matrix OR a sentinel value (e.g., NESPaletteBlock value of -1 = unassigned, 0..3 = explicitly chosen). Decision: use sentinel value -1 for unassigned; defaults populated via `applyDefaults` in U1 to set new TileAtlas blocks to -1. (Updates U1 approach slightly — adds `applyDefaults` handling for NESPaletteBlock initial values.)
  3. For each flagged block, paint a yellow outline rectangle around the 16×16 pixel area (2 tiles × 8 px each).
- Tooltip text: brainstorm-prescribed `"Real NES requires every 2×2 background block share one palette quadrant."`

**Patterns to follow:** same as U6 (vector.DrawFilledRect for outline; tooltip pattern); idea #2's TileAtlas grid iteration.

**Test scenarios:**
- `TestPaletteBlockOverlay_DisabledNoPaint`: EditorOverlays.PaletteBlockEnabled=false; no outlines.
- `TestPaletteBlockOverlay_PaintedBlockUnassignedFlagged`: TileAtlas with painted cells in block (0,0) but NESPaletteBlock[0][0] == -1; yellow outline at pixel (0,0)-(16,16).
- `TestPaletteBlockOverlay_PaintedBlockAssignedNotFlagged`: TileAtlas with painted cells in block (0,0) and NESPaletteBlock[0][0] = 2 (explicitly bg_2); no outline.
- `TestPaletteBlockOverlay_EmptyBlockNotFlagged`: TileAtlas block (0,0) has all-zero cells (unpainted) and NESPaletteBlock[0][0] == -1; no outline (empty blocks aren't violations).
- `TestPaletteBlockOverlay_MultipleViolations`: 5 painted-but-unassigned blocks at various positions; 5 yellow outlines.
- `TestPaletteBlockOverlay_TooltipTextMatches`: hover an outline; tooltip contains "2×2 background block".
- `TestPaletteBlockOverlay_PerfBudget_LargeTileAtlas`: 64×64 tilemap; sceneGame.Draw completes in <16ms even with all blocks flagged.
- Covers AE6 (overlay persistence — same toggle mechanism as U6), F4 (paint incompatible palettes → outline).

**Verification:** `go test ./pixelforge_studio/editor/...` passes; manual smoke: paint tiles, leave palette unassigned, see yellow outlines; right-click block → assign palette → outline disappears.

---

### U8. View menu overlay toggles + per-project persistence

**Goal:** Add two toggleable menu entries under View menu: "Show 8-sprite-per-scanline overlay" and "Show 2×2 BG palette-block overlay." Both default ON. Toggles mutate `Project.EditorOverlays.{ScanlineEnabled, PaletteBlockEnabled}` and MarkDirty. Settings persist with the project file (loaded/saved via existing project save flow).

**Requirements:** R12 (toggleable, default ON, persisted per-project).

**Dependencies:** U1 (EditorOverlays struct on Project), U6 + U7 (overlays exist to toggle).

**Files:**
- `pixelforge_studio/editor/file_menu.go` (MODIFY — add two View menu entries)
- `pixelforge_studio/editor/editor_overlays_settings.go` (NEW — accessors + change handlers)
- `pixelforge_studio/editor/editor_overlays_settings_test.go` (NEW)

**Approach:**
- Two new `widgets.MenuItem` entries in the View menu structure (`Editor.buildMenuDefs()` at file_menu.go:190-196). Each carries a `Checked` state read from `Project.EditorOverlays.X` and an `OnSelect` closure that toggles the bool + MarkDirty.
- Menu rendering reads `Checked` per frame; updates reflect immediately.
- Persistence: `Project.EditorOverlays` is part of the Project struct (U1); existing project save/load handles it. No new save path needed.
- Per `docs/solutions/dirty-state-ux.md`: toggle calls MarkDirty so designer sees `*` in title bar.
- Per `docs/solutions/editor-pforge-schema-shape.md`: defaults via `applyDefaults` ensure pre-v1 projects load with both overlays ON.

**Patterns to follow:** existing menu entry pattern at file_menu.go:190-196; existing `widgets.MenuItem` with `Checked` field (if it exists; else add).

**Test scenarios:**
- `TestViewMenu_ScanlineToggleEntryPresent`: View menu contains "Show 8-sprite-per-scanline overlay" entry.
- `TestViewMenu_PaletteBlockToggleEntryPresent`: View menu contains "Show 2×2 BG palette-block overlay" entry.
- `TestViewMenu_ScanlineToggleFlipsProjectField`: simulate click; `Project.EditorOverlays.ScanlineEnabled` flips; MarkDirty called.
- `TestViewMenu_PaletteBlockToggleFlipsProjectField`: same for palette block.
- `TestViewMenu_CheckedStateReflectsCurrent`: when ScanlineEnabled=true, menu shows checkmark; when false, no checkmark.
- `TestEditorOverlays_PersistedAcrossSaveLoad`: save project with ScanlineEnabled=false, PaletteBlockEnabled=true; reload; values preserved.
- `TestEditorOverlays_NewProjectDefaultsBothTrue`: NewProject(); EditorOverlays.{ScanlineEnabled, PaletteBlockEnabled} both true.
- `TestEditorOverlays_LegacyProjectLoadsWithDefaults`: load editor.pforge (no editor_overlays key); after applyDefaults, both flags are true.
- Covers AE6 (overlays toggleable, settings persist).

**Verification:** `go test ./pixelforge_studio/editor/...` passes; manual smoke: toggle scanline overlay off; reload project; overlay still off.

---

### U9. End-to-end NES palette acceptance tests

**Goal:** Integration tests for AE1-AE7 + F1-F4. Loads fixtures, simulates user actions through the editor's public APIs (not raw cimgui-go events), verifies all acceptance examples.

**Requirements:** R1-R15 covered transitively via AE1-AE7.

**Dependencies:** U1-U8 all merged.

**Files:**
- `pixelforge_studio/integration_test/nes_palette_e2e_test.go` (NEW)
- `pixelforge_studio/integration_test/fixtures/hero_24bit.png` (NEW — 32×32 24-bit RGB image for quantizer)
- `pixelforge_studio/integration_test/fixtures/photograph_high_chroma.png` (NEW — 64×64 image with wide RGB range to trigger ΔE warning)
- `pixelforge_studio/integration_test/fixtures/nes_palette_full_project.pforge` (NEW — project with 8 populated sub-palettes, 2 sprites)
- `pixelforge_studio/integration_test/fixtures/scanline_violation_scene.pforge` (NEW — scene with 9 entities at TileY=10)

**Test scenarios** (one per AE plus F-flow coverage):
- `TestE2E_AE1_ColorChangeRestyleCascades`: load `nes_palette_full_project.pforge` with Hero on sprite_0 (slots 5,12,18,25) and Goomba on sprite_1 (slots 5,30,41,55); mutate `Palette.Base[5]` from "#8B4513" (brown) to "#808080" (grey); call sceneGame.Draw; pixel-read the rendered Hero and Goomba; both reflect the new grey color.
- `TestE2E_AE2_PNGImportShowsDiffWithAutoPickLabel`: import `hero_24bit.png` via `import_handler.Import(path)`; ImportResult captured; ChosenSubPalette = "sprite_0" (auto-pick); diff modal queued with that label.
- `TestE2E_AE3_RequantizeOpensSubMenu`: invoke diff modal Re-quantize; second-level modal opens with sub-palette picker; pick sprite_2; re-import fires; new diff result has ChosenSubPalette = "sprite_2".
- `TestE2E_AE4_HighDeltaEShowsBanner`: import `photograph_high_chroma.png`; ImportResult.MeanDeltaE > 10.0; warning banner text present in modal.
- `TestE2E_AE5_NinthEntityShowsScanlineBand`: load `scanline_violation_scene.pforge` (9 entities at Y=10); sceneGame.Draw; pixel-read at y=80..87 shows red band; tooltip text matches.
- `TestE2E_AE6_OverlayTogglesPersist`: toggle both overlays off; save project; reload; both still off; export-time check: project saves successfully despite the scanline violation in fixture (no gating).
- `TestE2E_AE7_NoOverlayCodeInShippedBinary`: compile-time assertion via import-graph analysis: confirm `pixelforge_ebiten` does not import `pixelforge_studio/editor`. (This is structural; the overlay paint code is unreachable from a shipped binary by construction.)
- `TestE2E_F1_PaletteChangeFullFlow`: open Palette workspace → mutate slot 5 via ColorEdit → preview restyles within one frame.
- `TestE2E_F2_PNGDropFullFlow`: drop hero_24bit.png path → import fires → diff modal → Accept → sprite added to project; SpriteAsset.SubPalette = "sprite_0".
- `TestE2E_F4_BlockViolationFlow`: paint tiles in a 2×2 block without assigning palette → outline appears → assign palette via tilepainter widget → outline disappears.
- `TestE2E_LegacyProjectLoadsWithAllDefaultsAndOverlays`: load `editor.pforge` (legacy empty palette); after load, all 8 sub-palettes populated; EditorOverlays both true; no errors.

**Verification:** `go test ./pixelforge_studio/integration_test/...` passes; all 7 AEs green; F1-F4 covered.

---

## Scope Boundaries

### Deferred to Follow-Up Work

- **Quantizer fast/balanced/quality presets**. `docs/solutions/palette-quantization-metric.md` describes them; code doesn't implement them. v1 ships single-mode Euclidean quantizer. v2 implements presets; Re-quantize sub-menu then adds a preset picker alongside the sub-palette picker.
- **Per-tile palette override** on TileAtlas (a sparse map of `tile_coord → sub_palette_index` that overrides the per-2×2-block default). v1 stores per-block only. v2 adds the override map + overlay flags genuine intra-block conflicts.
- **Drag-drop PNG import on asset browser**. v1 ships File menu only. Drag-drop is an additive UX improvement; the import_handler is the same.
- **Aseprite native palette inheritance** (`.aseprite` files with their own indexed palettes — skip quantizer when source is already indexed). Future asset-pipeline release.
- **Quantizer suggestion to extend palette** (existing capability per `palette-quantization-metric.md`: "this image needs N new slots"). v1 suppresses this — sub-palettes constrain valid slots per import. v2 may repurpose it as "this image would fit sub_palette_X better."
- **AI-assisted palette suggestions** ("here's a palette that fits your art"). Likely out of product identity.
- **Palette swap animations** (authoring UI for `ColorTables` animation). Engine supports it; UI deferred.
- **Per-sprite color override** (sprite uses colors outside its sub-palette). v1: sub-palette assignment is binding.
- **Additional NES authenticity overlays** beyond scanline + 2×2 block: attribute-table over-budget warning, 256-tiles-per-CHR-bank, sprite-0-hit, PPU timing. Two constraints in v1; more layer-on later.
- **NES-strict 32-color palette reduction** — origin scope rejected this; 64-color base stays.
- **Per-project strict-vs-extended mode toggle** — origin scope rejected this.
- **Export-time constraint gating** — origin scope says soft-warn only.
- **Custom user-defined constraints** — designer can't author new ones in v1.
- **Per-frame palette changes mid-scanline** (NES sky-gradient technique). Defers indefinitely.

### Outside this product's identity

- Cloud-hosted palette library (community palette gallery built into studio).
- Browser-based / mobile palette workspace.

---

## Key Technical Decisions

- **Zero external dependencies.** Four candidates evaluated (color science libs, image quantization libs, ImGui modal libs, image diff libs); all rejected via the leverage doctrine. Total custom ~300 LOC.
- **Sub-palette storage: two fixed arrays** (`BGSubPalettes [4]SubPalette` + `SpriteSubPalettes [4]SubPalette`). Three candidates considered; fixed-array form picked because NES has exactly 4 of each (semantic match), reflection is straightforward, defaults are deterministic, and JSON shape is stable.
- **`SpriteAsset.SubPalette`: string field**, not integer index. Easier debugging, plays naturally with the new `WidgetSubPalette` dropdown.
- **`TileAtlas.NESPaletteBlock`: per-2×2-block storage** (not per-tile). v1 ships block-only granularity; per-tile override deferred to v2. Idea #2's plan reserved the field; this plan defines its semantics.
- **Sentinel value `-1` for unassigned NESPaletteBlock entries** (vs `0..3` for explicitly chosen). Lets the 2×2 BG overlay distinguish "unassigned default" from "explicitly bg_0." Updates idea #2's plan U1 with this semantic decision.
- **ΔE warning threshold = 10.0 mean per-pixel Euclidean RGB distance.** Picked from color-science guidance: ΔE > 5 is noticeable, > 10 is significant. Tunable constant in U4; implementer can adjust after trial against the photograph fixture.
- **Re-quantize sub-menu in v1 = sub-palette picker only** (no preset picker). Quantizer presets don't exist in code; implementing them is its own scope. Document the gap explicitly in scope boundaries.
- **Diff modal uses cimgui-go `BeginPopupModal`** — matches post-ImGui-migration substrate. Existing `confirm_modal.go` is pre-migration legacy and not the convention going forward.
- **Overlays paint into scene preview `ebiten.Image` inside `sceneGame.Draw`** — not via `imgui.WindowDrawList()` (no precedent in this codebase) and not via a separate render path. Satisfies R13 (studio-only) by construction.
- **`Project.EditorOverlays` is project-scoped, not editor-wide.** Per R12. Stored on Project, persisted via existing save/load.
- **`WidgetSubPalette` mirrors `WidgetSpriteRef` precedent.** Dynamic options come from `widgets.Context.{BGSubPaletteNames, SpriteSubPaletteNames}`, populated by Editor per render. No new dynamic-enum primitive needed.
- **`palette.Import` is the canonical seam.** No new pipeline; just wire it from a new File menu entry (and optionally drag-drop). Research confirmed the pipeline exists but has no production caller — wiring IS the work.
- **R5 live restyle works automatically** via the engine's indexed rendering (`palettemap.go:8-14`). The studio preview (idea #1's `pixelforge_entity.RenderAll`) goes through the same path. No extra wiring in U5.
- **Auto-pick algorithm = nearest-color total distance per sub-palette**. Simple, deterministic, ~30 LOC. Subsamples 16×16 grid for large images.
- **`PaletteData.applyDefaults` is new** — adds the missing discipline (only Theme had it). Plugs into `Project.applyDefaults` per the existing pattern.

---

## Dependencies / Assumptions

- **Strict dependency on idea #2's plan U1** (TileAtlas struct + `NESPaletteBlock [][]int` reserved field). If idea #2 slips, R3 degrades to scene-level Tilemaps assignment (less granular) per brainstorm fallback.
- **Strict dependency on idea #2's plan U5** (tilepainter widget) for U4's per-2×2-block palette picker extension.
- **Soft dependency on idea #1's plan U4** (`pixelforge_entity.RenderAll` for entity sprite rendering in the editor preview). Without it, the scene preview shows entity markers (flat rects) and R5's "live restyle in preview" isn't observable on entities specifically (it still works on tiles via idea #1's tilemap renderer).
- **Existing `pixelforge_studio/palette/quantize.go`** continues to work unchanged in v1.
- **Existing `pixelforge_studio/palette/import_pipeline.go`** continues to work; extends with optional target sub-palette parameter + ImportResult return type.
- **Existing engine indexed-palette rendering** (`palettemap.go`) cascades palette mutations to next-frame output. No engine change required.
- **Existing reflection inspector** (post-ImGui migration U4). New `WidgetSubPalette` kind plugs into existing dispatch.
- **Existing file picker** (`pixelforge_studio/editor/widgets/file_picker.go`). Used by U3's File → Import PNG menu entry.
- **Existing scene-as-texture preview** (idea #1's plan U5 / canvas_input.go). Overlays paint into the underlying `ebiten.Image`; cimgui-go composites the texture into the workspace via `imgui.ImageWithBgV`.
- **`docs/solutions/palette-quantization-metric.md`** is the source-of-truth for quantizer behavior (existing metric; reuse same score for warning).
- **`docs/solutions/dirty-state-ux.md`** — every palette mutation through MarkDirty.
- **`docs/solutions/focus-manager-design.md`** — diff modal + Re-quantize sub-modal participate in ModalStack.
- **`docs/solutions/always-on-game-embedding.md`** — overlays in the chrome composite pass; one render path.
- **`docs/solutions/editor-pforge-schema-shape.md`** — additive omitempty + sanitize discipline.
- **Solution doc `palette-quantization-metric.md` is ahead of code.** Plan acknowledges; updates `last_verified` field if touched.

---

## Risk Analysis & Mitigation

| Risk | Severity | Mitigation |
|---|---|---|
| Diff modal's image-comparison rendering is slower than 60fps for large imports (e.g., 256×256 source) | Low | Diff modal reuses scene-texture pattern; ImGui `imgui.Image` is GPU-blit (fast). Perf is bounded by texture upload, not per-pixel compute. Worst case: downscale source to 128×128 max for display in modal (preserves visual diff intent). |
| ΔE threshold of 10.0 fires too often / not often enough on typical imports | Low | Tunable constant; implementer adjusts after testing against the photograph fixture. Origin says "noticeable to the eye" — 10.0 is a reasonable starting point per color science. |
| Auto-pick algorithm returns "wrong" sub-palette for designer's intent | Low | Designer can override via Re-quantize sub-menu. Auto-pick is a default; designer agency wins. |
| Scanline overlap detection mis-counts due to entity sprite-height variation | Medium | Default to entity sprite-height when available; fallback to 8 px (the NES sprite-height default). Test scenarios cover both. |
| 2×2 palette-block overlay's "unassigned" detection generates false positives for explicitly-bg_0-assigned blocks | Medium | Sentinel value `-1` for unassigned vs `0..3` for explicit choice (per Key Technical Decisions). Test scenarios cover the distinction. |
| Sub-palette reassignment in Palette workspace breaks existing per-tile assignments (NESPaletteBlock matrix references stale indices) | Low | Sub-palette indices 0..3 stay stable; only sub-palette CONTENTS (Name + Slots) change. NESPaletteBlock matrix references indices, not names — reassignment is transparent to tile assignments. |
| Project save with EditorOverlays:{ScanlineEnabled:false, PaletteBlockEnabled:false} produces non-zero JSON but `omitempty` discards it | Medium | Per U1 decision: use plain bool fields, serialize always when struct contains any non-default value. Document that a `{false, false}` struct round-trips faithfully. |
| Diff modal's Re-quantize sub-modal Esc handling steals Esc from the underlying diff modal | Medium | FocusManager ModalStack handles this: Esc pops the topmost modal first. Explicit test `TestDiffModal_RequantizeOpensSecondLevelModal` covers it. |
| Palette workspace's ColorEdit picker isn't available in cimgui-go binding version | Low | `imgui.ColorEdit3` and `imgui.ColorButton` are standard ImGui widgets; cimgui-go exposes them. Verified plausible from third_party/cimgui-go. |
| Coupling with idea #2's plan creates schema-rename collisions (TileAtlas vs TilemapLayer) | Medium | Coordinate execution order: idea #2's plan U1 (schema rename) lands first; this plan's U1 builds on the renamed type. Both plans modify `pixelforge_project/scenes.go` — careful merge or sequential execution. |
| The shipped binary inadvertently includes overlay code via transitive import | Low | Confirmed structurally: `pixelforge_ebiten` doesn't import `pixelforge_studio/editor`. Integration test (`TestE2E_AE7_NoOverlayCodeInShippedBinary`) asserts via import-graph analysis. |
| New `View → overlay toggles` items break menu rendering due to layout changes | Low | Adds 2 menu items to existing menu; menu width auto-expands. Test scenario covers presence. |

---

## System-Wide Impact

**New packages introduced:** none (extensions of existing).

**Modified packages:**
- `pixelforge_project` — `PaletteData` (sub-palette overlay), `SpriteAsset` (SubPalette field), `Project` (EditorOverlays field), new `palette_defaults.go` for applyDefaults, semantic claim on `TileAtlas.NESPaletteBlock` (idea #2's reserved field).
- `pfcomponent` — new `WidgetSubPalette` kind + applyPFTag extension for `subpalette,family=X` syntax.
- `pixelforge_studio/palette` — new auto-pick algorithm, extended import pipeline (target sub-palette param + ImportResult return type).
- `pixelforge_studio/editor` — new diff modal, new palette workspace, two new overlay paint routines, View menu extensions, file menu Import PNG entry, widgets/context extensions for sub-palette names, inspector dispatch arm for WidgetSubPalette, tilepainter widget extension (from idea #2) for per-block palette picker.

**Affected workflows:**
- **Designer authoring** — primary target. New: import diff modal, Palette workspace, sub-palette dropdown on SpriteAsset, per-block palette picker on TileAtlas, two soft-warn overlays.
- **Engine** — unchanged; palette mutations cascade via existing palettemap.go indirection.
- **Shipped runtime** — unchanged; overlay code unreachable by construction.
- **Codegen / build pipeline (idea #7)** — unchanged; idea #7's Capsule emits the same `pixelforge_ebiten.Run` entry point.

**Documentation impact:**
- Update `docs/solutions/palette-quantization-metric.md`'s `last_verified` field to reflect this plan's date; add a "v1 wiring status: invoked from File → Import PNG menu; presets remain aspirational" note.
- Post-v1, three `docs/solutions/` entries are worth capturing:
  1. Sub-palette schema pattern (additive omitempty + sentinel-value handling).
  2. Studio-only overlay pattern (paint into scene preview ebiten.Image; never reaches runtime).
  3. ΔE warning threshold tuning rationale.

**Operational / rollout:**
- Standard release. Coupled with idea #1 + idea #2 in the same milestone.
- Single existing fixture (`editor.pforge`) loads with auto-populated sub-palettes; designer's existing per-asset (non-existent) state isn't impacted.
- The diff modal is a learnable UX moment; consider linking to a brief docs page from the warning banner or auto-pick label for first-time designers (deferred — see follow-up).

---

## Notes for Implementer

**Coordination with idea #1, idea #2, and the substrate question:**
1. Execute idea #1's plan U1-U5 first (engine renderers + preview integration) and idea #2's plan U1-U7 (TileAtlas schema + RegisterWidget primitive + tilepainter widget) before starting this plan.
2. This plan's U1 schema additions can land in parallel with idea #2's plan U1 (different fields on different types); coordinate the `pixelforge_project/scenes.go` edits to avoid merge conflicts.
3. This plan's U4 extends idea #2's tilepainter widget (`tilepainter_widget.go`); execute idea #2 fully before this plan's U4.
4. The diff modal (U4) and overlays (U6-U7) target different render substrates (cimgui-go for modal; ebiten paint for overlays). This is intentional — modals are UI chrome, overlays are part of the preview composition. The split matches established patterns in this codebase.
5. The Re-quantize preset picker scope cut (v1 = sub-palette only) is documented in the diff modal's UI text — make sure the modal doesn't suggest the picker is incomplete; it's the v1 contract.
