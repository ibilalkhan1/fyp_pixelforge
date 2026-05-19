---
date: 2026-05-18
topic: pixelforge-ship-loop-v2
focus: complete the GUI/studio so users can build + ship the four named reference games (Asteroids, Bomberman, Mario, Donkey Kong) end-to-end no-code, with the resulting games runnable as self-contained executables (host system or browser) — no engine ship required, just share the binary/.html
mode: repo-grounded
---

# Ideation: Pixelforge No-Code Ship Loop v2 (Post-Plan-008)

## Grounding Context

### Codebase context (post-plan-008, 2026-05-18)

**Engine (`/`):** Go + Ebitengine 2.9.9; 64-color indexed palette; 4-channel Paula-style audio mixer; zero-alloc pub/sub (`pixelforge_event`); coroutine-style Step sequencer (`pixelforge_routine`); ImGui via cimgui-go.

**Studio (`pixelforge_studio/`):** Dockable workspaces (palette, capture, scripting, input, blackboard, audiolib, dialogue, items, menus, build); plan-008 added `assetlibrary/` (manifest + downloader + workspace + credits), `capsuleruntime/` (Boot, sinks, credits registry), `ingest/` (classifier + dispatcher + watcher + drag-drop), `buildpipeline/builders_long.go` (real `go build` behind `//go:build long`), `buildpipeline/icon_logo.go` (SVG raster → .ico/.icns/.syso/favicon).

**Plan-008 ship state (just completed 2026-05-18):**

| Layer | State |
|---|---|
| Capsule runtime loaders (U1) | ✅ Real — sprites/audio/dialogue decode into registries at boot |
| `verbs.bus` typed pub/sub (U2) | ✅ Real — recipes publish; subscribers route by topic |
| Real `go build` Host/WASM (U3) | ✅ Real — `-tags=long` builds runnable artifacts |
| Two-button Build UI + cross-OS rejection (U4) | ✅ Real |
| Logo rasterization → .ico/.icns/.syso/favicon (U5) | ✅ Real |
| WASM save backend on localStorage (U6) | ✅ Real |
| Capability matrix doc + 11 arcade recipes (U7+U8) | ✅ Real (recipes registered) |
| Asset library manifest + downloader (U9) | ✅ Substrate only — manifest URL 404s, no published packs |
| Credits UX (U10) | ✅ Real — embeds in capsule + WASM splash |
| Custom-ingest watcher + drag-drop (U11) | ⚠ Watcher runs; **dispatcher has no runners wired** |
| Library workspace (U12) | ✅ Real — but renders empty until S3 lands |

**Verified runtime gaps (the "no, not yet" set):**
1. **8 of 11 verb-recipe sinks are `log*Sink` stubs** (`capsuleruntime/subscribers.go:503–537`): Music, Scene, Damage, Motion, Spawn, Visual, Dialogue rendering, snapshot capture. Authored `jump` → log line, sprite doesn't move.
2. `save_now` writes literal `pisave.Snapshot{}` — saves "work" but persist nothing (`subscribers.go:651`).
3. Asset library URL `https://github.com/ibilalkhan1/fyp_pixelforge/releases/download/asset-library-v1/manifest.json` returns 404; no packs published. (Plan-008 explicitly defers asset-pack assembly.)
4. `main.go:95` creates `ingest.NewDispatcher()` but never calls `SetSpriteRunner` / `SetSFXRunner` / `SetBGMRunner` — drag-drop silently no-ops.
5. Only `mario_strip_scene.pforge` + `goomba_scene.pforge` exist as fixtures; no Asteroids / Bomberman / DK fixture, no per-game e2e test, no preview-vs-shipped parity check.
6. Input bindings are dropdown-only (`input/workspace.go:127–142`) — no "press key to bind" capture.
7. WASM HTML size is unmeasured + unbounded; bundler inlines base64 with no gzip / wasm-opt / budget gate.

**Load-bearing prior decisions (do not relitigate):**
- Node graphs explicitly rejected for authoring (Blueprint anti-pattern; Godot 4 removed Visual Script).
- ImGui via cimgui-go is the substrate; `pixelforge_gui` frozen.
- Engine packages stay unaware of verb-recipe topic surface (cycle-break via runners injected from outer packages).
- `verbs.bus` is a dedicated `pievent.Target[*VerbEvent]`; loop.main stays untouched `Target[Event]`.
- Schema additivity discipline (omitempty + sanitize on load; never remove).
- Codegen emits a thin shim; never string-concat source generation.
- Build UI is Host + WASM only; cross-OS native rejected fast.
- All-pure-Go toolchain (oksvg, rasterx, golang-ico, icns/v2, goversioninfo, fsnotify); engine `CGO_ENABLED=0` for WASM.

**External landscape (web research):**
- GDevelop compiles event sheets to JS at export — no interpreter at runtime.
- Godot's "Embed PCK" appends game data to a stripped engine binary; single .exe with all resources self-contained.
- PuzzleScript / Bitsy / PICO-8: game-as-text embedded in single HTML; runtime parses and runs.
- eihigh/wasmgame is the community standard for Go WASM builds; `wasm_exec.js` version-coupled to Go compiler; true single-HTML requires base64 embedding.
- Kenney.nl + OpenGameArt have CC0 packs perfectly sized for the four named games; embedding via `embed.FS` is Go-native fit.
- `solarlune/resolv` is the proven pure-Go 2D collision lib for Ebitengine.

### User authorization (constraint expansion, 2026-05-18)

External libraries, downloaded code, and online tooling are explicitly OK provided they don't break what plan-008 just landed and respect the cycle-break + ImGui + schema-additivity + CGO=0 invariants.

## Topic Axes

- A1 — Runtime subsystems (scene controller, motion, collision, spawn-destroy, visual state, damage, BGM, dialogue rendering, snapshot capture)
- A2 — Authoring UX completeness (input-binding capture, animation timeline, collision-mask editor, scene templates, debug overlay)
- A3 — Ship-loop distribution (true single-HTML WASM, single-exe asset embed, size, signing, install-free play)
- A4 — Content & assets (curated CC0/CC-BY library, project wizards, custom-ingest dispatcher wiring)
- A5 — Validation & proof (four built-in example games, golden-path tests, hot reload, preview-vs-shipped parity)

## Ranked Ideas

### 1. Genre-Templated Zero-Config Authoring
**Description:** Each of the four named games ships as a starter template with every default pre-wired: physics preset, collision mode (auto-derive from alpha), input bindings (WASD+Space for platformers, arrows+Z for shooters), save mode (auto-snapshot every component), verb-recipe bindings, scene layout, sample sprites. `File → New → Mario` yields a runnable game in 0 clicks after the template loads. The user *only* edits content (sprites, levels, numbers, behavior tweaks); they never assemble Mario from atoms.
**Axis:** A2 + A4
**Basis:** `direct:` + `external:` — plan-008 ships 11 arcade recipes but the authoring path starts from a blank `.pforge`; no `templates/` dir exists; fixtures hold only `mario_strip_scene.pforge` and `goomba_scene.pforge`. PuzzleScript and PICO-8 splore prove genre starters drive ~80% of new no-code projects.
**Rationale:** The four reference games exist as the *target* of the ship loop. They should also be the *starting point* of every new user's onboarding. Building Bomberman from a blank project is a graduate exercise; forking the Bomberman template and re-skinning it is a first-five-minutes experience.
**Downsides:** Templates must stay maintained when verb catalog or schema changes (mitigated by S5's CI gating).
**Confidence:** 90%
**Complexity:** Medium
**Status:** Unexplored

### 2. Universal Physics Core + Systematic Sink-Filling via Registry Pattern
**Description:** Implement the 8 stubbed `log*Sink` types in `capsuleruntime/subscribers.go` (Music, Scene, Damage, Motion, Spawn, Visual, Dialogue, Snapshot). Build **one** configurable physics core that all four games preset differently — Asteroids = zero-g + wrap; Bomberman = grid-snap + zero-g; Mario = gravity + platform; DK = gravity + ladders + slopes — not four bespoke physics paths. Apply the proven cycle-break + registry pattern (`PNGImportRunner`/`WAVImportRunner` shape) so each sink is a half-day's work with built-in test scaffolding. Auto-snapshot every entity's component map for save (vs. asking users to declare saveables — which is why `save_now` currently writes empty).
**Axis:** A1
**Basis:** `direct:` — `capsuleruntime/subscribers.go:503–537` defines 8 log-stub sinks; `subscribers.go:651` writes literal empty `Snapshot{}`. The four games are different parameterizations of "rigid body + tile collision + input intent," not different physics families. External help: `solarlune/resolv` (pure-Go 2D collision lib, MIT, proven on Ebitengine).
**Rationale:** Until the sinks land, every authored verb is a log line. This is the gating runtime work; without it the ship loop is theatre.
**Downsides:** Touches engine architecture; physics-core parameter space must be validated against all four games before locking in (S5 gates this); resolv adds a dependency.
**Confidence:** 95%
**Complexity:** High
**Status:** Unexplored

### 3. Pixelforge Commons — Embedded CC0 Asset Library (Kill the 404)
**Description:** Vendor a curated CC0 mega-pack (Kenney space + Kenney bomb-party + an OpenGameArt platformer set + FamiStudio BGM exports) directly into the studio binary via `embed.FS`. The online-library manifest path stays as the *extension* mechanism (CC-BY assets keep flowing through the existing credits path); the default library is in-binary and indexed by named ID. Also wire `dispatcher.SetSpriteRunner(palette.runner)` / `SetSFXRunner(audiolib.runner)` / `SetBGMRunner(audiolib.runner)` in `main.go` so the drag-drop ingest dispatcher (currently silent no-op) actually routes to the editor's existing import handlers. First-run experience: open studio offline, every sprite slot has real art.
**Axis:** A4
**Basis:** `external:` + `direct:` — `assetlibrary.LaunchBackgroundBootstrap` fetches a 404 URL today; plan-008 explicitly defers asset-pack assembly. `main.go:95` creates `ingest.NewDispatcher()` but never connects any runner. Kenney.nl ships ~50MB of CC0 packs perfectly sized for the four named games; `embed.FS` is the Go-native fit.
**Rationale:** A no-code game tool with no art is a text editor. Shipping art in the binary makes the studio self-sufficient and demo-able offline. Eliminates a whole class of "URL is dead" failures.
**Downsides:** Studio binary grows ~50MB (acceptable for the "first-run is real" UX). License hygiene: only CC0 in the mega-pack (CC-BY adjacent assets go through the existing manifest path so credits inject correctly).
**Confidence:** 90%
**Complexity:** Low
**Status:** Unexplored

### 4. Single Pixelforge Player + .pforge Cart (Append-PCK Pattern)
**Description:** Ship **one** signed `pixelforge-player` binary per platform (Windows / macOS / Linux / WASM). A "shipped game" = one .pforge cart + a copy of the player binary with the cart bytes appended (Godot's "Embed PCK", Löve2D "fused" build, Wolfram CDF Player pattern). The codegen template's role shrinks to concatenation; WASM follows the same shape (single .html = player.html + base64-appended cart). One runtime to harden, one cart format to evolve. Could be staged: ship S2 first; once stable, refactor codegen to append-to-player.
**Axis:** A1 + A3
**Basis:** `external:` + `reasoned:` — Godot's Embed PCK, Löve2D fused builds, Wolfram CDF Player, PICO-8 splore all prove the cart + universal player pattern. Today's codegen-per-project model means every shipped game has its own compiled Go runtime; N games × M platforms = NM compile artifacts to test.
**Rationale:** One runtime to harden, one cart format to evolve. The "stubbed sinks" problem only has to be solved once — the player binary becomes the only place runtime logic lives. Reduces ship-test surface area dramatically.
**Downsides:** Larger architectural pivot than S2 alone. The player binary carries every code path every game *could* use (bigger baseline binary, no per-game tree-shake). Mostly orthogonal to S2 in the short term — choose S4 only if you commit to it; otherwise S2's per-game codegen path is fine.
**Confidence:** 75%
**Complexity:** High
**Status:** Unexplored

### 5. Continuous Validation Pipeline (Reference Games as CI Fixtures)
**Description:** Three moves stacked: (a) check in `.pforge` fixtures for **all four** games + a recorded input trace per game (90s of Mario clearing 1-1, an Asteroids sequence destroying all rocks, etc.); (b) collapse the editor's preview renderer and the capsule's shipped renderer to ONE function: `RenderTickAt(.pforge, tick, inputs) → framebuffer`; (c) CI runs each trace headlessly per commit and asserts pixel-hash parity + verbs.bus event sequence parity. The capability matrix becomes a *generated* artifact from CI passes/fails, not human-maintained markdown.
**Axis:** A5
**Basis:** `direct:` + `external:` — `pixelforge_studio/integration_test/fixtures/` contains only `mario_strip_scene.pforge` + `goomba_scene.pforge`; no Asteroids/Bomberman/DK fixture exists; `mario_strip_e2e_test.go` is the only e2e test. No test asserts preview-frame == capsule-frame parity. Factorio's headless test harness + TAS speedrun frame-exact replay prove the determinism approach scales.
**Rationale:** Today the capability matrix can drift from reality and three of four named games have never been built end-to-end. Generated means every PR sees real status, every YELLOW cell has a linked failing test, and "are we done?" becomes a single CI check.
**Downsides:** Recording the trace per game takes hours per game; pixel-hash fragility from font/OS rendering (mitigate by rendering with the engine's embedded font and disabling vsync in test mode).
**Confidence:** 90%
**Complexity:** Medium
**Status:** Unexplored

### 6. WASM HTML Size Cap + Share-Verb Polish
**Description:** Concrete distribution gates: (a) measure baseline (`du -h` each long-tag build output), publish in capability matrix; (b) enforce upper bound (warn at 15MB HTML, error at 30MB unless `--force`); (c) auto-gzip the `.html` alongside the raw (mobile bandwidth halved); (d) bundle [wasm-opt](https://github.com/WebAssembly/binaryen) into the long-tag builder when present (typical 20–40% size cut). The "Build → WASM" success toast shows `mygame.html (8.2MB, gzip 3.1MB)` so the user sees what they're sharing.
**Axis:** A3
**Basis:** `direct:` + `external:` — `pixelforge_studio/buildpipeline/wasm_bundler.go` inlines `.wasm` as base64 with no size budget check or gzip pass. Typical Ebitengine WASM is 10–20MB → 14–28MB HTML uncompressed. Discord file-attach limit is 25MB; the median game-jam audience is on mobile data.
**Rationale:** "Just share the .html" is the *entire pitch* of the WASM ship loop. A 30MB single-HTML file fails the pitch the moment it leaves the dev's machine.
**Downsides:** `wasm-opt` is a C++ binary (external dep); pure-Go alternative awaits TinyGo support for Ebitengine. Gzip + measurement parts ship today with no new deps.
**Confidence:** 95%
**Complexity:** Low
**Status:** Unexplored

### 7. Verb Library Completeness as the Gating Work
**Description:** Three coupled moves: (a) promote `verb_recipes.go` to be the **single source of truth** driving Inspector dropdown / codegen template / auto-generated reference docs / `.pforge` lint / capability matrix rows — adding one verb is one struct edit that propagates everywhere; (b) front-load the library to ~50–80 verbs covering all four games' canonical event sheets (LLM-generate candidates, hand-curate, gated by S5's CI); (c) ship "press key to bind" capture in the input workspace so binding "Space → jump" is one keypress, not six dropdown clicks.
**Axis:** A2
**Basis:** `direct:` + `reasoned:` — 11 arcade recipes shipped in plan-008 U8 cover matrix RED rows, but real authoring needs more: `patrol_until_wall`, `stomp_from_above`, `shoot_toward_player`, `drop_bomb_with_timer`, `ladder_climb_and_jump_off`, `screen_shake_on_hit`, `particle_burst_on_destroy`, `score_multiplier_streak`. `input/workspace.go:127–142` shows keyboard binding via dropdown only.
**Rationale:** "Verb completeness" is what actually gates no-code Mario. Without enough named verbs, users hit the "I need a behavior that doesn't exist" wall and abandon. Treating the catalog as the bottleneck and front-loading it is the highest-leverage authoring move.
**Downsides:** Verb sprawl risk (mitigated by S5's CI gating — verbs unused by any reference-game fixture get flagged); requires curating LLM-generated candidates rather than blind-accepting.
**Confidence:** 80%
**Complexity:** Medium
**Status:** Unexplored

## Rejection Summary

| # | Idea | Reason Rejected |
|---|------|-----------------|
| F1#8 | Collision-mask editor | Auto-derive from alpha (already in `import_pipeline.go`) covers most cases; manual editor is niche enough to defer until users actually request it |
| F2#3 | Continuous background build (kill the Build button) | Disruptive UI inversion vs. existing two-button shipped surface; the substance lives in S6 (always-fresh local) |
| F2#4 / F5#1 | Steganographic `.pforge.png` cart | Cute but expensive novelty; S4's append-PCK pattern serves "share cart" without the steganography complexity |
| F3#3 | Author-by-play (infer recipes from playback diff) | Research-grade; inference of recipes from state-diff is unproven; high risk of becoming a research project rather than a product feature |
| F3#5 | Split studio (headless authoring server + thin viewport) | Too expensive for too little value; the save-empty bug it would fix is fixed cheaper by auto-snapshot in S2 |
| F3#6 | Asset library as CAS mirror | Too complex vs. S3's embedded mega-pack; content-addressable storage adds infra concerns for marginal win |
| F3#8 / F6#1 | Browser-first / webview-wrapped host | Too disruptive given plan-008's native path just shipped; throwing away the Ebitengine native investment is a high-cost pivot |
| F4#4 / F5#7 | Unified export / Twine publish-to-file | Already covered by plan-008's two-button UI + `builders_long.go` |
| F5#3 | Tracker UI for scene authoring | Scope overrun — adds a new authoring paradigm alongside event-sheets / Step lanes; "node graphs rejected" simplicity was hard-won, don't add a third paradigm |
| F5#6 / F6#2 | Postcard / recipe-card project format | Format change is a major project; current JSON `.pforge` works; diffability comes from cleaner JSON, not new format |
| F6#3 | Editor embedded in every game | Too expensive (binary bloat); "share `.pforge` and studio opens it" is the simpler version |
| F6#5 | Phone-first authoring | Scope overrun — mobile UI is its own product; not in scope for the four named games right now |
| F6#6 | Generate four reference games from prompts | Fragile for canonical fixtures; the four games must be deterministic test inputs, not LLM-regenerated |
| F6#7 | Cloud build farm (full) | Infra cost not justified vs. existing local builders; the local-incremental piece is absorbed into S6 |

## Minimum Credible Path

The minimum credible path from plan-008's "loop is end-to-end shippable but mostly empty inside" to "ships playable games":

1. **S2** (runtime sinks + universal physics core) — gating runtime work
2. **S3** (embedded CC0 library + ingest runner wiring) — gating content work
3. **S5** (CI fixtures for all four games + render-seam collapse) — gating proof work
4. **S1** (genre templates) — gating authoring work

S6 + S7 are next polish wave once the four games are demonstrably playable. S4 is the architectural alternative to consider once S2 has stabilized — it's not strictly necessary for the stated goal, but locks in long-term simplicity.
