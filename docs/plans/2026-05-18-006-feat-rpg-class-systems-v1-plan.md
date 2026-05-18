---
title: "feat: RPG-class systems — Save UI + Dialogue + Menus + Inventory (idea #6 v1)"
type: feat
status: active
date: 2026-05-18
depth: deep
origin: docs/brainstorms/2026-05-18-rpg-class-systems-v1-requirements.md
ideation: docs/ideation/2026-05-18-pixelforge-nes-class-no-code-ideation.md (idea #6)
ships_with:
  - docs/plans/2026-05-18-001-feat-entity-verb-sheet-v1-plan.md (idea #5 — strict dependency on blackboard + verb catalog + input intents)
  - docs/plans/2026-05-18-002-feat-screenroom-mario-strip-v1-plan.md (idea #1)
  - docs/plans/2026-05-18-003-feat-tileatlas-emergent-rules-v1-plan.md (idea #2)
  - docs/plans/2026-05-18-004-feat-nes-palette-art-director-v1-plan.md (idea #3 — template palette refs)
  - docs/plans/2026-05-18-005-feat-audio-library-picker-v1-plan.md (idea #4)
related_plans:
  - (idea #7 plan not yet written — Capsule + Build pipeline; required for shipped-binary save loop)
---

# feat: RPG-class systems v1 (idea #6)

## Summary

v1 ships three coupled deliverables on top of idea #5's blackboard primitive: (1) **Save UI** — 3 named slots + 1 autosave; save unit = blackboard snapshot + current scene + active-scene entity state; saved to `os.UserConfigDir() / "pixelforge-games" / <sanitized-game-title> / slot{n}.json` (WASM: browser `localStorage`); save format is JSON with `SchemaVersion: 1` + additive + sanitize-on-load discipline (**overrides brainstorm's "forward-incompatible" stance** per `editor-pforge-schema-shape.md` convention — low cost, high forward-compat value). (2) **Dialogue system** — new engine package `pixelforge_dialogue` (parser + tree + runtime text-box renderer using `pixelforge_cofont`); new studio Dialogue workspace with `imgui.InputTextMultiline` script editor; screenplay-style syntax (`SPEAKER: text`) + Twine-style labels (`:: label`) + choices (`[[text -> label]]`) + conditional jumps (`[[text -> label | if state.flag]]`) + `{state.key}` interpolation + stage directions (5 in v1: `walk_left N`, `walk_right N`, `walk_up N`, `walk_down N`, `pause N`). (3) **Menus + Inventory** — new engine package `pixelforge_menus` (template registry + 9 NES-canonical renderers + nav state machine consuming `pixelforge_input` intents); studio Menus workspace (template picker + parameter editor); flat ItemDatabase via new `Project.Items []ItemDefinition`; Items workspace (table editor with new `SpriteThumbnailWidget`); inventory state on blackboard `inventory` key as `[]InventoryEntry{item_id, count}`. Plus ~12 new verb recipes wired into idea #5's catalog (wires placeholder shells `open_dialogue`, `open_menu`, `give_item`, `take_item` + adds `close_dialogue`, `close_menu`, `has_item` (condition), `set_item_count`, `save_now`, `load_slot`, `delete_slot`). New scene-pause primitive (greenfield — no precedent) blocks `EventUpdate` dispatch to rules; keeps `EventLateDraw` + input flowing. Three new dockable workspaces (Dialogue, Items, Menus) wired to View menu (Ctrl+5/6/7). Zero external dependencies. **Largest plan in this milestone** — 12 units, three coupled engine packages + three new studio workspaces.

---

## Leverage Doctrine (applied)

Per `docs/plans/2026-05-18-001-feat-entity-verb-sheet-v1-plan.md`'s Leverage Doctrine appendix.

**Candidates evaluated:**

| Candidate | Status | Verdict |
|---|---|---|
| Dialogue authoring tools (Yarn, Ink, Twine, Articy) | Mature, established | **Skip — port the syntax, not the runtime.** Screenplay + Twine syntax is what the brainstorm specifies; Yarn/Ink runtimes are 5,000+ LOC each. Hand-rolled recursive-descent parser for our restricted grammar is ~200 LOC. |
| Save/load libraries (encoding/gob, msgpack-go, etc.) | Mature | **Skip — JSON.** Repo convention is `encoding/json` (5 import sites, zero gob/msgpack). Save files in JSON are git-friendly during development; format determinism already established by `pixelforge_project/saver.go`. |
| Cross-platform user-config-dir libraries (`adrg/xdg`, `kirsle/configdir`) | Available | **Skip — `os.UserConfigDir()`** is in stdlib and already used at `settings.go:117`. |
| Menu UI framework libraries (Bubbletea-style TUI, immediate-mode game UI libs) | Various | **Skip — `pixelforge_gui` widgets + `pixelforge_cofont`** are the canvas-resident primitives the runtime already ships. Wrapping a TUI framework would require porting it into ebiten draw calls anyway. |
| Inventory management libraries (Go) | None mature for game inventories | **Build native** — Inventory is `map[string]any` slice on blackboard; ~50 LOC of verb wiring. |
| Dialogue parser generators (yacc/PEG/participle) | participle would work | **Skip** — restricted grammar is small enough that hand-rolled parser is cheaper than learning a parser-gen and maintaining the schema. |

Total custom: ~200 LOC dialogue parser + ~400 LOC dialogue+menu renderers + ~300 LOC studio workspaces + ~150 LOC save serializer + ~200 LOC verb-recipe wiring + supporting tests. Well below wrap costs for any candidate.

---

## Problem Frame

Pixelforge after ideas #1-#5 has the substrate for arcade and platformer games. Half the NES reference set is still unreachable:

- **Final Fantasy** (50% menus + 30% dialogue + 20% combat), **Zelda** (NPCs, inventory, save), **Metroid** (save points, items-as-flags), **Megaman** (passwords, stage select), **Punch-Out** (rosters), **Tetris** (high-score table), **Excitebike** (track-select). None ship-able without: persistent save state, dialogue trees, item databases, menu rendering.

Idea #5's blackboard gives Pixelforge the variable-store half of RPG state. The *content* layer — dialogue trees, menu templates, item databases, save UI — is missing. Specifically:

- **No save mechanism.** Players' progress vanishes when the game closes. The blackboard is in-memory; nothing serializes it to disk. The Capsule (idea #7) will ship binaries but those binaries currently have no save path.
- **No dialogue runtime.** `pixelforge` engine has font rendering (`pixelforge_cofont`); it has no concept of a dialogue tree, advance-on-A semantics, or `{state.key}` interpolation.
- **No menu renderer.** `pixelforge_gui` has buttons; it has no template-based menu rendering (Title screen, Inventory, Pause, etc.). Each game would build menus from scratch — defeating the no-code premise.
- **No inventory data model.** `AudioBinding` exists; `ItemBinding` doesn't. The blackboard could store items via convention, but nothing standardizes the verb interface (`give_item`, `has_item`).
- **No engine-side overlay pattern that ships.** Idea #3's overlays are studio-only by construction (paint into preview only); the runtime never sees them. Dialogue boxes and menu overlays need the OPPOSITE: render in both editor preview AND shipped binary. **This requires a new architectural pattern**: canvas-resident renderers that paint into the `ebiten.Image` from engine packages the Capsule imports.
- **No scene-pause primitive.** Per `docs/solutions/always-on-game-embedding.md`, the game is always ticking. Overlays (Pause/Inventory/Status) need to freeze the underlying scene while remaining visible and responsive — greenfield work.

The brainstorm's bet: a designer authors a Zelda-class NPC interaction (5 lines of screenplay + one branch) in under 10 minutes, never edits JSON, never touches a node-graph editor. A second designer authors an FF-class inventory menu in under 15 minutes. A third designer adds save points and ships a binary their classmate plays across multiple sessions. All three workflows compose from the same three subsystems: dialogue, menus, save.

---

## Carried Forward from Origin

All 26 requirements, 8 acceptance examples, 4 flows, 5 actors from origin are in scope.

| Origin | Scope summary | Plan unit(s) |
|---|---|---|
| R1-R5, R26 | Save UI: 3 slots + 1 autosave; user-config-dir; save unit + restore determinism | U1 (save engine), U7 (menu integration for Save/Load templates), U12 (E2E) |
| R6-R10, R23 | Dialogue: workspace, screenplay syntax, Twine branching, interpolation, runtime text-box, stage directions | U5 (engine package), U6 (studio workspace), U12 |
| R11-R14 | Menus: 9 templates, scene + verb application, parameter editor, D-pad/A/B nav | U7 (engine package), U8 (studio workspace) |
| R20-R22 | Inventory: ItemDatabase, Items workspace, inventory on blackboard | U9 |
| R24 | Schema additions (Dialogues, Menus, Items, SaveConfig) — additive omitempty | U4 |
| R25 | Verb-recipe additions (~12 new) + condition support | U10 |
| AE1-AE8, F1-F4 | All eight acceptance examples + four flows | U12 |
| A1-A5 | Designer, power-user designer, Studio, shipped game, end-player | All units |

Origin's "Deferred to Planning" section: all 11 technical/design questions resolved in Phase 2. Three discovered questions from research (Blackboard.Snapshot missing; Recipe schema is action-only; scene-pause greenfield) are resolved in plan units U2, U10, U3 respectively.

---

## High-Level Technical Design

How the three subsystems compose, with the editor-vs-runtime split made explicit:

```
                  AUTHORING (studio chrome via cimgui-go — editor-only)
                  ═══════════════════════════════════════════════════════════
   ┌──────────────────────────────────────────────────────────────────────┐
   │ Dialogue Workspace (NEW, U6)                                         │
   │  ┌────────────────────────┐ ┌──────────────────────────────────┐     │
   │  │ Tree list (left)       │ │ Multi-line script editor (right) │     │
   │  │  ─ old_man_hint        │ │ imgui.InputTextMultiline         │     │
   │  │  ─ goblin_intro        │ │ KING: (bows) Welcome {state.name} │     │
   │  │  ─ shopkeeper          │ │ [[Accept -> accept]]              │     │
   │  │  + New tree            │ │ [[Decline -> decline | if hp>0]]  │     │
   │  └────────────────────────┘ └──────────────────────────────────┘     │
   │                                                                      │
   │ On change: re-parse → validate → update Project.Dialogues[name].Tree │
   │ Errors render under editor as ParseError list with line numbers      │
   └──────────────────────────────────────────────────────────────────────┘
   ┌──────────────────────────────────────────────────────────────────────┐
   │ Items Workspace (NEW, U9)                                            │
   │ Table: ID | Name | Icon | Description | Effect verb | Category       │
   │  potion | Potion | [thumbnail] | "Restores HP" | restore_health(50) │
   │   | potion                                                           │
   │  sword  | Sword  | [thumbnail] | "Heroic blade" | attack(10)         │
   │   | weapon                                                           │
   │ + New Item                                                           │
   │                                                                      │
   │ Icon column uses NEW SpriteThumbnailWidget (lookup sprite by name)   │
   └──────────────────────────────────────────────────────────────────────┘
   ┌──────────────────────────────────────────────────────────────────────┐
   │ Menus Workspace (NEW, U8)                                            │
   │ Menu list (left) + parameter editor (right)                          │
   │  ─ title_screen   [Title template]                                  │
   │     game_name: "Adventure", subtitle: "...", bg_palette: bg_3       │
   │  ─ pause_inventory [Inventory template]                             │
   │     bg_palette: bg_0, text_color_slot: 7, category_filter: ""      │
   │  + New Menu                                                          │
   └──────────────────────────────────────────────────────────────────────┘

                  SHARED ENGINE LAYER (runs in BOTH editor preview AND shipped binary)
                  ═══════════════════════════════════════════════════════════
   ┌──────────────────────────────────────────────────────────────────────┐
   │ pixelforge_dialogue (NEW, U5)                                        │
   │  ─ Parser: script string → Tree{Nodes, Labels}                       │
   │  ─ Runtime: TextBoxRenderer{tree, currentNode, currentLine, ...}     │
   │     Update() — A advances; D-pad navigates choices                   │
   │     Draw(dst *ebiten.Image) — paints text box at bottom              │
   │       uses pixelforge_cofont.Print + pixelforge_gui rect helpers     │
   │     consumes pixelforge_input.IntentUse, IntentDown, IntentUp        │
   │  ─ Stage directions: walk_left/right/up/down N, pause N (v1; ~5)    │
   │     emit verb_recipe.Apply calls when matched; else italic flavor   │
   └──────────────────────────────────────────────────────────────────────┘
   ┌──────────────────────────────────────────────────────────────────────┐
   │ pixelforge_menus (NEW, U7)                                           │
   │  ─ Template registry: 9 templates                                    │
   │     Title, Game-Over, High-Score, Pause, Save-Game, Load-Game,       │
   │     Inventory, Status, Stage-Select                                  │
   │  ─ Per template: Render(dst, params, blackboard) + Nav(intents)      │
   │  ─ MenuStack: open_menu/close_menu push/pop; rendered top-to-bottom  │
   │  ─ Pause-on-overlay: when an overlay-type menu opens, calls          │
   │     SceneEngine.Pause(); when last overlay closes, Resume()          │
   └──────────────────────────────────────────────────────────────────────┘
   ┌──────────────────────────────────────────────────────────────────────┐
   │ pixelforge_save (NEW, U1)                                            │
   │  ─ Backend interface (Native fs / WASM localStorage)                 │
   │  ─ Snapshot{SchemaVersion, GameTitle, Timestamp, Blackboard,         │
   │     CurrentSceneID, PlayerPos, SceneEntities[]}                      │
   │  ─ SaveToSlot(n int) / LoadFromSlot(n int) / DeleteSlot(n int)     │
   │  ─ Autosave throttle (1 per 30s globally)                            │
   │  ─ Path: UserConfigDir/pixelforge-games/<sanitized-title>/           │
   │     slot{n}.json or autosave.json                                    │
   └──────────────────────────────────────────────────────────────────────┘
   ┌──────────────────────────────────────────────────────────────────────┐
   │ pixelforge_blackboard.Snapshot/Restore (AMENDS idea #5 U1, U2)       │
   │  ─ Snapshot() map[string]any (deep-copy under RWMutex)              │
   │  ─ Restore(m map[string]any) (atomic replace + publish change events)│
   └──────────────────────────────────────────────────────────────────────┘
   ┌──────────────────────────────────────────────────────────────────────┐
   │ SceneEngine.Pause()/Resume() (NEW primitive, U3)                     │
   │  ─ Gate inside Engine: when paused, blocks EventUpdate dispatch      │
   │     to behavior graphs + event-sheet rules                           │
   │  ─ EventLateDraw + input events CONTINUE (overlays render + respond) │
   └──────────────────────────────────────────────────────────────────────┘

                  VERB CATALOG INTEGRATION (idea #5's catalog seam)
                  ═══════════════════════════════════════════════════════════
   ┌──────────────────────────────────────────────────────────────────────┐
   │ pixelforge_studio/scripting/catalog/builtin_rpg.go (NEW, U10)        │
   │  init() {                                                             │
   │    RegisterAction("open_dialogue", buildOpenDialogue)                 │
   │    RegisterAction("close_dialogue", buildCloseDialogue)               │
   │    RegisterAction("open_menu", buildOpenMenu)                         │
   │    RegisterAction("close_menu", buildCloseMenu)                       │
   │    RegisterAction("give_item", buildGiveItem)                         │
   │    RegisterAction("take_item", buildTakeItem)                         │
   │    RegisterAction("set_item_count", buildSetItemCount)                │
   │    RegisterAction("save_now", buildSaveNow)                           │
   │    RegisterAction("load_slot", buildLoadSlot)                         │
   │    RegisterAction("delete_slot", buildDeleteSlot)                     │
   │    RegisterCondition("has_item", buildHasItem)                        │
   │  }                                                                    │
   │                                                                       │
   │  Plus: Recipe schema extension (amends idea #5 U4 verb_recipes.go)   │
   │   add ConditionKind string (mutually exclusive with ActionKind)      │
   └──────────────────────────────────────────────────────────────────────┘

                  RUNTIME COMPOSITION (in sceneGame.Draw / Capsule.Run)
                  ═══════════════════════════════════════════════════════════

  sceneGame.Update() / Capsule.Run() each tick:
    1. process input (intents fire from pixelforge_input)
    2. if !engine.Paused: dispatch EventUpdate (entities + rules)
    3. else: skip rule dispatch (only overlays update)
    4. dialogueRenderer.Update(intents)
    5. menuStack.Update(intents)
    6. always: dispatch EventLateDraw

  sceneGame.Draw(dst *ebiten.Image) / Capsule.Draw():
    1. tilemap.Render(dst) (from idea #1)
    2. entity.RenderAll(dst) (from idea #1)
    3. dialogueRenderer.Draw(dst) (NEW, U5 — bottom of screen)
    4. menuStack.Draw(dst) (NEW, U7 — full or overlay per template)
    5. (in editor only) studio overlays: scanline, palette-block (from idea #3)
```

*This illustrates the intended approach and is directional guidance for review, not implementation specification.*

The structural insight: **canvas-resident engine renderers ship to both editor preview and shipped runtime by construction**. The shipped binary's `Capsule.Run` calls the same `sceneGame.Draw` equivalent that the studio's preview does; ImGui in the studio just samples the resulting `ebiten.Image` as a texture for the panel. Dialogue boxes and menu overlays are visible identically. This is the **opposite** structural pattern from idea #3's overlays (which were studio-only by construction).

---

## Output Structure

```
pixelforge_dialogue/                              (NEW package, U5)
├── parser.go                                     — script → tree
├── parser_test.go
├── tree.go                                       — Node, Label, Choice types
├── renderer.go                                   — TextBoxRenderer (Update/Draw)
├── renderer_test.go
├── interpolator.go                               — {state.key} resolution
├── interpolator_test.go
├── stage_directions.go                           — parse + dispatch to verbs
├── stage_directions_test.go
└── doc.go

pixelforge_menus/                                 (NEW package, U7)
├── registry.go                                   — RegisterTemplate; 9 templates
├── stack.go                                      — MenuStack push/pop, pause coordination
├── stack_test.go
├── templates/                                    — one file per template
│   ├── title.go
│   ├── game_over.go
│   ├── high_score.go
│   ├── pause.go
│   ├── save_game.go
│   ├── load_game.go
│   ├── inventory.go
│   ├── status.go
│   └── stage_select.go
├── templates_test.go
├── nav.go                                        — selection cursor state machine
├── nav_test.go
└── doc.go

pixelforge_save/                                  (NEW package, U1)
├── snapshot.go                                   — Snapshot struct + (un)marshal
├── snapshot_test.go
├── backend.go                                    — Backend interface
├── backend_native.go                             — os.UserConfigDir + os.WriteFile
├── backend_native_test.go
├── backend_wasm.go                               — localStorage; build tag js
├── backend_wasm_test.go                          (browser test stub)
├── service.go                                    — SaveToSlot/LoadFromSlot/DeleteSlot + autosave throttle
├── service_test.go
├── paths.go                                      — sanitize + path derivation
├── paths_test.go
└── doc.go

pixelforge_blackboard/                            (MODIFY, U2 — amends idea #5 U1)
├── blackboard.go                                 — add Snapshot/Restore methods
└── blackboard_test.go                            — add Snapshot/Restore coverage

pixelforge_loop/                                  (MODIFY, U3 — adds pause primitive)
└── engine.go                                     — Pause/Resume on Engine; gate EventUpdate dispatch

pixelforge_project/
├── project.go                                    (MODIFY, U4) — add Dialogues, Menus, Items, SaveConfig fields
├── rpg_schema.go                                 (NEW, U4)   — DialogueScript, MenuConfig, ItemDefinition, SaveConfig types
├── rpg_schema_test.go                            (NEW, U4)
└── rpg_defaults.go                               (NEW, U4)   — applyDefaults for new fields

pixelforge_studio/scripting/catalog/
├── verb_recipes.go                               (MODIFY, U10 — amends idea #5 U4) — Recipe.ConditionKind field; LookupRecipe returns Action OR Condition
├── builtin_rpg.go                                (NEW, U10) — register ~12 new recipes/actions/conditions
└── builtin_rpg_test.go                           (NEW, U10)

pixelforge_studio/dialogue/                       (NEW package, U6)
├── workspace.go                                  — Workspace impl + RegisterWith(e)
├── workspace_test.go
├── script_editor.go                              — multi-line text input
├── script_editor_test.go
├── tree_list.go                                  — list of dialogue trees
└── parse_errors_view.go                          — render parse errors

pixelforge_studio/items/                          (NEW package, U9)
├── workspace.go                                  — Workspace impl + RegisterWith(e)
├── workspace_test.go
├── table_editor.go                               — table-style row editor
└── table_editor_test.go

pixelforge_studio/menus/                          (NEW package, U8)
├── workspace.go                                  — Workspace impl + RegisterWith(e)
├── workspace_test.go
├── menu_list.go                                  — list of menus per project
├── template_picker.go                            — choose template + parameters
└── parameter_editor.go                           — per-template parameter UI

pixelforge_studio/editor/widgets/
├── sprite_thumbnail.go                           (NEW, U9) — render sprite by name as preview
└── sprite_thumbnail_test.go

pixelforge_studio/editor/
├── file_menu.go                                  (MODIFY, U11) — add Dialogue/Items/Menus to View; routing
├── keymap.go                                     (MODIFY, U11) — add workspace.dialogue/items/menus Ctrl+5/6/7
└── editor.go                                     (MODIFY, U3) — wire engine pause/resume into editor pipeline

pixelforge_studio/main.go                         (MODIFY, U6, U8, U9) — add RegisterWith calls for the three new workspaces

pixelforge_studio/integration_test/
├── rpg_e2e_test.go                               (NEW, U12)
└── fixtures/
    ├── npc_dialogue_project.pforge               (NEW — fixture for F1)
    ├── inventory_menu_project.pforge             (NEW — fixture for F2)
    ├── save_point_project.pforge                 (NEW — fixture for F3)
    ├── title_screen_project.pforge               (NEW — fixture for F4)
    └── pre_v1_no_rpg_fields.pforge               (NEW — for AE7 regression)
```

Implementer may consolidate or split files; per-unit `**Files:**` sections remain authoritative.

---

## Implementation Units

Dependency-ordered. Foundation primitives (U1-U3) → schema (U4) → dialogue (U5, U6) → menus (U7, U8) → items+inventory (U9) → verb wiring (U10) → integration (U11) → tests (U12).

### U1. `pixelforge_save` package — snapshot serializer + slot file IO + paths + autosave throttle

**Goal:** New engine package that defines the save snapshot schema, serializes to JSON with `SchemaVersion`, writes to/reads from per-OS user-config-dir, supports 3 named slots + 1 autosave, throttles autosave to at most once per 30s, and exposes a swappable `Backend` interface (Native fs / WASM localStorage).

**Requirements:** R1 (3+1 slot config), R2 (user-config-dir per game title), R3 (save unit shape), R4 (Save/Load templates use this), R5 (autosave + throttle), R26 (deterministic restore).

**Dependencies:** none (foundational); pairs with U2 for Blackboard Snapshot/Restore.

**Files:**
- `pixelforge_save/snapshot.go` (NEW — Snapshot struct + JSON marshal/unmarshal)
- `pixelforge_save/snapshot_test.go` (NEW)
- `pixelforge_save/backend.go` (NEW — Backend interface)
- `pixelforge_save/backend_native.go` (NEW — os.UserConfigDir + os.WriteFile impl)
- `pixelforge_save/backend_native_test.go` (NEW)
- `pixelforge_save/backend_wasm.go` (NEW — build tag `//go:build js` — localStorage impl)
- `pixelforge_save/backend_wasm_test.go` (NEW — placeholder; full browser test deferred)
- `pixelforge_save/service.go` (NEW — SaveToSlot/LoadFromSlot/DeleteSlot/Autosave)
- `pixelforge_save/service_test.go` (NEW)
- `pixelforge_save/paths.go` (NEW — sanitize + path derivation)
- `pixelforge_save/paths_test.go` (NEW)
- `pixelforge_save/doc.go` (NEW)

**Approach:**
- `Snapshot` struct:
  - `SchemaVersion int                     ` (= 1 in v1)
  - `GameTitle string`
  - `SavedAt time.Time`
  - `Blackboard map[string]any             ` (from `pixelforge_blackboard.Snapshot()`)
  - `CurrentSceneID string`
  - `PlayerPos struct{TileX, TileY int}    ` (player entity's tile cell)
  - `SceneEntities []EntitySnapshot        ` (per-entity for current scene: ID, TileX/Y, Components.Values delta — for v1 just store full Values)
- JSON marshal with `SetIndent("", "  ")` + `SetEscapeHTML(false)` — mirrors `pixelforge_project/saver.go:24-45` for git-diff stability.
- `Backend` interface: `Read(slot string) ([]byte, error)`, `Write(slot string, data []byte) error`, `Delete(slot string) error`, `List() ([]SlotMeta, error)`.
- `BackendNative`: builds `os.UserConfigDir() / "pixelforge-games" / sanitize(gameTitle) /` directory; reads/writes `slot{1,2,3}.json` and `autosave.json`. Creates dir on first write (`MkdirAll(..., 0o755)`).
- `BackendWASM`: `js.Global().Get("localStorage")` Set/Get/Remove. Key format: `pixelforge-games:<sanitized-title>:slot{n}`. Localstorage size limit ~5MB; throw if exceeded.
- `Service` wraps Backend with throttle state for autosave:
  - `SaveToSlot(snapshot Snapshot, slotName string)` — writes named slot; bypasses throttle.
  - `LoadFromSlot(slotName string) (Snapshot, error)` — reads + JSON-unmarshal + sanitize.
  - `DeleteSlot(slotName string) error`
  - `Autosave(snapshot Snapshot) error` — checks `time.Since(lastAutosave) >= 30*time.Second`; if not, returns nil (no-op silently); else writes to "autosave" slot + updates `lastAutosave`.
- **`SchemaVersion`-based forward compat** (overrides brainstorm's "forward-incompatible" stance): on load, if `SchemaVersion < currentVersion`, apply `migrateSnapshot()` (no-op in v1; future versions add migration steps). On load with unknown future version, log warning + attempt best-effort load.
- `paths.go`: `Sanitize(gameTitle) string` — lowercase + replace spaces with `_` + strip non-`[a-z0-9-_]`. Empty result → "untitled".
- Save files are JSON, additive `omitempty`, sanitize-on-load per `editor-pforge-schema-shape.md` discipline — overrides brainstorm's R-section "forward-incompatible" stance.

**Patterns to follow:** `pixelforge_project/saver.go:24-80` for JSON-save discipline; `pixelforge_studio/editor/settings.go:117` for `os.UserConfigDir` usage; `pixelforge_studio/editor/imgui_theme.go:221` for path-join pattern; `pixelforge_audio/decode.go` for failure-mode discipline (logs + skips, never panics).

**Test scenarios:**
- `TestSnapshot_RoundTripsJSON`: serialize Snapshot with all fields populated; deserialize; field-by-field equal.
- `TestSnapshot_SchemaVersionPresent`: marshaled JSON contains `"schema_version": 1`.
- `TestSnapshot_FutureSchemaVersionLogsWarning`: load JSON with `"schema_version": 99`; logs warning; returns snapshot with best-effort fields filled.
- `TestSnapshot_MissingKeyDefaults`: load JSON missing `PlayerPos`; resulting struct has zero-value Position; no error.
- `TestPaths_SanitizeRemovesSpaces`: `Sanitize("My Game")` → `"my_game"`.
- `TestPaths_SanitizeStripsSpecialChars`: `Sanitize("Hero's-Quest 2!")` → `"heros-quest_2"`.
- `TestPaths_SanitizeEmptyReturnsUntitled`: `Sanitize("")` → `"untitled"`.
- `TestPaths_NativeBackendPath`: on Linux, derives `~/.config/pixelforge-games/my_game/slot1.json` (or `$XDG_CONFIG_HOME` if set).
- `TestBackendNative_WriteThenRead`: write bytes to "slot1"; read returns same bytes.
- `TestBackendNative_CreatesDirIfMissing`: write to a slot in a path that doesn't exist; dir created with 0755 perms; file written.
- `TestBackendNative_DeleteRemovesFile`: write + delete; file no longer exists; subsequent List doesn't show slot.
- `TestBackendNative_ListReturnsSavedSlots`: after writing slot1 + autosave; List returns [{slot1, mtime}, {autosave, mtime}].
- `TestService_SaveAndLoad`: SaveToSlot snapshot to "slot1"; LoadFromSlot("slot1") returns equal snapshot.
- `TestService_AutosaveThrottleSkipsWithinWindow`: Autosave; sleep 5s; Autosave again; second call no-ops (file mtime unchanged).
- `TestService_AutosaveAcceptedAfterWindow`: Autosave; mock clock advance 31s; Autosave again; file mtime updated.
- `TestService_SaveToSlotBypassesThrottle`: SaveToSlot 5 times in a second; all succeed; file mtime updates each time.
- `TestSnapshot_EntitySnapshotsScopedToCurrentScene`: snapshot with 3 entities in scene_a + 5 in scene_b + CurrentScene=scene_a; SceneEntities contains only the 3 from scene_a.
- `TestSnapshot_PlayerPosIsTileCells`: snapshot with PlayerPos{TileX: 5, TileY: 24}; round-trips as integers.
- `TestBackendWASM_StorageKeyFormat`: WASM Backend uses key "pixelforge-games:<title>:slot{n}".
- `TestBackendWASM_ExceedsQuotaReturnsError`: write 10MB; returns quota-exceeded error (mocked).
- Covers AE1 (file written to user-config-dir).

**Verification:** `go test ./pixelforge_save/...` passes; manual smoke (after U7/U12 wire the Save template): trigger save; verify `~/.config/pixelforge-games/<title>/slot1.json` exists with valid JSON; reload game; data restored.

---

### U2. Blackboard Snapshot/Restore (amends idea #5's `pixelforge_blackboard`)

**Goal:** Add `Snapshot() map[string]any` and `Restore(map[string]any)` methods to idea #5's Blackboard. Snapshot deep-copies all keys under existing RWMutex. Restore atomically replaces all keys and publishes change events for each.

**Requirements:** R3 (blackboard snapshot IS save unit), R26 (deterministic restore).

**Dependencies:** strict on idea #5's plan U1 (Blackboard package must exist).

**Files:**
- `pixelforge_blackboard/blackboard.go` (MODIFY — add Snapshot/Restore methods)
- `pixelforge_blackboard/blackboard_test.go` (MODIFY — add Snapshot/Restore tests)

**Approach:**
- `Snapshot()`: acquires `bb.mu.RLock()`; iterates all keys; deep-copies values. For values of type `int`, `string`, `bool`, `float64` — simple copy. For `map[string]any` or `[]any` — recursive deep-copy. For arbitrary types (unlikely in v1 but possible) — use `json.Marshal`+`json.Unmarshal` round-trip as a safe deep-copy. Returns a new `map[string]any`.
- `Restore(m map[string]any)`: acquires `bb.mu.Lock()`; clears existing keys; copies provided map into internal store; publishes change event for each key (so subscribers re-react). Returns nothing.
- **Atomicity:** Restore is single-write-lock atomic — observers either see the old state OR the fully-restored new state, never a partial intermediate.
- **Change-event publication:** Snapshot doesn't publish (read-only). Restore publishes for EVERY new key + every key that existed and was removed.

**Patterns to follow:** existing `pixelforge_blackboard` package shape from idea #5's plan U1; existing pievent target publish pattern.

**Test scenarios:**
- `TestSnapshot_EmptyBlackboardReturnsEmptyMap`: empty BB; Snapshot returns `{}`.
- `TestSnapshot_PopulatedBlackboardReturnsAllKeys`: BB with 3 keys; Snapshot returns map with those 3 entries.
- `TestSnapshot_DeepCopyOfMaps`: BB has key="inventory" with value `[]any{"sword"}`; Snapshot; mutate returned map's inventory slice; original BB unchanged.
- `TestSnapshot_DoesNotPublishEvents`: subscribe to BB changes; call Snapshot; subscriber received no events.
- `TestRestore_ReplacesAllKeys`: BB has {a:1, b:2}; Restore({c:3, d:4}); BB.Get("a") returns not-found; BB.Get("c") returns 3.
- `TestRestore_PublishesChangeEventForEveryKey`: subscribe to BB changes; Restore({a:1, b:2}); subscriber received 2 change events (a, b).
- `TestRestore_PublishesEventForRemovedKeys`: BB has {a:1}; Restore({}); subscriber received change event for "a" (with value nil or some "removed" sentinel).
- `TestSnapshotRestore_RoundTripPreservesState`: populate BB; snapshot; populate differently; Restore(snapshot); BB matches original.
- `TestRestore_ConcurrentReadDuringRestoreSafe`: race-condition test with go test -race; goroutine reads BB.Get while another calls Restore; no race.

**Verification:** `go test ./pixelforge_blackboard/... -race` passes; existing idea #5 blackboard tests still pass.

---

### U3. Scene-pause primitive (greenfield engine extension)

**Goal:** Add `Pause()` and `Resume()` methods to the scene's runtime engine (likely `pixelforge_loop.Engine` or a `*sceneGame`-level gate). When paused, blocks dispatch of `EventUpdate` to behavior graphs and event-sheet rules. `EventLateDraw` and input events continue (so overlays can render and receive D-pad input).

**Requirements:** R12 (overlays pause underlying scene), R10 (dialogue advance via input intent while underlying scene paused).

**Dependencies:** none (engine-side, parallel with U1/U2).

**Files:**
- `pixelforge_loop/engine.go` (MODIFY — add Pause/Resume + paused gate inside Update dispatch)
- `pixelforge_loop/engine_test.go` (MODIFY)
- `pixelforge_studio/editor/canvas_input.go` (MODIFY — sceneGame.Update respects engine.Paused() for entity ticks)
- `pixelforge_studio/editor/canvas_input_test.go` (MODIFY)

**Approach:**
- `Engine.Pause()`: sets internal `paused = true`. Idempotent.
- `Engine.Resume()`: sets `paused = false`. Idempotent.
- `Engine.IsPaused() bool`: read accessor.
- Inside `Engine.Update()` (or wherever `EventUpdate` is dispatched): if paused, skip the dispatch loop but DO continue dispatching `EventLateDraw` (so overlays paint) and `EventInput` (so D-pad nav works inside overlay).
- **What pauses:** behavior graph rules, event-sheet rule evaluation, entity Update steps (`Tick()`), routine engine ticks (`pixelforge_routine.Engine.Tick()`).
- **What continues:** `EventLateDraw` dispatch (overlays + scene preview re-render with old state), input event dispatch (so menu nav + dialogue-advance work), the visible game (renders, just doesn't tick), audio playback (BGM keeps looping; SFX queued during pause play when resumed).
- This is a per-scene engine gate; overlays signal pause/resume via the menu stack (U7) on overlay open/close.

**Patterns to follow:** existing `scripting/debugger.go:79-85` + `scripting/runtime/engine.go:75-94` for the existing breakpoint-pause primitive — same shape, but for game scenes instead of script debugging.

**Test scenarios:**
- `TestEngine_PauseStopsUpdateDispatch`: subscribe to EventUpdate; Pause; tick; subscriber receives no event.
- `TestEngine_PauseAllowsLateDrawDispatch`: subscribe to EventLateDraw; Pause; tick; subscriber DOES receive event.
- `TestEngine_PauseAllowsInputDispatch`: subscribe to input intent; Pause; fire intent; subscriber receives.
- `TestEngine_ResumeRestartsUpdate`: Pause; tick (no Update); Resume; tick; subscriber receives Update event.
- `TestEngine_PauseIdempotent`: Pause; Pause; IsPaused returns true; one Resume call is enough.
- `TestEngine_PauseDuringScriptingEngineTick`: scripting engine has a rule scheduled; Pause; tick; rule does not fire.
- `TestSceneGame_UpdateRespectsEnginePause`: studio sceneGame.Update; engine paused; entity.Tick not called.

**Verification:** `go test ./pixelforge_loop/... ./pixelforge_studio/editor/...` passes; manual smoke (after U7/U12): open Pause menu; verify enemies stop moving but menu cursor responds.

---

### U4. Schema additions on Project — Dialogues, Menus, Items, SaveConfig

**Goal:** Additive `omitempty` fields on `Project`: `Dialogues map[string]DialogueScript`, `Menus map[string]MenuConfig`, `Items []ItemDefinition`, `SaveConfig SaveConfig`. New types defined in `rpg_schema.go`. `applyDefaults` ensures pre-v1 projects load cleanly with empty maps/slices and default save config.

**Requirements:** R24 (additive omitempty; pre-v1 projects load cleanly), AE7 (regression test).

**Dependencies:** none (foundational); independent of U1-U3.

**Files:**
- `pixelforge_project/project.go` (MODIFY — add 4 new fields)
- `pixelforge_project/rpg_schema.go` (NEW — type definitions)
- `pixelforge_project/rpg_schema_test.go` (NEW)
- `pixelforge_project/rpg_defaults.go` (NEW — applyDefaults helpers)
- `pixelforge_project/project.go` (MODIFY — `applyDefaults` calls the new helpers)

**Approach:**
- `DialogueScript`: `{Name string; Source string; Tree DialogueTree}` — Source is the raw script text (round-trippable); Tree is the parsed representation (rebuilt at load time from Source). JSON tag stores Source only; Tree marshaling omitted (regenerated).
- `MenuConfig`: `{Name string; Template string; Params map[string]any}` — Template names one of the 9 registered templates; Params holds template-specific fields (e.g., for Title: `{"game_name": "Adventure", "subtitle": "..."}`).
- `ItemDefinition`: `{ID, Name, IconSpriteRef, Description string; Effect string; Category string}` — Effect is a verb-recipe reference string (e.g., `"restore_health(50)"` — actually a recipe name with args); Category is one of `"weapon" | "potion" | "key" | "misc"` (enum; sanitize-clamp to "misc" if unknown).
- `SaveConfig`: `{Slots int; AutosaveTriggers []string}` — Slots fixed at 3 in v1 (sanitize-clamp to 3); AutosaveTriggers default `["scene_change"]`.
- `applyDefaults`: backfills empty maps/slices + default SaveConfig + clamps Items' Category to enum.
- All fields `omitempty` on JSON tags; pre-v1 projects (no fields present) load with backfilled defaults.
- DialogueScript's Tree: re-parse Source on load via `pixelforge_dialogue.Parse(Source)` (from U5). If parse fails, store empty Tree + log warning (per `editor-pforge-schema-shape.md` — never panic on malformed).

**Patterns to follow:** existing additive-omitempty pattern (e.g., `Theme.SanitizeSlots` called from project.go:107-113); existing JSON-tag discipline; existing `normalizeSlices` for nil-backfill (project.go:119-159); `editor-pforge-schema-shape.md` defensiveness.

**Test scenarios:**
- `TestProject_Pre_V1_LoadsWithEmptyRPGFields`: load JSON without `dialogues`/`menus`/`items`/`save_config`; after applyDefaults, all 4 fields are zero/empty; no errors. (Covers AE7.)
- `TestProject_RoundTripWithRPGFields`: project with 2 dialogues + 3 menus + 5 items + custom SaveConfig; marshal + unmarshal; field-by-field equal.
- `TestProject_ItemCategoryClampedToEnum`: load Item with Category="invalid"; after applyDefaults, Category="misc"; warning logged.
- `TestProject_SaveConfigDefaults`: pre-v1 project; SaveConfig defaults to `{Slots: 3, AutosaveTriggers: ["scene_change"]}`.
- `TestProject_DialogueScriptTreeRegeneratedOnLoad`: project with DialogueScript.Source="KING: hi"; after load, Tree is populated; Source preserved unchanged in subsequent re-save.
- `TestProject_DialogueScriptMalformedSourceLogsWarning`: DialogueScript.Source has parse error; after load, Tree is empty; warning logged; project still loads.
- `TestProject_OmitEmptyAllFields`: marshal default project (no RPG content); JSON does NOT contain `dialogues`/`menus`/`items`/`save_config` keys.
- `TestProject_NormalizeSlicesNilItemsBackfill`: project with nil Items slice; after normalizeSlices, Items is `[]ItemDefinition{}`.

**Verification:** `go test ./pixelforge_project/...` passes; round-trip of `editor.pforge` (the only existing fixture) produces zero key churn.

---

### U5. `pixelforge_dialogue` package — parser + tree + runtime text-box renderer

**Goal:** New engine package that parses screenplay-style scripts to a tree, exposes runtime `TextBoxRenderer` (Update/Draw using `pixelforge_cofont`), supports Twine-style branching + `{state.key}` interpolation + 5 stage directions. Runs in BOTH editor preview AND shipped runtime (paints into ebiten.Image).

**Requirements:** R6 (one workspace hosts many trees), R7 (screenplay-style + round-trip), R8 (Twine branching + conditions), R9 (interpolation), R10 (runtime text box + A advances + choices via D-pad), R23 (5 stage directions parsed to verbs or italic flavor).

**Dependencies:** idea #5 plan U1 (blackboard — for interpolation + condition eval), idea #5 plan U3 (input intents — for advance/nav), idea #5 plan U4 (verb recipes — for stage-direction dispatch); pairs with U3 (pause primitive — dialogue open pauses underlying scene).

**Files:**
- `pixelforge_dialogue/parser.go` (NEW)
- `pixelforge_dialogue/parser_test.go` (NEW)
- `pixelforge_dialogue/tree.go` (NEW — Node, Label, Choice, StageDirection types)
- `pixelforge_dialogue/tree_test.go` (NEW)
- `pixelforge_dialogue/renderer.go` (NEW — TextBoxRenderer)
- `pixelforge_dialogue/renderer_test.go` (NEW)
- `pixelforge_dialogue/interpolator.go` (NEW — `{state.key}` resolution)
- `pixelforge_dialogue/interpolator_test.go` (NEW)
- `pixelforge_dialogue/stage_directions.go` (NEW)
- `pixelforge_dialogue/stage_directions_test.go` (NEW)
- `pixelforge_dialogue/doc.go` (NEW)

**Approach:**
- **Grammar (recursive descent):**
  ```
  script         := line*
  line           := label_decl | text_line | choice | comment | blank
  label_decl     := ':: ' identifier
  text_line      := speaker ': ' (stage_direction | text_segment)*
  stage_direction := '(' identifier arg* ')'
  choice         := '[[' text_segment ' -> ' identifier (' | if ' condition)? ']]'
  text_segment   := chars | interpolation | escaped
  interpolation  := '{' 'state.' identifier '}'
  escaped        := '\{' | '\}'
  condition      := expr (==|!=|<|>|>=|<=) literal
  ```
- Hand-rolled parser; ~200 LOC. Line-based with peek/expect helpers.
- `Tree`:
  - `Nodes []Node` (ordered) — Node is either TextLine, Choice list, or LabelTarget.
  - `LabelMap map[string]int` — label name to index in Nodes.
  - `EntryLabel string` — starts at first label OR "start" implicit label OR first text line.
- `TextBoxRenderer`:
  - State: `currentNodeIdx int`, `currentLineCharCount int` (for typewriter effect — deferred to v2; v1 shows full line at once), `choiceHighlightIdx int` when showing choices.
  - `Update(input *pixelforge_input.IntentEvent, bb *pixelforge_blackboard.Blackboard)`:
    - If showing text: IntentUse → advance to next node.
    - If showing choices: IntentUp/Down → cycle choiceHighlightIdx; IntentUse → follow chosen label.
    - On advance into a text line with stage directions: dispatch stage_direction.Apply(...) sequentially before rendering text.
  - `Draw(dst *ebiten.Image, bb *pixelforge_blackboard.Blackboard)`:
    - Paint a text box rectangle at the bottom of the screen (height ~48 pixels for 256x240, configurable).
    - Render speaker name in one row, then text with `{state.key}` resolved via `interpolator.Interpolate(text, bb)`.
    - If at a choice node: render numbered list above the text box with highlighted entry colored differently.
    - Use `pixelforge_cofont.Print` (existing engine font primitive) + `pixelforge_gui` rect helpers.
- `Interpolator.Interpolate(text string, bb *Blackboard) string`: replace `{state.key_name}` with `bb.Get("key_name")` value (coerced to string). Escaped braces `\{` and `\}` survive as literal `{` and `}`.
- `StageDirections.Apply(direction string, entity any, verbCatalog Catalog)`: matches known directions; dispatches verb recipe; unknown → returns flavor-only marker.
- **v1 stage directions** (5): `walk_left N`, `walk_right N`, `walk_up N`, `walk_down N`, `pause N` (where N is integer tiles or seconds). Match against this fixed list; anything else displays as italic flavor.
- **Condition evaluation** for `[[Choice -> label | if state.flag > 0]]`: parse simple binary expression `state.<key> <op> <literal>`. Eval against bb. v1 supports `==`, `!=`, `<`, `>`, `<=`, `>=` and bool/int/string literals.

**Patterns to follow:** existing `pixelforge_cofont.Print` API + `pixelforge_gui` rect-draw helpers; idea #5's blackboard read API; idea #5's input intent layer.

**Test scenarios:**
- **Parser:**
  - `TestParser_SingleLineScript`: parse `"MARIO: Hi."`; Tree has 1 TextLine node with speaker="MARIO", text="Hi.".
  - `TestParser_MultipleSpeakers`: parse 2 speaker lines; Tree has 2 nodes.
  - `TestParser_LabelDecl`: parse `":: start\nMARIO: hi"`; LabelMap has "start" → 0.
  - `TestParser_Choice`: parse `"[[Continue -> next]]"`; node is Choice with text="Continue", target="next".
  - `TestParser_ConditionalChoice`: parse `"[[Decline -> decline | if state.lives > 0]]"`; Choice has Condition struct.
  - `TestParser_Interpolation`: parse `"MARIO: Hi {state.name}."`; node's text segments include InterpolationSegment(key="name").
  - `TestParser_StageDirection`: parse `"MARIO: (walks left 4) Hi"`; node has StageDirection("walk_left", [4]) + text.
  - `TestParser_EscapedBrace`: parse `"MARIO: Use {{x}}"` or `"MARIO: Use \\{x\\}"` (per chosen escape syntax); literal `{x}` preserved.
  - `TestParser_RoundTripsLosslessly`: parse script S; serialize back to text; equal to S (covers R7).
  - `TestParser_ErrorOnUnclosedChoice`: parse `"[[Continue"` (missing `->...]]`); returns ParseError with line number.
  - `TestParser_ErrorOnUnknownDirective`: parse `":: bad label name with spaces"`; returns ParseError or label-name-sanitized.
- **Interpolator:**
  - `TestInterpolate_BlackboardHasKey`: text="Hi {state.name}", bb has name="Mario"; result="Hi Mario".
  - `TestInterpolate_MissingKeyShowsPlaceholder`: missing key; result has `?` or `{state.name}` literal (decision: render literal); no panic.
  - `TestInterpolate_NestedBracesNotInterpolated`: `"Hi {{state.x}}"` → literal `Hi {state.x}` (escape).
  - `TestInterpolate_NonStringValueCoerced`: bb has name=42; result="Hi 42".
- **Stage directions:**
  - `TestStageDirection_WalkLeftDispatchesVerb`: parser produces StageDirection("walk_left", [4]); Apply dispatches `move_with_intent` verb with appropriate args.
  - `TestStageDirection_UnknownDirectionFlagsAsFlavor`: parser produces StageDirection("unknown"); Apply returns `FlavorMarker`; renderer shows italic.
  - `TestStageDirection_PauseDispatchesPause`: `(pause 2)` dispatches the pause verb; entity waits 2 ticks before advancing.
- **TextBoxRenderer:**
  - `TestRenderer_AdvanceOnIntentUse`: tree at node 0; IntentUse; currentNodeIdx becomes 1.
  - `TestRenderer_ChoicesNavigateWithIntentUpDown`: at choice node; IntentDown; choiceHighlightIdx=1; IntentUp; 0.
  - `TestRenderer_IntentUseFollowsHighlightedChoice`: at choice; highlight=1; IntentUse; currentNodeIdx jumps to label of choice 1.
  - `TestRenderer_ConditionalChoiceHiddenWhenFalse`: choice has condition `state.x > 0`; bb has x=0; choice not rendered.
  - `TestRenderer_DrawProducesTextBoxAtBottom`: Draw with fake ebiten.Image; pixel-check that bottom 48px is non-empty.
  - `TestRenderer_NoLeadingChoiceWhenNotAtChoiceNode`: text-only node; rendering does NOT show choice list.
  - `TestRenderer_OnLoadResetsToEntry`: renderer.Reset(); currentNodeIdx becomes entry node.
  - Covers F1 (NPC dialogue), AE2 (full dialogue example), AE3 (advance/nav), AE6 (stage directions).

**Verification:** `go test ./pixelforge_dialogue/...` passes; integration smoke: parse AE2's example script; render via TextBoxRenderer; verify line+choice progression matches AE2.

---

### U6. Studio Dialogue workspace — script editor + tree list + parse errors view

**Goal:** New dockable Dialogue workspace. Left panel: list of dialogue trees per project (Name + line count). Right panel: `imgui.InputTextMultiline` script editor for the selected tree. On edit, re-parse via `pixelforge_dialogue.Parse(text)`; show ParseErrors as red text below the editor with line numbers. Save back to `Project.Dialogues[name].Source` on every edit (debounced); MarkDirty.

**Requirements:** R6 (dedicated workspace, one per tree per project).

**Dependencies:** U4 (Project.Dialogues schema), U5 (parser).

**Files:**
- `pixelforge_studio/dialogue/workspace.go` (NEW — Workspace impl + RegisterWith(e))
- `pixelforge_studio/dialogue/workspace_test.go` (NEW)
- `pixelforge_studio/dialogue/script_editor.go` (NEW)
- `pixelforge_studio/dialogue/script_editor_test.go` (NEW)
- `pixelforge_studio/dialogue/tree_list.go` (NEW)
- `pixelforge_studio/dialogue/parse_errors_view.go` (NEW)

**Approach:**
- `DialogueWorkspace`: standard `Workspace` interface (Name="dialogue", DisplayName="Dialogue", Render(e)).
- Render: `imgui.Begin("Dialogue")`; horizontal split — left=tree list, right=script editor.
- **Tree list**:
  - Walks `e.Project.Dialogues` (sorted alphabetically by name).
  - Each entry: name + clickable; clicked entry becomes "active."
  - Bottom: "+ New tree" button → opens text input for new name; creates `DialogueScript{Name: input, Source: "", Tree: empty}`; MarkDirty.
  - Per-entry right-click menu: Rename / Delete.
- **Script editor**:
  - When tree is active, render `imgui.InputTextMultiline` with current Source as buffer.
  - Buffer size: 16 KB initial; grows. Use cimgui-go's `InputTextMultiline` standard.
  - On every change: re-parse via `pixelforge_dialogue.Parse(buffer)`; update `Project.Dialogues[name].Source` + `.Tree`; MarkDirty (debounced — at most once per 500ms to avoid thrashing).
  - Below editor: `ParseErrorsView` — list ParseErrors with line numbers (e.g., "line 5: unclosed choice").
- Workspace registers via standard `RegisterWith(e)` pattern.

**Patterns to follow:** existing Workspace interface from `palette/workspace.go:187-192`; idea #4's plan U6 for the workspace+two-panel pattern; `dirty-state-ux.md` for MarkDirty discipline.

**Test scenarios:**
- `TestDialogueWorkspace_RegisteredWithEditor`: after RegisterWith, `e.GetWorkspace("dialogue")` non-nil.
- `TestDialogueWorkspace_ScriptEditorEmptyByDefault`: no active tree; right panel shows "Pick or create a tree" placeholder.
- `TestTreeList_NewTreeButtonCreatesEntry`: click + type name; `Project.Dialogues[name]` exists with empty Source; MarkDirty.
- `TestTreeList_RenameTreeUpdatesKey`: rename "old" to "new"; `Project.Dialogues["old"]` gone; `Project.Dialogues["new"]` present with same Source; MarkDirty.
- `TestTreeList_DeleteRemovesEntry`: delete tree; `Project.Dialogues[name]` removed; MarkDirty.
- `TestScriptEditor_EditUpdatesProjectSource`: type "MARIO: hi" into editor; `Project.Dialogues[name].Source` == "MARIO: hi"; MarkDirty after debounce.
- `TestScriptEditor_EditTriggersReparse`: type valid script; `Project.Dialogues[name].Tree` is populated (non-empty Nodes).
- `TestScriptEditor_ParseErrorsRenderedBelow`: type "[[Continue"; ParseErrorsView shows "line 1: unclosed choice".
- `TestScriptEditor_DebouncedSaveAtMostOncePer500ms`: rapid typing (10 events in 100ms); MarkDirty called at most 1-2 times (debounce).
- Covers F1's authoring steps (designer writes script in editor).

**Verification:** `go test ./pixelforge_studio/dialogue/...` passes; manual: open studio, View → Dialogue (Ctrl+5); create tree "test"; type "MARIO: hi"; save project; reopen; tree + source preserved.

---

### U7. `pixelforge_menus` package — template registry + 9 templates + nav state machine + menu stack

**Goal:** New engine package implementing 9 NES-canonical menu templates (Title, Game-Over, High-Score, Pause, Save-Game, Load-Game, Inventory, Status, Stage-Select). Each template renders to `*ebiten.Image` using `pixelforge_cofont` + `pixelforge_gui`. Menu stack supports push/pop (open_menu/close_menu) with pause coordination — overlay templates (Pause/Inventory/Status) call `engine.Pause()` on open; scene templates (Title/Game-Over/etc.) don't pause but ARE the active scene.

**Requirements:** R11 (9 templates), R12 (per-scene vs per-verb application; overlays pause), R13 (template-tweakable parameters), R14 (D-pad/A/B nav with intent layer).

**Dependencies:** idea #5 plan U3 (input intents), idea #5 plan U1 (blackboard), idea #3 plan U1 (palette refs for template params), U1 (pixelforge_save for Save/Load templates), U3 (pause primitive), U9 (Inventory items list).

**Files:**
- `pixelforge_menus/registry.go` (NEW)
- `pixelforge_menus/stack.go` (NEW)
- `pixelforge_menus/stack_test.go` (NEW)
- `pixelforge_menus/templates/title.go` (NEW)
- `pixelforge_menus/templates/game_over.go` (NEW)
- `pixelforge_menus/templates/high_score.go` (NEW)
- `pixelforge_menus/templates/pause.go` (NEW)
- `pixelforge_menus/templates/save_game.go` (NEW)
- `pixelforge_menus/templates/load_game.go` (NEW)
- `pixelforge_menus/templates/inventory.go` (NEW)
- `pixelforge_menus/templates/status.go` (NEW)
- `pixelforge_menus/templates/stage_select.go` (NEW)
- `pixelforge_menus/templates_test.go` (NEW — per-template behaviorals)
- `pixelforge_menus/nav.go` (NEW)
- `pixelforge_menus/nav_test.go` (NEW)
- `pixelforge_menus/doc.go` (NEW)

**Approach:**
- **Template interface:**
  ```
  type Template interface {
      Name() string
      Kind() TemplateKind   // KindScene or KindOverlay
      Render(dst *ebiten.Image, params map[string]any, bb *Blackboard, nav *NavState)
      HandleInput(intent IntentEvent, params, bb, nav) (transition Transition)
      DefaultParams() map[string]any
  }
  ```
- **TemplateKind:**
  - `KindScene`: Title, Game-Over, High-Score, Save-Game, Load-Game, Stage-Select. Applied as the entire scene (no underlying tilemap renders behind).
  - `KindOverlay`: Pause, Inventory, Status. Renders over the active scene; auto-pauses scene on push (via U3's Engine.Pause).
- **Registry**:
  - `RegisterTemplate(t Template)` — package-level. Called from each template's `init()`.
  - `LookupTemplate(name string) Template`.
- **Menu Stack** (per-game runtime singleton):
  - `Stack` has `[]openMenu` (each = template + params + nav state).
  - `Push(name, params)` — looks up template; pushes; if KindOverlay, calls `engine.Pause()`.
  - `Pop()` — pops; if no overlays remaining, calls `engine.Resume()`.
  - `Update(intent IntentEvent, bb *Blackboard)` — dispatches to top menu's HandleInput.
  - `Draw(dst *ebiten.Image, bb *Blackboard)` — draws all menus bottom-to-top.
- **NavState**: `{SelectionIndex int, Cursor [2]int, ...}` — per-template state for the cursor position.
- **Transitions**: `HandleInput` returns one of `{None, CloseSelf, OpenAnotherMenu("name"), ChangeScene("id"), FireVerb("recipe")}`. Stack dispatches accordingly.
- **9 templates — parameter contract (visual design is implementation-iteration):**
  - **Title**: params = `{game_name, subtitle, bg_palette_ref}`. Buttons: "Start" → ChangeScene to project's first scene; "Continue" → OpenAnotherMenu("load_game").
  - **Game-Over**: params = `{header_text, bg_palette_ref}`. Buttons: "Retry" → ChangeScene to last save scene; "Title" → ChangeScene to title scene.
  - **High-Score**: params = `{header_text, bg_palette_ref, scores_key}` (scores_key reads `bb[scores_key]` as `[]ScoreEntry`).
  - **Pause**: KindOverlay. params = `{bg_palette_ref}`. Buttons: "Resume" → CloseSelf; "Save" → OpenAnotherMenu("save_game"); "Title" → ChangeScene to title.
  - **Save-Game**: params = `{bg_palette_ref}`. Lists 3 slots + autosave with metadata (timestamp). Selecting slot → calls `pixelforge_save.Service.SaveToSlot(slot, currentSnapshot)`; on overwrite, confirm.
  - **Load-Game**: params = `{bg_palette_ref}`. Lists 3 slots + autosave. Selecting slot → calls `pixelforge_save.Service.LoadFromSlot(slot)` → blackboard.Restore + change scene.
  - **Inventory**: KindOverlay. params = `{bg_palette_ref, text_color_slot, category_filter}`. Renders `bb["inventory"]` (from U9 + U10) sorted by category then insertion order, with sprite thumbnails (via U9 SpriteThumbnailWidget — although in runtime it's just sprite rendering, not the widget). Selecting → fires item's Effect verb; if count → 0, removes from inventory.
  - **Status**: KindOverlay. params = `{stats_keys[]}` — list of `bb` keys to display (e.g., `["state.hp", "state.lives", "state.gold"]`).
  - **Stage-Select**: params = `{stages[]}` — list of `{name, scene_id, unlock_key}`; renders grid; locked stages dimmed; unlocked → ChangeScene.
- Each template is ~80 LOC + 30 LOC test. Total ~1000 LOC for all 9 + registry + stack.
- **Stack handles pause coordination** when push/pop changes the count of overlay-kind menus.
- D-pad nav: IntentUp/Down/Left/Right; IntentUse confirms; IntentMenu cancels (`Pop()` if at overlay).

**Patterns to follow:** existing `pixelforge_cofont.Print` for text rendering; existing `pixelforge_gui` widget patterns; idea #5's input intent layer; idea #3's palette ref pattern for `bg_palette_ref` params.

**Test scenarios** (selected — per-template tests + stack tests):
- **Registry/Stack:**
  - `TestRegistry_All9TemplatesRegisteredAtInit`: after package init, LookupTemplate returns non-nil for all 9 names.
  - `TestStack_PushOverlayPausesEngine`: stack.Push("inventory"); engine.IsPaused() == true.
  - `TestStack_PopOverlayResumesEngine`: stack.Push("inventory"); stack.Pop(); engine.IsPaused() == false.
  - `TestStack_PushSceneDoesNotPause`: stack.Push("title"); engine.IsPaused() == false.
  - `TestStack_MultipleOverlaysCorrectlyCounted`: Push("inventory"); Push("status"); Pop(); engine still paused (status still open); Pop(); resumed.
  - `TestStack_TopMenuReceivesInput`: Push("title"); Push("inventory"); fire input intent; top (inventory) handler receives it, title does not.
- **Title template:**
  - `TestTitleTemplate_RenderShowsGameName`: render with params={game_name: "Adventure"}; rendered output contains "Adventure" text.
  - `TestTitleTemplate_StartFiresChangeScene`: nav cursor on "Start"; IntentUse; returns ChangeScene to first scene.
  - `TestTitleTemplate_ContinueOpensLoadGame`: cursor on "Continue"; IntentUse; returns OpenAnotherMenu("load_game").
  - Covers AE4.
- **Inventory template:**
  - `TestInventoryTemplate_RendersBlackboardInventoryItems`: bb["inventory"] = [{potion, 2}, {sword, 1}]; render; output contains "Potion x2", "Sword x1".
  - `TestInventoryTemplate_CategoryFilterShowsOnlyMatching`: bb has items in weapon + potion + key categories; params.category_filter="potion"; only potions render.
  - `TestInventoryTemplate_SortedByCategoryThenInsertion`: bb has [{coin, 5}, {potion, 1}, {gold, 3}, {sword, 1}]; sorted = sword (weapon) → potion (potion) → coin/gold (misc, insertion order).
  - `TestInventoryTemplate_SelectingFiresEffectVerb`: cursor on potion; IntentUse; returns FireVerb("restore_health(50)"); count decremented (or removed if 0).
  - `TestInventoryTemplate_IntentMenuClosesSelf`: IntentMenu; returns CloseSelf.
  - Covers AE5.
- **Save-Game template:**
  - `TestSaveGameTemplate_ListsAllSlots`: render with 3 named slots + autosave; output shows 4 entries.
  - `TestSaveGameTemplate_SelectingEmptySlotWritesSnapshot`: select empty slot 2; calls pixelforge_save.Service.SaveToSlot(2); file exists.
  - `TestSaveGameTemplate_OverwriteShowsConfirm`: slot 1 has data; select slot 1; confirm prompt appears; on confirm, overwrites.
- **Load-Game template:**
  - `TestLoadGameTemplate_SelectingSlotLoadsSnapshot`: slot 1 has data; select; blackboard.Restore called with slot's data; ChangeScene to saved scene fired.
  - `TestLoadGameTemplate_EmptySlotsDisabled`: only slot 1 has data; slot 2 and 3 show as "Empty" + non-selectable.

**Verification:** `go test ./pixelforge_menus/...` passes; integration smoke (after U12): create title scene, apply Title template; run; see title screen with Start/Continue.

---

### U8. Studio Menus workspace — menu list + template picker + parameter editor

**Goal:** New dockable Menus workspace. Left panel: list of menus per project. Right panel: template picker (combo of 9 template names) + parameter editor (template-specific fields rendered via reflection or hand-coded per template).

**Requirements:** R13 (designer-tweakable parameters via Menus workspace).

**Dependencies:** U4 (Project.Menus schema), U7 (template registry to enumerate available templates + lookup parameter shape).

**Files:**
- `pixelforge_studio/menus/workspace.go` (NEW)
- `pixelforge_studio/menus/workspace_test.go` (NEW)
- `pixelforge_studio/menus/menu_list.go` (NEW)
- `pixelforge_studio/menus/template_picker.go` (NEW)
- `pixelforge_studio/menus/parameter_editor.go` (NEW)

**Approach:**
- `MenusWorkspace`: standard `Workspace` interface (Name="menus", DisplayName="Menus").
- Layout: left = menu list (sorted alphabetically), right = template picker + parameter editor.
- **Menu list**:
  - Walks `Project.Menus`; each entry shows Name + Template type.
  - "+ New Menu" button → text input for name → creates empty MenuConfig + MarkDirty.
  - Per-entry: Rename / Delete.
- **Template picker**:
  - `imgui.BeginCombo("Template")` with all 9 registered template names (from `pixelforge_menus.Templates()`).
  - On change: calls `pixelforge_menus.LookupTemplate(name).DefaultParams()` and replaces `MenuConfig.Params`; MarkDirty.
- **Parameter editor**:
  - For each param key in `MenuConfig.Params`, render an appropriate input:
    - String → InputText
    - Int → InputInt
    - Palette ref (key ends in "_palette_ref") → custom palette dropdown (use idea #3's sub-palette dropdown widget)
    - Slot ref (key ends in "_slot") → IntSlider
    - List → multi-row editor (e.g., Stage-Select's `stages[]`)
  - On any change: MarkDirty.
- Each template can expose a `ParameterSchema()` method (optional) for cleaner UI; v1 falls back to type-inferred rendering when not provided.

**Patterns to follow:** existing `Workspace` interface; idea #4's plan U6 for left+right panel pattern; idea #3's plan U4 sub-palette dropdown for palette refs.

**Test scenarios:**
- `TestMenusWorkspace_RegisteredWithEditor`: after RegisterWith, `e.GetWorkspace("menus")` non-nil.
- `TestMenuList_NewMenuCreatesEntry`: create menu "title_screen"; `Project.Menus["title_screen"]` exists; MarkDirty.
- `TestMenuList_DeleteRemovesEntry`: delete menu; entry gone; MarkDirty.
- `TestTemplatePicker_ShowsAll9Templates`: combo has 9 entries.
- `TestTemplatePicker_OnChangeReplacesParamsWithDefaults`: change template from Title to Inventory; Params keys change to Inventory's defaults.
- `TestParameterEditor_StringParamRendersTextInput`: param with string value; InputText rendered.
- `TestParameterEditor_PaletteRefRendersSubPaletteDropdown`: param key "bg_palette_ref"; sub-palette dropdown rendered (sources from project's sub-palettes).
- `TestParameterEditor_EditParamUpdatesProject`: edit a param; Project.Menus[name].Params[key] updates; MarkDirty.

**Verification:** `go test ./pixelforge_studio/menus/...` passes; manual: open studio, View → Menus (Ctrl+7); create menu; pick Title template; edit game_name param.

---

### U9. Items workspace + ItemDefinition schema + SpriteThumbnailWidget + inventory blackboard semantics

**Goal:** New dockable Items workspace (table editor). Each row maps to an `ItemDefinition` field. Icon column uses NEW `SpriteThumbnailWidget` that looks up a sprite by name and renders preview. Inventory state on blackboard is `[]InventoryEntry{ItemID, Count}`; verbs (from U10) mutate this list.

**Requirements:** R20 (flat ItemDatabase schema), R21 (Items workspace), R22 (inventory on blackboard).

**Dependencies:** U4 (Project.Items schema).

**Files:**
- `pixelforge_studio/items/workspace.go` (NEW)
- `pixelforge_studio/items/workspace_test.go` (NEW)
- `pixelforge_studio/items/table_editor.go` (NEW)
- `pixelforge_studio/items/table_editor_test.go` (NEW)
- `pixelforge_studio/editor/widgets/sprite_thumbnail.go` (NEW)
- `pixelforge_studio/editor/widgets/sprite_thumbnail_test.go` (NEW)
- `pixelforge_blackboard/inventory.go` (NEW — convention helpers for "inventory" key)
- `pixelforge_blackboard/inventory_test.go` (NEW)

**Approach:**
- `ItemsWorkspace`: standard `Workspace` interface (Name="items", DisplayName="Items").
- **Table editor**:
  - Header row: ID | Name | Icon | Description | Effect | Category.
  - One row per `Project.Items[i]`:
    - ID: `InputText`
    - Name: `InputText`
    - Icon: SpriteThumbnailWidget (renders thumbnail; on click, opens sprite-ref dropdown — uses existing WidgetSpriteRef dispatch).
    - Description: `InputText` (multi-line).
    - Effect: `InputText` (the verb recipe string like `"restore_health(50)"`).
    - Category: combo `{weapon, potion, key, misc}`.
    - Right-most: Delete button.
  - Bottom: "+ New Item" button → appends empty ItemDefinition with auto-generated ID (e.g., "item_1"); MarkDirty.
- **SpriteThumbnailWidget**:
  - Signature: `Render(ctx widgets.Context, spriteName string, size [2]int) (clicked bool)`.
  - Looks up `Project.Sprites[]` by name; finds the SpriteAsset; decodes its PNG bytes (cached); renders as `imgui.ImageWithBgV` at given size.
  - If not found, renders a placeholder gray square with `?`.
  - Returns true if clicked (caller can then open a sprite-ref dropdown).
  - Cache strategy: package-level `map[string]*ebiten.Image` keyed by sprite name; invalidate on project change.
- **Inventory convention helpers** (`pixelforge_blackboard/inventory.go`):
  - `Add(bb, itemID, count)` — reads `bb["inventory"]` as `[]InventoryEntry`; if itemID exists, increment count; else append.
  - `Remove(bb, itemID, count)` — decrements; if count goes to 0, removes entry.
  - `Has(bb, itemID) bool`
  - `Count(bb, itemID) int`
  - These are used by the verb recipes in U10.

**Patterns to follow:** existing Workspace interface; existing `WidgetSpriteRef` dispatch in `inspector.go:183-184` (for the on-click dropdown); existing png-decode patterns from `palette/import_pipeline.go`.

**Test scenarios:**
- `TestItemsWorkspace_RegisteredWithEditor`: after RegisterWith, `e.GetWorkspace("items")` non-nil.
- `TestTableEditor_NewItemAppendsRow`: click +New Item; `Project.Items` grows by 1; new entry has auto-ID; MarkDirty.
- `TestTableEditor_EditFieldUpdatesProject`: change Name field; `Project.Items[i].Name` updates; MarkDirty.
- `TestTableEditor_DeleteRemovesRow`: delete row 1; `Project.Items` has 2 rows (rows 0 and 2 from original 3); MarkDirty.
- `TestTableEditor_CategoryComboLimitedToEnum`: combo shows only weapon/potion/key/misc.
- `TestSpriteThumbnail_RendersExistingSprite`: project has sprite "hero"; SpriteThumbnailWidget for "hero" produces output with non-empty texture.
- `TestSpriteThumbnail_PlaceholderForMissingSprite`: SpriteThumbnailWidget for "nonexistent"; placeholder rendered.
- `TestSpriteThumbnail_CachesAcrossRenders`: render same sprite twice; PNG decoded once (cache hit on second).
- `TestSpriteThumbnail_InvalidatesOnProjectChange`: render sprite "hero"; switch project; render again; PNG re-decoded.
- `TestInventory_AddItemAppendsEntry`: bb["inventory"] empty; Add(bb, "potion", 1); bb["inventory"] = [{potion, 1}].
- `TestInventory_AddExistingItemIncrements`: bb has [{potion, 1}]; Add(bb, "potion", 2); bb has [{potion, 3}].
- `TestInventory_RemoveDecrementsThenDeletes`: bb has [{potion, 2}]; Remove("potion", 1); bb has [{potion, 1}]; Remove("potion", 1); bb has [].
- `TestInventory_HasReturnsBool`: bb has [{potion, 1}]; Has("potion") true, Has("sword") false.
- `TestInventory_CountReturnsCount`: bb has [{potion, 3}]; Count("potion") == 3; Count("sword") == 0.

**Verification:** `go test ./pixelforge_studio/items/... ./pixelforge_blackboard/...` passes; manual: open studio, View → Items (Ctrl+6); add 3 items; assign sprite icons; save.

---

### U10. Verb-recipe additions + Recipe schema extension (amends idea #5 U4)

**Goal:** Register ~12 new verb recipes covering dialogue, menu, inventory, save/load actions. Extend idea #5's `Recipe` struct to support `ConditionKind string` (mutually exclusive with `ActionKind`) so `has_item` can be a condition-verb. Update `LookupRecipe` to return either Action or Condition.

**Requirements:** R25 (verb-recipe additions).

**Dependencies:** idea #5 plan U4 (verb_recipes.go to amend), U1 (pixelforge_save for save_now/load_slot), U5 (pixelforge_dialogue for open_dialogue), U7 (pixelforge_menus for open_menu), U9 (inventory helpers for give/take/has).

**Files:**
- `pixelforge_studio/scripting/catalog/verb_recipes.go` (MODIFY — extend Recipe with ConditionKind; adjust LookupRecipe + Apply)
- `pixelforge_studio/scripting/catalog/builtin_rpg.go` (NEW — register the 12 new recipes)
- `pixelforge_studio/scripting/catalog/builtin_rpg_test.go` (NEW)

**Approach:**
- **Recipe extension** (amends idea #5 U4):
  ```
  type Recipe struct {
      Name             string
      ActionKind       string   // mutually exclusive with ConditionKind
      ConditionKind    string   // NEW; mutually exclusive with ActionKind
      DefaultArgs      map[string]any
      RelevantTriggers []string
  }
  ```
  Validation: exactly one of ActionKind/ConditionKind must be set.
  - `Apply(recipe Recipe, overrides map[string]any) (Effect, Predicate)` — returns one of: Effect (if ActionKind), Predicate (if ConditionKind), with the other nil.
- **New recipes registered:**
  1. `open_dialogue` — Action; ActionKind="open_dialogue"; arg=`dialogue_name`. Builder dispatches to `pixelforge_dialogue.TextBoxRenderer.LoadAndShow(name)`.
  2. `close_dialogue` — Action; ActionKind="close_dialogue". Builder dispatches to `pixelforge_dialogue.TextBoxRenderer.Close()`.
  3. `open_menu` — Action; ActionKind="open_menu"; arg=`menu_name`. Builder dispatches to `pixelforge_menus.Stack.Push(name, params)`.
  4. `close_menu` — Action; ActionKind="close_menu". Builder dispatches to `Stack.Pop()`.
  5. `give_item` — Action; ActionKind="give_item"; arg=`item_id` and optional `count` (default 1). Builder calls `pixelforge_blackboard.Add(bb, item_id, count)`.
  6. `take_item` — Action; ActionKind="take_item"; arg=`item_id` and optional `count` (default 1). Builder calls `pixelforge_blackboard.Remove(bb, item_id, count)`.
  7. `set_item_count` — Action; ActionKind="set_item_count"; arg=`item_id`, `count`. Builder sets the count directly.
  8. `has_item` — **Condition**; ConditionKind="has_item"; arg=`item_id`. Builder returns Predicate that calls `pixelforge_blackboard.Has(bb, item_id)`.
  9. `save_now` — Action; ActionKind="save_now"; optional arg=`slot` (default "autosave"). Builder calls `pixelforge_save.Service.SaveToSlot(slot, currentSnapshot)` OR `Autosave(snapshot)` if slot="autosave".
  10. `load_slot` — Action; ActionKind="load_slot"; arg=`slot`. Builder calls `pixelforge_save.Service.LoadFromSlot(slot)` → restore blackboard → change scene.
  11. `delete_slot` — Action; ActionKind="delete_slot"; arg=`slot`. Builder calls `pixelforge_save.Service.DeleteSlot(slot)`.
  12. Also re-affirm `change_scene` is registered (from idea #5; just confirm).
- Each registered via `init()` in `builtin_rpg.go`.
- Document the dispatch shape — these recipes need access to runtime singletons (TextBoxRenderer, MenuStack, SaveService, Blackboard). The recipe builders receive these via the existing `Context` interface (idea #5's `catalog.Context` exposes blackboard / engine accessors).

**Patterns to follow:** existing `pixelforge_studio/scripting/catalog/builtin_actions.go:11-16` for registration shape; idea #5 plan U4 for `RegisterVerbRecipe` API; idea #5's `catalog.Context` interface for runtime lookup.

**Test scenarios:**
- `TestRecipe_ActionAndConditionMutuallyExclusive`: define Recipe with both ActionKind + ConditionKind; validation fails.
- `TestRecipe_HasItemIsConditionRecipe`: LookupRecipe("has_item") returns recipe with ConditionKind set, ActionKind empty.
- `TestApply_ActionRecipeReturnsEffect`: Apply(give_item recipe, args); returns Effect, Predicate is nil.
- `TestApply_ConditionRecipeReturnsPredicate`: Apply(has_item recipe, args); returns Predicate, Effect is nil.
- `TestGiveItem_AddsToInventory`: bb["inventory"] empty; dispatch give_item("potion"); bb["inventory"] = [{potion, 1}].
- `TestTakeItem_RemovesFromInventory`: bb has [{potion, 1}]; dispatch take_item("potion"); bb["inventory"] = [].
- `TestHasItem_TrueWhenPresent`: bb has [{potion, 1}]; Predicate(has_item("potion")) returns true.
- `TestHasItem_FalseWhenAbsent`: bb empty inventory; Predicate(has_item("sword")) returns false.
- `TestOpenDialogue_LoadsTreeAndShows`: dispatch open_dialogue("test_tree"); TextBoxRenderer.currentTree == project.Dialogues["test_tree"].Tree; renderer is visible.
- `TestOpenMenu_PushesOntoStack`: dispatch open_menu("pause"); Stack.Len() increases by 1; top menu is "pause".
- `TestSaveNow_SavesToAutosaveByDefault`: dispatch save_now; pixelforge_save.Service.Autosave called.
- `TestSaveNowWithSlotArg_SavesToNamedSlot`: dispatch save_now(slot="slot1"); SaveToSlot("slot1") called.
- `TestLoadSlot_RestoresBlackboardAndChangesScene`: dispatch load_slot("slot1"); blackboard.Restore called; ChangeScene dispatched.

**Verification:** `go test ./pixelforge_studio/scripting/catalog/...` passes; integration smoke (after U12): bind `when destroyed: give_item:gold` in editor; defeat enemy; gold appears in inventory.

---

### U11. View menu integration + keymap + workspace registration

**Goal:** Wire the three new workspaces into the studio's menu + keymap system. Add View menu entries for Dialogue (Ctrl+5), Items (Ctrl+6), Menus (Ctrl+7). Update `pixelforge_studio/main.go` to call `RegisterWith(e)` for all three.

**Requirements:** R6 (Dialogue workspace), R21 (Items workspace), R13 (Menus workspace) — all surface via View menu.

**Dependencies:** U6 (Dialogue workspace exists), U8 (Menus workspace exists), U9 (Items workspace exists).

**Files:**
- `pixelforge_studio/main.go` (MODIFY — add `dialogue.RegisterWith(e); items.RegisterWith(e); menus.RegisterWith(e)`)
- `pixelforge_studio/editor/keymap.go` (MODIFY — add Ctrl+5/6/7 bindings)
- `pixelforge_studio/editor/file_menu.go` (MODIFY — add 3 entries to View menu; route in handleShortcuts)

**Approach:**
- `pixelforge_studio/main.go:55-57` extends with three new RegisterWith calls.
- `pixelforge_studio/editor/keymap.go:70-71` extends with:
  - Ctrl+5 → action "workspace.dialogue"
  - Ctrl+6 → action "workspace.items"
  - Ctrl+7 → action "workspace.menus"
- `pixelforge_studio/editor/file_menu.go:190-196` View menu adds:
  - `{Label: "Dialogue", Shortcut: "Ctrl+5", OnSelect: func() { e.SetActiveWorkspaceByName("dialogue") }}`
  - `{Label: "Items", Shortcut: "Ctrl+6", OnSelect: ...}`
  - `{Label: "Menus", Shortcut: "Ctrl+7", OnSelect: ...}`
- `handleShortcuts` (file_menu.go:251-262) gains corresponding dispatch.

**Patterns to follow:** existing entries from file_menu.go:191-196; existing keymap entries from keymap.go:70-71.

**Test scenarios:**
- `TestFileMenu_DialogueEntryPresent`: View menu has Dialogue entry with Ctrl+5 shortcut.
- `TestFileMenu_ItemsEntryPresent`: View menu has Items entry.
- `TestFileMenu_MenusEntryPresent`: View menu has Menus entry.
- `TestFileMenu_DialogueActivatesWorkspace`: click View → Dialogue; e.ActiveWorkspaceName() == "dialogue".
- `TestKeymap_Ctrl5BindingExists`: keymap has Ctrl+5 → workspace.dialogue.
- `TestMain_AllThreeRegistered`: after main initialization; e.GetWorkspace("dialogue"), e.GetWorkspace("items"), e.GetWorkspace("menus") all non-nil.

**Verification:** `go test ./pixelforge_studio/editor/...` passes; manual: launch studio; press Ctrl+5 → Dialogue workspace; Ctrl+6 → Items; Ctrl+7 → Menus.

---

### U12. End-to-end RPG acceptance tests

**Goal:** Integration tests covering AE1-AE8 + F1-F4. Loads fixtures, simulates designer + player actions via public APIs, verifies acceptance examples.

**Requirements:** R1-R26 covered transitively.

**Dependencies:** U1-U11 all merged.

**Files:**
- `pixelforge_studio/integration_test/rpg_e2e_test.go` (NEW)
- `pixelforge_studio/integration_test/fixtures/npc_dialogue_project.pforge` (NEW)
- `pixelforge_studio/integration_test/fixtures/inventory_menu_project.pforge` (NEW)
- `pixelforge_studio/integration_test/fixtures/save_point_project.pforge` (NEW)
- `pixelforge_studio/integration_test/fixtures/title_screen_project.pforge` (NEW)
- `pixelforge_studio/integration_test/fixtures/pre_v1_no_rpg_fields.pforge` (NEW)

**Test scenarios** (one per AE + F):
- `TestE2E_AE1_SavePointTouchOpensSaveMenu`: load save_point fixture; simulate player touching save point; pixelforge_menus.Stack has "save_game" template open; select slot 1; pixelforge_save.Service.SaveToSlot called; file exists at `<UserConfigDir>/pixelforge-games/<title>/slot1.json`.
- `TestE2E_AE2_DialogueExampleWithInterpolationAndConditionalChoices`: load dialogue project; bb.Set("player_name", "Mario"); bb.Set("lives", 3); open dialogue "test"; render shows "Welcome to Mario, hero of the land."; both choices visible. Same with bb.Set("lives", 0); only "Accept quest" visible.
- `TestE2E_AE3_DialogueAdvanceAndChoiceNav`: dialogue open; IntentUse → next line; choice node; IntentDown → highlight=1; IntentUp → highlight=0; IntentUse → follow choice.
- `TestE2E_AE4_TitleScreenLaunchesGame`: load title fixture; sceneGame runs title scene; Title template renders; cursor on "Start"; IntentUse; ChangeScene to "level_1".
- `TestE2E_AE5_GiveItemAndUseInventoryFlow`: load inventory fixture; dispatch give_item("potion"); bb["inventory"] = [{potion, 1}]; open inventory menu; render shows "Potion x1"; select; potion's effect (restore_health) fires; bb["inventory"] = [].
- `TestE2E_AE6_StageDirectionExecutesAndRenders`: dialogue "(walks left 4 tiles)"; on advance, the named entity moves 4 tiles left (verb fires); text renders. Unknown direction → italic rendering only, no verb call.
- `TestE2E_AE7_LegacyProjectLoadsWithEmptyRPGFields`: load pre_v1 fixture; no Dialogues/Menus/Items/SaveConfig; project loads; SaveConfig default applied; no errors.
- `TestE2E_AE8_SaveLoadDeterministicallyRestores`: bb.Set("health", 5); bb.Set("lives", 2); scene "dungeon_3" with 3 enemies; SaveToSlot("slot1"); blackboard.Set("health", 100); change scene; LoadFromSlot("slot1"); blackboard.Get("health") == 5; current scene == "dungeon_3"; enemy count == 3.
- `TestE2E_F1_NPCInteractionFullFlow`: place NPC entity with `when interacted: open_dialogue:old_man_hint`; simulate interaction; dialogue opens; player advances through; dialogue closes; engine resumes.
- `TestE2E_F2_InventoryMenuFullFlow`: defeat enemy with `when destroyed: give_item:gold`; pickup with `when touched: give_item:potion`; press input/menu; pause inventory opens; gold + potion visible; navigate; use potion; effect fires.
- `TestE2E_F3_SaveAndResumeAcrossSessions`: save game; close editor; reopen editor; load same project; LoadFromSlot("slot1") works; restored state matches.
- `TestE2E_F4_TitleAndGameOverFlow`: title scene with Title template; "Start" → "level_1"; Player damaged with `if lives==0: change_scene:game_over`; game_over scene shows Game-Over template.
- `TestE2E_PauseOverlayActuallyPausesEngine`: scene with moving enemy; press input/menu; pause overlay opens; enemy stops; press input/menu; pause closes; enemy resumes.
- `TestE2E_AutosaveThrottle`: dispatch save_now (autosave) 5 times in 10 seconds; only 1 file write occurs (after first; rest throttled).
- `TestE2E_HasItemConditionGatesRule`: rule fires `if has_item:key: open_door`; bb has no key; rule doesn't fire. Add key via give_item; rule fires next tick.

**Verification:** `go test ./pixelforge_studio/integration_test/...` passes; all 8 AEs green; all 4 flows green.

---

## Scope Boundaries

### Deferred to Follow-Up Work

- **Turn-based combat system** (FF battles, Pokémon, Punch-Out timing) — out per origin.
- **Localization / multi-language dialogue** — out per origin; single-language only.
- **Animated cutscenes** beyond basic stage directions — out per origin.
- **Voice acting** (audio per dialogue line) — out per origin.
- **Dialogue node-graph visualizer** — out; script is source of truth.
- **Designer-authored menu templates** — out; 9 fixed in v1.
- **Per-template visual customization** beyond bg palette + text color + header text + template-specific params — out; complex layouts via event-sheet escape hatch.
- **Equippable items / equipment slots** — out per origin.
- **Item crafting / recipes / shops** — out per origin.
- **Save file migration** — v1 ships SchemaVersion=1; future versions add migration steps.
- **Cloud save / sync** — out per origin.
- **Save thumbnails** — out per origin; slot # + timestamp only.
- **Dialogue translation export** — deferred with localization.
- **Per-actor dialogue portraits** (Earthbound-style) — out per origin.
- **Streamed / scrolling long text** — out per origin; click-to-advance per-line only.
- **Choice systems beyond labels** — out per origin; Twine-only.
- **Inventory tabs / categories beyond single filter** — out per origin.
- **Party management** — out per origin; single Player entity.
- **Drop-in / drop-out save** — out per origin; save points only.
- **Item stacking limits / weight** — out per origin.
- **Stage directions beyond 5 listed** (`flash_screen`, `play_sound`) — extend in v2.
- **Typewriter text animation** in TextBoxRenderer — v1 shows full line at once.
- **Mid-dialogue music change** — runtime audio stays on whatever's playing.
- **Persistent inventory icons cache invalidation strategy** for very large projects — v1 caches per-process; if memory becomes a concern, add LRU.

### Outside this product's identity

- AI-generated dialogue or item descriptions.
- Multi-player save (shared file or sync).
- Browser-based or mobile workspace.
- Designer programmatically authoring custom templates via code (templates are code-shipped).

---

## Key Technical Decisions

- **Zero external dependencies.** Six candidates evaluated; all rejected via leverage doctrine. Total custom ~1200 LOC across all units.
- **Save format is JSON with SchemaVersion, additive omitempty, sanitize-on-load** — **overrides brainstorm's "forward-incompatible v1" stance**. Cost: 1 struct field + default backfill. Benefit: established discipline carries forward; v2 ships without invalidating community saves.
- **Dialogue + menu renderers live in engine packages** (`pixelforge_dialogue`, `pixelforge_menus`), not in studio packages. They paint into `*ebiten.Image` using `pixelforge_cofont` + `pixelforge_gui`. Shipped runtime gets them by importing. Editor preview shows them by construction since `sceneGame.Draw` calls the same code.
- **Three new engine packages, three new studio workspaces.** Engine: `pixelforge_dialogue`, `pixelforge_menus`, `pixelforge_save`. Studio: `pixelforge_studio/dialogue`, `pixelforge_studio/menus`, `pixelforge_studio/items`. Pattern matches existing structure (`palette/` + engine helpers; `capture/` + engine helpers).
- **Save unit = blackboard snapshot + current scene + active-scene entity state.** Other scenes restore from project defaults on load. Trade-off acknowledged: enemies in other scenes "reset" on save/load if player travels back to them. v2 could store per-scene state if demand exists.
- **Pause primitive blocks `EventUpdate`, keeps `EventLateDraw` + input.** Greenfield; mirrors `Engine.Stop()` debugger gate. Documented invariant.
- **Recipe schema extended with `ConditionKind`** (mutually exclusive with `ActionKind`). Amends idea #5's U4. Needed because `has_item` is a condition.
- **Save path strategy**: `os.UserConfigDir() / "pixelforge-games" / sanitize(title) / slot{n}.json`. WASM: `localStorage["pixelforge-games:<title>:slot{n}"]`. Sanitize: lowercase + underscores + strip non-`[a-z0-9-_]`.
- **Autosave throttle = 1 per 30 seconds globally.** Configurable in `SaveConfig.AutosaveThrottleSeconds` (default 30; planner-set, not designer-set in v1).
- **9 templates as parameter contracts in v1**; visual designs are implementation-iteration. The plan specifies the parameter schema per template; the actual rendering (font sizes, layouts, button positions) is an implementer's creative call.
- **Inventory rendering order: by category (weapon → potion → key → misc) then by insertion order.** Deterministic; designer doesn't configure.
- **5 stage directions in v1**: `walk_left N`, `walk_right N`, `walk_up N`, `walk_down N`, `pause N`. `flash_screen` and `play_sound` deferred.
- **Hand-rolled recursive-descent dialogue parser** — no parser-gen dependency. ~200 LOC; trade-off: less general than a real parser-gen, but the grammar is restricted enough that it's net-cleaner.
- **DialogueScript.Source is source of truth**; Tree is regenerated at load. Designer git-diffs Source; Tree is computed.
- **Menu stack and pause coordination**: overlay templates auto-Pause on push, auto-Resume on pop. Scene templates don't touch pause.
- **`SpriteThumbnailWidget` is new** for Inventory icon rendering. Cached per-process; invalidated on project change.
- **Verb recipes dispatch from runtime singletons** (TextBoxRenderer, MenuStack, SaveService, Blackboard). Builders pull them from `catalog.Context`. Idea #5's Context interface needs extending OR a parallel runtime-singleton accessor.
- **WASM saves via `js.Global().Get("localStorage")`** with size limits (~5MB / per-origin). Backend interface lets the runtime swap based on build tag.
- **Click-to-advance dialogue uses `IntentUse`** (the same A button intent that confirms menu choices). Reuses idea #5's input intent layer.
- **All template params editable via reflection-style param-editor** in Menus workspace. Specific widgets per param-key suffix (`_palette_ref` → palette dropdown; `_slot` → slot picker; etc.).

---

## Dependencies / Assumptions

- **Strict dependencies on idea #5's plan**: U1 (blackboard package), U2 (Archetype + Entity schema), U3 (input intents), U4 (verb_recipes.go). All four MUST land before this plan's units that depend on them.
- **Strict dependency on idea #5's plan U4 amendment**: this plan extends Recipe with ConditionKind. Implementer of idea #5's U4 should be aware; coordinate at execution time.
- **Strict dependency on idea #2's plan U1** (TileAtlas if menus use NESPaletteBlock — but they don't; menus use `bg_palette_ref` directly).
- **Strict dependency on idea #3's plan U1** (sub-palette schema) for template `bg_palette_ref` params.
- **Soft dependency on idea #4's plan U2** (Allocator) — if templates trigger audio (e.g., menu navigation sounds), they need the allocator to pick channels.
- **Strict dependency on idea #7's plan** for the shipped-binary save loop. v1 of this plan tests save IO in the editor preview; the cross-session "classmate saves and resumes" demo depends on idea #7.
- **Existing `pixelforge_cofont`** for text rendering. Confirmed shipped in engine.
- **Existing `pixelforge_gui`** widget primitives for rect helpers, buttons. Confirmed shipped.
- **Existing `pixelforge_event` registry** for target-publish patterns.
- **`docs/solutions/`** anchors: `always-on-game-embedding.md` (one render path), `canvas-vs-native-chrome-split.md` (engine vs studio widgets), `scripting-runtime-design.md` (catalog register seam), `editor-pforge-schema-shape.md` (additive + sanitize), `dirty-state-ux.md` (MarkDirty), `focus-manager-design.md` (modal stack).
- **Go's `os.UserConfigDir()`** is the canonical cross-platform user-config-dir helper. Already used at `settings.go:117`.
- **WASM save backend** ships via `//go:build js` build tag; assumes Ebitengine's WASM target works (it does per existing engine).
- **Engine pause primitive doesn't conflict with existing debugger pause** (scripting/debugger.go); they're orthogonal — game pause from overlays vs. rule-debugger pause from breakpoints.

---

## Risk Analysis & Mitigation

| Risk | Severity | Mitigation |
|---|---|---|
| Dialogue parser grammar surprises break designer scripts | Medium | Parser tests cover all grammar constructs explicitly. Round-trip test guarantees no data loss. Error messages name line numbers. v2 can extend with EBNF parser-gen if hand-rolled becomes brittle. |
| Save unit too small (other-scene state lost) confuses designers | Medium | Documented in scope boundaries: "other scenes restore from project defaults." If demand surfaces, v2 adds per-scene state on disk. |
| Save unit too large (active scene has 100s of entities) causes large save files | Low | Active scene entity count is bounded by what fits on screen + nearby. For 99% of games, < 50 entities. Save file ~few KB. JSON is verbose but human-readable; gzip if needed in v2. |
| WASM save via localStorage has 5MB limit, exceeded by large blackboards | Medium | Document the limit. v2 could use IndexedDB. Quota-exceeded error surfaces in UI as toast. |
| Pause primitive subtly differs in studio preview vs shipped runtime | Medium | Same code path (`sceneGame.Update` calls `engine.Update` which checks `engine.IsPaused()`); structural guarantee. Integration test covers both paths. |
| Engine.Pause() called twice in a row without matching Resume() leaves engine permanently paused | Low | Idempotent; tracks pause count internally; Resume only resumes when count hits 0. Tests cover this. |
| Menu stack overlay-vs-scene confusion | Low | TemplateKind constant on each template; stack reads it; pause coordination is deterministic. Tests cover both kinds. |
| Stage-direction parser dispatches verb that mutates wrong entity | Medium | Stage direction context includes the dialogue's "current speaker entity" (looked up by name in Project.Entities); if unknown speaker, falls back to flavor-only. Tests cover the entity-resolution path. |
| Inventory rendering caches sprite thumbnails infinitely | Low | Cache invalidates on project change; for v1 acceptable. If memory becomes an issue, add LRU eviction. |
| Recipe schema extension breaks idea #5 plan's tests if not coordinated | Medium | This plan's U10 amends `verb_recipes.go` from idea #5. Execution order matters: idea #5's U4 lands first; this plan's U10 amends; tests update for both. |
| Save file written to wrong directory on macOS sandboxed environment | Medium | `os.UserConfigDir()` returns app-sandbox-friendly path. Manual smoke test on macOS confirms (per planning verification). |
| Dialogue text overflows the text box for long lines | Low | TextBoxRenderer word-wraps via `pixelforge_cofont.Print`'s line-break behavior (or measure-then-wrap manually). Wrap point: text box width. |
| Twine condition grammar (`if state.x > 0`) doesn't parse properly | Medium | Restricted to `state.<key> <op> <literal>` with explicit op list. Tests cover all 6 ops + bool/int/string literals. Anything more complex (logical AND/OR) deferred to v2. |
| `imgui.InputTextMultiline` buffer overflow on very long scripts | Low | Buffer grows as needed via callback; 16 KB initial; can extend. cimgui-go pattern is well-known. |

---

## System-Wide Impact

**New packages introduced:**
- Engine: `pixelforge_dialogue`, `pixelforge_menus`, `pixelforge_save` — three new engine packages, all shipped in runtime binary.
- Studio: `pixelforge_studio/dialogue`, `pixelforge_studio/menus`, `pixelforge_studio/items` — three new studio workspaces.

**Modified packages:**
- `pixelforge_blackboard` (idea #5) — adds Snapshot/Restore + inventory helpers.
- `pixelforge_loop` — adds Pause/Resume on Engine.
- `pixelforge_project` — adds 4 schema fields + new types.
- `pixelforge_studio/scripting/catalog` — extends Recipe; registers ~12 new recipes.
- `pixelforge_studio/editor` — adds View menu entries, keymap bindings, sprite-thumbnail widget.
- `pixelforge_studio/main` — adds 3 RegisterWith calls.

**Affected workflows:**
- **Designer authoring** — primary target. New workflows: open Dialogue workspace → write screenplay; open Items workspace → define items; open Menus workspace → apply templates. Plus: bind RPG verbs from idea #5's verb sheet on entities.
- **Engine runtime** — adds dialogue + menu rendering inside the game loop; adds pause primitive. No regression on existing engine behavior (additive only).
- **Shipped runtime** — Capsule (idea #7) imports the new engine packages automatically since they're additive to `pixelforge_project`. Save IO writes to user-config-dir at runtime.
- **Build pipeline (idea #7)** — must vendor the new engine packages. Idea #7's plan handles this.

**Documentation impact:**
- Post-v1, capture as `docs/solutions/` entries:
  1. Canvas-resident overlay pattern (text box + menus painting into ebiten.Image; ships with runtime).
  2. Scene-pause definition (what pauses vs what continues).
  3. Save file format + schema-version forward-compat discipline.
  4. Verb-recipe condition vs action mutual exclusivity.
  5. Engine package vs studio workspace separation for content-author features.

**Operational / rollout:**
- Standard release. Coupled with ideas #1, #2, #3, #4, #7 in the same milestone — RPG-class is the largest single addition.
- Pre-v1 projects load with empty RPG fields per AE7.
- Save files appear in user-config dir on first save; designers can inspect/edit (JSON).
- Three new workspaces appear immediately in View menu.

---

## Notes for Implementer

**Coordination with other plans:**
1. **Idea #5 must land first** — U1 (Blackboard), U2 (Archetype/Entity schema), U3 (input intents), U4 (verb_recipes). This plan amends U1 (Snapshot/Restore) and U4 (Recipe.ConditionKind). Coordinate.
2. **Idea #3 should land first** for `bg_palette_ref` template params to work fully (otherwise the sub-palette dropdown won't have options). If idea #3 slips, template params can be plain integer slot indices as fallback.
3. **Idea #7 is required for shipped-binary save loop verification.** v1 of this plan's E2E tests (U12) cover save IO in the editor preview; cross-session "classmate saves and resumes" requires idea #7.
4. **Execution order suggestion:**
   - Phase A (parallel): U1 (pixelforge_save), U2 (Blackboard Snapshot), U3 (engine pause), U4 (project schema additions).
   - Phase B (parallel): U5 (pixelforge_dialogue parser+renderer), U7 (pixelforge_menus templates+stack), U9 (Items workspace + inventory helpers).
   - Phase C: U10 (verb recipes — needs U5, U7, U9), U6 (dialogue workspace — needs U5), U8 (menus workspace — needs U7).
   - Phase D: U11 (view menu wiring — needs U6/U8/U9), U12 (E2E tests — needs all).

**Critical handoff points:**
- The dialogue + menu renderers MUST live in engine packages (`pixelforge_dialogue`, `pixelforge_menus`), not studio. If implementer instinct puts them in `pixelforge_studio/...`, the shipped runtime won't have them.
- The pause primitive's "what pauses" contract is critical. Document the invariant in `pixelforge_loop` docs.
- The save format's SchemaVersion field is the forward-compat hook. Don't ship v1 without it.
- Stage-direction parsing falls back to flavor-only for unknown directions. This is the contract; don't error.
- Inventory ordering is deterministic by design. Designers don't configure it in v1.

**Sound-design / asset coordination:**
- Each menu template renders text + icons. v1 templates can use placeholder layouts; visual polish (color, spacing) is implementation-iteration.
- Inventory item icons come from existing sprite imports (idea #3's import pipeline). No new sprite-curation work needed.
