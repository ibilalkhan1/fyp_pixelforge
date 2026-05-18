---
title: "feat: TileAtlas component reframe + emergent auto-rule UX (idea #2 v1)"
type: feat
status: active
date: 2026-05-18
depth: deep
origin: docs/brainstorms/2026-05-18-tileatlas-emergent-rules-v1-requirements.md
ideation: docs/ideation/2026-05-18-pixelforge-nes-class-no-code-ideation.md (idea #2)
ships_with:
  - docs/plans/2026-05-18-002-feat-screenroom-mario-strip-v1-plan.md (idea #1 — same milestone)
related_plans:
  - docs/plans/2026-05-18-001-feat-entity-verb-sheet-v1-plan.md (idea #5 — Archetype + pfcomponent pattern)
supersedes_partially:
  - docs/plans/2026-05-18-002-feat-screenroom-mario-strip-v1-plan.md#u6-tile-painter-ui  (painter home moves from Scene-workspace toolbar into TileAtlas inspector widget)
---

# feat: TileAtlas component reframe + emergent auto-rule UX (idea #2 v1)

## Summary

v1 of idea #2 ships **alongside** idea #1's Mario-strip release as a coupled architectural + UX move. Three layered changes: (1) reframe the tilemap from a special-case `Scene.Tilemaps []TilemapLayer` field into a registered `pfcomponent` called `TileAtlas`, with the painter UI rendered by the existing reflection inspector as a custom widget; (2) build the missing `pfcomponent.RegisterWidget` extension so custom widgets become a reusable primitive (used here by the tilepainter; consumed by idea #4 patch-cast surface and idea #6 dialogue-tree editor); (3) wire the already-working `AutoTileRuleSynth` engine code into the painter via a new promotion-event hook, and surface promoted rules through an inline ImGui toast near the brush cursor with Yes/Esc-No interaction. Threshold bumped from 2 (current code) to 3 (per `docs/solutions/auto-tile-heuristic.md` invariant). Schema reservations for animation frames, parallax factor, slope flags, NES attribute-table palette index — `omitempty`, no v1 UI walks them. Zero external dependencies (the only candidates — ImGui toast libraries — don't exist for cimgui-go; native `OpenPopup` is the path). The decision to put the painter in the inspector (vs the Scene-workspace toolbar that idea #1's plan U6 specified) **supersedes idea #1's U6** in this milestone; the brush/bucket/rect handler code that U6 plans to write still lands in `pixelforge_studio/editor/tile_painter.go`, but it is invoked from the canvas's `ToolPaint` dispatch reading inspector-published state rather than from a toolbar handler reading a sub-mode radio.

---

## Leverage Doctrine (applied)

Per `docs/plans/2026-05-18-001-feat-entity-verb-sheet-v1-plan.md`'s Leverage Doctrine appendix.

**Candidates evaluated:**

| Candidate | Status | Verdict |
|---|---|---|
| `cimgui-go` toast/notification libraries | None exist for this binding | **Build native** — `imgui.OpenPopup` + `imgui.BeginPopup` is the canonical ImGui pattern; ~40 LOC custom |
| Generic auto-tile / WFC libraries (Go) | `Gan-Of-Culture/wfc` exists; ports of WaveFunctionCollapse abundant | **Skip** — solution doc explicitly rejected WFC ("opaque, can't introspect rules"); engine-side synth is already the chosen approach and is already shipped |
| Reflection / inspector framework libraries (cimgui-go) | None mature; community examples are toy-sized | **Build native** — extending existing `pfcomponent` registry is ~80 LOC; wrapping a third-party framework would force a rewrite of all 11 existing widget kinds |
| Tile-painter command/undo libraries | None worth depending on | **Reuse idea #1 plan's undo command stack** (U6 of idea #1 plan) |

Total custom: ~80 LOC RegisterWidget extension + ~40 LOC toast + ~50 LOC promotion hook + ~120 LOC TileAtlas widget drawer + ~80 LOC schema migration + supporting tests. Well below the wrap cost of any candidate.

---

## Problem Frame

Two coupled gaps in Pixelforge today:

**Gap 1 — the auto-rule synth is unwired.** `pixelforge_studio/palette/autotile.go` ships a working `AutoTileRuleSynth`: it watches paint strokes, learns 3×3 neighborhood patterns, promotes patterns to active rules after N matches, and silently auto-applies them on subsequent strokes via `Painter.Paint` (painter.go:67-71). The schema (`pixelforge_project.AutoTileRule`) is on `TilemapLayer` and round-trips through Save/Load already (`TestTilemap_RoundTripsThroughProject` in painter_test.go:158). **The studio never invokes any of this.** Idea #1's Mario-strip designer paints 32-tile-wide ground strips one cell at a time. They never see the auto-rule because the painter UI never calls `RecordStroke`, and even if it did, the synth has no promotion-event hook — `Apply` returns silently, so there's no moment to surface a "we learned a rule, want to use it?" prompt. Two specific defects to fix: (a) wiring, (b) the synth's threshold constant is `2` (autotile.go:141) but the brainstorm and `docs/solutions/auto-tile-heuristic.md` invariant both specify **3** — code needs to align with the documented invariant.

**Gap 2 — tilemaps don't use the established component-registry primitive.** Every other piece of per-entity / per-scene authoring in Pixelforge flows through `pfcomponent.Register[T]` + the reflection inspector dispatch (`pixelforge_studio/editor/inspector.go:130-197`). Tilemaps sit outside that pattern as a special-case schema field. Today that's tolerable because the only tile-feature is "an int grid." But the schema reservations the brainstorm calls for (animation frames, parallax factor, slope flags, NES attribute-table palette index) and the v2+ features they enable (animated tiles, parallax layers, slope physics, NES-correct palette constraints) each become a bespoke editor surface unless tilemaps live on the same primitive. The leverage move is to reframe **once**, in the v1 release, while the surface is small and the only painted fixture is `editor.pforge` with empty `"tilemaps": []`.

A subordinate gap: **`pfcomponent` has no custom-widget registry.** Today the inspector dispatch is a switch over 11 built-in `WidgetKind` values (`metadata.go:25-41`); `pf:"slider,0..100"` works but `pf:"widget=tilepainter"` would land on the `WidgetUnknown` fallback (`metadata.go:209-213`). Building the custom-widget registry is mandatory new code, but it pays leverage forward — idea #4's patch-cast surface and idea #6's dialogue-tree editor both consume the same primitive.

The brainstorm's bet: a designer painting Mario-strip ground for the third time sees a small inline toast: *"Auto-apply this pattern? [Yes / No]"*. Yes → the next 200 ground tiles fill themselves. The designer never learns the term "auto-rule"; the feature is the moment of relief.

---

## Carried Forward from Origin

All 9 requirements (R1-R9), all 5 acceptance examples (AE1-AE5), and all 3 actors (A1-A3) from the origin doc are in scope. The 3 key flows (F1-F3) map cleanly to acceptance tests.

| Origin | Scope summary | Plan unit(s) |
|---|---|---|
| R1, R3 | TileAtlas registered as pfcomponent; legacy `Scene.Tilemaps` migrates at load | U1 (schema reframe), U3 (registration) |
| R2, R9 | Painter rendered as inspector widget; widget-registration extension reusable for idea #4/#6 | U2 (RegisterWidget primitive), U5 (TileAtlas widget) |
| R4 | Studio invokes synth on each stroke; toast on promotion | U4 (promotion-event hook), U7 (toast UX) |
| R5 | Yes activates session; No/Esc dismisses + suppresses for session, persists rule | U7 (toast + session-suppression map) |
| R6, R7 | Accepted rules persist; silent auto-apply on returning sessions | U4 (already on schema; ensure no toast when rule pre-accepted), U7 (suppression seed from project) |
| R8 | Schema reservations for animation, parallax, slope, NES palette | U1 (omitempty fields) |
| AE1-AE5, F1-F3 | All five acceptance examples + three flows | U8 (integration tests) |
| A1-A3 | Designer, Studio, AutoTileRuleSynth — all referenced in flows | All units |

Origin's "Deferred to Planning" section: 5 technical questions resolved in Phase 2 (see Key Technical Decisions). No blocking product questions. One discovered planning question — the 2-vs-3 threshold contradiction between autotile.go:141 and the solution doc — resolved: align code with doc, bump to 3.

---

## High-Level Technical Design

How the four moving parts fit together:

```
                 ┌──────────────────────────────────────────┐
                 │ Scene (.pforge)                          │
                 │ ─ TileAtlases []TileAtlas (NEW field —   │  U1
                 │   replaces Tilemaps; legacy migrates     │
                 │   at load)                               │
                 │   ─ Name, TileW, TileH                   │
                 │   ─ Grid [][]int                         │
                 │   ─ SpriteSheetRef (from idea #1 U1)     │
                 │   ─ AutoTileRules []AutoTileRule         │
                 │   ─ AnimationFps   (reserved omitempty)  │
                 │   ─ ParallaxFactor (reserved omitempty)  │
                 │   ─ SlopeFlags     (reserved omitempty)  │
                 │   ─ NESPaletteBlock(reserved omitempty)  │
                 └────────────┬─────────────────────────────┘
                              │
            pfcomponent.Register[TileAtlas]("TileAtlas")  ── U3
            with pf:"widget=tilepainter" on inspector hook
                              │
       ┌──────────────────────┴─────────────────────────────┐
       ▼                                                    ▼
 ┌──────────────────┐                              ┌──────────────────┐
 │ INSPECTOR        │                              │ CANVAS (Scene    │
 │ (post-ImGui-     │                              │  workspace)      │
 │  migration U4)   │                              │                  │
 │                  │                              │ Toolbar still    │
 │ renderField sees │                              │ shows Select /   │
 │ WidgetCustom →   │                              │ Place / Delete / │
 │ looks up         │                              │ Paint (idea #1   │
 │ registry → calls │                              │ unchanged)       │
 │ tilepainter      │                              │                  │
 │ Drawer           │                              │ ToolPaint case   │
 │                  │                              │ in UpdateAt      │
 │ Drawer renders:  │── publishes ─────────────▶  │ reads:           │
 │ ─ Tool sub-mode  │  e.SelectedTile()           │  e.SelectedTile()│  U6
 │   (Brush/Bucket/ │  e.PaintSubMode()           │  e.PaintSubMode()│
 │    Rect)         │                              │                  │
 │ ─ Tile palette   │                              │ Routes to        │
 │   grid           │                              │ tile_painter.go  │
 │ ─ Undo / Redo    │                              │ handlers (the    │
 │ ─ Active-rules   │                              │ ones idea #1     │
 │   indicator      │                              │ U6 implements)   │
 └──────────────────┘                              └────────┬─────────┘
                                                            │ on stroke end
                                                            ▼
                                              ┌──────────────────────┐
                                              │ AutoTileRuleSynth.   │
                                              │ RecordStrokeWith     │  U4
                                              │ Promotions(layer,    │
                                              │ stroke)              │
                                              │  → []PromotedRule    │
                                              └─────────┬────────────┘
                                                        │ if any
                                                        ▼
                                              ┌──────────────────────┐
                                              │ Toast at brush       │
                                              │ cursor:              │  U7
                                              │ "Auto-apply? Y / N"  │
                                              │                      │
                                              │ Yes → mark active    │
                                              │       in session     │
                                              │ No/Esc → suppress    │
                                              │       for session,   │
                                              │       persist rule   │
                                              │                      │
                                              │ FocusManager-aware   │
                                              │ (Esc dismisses top   │
                                              │ modal first)         │
                                              └──────────────────────┘

                  pfcomponent NEW PRIMITIVE (independent of TileAtlas):
                  ─────────────────────────────────────────────────────
                  pfcomponent.RegisterWidget(name, drawer)            ── U2
                  + extend applyPFTag to recognize pf:"widget=name"
                  + add WidgetCustom kind
                  + inspector dispatch arm

                  Reusable for: idea #4 patch-cast surface,
                                idea #6 dialogue-tree editor
```

*This illustrates the intended approach and is directional guidance for review, not implementation specification.*

The structural insight: **the painter UI and the auto-rule UX live in two different surfaces but share state through `*Editor`.** The painter widget (inspector) publishes selected-tile + active-sub-mode; the canvas (Scene workspace) reads them when Paint dispatch fires; the synth observes the resulting stroke and offers a toast back in the canvas's cursor space. State flows: inspector → editor → canvas → synth → toast → editor (suppression map). No new global state; the existing `*Editor` is the bus.

---

## Output Structure

```
pfcomponent/                                  (MODIFY, U2)
├── metadata.go                               — applyPFTag extension for widget=...
├── widget_registry.go                        (NEW) — RegisterWidget(name, drawer)
└── widget_registry_test.go                   (NEW)

pixelforge_project/
├── scenes.go                                 (MODIFY, U1) — TilemapLayer → TileAtlas; reserved fields
├── scenes_migration.go                       (NEW, U1)   — legacy tilemaps → tile_atlases load-time map
├── scenes_test.go                            (MODIFY, U1)
└── scenes_migration_test.go                  (NEW, U1)

pixelforge_studio/palette/
├── autotile.go                               (MODIFY, U4) — threshold 2→3; new RecordStrokeWithPromotions
├── autotile_test.go                          (MODIFY, U4) — promotion-event coverage
└── painter.go                                (MODIFY, U4) — Paint calls new API; surfaces promotions

pixelforge_studio/editor/
├── inspector.go                              (MODIFY, U2/U5) — WidgetCustom dispatch arm
├── tilepainter_widget.go                     (NEW, U5)       — TileAtlas Drawer + registration
├── tilepainter_widget_test.go                (NEW, U5)
├── tile_painter.go                           (FROM idea #1 U6) — brush/bucket/rect handlers (unchanged from idea #1; just no longer toolbar-driven)
├── tile_palette.go                           (MOVED — was in idea #1 U6 toolbar; now inside tilepainter_widget.go) — see "Supersedes idea #1 U6" section
├── editor.go                                 (MODIFY, U5/U6) — SelectedTile() + PaintSubMode() accessors
├── canvas.go                                 (MODIFY, U6) — ToolPaint case in UpdateAt
├── autotile_toast.go                         (NEW, U7) — popup near cursor; Yes/Esc handling
├── autotile_toast_test.go                    (NEW, U7)
└── session_state.go                          (NEW, U7) — in-memory rule-suppression map keyed by signature

pixelforge_studio/editor/cart_assets/
└── (no change in v1; legacy fixture is empty so migration is a no-op there)

pixelforge_studio/integration_test/
├── tileatlas_e2e_test.go                     (NEW, U8) — covers AE1–AE5
└── fixtures/
    ├── legacy_tilemaps_with_content.pforge   (NEW, U8) — synthetic legacy-shape file
    └── tileatlas_with_accepted_rule.pforge   (NEW, U8) — for AE3 (returning-session silent apply)
```

The implementer may consolidate or split files if implementation reveals a better layout — per-unit `**Files:**` sections remain authoritative.

---

## Implementation Units

Dependency-ordered. Phases roughly: schema + primitives (U1-U4) → integration (U5-U7) → tests (U8).

### U1. Schema reframe — TilemapLayer → TileAtlas + legacy migration

**Goal:** Rename `TilemapLayer` to `TileAtlas`. Rename `Scene.Tilemaps` to `Scene.TileAtlases`. Reserve omitempty fields for future genres. Migrate legacy `tilemaps` JSON field at load time. Schema round-trips cleanly through Save/Load.

**Requirements:** R1 (component-registry framing), R3 (legacy load), R8 (forward-compat schema), AE4 (legacy file loads cleanly), F3 (cheap-to-extend later).

**Dependencies:** none (foundational).

**Files:**
- `pixelforge_project/scenes.go` (MODIFY — rename type; add reserved fields; rename Scene slice)
- `pixelforge_project/scenes_migration.go` (NEW — `UnmarshalJSON` shim for legacy `tilemaps` field)
- `pixelforge_project/scenes_test.go` (MODIFY)
- `pixelforge_project/scenes_migration_test.go` (NEW)
- `pixelforge_project/project.go` (MODIFY — `normalizeSlices` rename per project.go:148-152; `applyDefaults` covers reserved fields)
- All callers of `TilemapLayer` or `Scene.Tilemaps` repo-wide: `pixelforge_studio/palette/autotile.go`, `pixelforge_studio/palette/painter.go`, `pixelforge_studio/palette/painter_test.go`, idea #1's planned tile renderer (`pixelforge_tilemap/`), idea #1's plan U5 preview-integration code — all switch to `TileAtlas` / `TileAtlases`

**Approach:**
- Rename `TilemapLayer` → `TileAtlas` in `pixelforge_project/scenes.go`. Same fields (Name, TileW, TileH, Grid, AutoTileRules) plus `SpriteSheetRef string` added in idea #1's plan U1. Add reserved fields with explicit JSON tags + `omitempty`:
  - `AnimationFps int          `json:"animation_fps,omitempty" pf:"slider,0..30"``
  - `ParallaxFactor float64    `json:"parallax_factor,omitempty" pf:"slider,0.0..2.0"``
  - `SlopeFlags []int          `json:"slope_flags,omitempty"`` (per-tile slope bitfield index; no UI in v1)
  - `NESPaletteBlock [][]int   `json:"nes_palette_block,omitempty"`` (2×2 per-block palette index; idea #3's domain)
- The reserved fields have `pf:` tags so the inspector starts rendering them automatically *the moment* a future change populates them — this is what makes R9's "developer writes a struct field + `pf:` tag and is done" claim real.
- Rename `Scene.Tilemaps []TilemapLayer` → `Scene.TileAtlases []TileAtlas`. JSON tag becomes `json:"tile_atlases,omitempty"`.
- **Legacy migration via Scene.UnmarshalJSON** (`scenes_migration.go`): custom unmarshaler reads the raw JSON, checks for legacy `tilemaps` field; if present, decodes into a `[]TileAtlas` (same shape, the only thing changing is the Go type's name) and assigns to `TileAtlases`. Logs an info-level message about the migration. Save writes only `tile_atlases` (since `tilemaps` field is no longer in the struct, the omitempty discards it cleanly on re-marshal).
- Per `docs/solutions/editor-pforge-schema-shape.md`: `sanitize()` repairs out-of-range fields (e.g., `ParallaxFactor < 0` → clamp to 0; `AnimationFps > 60` → clamp to 60); parse failures degrade to defaults; never fatal.
- `normalizeSlices` (project.go:148-152) renames Tilemaps→TileAtlases; the inner `AutoTileRules` nil-backfill stays.

**Patterns to follow:** existing additive-omitempty pattern (e.g., `TilemapLayer.AutoTileRules` itself); existing `applyDefaults` / `sanitize` discipline per `editor-pforge-schema-shape.md`; existing `[]EventSubscription` slice with nil backfill (project.go); existing JSON-tag conventions on `TilemapLayer`.

**Test scenarios:**
- `TestScene_TileAtlasesRoundTrip`: scene with 2 TileAtlases (one with painted cells, one empty) marshals + unmarshals; field identity preserved.
- `TestScene_LegacyTilemapsLoadsAsTileAtlases`: load JSON with `"tilemaps":[{...painted...}]`; resulting Scene has `TileAtlases` populated, `tilemaps` field gone from re-marshal output.
- `TestScene_LegacyAndNewFieldsCoexistThrowsError` OR resolves predictably: load JSON with BOTH `"tilemaps"` and `"tile_atlases"` keys; behavior is "tile_atlases wins, tilemaps logged as ignored." (Defensive — designers writing files by hand could create this.)
- `TestScene_ReservedFieldsOmitempty`: TileAtlas with zero-valued reserved fields marshals without `animation_fps`, `parallax_factor`, etc. keys.
- `TestScene_ReservedFieldsRoundTrip`: TileAtlas with `AnimationFps=12` marshals with the key; re-unmarshal preserves the value.
- `TestScene_SanitizeClampsOutOfRange`: load JSON with `"animation_fps":99`; after sanitize, value is clamped to 60 (the slider's max from `pf:"slider,0..30"` — wait, decide: clamp to 30 to match the pf-tag declared range, or 60 to allow higher? Decision: clamp to the pf-tag's max so the schema and UI never disagree).
- `TestScene_LegacyEmptyTilemapsMigratesToEmptyTileAtlases`: existing `editor.pforge` with `"tilemaps":[]` migrates to empty `TileAtlases` without errors (regression guard for the only real fixture).
- `TestScene_NormalizeSlices_TileAtlasesNilBackfill`: scene with `nil` TileAtlases is normalized to `[]TileAtlas{}` (matches existing slice-nil-backfill discipline).
- Covers AE4 (legacy file loads cleanly).

**Verification:** `go test ./pixelforge_project/...` passes; round-trip of `editor.pforge` produces no key churn beyond intended renames; all callers of `TilemapLayer` / `Scene.Tilemaps` repo-wide compile after rename.

---

### U2. pfcomponent custom widget registry primitive

**Goal:** Add `pfcomponent.RegisterWidget(name string, drawer Drawer)` registry. Extend `applyPFTag` to recognize `pf:"widget=<name>"` syntax that resolves to a `WidgetCustom` kind. Extend `inspector.renderField` to dispatch `WidgetCustom` through the registry. Drawer receives typed component state, not just `values map[string]any`.

**Requirements:** R2 (widget extension), R9 (primitive reused for future idea #4 + idea #6 surfaces).

**Dependencies:** none (parallel with U1).

**Files:**
- `pfcomponent/widget_registry.go` (NEW)
- `pfcomponent/widget_registry_test.go` (NEW)
- `pfcomponent/metadata.go` (MODIFY — `applyPFTag` parses `widget=...`; new `WidgetCustom WidgetKind`; metadata.go:25-41 + metadata.go:166-215 are the edit sites)
- `pixelforge_studio/editor/inspector.go` (MODIFY — `renderField` switch gains `case pfcomponent.WidgetCustom:` arm; inspector.go:130-197)
- `pixelforge_studio/editor/inspector_test.go` (MODIFY — add coverage)

**Approach:**
- Add `WidgetCustom WidgetKind = "custom"` constant alongside the other 11 in `metadata.go:25-41`.
- Define `Drawer` signature:
  ```go
  type Drawer func(ctx widgets.Context, fieldOwner any, fieldName string, value any)
  ```
  `fieldOwner` is the typed component struct (e.g., `*TileAtlas`); `value` is the current field value; the drawer mutates `value` or `fieldOwner` directly. (This is directional; the implementer may refine the signature after seeing the first consumer — but the constraint is "drawer can access the typed component, not just a string-keyed map.")
- `RegisterWidget(name string, drawer Drawer)`: panics on duplicate registration of `name`; matches existing `Register[T]` discipline (registry.go:50-60). Stores in a package-level `map[string]Drawer`.
- `Lookup(name string) (Drawer, bool)`: read accessor for inspector dispatch.
- Extend `applyPFTag` (metadata.go:166-215) to detect `widget=...` prefix on the first comma-separated token: if found, parse the value as the registered name; set field's WidgetKind = `WidgetCustom`; stash the name in `Field.CustomWidget string` (new field on `Field` metadata struct).
- Extend `Inspector.renderField` (`inspector.go:130-197`): add `case pfcomponent.WidgetCustom:` arm that looks up `pfcomponent.Lookup(f.CustomWidget)`; if found, calls it with the appropriate context; if not found, logs a warning and falls through to read-only-text default (so an unregistered name is recoverable, not fatal).
- **Critical constraint per research finding #4:** the drawer must receive the typed component value, not just the `values map[string]any` that today's dispatch passes. The inspector's `renderEntity` walks `ent.Components` and has the typed `comp.Type` string + the components' `Data map[string]any` payload. For the drawer to get a typed struct, the dispatch will need to construct (or pass through) the typed value — likely by looking up the component's registered Go type via `pfcomponent.Get(comp.Type)` and reflecting Data into it. The implementer makes the final signature call; the requirement is "the drawer can operate on a typed component, not a stringly-typed map."

**Patterns to follow:** existing `pfcomponent.Register[T]` (registry.go:30) for panic-on-duplicate discipline; existing `WidgetKind` enum convention (metadata.go:25-41); existing `applyPFTag` first-token parsing (metadata.go:166-215); `docs/solutions/scripting-runtime-design.md`'s Kind-catalog `Register(name, builder)` pattern.

**Test scenarios:**
- `TestRegisterWidget_StoresAndLooksUp`: register "tilepainter" with a stub drawer; `Lookup("tilepainter")` returns it.
- `TestRegisterWidget_DuplicateNamePanics`: register "tilepainter" twice with different drawers; second call panics. (Match `Register[T]`'s discipline.)
- `TestApplyPFTag_WidgetEqualsSyntax`: field with `pf:"widget=tilepainter"` resolves to WidgetKind=`WidgetCustom`, `Field.CustomWidget="tilepainter"`.
- `TestApplyPFTag_UnknownTagStillFallsThroughToUnknown`: field with `pf:"nonsense"` (no `widget=`) still resolves to `WidgetUnknown` (no regression on existing fallback).
- `TestInspector_DispatchesToRegisteredDrawer`: register a stub drawer; render a component with a `WidgetCustom` field; assert drawer was called with the typed component value.
- `TestInspector_UnregisteredCustomWidgetFallsThroughToReadOnly`: render a `WidgetCustom` field whose name is not in the registry; output is the read-only-text fallback; no panic, warning logged.
- `TestRegisterWidget_NamePresentBeforeRender`: dispatch when registration is missing logs once, doesn't spam (idempotent warning).
- Covers AE5 (developer adds new tile field; inspector renders automatically) — for the `pf:"slider,0..30"` case, AE5 doesn't actually exercise the WidgetCustom path; it exercises the existing slider path. The relevant coverage is "registered TileAtlas drawer renders when field has `widget=tilepainter`" (which is U5's job to wire up — U2 just provides the primitive).

**Verification:** `go test ./pfcomponent/...` and `go test ./pixelforge_studio/editor/...` pass; manual smoke (after U5 lands): selecting a TileAtlas in the inspector shows the custom widget instead of the fallback.

---

### U3. pfcomponent.Register[TileAtlas] production registration

**Goal:** Register `TileAtlas` with `pfcomponent`. Establish the production-registration call site convention (currently only test-only registrations exist per research). Wire the `pf:"widget=tilepainter"` tag on the appropriate field.

**Requirements:** R1 (TileAtlas as registered component).

**Dependencies:** U1 (TileAtlas type exists), U2 (RegisterWidget primitive exists), U5 (the tilepainter drawer to register against — though U3 can land before U5 with a stub drawer registered, replaced when U5 lands).

**Files:**
- `pixelforge_studio/editor/registrations.go` (NEW — central init() block for all production pfcomponent registrations; this establishes the convention idea #1 and idea #5 also need)
- `pixelforge_studio/editor/registrations_test.go` (NEW)
- `pixelforge_project/scenes.go` (MODIFY — add `pf:"widget=tilepainter"` tag on the TileAtlas field that should host the painter UI; likely a synthetic field like `Painter struct{} `pf:"widget=tilepainter"`` or the existing Grid field; decision in approach)

**Approach:**
- `pixelforge_studio/editor/registrations.go` declares an `init()` block that calls `pfcomponent.Register[TileAtlas]("TileAtlas")` (and, ideally, the components idea #1 and idea #5 also need: `Sprite`, `Camera`, plus idea #5's existing registrations — but those are in idea #5's plan; this plan covers only TileAtlas).
- The `pf:"widget=tilepainter"` tag attaches to a designated field that the inspector renders as the painter UI. **Approach decision:** the tag attaches to a synthetic `Painter` field of zero-size type (`struct{}` or a sentinel type), not to `Grid` (which is a 2D int array — its existing rendering as a default would conflict). The synthetic field is the inspector's hook point; the drawer in U5 ignores the field value and operates on the parent TileAtlas struct via the drawer's `fieldOwner` arg.
- Pattern for production registrations: each production component gets a single registration call in `registrations.go`; the file's package-level `init()` runs at studio startup. This establishes the convention that idea #1's `Sprite` and `Camera` registrations (currently absent — see research finding #5) would follow.
- Until U5 lands, U3 can register a stub drawer that renders placeholder text. U5 replaces the stub with the real drawer.

**Patterns to follow:** existing `Register[Facing]` test-only call (inspector_test.go:155); the discipline in `pfcomponent/registry.go:30-60` (panic-on-duplicate, idempotent for same `(T, name)`).

**Test scenarios:**
- `TestRegistrations_TileAtlasRegistered`: after `init()` runs, `pfcomponent.Get("TileAtlas")` returns non-nil metadata.
- `TestRegistrations_TileAtlasHasWidgetCustomField`: the metadata exposes a field with `WidgetKind=WidgetCustom` and `CustomWidget="tilepainter"`.
- `TestRegistrations_TileAtlasReservedFieldsHavePFTags`: AnimationFps has WidgetKind=`WidgetSlider`; ParallaxFactor likewise; SlopeFlags = WidgetUnknown (no v1 widget); NESPaletteBlock = WidgetUnknown.
- `TestRegistrations_IdempotentOnReinit`: calling `Register[TileAtlas]("TileAtlas")` again does not panic (matches existing idempotency contract).

**Verification:** `go test ./pixelforge_studio/editor/...` passes; manual smoke: studio launches without panic.

---

### U4. AutoTileRuleSynth promotion-event hook + threshold alignment

**Goal:** Add `RecordStrokeWithPromotions(layer, stroke) []PromotedRule` API to `AutoTileRuleSynth`. Bump activation threshold from 2 to 3 per `docs/solutions/auto-tile-heuristic.md` invariant. Existing `RecordStroke` continues to work (called by `Painter.Paint`); the new API wraps it and returns rules promoted by this specific stroke.

**Requirements:** R4 (studio invokes synth on stroke; surface promotion).

**Dependencies:** U1 (TileAtlas is the renamed TilemapLayer — the synth's existing signature changes).

**Files:**
- `pixelforge_studio/palette/autotile.go` (MODIFY — new API, threshold change)
- `pixelforge_studio/palette/painter.go` (MODIFY — `Painter.Paint` invokes new API; returns or stashes promotions)
- `pixelforge_studio/palette/autotile_test.go` (MODIFY — add promotion-event coverage; update existing tests for threshold 3)
- `pixelforge_studio/palette/painter_test.go` (MODIFY — update for threshold 3 in existing `TestAutoTile_PatternPaintedTwicePromotesRule` → renamed `TestAutoTile_PatternPaintedThriceCorrespondingly`)

**Approach:**
- Change `AutoTileActivationThreshold = 2` → `AutoTileActivationThreshold = 3` (autotile.go:141). This breaks the existing `TestAutoTile_PatternPaintedTwicePromotesRule` (painter_test.go:52) — rename + extend it to test thrice-paints-promotes. Document the change in the test rename.
- Define:
  ```go
  type PromotedRule struct {
      RuleIndex int       // index into layer.AutoTileRules
      Pattern   [9]int
      Output    int
  }
  func (s *AutoTileRuleSynth) RecordStrokeWithPromotions(layer *TileAtlas, stroke []PaintCell) []PromotedRule
  ```
- Implementation: before calling existing `RecordStroke`, capture per-rule `Count` values (or the indices of rules with `Count == threshold-1`); after `RecordStroke`, detect any rule that crossed the threshold (was below, now at-or-above); package those as `[]PromotedRule` and return.
- A rule that promotes once should not re-promote on later strokes (the toast fires once per rule per session). The promotion detection is "Count crossed the threshold this stroke," not "Count >= threshold." Existing behavior of `Apply` (returns true when Count >= threshold) is unchanged.
- `Painter.Paint` (painter.go:67-71) updated to call `RecordStrokeWithPromotions` instead of `RecordStroke`. The returned `[]PromotedRule` is stashed on the Painter (or returned from Paint) so the studio's stroke-end hook can read it.
- Per `docs/solutions/auto-tile-heuristic.md` invariant (d): rules are HINT, on-disk grid is source-of-truth. The new API does not change this — promotions are signals to the UI, not mutations to the grid.

**Patterns to follow:** existing `RecordStroke` (autotile.go:30) for the observation pattern; existing test discipline in `painter_test.go:52-200` for behavior assertions; the invariants in `docs/solutions/auto-tile-heuristic.md`.

**Test scenarios:**
- `TestAutoTile_PatternPaintedTwiceDoesNotPromote`: paint a pattern twice; `RecordStrokeWithPromotions` returns empty `[]PromotedRule` on both strokes; rule has Count=2; not yet active.
- `TestAutoTile_PatternPaintedThricePromotes`: paint a pattern three times; third stroke's `RecordStrokeWithPromotions` returns one `PromotedRule` for that pattern; rule has Count=3; `Apply` now returns true for matching cells.
- `TestAutoTile_AlreadyActiveRuleDoesNotRepromote`: paint a fourth, fifth, sixth time; subsequent strokes do NOT include the rule in promotions (it's already past threshold). Count keeps incrementing but promotion fires once.
- `TestAutoTile_TwoPatternsPromoteIndependently`: paint pattern A three times, pattern B three times; promotions fire once each, at the third stroke for each.
- `TestAutoTile_ThresholdConstantIs3`: assert `AutoTileActivationThreshold == 3` (regression guard; the solution doc is the source of truth).
- `TestAutoTile_RuleHintNotTruth_GridUnchangedOnRuleCorruption`: paint a pattern three times normally; manually corrupt the rule's `Output` to an invalid tile ID; subsequent `Apply` calls return false (or the invalid ID, but the grid's painted cells remain whatever the designer painted); painted cells are NOT mutated retroactively by rule changes (verifies the hint-vs-truth invariant).
- Existing tests updated: `TestAutoTile_PatternPaintedTwicePromotesRule` → renamed and reframed for threshold 3.
- Covers F1 step 2 (synth promotes pattern to active rule).

**Verification:** `go test ./pixelforge_studio/palette/...` passes; the threshold-3 change does not regress any existing painter behavior beyond the renamed test; `RecordStrokeWithPromotions` is the new canonical entrypoint for the studio.

---

### U5. TileAtlas inspector widget (the tilepainter Drawer)

**Goal:** Implement the Drawer registered as `"tilepainter"` in U3. The drawer renders, inside the inspector: a tool sub-mode picker (Brush / Bucket / Rect), a tile palette grid sourced from the TileAtlas's bound `SpriteSheetRef`, undo/redo buttons, and a small active-rules indicator. Publishes selected-tile + active-sub-mode to `*Editor` so the canvas's Paint dispatch (U6) can read them.

**Requirements:** R2 (painter is inspector widget), R5 (undo/redo present), R7 (auto-apply happens silently — implies state must be readable to apply without UI interruption), R9 (drawer is reusable pattern for idea #4/#6).

**Dependencies:** U1 (TileAtlas type), U2 (RegisterWidget + WidgetCustom dispatch), U3 (TileAtlas registered).

**Files:**
- `pixelforge_studio/editor/tilepainter_widget.go` (NEW — Drawer impl + RegisterWidget call)
- `pixelforge_studio/editor/tilepainter_widget_test.go` (NEW)
- `pixelforge_studio/editor/editor.go` (MODIFY — add `SelectedTile() int`, `PaintSubMode() PaintSubMode`, `SetSelectedTile(int)`, `SetPaintSubMode(PaintSubMode)` accessors; analogous to `SelectedSpriteName()` at editor.go:113-114 area)
- `pixelforge_studio/editor/registrations.go` (MODIFY from U3 — replace stub drawer with the real one once U5 lands; the registration call is `pfcomponent.RegisterWidget("tilepainter", tilepainterDraw)`)
- `pixelforge_studio/editor/tile_painter.go` (NEW — brush/bucket/rect handlers from idea #1's plan U6, ported to read state from inspector accessors; **this file is the home for the painter logic that idea #1 U6 owns conceptually, but the dispatcher inside is invoked from the canvas's Paint case (U6 of this plan), not from a toolbar**)
- `pixelforge_studio/editor/undo_stroke.go` (FROM idea #1 plan U6 — command-pattern stack with cell-diff lists; unchanged from idea #1's plan)

**Approach:**
- The Drawer function signature (final form decided in U2):
  ```go
  func tilepainterDraw(ctx widgets.Context, fieldOwner any, fieldName string, value any)
  ```
  Type-asserts `fieldOwner` to `*TileAtlas`. Renders ImGui controls.
- UI elements:
  - **Tool sub-mode picker**: three radio buttons (Brush / Bucket / Rectangle). On click, calls `editor.SetPaintSubMode(...)`. Initial state read from `editor.PaintSubMode()`.
  - **Tile palette grid**: child window showing tiles from the TileAtlas's `SpriteSheetRef` as a grid of `ImageButton` (or `Selectable` with image). Click selects active tile → `editor.SetSelectedTile(id)`. Highlight current selection.
  - **Undo / Redo buttons**: standard `imgui.Button("Undo")` / `imgui.Button("Redo")`; dispatch to the existing undo stack from idea #1's plan U6 (`undo_stroke.go`).
  - **Active-rules indicator**: small subtle text like `🟢 12 auto-rules active` (no emoji per project conventions — use text or icon font); on hover, tooltip lists pattern signatures and outputs. v1 has no rule-management UI (per origin scope boundary), so this is read-only.
- `*Editor` accessors:
  ```go
  type PaintSubMode int
  const (
      PaintBrush PaintSubMode = iota
      PaintBucket
      PaintRectangle
  )
  func (e *Editor) SelectedTile() int
  func (e *Editor) SetSelectedTile(id int)
  func (e *Editor) PaintSubMode() PaintSubMode
  func (e *Editor) SetPaintSubMode(m PaintSubMode)
  ```
  Default selected tile = 0; default sub-mode = `PaintBrush`. These are the state the canvas dispatch (U6) reads.
- Per `docs/solutions/dirty-state-ux.md`: any change inside the widget that mutates the project (e.g., the future "right-click rule → delete" — out of v1 scope) routes through `MarkDirty()`. State changes that are session-only (selected tile, current sub-mode) do NOT mark dirty.
- Per `docs/solutions/focus-manager-design.md`: the inspector widget is not a modal; existing FocusManager rules apply unchanged.

**Patterns to follow:** existing inspector widget patterns under `pixelforge_studio/editor/widgets/`; existing `editor.SelectedSpriteName()` accessor pattern; existing tool-radio pattern from `pixelforge_studio/editor/workspaces.go:200-208` (just relocated from toolbar to inspector); existing `pixelforge_studio/editor/asset_browser.go` for tile-grid layout reference.

**Test scenarios:**
- `TestTilepainterDraw_RegistersOnInit`: after init, `pfcomponent.Lookup("tilepainter")` returns the drawer.
- `TestTilepainterDraw_RendersSubModePicker`: with TileAtlas selected, drawer outputs the three radio buttons (verified via stub ImGui context capturing operations).
- `TestEditor_SelectedTileAccessor`: `SetSelectedTile(5)` then `SelectedTile()` returns 5; default is 0.
- `TestEditor_PaintSubModeAccessor`: `SetPaintSubMode(PaintBucket)` then `PaintSubMode()` returns `PaintBucket`; default is `PaintBrush`.
- `TestTilepainterDraw_TileClickUpdatesSelection`: simulate click on tile at index 7 in the palette grid; `editor.SelectedTile()` becomes 7.
- `TestTilepainterDraw_UndoButtonCallsUndoStack`: click Undo; the undo stack's Undo method invoked.
- `TestTilepainterDraw_ActiveRulesCountReflectsLayer`: TileAtlas with 12 AutoTileRules where 5 have Count >= 3; indicator text shows "5 auto-rules active" (the active subset, not the total).
- `TestTilepainterDraw_NoSpriteSheetRefShowsEmptyPalette`: TileAtlas without SpriteSheetRef; palette grid is empty; toast or hint suggests "bind a sprite sheet to start painting" (graceful empty state).
- `TestTilepainterDraw_NoDirtyOnSubModeChange`: switching sub-mode does NOT call `editor.MarkDirty()` (session-only state).
- Covers AE1's painter-rendering half (the "designer paints" precondition assumes the inspector widget is functional).

**Verification:** `go test ./pixelforge_studio/editor/...` passes; manual smoke: open studio, select a scene, select a TileAtlas in the inspector, see the painter UI render; click tiles in the palette; switch sub-modes.

---

### U6. Canvas ToolPaint dispatch — wire painter into scene clicks

**Goal:** Add `case ToolPaint:` in `Canvas.UpdateAt` (canvas.go). The case reads `editor.SelectedTile()` + `editor.PaintSubMode()` and routes to the appropriate handler in `tile_painter.go` (brush / bucket / rect). Routes the result through the `AutoTileRuleSynth` promotion API from U4. **Supersedes idea #1's plan U6's toolbar wiring** — the toolbar still shows Paint as a tool button (mode selector) but the sub-mode picker is in the inspector widget, not the toolbar.

**Requirements:** R7 (auto-apply happens silently inline with painting — the dispatch is the seam where rule application meets canvas clicks).

**Dependencies:** U1 (TileAtlas type), U4 (RecordStrokeWithPromotions API), U5 (Editor state accessors). Tightly couples with idea #1's plan U6 — both modify `canvas.go`; this plan's U6 takes ownership of the Paint dispatch case while idea #1's U6 ships the underlying tile_painter.go handlers and the toolbar-level Paint button.

**Files:**
- `pixelforge_studio/editor/canvas.go` (MODIFY — add `case ToolPaint:` arm in `UpdateAt`; ToolPaint dead-code today per research)
- `pixelforge_studio/editor/canvas_test.go` (MODIFY)
- `pixelforge_studio/editor/tile_painter.go` (CO-OWN with idea #1's plan U6 — exports handler functions invoked from canvas)
- `pixelforge_studio/editor/workspaces.go` (MODIFY — remove Paint sub-mode picker from toolbar per supersession; keep the Select/Place/Delete/Paint top-level radio buttons)

**Approach:**
- `Canvas.UpdateAt` (canvas.go:78) gains:
  ```
  case ToolPaint:
      tile := e.SelectedTile()
      mode := e.PaintSubMode()
      stroke := dispatchPaintToolMode(mode, layer, x, y, tile, lmbState)  // delegates to tile_painter.go
      if stroke != nil {  // stroke completed (LMB-up after drag, or single click)
          promotions := synth.RecordStrokeWithPromotions(layer, stroke)
          if len(promotions) > 0 {
              e.QueueToast(promotions[0], cursorScreenPos)  // U7 handles toast
          }
      }
  ```
- `dispatchPaintToolMode` is the new entry point in `tile_painter.go` that fans out to `brush.go`, `bucket.go`, `rect.go` handlers (the same logic idea #1's plan U6 was going to ship — now invoked from canvas dispatch with inspector-published state).
- Strokes are collected across LMB-down → LMB-up for brush; a single LMB-click for bucket/rect (with rect's drag preview tracked separately).
- Auto-apply (the silent `Apply`) already happens inside `Painter.Paint` (painter.go:67-71). When a click would paint cell X with value V, but an active rule matches the neighborhood with output W, the painted value is W instead. **This is unchanged from existing behavior.** The new piece is the promotion-event surfacing.
- **Supersedes idea #1 plan U6's toolbar sub-mode picker**: workspaces.go's toolbar still shows the 4 top-level tool buttons (Select / Place / Delete / Paint) per current state, but the sub-mode radio (Brush / Bucket / Rect) that idea #1's plan was going to put on the toolbar moves to the inspector widget (U5). Idea #1's plan U6 file `pixelforge_studio/editor/tile_palette.go` is folded into `tilepainter_widget.go`.
- Per `docs/solutions/always-on-game-embedding.md`: the canvas paint dispatch must not pause the always-on game preview; the dispatch is per-click + per-stroke-end, not a long-running modal handler.

**Patterns to follow:** existing tool dispatch in `canvas.go:88-135` for `ToolPlace`, `ToolDelete`, `ToolSelect`; existing inspector→canvas state-read precedent (`e.SelectedSpriteName()` at canvas.go:113-114).

**Test scenarios:**
- `TestCanvas_ToolPaintBrushClickPaintsCell`: editor in PaintBrush mode with SelectedTile=3; LMB click at cell (5, 10); resulting Grid[10][5] == 3.
- `TestCanvas_ToolPaintBucketClickFlooFills`: editor in PaintBucket mode; click at (0,0) on grid `[[0,0,1],[0,0,1]]` with SelectedTile=5; cells (0,0), (0,1), (1,0), (1,1) become 5; (0,2), (1,2) unchanged.
- `TestCanvas_ToolPaintRectangleDragFills`: editor in PaintRectangle mode; LMB-down at (5,10), drag to (8,12), LMB-up; rect (5,10)-(8,12) all painted with SelectedTile.
- `TestCanvas_ToolPaintStrokeInvokesPromotionAPI`: paint a stroke that promotes a rule; `synth.RecordStrokeWithPromotions` called with the stroke cells; promotions are non-empty for the third-time pattern.
- `TestCanvas_ToolPaintSilentApplyRespectsActiveRules`: TileAtlas with one active rule (Pattern X → Output W); click at a cell where painting would complete Pattern X; resulting cell value is W (not the SelectedTile).
- `TestCanvas_ToolPaintNoSubMode_DefaultsToBrush`: editor in PaintMode with PaintSubMode never set; behavior matches PaintBrush default.
- `TestCanvas_ToolPaintNoSelectedTile_ClearsCell`: editor in PaintMode with SelectedTile=0 (the default); clicks clear the cell (paint 0 == clear; standard convention).
- `TestCanvas_ToolPaintNoTileAtlasSelected_NoOp`: editor in PaintMode but no TileAtlas selected in the inspector; clicks do nothing; no error.
- `TestWorkspaces_ToolbarShowsPaintWithoutSubModeRadio`: toolbar render lists Paint button but no sub-mode picker (regression guard for the supersession).
- Covers AE1's mechanics (paint a pattern; tile lands).

**Verification:** `go test ./pixelforge_studio/editor/...` passes; manual smoke: in studio, select TileAtlas in inspector, pick a tile from palette, set Paint tool on toolbar, click in canvas → cell paints.

---

### U7. Auto-rule toast UX — promotion popup + Yes/No/Esc handling + session suppression

**Goal:** When U6's dispatch surfaces a `PromotedRule`, render an ImGui popup near the brush cursor reading "Auto-apply this pattern? [Yes / No]." Yes = rule activated for session (next matching strokes silently auto-apply). No / Esc / click-outside = rule persists in project file but is suppressed for the current session via an in-memory map keyed by rule signature. FocusManager-aware: Esc dismisses this popup before any chrome-visibility toggle.

**Requirements:** R4 (toast on promotion), R5 (Yes/No/Esc interaction + session-suppression semantics), R6 (accepted rules persist), R7 (silent auto-apply).

**Dependencies:** U4 (PromotedRule data flowing from synth), U6 (dispatch queues toast).

**Files:**
- `pixelforge_studio/editor/autotile_toast.go` (NEW — popup component; opens, renders, dismisses)
- `pixelforge_studio/editor/autotile_toast_test.go` (NEW)
- `pixelforge_studio/editor/session_state.go` (NEW — `SessionRuleSuppression map[ruleSignature]struct{}` + accessor methods on Editor)
- `pixelforge_studio/editor/session_state_test.go` (NEW)
- `pixelforge_studio/editor/editor.go` (MODIFY — wire toast queue + session state into editor lifecycle)

**Approach:**
- `ruleSignature` = a stable hash of `(Pattern, Output)`. Used as map key for session-suppression. Defined in `session_state.go`.
- `SessionRuleSuppression`: `map[ruleSignature]struct{}` on the editor. Initialized empty on session start (editor construction). When the designer picks "No" in the toast, the rule's signature is inserted. The toast subsystem (and the synth-wiring in U6) consults this map; if a promoted rule's signature is in the map, do NOT surface the toast.
- **Session-suppression seeding** on project load (R6 / AE3): when a project loads, any rules with `Count >= threshold` (i.e., already accepted) are NOT added to the suppression map. They simply auto-apply silently — the existing `Painter.Paint` behavior already does this. The suppression map is only populated by user "No" actions in this session.
- Toast popup: ImGui `OpenPopup` + `BeginPopup` (or `BeginPopupContextItem` for click-outside-dismiss semantics). Position via `imgui.SetNextWindowPos(cursorScreenPos, imgui.CondAppearing)` so it lands near the brush cursor at the moment of promotion.
- Toast UI:
  ```
  Auto-apply this pattern?
  [Yes]   [No]
  ```
  - Yes (or Enter key): rule already promoted; do nothing extra (auto-apply happens on next matching paint via `Painter.Paint`). Dismiss popup.
  - No (or Esc, or click-outside): insert rule signature into `SessionRuleSuppression`. Dismiss popup. Rule persists in the project file because it's still on `TileAtlas.AutoTileRules` (no mutation needed).
- **Esc handling** per `docs/solutions/focus-manager-design.md`: register the popup with the FocusManager / ModalStack so Esc dismisses the popup before dismissing other modals or toggling chrome visibility. Toast counts as a transient modal in the stack.
- **Never pause the always-on game** per `docs/solutions/always-on-game-embedding.md`: the toast renders but doesn't block input dispatch to the game. ImGui popups by default don't pause game updates; the FocusManager registration ensures Esc routes correctly.
- Toast queue: if multiple promotions land in a single stroke (unusual but possible for complex strokes), only the first promotion shows a toast in v1. Other promotions are auto-suppressed (added to session suppression) until the user dismisses the visible toast — this avoids stacking toasts. (Trade-off documented; v2 could chain them.)

**Patterns to follow:** `docs/solutions/focus-manager-design.md` for Esc handling; `docs/solutions/always-on-game-embedding.md` for don't-pause discipline; existing `confirm_modal.go` for modal pattern reference (though confirm_modal is heavier — toast is lighter); cimgui-go's standard `OpenPopup`/`BeginPopup` from `third_party/cimgui-go/imgui/enums.go:1581+`.

**Test scenarios:**
- `TestToast_PromotionQueuesToast`: U6 calls `editor.QueueToast(promotedRule, cursorPos)`; toast state shows pending toast for that rule.
- `TestToast_RenderShowsYesNoButtons`: render toast; output contains "Yes" and "No" buttons.
- `TestToast_YesActivatesRule_NoSuppressionInsert`: simulate Yes click; suppression map remains empty; toast dismisses.
- `TestToast_NoInsertsRuleSignatureInSuppression`: simulate No click; suppression map contains the rule's signature; toast dismisses.
- `TestToast_EscBehavesLikeNo`: simulate Esc keypress; suppression map contains signature; toast dismisses.
- `TestToast_ClickOutsideBehavesLikeNo`: simulate click outside popup bounds; suppression map contains signature; toast dismisses.
- `TestToast_SuppressedRuleDoesNotResurface`: after Yes/No on rule R; queue another promotion for R (re-paint enough to re-promote); toast does NOT appear (suppression check fires).
- `TestToast_RuleAcceptedInPriorSession_NoToastOnLoad`: load project with TileAtlas containing a rule at Count=4 (above threshold); open the painter; paint a matching pattern; auto-apply happens; NO toast surfaces (R6 / AE3).
- `TestToast_SessionSuppressionDoesNotPersist`: write a project, dismiss a toast with No, save+close+reload; suppression map is empty after reload (correct — suppression is session-scoped).
- `TestToast_MultiplePromotionsOnlyShowsFirst`: synth returns 2 PromotedRules in one stroke; toast shows for rule 1; rule 2's signature auto-added to suppression (no stacked toasts).
- `TestToast_FocusManagerEscPrecedence`: open another modal stack on top of toast; Esc dismisses the top modal first (FocusManager invariant).
- `TestSessionState_RuleSignatureStable`: same (Pattern, Output) → same signature; different Pattern or Output → different signature.
- Covers AE1 (Yes case + subsequent auto-apply), AE2 (No / Esc dismiss + persist), AE3 (returning-session silent apply).

**Verification:** `go test ./pixelforge_studio/editor/...` passes; manual smoke: in studio, paint a 3×3 pattern three times; toast appears; click Yes; paint a fourth matching pattern; tile auto-fills silently. Test No path: dismiss toast, paint another 3 matching patterns, observe re-promotion (count keeps rising but suppression blocks toast).

---

### U8. End-to-end TileAtlas + auto-rule acceptance tests

**Goal:** Integration test ties U1-U7 together. Loads fixtures (legacy-shape + new-shape), simulates paint strokes, verifies all 5 acceptance examples from origin.

**Requirements:** R1-R9 covered transitively via AE1-AE5.

**Dependencies:** U1-U7 all merged.

**Files:**
- `pixelforge_studio/integration_test/tileatlas_e2e_test.go` (NEW)
- `pixelforge_studio/integration_test/fixtures/legacy_tilemaps_with_content.pforge` (NEW — hand-built `.pforge` using pre-v1 `"tilemaps":[...]` field with painted cells)
- `pixelforge_studio/integration_test/fixtures/tileatlas_with_accepted_rule.pforge` (NEW — `.pforge` with one TileAtlas containing an AutoTileRule at Count=4; used for AE3 returning-session test)

**Approach:**
- Each test loads a fixture via the project loader, constructs an editor state, simulates the actor's interactions through the painter API (not ImGui — direct calls to `editor.SetPaintSubMode`, `editor.SetSelectedTile`, `canvas.UpdateAt`), and asserts on resulting state.
- Toast interactions tested via direct calls to the toast subsystem's handlers (Yes/No/Esc), not via simulated ImGui events.
- Auto-apply assertions check `Grid[row][col]` values after strokes that should trigger silent rule application.

**Test scenarios (one per AE + the flows):**
- `TestE2E_AE1_BrickCornerThirdPaintTriggersToast_YesActivates`: paint 3×3 brick-corner pattern three times; assert toast is queued after third stroke; simulate Yes; paint a fourth matching neighborhood elsewhere; assert auto-filled cell matches the rule's Output (not the SelectedTile).
- `TestE2E_AE2_NoDismissesAndPersists`: paint 3×3 pattern three times; toast queued; simulate Esc; assert toast dismisses, suppression map has rule signature, rule still present in `TileAtlas.AutoTileRules`; save project to disk; load it again in a fresh editor; rule is still in `AutoTileRules` (persistence holds; suppression resets).
- `TestE2E_AE3_AcceptedRuleSilentApplyOnReload`: load `tileatlas_with_accepted_rule.pforge` (rule at Count=4); paint a matching cell; assert auto-fill happens silently; assert NO toast queued.
- `TestE2E_AE4_LegacyTilemapsFixtureLoadsCorrectly`: load `legacy_tilemaps_with_content.pforge` (has `"tilemaps":[{...painted cells...}]`); resulting Scene has `TileAtlases` populated with the painted cells; round-trip save produces only `"tile_atlases":[...]` (no `"tilemaps"` key); painter renders the migrated content.
- `TestE2E_AE5_NewTileFieldRendersAutomatically`: programmatically add a hypothetical `TestField int `pf:"slider,0..30"`` to a copy of TileAtlas struct; register it; assert `pfcomponent.Get("TileAtlas").Fields` includes the new field with WidgetSlider; assert inspector dispatches the slider widget for it. (This test simulates the "developer adds a field" flow in isolation, since modifying TileAtlas mid-test is awkward.)
- `TestE2E_F1_FullFlow_PaintToToastToActivateToAutoApply`: end-to-end flow from F1 — designer paints third matching cell; toast appears; designer clicks Yes; subsequent matching paints silently auto-fill.
- `TestE2E_F2_ReturningSessionAppliesPreviousRules`: end-to-end flow from F2 — load a project with one accepted rule, paint a matching pattern in a brand-new editor session; auto-fill happens silently with no toast.

**Verification:** `go test ./pixelforge_studio/integration_test/...` passes; all 5 AEs + 2 of 3 flows green (F3 is forward-looking — a v2 dev experience — and is exercised indirectly via AE5).

---

## Scope Boundaries

### Deferred to Follow-Up Work

- **Rule-management inspector panel** (toggle rules, delete bad rules, edit patterns) — out of v1. A "No" decision is sticky for the session; designer can't easily revisit without re-painting enough to re-trigger. Lands in v2.
- **Animated tiles** (per-tile frame-cycling) — schema reserved (`AnimationFps`); no v1 UI walks it. v2 adds a frame-strip editor in the TileAtlas widget.
- **Parallax tile layers** (per-layer scroll factor for multi-plane scrolling) — schema reserved (`ParallaxFactor`); no v1 UI. v2 adds the factor slider to the TileAtlas widget and engine code multiplies the camera offset.
- **Slope / per-tile collision flags** — schema reserved (`SlopeFlags`); no v1 UI walks it. v2 adds a per-tile collision-class picker.
- **NES attribute-table colors per 2×2 background block** — schema reserved (`NESPaletteBlock`); v1 has no NES-palette-correct preview. Idea #3's plan owns this.
- **Multi-layer tilemaps** (background + collision + decoration as separate paintable TileAtlases in one scene) — out of v1. v1 ships one TileAtlas per scene. The component-registry reframe makes adding a second `[]TileAtlas` instance per scene cheap; v2 turns it on with UI.
- **Author-time rule editor** (designer writes 3×3 patterns explicitly) — out of v1 (and likely out of product identity per origin scope).
- **Semantic IntGrid painting** (LDtk-style 5-color "ground / wall / spike / ladder / empty") — out of v1; emergent learning is the chosen mechanism.
- **Cross-tileset rule sharing** — out of v1; rules are scoped to their TileAtlas.
- **Multiple promotions per stroke surfaced as stacked toasts** — v1 shows the first; chains in v2 if needed.
- **Tileset slicer + Aseprite/Tiled/LDtk importers** — out of v1; separate asset-pipeline concern.

### Outside this product's identity

- Globally-shared rule library (community gallery for rule packs) — out (origin scope).
- Browser-based / mobile painter — out (matches idea #1 boundaries).

---

## Key Technical Decisions

- **Zero external dependencies.** Three candidates evaluated (cimgui-go toast libs, WFC/auto-tile libs, inspector framework libs); all rejected via the leverage doctrine. Total custom code ~370 LOC, well below wrap costs.
- **Schema rename TilemapLayer → TileAtlas with backward-compat unmarshaler.** Option (a) modified from the origin's three candidates: TileAtlas is the same struct shape as TilemapLayer plus reserved fields. Scene.TileAtlases replaces Scene.Tilemaps. Custom `Scene.UnmarshalJSON` migrates legacy `tilemaps` JSON keys at load. Save writes only `tile_atlases`. Migration is invisible to designers.
- **Widget registration = `pfcomponent.RegisterWidget(name, drawer)` + extend `applyPFTag` for `pf:"widget=name"` syntax.** New `WidgetCustom` kind; new `Field.CustomWidget` metadata field; new `case WidgetCustom:` arm in `inspector.renderField`. Drawer receives the typed component value, not just `values map[string]any`. The primitive is reusable for idea #4 (patch-cast surface) and idea #6 (dialogue-tree editor).
- **Painter lives in the inspector** (user decision Phase 0). Supersedes idea #1's plan U6's toolbar wiring for the tile-painter sub-modes. Toolbar keeps Select/Place/Delete/Paint top-level mode buttons; sub-mode picker (Brush/Bucket/Rect) and tile palette grid move to the inspector widget.
- **Auto-rule threshold bumped from 2 to 3** to align code with `docs/solutions/auto-tile-heuristic.md` invariant and the brainstorm's "third matching stroke" specification. Single-line constant change in `autotile.go:141`. Existing test `TestAutoTile_PatternPaintedTwicePromotesRule` is renamed and reframed for threshold 3.
- **Promotion-event hook on synth: `RecordStrokeWithPromotions`** returns `[]PromotedRule` for rules that crossed the threshold this stroke. Existing `RecordStroke` continues to work (`Painter.Paint` switches to the new API). Promotion fires once per rule per session; subsequent same-pattern strokes do not re-promote.
- **Toast = ImGui `OpenPopup` + `BeginPopup` near cursor; FocusManager-registered.** First popup in the editor; sets the convention for future toasts. Esc / click-outside dismiss (semantically equivalent to No).
- **Session-suppression in-memory map keyed by rule signature** `(Pattern, Output)` hash. Not persisted. Cleared on editor restart / scene reload. A "No" choice is sticky for the session only; the rule itself persists on the project file (so the synth re-promotes it on the next session and the designer gets a fresh chance to accept).
- **The pf-tag on the painter hook uses a synthetic field** (e.g., `Painter struct{} \`pf:"widget=tilepainter"\``) rather than overloading an existing field like `Grid`. Avoids conflict between the default 2D-int-array rendering of Grid and the custom widget; the synthetic field is purely an inspector hook point.
- **Production registration call site convention established:** `pixelforge_studio/editor/registrations.go` with package `init()`. Idea #1's `Sprite`/`Camera` registrations (currently missing per research) would follow the same pattern when those plans land.
- **No new schema field for "accepted" or "session-suppressed":** session-suppression is in-memory; "accepted" is implicit in `Count >= threshold`. Keeps `AutoTileRule` schema unchanged (origin decision).
- **Schema reservation `pf:` tags are set NOW** so future feature units (animated tiles, parallax) only add the code to populate the fields — the inspector renders them the moment they're nonzero. This is the leverage move R9 promises.

---

## Dependencies / Assumptions

- **Couples with idea #1's plan** (`docs/plans/2026-05-18-002-feat-screenroom-mario-strip-v1-plan.md`). Ships in the same release. Specifically:
  - Idea #1's plan U1 adds `SpriteSheetRef` to TilemapLayer — this plan renames the field's host type to TileAtlas, so idea #1's planned change applies to TileAtlas.
  - Idea #1's plan U6 (tile painter UI) is **superseded for the toolbar wiring** — the brush/bucket/rect handlers in `tile_painter.go` still land per idea #1, but they're invoked from this plan's U6 canvas dispatch (reading inspector state), not from a toolbar handler. Idea #1's plan U6's `tile_palette.go` file is folded into this plan's U5 `tilepainter_widget.go`.
  - Idea #1's plan U2-U5 (engine renderers + preview integration) are unaffected by this plan and proceed as planned.
- **Existing reflection inspector** from the ImGui migration U4 (`pixelforge_studio/editor/inspector.go`). The widget-registration extension (U2) hangs off the existing dispatch.
- **Existing `AutoTileRuleSynth`** continues to work; v1 of this plan changes only the activation threshold and adds the promotion-event hook. Pattern-matching algorithm unchanged.
- **Existing `pixelforge_project.AutoTileRule` schema** unchanged (per origin Key Decisions).
- **`docs/solutions/auto-tile-heuristic.md` invariants** are load-bearing: 3×3 + 3-reps + hint-only + introspectable. Plan must not violate any.
- **`docs/solutions/focus-manager-design.md`** — the toast registers with FocusManager; Esc dismisses top modal.
- **`docs/solutions/always-on-game-embedding.md`** — toast doesn't pause the game preview.
- **`docs/solutions/editor-pforge-schema-shape.md`** — additive omitempty + sanitize discipline applies to the reserved fields.
- **`docs/solutions/dirty-state-ux.md`** — schema changes route through `MarkDirty()`; toast Yes/No do NOT mark dirty (rule persistence is via the unchanged schema, not a separate save event).
- **Assumes idea #5's plan U2 (Archetype field)** is not blocked by this plan — they're independent.
- **Assumes idea #1's plan U1 (Scene grid + SpriteSheetRef)** lands before or alongside this plan's U1 so the schema additions don't conflict. Coordinate sequencing during execution.

---

## Risk Analysis & Mitigation

| Risk | Severity | Mitigation |
|---|---|---|
| Schema rename TilemapLayer → TileAtlas misses a caller; repo doesn't compile | Medium | U1's test suite + repo-wide grep before merge. Compiler errors are loud and immediate. |
| Legacy fixture migration breaks `editor.pforge` (the only real fixture) | Low | U1 test `TestScene_LegacyEmptyTilemapsMigratesToEmptyTileAtlases` is explicit. Fixture has empty tilemaps so migration is a no-op. |
| Threshold-3 change breaks user expectations from earlier ad-hoc testing of threshold-2 behavior | Low | The behavior was never user-exposed (no UI wired). The change aligns code with documentation; no users are depending on the old constant. |
| RegisterWidget's typed-state Drawer signature requires more reflection plumbing than estimated | Medium | The implementer may need to extend `Field` metadata to carry the parent component's typed pointer. If the existing `pfcomponent.Get(comp.Type)` returns enough metadata to reflect a typed instance from the component's Data map, the plumbing is straightforward. If not, U2 may need an extra reflection helper. Document the constraint in U2's approach. |
| Toast popup steals Esc from the game preview (game can't pause via Esc anymore) | Medium | FocusManager registration per `focus-manager-design.md` — Esc routes to top modal first. Toast counts as top modal while visible. After dismissal, Esc behavior reverts. Explicit test: `TestToast_FocusManagerEscPrecedence`. |
| Designer Yes-clicks a toast for a pattern they didn't actually want to auto-fill (mistake) | Medium | The auto-apply only happens on subsequent matching paints, not retroactively. Designer can paint the "wrong" pattern with a different tile + retain the rule in the project file. v2's rule-management inspector solves the longer-term version. Document this in scope-boundaries.dialed. |
| Promotion fires on patterns the designer intended as one-offs (not all 3 paints were "I want this everywhere") | Medium | Toast IS the consent mechanism — designer chooses No if it was coincidental. The 3-rep threshold reduces false promotions. Documented in `docs/solutions/auto-tile-heuristic.md` already. |
| Coupling with idea #1's plan U6 produces merge conflicts or contradicts during implementation | Medium | This plan's U6 explicitly supersedes idea #1's U6 toolbar wiring. Both plans should be executed in coordinated order: idea #1's U1-U5 first, then this plan's U1-U4, then this plan's U5-U7, then idea #1's U6/U7/U8/U9. Document the order in the milestone tracker. |
| TileAtlas's synthetic Painter hook field bloats the schema (extra zero-size field on every TileAtlas in JSON) | Low | The field is `struct{}`; JSON-marshals as `{}` or omits with `omitempty` — pick the latter via `json:"-"` tag (it's UI metadata, not persistence data). The pf-tag still functions for the inspector. |
| `Painter.Paint` (painter.go:67-71) currently silently substitutes auto-rule outputs; users who paint with "I want this exact tile" and get a rule-substituted tile may be confused | Low | This was existing behavior pre-v1; not changing it. The toast on promotion makes the rule's existence visible the moment it activates, which is the discoverability fix. If silent substitution becomes a complaint, v2 adds a "rule applied here" indicator. |

---

## System-Wide Impact

**New packages introduced:** none (extensions of existing packages).

**Modified packages:**
- `pfcomponent` — adds `WidgetCustom` kind, `RegisterWidget` API, `applyPFTag` extension, `Field.CustomWidget` metadata.
- `pixelforge_project` — `TilemapLayer` renamed to `TileAtlas`; reserved fields added; `Scene.Tilemaps` renamed to `Scene.TileAtlases`; custom UnmarshalJSON migration shim.
- `pixelforge_studio/palette` — `AutoTileRuleSynth` gains `RecordStrokeWithPromotions`; threshold bumped to 3; `Painter.Paint` switches to new API.
- `pixelforge_studio/editor` — inspector dispatch arm for WidgetCustom; new tilepainter widget; new toast popup; new session-suppression state; new accessors on `*Editor`; canvas's ToolPaint dispatch wired (co-owned with idea #1's plan U6); registrations.go establishes production-registration convention.

**Affected workflows:**
- **Designer authoring** — primary target. New: painter UI in inspector, auto-rule toast moments. Familiar: the brush/bucket/rect mechanics that idea #1 implements (just relocated).
- **Future feature implementers** — the RegisterWidget primitive is the new extension seam for custom inspectors. Idea #4 patch-cast surface and idea #6 dialogue-tree editor both register their own custom widgets via this API.
- **Schema migrations** — first non-trivial load-time migration in the project. Sets the precedent for future field renames or shape evolutions.
- **Auto-tile rule lifecycle** — the synth has been unwired until now. Wiring it changes what the designer sees the first time they paint a repeating pattern. Documentation impact: README / user docs (if any) should mention the auto-rule feature.

**Documentation impact:** Post-v1, three `docs/solutions/` entries are worth capturing:
1. The widget-registration extension pattern (how to add a custom inspector widget).
2. The promotion-event hook discipline (why the synth needs an event API rather than just silent observation).
3. The schema rename + load-time migration shim (pattern for future renames).

**Operational / rollout:** Standard release. No data migration required at runtime — the only existing populated `.pforge` file (`editor.pforge`) has empty `tilemaps`, so the migration is a no-op there. Synthetic fixtures (U8) exercise the migration logic in tests. Coupled release with idea #1 — both ship together as the v1 Mario-strip milestone.

---

## Notes for Implementer

**Coordination with idea #1's plan:**
1. Execute idea #1's plan U1-U5 (schema additions, engine renderers, preview integration) first — those are independent of this plan and unblock the canvas's ability to render tiles.
2. Then execute this plan's U1-U4 (schema reframe, widget registry, registration, synth hook) — these are the foundation layers.
3. Then execute this plan's U5-U7 (widget drawer, canvas dispatch, toast) — these integrate everything user-visible.
4. Then idea #1's plan U6/U7/U8/U9 in light of supersession: U6's `tile_painter.go` content still lands (handler logic), but the toolbar sub-mode picker and tile palette panel from U6 are NOT shipped (they live in this plan's U5 inspector widget instead). U7 (entity placement + Player marking + spawn) is unchanged. U8 (scene resize) is unchanged. U9 (E2E acceptance tests) is unchanged.

**If the user wants to update idea #1's plan to reflect the supersession explicitly:** edit `docs/plans/2026-05-18-002-feat-screenroom-mario-strip-v1-plan.md` U6's approach section to point at this plan, and remove the toolbar-sub-mode-picker bullet. Per ce-plan convention, plans don't carry per-unit state, so this is a clean text edit. Worth doing for clarity if execution is not happening immediately.
