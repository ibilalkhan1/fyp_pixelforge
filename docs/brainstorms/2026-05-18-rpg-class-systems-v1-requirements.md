---
date: 2026-05-18
topic: rpg-class-systems-v1
origin: docs/ideation/2026-05-18-pixelforge-nes-class-no-code-ideation.md (idea #6)
depends_on: docs/brainstorms/2026-05-18-entity-verb-sheet-v1-requirements.md (idea #5 blackboard + verb catalog)
---

# RPG-Class Systems — Save UI + Dialogue + Menus + Inventory — v1

## Summary

v1 ships three coupled RPG-class systems on top of idea #5's blackboard: a save UI (3 named slots + 1 autosave, blackboard snapshot is the serialization unit), a dialogue system (screenplay-style text editor + branching choices + variable interpolation + runtime text-box renderer), and a menu system (~9 NES-canonical templates plus a flat `ItemDatabase`). Designer authors content in dedicated RPG-systems workspaces; idea #5's verbs trigger them at runtime via new recipes (`open_dialogue`, `open_menu`, `give_item`, `save_now`). No combat engine, no localization, no animated cutscenes.

---

## Problem Frame

Pixelforge has the foundations for arcade games — world authoring (idea #1), tilemap painting (#2), visual identity (#3), audio (#4), and entity logic via the verb-sheet (#5). Half the NES reference set is unreachable with those alone:

- **Final Fantasy** is 50% menu navigation + 30% dialogue + 20% combat. Without dialogue trees, item databases, menu rendering, and persistent save state, FF-class authoring isn't possible.
- **Zelda** has dialogue (NPCs, hint text), inventory (sword, bombs, keys), save slots, and pause/inventory menus.
- **Metroid** has a save-point mechanism (file-based in later iterations) and an items-as-flags inventory.
- **Megaman** has password saves, stage-select menus, and weapon-acquisition inventory.
- **Punch-Out** has roster menus and password progression.
- Even non-RPG games (**Tetris** high-score tables, **Excitebike** track-select) need menu rendering.

Idea #5's blackboard already gives Pixelforge the variable-store half of RPG state (R10–R13). What's missing is the *content* layer: dialogue trees authors compose in text, menu templates designers apply to scenes, item databases authors fill in, and the save UI that lets players persist across sessions. Those four split cleanly into three sub-deliverables (save UI + dialogue + menus/inventory) because variables/switches collapses into the existing blackboard.

This brainstorm ships all three as one coupled release. The bet: a designer in your community can author a Zelda-class game (NPCs talking, inventory items, save slots) or a Final Fantasy-class town (dialogue + party menu + item database + save) in their first project, without writing a script outside the studio's text editor.

---

## Actors

- **A1. Designer.** Authors dialogue trees in a script editor; applies menu templates to scenes or verbs; fills the item database; binds RPG verbs from idea #5's verb-sheet on entities. Doesn't write a save-file format, doesn't render menus by hand, doesn't compose dialogue grammar.
- **A2. Power-user designer.** Edits menu template parameters in advanced mode; writes complex dialogue trees with conditional branches; defines custom items with custom verb effects.
- **A3. Pixelforge Studio.** Hosts three new dockable workspaces (Dialogue, Items, Menus); compiles dialogue scripts into runtime trees at load; renders menu templates over the active scene; serializes blackboard + scene state for saves.
- **A4. The shipped game.** Loads dialogue trees, renders text boxes, opens menu overlays, mutates inventory in the blackboard, reads/writes save files in the user-config directory. Runs entirely off the `.pforge` project file + saved blackboard snapshots.
- **A5. End-player (of a shipped game).** Sees dialogue, navigates menus with D-pad + A/B, picks save slots, plays across sessions.

---

## Key Flows

- **F1. Designer authors a Zelda-style NPC with dialogue**
  - **Trigger:** Designer drops an NPC sprite into a scene and wants it to talk to the player
  - **Actors:** A1, A3, A4, A5
  - **Steps:** (1) Designer selects the NPC, sees archetype = `NPC` (from idea #5); (2) NPC inspector shows `when interacted` slot empty; (3) designer opens the Dialogue workspace (new dock), creates a tree named `old_man_hint`, writes screenplay-style text in the editor; (4) returns to NPC inspector, picks `open_dialogue:old_man_hint` from the verb dropdown; (5) saves and plays; (6) player walks up to the NPC and presses A; dialogue box appears at the bottom of the screen
  - **Outcome:** NPC speaks the authored dialogue when the player interacts; advance-on-click works; tree branches respect blackboard flags.
  - **Covered by:** R6, R7, R8, R9, R10, R23

- **F2. Designer adds a Final Fantasy-style inventory menu**
  - **Trigger:** Designer wants a pause menu showing the player's inventory items
  - **Actors:** A1, A3, A4
  - **Steps:** (1) Designer opens the Items workspace, defines 5 items: potion, sword, key, scroll, gold (each with sprite icon + name + effect); (2) opens the Menus workspace, creates a menu named `pause_inventory`, picks the "Inventory" template, customizes background palette + text color; (3) opens the Player entity inspector, picks `open_menu:pause_inventory` on `when input/menu`; (4) authors enemy-drop verbs (`when destroyed: give_item:gold`) and pickup verbs (`when touched: give_item:potion`); (5) plays and presses Menu mid-game
  - **Outcome:** A pause overlay appears with the inventory template showing the player's current items + counts; D-pad navigates; A uses, B closes.
  - **Covered by:** R11, R12, R13, R14, R20, R21, R22

- **F3. Player saves and resumes a game**
  - **Trigger:** Player wants to save mid-progress and resume later
  - **Actors:** A5, A4
  - **Steps:** (1) Designer has placed a "Save Point" Pickup with `when touched: save_now`; (2) player walks onto the save point; (3) save UI opens showing 3 slots + autosave; player picks slot 1; (4) blackboard snapshot + current scene + entity state writes to user-config dir; (5) player closes the game; (6) player relaunches the game; (7) title screen shows "Continue" / "New Game"; (8) "Continue" opens Load template, player picks slot 1; (9) game restores blackboard + scene + entities
  - **Outcome:** Player's mid-game state survives across sessions. No data loss for at least the saved slots.
  - **Covered by:** R1, R2, R3, R4, R5, R26

- **F4. Designer creates a title screen + game-over screen**
  - **Trigger:** Designer wants a proper title screen with "Start" / "Continue" and a game-over screen with "Retry" / "Title"
  - **Actors:** A1, A3, A4
  - **Steps:** (1) Designer creates a new scene named `title`; (2) opens Menus workspace, creates menu `title_screen` with the "Title" template, sets game title, picks bg palette; (3) the title scene's `when scene starts` verb fires `open_menu:title_screen`; (4) the menu template's "Start" button is pre-wired to `change_scene:level_1`; "Continue" is pre-wired to `open_menu:load_game`; (5) similarly authors `game_over` scene with the "Game-Over" template; (6) any Player entity on `when damaged: if state.lives == 0 → change_scene:game_over`
  - **Outcome:** Game has a proper title flow + game-over flow; designer never hand-laid-out a menu, only picked templates and tweaked palette.
  - **Covered by:** R15, R16, R17, R18, R19, R24

---

## Requirements

**Save UI**

- R1. The save system supports exactly **3 named save slots + 1 autosave slot** in v1. Slot count is fixed; designers cannot configure it per project.
- R2. Save files live in the **user-config directory** (cross-platform: `%APPDATA%` on Windows, `~/Library/Application Support/` on macOS, `~/.local/share/` on Linux) under a per-game subdirectory derived from the game's title. The shipped binary writes there; no special permissions, no user-facing path management.
- R3. A **save unit** = blackboard snapshot + current scene ID + player position + entity state for the current scene (entities spawned/destroyed in the active scene; other scenes restore from project defaults on load).
- R4. **Save UI is rendered via menu templates** (Save-Game template + Load-Game template — see R15-R19). Designer applies these templates to dedicated save/load scenes OR triggers them as overlays via `open_menu:save_game` / `open_menu:load_game` verbs.
- R5. **Autosave fires automatically on scene change** + when designer explicitly invokes `save_now` with the "autosave" target. The autosave slot is overwritten silently each time.
- R26. Loading a saved game **restores the blackboard + scene + entity state to exactly what was saved**. Determinism is required: a saved-then-loaded game plays identically to one that wasn't saved (no entity drift, no random reseeding).

**Dialogue**

- R6. Dialogue trees are authored in a **dedicated Dialogue workspace** (new dockable workspace in the studio). One workspace hosts many trees per project; each tree has a name (e.g., `goblin_intro`, `old_man_hint`) referenced by `open_dialogue:<name>` verbs.
- R7. Dialogue is written in a **screenplay-style text editor**: `MARIO: (walks left 4 tiles) Where is the princess?\nBOWSER: (laughs) Try the next castle.` Round-trips losslessly through a parser; the runtime tree is derived from the script at load time.
- R8. **Branching uses Twine-style labels and choices**. Text blocks declare `:: label_name` headers; choices use `[[Choice text -> label_name]]`. Conditional jumps use `[[Choice text -> label_name | if state.flag_x]]` (condition references the blackboard from idea #5).
- R9. **Variable interpolation**: text embeds `{state.key_name}` references that resolve at display time against the blackboard (e.g., `BOWSER: Welcome, {state.player_name}.`).
- R10. **Runtime text-box renderer**: a fixed-height text box at the bottom of the screen, NES-styled (palette-aware via idea #3), one line at a time with **click/A-button to advance**. Choices show as a numbered vertical list above the text box; D-pad navigates, A confirms.
- R23. **Stage directions in parentheses** (e.g., `(walks left 4 tiles)`) parse to engine actions if they match known verb recipes from idea #5 (`move_with_intent` etc.); otherwise they display as italic flavor text only. v1 ships parsing for ~5 common directions (walk_direction_N, pause, flash_screen, play_sound).

**Menus + Inventory**

- R11. The **menu system ships ~9 NES-canonical templates** in v1: Title, Game-Over, High-Score, Pause, Save-Game, Load-Game, Inventory, Status (HP/MP/level/stats), Stage-Select. Concrete template visual designs are a planning detail; the requirement is that nine exist and cover the named purposes.
- R12. Templates are applied **per-scene OR per-verb**:
  - Scene-based: Title / Game-Over / High-Score / Save-Game / Load-Game / Stage-Select templates apply to a dedicated scene. Designer creates a scene and picks the template name; the scene renders that template instead of a tilemap.
  - Verb-based (overlays): Pause / Inventory / Status templates open as **overlays** on top of the current scene, invoked by verbs (`open_menu:pause`, `open_menu:inventory`). The scene underneath pauses while the overlay is active.
- R13. Each template has **designer-tweakable parameters**: background palette ref, text color slot, header text string, optional template-specific fields (e.g., Title template has "game name" + "subtitle"; Inventory template has "item category filter"). Edited via the Menus workspace.
- R14. **Menu navigation**: arrow keys / D-pad to move selection; A button (`input/use`) to confirm; B button (`input/menu`) to cancel / close overlay. Bindings ride idea #5's input intent layer; templates ship with these bindings pre-wired.
- R20. **Item Database is flat**: each item has `id` (string), `name` (display string), `icon` (sprite ref), `description` (text), `effect` (verb recipe — e.g., `restore_health(50)`), and optional `category` (`"weapon" | "potion" | "key" | "misc"`).
- R21. **Items are authored in a dedicated Items workspace** (new dockable workspace). A table view with one row per item; each row's columns map to the schema fields.
- R22. **Inventory state lives in the blackboard** at the reserved `inventory` key as a flat list of `{item_id, count}` records. Verbs (`give_item:<id>`, `take_item:<id>`, `has_item:<id>`) mutate or query this list. The Inventory template renders directly from this list.

**Schema additions (all additive `omitempty`)**

- R24. New schema fields on `Project`: `Dialogues` (map of dialogue name → parsed tree representation), `Menus` (map of menu name → template id + parameters), `Items` (flat list of `ItemDefinition` records), `SaveConfig` (slot count fixed at 3+1; autosave trigger preferences). Old projects load cleanly with empty maps and default save config.

**Verb-recipe additions (extend idea #5's catalog)**

- R25. v1 adds these verb recipes to idea #5's curated catalog:
  - **Dialogue:** `open_dialogue:<name>`, `close_dialogue`
  - **Menu:** `open_menu:<name>`, `close_menu`, `change_scene:<name>` (already in #5; reused for menu navigation)
  - **Inventory:** `give_item:<id>`, `take_item:<id>`, `has_item:<id>` (condition-verb), `set_item_count:<id>:<n>`
  - **Save/load:** `save_now` (writes to current slot or autosave), `load_slot:<n>`, `delete_slot:<n>`
  - **State helpers (already in #5):** `set_flag`, `check_flag`, `give_points`, `lose_life`, `restore_health`

---

## Acceptance Examples

- **AE1. Covers R1, R2, R3, R4, R5.** Given the designer has placed a save point Pickup with `when touched: save_now`, when the player walks onto it, the Save-Game menu overlay opens listing 3 named slots + 1 autosave slot. Picking slot 1 writes the blackboard snapshot + current scene + entity state to `~/.local/share/<game-title>/slot1.save` (Linux) / corresponding paths on macOS/Windows. Restarting the game and opening the Load-Game menu shows slot 1 with the saved game's timestamp.
- **AE2. Covers R7, R8, R9.** Given a dialogue tree authored as:
  ```
  KING: (bows) Welcome to {state.player_name}, hero of the land.
  [[Accept quest -> accept]]
  [[Decline -> decline | if state.lives > 0]]
  
  :: accept
  KING: Excellent. Take this sword.
  
  :: decline
  KING: As you wish.
  ```
  when the player triggers the dialogue at runtime with `state.player_name = "Mario"` and `state.lives = 3`, the text box reads "Welcome to Mario, hero of the land." then shows both choices. Picking "Decline" jumps to the `:: decline` label. With `state.lives = 0`, only "Accept quest" appears.
- **AE3. Covers R10, R14.** Given an open dialogue, when the player presses A button, the next line advances. When choices appear, D-pad changes the highlighted option; A confirms the highlighted choice; B does nothing (B doesn't cancel dialogue — `close_dialogue` is verb-triggered only).
- **AE4. Covers R11, R12.** Given the designer creates a scene called `title` and assigns the "Title" template, when the game launches, the title scene renders with the template (game name + subtitle + "Start" / "Continue" buttons). Picking "Start" fires `change_scene:level_1`; picking "Continue" fires `open_menu:load_game` (overlay).
- **AE5. Covers R20, R21, R22.** Given the designer defines item `potion` with effect `restore_health(50)` in the Items workspace, when a level entity has `when destroyed: give_item:potion`, defeating that entity adds one potion to the player's inventory (blackboard `inventory` key gains `{item_id: "potion", count: 1}`). Opening the Inventory menu shows "Potion x1"; selecting it fires `restore_health(50)` and decrements count to 0 (removing the row).
- **AE6. Covers R23.** Given a dialogue line `MARIO: (walks left 4 tiles) Where is the princess?` and the v1 stage-direction parser knowing `walk_direction_N`, when the line renders at runtime, the Mario entity walks 4 tiles left (via the verb recipe) and the dialogue text appears in the box. An unknown direction like `(suddenly remembers)` displays as italic flavor text under the speaker name and triggers no engine action.
- **AE7. Covers R24.** Given a `.pforge` file saved before v1 (no `Dialogues` / `Menus` / `Items` / `SaveConfig` fields), when the file loads in v1, all four fields default to empty maps + default save config. No data loss; pre-v1 game plays identically.
- **AE8. Covers R26.** Given a player saves at scene `dungeon_3` with `state.health = 5`, `state.lives = 2`, and 3 enemies remaining in the scene, when the player loads that save, the game restores to `dungeon_3` with `state.health = 5`, `state.lives = 2`, and the same 3 enemies at their saved positions. No additional enemies spawn from "scene start" triggers.

---

## Success Criteria

- **Designer outcome (RPG):** A first-time designer with no game-editor background authors a Zelda-style NPC interaction in under 10 minutes: opens the Dialogue workspace, writes a 5-line script with one choice + one conditional branch, returns to the NPC inspector, picks `open_dialogue` on `when interacted`, plays, sees the dialogue work. Never edits JSON, never touches a node-graph editor.
- **Designer outcome (Inventory):** A designer authors a working Final Fantasy-style pause-menu inventory in under 15 minutes: opens the Items workspace, defines 5 items in the table, opens the Menus workspace, picks the Inventory template, opens the Player inspector, binds `open_menu:pause_inventory` to `when input/menu`, plays — menu opens, items appear with icons, A uses them, B closes.
- **Designer outcome (Save):** A designer adds save points to their game in under 5 minutes: drops a "Save Point" Pickup template (shipped with the v1 archetype defaults), wires `when touched: save_now`. Their classmates can save and resume across sessions, including reloading after a crash.
- **Player outcome:** A classmate playing the shipped game saves their progress, quits, reopens days later, picks "Continue" → "Slot 1", and resumes exactly where they left off. Lost saves are impossible barring filesystem failure.
- **Ship-loop outcome:** A game authored with RPG-class features ships as a self-contained binary (per idea #7) — save files write to the user-config dir at runtime; the shipped game never depends on the studio.
- **Downstream handoff outcome:** Planning consumes this doc and does not need to invent dialogue grammar, menu template list, item schema, save format, or workspace shape. Only implementation specifics (exact parser tokenizer, exact ImGui workspace layouts, exact save-file binary format) are open.

---

## Scope Boundaries

- **Turn-based combat system** (FF battles, Pokémon, Punch-Out timing). v1 ships menus/inventory but no combat engine. Designers wanting combat compose it from verbs.
- **Localization / multi-language dialogue.** Single-language only in v1.
- **Animated cutscenes** beyond basic stage directions. Complex sprite choreography during dialogue deferred.
- **Voice acting** (audio per dialogue line). Out.
- **Dialogue node-graph visualizer.** Script is the source; no graph view in v1.
- **Designer-authored menu templates.** v1 ships 9 templates; designers cannot create new ones through the studio. Custom layouts require code (template definition + renderer).
- **Per-template visual customization** beyond background palette + text color + header text + template-specific param fields. Unique menu layouts use the event-sheet escape hatch + custom Scene + verbs.
- **Item stacking limits or weight systems** (D&D encumbrance). Items have a `count` field; no per-item or per-character cap.
- **Equippable items / equipment slots.** All items are "use-once" or "passive" in v1. Sword-equipped-in-slot-1 model deferred.
- **Item crafting / recipes / shops.** Items are given/taken via verbs; no in-game economy in v1.
- **Save file migration** (loading v1.x saves in v2.x). Save format is forward-incompatible in v1; future versions need explicit migration design.
- **Cloud save / sync** (Steam Cloud, browser localStorage). Local save files only.
- **Save thumbnails** (FF-style "save slot 1: chapter 3, party of 4"). v1 shows slot number + timestamp.
- **Dialogue translation export.** Deferred with localization.
- **Per-actor dialogue portraits** (Earthbound-style character bust). Text box only.
- **Streamed / scrolling long text** (visual novel flash-fill). Click-to-advance per-line only.
- **Choice systems beyond labels** (rating, multi-axis, time-pressured). Twine-style label jumps only.
- **Inventory tabs / categories** beyond a single category filter parameter on the Inventory template. Multi-tab inventories (FF VII-style weapons/magic/items/key items) deferred.
- **Party management** (multiple playable characters with separate inventories / HP / position). Single Player entity; one blackboard; one inventory.
- **Drop-in / drop-out save** (mid-frame state capture during combat). Save points only — `save_now` fires at well-defined moments, not continuously.

---

## Key Decisions

- **All three sub-deliverables in one v1.** RPG-class is the entire half of the NES library that the prior brainstorms (#1-#5) don't cover. Splitting save/dialogue/menus into separate releases would leave the designer with partial RPG capability for months. Bundle them.
- **Blackboard from idea #5 is the variables/switches store.** No parallel state system. The blackboard's `inventory` reserved key is the inventory store. The blackboard's snapshot IS the save unit.
- **Save = blackboard + scene + entity state.** Includes enough for deterministic restore; excludes asset state, current frame, audio playback position (saved games restart audio from the current scene's bound BGM on load).
- **3 named slots + 1 autosave; no designer config.** Standard NES count (Zelda 3, FF 4). Designers don't need a slot-count picker; 4 is enough. Multi-game save management isn't a v1 concern.
- **Save files in user-config-dir per game title.** Cross-platform standard. The shipped game never asks for filesystem permissions; no per-game path config.
- **Save UI = menu template, not a special-case UI.** Save-Game and Load-Game are two of the 9 menu templates; same rendering pipeline, same navigation, same palette discipline. Reduces special-case code.
- **9 menu templates fixed in v1.** Title, Game-Over, High-Score, Pause, Save-Game, Load-Game, Inventory, Status, Stage-Select cover the NES-class menu surface. Designer-authored custom templates deferred.
- **Templates apply per-scene OR per-verb.** Title / Game-Over / Save / Load / Stage-Select / High-Score are scene-based (one scene = one menu). Pause / Inventory / Status are overlays (verb-triggered, pauses underlying scene).
- **Screenplay-style dialogue syntax.** Writers know it; non-coders can read it; the parser is simple (no AST-shaped grammar). Round-trip preservation matters because designers will git-diff dialogue.
- **Twine-style branching (labels + choices + conditions).** Familiar to anyone who has touched interactive fiction; matches the audience's likely exposure. Conditions reference blackboard via `state.key` — same language as variable interpolation.
- **Stage directions parse to known verbs OR display as italic flavor.** Bridges authored prose with engine action without forcing designers to choose between "story" and "code."
- **Three dedicated workspaces** (Dialogue, Items, Menus). Each is dockable in the existing ImGui dockspace; reuses the workspace registration pattern from U3 of the ImGui migration.
- **Items as flat database, not hierarchical / categorized.** Categories are a tagging field; sorting / filtering happens in the rendering template. No multi-level item taxonomy in v1.
- **No combat engine in v1.** Combat is a complete subsystem (turn order, attack resolution, status effects, AI). Bundling it would double v1's scope. Designers wanting combat compose it from verbs and menus.
- **Single-language only.** Localization is its own product surface; deferring keeps v1 from carrying multi-language schema discipline that costs ongoing maintenance.

---

## Dependencies / Assumptions

- **Depends on idea #5's blackboard** (R10-R13 of #5). Without it, v1 has no variable store, no inventory store, no save unit. Idea #6 cannot ship before #5.
- **Depends on idea #5's verb catalog mechanism** (`catalog.RegisterAction`). RPG verb recipes (`open_dialogue`, `save_now`, etc.) ride the existing catalog registry.
- **Depends on idea #5's input intent layer** (R7-R9 of #5). Menu navigation uses `input/up`, `input/down`, `input/use`, `input/menu`.
- **Depends on idea #3 (NES palette art-director)** for template palette parameters. Templates expose `background_palette_ref` which the engine renders against the current palette. Without #3, the field exists but template visuals are unconstrained.
- **Depends on idea #7 (Project Capsule + Build pane)** for the shipped game to write save files into the user-config dir. The runtime needs filesystem permission + path resolution; planning verifies Ebitengine + Go cross-platform user-config-dir access on Linux/macOS/Windows. WASM saves are a special case (browser localStorage) — flag for planning.
- **Assumes the existing reflection inspector** can render the new RPG-systems workspaces' content (table editors for Items, text editor for Dialogue, template-picker dropdowns for Menus). Planning verifies each new workspace can register a custom rendering path via the same widget-dispatch extension from idea #2 (R2 of #2).
- **Assumes the Dialogue script parser** can be a hand-rolled recursive descent for screenplay format. No external parser-generator dependency. Planning confirms.
- **Assumes ImGui can render** text boxes with line-by-line click-advance + numbered choice lists. The runtime renderer rides the same dockspace-isolated pattern the studio uses; planning confirms.
- **Save format is not human-readable in v1.** Binary (likely encoded `gob` or JSON in a fixed schema). Designers don't edit save files; players don't either.

---

## Outstanding Questions

### Resolve Before Planning

- *(none — scope is fully resolved at the brainstorm level)*

### Deferred to Planning

- **[Affects R11] [Needs design]** Concrete visual designs for the 9 templates. Each template needs a wireframe + the parameter list. Planning produces a draft and validates by walking the NES-class reference set ("can Title template render Mario's title screen? Can Inventory render Zelda's subscreen?").
- **[Affects R20] [Needs design]** Exact `ItemDefinition` schema fields and the categories enum. Planning enumerates and validates against a sample item set covering FF / Zelda / Metroid items.
- **[Affects R7, R8, R9] [Technical]** Exact script grammar for the dialogue parser. Edge cases: how do escaped braces in `{state.key}` work? How does the parser handle nested labels? How are speakers without dialogue lines (e.g., a section break) represented? Planning produces a grammar spec.
- **[Affects R3, R26] [Technical]** Exact entity-state serialization scope for saves. Candidates: all entities in all scenes; only entities in the current scene; only entities with explicit "persistent" flags. Planning picks the cleanest based on file size + restore correctness.
- **[Affects R2] [Technical]** Exact user-config-dir path strategy across platforms. Standard Go libraries provide cross-platform user-config-dir lookup; planning confirms the exact derivation of "per-game subdirectory from game title" (sanitize spaces? lowercase?).
- **[Affects R2] [Needs research]** Save behavior for WASM-shipped games (per idea #7 WASM target). Browser localStorage has different semantics from filesystem; need a separate save path. Planning resolves whether WASM saves write through `localStorage` (size-limited; per-origin) or via the File System Access API.
- **[Affects R5] [Technical]** Exact autosave triggering semantics. Every scene change = potentially noisy autosave (every door triggers a save). Planning picks throttling rules (e.g., autosave at most once per 30 seconds, regardless of scene-change frequency).
- **[Affects R12] [Technical]** Overlay scene-pause semantics. When `open_menu:inventory` fires, what exactly pauses? All entity update? Audio? Scrolling? The Player entity but not background animations? Planning specifies the precise pause contract.
- **[Affects R10] [Technical]** Click-to-advance dialogue + on-screen choices rendering inside the existing scene-as-texture preview workflow. Planning confirms ImGui can paint a translucent UI layer on top of the scene texture without breaking the texture itself.
- **[Affects R23] [Needs design]** Exact list of stage directions parsed to engine actions in v1. The 5 named (`walk_direction_N`, `pause`, `flash_screen`, `play_sound`, etc.) need final selection; mismatched stage directions display as flavor text.
- **[Affects R22] [Technical]** Inventory rendering order in the Inventory template. By insertion order? By category then alphabetical? By rarity? Planning picks a default; designer cannot configure per-game in v1.
