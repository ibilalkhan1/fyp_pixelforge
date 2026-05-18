---
title: "feat: Entity verb-sheet + input intents + blackboard state (idea #5 v1)"
type: feat
status: active
date: 2026-05-18
depth: deep
origin: docs/brainstorms/2026-05-18-entity-verb-sheet-v1-requirements.md
ideation: docs/ideation/2026-05-18-pixelforge-nes-class-no-code-ideation.md (idea #5)
---

# feat: Entity verb-sheet + input intents + blackboard state (idea #5 v1)

## Summary

v1 ships three coupled foundations on top of the existing Pixelforge infrastructure (catalog registries, pievent bus, reflection inspector, EventSheetRule runtime): a global blackboard (single `pievent.Target` wrapping `map[string]any`, ~60 LOC), an input-intent layer (~9 named intents mapped to physical keys/gamepad via a new `Project.InputMap` slice; ~200 LOC), and the entity character-sheet inspector with per-archetype trigger-slot templates + curated verb recipes that compile to `EventSheetRule` entries at load time. Six archetypes, ~25-30 verbs. Zero new external dependencies; ports the multi-binding pattern from `quasilyte/ebitengine-input` for reference but builds native on existing `pixelforge_key`/`pixelforge_pad`/`pixelforge_event` substrate.

---

## Leverage Doctrine (applies to all seven brainstorms)

The user's directive: *evaluate before depending; port a pattern rather than wrap a library when the wrapper exceeds the from-scratch cost; build custom when in-repo primitives already cover 80%; leverage when the library is mature, narrow, and the integration cost stays below the from-scratch cost.*

The Phase 1 research for this plan (ideation #5) evaluated four leverage candidates and rejected all of them:

| Candidate | Status | Verdict |
|---|---|---|
| `quasilyte/ebitengine-input` (MIT, v0.9.1) | Stale (no commits 17 mo); colliding `Key` enum with `pixelforge_key`; bypasses `pievent` | **Port pattern, do not depend** |
| Ebitengine `inpututil` + `pixelforge_key/pad` | Already in tree; supports gamepad + chord modifiers | **Foundation; build native** |
| Go visual-scripting libs (FSM / behavior tree) | Wrong abstraction for verb-sheet (dispatch table, not state machine) | **Skip** |
| Reactive state / blackboard libs | None worth a dep; `pievent` already covers the substrate | **Build native (~60 LOC)** |

Plans for the other six brainstorms (idea #1 ScreenRoom, #2 TileAtlas, #3 NES palette, #4 Audio library, #6 RPG-class, #7 Project Capsule) must apply the same discipline. Specifically:
- **Idea #2 (auto-tile)**: existing `pixelforge_studio/palette/autotile.go` already covers the synth; reject any "let's use a tilemap framework" temptation.
- **Idea #3 (palette quantizer)**: existing `pixelforge_studio/palette/quantize.go` is the foundation; do not import a color-quantization library.
- **Idea #4 (audio library)**: existing `pixelforge_audio` Paula mixer is sufficient; reject any "let's wrap a synth library" move — the patches are pre-rendered WAVs.
- **Idea #6 (RPG-class)**: dialogue parser is hand-rolled recursive descent (~200 LOC); menu templates are pure ImGui; reject any "let's use a dialogue framework" or "let's port RPG Maker's runtime" temptation.
- **Idea #7 (Capsule + Build)**: the only real leverage opportunity is `goversioninfo` for Windows .syso icon embedding — a single-purpose Go library, mature, widely used. Evaluate at planning time; likely accept. Everything else (WASM bundling, .icns generation, build-pane UX) is native ImGui + Go shell-outs.

The discipline is **not** "always build from scratch." It's **always evaluate, reject when wrapping costs exceed building, leverage when the math is positive.** Document the evaluation in each plan's Phase 1 research summary even when the answer is "leverage nothing" — so the next reviewer sees the work was done.

---

## Problem Frame

Pixelforge's engine has all the substrate needed to author no-code gameplay logic — `catalog.RegisterStep/Condition/Action` registries, `pievent` pub/sub bus with `Inspectable` sidecar, `pixelforge_routine` coroutine sequencer, `pixelforge_key`/`pixelforge_pad` input packages publishing events, the U4 reflection inspector dispatch in `pixelforge_studio/editor/inspector.go`, and the `EventSheetRule` schema (M5 reservation) waiting for runtime consumption. The studio's scripting workspace renders a Step lane editor (U8 ImGui rewrite). All of that works for power-users.

It does not work for the brainstorm's stated audience: friends and classmates with no game-editor pre-training. A Step lane shows ordering, parameters, and pub/sub topics as first-class concepts; designers without that background read it as code-shaped, freeze, and stop. The existing inspector renders pfcomponent fields generically but doesn't surface entities as *characters* with roles (Player vs. Enemy vs. Pickup) and trigger-slotted behavior (when stepped on / when shot / when player nearby).

Three coupled gaps compound the problem:
- **No input indirection.** Verbs would have to reference physical keys (`KeyZ`). Remapping or supporting gamepad means rewriting every verb that touches input.
- **No shared mutable state.** Score, lives, health, inventory have no canonical home. Each entity would have to invent its own per-instance state, with no way to read across entities or persist for saves.
- **No reduction of the catalog to "named verbs."** A designer who picks `publish_event(target=loop.main, event=damage, ...)` from a parameter form is doing what feels like coding; a designer who picks `hurt_player` from a dropdown is doing what feels like authoring.

This plan ships the three coupled foundations as one milestone. After it lands, idea #1's Player tag becomes a meaningful concept (Archetype: Player); idea #4's audio bindings have intents to fire on; idea #6's RPG menus have a blackboard to read state from. The brainstorm's bet ("a non-coder authors a Goomba in 60 seconds via a character-sheet inspector") is only proven when all three foundations exist simultaneously.

---

## Carried Forward from Origin

All 15 requirements (R1-R15), all 7 acceptance examples (AE1-AE7), and all 4 actors (A1-A4) from `docs/brainstorms/2026-05-18-entity-verb-sheet-v1-requirements.md` are in scope. Specifically:

| Origin | Scope summary | Plan unit(s) |
|---|---|---|
| R1, R2, R3, R4 | Six archetypes; per-archetype trigger-slot templates; character-sheet inspector layout; "Show advanced" parameter-form fallback | U5, U8 |
| R5, R6 | Verbs are catalog recipes; verb dropdown filters by trigger relevance | U4, U8 |
| R7, R8, R9 | Project.InputMap; ~9 intents with sensible defaults; verbs reference intents not raw keys | U2, U3, U7 |
| R10, R11, R12 | Global blackboard; reserved canonical keys; flat key-value store | U1, U9 |
| R13 | Blackboard is save serialization unit (UI deferred to idea #6) | U1 (primitive only) |
| R14 | Three additive `omitempty` schema fields (Project.InputMap, Entity.Archetype, Entity.VerbBindings) | U2 |
| R15 | Verb-sheet binding compiles to EventSheetRule at load; "advanced (composed)" when round-trip diverges | U6 |
| AE1–AE7 | All seven acceptance examples preserved as integration test scenarios | U10 |
| A1–A4 | Designer, Power-user designer, Studio, Engine — all referenced in unit flows | All units |

Origin's "Deferred to Planning" section: 8 technical questions, all resolved in Phase 2 above. No blocking product questions.

---

## High-Level Technical Design

The runtime + edit-time flow at a glance:

```
                   AUTHORING (studio, edit time)
                   ════════════════════════════════════
                          
   ┌─────────────────────┐    ┌──────────────────────┐
   │ Settings → Input    │    │ Character-Sheet      │
   │ panel (U7)          │    │ Inspector (U8)       │
   │ ─ edit InputMap     │    │ ─ pick archetype     │
   │   intent → keys     │    │ ─ pick verb per slot │
   └─────────┬───────────┘    └──────────┬───────────┘
             │ writes                    │ writes
             ▼                           ▼
   ┌────────────────────────────────────────────────┐
   │ Project (.pforge JSON)                         │
   │ ─ InputMap []InputBinding         (U2)         │
   │ ─ Entity.Archetype string         (U2)         │
   │ ─ Entity.VerbBindings []…         (U2)         │
   └────────────────────────────────────────────────┘
             │ load                       │ load
             ▼                            ▼
   ┌─────────────────────┐    ┌──────────────────────┐
   │ Intent Compiler     │    │ Verb-Binding         │
   │ (U3, init)          │    │ Compiler (U6, load)  │
   │ ─ register pievent  │    │ ─ binding → recipe   │
   │   targets per intent│    │ ─ recipe → Action    │
   │ ─ subscribe to      │    │ ─ wire as            │
   │   key.main/pad.btn  │    │   EventSheetRule     │
   │ ─ republish as      │    └──────────┬───────────┘
   │   "input/jump" etc. │               │ feeds
   └──────────┬──────────┘               ▼
              │ publishes      ┌─────────────────────┐
              ▼                │ Existing runtime    │
   ┌─────────────────────┐     │ (compile.go)        │
   │ pievent targets:    │◀────┤ ─ EventSheetRule    │
   │ ─ "input/jump"      │     │   → subscribes on   │
   │ ─ "input/attack"    │     │   condTarget        │
   │ ─ "state/*"  (U1)   │     │ ─ on event, runs    │
   └─────────────────────┘     │   Action Effect     │
              ▲                └─────────────────────┘
              │ reads/writes
              │
   ┌─────────────────────┐    ┌──────────────────────┐
   │ Blackboard          │    │ Blackboard Inspector │
   │ Inspector (U9)      │    │ panel (U9)           │
   │ ─ live key/value    │◀───┤ ─ subscribes via     │
   │   live key/value    │    │   SubscribeAll       │
   └─────────────────────┘    └──────────────────────┘

                   RUNTIME (shipped game)
                   ════════════════════════════════════
                   Same pievent bus + EventSheetRule + Action Effects.
                   InputMap loads from .pforge; intent targets fire from
                   key/pad events; blackboard state mutated by verbs;
                   no studio-side surfaces present.
```

*This illustrates the intended approach and is directional guidance for review, not implementation specification. The implementing agent should treat it as context, not code to reproduce.*

The structural insight: **no new runtime path is added.** Every new surface (intent targets, blackboard, verb recipes, character-sheet inspector) compiles down to existing `pievent.Target` registrations and existing `EventSheetRule` entries. The runtime side (`compile.go`'s `compileRule`) walks `EventSheetRule` already — it doesn't need to know whether the rule came from a step-lane authoring path or a verb-sheet authoring path. R15's compile-to-EventSheetRule promise is what makes this honest.

---

## Output Structure

This plan creates two new packages plus several new files inside existing packages:

```
pixelforge_blackboard/                   (NEW package, U1)
├── blackboard.go                        — Target, Set, Get, Inspectable
├── canonical_keys.go                    — reserved key list (score/lives/etc.)
├── blackboard_test.go
└── doc.go

pixelforge_input/                        (NEW package, U3)
├── intents.go                           — IntentName constants, target registry
├── compiler.go                          — InputMap → subscriptions at init
├── intents_test.go
├── compiler_test.go
└── doc.go

pixelforge_project/
├── input_map.go                         (NEW, U2) — InputBinding struct + defaults
├── archetype.go                         (NEW, U2) — Archetype constants + Entity field
└── verb_binding.go                      (NEW, U2) — VerbBinding struct + sanitize

pixelforge_studio/scripting/catalog/
├── verb_recipes.go                      (NEW, U4) — RegisterVerbRecipe + builtin recipes
└── verb_recipes_test.go

pixelforge_studio/scripting/archetype/   (NEW package, U5)
├── archetype.go                         — Template, TriggerSlot, RegisterArchetype
├── builtin_archetypes.go                — Player/Enemy/Pickup/Hazard/NPC/World templates
└── archetype_test.go

pixelforge_studio/scripting/runtime/
└── verb_binding_compile.go              (NEW, U6) — VerbBinding → EventSheetRule

pixelforge_studio/input/                 (NEW package, U7)
├── workspace.go                         — Settings → Input panel
└── workspace_test.go

pixelforge_studio/editor/
└── inspector.go                         (MODIFY, U8) — archetype-aware renderEntity

pixelforge_studio/blackboard/            (NEW package, U9)
├── workspace.go                         — Blackboard inspector panel
└── workspace_test.go

pixelforge_studio/integration_test/      (NEW, U10)
└── verb_sheet_e2e_test.go               — AE1–AE7 acceptance tests
```

This is a scope declaration. The implementer may consolidate or split files if implementation reveals a better layout — per-unit `**Files:**` sections remain authoritative.

---

## Implementation Units

Dependency-ordered. Each unit ships a complete, testable capability — "one-shot complete features per unit" per the user's directive.

### U1. Blackboard primitive (pievent.Target wrapping map[string]any)

**Goal:** A single global `pievent.Target` named `state/*` wrapping `map[string]any` with publish-on-change semantics. Exposes `Set(key, value)`, `Get(key) (any, bool)`, `Keys() []string` via an `Inspectable`-conformant sidecar. Zero studio surface; pure engine addition.

**Requirements:** R10, R11 (canonical key registry), R12, R13 (primitive only; UI deferred).

**Dependencies:** none (foundational).

**Files:**
- `pixelforge_blackboard/blackboard.go` (NEW)
- `pixelforge_blackboard/canonical_keys.go` (NEW)
- `pixelforge_blackboard/blackboard_test.go` (NEW)
- `pixelforge_blackboard/doc.go` (NEW)

**Approach:**
- New package `pixelforge_blackboard` registers a single `*Blackboard` at `init()` via `pievent.RegisterTarget("state/*", bb)`.
- `Blackboard` carries `map[string]any` guarded by `sync.RWMutex`; `Set(k, v)` publishes a `BlackboardChange{Key, Old, New}` event on an internal `pievent.Target[BlackboardChange]`; `Get(k)` and `Keys()` are read-only.
- `Inspectable` surface: existing two methods (`SubscriberCount`, `PublishCount`) plus extension methods (`KeysSnapshot()`, `Lookup(k)`) for the blackboard inspector workspace (U9) — non-generic so the studio can enumerate without type parameters.
- Canonical key registry: `canonical_keys.go` declares `var CanonicalKeys = map[string]CanonicalKeySpec{"score": {Type: TypeInt, Default: 0}, ...}` — used by U9's inspector for typed widgets and by U6's compile-time validation.

**Patterns to follow:** existing `pievent.RegisterTarget` self-registration at `init()` (see `pixelforge_loop/piloop.go:17-25`, `pixelforge_key/pikey.go:133-134`).

**Test scenarios:**
- `TestBlackboard_SetGet_RoundTrip`: Set("score", 100); Get("score") returns (100, true).
- `TestBlackboard_Get_MissingKey`: Get("nonexistent") returns (nil, false).
- `TestBlackboard_Set_PublishesChange`: subscribe to the change target; Set("score", 100); receive `{Key: "score", Old: nil, New: 100}`.
- `TestBlackboard_Set_PublishesOldValueOnUpdate`: Set("score", 100); Set("score", 200); second event is `{Key: "score", Old: 100, New: 200}`.
- `TestBlackboard_ConcurrentSet_NoRace`: 100 goroutines × 100 writes; final state consistent; race detector clean.
- `TestBlackboard_Inspectable_RegistersOnInit`: `pievent.LookupTarget("state/*")` returns non-nil after package import.
- `TestBlackboard_CanonicalKeys_TypeDeclared`: each canonical key (score, lives, health, current_scene, player_x, player_y) has a declared `Type` and `Default`.
- `TestBlackboard_KeysSnapshot_Stable`: KeysSnapshot returns a copy, not a reference (mutating the returned slice doesn't affect internal state).

**Verification:** `go test ./pixelforge_blackboard/...` passes; race detector clean; `go vet` clean; the package import has no side effects beyond the init-time registration.

---

### U2. Schema additions (Project.InputMap, Entity.Archetype, Entity.VerbBindings)

**Goal:** Three additive `omitempty` fields land in `pixelforge_project`. Pre-v1 `.pforge` files load cleanly with defaults applied. Round-trip preserves new fields.

**Requirements:** R14, AE6.

**Dependencies:** none (parallel with U1).

**Files:**
- `pixelforge_project/input_map.go` (NEW)
- `pixelforge_project/archetype.go` (NEW)
- `pixelforge_project/verb_binding.go` (NEW)
- `pixelforge_project/project.go` (MODIFY — extend `applyDefaults`, add fields)
- `pixelforge_project/scenes.go` (MODIFY — add `Archetype` + `VerbBindings` to `Entity`)
- `pixelforge_project/input_map_test.go` (NEW)
- `pixelforge_project/archetype_test.go` (NEW)
- `pixelforge_project/verb_binding_test.go` (NEW)

**Approach:**
- `InputBinding{Intent string; Keyboard []string; GamepadButton string; Modifier string}` slice on `Project.InputMap`. Defaults: hand-coded list of 9 intents with NES-style bindings (`input/jump` → `Z`/`Space`/`GamepadA`, etc.). Slice convention matches existing `[]EventSubscription` precedent (project.go:52-64).
- `Entity.Archetype string` field (omitempty). Six valid values declared as constants in `archetype.go`: `ArchetypePlayer`, `ArchetypeEnemy`, `ArchetypePickup`, `ArchetypeHazard`, `ArchetypeNPC`, `ArchetypeWorld`. Default for unset (sanitized at load): `ArchetypeWorld`.
- `Entity.VerbBindings []VerbBinding` (omitempty). `VerbBinding{Trigger string; RecipeName string; ArgOverrides map[string]any}`. Unknown recipe name at load → keep as-is (do not silently drop); compiler in U6 marks the slot "advanced (composed)" in the inspector.
- `applyDefaults` (project.go:107-113) extends to: populate InputMap from `DefaultInputMap()` if empty; iterate `Scenes[].Entities[]` and default Archetype to `ArchetypeWorld` if empty.
- Per `docs/solutions/editor-pforge-schema-shape.md`: malformed entries log + fall through to defaults, never fatal.

**Patterns to follow:** existing `Theme` field on Project (project.go:31-33) and `EventSubscription` slice convention.

**Test scenarios:**
- `TestProject_InputMap_RoundTrip`: marshal project with InputMap, unmarshal, fields equal.
- `TestProject_InputMap_DefaultsAppliedOnLoad`: load a project JSON with no `input_map` key; `applyDefaults` populates 9 intents with default bindings.
- `TestProject_InputMap_OmitemptyWhenDefault`: a project whose InputMap equals the defaults marshals without the `input_map` key in the JSON (size-stable for pre-v1 projects).
- `TestEntity_Archetype_DefaultsToWorld`: load a pre-v1 entity (no `archetype` key); after sanitize, `Archetype == "World"`.
- `TestEntity_Archetype_RoundTrip`: marshal/unmarshal with each of the 6 archetype constants.
- `TestEntity_VerbBindings_UnknownRecipePreserved`: project with `{Trigger: "when_stepped_on", RecipeName: "nonexistent_verb"}`; after load, the binding is preserved (not dropped). The inspector will display "advanced (composed)" later (U8).
- `TestEntity_VerbBindings_ArgOverridesPreserved`: marshal binding with `{RecipeName: "give_points", ArgOverrides: {"amount": 50}}`; unmarshal; override map equal.
- `TestProject_PreV1Roundtrip`: load an existing `.pforge` fixture from `pixelforge_studio/editor/cart_assets/editor.pforge`; verify no behavior change; resave; resulting JSON has no spurious new keys.
- Covers AE6 (pre-v1 .pforge loads cleanly with `Archetype: "World"` and empty maps).

**Verification:** existing project tests pass; new schema tests pass; manually open and save an existing test fixture, diff for unexpected key additions.

---

### U3. Input intent layer (pixelforge_input package)

**Goal:** ~9 named intents register as `pievent.Target` at init. A subscription bridge listens to `key.main` + `pad.button`, looks up the matching intent in the active `Project.InputMap`, and republishes on the intent target. Verbs subscribe to `input/jump` etc. via the existing `pievent` `subscribeAny` reflection in `compile.go`.

**Requirements:** R7, R8, R9.

**Dependencies:** U1 (uses the same Inspectable pattern), U2 (reads InputMap field).

**Files:**
- `pixelforge_input/intents.go` (NEW) — intent name constants + target registrations
- `pixelforge_input/compiler.go` (NEW) — InputMap → subscriptions
- `pixelforge_input/intents_test.go` (NEW)
- `pixelforge_input/compiler_test.go` (NEW)
- `pixelforge_input/doc.go` (NEW)

**Approach:**
- Intent constants: `IntentJump = "input/jump"`, `IntentAttack = "input/attack"`, `IntentUp/Down/Left/Right`, `IntentUse`, `IntentMenu`, `IntentStart` (9 total per R7).
- Each intent registers its own `pievent.Target[IntentEvent]` at `init()` via `pievent.RegisterTarget(IntentJump, target)` etc.
- `IntentEvent{Type IntentEventType; Intent string}` carries Down/Up event types (mirrors `pixelforge_key.Event` shape).
- Compiler (compiler.go): `Compile(im InputMap)` subscribes once to `key.main` and `pad.button` targets (via `pievent.LookupTarget` + reflection or direct typed subscription); for each incoming event, walks the InputMap to find matching intent bindings; publishes on each matched intent target. **Subscription happens at compile time, not per-tick** (per `docs/solutions/scripting-runtime-design.md` — reflection cost is one-shot).
- `Recompile(im InputMap)` for InputMap-edit live-update: unsubscribes the prior bridge subscribers and re-subscribes with the new map. Called by U7's Settings → Input workspace on save.
- **No new external dependency.** Ports the multi-binding-per-intent pattern from `quasilyte/ebitengine-input/keymap.go` (read-only reference; no import).
- Modifier support (Ctrl/Shift/Alt): bindings carry an optional `Modifier` field; intent fires only when the modifier key is also held (uses `pixelforge_key.Duration` polling).

**Patterns to follow:** `pixelforge_key/pikey.go:116-134` for `Target()` exposure + init-time registration; `pixelforge_studio/scripting/runtime/compile.go:177-200` for `subscribeAny`-style reflection (the same pattern verbs use to subscribe to intent targets).

**Test scenarios:**
- `TestIntents_AllRegisteredOnInit`: after package import, `pievent.LookupTarget("input/jump")` returns non-nil for all 9 intents.
- `TestCompiler_KeyboardEventPublishesIntent`: with `InputMap{IntentJump: {Keyboard: ["Z"]}}`, simulate a `pixelforge_key.Event{Type: Down, Key: "Z"}`; the `input/jump` target receives an `IntentEvent{Type: Down, Intent: "input/jump"}`.
- `TestCompiler_GamepadEventPublishesIntent`: with `InputMap{IntentJump: {GamepadButton: "A"}}`, simulate a `pixelforge_pad.EventButton{Type: Down, Button: "A"}`; intent fires.
- `TestCompiler_MultiBindingFiresOnAny`: with `IntentJump: {Keyboard: ["Z", "Space"], GamepadButton: "A"}`, all three sources independently trigger the intent.
- `TestCompiler_ModifierGate`: with `IntentMenu: {Keyboard: ["S"], Modifier: "Ctrl"}`, pressing S alone does NOT fire; Ctrl+S does fire.
- `TestCompiler_Recompile_UnsubscribesOldBindings`: Compile with `IntentJump → "Z"`; Recompile with `IntentJump → "X"`; subsequent `Z` event does NOT fire intent.
- `TestCompiler_UnknownIntentInMap_LogsWarning`: InputMap contains `"input/nonexistent"`; Compile logs a warning and proceeds (no crash).
- `TestCompiler_NoMatchingBinding_NoEvent`: unmapped key press produces no intent event.

**Verification:** `go test ./pixelforge_input/...` passes; race detector clean; manual smoke (`go run ./pixelforge_studio`) confirms intent-bound verbs respond to keys.

---

### U4. Verb recipe registry (catalog extension)

**Goal:** Extend `pixelforge_studio/scripting/catalog` with `RegisterVerbRecipe(name, recipe)` where a `Recipe` wraps an existing catalog Action with default args baked in. Ship ~25-30 baked recipes covering arcade essentials (per origin R5 + the brainstorm's enumerated recipe list).

**Requirements:** R5, R6.

**Dependencies:** U1 (verbs that mutate blackboard use the canonical key registry).

**Files:**
- `pixelforge_studio/scripting/catalog/verb_recipes.go` (NEW)
- `pixelforge_studio/scripting/catalog/verb_recipes_test.go` (NEW)

**Approach:**
- `Recipe{ActionKind string; DefaultArgs map[string]any; RelevantTriggers []string}` — the recipe references an existing Action kind by name (looked up via existing `LookupAction`) and pre-fills its args.
- `RegisterVerbRecipe(name string, recipe Recipe)` registers in a package-level map. Re-registration logs warning + overwrites (mirrors `RegisterStep` / `RegisterAction` semantics in catalog.go:84-120).
- `LookupRecipe(name string) (Recipe, bool)` — returned by the inspector to render verb dropdowns.
- `Apply(recipe Recipe, overrides map[string]any) (Effect, error)`: merges defaults + overrides, looks up the underlying Action builder, builds the Effect.
- v1 recipe list (~28 entries, grouped):
  - **Damage**: `die`, `hurt_player`, `take_damage`
  - **State**: `give_points`, `lose_life`, `restore_health`, `set_flag`, `check_flag`, `give_item`, `take_item`
  - **Audio**: `play_sound`, `play_music`, `stop_music`
  - **Motion**: `move_with_intent`, `move_pattern`, `bounce`, `teleport_to`
  - **Spawn**: `spawn_entity`, `destroy_self`, `destroy_other`
  - **Flow**: `change_scene`, `wait`, `restart_scene`
  - **Visual**: `hide`, `show`, `flash`, `swap_sprite`
  - **Dialogue/menu** (placeholder shells, fully wired by idea #6): `open_dialogue`, `open_menu`
- Per-trigger relevance: `RelevantTriggers []string` filters which dropdown the verb appears in (e.g., `die` is relevant to `when_stepped_on`/`when_shot`/`when_damaged` but not `when_scene_starts`).
- **No new Action builders in this unit.** Recipes wrap existing Actions in `builtin_actions.go`. Where an Action doesn't yet exist (e.g., `give_points` needs `set_value` against blackboard `score`), the recipe's `DefaultArgs` configures the existing Action.

**Patterns to follow:** existing `catalog.RegisterAction` registration shape (catalog.go:110-120); existing builtin Action style in `pixelforge_studio/scripting/catalog/builtin_actions.go` (e.g., `play_sample` line 18, `set_value` line 38).

**Test scenarios:**
- `TestRegisterVerbRecipe_LookupReturnsRecipe`: register a recipe; LookupRecipe returns it.
- `TestRegisterVerbRecipe_DuplicateOverwritesAndLogs`: register same name twice; second call wins; warning logged.
- `TestApplyRecipe_DefaultArgsBaked`: `Apply("give_points", nil)` returns an Effect that, when invoked, increments blackboard `score` by the recipe's default amount.
- `TestApplyRecipe_OverridesWin`: `Apply("give_points", {"amount": 50})` increments by 50, not the default 10.
- `TestApplyRecipe_UnknownActionKind`: recipe references non-existent Action; Apply returns an error (handled by the runtime compiler in U6).
- `TestVerbRecipes_AllBuiltinRecipesValid`: iterate all registered recipes; each one's `ActionKind` resolves via `LookupAction`; each `DefaultArgs` produces a valid Effect when applied.
- `TestVerbRecipes_RelevantTriggersDeclaredForEach`: every recipe has at least one `RelevantTrigger` value (no empty list means it'd never appear in a dropdown).
- Covers AE4 indirectly (override flow exercised here; full inspector-driven override in U8).

**Verification:** `go test ./pixelforge_studio/scripting/catalog/...` passes; each v1 recipe demonstrably callable through `Apply`.

---

### U5. Archetype template registry

**Goal:** Six archetypes (Player/Enemy/Pickup/Hazard/NPC/World) each declare a `Template` listing trigger slots + default verb assignments. The inspector (U8) renders the per-archetype slot list; defaults pre-fill so newly-dropped entities of common archetypes work without designer configuration.

**Requirements:** R1, R2, R3 (template-driven slot list), AE1 (Enemy archetype's `when_stepped_on` pre-fills with `die`).

**Dependencies:** U4 (templates reference verb recipe names).

**Files:**
- `pixelforge_studio/scripting/archetype/archetype.go` (NEW)
- `pixelforge_studio/scripting/archetype/builtin_archetypes.go` (NEW)
- `pixelforge_studio/scripting/archetype/archetype_test.go` (NEW)

**Approach:**
- `Template{Archetype string; TriggerSlots []TriggerSlot}` where `TriggerSlot{Name string; DisplayName string; DefaultVerb string; RelevantRecipes []string}`.
- `RegisterArchetype(t Template)` — same package-level registry shape as catalog and pfcomponent.
- `LookupArchetype(name string) (Template, bool)`.
- Six built-in templates registered at `init()` in `builtin_archetypes.go`:
  - **Player**: slots = `when_input/jump` (default: `jump`), `when_input/attack` (default: `play_sound:laser`), `when_input/left` (default: `move_with_intent`), `when_input/right` (default: `move_with_intent`), `when_input/up`, `when_input/down`, `when_damaged` (default: `take_damage(1)`), `when_scene_starts`, `every_tick`.
  - **Enemy**: slots = `when_stepped_on` (default: `die`), `when_shot` (default: `die`), `when_player_nearby` (default: empty), `when_scene_starts`, `every_tick`, `when_damaged`, `when_destroyed` (default: `give_points(10)`).
  - **Pickup**: slots = `when_touched` (default: `give_points(10) + destroy_self`), `when_scene_starts`, `every_tick`.
  - **Hazard**: slots = `when_touched` (default: `hurt_player`), `when_scene_starts`, `every_tick`.
  - **NPC**: slots = `when_interacted` (default: empty — designer picks `open_dialogue:<name>` in idea #6), `when_player_nearby`, `when_scene_starts`.
  - **World**: slots = `when_scene_starts`, `every_tick` (minimal — for decorative entities). Default for unset entities per U2.
- **Compound default verbs** (like Pickup's "give_points + destroy_self"): represented as a single composite verb recipe `pickup_default` registered in U4's catalog rather than special-casing the archetype template.

**Patterns to follow:** registry-init pattern in `pixelforge_studio/scripting/catalog/catalog.go:84-120`.

**Test scenarios:**
- `TestArchetype_AllSixRegisteredOnInit`: all six archetypes resolvable via `LookupArchetype`.
- `TestArchetype_Player_TriggerSlotsMatchSpec`: Player has exactly the 9 slots declared above, in order.
- `TestArchetype_Enemy_DefaultVerbs`: Enemy's `when_stepped_on` slot has `DefaultVerb == "die"`.
- `TestArchetype_World_MinimalSlots`: World has exactly `when_scene_starts` and `every_tick` — no other slots.
- `TestArchetype_RelevantRecipesPopulated`: each slot has `RelevantRecipes` populated such that the dropdown isn't empty for any slot.
- `TestArchetype_AllDefaultVerbsResolveInCatalog`: every `DefaultVerb` reference resolves via `catalog.LookupRecipe` (catches typos at test time).
- Covers AE1 (Enemy archetype shows `when_stepped_on: die` pre-filled).

**Verification:** `go test ./pixelforge_studio/scripting/archetype/...` passes; manual smoke: open the studio with a fresh project, drop a sprite, set archetype to Enemy, observe pre-filled `die` verb in `when_stepped_on` slot.

---

### U6. Verb-binding compiler (load-time)

**Goal:** At project-load time, each `Entity.VerbBindings` entry compiles to one `EventSheetRule` (existing schema), wired into the entity's BehaviorGraph. Unknown recipe names → preserve binding; mark "advanced (composed)" in the inspector. Round-trip preserves verb-sheet shape when bindings match a recipe form.

**Requirements:** R15, AE7.

**Dependencies:** U2 (schema fields), U4 (verb recipe lookup), U5 (trigger name validation against archetype templates).

**Files:**
- `pixelforge_studio/scripting/runtime/verb_binding_compile.go` (NEW)
- `pixelforge_studio/scripting/runtime/verb_binding_compile_test.go` (NEW)
- `pixelforge_studio/scripting/runtime/compile.go` (MODIFY — hook into existing project-load path)

**Approach:**
- `CompileVerbBindings(entity *pixelforge_project.Entity, archetype archetype.Template) ([]pixelforge_project.EventSheetRule, error)`.
- For each `VerbBinding`:
  - Lookup recipe via `catalog.LookupRecipe(binding.RecipeName)`.
  - If found: build the underlying Action via `catalog.Apply(recipe, binding.ArgOverrides)`; construct an `EventSheetRule{Conditions: [{Kind: "event_fired", Args: {target: triggerNameToTarget(binding.Trigger)}}], Actions: [{Kind: recipe.ActionKind, Args: merged(recipe.DefaultArgs, binding.ArgOverrides)}]}`.
  - If recipe not found: preserve the binding as a placeholder rule (so the EventSheet remains intact); mark the entity for "advanced (composed)" rendering in U8's inspector.
- Round-trip detection (for "show as named recipe" in U8): given an EventSheetRule, walk back — if Conditions/Actions exactly match a recipe's expected shape (Action.Kind == recipe.ActionKind, Action.Args == recipe.DefaultArgs ∪ binding.ArgOverrides), render as the named recipe; else render as "advanced (composed)".
- Hook into existing load: extend `runtime.NewEngine(project)` (or equivalent loader) to call `CompileVerbBindings` per entity and merge results into the entity's BehaviorGraph's EventSheet before passing to existing `compileRule` (compile.go:78-83).
- **No runtime behavior change**: the produced EventSheetRule entries go through the same `compileRule` path as power-user-authored rules. Live-edit propagation via `Engine.Reload(graphName)`.

**Patterns to follow:** existing `compile.go:78-83` rule iteration; existing `EventSheetRule` shape in `pixelforge_project/behaviors.go:38-42`.

**Test scenarios:**
- `TestCompileVerbBindings_SimpleBinding`: Entity with `{Trigger: "when_stepped_on", RecipeName: "die"}`; compile produces one EventSheetRule with the expected Condition kind + Action kind.
- `TestCompileVerbBindings_WithOverrides`: `{RecipeName: "give_points", ArgOverrides: {"amount": 50}}`; compiled Action's Args has `amount: 50` (override wins over default 10).
- `TestCompileVerbBindings_UnknownRecipePreservedAsPlaceholder`: `{RecipeName: "nonexistent"}`; compile returns no error; rule is preserved as a placeholder; entity is marked "has unknown bindings".
- `TestCompileVerbBindings_TriggerMatchesArchetypeSlot`: binding's Trigger references a slot that exists in the entity's archetype template — happy path.
- `TestCompileVerbBindings_TriggerNotInArchetype_LogsWarning`: binding's Trigger doesn't match any slot in the entity's archetype; warning logged; rule still compiled (don't drop user content).
- `TestRoundtrip_BindingToRuleToBinding`: given a binding, compile to rule; given the rule, detect the source recipe + overrides; reconstruct binding; original == reconstructed.
- `TestRoundtrip_PowerUserEditedRule_DetectedAsAdvanced`: take a compiled rule, manually mutate its Actions to add a second action; the detector returns "advanced (composed)" instead of the named recipe.
- `TestCompileVerbBindings_HotReloadIdempotent`: compile twice with the same input; results equal; no duplicate subscriptions in the runtime.
- Covers AE7 directly (power-user edits in event-sheet view → inspector shows "advanced (composed)").

**Verification:** `go test ./pixelforge_studio/scripting/runtime/...` passes; integration: open a project with verb-sheet bindings, launch the runtime, trigger a verb, observe expected Action fires.

---

### U7. Settings → Input panel (pixelforge_studio/input workspace)

**Goal:** New dockable workspace where the designer edits Project.InputMap. One row per intent; each row shows current keyboard + gamepad + modifier bindings with dropdown pickers. MarkDirty on edit; saves trigger U3's `Recompile` so the intent layer reflects the new map without restart.

**Requirements:** R7, R8, F4 (designer remaps Space → Z and all verbs update).

**Dependencies:** U2 (schema), U3 (intent layer's Recompile API).

**Files:**
- `pixelforge_studio/input/workspace.go` (NEW)
- `pixelforge_studio/input/workspace_test.go` (NEW)
- `pixelforge_studio/main.go` (MODIFY — add `input.RegisterWith(e)` registration)

**Approach:**
- `Workspace` struct implementing the `editor.Workspace` interface (Name, DisplayName, Render). Pattern from `pixelforge_studio/scripting/workspace.go:245-254`.
- `Render(e *Editor)` draws an ImGui table: column 1 = intent display name, columns 2-4 = keyboard / gamepad / modifier dropdowns.
- Each dropdown is a `comboField`-style selector (reuse existing helper from `inspector.go:203-225`). Keyboard column lists all `pixelforge_key.Key` values; gamepad column lists `pixelforge_pad.Button` values; modifier column lists `none/Ctrl/Shift/Alt`.
- On any dropdown change: mutate the corresponding `InputBinding` in `e.Project().InputMap`; call `e.MarkDirty()`; call `pixelforge_input.Recompile(e.Project().InputMap)` so the intent layer reflects the edit live (per `docs/solutions/always-on-game-embedding.md` — no restart needed).
- "Restore defaults" button: replaces InputMap with `pixelforge_project.DefaultInputMap()`, marks dirty, recompiles.
- FocusManager: each dropdown registers as a focus-capable widget (per `docs/solutions/focus-manager-design.md`); Tab/Shift+Tab cycles through.
- **No "press the key you want" capture flow in v1** (per origin's deferred section). Dropdown picker only.

**Patterns to follow:** existing workspace registration in `pixelforge_studio/scripting/workspace.go:31-44`; existing `comboField` dropdown helper in `pixelforge_studio/editor/inspector.go:203-225`; `MarkDirty` + `PromptIfDirty` pattern from `docs/solutions/dirty-state-ux.md`.

**Test scenarios:**
- `TestWorkspace_RegisteredAlongsideOtherWorkspaces`: after `RegisterWith(editor)`, `editor.Workspaces()` includes one with Name == "input".
- `TestWorkspace_EditDropdownMutatesInputMap`: simulate dropdown change for IntentJump's keyboard from "Z" to "Space"; the Project's InputMap reflects the change.
- `TestWorkspace_EditCallsMarkDirty`: after any dropdown change, `editor.IsDirty()` is true.
- `TestWorkspace_EditCallsRecompile`: after a change, `pixelforge_input.Recompile` was invoked (verify via mock or by checking that subsequent intent events route per the new map).
- `TestWorkspace_RestoreDefaults_RewritesMap`: click "Restore defaults"; InputMap equals `DefaultInputMap()`; dirty flag set.
- `TestWorkspace_F4Flow_RemapTakesEffectWithoutRestart`: load project; observe IntentJump fires on Z; via workspace, change IntentJump to Space; simulate Space press; IntentJump fires (no engine restart in between).
- Covers F4 (designer remaps and all verbs update).

**Verification:** `go test ./pixelforge_studio/input/...` passes; manual smoke: open studio, open Settings → Input panel, change a binding, verify game responds to new key in preview.

---

### U8. Character-sheet inspector layout

**Goal:** Modify `pixelforge_studio/editor/inspector.go`'s `renderEntity` to dispatch an archetype-aware layout. Archetype dropdown at the top; trigger slots from the archetype template below; verb-pick dropdown per slot; "Show advanced" per-verb toggle drops to the existing Action parameter form.

**Requirements:** R3, R4, AE1, AE2, AE3 (multi-entity selection still shows single entity per origin).

**Dependencies:** U2 (Archetype + VerbBindings fields), U4 (verb recipe registry), U5 (archetype templates), U6 (round-trip detection for "advanced (composed)").

**Files:**
- `pixelforge_studio/editor/inspector.go` (MODIFY — extend `renderEntity` with archetype branch)
- `pixelforge_studio/editor/inspector_test.go` (MODIFY — add archetype rendering tests)
- `pixelforge_studio/editor/widgets/widgets.go` (MODIFY — extend `Context` with `Intents []string` and `VerbRecipes []string` per per-trigger filtering)

**Approach:**
- Before iterating `ent.Components` (inspector.go:98), render the archetype dropdown using `comboField` against the six archetype constants.
- On archetype change: mutate `ent.Archetype`, mark dirty.
- For the selected archetype, lookup the template via `archetype.LookupArchetype`. For each `TriggerSlot`:
  - Render the slot label (display name).
  - Render a verb-pick dropdown listing `RelevantRecipes` for the slot.
  - If `ent.VerbBindings` has an entry for this trigger, pre-select it; else show the slot's `DefaultVerb` (greyed if it's a default; full color if designer-set).
  - On dropdown change: upsert the entry in `ent.VerbBindings`, mark dirty, trigger U6 recompile + `Engine.Reload(entity's behavior graph)`.
  - Render "Show advanced" toggle. If on: render the existing per-Action parameter form for the picked recipe's underlying Action (reuse existing pfcomponent dispatch).
  - If U6's round-trip detector flags the binding as "advanced (composed)" (unknown recipe or power-user-edited rule), render the slot's dropdown as a read-only "advanced (composed)" label with a "View in scripting workspace" link.
- The existing per-component pfcomponent rendering (inspector.go:98-123) stays — components other than archetype/verb-bindings continue to render after the character-sheet block.
- Widget Context (`widgets.go:43-58`) extends with `Intents []string` (for any future intent-name dropdowns inside verb parameter forms).

**Patterns to follow:** existing `comboField` helper (inspector.go:203-225); existing pfcomponent dispatch (inspector.go:98-123) as the "advanced" fallback.

**Test scenarios:**
- `TestInspector_RendersArchetypeDropdown`: render an entity; assert archetype dropdown is present with all 6 options.
- `TestInspector_ArchetypeChangeMutatesEntity`: simulate dropdown change; entity's Archetype field updates; dirty flag set.
- `TestInspector_RendersTriggerSlotsFromArchetype`: with Enemy archetype, slots rendered match the Enemy template's TriggerSlots.
- `TestInspector_PreFillsDefaultVerb`: Enemy's `when_stepped_on` slot shows `die` pre-filled (matching AE1).
- `TestInspector_DesignerSetVerbOverridesDefault`: simulate picking `bounce` instead of `die`; entity's VerbBindings reflects the override.
- `TestInspector_ShowAdvancedDropsToParameterForm`: toggle "Show advanced" on a `give_points` slot; assert parameter form for `set_value` Action is rendered with `amount: 10` pre-filled.
- `TestInspector_OverrideArgPersists`: change parameter form's `amount` from 10 to 50; entity's VerbBindings has `ArgOverrides: {amount: 50}`; subsequent runtime fires give 50 points.
- `TestInspector_UnknownRecipeShowsAdvancedComposed`: entity has a binding with `RecipeName: "nonexistent"`; inspector renders the slot as read-only "advanced (composed)".
- `TestInspector_PowerUserEditedRuleShowsAdvancedComposed`: load entity, mutate the compiled EventSheet to add a second Action (simulating power-user edit), re-render; slot shows "advanced (composed)".
- `TestInspector_WorldArchetypeShowsMinimalSlots`: World archetype shows only `when_scene_starts` + `every_tick` slots.
- Covers AE1 (Enemy default), AE2 (Player tag swap exclusivity — but Player tag handled via archetype = Player, ensured by mutual-exclusivity at the archetype level), AE3 (spawn cell logic is in idea #1, not here).

**Verification:** `go test ./pixelforge_studio/editor/...` passes; manual smoke: open project, select Goomba entity, see Enemy archetype + pre-filled `die`; change to Hazard; slots refresh; pick `hurt_player`; preview Mario stepping on Goomba — Mario takes damage.

---

### U9. Blackboard inspector panel

**Goal:** New dockable workspace surfacing live blackboard state. Canonical keys (score/lives/health/current_scene/player_x/player_y per R11) at the top with typed widgets (int sliders, integer fields); arbitrary designer-added keys below as a flat string→value list.

**Requirements:** R10, R11, R12.

**Dependencies:** U1 (blackboard primitive).

**Files:**
- `pixelforge_studio/blackboard/workspace.go` (NEW)
- `pixelforge_studio/blackboard/workspace_test.go` (NEW)
- `pixelforge_studio/main.go` (MODIFY — `blackboard.RegisterWith(e)`)

**Approach:**
- Same `Workspace` interface implementation as U7.
- `Render(e *Editor)` iterates the blackboard's `KeysSnapshot()`:
  - For each key in `CanonicalKeys`: render a typed widget per its declared type (int slider for `score`, int field for `lives`, etc.). Edits write back via `bb.Set(k, v)`.
  - For each non-canonical key: render a read-only `Text` widget with `key: value`. (Editing designer-added keys not in v1 — those are written by verbs at runtime, not by the inspector.)
- Live updates: subscribe to the blackboard's change target via `SubscribeAll` (per `docs/solutions/ring-buffer-snapshot-store.md` pattern). On `BlackboardChange` event, force a UI redraw.
- **Live during preview** (per `docs/solutions/always-on-game-embedding.md`): blackboard mutations from running verbs reflect immediately in the inspector.

**Patterns to follow:** existing workspace registration; existing pievent `SubscribeAll` from `pixelforge_studio/capture/recorder.go` (the ring-buffer recorder uses this pattern).

**Test scenarios:**
- `TestBlackboardWorkspace_RegistersOnInit`: after RegisterWith, `editor.Workspaces()` includes one with Name == "blackboard".
- `TestBlackboardWorkspace_RendersCanonicalKeys`: with score=100 in blackboard, render returns content containing "score" and "100".
- `TestBlackboardWorkspace_RendersTypedWidgetForScore`: score widget is an int slider (per CanonicalKeys spec), not a text input.
- `TestBlackboardWorkspace_EditScoreWritesToBlackboard`: simulate slider change from 100 to 200; `bb.Get("score")` returns 200.
- `TestBlackboardWorkspace_DesignerAddedKeyRenderedReadOnly`: blackboard has key "my_custom_flag" = true; rendered as read-only text "my_custom_flag: true".
- `TestBlackboardWorkspace_LiveUpdateOnBlackboardChange`: in preview, simulate verb writing `score = 100`; next render reflects 100 without manual refresh.
- Covers AE5 (Player damage decrements health 3 → 2 → 1 → 0; blackboard inspector shows the live transition).

**Verification:** `go test ./pixelforge_studio/blackboard/...` passes; manual smoke: run a game with `give_points` verbs; observe score live-updating in inspector.

---

### U10. End-to-end acceptance tests (Goomba scene)

**Goal:** Integration test that ties together U1-U9. Authored a Goomba via the verb-sheet; preview the scene; player steps on Goomba; Goomba dies; player earns points; score reflects in blackboard inspector. All seven acceptance examples (AE1-AE7) covered as test scenarios — not unit-mocked but exercising the real composition.

**Requirements:** all (every R is covered by at least one AE per the brainstorm).

**Dependencies:** U1-U9 all merged.

**Files:**
- `pixelforge_studio/integration_test/verb_sheet_e2e_test.go` (NEW)
- `pixelforge_studio/integration_test/fixtures/goomba_scene.pforge` (NEW — fixture project)

**Approach:**
- Create a minimal `.pforge` fixture with: one scene, one Player entity (archetype: Player, default verbs), one Enemy entity (archetype: Enemy, default `die` + `give_points(10)`), basic ScreenRoom (16×1 from idea #1).
- Test harness loads the fixture; constructs the runtime Engine; simulates input events; asserts blackboard transitions.
- **No headless rendering** — tests assert on model state (entity removed, blackboard mutated) rather than pixels.

**Test scenarios (one per AE):**
- `TestE2E_AE1_GoombaPreFilledWithDieOnSteppedOn`: load fixture; inspect Goomba; assert archetype == Enemy, `when_stepped_on` slot has default `die`.
- `TestE2E_AE2_PlayerArchetypeIsTheTag`: load fixture; assert Player entity's Archetype is "Player"; switching another entity to "Player" un-marks the prior (mutual exclusivity at archetype level).
- `TestE2E_AE3_SpawnPointSetViaArchetypeOrSceneProperty`: (defer to idea #1 plan — spawn cell is idea #1's scope; here just verify the Player entity's position is honored at runtime start).
- `TestE2E_AE4_CameraFollowsPlayer`: (defer to idea #1 plan — camera is idea #1's scope).
- `TestE2E_AE5_DamageDecrementsHealth`: configure Player with `when_damaged: take_damage(1)`; blackboard starts `health=3`; simulate damage event three times; blackboard transitions 3→2→1→0; final state asserted.
- `TestE2E_AE6_PreV1FileLoads`: load an existing pre-v1 fixture; assert no error; assert all entities defaulted to Archetype "World"; assert game still runs.
- `TestE2E_AE7_AdvancedComposedDetectedOnRuleEdit`: load fixture with Goomba `when_stepped_on: die`; programmatically mutate the compiled EventSheet to add a second Action; re-inspect; slot reports "advanced (composed)".
- `TestE2E_GoombaCompleteFlow_StepOnEnemyDiesGivesPoints`: load Goomba fixture; simulate Player walking into Enemy (publish a collision event); assert Enemy entity removed; assert blackboard score == 10.
- `TestE2E_InputMapEditPropagatesWithoutRestart`: load fixture; simulate intent/jump on Z fires Player jump; via input workspace API, change IntentJump to Space; simulate Space; Player jumps; original Z press no longer fires.

**Verification:** `go test ./pixelforge_studio/integration_test/...` passes; all AE1-AE7 scenarios green; manual smoke: launch studio with goomba_scene.pforge, play through scene, observe expected behaviors.

---

## Scope Boundaries

### Deferred to Follow-Up Work

- **Designer-authored verb recipes** (designer creates new recipes through the studio). v2.
- **Multi-condition triggers** ("when stepped on AND while flag X true"). Drops to event-sheet escape hatch in v1; v2 may add compound trigger UI.
- **Multi-verb slots** ("when stepped on: die AND give_points"). v1 ships a composite `pickup_default` recipe; broader multi-verb is v2 or escape-hatch.
- **Conditional verb selection per-other-entity** ("when stepped on by Player → die; when stepped on by Bullet → bounce"). v2.
- **Per-scene / per-entity blackboard scopes**. Single global blackboard in v1.
- **Custom archetypes** (designer defines new archetype). v2 or never (the six cover the NES reference set).
- **AI behavior trees / utility AI / state machines** on entities. v1's verb-sheet is intentionally flat.
- **Intent-capture UX** ("press the key you want to bind"). v1 uses dropdown pickers in the Settings → Input panel.
- **Save/load UI** (named slots, autosave). Handled by idea #6's brainstorm; v1 of #5 ships only the blackboard serialization primitive.
- **Per-player input maps** (multiplayer splitscreen). v1 ships one global InputMap.
- **Live-debugging time-travel** of blackboard state. v1 ships static live inspection; time-travel scrubbing deferred.
- **Visual debugging overlays** for verb execution (which verb fired when). Deferred.

### Outside this product's identity

- Multiplayer / network synchronization for blackboard.
- AI-assisted verb suggestions (LLM helps the designer pick verbs).
- Cloud-hosted blackboard for cross-device save sync.
- Mobile / touch UX for the inspector workspaces.

---

## Key Technical Decisions

- **Zero new external dependencies.** Phase 1 research evaluated `quasilyte/ebitengine-input`, generic FSM libs, behavior-tree libs, and reactive-state libs. All rejected per the leverage doctrine. Total custom code: ~700 LOC across U1+U3+U6, well under the wrap-cost of any candidate. Port the multi-binding pattern from `quasilyte/ebitengine-input/keymap.go` as design guidance only — no import.
- **Single `Inspectable` Target for blackboard, not per-key Targets.** Per `docs/solutions/scripting-runtime-design.md` — the non-generic Inspectable sidecar exists precisely for this kind of enumerable-shared-state target. Per-key targets would require dynamic `pievent.Register/Unregister` lifecycle that doesn't match the convention.
- **Verb recipes wrap existing catalog Actions; no parallel verb registry.** Promotes the existing `RegisterAction` seam (per learnings — "custom user Kinds plug in via the same Register seam"). Reduces the runtime surface to one extension point.
- **Verb-sheet bindings compile to EventSheetRule.** Single runtime path. The existing `compile.go`/`compileRule` walks EventSheetRule; doesn't care whether the rule came from step-lane authoring or verb-sheet authoring. This is R15's promise.
- **Six archetypes fixed in v1.** Closed list documented as a Key Decision in the brainstorm. Custom archetypes deferred (code change, not designer choice).
- **InputMap as `[]InputBinding` slice, not `map[string]…`.** Matches existing `Project` schema convention (`[]EventSubscription`); avoids being the first map-typed Project field (which would introduce stylistic inconsistency).
- **Designer-added blackboard keys are untyped (string-value fallback).** Per Phase 2 resolution. Canonical keys are typed via the reserved key registry; arbitrary keys default to string. Avoids adding a type-tracking schema for marginal v1 value.
- **InputMap edits live via `Recompile`; no engine restart.** Per `docs/solutions/always-on-game-embedding.md` — no Run/Edit mode distinction. The intent layer's `Recompile(im)` unsubscribes prior bridge subs + re-subscribes with the new map; takes effect on the next intent event.
- **No drag-and-drop in inspector; dropdowns only.** Per cimgui-go drag-drop deferral (established convention across prior brainstorms). Verb-pick dropdowns reuse `comboField` (`inspector.go:203-225`).
- **No `RegisterWidget` extension to pfcomponent in this plan.** The character-sheet layout is a layout shape inside `renderEntity` (a switch on archetype, not a new widget kind). Widget extensibility (idea #2's R2 ask) is left to idea #2's plan.

---

## Dependencies / Assumptions

- **Existing engine packages** (`pixelforge_event`, `pixelforge_key`, `pixelforge_pad`, `pixelforge_routine`) continue to work as documented. v1 of this plan modifies none of them.
- **Existing `catalog` registries + builtin Actions** (`pixelforge_studio/scripting/catalog/builtin_actions.go`) continue to expose `set_value`, `play_sample`, `publish_event`, `move_entity` etc. Verb recipes in U4 reference these by name.
- **Existing `EventSheetRule` schema** + `compile.go`'s `compileRule` are unchanged. U6 produces standard EventSheetRule entries that flow through the existing runtime.
- **Existing reflection inspector** (U4 of the ImGui migration) dispatches via switch on `pfcomponent.WidgetKind` in `inspector.go:130-197`. U8 modifies `renderEntity` (inspector.go:95-125) to add an archetype branch ahead of the per-component loop.
- **Existing `Workspace` interface** + `RegisterWith` pattern (workspaces.go:24-28 + `pixelforge_studio/scripting/workspace.go:245-254`). U7 and U9 register new workspaces via this pattern.
- **Existing dirty-state convention** (`MarkDirty` + `PromptIfDirty` per `docs/solutions/dirty-state-ux.md`). All edits in U7 and U8 route through it.
- **Idea #1 (ScreenRoom)** dependency on idea #5's Player archetype: U5's Player template + U2's Archetype field together provide the "Player tag" idea #1's R8 + camera follow needs. Idea #1's plan will reference this.
- **Idea #4 (audio library)** dependency on idea #5's event topics: U4 ships `play_sound` and `play_music` verb recipes that publish events on the topics idea #4's audio bindings subscribe to. Idea #4's plan will reference this.
- **Idea #6 (RPG-class)** dependency on idea #5's blackboard: U1 ships the primitive; idea #6's save UI serializes the blackboard's `KeysSnapshot()`. Idea #6's plan will reference this.

---

## Risk Analysis & Mitigation

| Risk | Severity | Mitigation |
|---|---|---|
| `pievent.Target[any]` for blackboard hits the same generic-iteration wall as scripting runtime (per learnings) | Medium | Use the established Inspectable sidecar pattern — the blackboard's `KeysSnapshot()` + `Lookup(k)` methods expose state non-generically. Verified pattern. |
| Verb-binding compilation produces "advanced (composed)" too aggressively (false positives where the inspector should round-trip cleanly) | Medium | U6 includes explicit round-trip-detection tests (`TestRoundtrip_BindingToRuleToBinding`). Edge cases for ArgOverrides equality (slice vs map comparison) covered in tests. |
| Six archetypes is the wrong cut — actual usage reveals a needed seventh (e.g., "Vehicle" for racing genres) | Low | Six was a deliberate choice per the brainstorm + research; expansion to seventh is `archetype.RegisterArchetype` away. Schema is forward-compatible (unknown archetype on load → falls back to World per U2's sanitize). |
| InputMap edit live-update introduces subscription leak (Recompile doesn't fully unsubscribe prior bridge) | Medium | U3's `TestCompiler_Recompile_UnsubscribesOldBindings` explicitly asserts subscription churn correctness. Race detector run in CI. |
| Workspace count is getting crowded (Scene/Inspector/Assets/Capture/Behavior/Palette + Audio + Dialogue + Items + Menus + Input + Blackboard = 12 in v1 of all 7 ideas) | Low | Per origin's brainstorm — "it'll be crowded but follows the same pattern." DockSpace handles tabs/groups; users dock per preference. |
| Verb library expansion (post-v1) breaks existing project files that reference older recipes | Low | Schema discipline preserves unknown bindings (U2's `TestEntity_VerbBindings_UnknownRecipePreserved`); inspector renders as "advanced (composed)" until the recipe is restored. Forward-compatible by construction. |
| Idea #5 plan ships before idea #1 — Player archetype is defined but no ScreenRoom camera consumes it yet | Low | The Archetype field is additive omitempty; nothing breaks. The Player archetype's `when_input/jump` slot just produces a verb recipe that fires; whether anything visible happens depends on whether the entity has a sprite + position. Idea #1's plan ships the camera consumer. |
| Test fixture for U10 (goomba_scene.pforge) requires coordination with idea #1 (ScreenRoom) which hasn't shipped | Medium | U10's E2E test for AE3 (spawn) and AE4 (camera) explicitly defer to idea #1's plan; this plan's E2E only verifies the verb-sheet + blackboard + intent flow. ScreenRoom integration tests live in idea #1's plan. |

---

## System-Wide Impact

**New packages introduced:** `pixelforge_blackboard`, `pixelforge_input`, `pixelforge_studio/scripting/archetype`, `pixelforge_studio/input`, `pixelforge_studio/blackboard`, `pixelforge_studio/integration_test`. Six new package roots; all follow the existing `pixelforge_*` / `pixelforge_studio/*` naming convention. None pull external dependencies.

**Modified packages:**
- `pixelforge_project` — three additive `omitempty` schema fields; one new sanitize helper.
- `pixelforge_studio/editor` — `inspector.go` extended with archetype branch; `widgets/widgets.go` Context extended with `Intents` + `VerbRecipes`.
- `pixelforge_studio/scripting/catalog` — `verb_recipes.go` extends the registry without modifying existing builders.
- `pixelforge_studio/scripting/runtime` — `verb_binding_compile.go` adds load-time compilation; `compile.go` extended with one hook.
- `pixelforge_studio/main.go` — three new `RegisterWith` lines (input, blackboard, plus the verb-recipe init via package import).

**Affected workflows:**
- **Designer authoring** — primary workflow target. The character-sheet inspector replaces script-lane authoring for the no-code surface. Power-user script-lane authoring remains untouched.
- **Engine runtime** — no new runtime path. EventSheetRule processing continues unchanged. The intent layer and blackboard expose new `pievent` targets that participate in the existing pub/sub bus.
- **Codegen / shipping** — no codegen change required. The shipped binary loads the same `.pforge` and runs the same runtime. Idea #7's Capsule already embeds the project; the additional schema fields ride along via the existing embed.
- **Tests** — new packages add new tests; existing tests untouched. Integration tests (U10) introduce a new `integration_test/` directory with `.pforge` fixtures.

**Documentation impact:** None at v1 — `docs/studio.md` covers the studio at the ImGui-migration level; the new workspaces are discoverable in-product. Post-v1 `/ce-compound` should capture the archetype-template + verb-recipe convention as a `docs/solutions/` entry (flagged in the learnings researcher's recommendations).

**Operational / rollout:** Standard release flow. No data migration required. Existing projects load with default Archetype + empty bindings; designers opt in by editing entities. No feature flag — the inspector simply branches on archetype presence.
