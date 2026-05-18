---
date: 2026-05-18
topic: tileatlas-emergent-rules-v1
origin: docs/ideation/2026-05-18-pixelforge-nes-class-no-code-ideation.md (idea #2)
ships_with: docs/brainstorms/2026-05-18-screenroom-mario-strip-v1-requirements.md (idea #1)
---

# TileAtlas as Registered Component + Emergent Auto-Rules — v1

## Summary

v1 of idea #2 ships **alongside** idea #1's Mario-strip release as a coupled architectural + UX move. The tilemap moves from a direct `Scene.Tilemaps` schema field to a **registered component** rendered through the existing reflection inspector via a new tilepainter widget. The engine-side `AutoTileRuleSynth` (learning-by-example from paint strokes) wires into the painter; on rule promotion a small toast next to the brush offers "Auto-apply this pattern? [Yes / No]." No rule-management inspector and no LDtk-style semantic-IntGrid authoring in v1 — those land in v2+.

---

## Problem Frame

Idea #1's Mario-strip v1 ships explicit tile selection — designer picks a tile from the palette, paints into a cell, repeats. That works, but it's tedious for the long-period patterns NES levels are full of: 32-tile-wide ground strips, the inside of caves, repeating brick walls, the same corner-piece every time a wall meets a floor.

Pixelforge already ships `pixelforge_studio/palette/autotile.go` — an engine-side `AutoTileRuleSynth` that observes 3×3 paint patterns, promotes patterns to active rules on the 3rd matching stroke, and applies them on subsequent strokes. The schema (`AutoTileRule`) is reserved on `TilemapLayer`. **None of it is wired into the studio.** It exists, it works, no UI surface invokes it. The Mario-strip designer paints the same brick corner four times in a row instead of having the editor offer to auto-fill from the second occurrence.

Separately, the tilemap today is a special-case schema field (`Scene.Tilemaps []TilemapLayer`) sitting outside the project's established pattern of "everything is a registered component the reflection inspector dispatches against." That works for v1 of the Mario-strip but pays compounding cost as tile features land later — every new feature (animated tiles, parallax layers, slope flags, NES attribute-table colors, per-tile collision shapes) becomes its own bespoke editor surface instead of a struct field on a component.

This brainstorm scopes the smallest version of two coupled moves: surface the existing synth so designers benefit immediately, and reframe the tilemap onto the established component-registry primitive so future tile work is cheap to add.

---

## Actors

- **A1. Designer.** Paints the Mario-strip level using the v1 explicit-selection brush + paint-bucket + rectangle tools. Sees the auto-apply toast appear next to the brush after repeating a pattern; chooses Yes or No per session.
- **A2. Pixelforge Studio.** Hosts the reflection-driven inspector (the U4 surface from the ImGui migration); the new tilepainter widget runs inside that inspector. Coordinates the toast UX. Persists rule decisions to the project file.
- **A3. AutoTileRuleSynth (existing).** Engine-side observer in `pixelforge_studio/palette/autotile.go`. Watches paint strokes for 3×3 pattern repetition; promotes patterns to rules on third occurrence; auto-applies active rules on matching subsequent strokes.

---

## Key Flows

- **F1. Designer paints a repeating pattern and gets auto-fill offered**
  - **Trigger:** Designer paints a tile that completes a 3×3 neighborhood pattern matching one the synth has seen twice before
  - **Actors:** A1, A2, A3
  - **Steps:** (1) Designer paints the third matching cell; (2) `AutoTileRuleSynth` promotes the pattern to an active rule; (3) studio surfaces an inline toast next to the brush cursor: "Auto-apply this pattern? [Yes / No]"; (4) designer clicks Yes (or hits Enter) or No (or hits Escape / clicks outside); (5) toast dismisses
  - **Outcome:** If Yes — subsequent paint strokes that match the rule's 3×3 pattern auto-fill the resulting tile placement. If No — rule is stored but suppressed for the current session; designer continues painting manually.
  - **Covered by:** R1, R2, R3, R4, R5

- **F2. Designer benefits from a previously-accepted rule on a future session**
  - **Trigger:** Designer reopens a project where they previously accepted at least one auto-rule, and paints a pattern matching that rule
  - **Actors:** A1, A2, A3
  - **Steps:** (1) Designer paints a cell completing a matching 3×3 neighborhood; (2) synth recognizes the persistent rule (stored from prior session); (3) tile auto-fills without surfacing a toast (the rule is already accepted)
  - **Outcome:** Auto-fill happens silently; no UX interruption.
  - **Covered by:** R6, R7

- **F3. Designer adds a new tile feature in a later release (forward-looking)**
  - **Trigger:** A v2+ feature unit lands that needs new per-tile data (e.g., per-tile animation frames, parallax layer factor)
  - **Actors:** A2 (studio implementers, not the designer)
  - **Steps:** (1) Implementer adds a struct field on the registered `TileAtlas` component with a `pf:` tag declaring the widget kind; (2) reflection inspector renders the new field automatically; (3) codegen template handles the new field without bespoke logic
  - **Outcome:** A new tile feature shipped without building a new editor surface — the cost amortizes the v1 architectural reframe.
  - **Covered by:** R8, R9

---

## Requirements

**Architectural reframe**

- R1. The tilemap data structure is registered as a **component** through the existing `pfcomponent` reflection registry (the same primitive that powers the U4 inspector dispatch). Specific schema shape — whether it remains a scene-level field migrated to use the component metadata, or moves to an entity-level component — is a planning decision; the requirement is that the inspector pathway is the unified one.
- R2. The tile painter is rendered by the reflection inspector as a **widget for the TileAtlas component type** — not as a separate editor pane or workspace. Adding the painter requires the inspector to support a new widget-kind registration mechanism (the existing dispatch only knows built-in widget kinds); the simplest extension that supports this is in v1.
- R3. Existing projects (those with `Scene.Tilemaps []TilemapLayer` in the file) load cleanly under v1 — migration happens **automatically at project load time** per the established schema-first / additive-only discipline. Designer notices no behavior change.

**Emergent auto-rules — surfacing the existing synth**

- R4. The studio invokes `AutoTileRuleSynth` on each paint stroke completed by the designer. When the synth promotes a pattern to an active rule (third occurrence of the same 3×3 neighborhood), the studio surfaces a **small inline toast** next to the brush cursor reading "Auto-apply this pattern? [Yes / No]."
- R5. Toast interaction:
  - "Yes" / Enter → rule is marked active for the current session; subsequent matching paint strokes auto-fill the rule's tile placement.
  - "No" / Escape / click-outside → rule is stored in the project file (so a future session can re-promote it on a third match) but **suppressed for the current session** — no auto-fill applies until the session restarts or the designer re-paints the pattern enough times to re-trigger the toast.
- R6. Accepted rules **persist with the project**: a v1 project saved with two accepted rules opens with both rules active in any future session (and applies them silently — no toast on already-accepted rules).
- R7. Auto-fill happens **silently and inline** with painting: when the designer paints a cell that completes a matching 3×3 pattern under an active rule, the tile resolves to the rule's output tile without modal interaction, status text, or noticeable delay.

**Schema forward-compatibility**

- R8. The TileAtlas component schema reserves these fields as `omitempty` even though no v1 UI walks them: per-tile animation frames, parallax layer factor, slope/collision-class flags, NES attribute-table palette index per 2×2 background block. v1 ships with the inspector dispatch wiring in place so v2+ features only need to add struct fields + `pf:` tags.
- R9. The widget-registration extension to the reflection inspector (added in R2) is **the same primitive future feature units use** for their own custom widgets (e.g., the dialogue-tree editor in idea #6, the patch-cast surface in idea #4). Idea #2 v1 builds it once and leaves it usable.

---

## Acceptance Examples

- **AE1. Covers R4, R5.** Given a brand-new project where the designer paints a 3×3 brick-corner pattern for the third time, when the third paint stroke completes, an inline toast appears next to the brush reading "Auto-apply this pattern? [Yes / No]." When the designer clicks "Yes," the next time they paint a cell that completes the same 3×3 pattern, the tile is auto-filled without further prompting.
- **AE2. Covers R5.** Given a toast is showing, when the designer clicks outside the toast (or presses Escape), the toast dismisses with no auto-rule activation for this session. The rule is preserved in the project file for future sessions.
- **AE3. Covers R6, R7.** Given a project saved with one accepted auto-rule, when the designer reopens the project in a later session and paints a cell completing the matching 3×3 pattern, the tile is auto-filled silently — no toast, no notification.
- **AE4. Covers R3.** Given an existing project file using the pre-v1 `Scene.Tilemaps []TilemapLayer` schema, when the designer opens it in v1, the project loads, displays its tilemap correctly in the painter, and round-trips on save without data loss. Designer notices no migration step.
- **AE5. Covers R1, R2, R9.** Given a v2 unit adds a new per-tile field "animationFps" to the TileAtlas component with a `pf:"slider,0..30"` tag, when the developer compiles and opens the studio, the inspector renders the field as a slider under the TileAtlas inspector view automatically — no inspector code touched.

---

## Success Criteria

- **Designer outcome (immediate):** A designer painting a Mario-strip level notices that after they paint the third matching brick-corner (or grass-edge, or pipe-end) pattern, the studio offers to auto-fill the rest. Clicking Yes saves them from manually painting hundreds of corner tiles. They never learn the term "auto-rule" — the feature is the toast and what it does.
- **Designer outcome (returning):** A designer opening a project from a prior session sees their previously-accepted rules already at work — corners auto-fill without explanation, the painting feels "smart."
- **Architectural outcome:** The next person who adds a tile feature (parallax, animation, slopes, NES attribute table) writes a struct field + `pf:` tag and is done — they do not build a new editor pane. If they reach for a new editor, the leverage move failed.
- **Downstream handoff outcome:** Planning consumes this doc and does not need to invent painter UX, toast behavior, or component registration mechanism. Only implementation specifics (which Go file, which package, exact migration shape, exact widget dispatch interface) are open.

---

## Scope Boundaries

- **Rule-management inspector** (panel to toggle rules on/off, delete bad rules, edit rule patterns) — out of v1. Cost: a "No" decision is sticky for the session; the designer can't easily revisit it without re-painting enough strokes to re-trigger the toast. Lands in v2.
- **Semantic IntGrid painting** (5-color "ground / wall / spike / ladder / empty" brush over meaning rather than visual tiles) — out of v1. The whole LDtk-style author-time pattern is deferred; emergent learning is the v1 mechanism.
- **Author-time rule editor** (designer explicitly writes 3×3 patterns and their output) — out of v1. Whole branch deferred.
- **Tileset-author vs level-designer role distinction** — out of v1; one person, one role.
- **Animated tiles, parallax tile layers, slope / collision-class flags, NES attribute-table colors per 2×2 block** — out of v1, but the component reframe makes them cheap to add later. Schema reservations in R8.
- **Per-tile collision shape editor** (Tiled-style polygon collision per tile) — out of v1.
- **NES attribute-table preview** (visualization of 2×2 palette-block constraint) — out of v1; relates to idea #3 of the ideation, not idea #2.
- **Aseprite / Tiled / LDtk importers** — out of v1; separate asset-pipeline concern.
- **Multi-layer tilemaps** (background + collision + decoration as separate paintable layers in the same scene) — out of v1. One TileAtlas per scene; multi-layer becomes a v2 feature once the component reframe makes adding more component instances cheap.
- **Cross-tileset rules** (a rule learned on tileset A also fires on tileset B) — out of v1; rules are scoped to their TileAtlas.
- **Globally-shared rule library** (designer downloads rule packs from a community gallery) — out of product identity, not just out of v1.

---

## Key Decisions

- **Emergent rules over LDtk's author-time rules.** Designer audience is not pre-trained on game-editor metaphors; asking them to write a rule grammar before painting would block onboarding. Emergent learning fits the existing engine code and matches the prior project decision in `docs/solutions/auto-tile-heuristic.md` ("the 3-repetition threshold; rules are hints, on-disk grid is source-of-truth").
- **Toast confirm over silent auto-apply.** Silent auto-apply (option A from the brainstorm menu) was smaller, but discoverability matters for a no-pre-training audience. The toast is the moment the designer learns the feature exists, in context. The cost (one moment of attention per new rule promotion) is worth it.
- **Toast over rule-management inspector for v1.** A full inspector would solve more problems (debug bad rules, re-enable "No"-suppressed rules) but is significantly more UX surface. The toast is the smallest cut that makes the feature visible and consensual.
- **Architectural reframe IN v1, not deferred.** The reframe pays its leverage forward — every future tile feature is cheap once it lands. Deferring it means the first v2 feature to land (likely animated tiles or parallax) pays the reframe cost anyway, with the additional friction of changing v1's already-working surface. Land it once, alongside the minimal UX it enables.
- **`AutoTileRule` schema unchanged.** The existing storage shape works; v1 only adds the inspector wiring + toast UX. Reduces risk of breaking the existing engine-side behavior.
- **Couple the release to idea #1.** Mario-strip + emergent auto-rules ship together as one milestone. They reinforce each other (auto-rules make Mario-strip painting practical; Mario-strip is the first surface that exercises auto-rules).

---

## Dependencies / Assumptions

- **Depends on idea #1 v1 (ScreenRoom Mario-strip)** shipping in the same release. Without the painter from idea #1, idea #2's auto-rule UX has no surface to attach to.
- **Depends on the existing U4 reflection inspector** (post-ImGui migration). The widget-registration extension (R2) is built on top of the existing `pfcomponent.WidgetKind` dispatch.
- **Depends on `pixelforge_studio/palette/autotile.go`** continuing to work as it does today. v1 of this brainstorm does not modify the synth's pattern-matching algorithm or rule-promotion threshold.
- **Depends on `pixelforge_project.AutoTileRule`** schema field on `TilemapLayer` (verified to exist). Persistence of accepted rules rides this field.
- **Assumes the schema migration from `Scene.Tilemaps []TilemapLayer` to the new registered-component shape is automatic at load time** — fits the project's established additive-only schema discipline. Migration design is a planning concern.
- **Assumes the toast UX is achievable inside the ImGui surface** without modal-popup blocking gestures. ImGui's `OpenPopup` + `BeginPopup` pattern should support this; planning verifies.

---

## Outstanding Questions

### Resolve Before Planning

- *(none — scope is fully resolved at the brainstorm level)*

### Deferred to Planning

- **[Affects R1] [Technical]** Exact schema shape after the reframe. Candidates: (a) `Scene.TileAtlases []TileAtlas` field with each TileAtlas registered through pfcomponent so the inspector dispatches on it; (b) TileAtlas becomes an `EntityComponent` on a scene-level "tilemap entity"; (c) keep `Scene.Tilemaps` as the source-of-truth slice but wire the inspector to render each layer via the registered component's metadata. Each preserves observable behavior; planning picks the cleanest fit.
- **[Affects R2, R9] [Technical]** Exact interface for the widget-registration extension. Candidates: a new `pfcomponent.RegisterWidget(kind, drawer)` registry parallel to `pfcomponent.Register`, a `WidgetKind` enum extension with a registry of renderers, or a tag-only approach where the renderer is selected by `pf:"widget=..."` parsing. Planning picks the smallest surface that supports R9's claim of reusability for future custom widgets.
- **[Affects R4] [Technical]** ImGui mechanism for the inline toast. Candidates: an `OpenPopup` near the cursor that auto-dismisses on click-outside; a custom-drawn floating window via `imgui.GetWindowDrawList().AddRectFilled`; reuse of a generic "toast" widget if one exists. Planning verifies which produces the right interaction model.
- **[Affects R5] [Technical]** How "suppressed-for-this-session" state is tracked. In-memory map keyed by rule ID is the obvious shape; planning confirms it doesn't need persistence (the design says it doesn't).
- **[Affects R3] [Needs research]** Whether any existing `.pforge` files in `pixelforge_studio/editor/cart_assets/` or example games (`/snake`, `/pacman`, etc.) carry `Scene.Tilemaps` content that needs migration testing. Planning verifies and writes round-trip tests.
