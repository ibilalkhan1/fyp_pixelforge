---
date: 2026-05-15
topic: pixelforge-editor
focus: complete visual no-code GUI game editor for the Pixelforge engine
mode: repo-grounded
---

# Ideation: Pixelforge Visual No-Code Game Editor

## Codebase Context

**Pixelforge engine (in `/home/red/Desktop/render/bilal-go`):** Go retro 2D engine on Ebitengine. 64-color indexed palette, 4 ColorTables for indexed-color compositing, 320×180 default canvas, ~3.46M pixels/sec at 60 FPS, all hot paths zero-alloc.

**Subsystems:**
- `pixelforge_audio` — Paula-style 4-channel mixer with sample-accurate scheduling
- `pixelforge_event` — zero-alloc Publish/Subscribe bus, `SubscribeAll`, source-attribution via `fileloc`
- `pixelforge_routine` — coroutine-style `Step` sequencer
- `pixelforge_gui` — minimal in-game widget library
- `pixelforge_scope` (piscope) — ring-buffer frame debugger
- `pixelforge_metrics` (pimetr) — FPS/budget/heatmap overlay (recently rewritten to draw at native window resolution via `SetNativeOverlay` hook)
- `pixelforge_snap` (pisnap) — paletted-PNG snapshot capture
- `colortable.go` — 4 user-mutable ColorTables; selection rule `(source | target) >> 6`

**Existing `pixelforge_studio` package — heavily stubbed:**
- `editor.go` — 1280×800 window, sprite Place/Select/Delete tools (V/P/X), JSON project file, hand-rolled rectangle chrome with **no real text rendering**
- `codegen/generator.go` — string-concat code-gen with two known defects:
  - line 24: hardcoded `replace github.com/ibilalkhan1/fyp_pixelforge => /home/tux/Pictures/basheer-go` (export broken on every machine that isn't Tux's)
  - line 80: `getSprite()` returns `nil` (exported game cannot draw its sprites)
- No animation, no audio editor, no inspector, no preview/play-in-editor, no undo, no asset hot-reload, no real visual programming
- `editor.go:254` hardcodes `frameW, frameH := 8, 8` — silently corrupts non-8×8 imports
- `EditorState.UpdateCode` and `DrawCode` are `string` blobs — the supposed no-code editor punts to "write Go in a textbox"
- User feedback: "current studio code has a lot of issues especially visual and it is highly incomplete"

**Tension surfaced:** "visual programming" vs "visual data preparation" — the current studio is neither (static placement + manual code blanks).

**External patterns surfaced by web research:**
- PICO-8 / TIC-80 / Picotron — tabbed all-in-one, editor==runtime, palette-as-constraint-and-tool, tools-as-carts
- LDtk — auto-layer rules, typed entity fields, world layouts, super-simple JSON+PNG export
- GDevelop / Construct 3 — event sheets (Conditions | Actions tables) beat node graphs for non-programmers; documented Blueprint spaghetti at >30 nodes
- Aseprite / Pro Motion NG — live palette swap, dithering brushes, indexed-mode workflows
- FamiStudio / BeepBox — piano roll + visual envelope curves, NO hex notation
- Houdini SOPs / Substance Designer — procedural node graphs that bake to deterministic outputs
- Pharo / Smalltalk — image-based dev, halos, "save image"

**No prior learnings** — `docs/solutions/` doesn't exist yet; this is greenfield from a captured-knowledge perspective.

## Topic Axes

1. Asset authoring (sprite/tile/palette/audio creation INSIDE editor)
2. Scene & world composition (drag-drop placement, layers, tilemaps, prefabs)
3. Behavior & logic (visual scripting, FSMs, routines, gameplay rules)
4. Live edit, debug & inspect (play-in-editor, hot reload, inspectors)
5. Project, export & distribute (project format, build, packaging)

## Ranked Ideas

### 1. Editor-as-Pixelforge-Cart (Picotron-pattern dogfooding)
**Description:** The studio chrome (panels, inspectors, asset browser, menus) is itself authored as a Pixelforge program running on Ebitengine, using the engine's own font, palette, ColorTables, sprite system, and event bus. The "running game" is one workspace; tools fade in over it via the same overlay pattern `pimetr`/`piscope` already use. There is no separate "play" mode — the game is always running underneath. WYSIWYG at indexed-color level.
**Axis:** Live edit, debug & inspect
**Basis:** `direct:` `pimetr` and `piscope` already render chrome over a live game canvas via `EventLateDraw` + the `pixelforge_ebiten.SetNativeOverlay` hook (recently added). `pixelforge_gui` exists as a widget library starting point. `external:` Picotron's tools-as-carts model — when the OS, file browser, and code editor are all carts on the same runtime, new tools cost almost nothing.
**Rationale:** Solves the user's reported pain (broken visual chrome, hand-rolled rectangle fills, no text rendering) at its root: today the studio reinvents UI primitives the engine already provides. Eliminates the editor↔runtime gap that produces "works in editor, breaks in build" bugs. Every editor pixel becomes a stress test of the engine. Massive credibility story: "open `editor.pforge` in itself."
**Downsides:** `pixelforge_gui` is currently minimal — needs scrollable lists, focus management, text input, modal dialogs, file pickers before it can host a real editor. Multi-pane high-DPI workflows are hard at 320×180; needs a "logical canvas" zoom layer for chrome.
**Confidence:** 75%
**Complexity:** High
**Status:** Unexplored

### 2. The `.pforge` Declarative Schema + Reflection-Driven Component Registry
**Description:** One declarative file format (CBOR or compact JSON) describes the entire game: `ScreenSize`, `TPS`, `Palette[64]`, `ColorTables[4]`, sprites, animations, scenes, entities, audio bindings, event subscriptions, routine graphs. The runtime loads it directly via a `pixelforge_project` package; the editor mutates the same in-memory struct. **Code-gen produces only a thin `main.go` shim that loads the schema.** A `pfcomponent.Register[T]("Player", meta)` API + struct tags (`pf:"slider,0..100"`, `pf:"color"`, `pf:"sprite"`) auto-emit inspector widgets, serializers, undo entries, and visual-script nodes for every new component — zero per-feature editor code. Save = Build = Live URL becomes trivial because the project IS the runtime state.
**Axis:** Project, export & distribute
**Basis:** `direct:` `codegen/generator.go:24` hardcodes the broken `replace` directive; line 79-83 stubs `getSprite()` to return `nil` — the exported game cannot draw. `EditorState.UpdateCode`/`DrawCode` are `string` (`editor.go:51-52`) — there's no schema for behavior at all. `external:` LDtk's single `.ldtk` JSON drives editor + runtime; Unity's attribute-driven inspector pattern.
**Rationale:** Two foundational defects collapse into one solved problem. Every future engine subsystem (heatmap zones, snapshot triggers, palette anims) gets editor support for free the moment it has a schema entry. Save/load, undo/redo, hot-reload, git-diff, headless CI builds all become one operation on the struct. This is the keystone — survivors 3, 4, 5, 6, 7 all need a place to put their data.
**Downsides:** Schema design lock-in — getting v1 wrong is expensive. Reflection-based registration adds a small startup cost. Need a clear policy on what stays as Go code (custom rendering, exotic algorithms) vs. what's data — that policy affects every other survivor.
**Confidence:** 85%
**Complexity:** Medium-High
**Status:** Unexplored

### 3. Coroutine-Step Visual Scripting + Event Sheets
**Description:** Behavior nodes compile **directly to `pixelforge_routine.Step` coroutines** rather than to a bespoke interpreter. A "Wait 30 frames", "Move to (x,y) over 1s", "Play SFX", "Branch on event" node each maps to an existing routine primitive. Author behaviors in two complementary surfaces: **(a)** a horizontal lane editor for sequences (drag Step cards, no spaghetti wires) and **(b)** GDevelop-style **event sheets** for reactive rules ("WHEN PlayerOverlapsEnemy → SUBTRACT HP"). Add a **recorded-demo** entry mode: enter Record, play the entity, the studio synthesizes the routine + event subscriptions from the input/state trace. Visual-script wires are the same `pievent.Subscribe` calls the rest of the game uses, so the event-bus topic catalog (live publisher↔subscriber graph, edge highlights when a message fires) doubles as the script debugger.
**Axis:** Behavior & logic
**Basis:** `direct:` `pixelforge_routine.New(steps...)` is variadic — *structurally a timeline*, not a graph. `pievent` is the engine's primary decoupling primitive (zero-alloc, `SubscribeAll`, `SetTracing`). `external:` Unreal's documented "Blueprint spaghetti at >30 nodes" anti-pattern (UnrealFest 2024); GDevelop event sheets stay readable past 100+ events; Construct 3's sub-event indenting.
**Rationale:** Picking the wrong visual metaphor (node graphs) locks in years of UX pain — and Pixelforge's coroutine model literally hands us the right metaphor. Because nodes ARE Steps, "view as Go code" is a cosmetic transform — semantics are identical, killing the lock-in fear that kills no-code adoption. Event-sheet style beats node graphs for non-programmers; recorded-demo entry kills the blank-canvas problem.
**Downsides:** Demo-recording works for character-shaped behaviors, less so for HUD/menu logic. Event sheets need careful UX for complex conditions to stay scannable. Step sequencer + event sheet + topic catalog is three connected surfaces, not one.
**Confidence:** 80%
**Complexity:** High
**Status:** Unexplored

### 4. Palette + ColorTables as the Central Animatable Preset Surface
**Description:** The 64-color palette and 4 ColorTables are treated as a single live-bound first-class asset surface, not buried in a code API. Features: **(a)** a swatch grid where every slot is an animatable parameter on a timeline (palette cycling, day/night, damage flash) with keyframes and event-bus triggers; **(b)** a Lightroom-style stack of named non-destructive **ColorTable Presets** ("Dawn", "Sickly Cave", "Boss Red Shift") with an A/B before-after wipe; **(c)** **paint-to-place** tile authoring — paint raw colors onto the world, the editor infers tile boundaries and synthesizes LDtk-style auto-transition rules from neighbor patterns it sees you paint twice; **(d)** palette-aware drop-import — a single PNG drop runs deterministic palette quantization, alpha-gutter slice, frame-strip detection, collision-mask derivation, and `.png.meta` sidecar parsing in one pipeline (no import dialog).
**Axis:** Asset authoring
**Basis:** `direct:` `colortable.go` defines `ColorTables[4]` and `(source | target) >> 6` indexing — uniquely Pixelforge, no PICO-8/TIC-80 analog. `pixelforge.RemapColor`/`SetTransparency` already exist. The studio currently has **zero** ColorTable references in `editor.go`. `editor.go:254` hardcodes `frameW, frameH := 8, 8`, silently corrupting any non-8×8 import. `external:` Aseprite live palette-swap; Pro Motion ordered-dither brushes; LDtk auto-layer rules.
**Rationale:** ColorTables are Pixelforge's signature feature — without an editor surface for them, no-code users will never discover them and the engine collapses into "just another indexed palette." Every preview surface in the editor (sprite browser, scene canvas, font preview, running play-in-editor) repaints in lockstep with the palette via the existing palette-mapping pipeline — one swatch edit, many surfaces updated free.
**Downsides:** Animated palette slots interact with sprite-cycle expectations — needs a clear mental model. Auto-tile rule synthesis from "two examples" is a heuristic that will sometimes guess wrong. Palette-aware quantization on import can be confusing if the user expected exact RGB.
**Confidence:** 90%
**Complexity:** Medium
**Status:** Unexplored

### 5. Continuous Capture Spine: Ring Buffer + Snapshots + Event Log
**Description:** A single always-on capture substrate fed by `pixelforge_snap` (paletted PNG snapshots), the existing frame ring buffer (piscope), and a `SubscribeAll` tap on every `pievent.Target`. From this one stream, the editor offers: **(a) time-travel scrub** — drag back N frames, see pixel state, fired events, active routine Steps; **(b) animation cliplets** — scrub piscope, mark a range, "save selection as clip" → that's your animation data, no separate keyframe model; **(c) regression tests** — promote any frame to a golden-image+input-log test; **(d) GIF/MP4 capture** — store-page assets one click away; **(e) shareable bug repros** — events + inputs + frames + palette state in one zip. Scenes can also be recorded play sessions: deterministic seed + initial state + input stream, edited by branching at any frame.
**Axis:** Live edit, debug & inspect
**Basis:** `direct:` `pixelforge_snap.PalettedImage`/`CaptureOrErr` already exist. `piscope` already maintains a ring buffer of frame snapshots. `pievent.SubscribeAll` already exists with `fileloc` source attribution. `pimetr` already has the heatmap overlay. `external:` rr-style record-replay debuggers, but applied to a 320×180 canvas where it's actually tractable.
**Rationale:** One capture stream serves four wildly different audiences (QA, marketer, developer, community) with no feature being purpose-built. Time-travel debugging without breakpoints is uniquely cheap because the ring buffer infrastructure already exists. A no-code user can't add log statements; their only debugging affordance is "watch it happen again" — this turns intermittent bugs into reproducible ones.
**Downsides:** Memory cost of continuous capture (paletted frames + event log) — needs a configurable budget. Determinism is required for replay — any nondeterministic source (random, time, network) needs a seeded wrapper. Audio capture is harder than visual.
**Confidence:** 85%
**Complexity:** Medium
**Status:** Unexplored

### 6. Paula Audio Without Trackers
**Description:** Replace tracker UI entirely. Three combinable input modes: **(a) 4-row comic strip** — Paula's 4 channels become 4 horizontal panels, each cell a discrete sound moment (sting, loop, noise burst) drawn as pixel-art waveforms in palette colors; arrange panels left-to-right to compose songs and SFX; **(b) Ableton Session-View grid** — rows = 4 channels, columns = scenes/states; cells in a row are mutually exclusive (channel-stealing made visible); trigger conditions drag from the event bus panel onto cells; clips quantize to next bar; **(c) optional hum-mode** — tap rhythms or hum melodies, the editor pitch-quantizes to the current scale and assigns to a free Paula channel. Auto-allocate channels by inferred priority (BGM=locked, SFX=stealable, voice=ducking), with a live mixer lane visualization that **flashes red when a Play() steals a still-active voice**.
**Axis:** Asset authoring
**Basis:** `direct:` `pixelforge_audio.Play(channels, sample, pitch, vol)` requires a manual channel argument that silently steals when overlapped — the #1 footgun. `AudioInfo` struct in studio has only `Name+Path`, no channel field, no SFX category. `external:` FamiStudio's piano-roll-replaces-hex; Ableton Live Session View; GarageBand's smart instruments.
**Rationale:** 4-channel mixers are the #1 frustration in chiptune-style engines — devs waste hours on "why did my footstep cut off the music." Pixelforge has the telemetry to solve it (event bus + piscope timing); surfacing the choice to the user is the bug. Comic-strip panels make Paula's 4-channel constraint a visible creative grid rather than an invisible wall. Hum/tap modes unlock audio for the 90% of solo devs who fear trackers.
**Downsides:** Auto-allocation must never strip a designer's intentional voice-steal. Hum-mode pitch detection is an entire subsystem and may underperform on noisy inputs — needs a clear fallback to manual edit. Three input modes means three UIs to maintain.
**Confidence:** 75%
**Complexity:** Medium-High
**Status:** Unexplored

### 7. Houdini-SOP Procedural Level Graph That Bakes to Tilemap
**Description:** A node graph where nodes are operators (`Scatter`, `CellularAutomata`, `FloodFill`, `PaletteRemap`, `PlaceEntities`, `BSP`, `WaveCollapse`) feeding into a final `BakeTilemap` node. Tweak a seed/radius upstream and the entire dungeon redraws downstream in real time within the editor. The bake step emits a static deterministic Pixelforge scene (just the schema from #2) — **procedural at design time, deterministic and zero-cost at runtime**. Optional: keep the seed in the project so different bake seeds become different scene variants (dungeon levels) without re-authoring.
**Axis:** Scene & world composition
**Basis:** `external:` Houdini SOPs, Substance Designer node graphs, World Machine, Spelunky's level templates. `reasoned:` Pixelforge games are tile-grid-shaped and pixel-tiny — procgen output fits in seconds; baking sidesteps shipping a runtime procgen that would compete with the engine's zero-alloc invariant.
**Rationale:** Pixel-art games need lots of varied levels; hand-authoring is the bottleneck. A SOP-style graph gives indie teams Spelunky-class variety without runtime procgen cost. Lives well alongside the schema (#2) and palette painting (#4): SOP nodes operate on the same color grid that paint-to-place produces. This is the one survivor that addresses the Scene & world axis with novelty rather than convention.
**Downsides:** Node graphs themselves carry the Blueprint-spaghetti risk — must be carefully scoped (10-15 ops max, no general control flow). Procgen designers will want runtime variation too — need to be clear that "bake" is a deliberate constraint. Big learning curve for hobbyists who've never seen Houdini.
**Confidence:** 65%
**Complexity:** High
**Status:** Unexplored

## Rejection Summary

| # | Idea | Reason Rejected |
|---|------|-----------------|
| F3.3 | ColorTables ARE the visual scripting language (paint state into pixels) | Clever but too cute — behavior-via-painting is hard to debug and conflicts with #3's routine-Step approach. ColorTable presets/animation in #4 captures the safe version. |
| F3.8 | Pimetr heatmap as the *primary* play-mode view | Too noisy as default; covered by existing pimetr toggle. Belongs as opt-in, not as a survivor. |
| F5.2 | Notion multi-view entity browser (grid/gallery/kanban/map) | Genuinely useful but presumes the schema in #2 first. Better as a v2 feature once #2 lands. |
| F5.4 | Observable-notebook reactive cells for tuning | Adds a runtime evaluation graph; partially covered by inspector live-edit in #5. Premature. |
| F5.5 | tldraw freeform sketch-then-promote level layout | Novel but the "promote sketch to tilemap" inference is a research project. #4's paint-to-place + #7's procgen cover the practical version. |
| F5.7 | Pharo/Smalltalk halos on every entity | Interesting power-user feature but extra UI complexity over a plain inspector. Reconsider as an "advanced mode" later. |
| F5.8 | Sourcegraph-style semantic search across palette/events/routines | Excellent but not foundational — needs the schema in #2 first to have something to query. v2. |
| F6.2 | Zero-asset editor (everything procedural, no PNGs) | Extreme constraint flip — alienates the pixel-artist target audience. #4's palette-aware import is the pragmatic version. |
| F6.7 | Genre-locked editor variants (JRPG/Shmup/Roguelike editions) | Interesting distribution strategy but premature — pick after the core editor proves itself. |
| — | F1.1, F1.2, F1.3, F1.4, F1.5, F1.6, F1.7, F1.8, F2.1-F2.8, F3.1, F3.2, F3.4, F3.5, F3.6, F3.7, F4.1-F4.8, F5.1, F5.3, F5.6, F6.1, F6.3, F6.4, F6.5, F6.6, F6.8 | Folded into survivors 1-7 to remove duplicates (28 ideas → 7 fused clusters). |
