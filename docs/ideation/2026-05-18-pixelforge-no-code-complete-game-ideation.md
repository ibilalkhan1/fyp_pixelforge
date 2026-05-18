---
date: 2026-05-18
topic: pixelforge-no-code-complete-game
focus: studio capable of authoring complete 2D retro games (Asteroids / Frogger / Bomberman class) end-to-end via the GUI without code — sprites moving, scenes, BGM/SFX, game logic via drag/drop or visual scripting
mode: repo-grounded
---

# Ideation: Pixelforge Studio as a No-Code Complete-Game Editor

## Grounding Context

### Codebase context (post-ImGui-migration, 2026-05-18)

**Engine (`/`):** Go + Ebitengine; 64-color indexed palette; 4 ColorTables; 4-channel Paula-style audio mixer; zero-alloc pub/sub events (`pixelforge_event`); coroutine-style Step sequencer (`pixelforge_routine`); sprite/canvas primitives; ring-buffer capture (`pixelforge_studio/capture` — engine-side); `pievent.RegisterTarget` + `Inspectable` sidecar convention; reflection-driven component registry (`pfcomponent`) with `pf:"slider,0..100"`-style struct-tag dispatch.

**Studio (`pixelforge_studio/`):** Dear ImGui 1.92.8 docking branch via cimgui-go (just shipped — 10-unit migration completed 2026-05-18); dockable Assets / Inspector / Scene / Capture / Behavior / Palette panels with `imgui.ini` layout persistence; reflection-driven inspector via `pf:"..."` tag dispatch; scene-as-texture preview via `backend.CreateTextureFromGame`; codegen exports a thin Go `main.go` shim + vendored engine + `.pforge` as game state; lane-editor step list with Move-Left/Right reorder buttons (drag-drop deferred per cimgui-go uintptr-payload tradeoff).

**Strategy summary:** The ImGui migration plan (`docs/plans/2026-05-17-001`, status: completed) is the most recent authoritative substrate decision. Plans 2026-05-15-001/002, 2026-05-16-001/002 are partially superseded — their *feature targets* (palette UI, audio editor, scripting workspace, asset import) remain authoritative but ride the ImGui substrate now.

**Verified capability gaps (codebase scan):**
1. No visual behavior beyond Step lane button list
2. No audio editor surface (M6 reserved, never shipped under ImGui)
3. No sprite-sheet slicer UI (`SpriteAsset.FrameW`/`FrameH` requires JSON edit)
4. No animation editor (timeline / keyframes)
5. No input-binding UI (`pixelforge_project` has no `InputBinding` type)
6. No Play / Pause / Step controls in the Scene workspace
7. No asset-import drag-drop (asset-browser empty state hints at a `File → Import` menu item that doesn't exist)
8. No scene-transition wiring UI
9. No standardized score / lives / health primitives in schema
10. No collision-mask editor

**Past learnings (load-bearing):**
- Schema reserves fields *ahead* of UI (M1 reserved BehaviorGraph/StepNode/EventSheet, M5 walked them — no migration needed)
- Catalogs over hardcoded switches (`pfcomponent.Register`, `catalog.RegisterStep`, `pievent.RegisterTarget`)
- One runtime owner per top-level project entity (Engine per Project; lifecycle pinned to ProjectListener)
- Codegen emits a thin `main.go` shim that loads `.pforge` — never string-concat source generation
- Live preview is always-on (no separate Play/Edit modes); now expressed as scene-as-texture inside the Scene window
- **Node graphs explicitly rejected for authoring** (Blueprint-spaghetti anti-pattern); step lanes + event sheets only. ImNodes reserved for read-only runtime debug visualization only.
- cimgui-go drag-drop with uintptr payload is deliberately deferred — reorder buttons mutate via `SwapSteps` and that's the established pattern
- No OS native file picker — ImGui modals

**External landscape (web research):**
- **Event sheets win, node graphs lose for non-coders.** GDevelop / Construct 3 / GameMaker all use event-sheet authoring at scale (100+ events stays readable). Godot removed Visual Script in v4 because node graphs didn't retain users.
- **Genre-locked completion rate is real.** RPG Maker is the canonical proof; Bitsy / PuzzleScript constrain scope to one room / one mechanic and reach completion in one session.
- **The "you can build any game" trap is fatal.** Every general-purpose no-code editor claims this; the complexity wall hits around 3+ interacting behaviors (player has health AND collision AND score). Mitigation: opinionated templates that pre-wire interactions.
- **Asset cliff & tutorial gap kill onboarding.** Non-coders stall when they must produce sprites/audio/levels before seeing their game run; tools that don't show how to wire score+lives+game-over into a shippable loop lose users before first completion.
- **Audio for non-musicians works via patches, not trackers.** FamiStudio / Bosca Ceoil / Beepbox / sfxr / Bfxr / ChipTone — preset-driven, three-panel UX (instrument / pattern / mixer), no piano roll required.

## Topic Axes
- A1 — Game logic authoring: step lanes, event sheets, conditions/actions catalog, drag/reorder UX, debug
- A2 — Asset pipeline: sprite import + sheet slicing + animation editor; WAV import; palette quantization UX
- A3 — Scene & gameplay primitives: entity drag-to-place, scene transitions, input bindings, arcade primitives (score/lives/health/spawn/lanes/grids)
- A4 — Audio authoring: comic-strip / session-grid / mixer-lane composition surface, SFX generation, auto-allocator
- A5 — Onboarding & shipping path: templates, tutorials, "one-session-to-shipped", starter asset packs, export polish

## Ranked Ideas

### 1. Behavior catalog + Genre Cartridges
**Description:** A closed verb vocabulary of named behavior blocks (`MoveWithArrows`, `WrapAtEdges`, `ShootOnSpace`, `DestroyOnCollide`, `ScoreOnDestroy`, `RespawnAfterDelay`, `PatrolLane`, `DropBomb`, …) registered via the existing `catalog.RegisterStep` seam, bundled into three Genre Cartridges (Asteroids / Frogger / Bomberman) — pre-wired Step lanes + standard event topics + standard input actions. User authors *content* (sprites, levels, palettes, tuning) by configuring catalog instances; scripting is forbidden until the catalog is provably insufficient.
**Axis:** A1
**Basis:** `direct:` `catalog.RegisterStep(name, builder)` already exists with the 7-Step seed (Wait/Tween/Move/Play/Publish/Branch/Custom); the M5 plan reserved `BehaviorGraph` / `StepNode` / `EventSheetRule` / `Condition` / `Action` schema fields. `external:` Godot removed Visual Script in v4 because node graphs lost non-coders; GDevelop event sheets scale past 100 events readably; RPG Maker's genre-locked completion rate is canonical.
**Rationale:** This is the answer to "how do you author logic without code?" — recognize patterns from a curated set, don't invent them. Cartridges sidestep the "you can build any game" failure mode by being unapologetically opinionated. The closed-vocabulary discipline matches the project's prior decisions (no node graphs, catalog over switches).
**Downsides:** Locks the team into the cartridge metaphor and closed catalog; if a user wants behavior outside it, they hit a wall and must request a new block. Catalog growth is permanent maintenance burden. Three cartridges may anchor the editor's identity too narrowly for users wanting non-arcade games.
**Confidence:** 85%
**Complexity:** High
**Status:** Explored

### 2. Boot to the canon
**Description:** Studio opens with a fully-playable cart already loaded (Snake / Pac-Man / Piano — the games already in-tree at `/snake`, `/pacman`, `/piano`, `pixelforge_examples/`). User's first act is "modify what's running" not "create from blank." Each cart carries inline tutorial annotations (next-step highlights pointing at specific docks); finishing the modification path IS the user's first ship.
**Axis:** A5
**Basis:** `direct:` `/snake`, `/pacman`, `/piano`, `pixelforge_examples/` are in-tree but `pixelforge_studio/editor/file_menu.go:25-31` instantiates a hard-coded blank `pixelforge_project.NewProject("untitled")` with zero sprites / zero audio / one empty scene named "Main" / zero behaviors. `external:` Bitsy / PuzzleScript / GameMaker default working projects; PICO-8 cart culture proves "forkable artifact as onboarding."
**Rationale:** Solves three failure modes simultaneously — blank-canvas paralysis, asset cliff, tutorial gap. Also dogfoods every subsystem on every launch so regressions surface before user content exists.
**Downsides:** The in-tree example games may need to be re-saved as proper `.pforge` carts (currently they're standalone Go programs); tutorial annotations need a schema reservation; some users may resent being shown a working game when they want a clean slate (mitigation: "New Empty Project" stays available).
**Confidence:** 90%
**Complexity:** Medium
**Status:** Unexplored

### 3. Score / Lives / Health / Timer as `pfcomponent` + auto-HUD
**Description:** Register `Score`, `Lives`, `Health`, `Timer`, `Cooldown`, `GridPos` as standard `pfcomponent.Register[T]` types with `pf:"slider,..."` and `pf:"event,..."` tags. Inspector edits them for free (U4 reflection dispatch); saver round-trips them for free; catalog conditions (`value_lt`, `value_eq`) see them via `ValueRef`; audio bindings can fire on `Health < 0`; HUD overlay is a `Text` component reading `"Score: {Score.Value}"`. Zero engine code added beyond the registrations.
**Axis:** A3
**Basis:** `direct:` `grep -E "score|lives|health" pixelforge_project/` returns zero matches — schema has nothing for these; `pfcomponent.Register[T]` with `pf:` tag dispatch already powers the inspector. `external:` RPG Maker / PuzzleScript ship these as built-ins precisely so first-game authors don't reinvent the wheel.
**Rationale:** Asteroids / Frogger / Bomberman are unplayable without score + lives + game-over. Routing them through the existing component registry means *every* future primitive (combo meter, fuel, ammo) is one `Register` call — no per-primitive editor work ever.
**Downsides:** The inspector is the editing surface; users still need to wire the events that mutate score/lives, which presupposes #1 (the catalog must include `+10 Score on enemy_destroyed`).
**Confidence:** 85%
**Complexity:** Low–Medium
**Status:** Unexplored

### 4. Input binding by capture
**Description:** No matrix UI for keybindings. User clicks "Bind Fire" on an action; the studio enters a 5-second capture; the next key/button press becomes the binding. Recording flows through the existing `capture.Recorder` `SubscribeAll` pipeline. Schema reserves `Inputs []InputBinding` ahead of UI. The same recording mechanism powers regression replay, recorded-demo synthesis, and end-user remap in shipped games.
**Axis:** A3
**Basis:** `direct:` `pixelforge_key` / `pixelforge_pad` packages exist; `capture.Recorder` already taps inputs via `SubscribeAll`; `grep InputBinding pixelforge_project/` returns nothing — schema has zero input concept. `reasoned:` capture infra already exists and "press the key you want" is structurally cheaper than rendering a settings matrix.
**Rationale:** Closes the input-binding gap with a UX that doubles as a viral demo. Slashes "input bindings" from a milestone to a sub-feature of capture.
**Downsides:** Multi-modifier chords (Ctrl+Shift+F) need extra design; capture-mode focus stealing inside the editor must be handled carefully so the user's keypress doesn't trigger an ImGui menu shortcut.
**Confidence:** 75%
**Complexity:** Medium
**Status:** Unexplored

### 5. Patch-cast audio editor
**Description:** Audio workspace is a three-panel `pievent` listener pane (sample list left, cell-grid bindings center, Paula mixer lanes right) registered under `audio.studio`. Ship 20-30 named arcade audio Patches (LaserShoot, CoinPickup, ExplosionSmall, HeroJump, BossSting + 4-5 looping BGM grooves) — each exposes 3-5 sliders (pitch, decay, brightness, accent). User picks a Patch, tweaks sliders, drags onto an event topic. No tracker, no piano roll.
**Axis:** A4
**Basis:** `direct:` `pixelforge_project.AudioBinding` schema is reserved with `Topic` / `ForceChannel` / `TriggerCondition` but no UI walks it; `pievent.RegisterTarget` + `Inspectable` pattern is the established convention. `external:` Bosca Ceoil / FamiStudio three-panel UX; Eurorack patch presets; sfxr / Bfxr / ChipTone for generative SFX.
**Rationale:** "No audio editor" is the largest user-visible gap. A patch-cast surface fits one workspace, ships in weeks not months, and respects the 4-channel Paula constraint as a feature rather than a limitation.
**Downsides:** Patches must be hand-crafted and curated — the quality of the bundled patch library determines whether anyone's BGM/SFX sound good. Bundle size grows with patch count.
**Confidence:** 80%
**Complexity:** Medium–High
**Status:** Unexplored

### 6. Sprite-sheet slicer with auto-detect + Aseprite importer
**Description:** Two import paths. **(a)** Drop a PNG → modal opens with grid overlay, "Detect" button infers cell size from transparent-gutter spacing or alpha-channel bounding boxes, user accepts or nudges with +/− buttons, persists to `SpriteAsset.FrameW`/`FrameH` + a single-frame `AnimationClip`. **(b)** Drop an `.aseprite` file → studio auto-slices using Aseprite's frame tags + layers, runs the existing palette quantizer against the 64-color palette, shows a previewable mismatch report. Both paths use an ImGui modal — no OS file picker, no JSON editing.
**Axis:** A2
**Basis:** `direct:` manual JSON editing for `SpriteAsset.FrameW`/`FrameH` is documented in the gap inventory; asset-browser empty state prints `"No assets — File → Import to add"` but `pixelforge_studio/editor/file_menu.go:170-181` has no Import menu item — the UI is lying to the user. `external:` LDtk's Aseprite-watch integration is the proven pattern in the same retro-2D niche.
**Rationale:** The single most painful current friction. Without this, even a templated starter cart can't be re-skinned without quitting the editor and editing JSON.
**Downsides:** `.aseprite` parsing pulls in a binary-format dependency; the Aseprite path could be deferred to a v2 follow-up if scope pressure demands.
**Confidence:** 90%
**Complexity:** Medium
**Status:** Unexplored

### 7. Prefabs + Spawn / Destroy / Each catalog Steps
**Description:** Add `pfcomponent.RegisterPrefab(name, []EntityComponent{...})` so an entity archetype (Bullet, Asteroid, Frogger-car, Bomb) is defined once and instances are spawned as sparse diffs against the prefab. Three new catalog Steps consume it: `Spawn(prefab, x, y)`, `Destroy(entity)`, `Each(prefab, body...)`. Bullet firing = one Step. Frogger's lane traffic = one `Each` loop with `Wait`. Bomberman's bomb explosion cascade = `Each` over neighbouring tiles. The catalog grows linearly; the gameplay surface it covers grows combinatorially.
**Axis:** A3
**Basis:** `direct:` `pfcomponent.Register[T]`, `catalog.RegisterStep`, `Scene.Entities []Entity` with sparse `EntityComponent.Values map[string]any` all already exist — this is composition of present primitives. `reasoned:` Without prefabs + spawn, every entity must be placed at edit time, which is incompatible with every arcade game in the target roster.
**Rationale:** Pairs tightly with #1 (catalog vocabulary) — without prefab spawning, no behavior catalog can author waves or fire. The leverage is asymmetric: small catalog addition, massive gameplay surface unlock.
**Downsides:** Prefab schema needs a sparse-diff format that round-trips cleanly; instance count management (entity slice growth) needs bounds to avoid runaway spawning.
**Confidence:** 85%
**Complexity:** Medium
**Status:** Unexplored

## Rejection Summary

| # | Idea | Reason Rejected |
|---|------|-----------------|
| S5 | Templates gallery on New | Subsumed by #2 (Boot to canon is the more aggressive version) |
| S6 | Collision contracts at sprite | Requires a collision system that doesn't yet exist; basis reasoned-only; high burden vs value |
| S7 | Scene transitions as local exits + minimap | Overlaps with #1 (cartridges author transitions); pievent lifecycle bus carries scene-change events |
| S8 | `.pforge` IS the running game; codegen at Ship only | Already true post-ImGui — scene-as-texture interprets `.pforge` live and codegen emits a thin shim |
| S9 | Cart Remixer / Fork-in-Pixelforge button | Distribution mechanism, not authoring; scope overrun for the focus — strong follow-up after the authoring story works |
| S10 | Aseprite importer (standalone) | Folded into #6 as the second import path |
| S11 | Live Preview as direct-manipulation authoring | Invalidates the just-shipped DockSpace + inspector; basis reasoned; too ambitious vs recently-shipped substrate |
| S12 | Target Device Presets | Depends on WASM/touch export targets that don't exist yet; not grounded |
| S13 | `pievent` lifecycle bus per scene | Strong leverage primitive but emerges as infrastructure under #3 + #1 + #5; not a discrete user-facing deliverable |
| S14 | `AssetRef[T]` primitive owning import-on-resolve | Folds into #6 (the slicer IS the inspector-driven import experience) |
| S15 | `runtime.Engine.Run` unified harness | Architectural pattern that emerges from #2 (editor IS the harness) + codegen (already a thin shim); not discrete to discuss separately |
| S16 | Mise-en-place Asset Bays | Naming-as-feature; actual work covered by #6 |
| S17 | DAW Session/Arrangement Duality | Clever but bottleneck is logic catalog (#1), not authoring-surface duality |
| S18 | OBS Scene-with-Sources | Parallel to #3 (HUD) + S7 (transitions); overlap |
| S19 | TCG Card Effects | Subsumed by #1 (the catalog IS cards under a different surface metaphor) |
| S20 | Speedrun Route as Tutorial Cart | Folded into #2 — boot-to-canon carts carry the tutorial annotations |
| S21 | WaveFunctionCollapse Tile Rules | Depends on a tilemap system not yet shipped; high burden |
| S22 | One-Scene Law | Too restrictive — Bomberman / Frogger want title + arena + game-over; cuts target genres' real shape |
| S23 | Five Sprites + Sprite Roles | Sprite-cap too extreme; the Roles half folds into #1's catalog vocabulary |
| S24 | Focus Mode (single panel, no docks) | Conflicts with the just-shipped DockSpace work; scope overrun |
| S25 | 4KB Demoscene .pforge | Curio; goal is shippable arcade games, not size-coded demos |
| S26 | Pro Mode w/ Codegen preview | Expands audience from "non-coders" to "non-coders + Unity engineers"; scope overrun |
| S1 | File → Import menu wiring | Tactical sub-feature of #6 (the slicer needs Import to feed it) |
| S2 | Inline step-arg editing in lane | Tactical sub-feature of #1 (catalog steps need editable args to be useful) |
| S3 | Play / Pause / Step toolbar | Tactical; ships alongside any survivor's first iteration |
| - | axis: A1 | only 1 survivor (#1) — that one block spans the whole game-logic question; no second authoring-surface idea passed the bar |
