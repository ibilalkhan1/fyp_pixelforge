---
date: 2026-05-18
topic: pixelforge-nes-class-no-code
focus: studio capable of authoring any complete 8-bit NES-class 2D retro game (Mario, Zelda, Metroid, Megaman, Tetris, Final Fantasy, Bubble Bobble, Punch-Out, Excitebike, Double Dragon — full NES width) end-to-end via the GUI, no code, shipping single-file executables per platform including WASM/web
mode: repo-grounded
supersedes_reasoning: docs/ideation/2026-05-18-pixelforge-no-code-complete-game-ideation.md (anchored to Asteroids/Frogger/Bomberman fixed-screen arcade — narrower than the actual scope; user explicitly broadened to NES-wide design space)
---

# Ideation: Pixelforge Studio as a No-Code NES-Class Game Editor (Re-Run)

## Grounding Context

### Codebase context (post-ImGui-migration, 2026-05-18)

**Engine (`/`):** Go + Ebitengine; 64-color indexed palette with 4 ColorTables; 4-channel Paula-style audio mixer; zero-alloc pub/sub events (`pixelforge_event`); coroutine-style Step sequencer (`pixelforge_routine`); sprite/canvas primitives; ring-buffer capture (`pixelforge_studio/capture`); `pievent.RegisterTarget` + `Inspectable` sidecar convention; reflection-driven component registry (`pfcomponent`) with `pf:"slider,0..100"`-style struct-tag dispatch.

**Studio (`pixelforge_studio/`):** Dear ImGui 1.92.8 docking branch via cimgui-go (ImGui migration completed 2026-05-18); dockable Assets / Inspector / Scene / Capture / Behavior / Palette panels with `imgui.ini` layout persistence; reflection-driven inspector via `pf:"..."` tag dispatch; scene-as-texture preview via `backend.CreateTextureFromGame`; codegen exports a thin Go `main.go` shim + vendored engine + `.pforge` as game state.

### Verified capability gaps for the NES-class scope

NES width is meaningfully larger than the arcade-only scope of the prior 2026-05-18 ideation. Newly-verified gaps from a fresh codebase scan + web research:

1. **No camera abstraction.** Zero matches for `camera`/`scroll`/`parallax` in `pixelforge_project/` or `pixelforge_studio/`. Mario, Metroid, Megaman, Excitebike, Double Dragon, Bubble Bobble all unauthorable without one. Distinction that matters: **Follow** (smooth lerp + dead zone) vs **Locked** (one fixed viewport per screen).
2. **No multi-screen world model.** `Scene.Tilemaps` is a single flat grid per scene with no neighbour relationship. No `World`/room concept anywhere. Zelda overworld, Metroid map, Final Fantasy worldmap blocked.
3. **No tilemap binding.** `TilemapLayer.Grid [][]int` stores raw integers; no `TileSet` first-class concept binding a sprite sheet's frames to named tiles with collision class. Painters write "tile value 7" with no UI saying "this is the brick-with-vine from grass_tiles.png frame 7."
4. **NES palette/sprite limits not enforced or visualized.** `PaletteData.Base [64]string` is freely editable; no sub-palette concept; no 8-sprites-per-scanline warning; no 2×2 palette-block constraint. "NES-class" is currently a vibe, not a constraint.
5. **No dialogue / menu / inventory / save-state primitives.** Zero matches for `dialogue`/`menu`/`inventory` in `pixelforge_project/`. Final Fantasy is 50% menu + 30% dialogue + 20% combat.
6. **No input-binding UI.** `pixelforge_key` / `pixelforge_pad` exist; `grep InputBinding pixelforge_project/` returns nothing. Behaviors hardcode keys.
7. **No scene-flow editor.** Title → gameplay → game-over → high-score state machine has no UI surface.
8. **No icon generation; no per-platform build; no WASM single-HTML bundle.** Codegen produces a Go source tree — designer has no `.exe`/`.app`/`.html`.
9. **No Aseprite / Tiled / LDtk importers.**
10. **The Behavior catalog has 7 verbs (Wait/Tween/Move/Play/Publish/Branch/Custom) and zero of them are "input" or "collide."** Even Tetris (key input + line collision) is unbuildable from the current catalog.

### Past learnings (carried forward from prior 2026-05-18 ideation)
- Schema reserves fields *ahead* of UI (M1 reserved BehaviorGraph/StepNode/EventSheet; M5 walked them — no migration needed)
- Catalogs over hardcoded switches (`pfcomponent.Register`, `catalog.RegisterStep`, `pievent.RegisterTarget`)
- One runtime owner per top-level project entity (Engine per Project)
- Codegen emits a thin `main.go` shim — never string-concat source
- Live preview is always-on (no separate Play/Edit modes)
- **Node graphs explicitly rejected for *gameplay* authoring** (Blueprint-spaghetti anti-pattern); step lanes + event sheets only. Node graphs OK for: scene-flow graph, dialogue trees, runtime debug visualization
- cimgui-go drag-drop with uintptr deferred — Move-Left/Right buttons mutate via `SwapSteps`
- No OS native file picker

### External landscape — NES-class specific (this run's fresh web research)

- **Scrolling primitives** — GDevelop models camera follow as a behavior with dead-zone; Construct uses per-layout scroll origin; GameMaker rooms make scroll-locked default. Two-mode toggle (Follow / Locked) covers the field.
- **Tilemaps** — **LDtk's IntGrid + Auto-layer split is the strongest pattern**: IntGrid layer stores integer collision IDs per cell (0=empty, 1=solid, 2=water); paired Auto-layer rules pattern-match the IntGrid and paint tile imagery automatically. Decouples collision from appearance. RPG Maker autotiles use a fixed 47-combination atlas — simple but inflexible.
- **Multi-screen worlds** — LDtk offers three world modes: **GridVania** (rooms on a world grid — Metroid/Zelda), **Free** (overworld with irregular rooms), **Linear** (Bubble Bobble stage progression). Neighbour detection is automatic.
- **RPG state** — RPG Maker MZ's **Switches** (booleans) and **Variables** (integers) are canonical. Save = snapshot of switch/variable tables. Dialogue trees via Visual Dialogue Branches plugin show the visual node-tree pattern (the exception to no-graphs rule).
- **WASM/web export** — Ebitengine WASM build needs 3 files (`game.wasm` + `wasm_exec.js` + HTML). Go proposal #72055 to single-bundle is open but not accepted. Workaround: inline `wasm_exec.js` and the `.wasm` (base64) into one `.html` at export time. Godot 4 web export also needs 3 files; confirms this is unsolved cross-engine.
- **Native single-file** — Windows true single-`.exe` via `go build` + `goversioninfo` `.syso` for icon. **macOS requires `.app` bundle** (directory, not file) with `Contents/Resources/AppIcon.icns`. Linux is single binary + XDG desktop entry.
- **NES constraints worth preserving** — 8 sprites max per scanline (>8 causes flicker — historically used as technique); 4 BG palettes + 4 sprite palettes × 3 colors each + 1 shared color = 25 on-screen colors max; **the 2×2 background tile blocks share one palette** (the "attribute table" constraint — the most distinctive NES visual signature). Recommendation: enforce sprite quantum + 64-color palette as **hard** constraints; surface scanline limit + 2×2 block as **soft warnings**.

### Stated audience & validation loop (from prior brainstorm session)
- **Designer who can make sprites/audio** (asset creation is not the bottleneck); imports BGM/SFX + sprites from internet or library
- **Audience = friends, classmates, a community** (small known group; validation via observed sessions, not external user research)
- **Not pre-trained on existing game-editor metaphors** (GDevelop/Construct/RPG Maker conventions cannot be assumed); the editor gets to pick its own vocabulary
- **"Shipped" = platform menu → single executable, custom icon, double-click runs offline like retro games / Dino-in-Chrome**

## Topic Axes
- **A1 — World & level authoring:** tilemaps (LDtk IntGrid + Auto-layer), scrolling cameras (Follow/Locked), multi-screen world layout (GridVania/Free/Linear), scene transitions, game-flow graph
- **A2 — Behavior & game logic:** catalog, prefabs, gameplay primitives (score/lives/health/timer), RPG-class state (Switches/Variables, dialogue, menu, inventory, save)
- **A3 — Visual asset pipeline:** sprite + animation editing/import, palette quantization, tile rules + autotile, Aseprite/Tiled/LDtk integration, NES constraint enforcement
- **A4 — Audio authoring:** BGM + SFX composition (patches, comic-strip / session-grid / mixer-lane), bindings to event topics, Paula 4-channel mixer surface
- **A5 — Ship + onboarding:** single-file native binaries (Win .exe, macOS .app, Linux binary), WASM/web (single-HTML inline-bundle), icon-from-sprite, boot-to-canon templates, cold-start-to-shipped path

## Ranked Ideas

### 1. ScreenRoom — the universal NES world primitive
**Description:** Add one schema concept — a `ScreenRoom` (W×H in fixed screen-sized units). A Scene is a grid of ScreenRooms with neighbour adjacency (LDtk GridVania-style). **One primitive covers all 10 reference games:** Mario level = 16×1 rooms, Zelda overworld = 8×8 rooms, Metroid map = 30×20 sparse, Megaman stage = 4×6, Final Fantasy town = 1×1. Camera primitive is "which room am I in + sub-pixel offset." Entities live on tile cells within a room (no free positioning) — the unified coordinate system is what kept NES save files in kilobytes.
**Axis:** A1
**Basis:** `external:` LDtk's GridVania / Free / Linear world modes; NES PPU's nametable architecture historically forced screen-as-unit authoring. `direct:` `Scene.Tilemaps` exists as single flat grid with no inter-scene structure; `pixelforge_project/scenes.go` has no `World`/room concept.
**Rationale:** Pick this primitive *once* and we never have to choose between "Mario-style scroll" and "Zelda-style screen flip" as two systems. Pick anything else and we end up with three world models that don't compose. Tile-cell entity coordinates also unify level diffs, deterministic replays, and undo.
**Downsides:** Locks the editor into screen-quantum thinking; free-scrolling shmups (Gradius-class) become awkward (acceptable: out of NES-class anyway). Designers who want pixel-precision entity placement will resist tile-cell snap.
**Confidence:** 90%
**Complexity:** Medium
**Status:** Unexplored

### 2. TileAtlas as `pfcomponent` + semantic-brush painter
**Description:** Register `TileAtlas` as `pfcomponent.Register[T]("TileAtlas")` so it reuses the U4 reflection inspector. The tile painter is a `pf:"widget=tilepainter"` tag dispatch — no new editor system. Designers paint *semantic* values ("ground" / "wall" / "spike" / "ladder") with a 5-color brush; visual tile selection (corners, edges, transitions) and collision shape are **100% inferred** from auto-rules attached to the tileset (LDtk IntGrid + Auto-layer pattern). One concept → tilemap data, autotile, collision, future damage-tiles, NES nametable previews — all add as struct fields on the same component.
**Axis:** A1
**Basis:** `external:` LDtk IntGrid + Auto-layer decouples collision from appearance (strictly more general than RPG Maker's fixed 47-tile autotile). `direct:` `Scene.Tilemaps` stores raw `int` grids with no `TileSet` binding; `pfcomponent.Register[T]` + `pf:` tag dispatch already power the inspector (U4 of the ImGui migration).
**Rationale:** Mario, Zelda, Metroid, Megaman are 80% tilemap. If the tilemap is "just another component," every future tile feature is a struct field + a widget tag — not a new editor pane. Designer paints 5 colors; gets correctly-tiled, correctly-collidable level. Codegen template for components already handles `pf:` tags, so exported games inherit it for free.
**Downsides:** Semantic-only painting is great for prototype but artists wanting per-tile visual control may chafe; needs an escape hatch. Auto-rule editor is its own UX problem if exposed.
**Confidence:** 80%
**Complexity:** Medium–High
**Status:** Unexplored

### 3. NES palette as art-director + central quantizer + soft-warn constraints
**Description:** Make the project's palette (NES-style: 4 sub-palettes × 3 colors + shared color-0) the project's *identity*. Every sprite editor, tile editor, and UI swatch reads the same palette object — change one color and the entire game restyles live in preview. Add one central `palette.QuantizeRGBA(img)` that every importer calls — drop any 24-bit PNG and it auto-quantizes against the palette with a ghosted "what changed" diff overlay. NES constraints (sprite-per-scanline, 2×2 palette block) surface as toggleable **soft-warn overlays**, not gates — flicker is sometimes intentional.
**Axis:** A3
**Basis:** `direct:` 64-color indexed palette + 4 ColorTables already in engine; `pixelforge_studio/palette/quantize.go` already quantizes (just no central asset-pipeline integration). `external:` Pico-8's reputation comes from constraints visible everywhere; NESdev confirms 2×2 palette-block constraint as the most distinctive NES visual signature; LDtk-style soft warnings vs Aseprite's hard quantize.
**Rationale:** "NES-class" is currently a vibe, not a constraint. Without enforcement, beginners produce slop that doesn't look NES; experts who *want* the constraint can't get it. Centralizing the quantizer means every import path, every export path, every AI-assisted asset workflow rides one well-tested function. Live preview gets free global recolor: pick a strong palette, every sprite/tile/HUD restyles in one frame.
**Downsides:** Designers who want full 24-bit control will resist; needs a per-project opt-out. NES constraints are nuanced (sprite vs background palettes) and the UX must teach without lecturing.
**Confidence:** 80%
**Complexity:** Medium
**Status:** Unexplored

### 4. Audio library/synth picker — no tracker
**Description:** Ship 30+ named NES-authentic SFX patches (LaserShoot, CoinPickup, ExplosionSmall, HeroJump, Damage, MenuConfirm, BossHit) and 4-5 BGM loops (town, dungeon, boss, title, victory) as curated assets that synthesize against the Paula 4-channel mixer. The "audio editor" is a **picker** + 3-5 sliders per patch (pitch, decay, brightness, accent). Drag a patch onto an event topic to bind. Optional advanced mode: sheet-music notation editor (4 voices = 4 staves) for users who want to compose; default is "audition the library, pick one, drag-bind." Carries forward the "Patch-cast" idea from prior ideation, refined.
**Axis:** A4
**Basis:** `external:` Bosca Ceoil / FamiStudio three-panel UX; Eurorack patch presets; sfxr / Bfxr / ChipTone generative SFX; sheet-music notation as compact 4-voice display (extra-credit option). `direct:` `pixelforge_project.AudioBinding` schema is reserved with Topic/ForceChannel/TriggerCondition but no UI walks it.
**Rationale:** Audio is where every no-code game editor's designers get stuck and ship games with no sound. Library-first sidesteps composition entirely while leaving a door open for advanced users. Patch quality determines whether games sound good — content curation is the work.
**Downsides:** Patch library quality is make-or-break — needs real audio design talent to bundle. Bundle size grows with patch count.
**Confidence:** 80%
**Complexity:** Medium–High
**Status:** Unexplored

### 5. Entity verb-sheet powered by input intents + blackboard state
**Description:** Two paired moves.

**(a) Entity verb-sheet:** A sprite/entity *is* its verb list. Selecting "Goomba" opens a **fixed-layout one-screen "character sheet" inspector** (TTRPG analog) with named slots: sprite, animations, stats, "What I do when stepped on: [die / hurt player / bounce]" picked from a closed enum, "What I do when player nearby: [attack / patrol / flee]". No script panel, no event lane, no node graph — behavior is structured-data fields on the entity. The reflection inspector already does this; the verbs are an enum on a pfcomponent.

**(b) Input intents + Blackboard:** Register `pievent` targets `input/jump`, `input/attack`, `input/up`, `input/menu` (backed by a Project-level `InputMap`: D-pad + A/B/Start/Select → intents); register one `pievent.Target` named `state/*` wrapping `map[string]any` (HP, coins, flags, inventory). The verb enums on the entity sheet reference intents and blackboard keys — never raw `pikey.KeyZ` — so remapping, gamepad support, touch buttons for WASM, and recorded-demo replay are all free.

**Axis:** A2
**Basis:** `direct:` `pievent.RegisterTarget` + `Inspectable` pattern is established (`loop.main` / `loop.debug` precedent); `pfcomponent.Register[T]` + `pf:` tags already power the inspector; `catalog.Context.ValueRef` already wires conditions to value paths. `external:` D&D character sheet (fixed-layout one-page entity summary); Unity InputActions / UE5 Enhanced Input prove the intent-indirection pattern.
**Rationale:** This is the catalog/cartridge answer from the prior ideation, sharpened against NES-width and the "designer not pre-trained" constraint. No metaphor to learn — designers see a "character sheet" and fill it in. Verbs are an enum so the runtime never sees an unknown behavior. Intents + blackboard cover ~95% of NES behavior verbs without scripting.
**Downsides:** Closed enum requires the team to identify ~50 verbs covering all 10 reference games; getting the vocabulary wrong forces breaking changes. Verbose enums on RPG-class entities (Final Fantasy character has dozens of stats) may strain the "one screen, no scroll" goal.
**Confidence:** 80%
**Complexity:** High
**Status:** Unexplored

### 6. RPG schemas + theatrical-script dialogue + menu template gallery
**Description:** Reserve four schema shapes now (additive `omitempty`): `DialogueTree`, `MenuLayout`, `ItemDatabase`, `Variables`/`Switches` (RPG Maker style, global + scene-scoped). Dialogue is authored as a **theatrical-script text editor** (`MARIO: (walks left 4 tiles) Where is the princess?\nBOWSER: (laughs) Try the next castle.`) — round-trips losslessly back to script form; parser converts to a runtime tree. Menus are NOT visually laid out — designer fills typed tables (party, items, stats) and picks one of 9 NES-canonical UI **templates from a thumbnail grid** (FF menu, Zelda subscreen, Megaman pause). Inventory is a flat item database. Save state = serialize the blackboard from #5.
**Axis:** A2
**Basis:** `direct:` zero matches for `dialogue` / `menu` / `inventory` in `pixelforge_project/`. `external:` RPG Maker MZ Switches+Variables as canonical RPG model; Visual Dialogue Branches plugin proves visual dialogue trees ship in this space; theatrical script as the most-tested human-readable encoding of time + multiple speakers + parallel action ever invented.
**Rationale:** Final Fantasy is 50% menu, 30% dialogue, 20% combat. Zelda has dialogue + inventory. Megaman has password save. Without these schemas, half the NES library is unreachable. Theatrical script gets writers (a common designer profile) on-ramp instantly; template gallery removes "lay out a menu" as an authoring task.
**Downsides:** Template gallery only works for canonical NES UI shapes; designers wanting unique menus hit the template wall. Script-format dialogue is great until complex branching is needed (use the dialogue-tree exception to the no-graphs rule).
**Confidence:** 75%
**Complexity:** High
**Status:** Unexplored

### 7. Project Capsule + Build pane
**Description:** Codegen emits exactly **two meaningful files**: `project.pforge` (data) and `capsule.go` (a generated `func Run(opts) error` plus typed accessors for every component, scene, ExtensionHook). `main.go` is 6 lines: `func main() { capsule.Run(capsule.Defaults()) }`. **The capsule loads any `.pforge`** — so editor live-preview, regression tests, the Forgequest tutorial cart, and the shipped game all run through one runtime contract.

A **Build pane** offers five checkboxes (Win / macOS / Linux / WASM / Source) invoking `go build` under each `GOOS/GOARCH`. WASM uses inline-HTML bundling (`game.wasm` + `wasm_exec.js` base64-embedded in `game.html`) for true **single-file double-clickable browser-playable** output. Icon auto-generated from the project's most-used 16×16 sprite via `goversioninfo` `.syso` for Windows, `.icns` for macOS, favicon for WASM. Build runs in background on every save — drag-the-file-to-Discord ship UX, no Export menu needed.

**Axis:** A5
**Basis:** `direct:` `pixelforge_studio/codegen/generator.go` already emits a thin `main.go` shim + vendored engine; no per-platform build step exists today. `external:` Ebitengine's `GOOS=js GOARCH=wasm` works; goversioninfo handles Windows `.syso` icons; macOS needs `.app` bundle; Pico-8 single-HTML export sets the bar; Go proposal #72055 confirms `wasm_exec.js` inline is the cross-engine workaround.
**Rationale:** Without this the user has nothing to send their friend. The "no code, ship a game like retro ROMs" promise collapses at the last step. Capsule architecture turns "thin shim" from a convention into a contract — every shipping path goes through the same import.
**Downsides:** Per-platform build requires `go` toolchain on user's machine OR studio ships a vendored toolchain. macOS `.app` bundle isn't truly a single file (it's a directory). Build-on-every-save consumes CPU; needs debouncing.
**Confidence:** 85%
**Complexity:** High
**Status:** Unexplored

## Rejection Summary

| # | Idea | Reason Rejected |
|---|------|-----------------|
| S1 | Camera-rig component | Folds into #1 — ScreenRoom defines "which room + sub-pixel offset" |
| S2 | Catalog verb expansion (OnKey/OnCollide/Spawn) | Folds into #5 — verb enums on entity sheets ARE the catalog |
| S3 | RegisterWidget public registry | Emerges naturally from #2's `pf:"widget=tilepainter"` use; not a discrete survivor |
| S4 | Lego instruction-booklet build steps | Interesting tutorial mechanism but tutorials need authoring tools to exist first |
| S5 | Sheet music notation for audio | Folded into #4 as optional advanced mode |
| S6 | Pinball zone-trigger authoring | Competes with #5; zones could be one verb-target in #5's vocabulary |
| S7 | Cellular automata for room generation | Premature — needs #1 (ScreenRoom) and #2 (TileAtlas) shipped first |
| S8 | Catalog freeze v1 governance | Discipline rule that emerges from #5's closed enum; doesn't need to be separate |
| S9 | Zero-channel default — opt-in audio | Too opinionated; punishes the 90% case where designers want sound |
| S10 | SQLite binary `.pforge` | Premature optimization; live-preview-with-fat-JSON may never materialize for retro-scale games; JSON gives free git-diffability |
| M7 | Capture ↔ Scrubber ↔ Permalink | Strong leverage but second-tier vs world/behavior foundations; revisit after #1-#7 land |
| M8 | Forgequest tutorial cart + jam timer | Onboarding depends on the authoring loop being real first |
| M12 | Parametric sprite recipes | User stated designer CAN design sprites — parametric is not the bottleneck; folds into #3's central quantizer for the import path |
| M13 | Inferred topology — scene-flow viz | Visualization, not foundation; useful after #5's event vocabulary stabilizes |

## Cross-cutting combinations worth knowing about

These are not separate survivors — they're the natural pairings that emerge when multiple survivors land together:

- **CX1: #1 + #2** = complete world-authoring foundation (ScreenRoom + TileAtlas). Camera follows from #1, painting follows from #2.
- **CX2: #5 + #6** = complete behavior surface. Entity verb-sheet (#5) consumes the RPG schemas (#6) — Final Fantasy character is a Goomba-style verb-sheet with more verbs.
- **CX3: #3 + #2** = complete visual identity stack. Palette change → all tiles re-derive correctly via the auto-rules in #2.
- **CX4: #7 + capsule-loads-any-pforge** = onboarding wins for free. A `Forgequest` tutorial cart (deferred) is just another `.pforge` the same capsule runs.

## Relationship to prior ideation

The previous 2026-05-18 ideation (`docs/ideation/2026-05-18-pixelforge-no-code-complete-game-ideation.md`) was anchored to Asteroids / Frogger / Bomberman fixed-screen arcade. Its idea #1 (Genre Cartridges) was correctly rejected during the brainstorm session that followed it because the user clarified the actual scope is NES-wide. This re-ideation broadens the design space.

**What survives from the prior ideation:**
- Boot-to-canon, score/lives/health as pfcomponent, prefab spawning, sprite-sheet slicer — these remain valuable but second-tier vs the world/behavior foundations surfaced here. They will likely re-emerge as deliverables under #1, #2, #5, #6.
- The "closed catalog" discipline (no scripting until provably insufficient) survives — sharpened into #5's entity verb-sheet with intent + blackboard infrastructure.

**What's new in this re-ideation:**
- **ScreenRoom** as the universal world primitive (was implicit in arcade scope; explicit and load-bearing for NES width)
- **TileAtlas as pfcomponent + semantic-brush painter** (arcade scope didn't need tilemaps)
- **RPG schemas + theatrical-script dialogue** (RPG-class was out of the prior scope)
- **NES-specific palette art-direction + soft-warn constraints** (arcade scope didn't engage the 2×2 palette block discipline)
- **Project Capsule + Build pane** (the prior ideation deferred ship pipeline as emergent; here it's a discrete survivor because the user explicitly required platform-select menu + WASM + icon)
