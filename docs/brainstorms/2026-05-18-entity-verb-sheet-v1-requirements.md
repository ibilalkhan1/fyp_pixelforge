---
date: 2026-05-18
topic: entity-verb-sheet-v1
origin: docs/ideation/2026-05-18-pixelforge-nes-class-no-code-ideation.md (idea #5)
---

# Entity Verb-Sheet + Input Intents + Blackboard State — v1

## Summary

v1 ships three coupled foundations: the entity character-sheet inspector (per-archetype templates with trigger slots; designer picks named verbs from a closed enum), an input-intent layer (named intents like `input/jump` mapped to physical keys/gamepad via a Project-level InputMap), and a global blackboard state store (a single shared key-value map exposed as `pievent.Target("state/*")`). Verbs are curated recipes wrapping the existing `catalog.RegisterAction` infrastructure with default parameters baked in — no parallel engine surface. Entities pick from six archetypes (Player / Enemy / Pickup / Hazard / NPC / World). The character sheet replaces explicit script/event-lane authoring for the no-code surface; the existing catalog stays as the power-user fallback.

---

## Problem Frame

Pixelforge has all the engine infrastructure needed to express game behavior. `pixelforge_studio/scripting/catalog/catalog.go` ships `RegisterStep`, `RegisterCondition`, `RegisterAction` registries. `pixelforge_event` is a zero-alloc pub/sub bus. `pixelforge_routine` is a coroutine Step sequencer. The reflection inspector (post-U4 of the ImGui migration) dispatches per-field widget rendering. The M5 plan reserved `BehaviorGraph`, `StepNode`, `EventSheetRule`, `Condition`, `Action` schema shapes back in 2026-05-16 — these are populated but no studio UI walks them at the level a non-coder can use.

The current scripting workspace renders a Step lane editor (post-ImGui rewrite in U8). Designers see a horizontal strip of step cards, can add steps from a kind-picker, can move steps left/right, can delete. That works for power users authoring behavior coroutines — but it's the wrong surface for the audience this product targets (designer + classmates community, not pre-trained on game-editor metaphors). The step lane shows ordering, parameters, and pub/sub topics as first-class concepts; the audience doesn't know what those *are*.

The verb-sheet is the alternative authoring surface — entity-shaped instead of script-shaped. A Goomba *is* its verb list: "When stepped on: die. When player nearby: chase." No script panel, no event lane, no parameters visible by default. The character-sheet metaphor maps to something the audience already knows (D&D character sheets, RPG status screens). The catalog stays as the power-user surface; the verb-sheet rides on top of it as curated recipes.

This brainstorm scopes v1 of three coupled primitives: the character-sheet inspector, an input-intent indirection layer (so verbs reference `input/jump` not `KeyZ`), and a global blackboard state store (so verbs can read/write shared state like score, HP, inventory). Together they form the foundation that ideas #1 (Player tag for the ScreenRoom camera-follow) and #4 (event topics for audio bindings) implicitly depend on.

---

## Actors

- **A1. Designer.** Authors entity behavior by picking archetype + adjusting verbs. Doesn't see parameters, doesn't see catalog Steps, doesn't see pub/sub topics. Knows what a "Goomba" should do; the studio gives them named verbs to express it.
- **A2. Power-user designer.** Same person at later experience; drops to the existing catalog Step / Action surface (via "Show advanced" toggle per verb) when the curated recipes don't fit. Edits InputMap, inspects blackboard state.
- **A3. Pixelforge Studio.** Renders the character-sheet inspector via the reflection inspector's archetype-aware template dispatch; compiles verb-sheet bindings to underlying EventSheet/Action runtime entries; surfaces the InputMap settings + blackboard inspector.
- **A4. Pixelforge engine.** Existing `catalog.RegisterAction`, `pievent.RegisterTarget`, `pixelforge_event`, `pixelforge_routine`, `pixelforge_key` + `pixelforge_pad` packages. v1 of this brainstorm adds the input-intent layer (as a `pievent.Target` registration) and the blackboard (another `pievent.Target`), but doesn't modify existing engine code.

---

## Key Flows

- **F1. Designer authors a Goomba enemy from scratch**
  - **Trigger:** Designer drops a "Goomba" sprite into a level and selects it
  - **Actors:** A1, A3
  - **Steps:** (1) Inspector opens; archetype dropdown at the top defaults to `Enemy`; (2) Enemy template surfaces trigger slots: `when stepped on`, `when shot`, `when player nearby`, `when scene starts`, `every tick`, `when damaged`, `when destroyed`; (3) `when stepped on` slot is pre-filled with `die` (Enemy default); designer accepts; (4) `when player nearby` is empty; designer picks `chase_player` from the verb dropdown; (5) saves
  - **Outcome:** Goomba dies when stepped on, chases the player when nearby. No script lane opened, no catalog visited, no pub/sub topic configured.
  - **Covered by:** R1, R2, R3, R4

- **F2. Designer wires the Player character with input intents**
  - **Trigger:** Designer adds the Player entity (marked Player via idea #1's R8) and configures movement
  - **Actors:** A1, A3, A4
  - **Steps:** (1) Inspector opens; archetype = `Player`; (2) Player template surfaces trigger slots: `when input/up`, `when input/down`, `when input/left`, `when input/right`, `when input/jump`, `when input/attack`, `when damaged`, `when scene starts`, `every tick`; (3) input/jump slot pre-filled with `jump` verb (Player default); designer accepts; (4) input/left, input/right pre-filled with `move_with_intent`; (5) designer adds `when damaged: take_damage(1)`
  - **Outcome:** Player jumps on the Z key, moves with arrows, loses 1 HP when damaged. The Z key is decided by the Project's InputMap, not by the Player's verb-sheet — remapping at any time updates Player behavior automatically.
  - **Covered by:** R5, R6, R7, R10

- **F3. Designer adds a score system**
  - **Trigger:** Designer wants the Player to earn points when enemies die
  - **Actors:** A1, A3, A4
  - **Steps:** (1) Designer opens any Enemy entity inspector; (2) finds `when destroyed` slot (empty by default for most enemies); (3) picks `give_points` verb; (4) hovers "Show advanced" → can adjust the score amount from the verb's default 10; (5) saves
  - **Outcome:** Every enemy of this type gives 10 points when destroyed. The score lives in the blackboard at key `score`; HUD overlay (idea #3 or similar) reads from `state/score`.
  - **Covered by:** R3, R8, R11

- **F4. Designer remaps a key**
  - **Trigger:** Designer wants `input/jump` to bind to Space instead of Z
  - **Actors:** A1, A3
  - **Steps:** (1) Designer opens Settings → Input panel; (2) sees the InputMap table — each intent in a row, with its current physical inputs listed; (3) finds `input/jump` row; (4) clicks the keyboard column dropdown, picks "Space"; (5) all verbs referencing `input/jump` now respond to Space instead of Z, with no change to entity verb-sheets
  - **Outcome:** Single edit, system-wide effect. No entity needs reconfiguration.
  - **Covered by:** R5, R6, R7

---

## Requirements

**Character sheet & archetypes**

- R1. Each `Entity` carries an **archetype tag** field (`omitempty`) with one of six values: `Player`, `Enemy`, `Pickup`, `Hazard`, `NPC`, `World`. Default for unset (including projects saved before v1) is `World`.
- R2. Each archetype defines a **trigger-slot template** — a fixed list of trigger types that appear in the inspector for entities of that archetype. v1 ships templates for all six archetypes (concrete slot lists are a planning detail; the requirement is that templates exist per-archetype and are baked into v1).
- R3. The Inspector renders a **character-sheet layout** for each entity: archetype dropdown at the top, trigger slots from the archetype template below, each slot showing a verb-pick dropdown. Slots are pre-filled with the archetype's default verb (where one is defined) so newly-dropped entities of common archetypes (Goomba-style Enemy, coin-style Pickup) already work without configuration.
- R4. Each slot has a **"Show advanced" toggle** that drops to the existing catalog Action parameter form for the picked verb. Designers who want to tweak baked-in defaults (e.g., change `give_points` from 10 to 50) use this; designers who don't never see parameters.

**Verbs as catalog recipes**

- R5. A **verb** is a named recipe wrapping a `catalog.RegisterAction` Action with default parameter values baked in. v1 ships approximately 25-30 curated recipes covering arcade essentials. (Concrete recipe list is a planning detail; the requirement is that recipes are the unit of the closed enum the designer picks from.)
- R6. The verb dropdown on each trigger slot lists **all v1 recipes filtered by relevance to the trigger context** (e.g., `when stepped on` shows damage / score / spawn recipes, not motion recipes). Filtering is per-trigger metadata in the recipe definitions; if a designer wants an unusual cross-trigger verb, they drop to advanced.

**Input intents**

- R7. The Project schema adds a new additive field `InputMap` mapping **named intents** to lists of **physical inputs** (keyboard keys + gamepad buttons). v1 ships approximately 9 intents covering NES D-pad + buttons: `input/up`, `input/down`, `input/left`, `input/right`, `input/jump`, `input/attack`, `input/use`, `input/menu`, `input/start`.
- R8. Sensible **default key mappings** ship out-of-box (e.g., `input/jump` → Z + Spacebar + GamepadA; `input/up` → ArrowUp + W + DpadUp). Designers can edit via Settings → Input panel; each intent's row shows current bindings with dropdown pickers for each modality.
- R9. Verbs reference **intent names, never raw keys**. Verb slots like `when input/jump` resolve at runtime through the InputMap. Remapping the InputMap takes immediate effect with no change to verb-sheets.

**Blackboard state**

- R10. The engine exposes a **global blackboard** as a single `pievent.Target("state/*")` wrapping a `map[string]any`. Writes publish `state/changed:<key>` events; reads return the current value. The blackboard is the canonical source for shared mutable state (score, HP, inventory, flags).
- R11. v1 surfaces **reserved canonical keys** for arcade primitives — `score`, `lives`, `health`, `current_scene`, `player_x`, `player_y` — with their canonical types known to the studio. The blackboard inspector (a dockable panel) shows these keys at the top with typed widgets (int sliders for score, integer field for lives, etc.); designer-added arbitrary keys appear below as a flat string→value list.
- R12. Verbs that read/write the blackboard (`give_points`, `lose_life`, `set_flag`, `check_flag`, `give_item`) treat it as a flat key-value store. No nested scopes (per-scene, per-entity, per-player) in v1.
- R13. The blackboard's value map is the **serialization unit for save/load** — saving captures the current blackboard, loading restores it. v1 supports the underlying primitive; save/load UI (named slots, autosave UX) is deferred to idea #6.

**Schema additions (all additive `omitempty`)**

- R14. New schema fields: `Project.InputMap` (intent name → physical input list), `Entity.Archetype` (string from the six-archetype enum), `Entity.VerbBindings` (trigger name → verb recipe name + optional parameter overrides). Projects saved before v1 load cleanly with empty maps and the `World` archetype default.

**Catalog reuse + compilation**

- R15. Each verb-sheet `(trigger → verb)` binding **compiles at load time to a single underlying EventSheetRule** in the existing M5 schema. The verb-sheet is an alternate inspector view over the same data; power users can open the existing event-sheet view to see (and edit) the underlying rules. Round-trip preserves verb-sheet shape when bindings match the recipe form.

---

## Acceptance Examples

- **AE1. Covers R1, R3.** Given a fresh project, when the designer drags an "Enemy" sprite into the scene and selects it, the Inspector shows: archetype dropdown set to `Enemy`, trigger slots `when stepped on / when shot / when player nearby / when scene starts / every tick / when damaged / when destroyed`, with `when stepped on` already showing `die` as its pre-filled verb. No script lane is opened.
- **AE2. Covers R4.** Given a Pickup entity with `when touched: give_points` (baked default 10), when the designer clicks the "Show advanced" toggle next to that slot, an inline parameter form appears showing the underlying Action's parameter (e.g., `amount: 10`). The designer changes the value to 100 and saves; the next time the verb fires, 100 points are added.
- **AE3. Covers R5, R6.** Given a Hazard entity's `when touched` slot, when the designer opens the verb dropdown, the list shows damage-shaped verbs (hurt_player, take_damage, destroy_other, change_scene) but NOT motion verbs (move_with_intent). When the designer drops to advanced view and opens the full catalog, all Actions are visible.
- **AE4. Covers R7, R8, R9.** Given the default InputMap with `input/jump` → [Z, Space, GamepadA] and a Player entity with `when input/jump: jump`, when the designer plays the level and presses Space, the Player jumps identically to pressing Z. When the designer edits Settings → Input and removes Space from input/jump, pressing Space no longer triggers the verb (Z still does); no Player entity edit was needed.
- **AE5. Covers R10, R11, R12.** Given a Player entity with `when damaged: take_damage(1)` and the blackboard's `health` key set to 3, when the player is damaged three times, the blackboard's `health` key transitions 3 → 2 → 1 → 0. The blackboard inspector panel shows the live value updating with each hit.
- **AE6. Covers R14.** Given a `.pforge` file saved before v1 with no `Archetype` field on any entity, when the file loads in v1, every entity is assigned `Archetype: "World"` (with the World template's minimal trigger slots) and no behavior changes from the pre-v1 state.
- **AE7. Covers R15.** Given a verb-sheet binding `Goomba.when_stepped_on = die`, when the designer opens the existing scripting workspace's event-sheet view, a single EventSheetRule appears matching the binding (trigger: collide_top, action: destroy_self). When the designer edits the rule in the event-sheet view to also publish a `player_score+=10` event, opening the Goomba inspector again shows the `when stepped on` slot now displaying "advanced (composed)" instead of the named verb — because the rule no longer matches a single named recipe.

---

## Success Criteria

- **Designer outcome:** A first-time designer with no prior game-editor experience opens Pixelforge, drops a Goomba sprite, sees it pre-configured as "Enemy" with `when stepped on: die` already set, and within five minutes has authored a working Mario-class scene where the player jumps, lands on enemies to defeat them, and collects coins to score points — without opening a script lane, learning what a "trigger" is, or seeing a single parameter.
- **Power-user outcome:** A designer who has built three games and wants behavior outside the curated recipes opens "Show advanced" on a verb, sees the underlying catalog Action with its parameters, edits them, and ships. Without the verb-sheet they can also build behavior directly via the existing scripting workspace event-sheet view (R15).
- **Remap outcome:** A designer changes a single InputMap entry and the change cascades to every entity that referenced that intent — no rewriting individual verb-sheets.
- **State outcome:** A designer adds a score system by picking `give_points` on an enemy's "when destroyed" slot and a HUD entity reading `state/score` — they never write a variable, declare a type, or hand-wire pub/sub.
- **Downstream handoff outcome:** Planning consumes this doc and does not need to invent verb recipes, archetype templates, intent names, blackboard structure, or compilation semantics. Only implementation specifics (exact recipe parameter shapes, exact ImGui inspector layout, exact `pievent.Target` registration interface for the blackboard) are open for planning.

---

## Scope Boundaries

- **Designer-authored verbs.** v1 verbs are bundle-only — designers cannot create new recipes through the studio. New verbs require a code change (`catalog.RegisterAction` registration plus a verb-sheet recipe wrapper). Community / user-defined verbs are deferred.
- **Trigger composition.** Each archetype's trigger slots are fixed; designers can't compose multi-condition triggers ("when stepped on AND while flag X is true"). One trigger, one verb per slot in v1. Compound triggers require the event-sheet escape hatch.
- **Multi-verb slots.** Each slot holds exactly one verb. "When stepped on: die AND give_points" requires the library to ship a `die_and_score` recipe, or the designer uses the event-sheet escape hatch.
- **Conditional verb selection.** No "when stepped on by Player → die; when stepped on by Bullet → bounce" without dropping to the event-sheet view. Per-other-entity conditions are v2 or escape-hatch.
- **Verb-sheet for nested entities.** Each entity has one sheet; child-entity hierarchies are not visualized. Schema preserves them if they exist; UX treats entities as flat.
- **Multi-blackboard scopes.** Per-scene, per-entity, per-player (splitscreen) blackboards out. One global blackboard in v1; all state shares the namespace.
- **Custom archetypes.** Six archetypes are fixed; designers can't define new ones in v1. A new archetype = new code (template definition + default verbs).
- **AI behavior trees / utility AI / state machines on entities.** Verb-sheet is intentionally flat (trigger → verb); no nested state machines. Designers wanting complex AI drop to the catalog event-sheet view.
- **Network / multiplayer state sync** for the blackboard. Out of product identity.
- **Visual debugging of running state** (real-time blackboard inspector during preview with time-travel scrubbing). The static blackboard inspector ships in v1 (R11); time-travel scrubbing is deferred.
- **Intent-capture-by-pressing-the-key UX.** v1's Settings → Input panel uses dropdown pickers; "press the key you want to bind" capture is polish for v2.
- **Save/load UI** (named slots, autosave). The blackboard *is* the save unit (R13) but the UI for managing saves is idea #6's brainstorm.
- **Per-player input maps** (player 1 uses gamepad 0, player 2 uses gamepad 1). Single InputMap in v1; splitscreen not supported.

---

## Key Decisions

- **Verbs as catalog recipes, not new opaque enums.** Reuses the existing `catalog.RegisterAction` infrastructure. Smaller engine surface, single source of truth for behavior, clear path between no-code and power-user views.
- **Per-archetype templates, not universal slots or add-as-needed lists.** Discoverability for the audience (designer not pre-trained). Pre-wired defaults mean a freshly-dropped Goomba already does the right thing; designer learns by mutation, not by assembly.
- **All three sub-deliverables in v1.** Verb-sheet without intents = brittle (remapping requires rewriting verbs). Verb-sheet without blackboard = unable to author games with shared state (no score, HP, lives). The bet (no-code authoring works end-to-end) is only proven when all three ship.
- **Six archetypes (Player / Enemy / Pickup / Hazard / NPC / World).** Covers the entity types in every NES-class reference game. World is the default for unset, so existing projects load cleanly. Custom archetypes deferred — adding one is a code change, not a designer choice.
- **Catalog event-sheet as the escape hatch, not a parallel system.** v1 verb-sheet bindings compile to existing EventSheetRules; power users can drop to the existing event-sheet view to see and edit the underlying rules. No parallel runtime, no schema migration when a designer drops to advanced.
- **Single global blackboard, not multi-scope.** Score / lives / health are conceptually global. Per-scene state is rare in NES-class arcade games (most save the global counters and load a fresh scene). Multi-scope adds significant complexity for diminishing returns; deferred.
- **Reserved canonical keys** for arcade primitives. The blackboard knows that `score`, `lives`, `health` are typed integers so the inspector can render them with the right widget and verbs can validate parameters. Arbitrary keys still work via a string→value fallback.
- **Settings → Input panel with dropdowns, not capture-style.** v1 picks the simpler edit UX; capture-by-pressing-the-key is polish for v2. The intent layer itself (the indirection between verb names and physical inputs) is the load-bearing piece, not the capture UX.
- **InputMap defaults shipped with v1.** Sensible NES-style defaults (Z = jump, X = attack, arrows = directional, Enter = start, Escape = menu) plus gamepad bindings. Designer can edit but doesn't have to.

---

## Dependencies / Assumptions

- **Depends on the existing `catalog.RegisterAction` / `RegisterCondition` infrastructure** in `pixelforge_studio/scripting/catalog/catalog.go`. v1 wraps these as named recipes; doesn't modify the underlying mechanism.
- **Depends on the existing `pievent` pub/sub bus + `pievent.RegisterTarget` + `Inspectable` pattern**. Input intents and the blackboard both ride this pattern (each is a `pievent.Target`).
- **Depends on the existing reflection inspector** (U4 of the ImGui migration). The character-sheet layout is rendered by the inspector's archetype-aware dispatch.
- **Depends on the existing M5 plan's `BehaviorGraph` / `EventSheetRule` schema** for the under-the-hood compilation target (R15). Without it, v1 needs to introduce its own runtime semantics.
- **Depends on `pixelforge_key` + `pixelforge_pad` packages** continuing to provide physical key + gamepad input. The intent layer wraps these; doesn't replace them.
- **Idea #1 (ScreenRoom Mario-strip)** depends on idea #5's Player archetype for the camera's "follow this entity" target. v1 of #5 ships the Player archetype + the Player tag mechanism (`Entity.Archetype = "Player"` is the tag).
- **Idea #4 (audio library picker)** depends on idea #5's verb library to fire audio bindings. Specifically, `play_sound` verb on entity triggers publishes an event topic that #4's bindings table maps to a patch.
- **Idea #6 (RPG schemas + dialogue + menu + save UI)** depends on idea #5's blackboard for save state and on the existing catalog Action mechanism for dialogue triggers. v1 of #5 ships the blackboard primitive; #6 builds the save UI on top.
- **Assumes the ImGui inspector** can render an archetype-templated layout (a different field-set for entities tagged Player vs Enemy etc.) on top of the existing `pf:` tag dispatch. The U4 reflection inspector dispatches per-field; the archetype-templated dispatch is a per-entity-type extension. Planning verifies the cleanest way to express this.
- **Assumes `pievent.Target` supports** wrapping a `map[string]any` with `Inspectable` (for the blackboard inspector to enumerate keys). The existing convention is per-type targets; the blackboard is a flat map target. Planning verifies the existing Inspectable contract supports this.

---

## Outstanding Questions

### Resolve Before Planning

- *(none — scope is fully resolved at the brainstorm level)*

### Deferred to Planning

- **[Affects R2] [Needs design]** Concrete trigger-slot list per archetype (Player's slots, Enemy's slots, Pickup's slots, etc.). Planning should enumerate the slot list per archetype and validate by walking the NES-class reference set ("can a Mario clone be built using only Player + Enemy + Pickup + Hazard archetype slots? a Zelda screen with NPC?").
- **[Affects R5, R6] [Needs design]** Concrete v1 verb recipe list — name, underlying catalog Action, default parameter values, per-trigger relevance metadata. ~25-30 recipes; planning produces the draft list grouped by category and validates each maps cleanly to an existing catalog Action.
- **[Affects R3, R4] [Technical]** ImGui layout shape for the character sheet. Candidates: a single tall vertical inspector with collapsible "Show advanced" rows per slot; a tabbed inspector (one tab per trigger-category); a fixed-grid layout. Planning picks the most compact for "one screen, no scroll" — the prior brainstorm flagged that Final Fantasy-class entities with many stats may strain this.
- **[Affects R7, R8] [Technical]** Exact `InputMap` schema shape. Candidates: flat map (`map[string][]InputBinding` where InputBinding wraps key + gamepad + mouse); nested by intent + by modality (`map[string]struct{Keys []string; GamepadButtons []string}`); single string-list per intent with prefix encoding ("kb:Z", "gp:A"). Planning picks the cleanest round-trip through `omitempty`.
- **[Affects R10, R11] [Technical]** Whether the blackboard's `state/*` target uses one `pievent.Target[any]` that publishes `state/changed:<key>` events, or per-key `pievent.Target[T]` registrations (`state/score`, `state/lives`, ...) with the canonical-keys being the registered set. Either delivers the same observable behavior; planning picks the cleaner Inspectable surface.
- **[Affects R15] [Technical]** Exact compilation semantics from verb-sheet bindings to `EventSheetRule` instances. How does load-time compilation handle verb recipe lookup (recipe by name → underlying Action)? How does round-trip detect when a rule no longer matches a named recipe (so the inspector shows "advanced (composed)")? Planning resolves with reference to the M5 plan's existing rule shape.
- **[Affects R12] [Needs research]** Whether designer-added blackboard keys (outside the canonical reserved set) should have type metadata (string / int / bool / list) tracked in the project file or be untyped (string-value only). Untyped is simpler but loses validation; typed adds schema surface. Planning picks based on real verb-recipe usage.
- **[Affects R13] [Needs research]** What's the minimum save/load API surface needed for v1's blackboard persistence to be usable from the verb library (e.g., a `save_game` verb)? Planning verifies whether a no-UI primitive is enough or if some skeleton save-slot logic must land alongside.
