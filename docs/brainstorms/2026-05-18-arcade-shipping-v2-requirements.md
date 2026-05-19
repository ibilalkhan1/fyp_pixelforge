---
date: 2026-05-18
topic: arcade-shipping-v2
---

# Arcade Shipping v2 — Universal Player + Cart-Append + Reference-Game Proof

## Summary

Plan-009 lands the universal-player + cart-append architecture so any user-authored game ships as one self-contained executable (host system or browser), fills the eight log-stub runtime sinks inside the player binary, and proves the architecture works by getting the four named reference games (Asteroids, Mario, Bomberman, Donkey Kong) playable end-to-end as CI fixtures. The engine and studio remain general-purpose; the four games are proof, not product.

---

## Problem Frame

Plan-008 shipped the scaffolding: capsule runtime loaders, `verbs.bus` typed pub/sub, real `go build -tags=long`, icon rasterization, WASM save backend, asset-library manifest substrate, custom-ingest watcher, Library workspace, 11 arcade recipes. None of that landed an actually-playable user-authored game.

Today the loop is honest about its substrate-ness: a user authors a Mario clone, clicks Build, gets a runnable `.exe`, presses Space — and the binary logs `[capsuleruntime] motion/jump map[strength:5]` to stderr without moving the sprite. Eight of eleven verb-recipe sinks are `log*Sink` stubs (`pixelforge_studio/capsuleruntime/subscribers.go:503–537`); `save_now` writes `pisave.Snapshot{}` literally; the asset-library manifest URL 404s; the ingest dispatcher has no runners wired (`pixelforge_studio/main.go:95`); three of four named games have no fixture in `pixelforge_studio/integration_test/fixtures/` (only `mario_strip_scene.pforge` and `goomba_scene.pforge` exist).

The cost shape is dishonesty: every "Build" produces an artifact, the artifact runs, the runtime does nothing recognizable as gameplay. A user encountering this would correctly conclude the engine is theatre — and we would have shipped substrate without ever proving the substrate connects to anything.

---

## Actors

- A1. **End user (game author):** Non-coder using the studio's GUI to author + ship a game. May start from a generic blank-genre template, may fork a reference example, may import their own assets via drag-drop or watched folder. Wants the studio to behave like a tool, not a tutorial.
- A2. **Player (game consumer):** Receives a single binary or `.html` from an end user; runs it; expects gameplay. Has no Pixelforge installation; doesn't know what Pixelforge is; never opens a config file.
- A3. **Pixelforge engineer:** Maintains the player runtime + studio; consumes the four named-game CI fixtures + traces as the proof that runtime changes haven't regressed.

---

## Key Flows

- F1. **First-author-ship loop (new user, blank platformer)**
  - **Trigger:** A1 installs `pixelforge-studio`, opens it, clicks `File → New → Blank Platformer`.
  - **Actors:** A1, A2.
  - **Steps:** (1) studio creates a project pre-wired with a platformer physics preset, auto-derive collision from sprite alpha, WASD+Space input bindings, auto-snapshot save mode, and one starter sprite from the embedded CC0 starter set; (2) A1 presses Play; the in-studio runtime plays the scene with a controllable character that jumps and lands on a placeholder platform; (3) A1 adds a sprite via drag-drop into the project; (4) A1 authors a verb sheet via the existing inspector; (5) A1 clicks `Build → WASM`; studio produces one `.html` under the size threshold; (6) A1 shares the `.html` with A2 via any channel.
  - **Outcome:** A2 double-clicks the `.html`, browser opens, splash → click-to-start → game plays. A2 never installs Pixelforge.
  - **Covered by:** R1, R2, R3, R7, R10, R14, R17.

- F2. **Reference-game study-and-fork (user wants to see how Mario was built)**
  - **Trigger:** A1 clicks `File → Open Example → Mario`.
  - **Actors:** A1.
  - **Steps:** (1) studio fetches the Mario reference example on first request (cached on subsequent); (2) the `.pforge` opens; (3) A1 can press Play to run Mario in-studio, edit any sprite/level/verb, fork into a new project, or just read the verb sheets to understand the technique.
  - **Outcome:** A1 understands how a complete Mario was authored without reading prose docs.
  - **Covered by:** R11, R16.

- F3. **Engineer regression check (post-commit CI)**
  - **Trigger:** A3 commits any change to the player runtime, codegen, verb catalog, or physics core.
  - **Actors:** A3.
  - **Steps:** (1) CI runs the long-tag test suite; (2) for each of the four reference-game fixtures, CI loads the `.pforge`, replays the recorded input trace tick-by-tick through the shared render function, asserts pixel-hash matches the baseline AND `verbs.bus` event sequence matches; (3) the capability matrix is regenerated from CI pass/fail status; (4) any divergence fails the build.
  - **Outcome:** Player-runtime regressions surface in CI before they ship; the "are the four games still playable?" question is answered by inspecting the latest CI run.
  - **Covered by:** R8, R9, R12, R13.

---

## Requirements

**Universal player + ship architecture**

- R1. The engine ships two binaries built from one Go codebase: `pixelforge-studio` (editor + embedded player runtime for in-studio Play) and `pixelforge-player` (stripped runtime that loads + plays `.pforge` carts). The studio's in-studio Play call path and the shipped `pixelforge-player` runtime invoke the same code — no parallel runtime implementations.
- R2. Build → Host produces one self-contained native binary per supported host OS (Windows / macOS / Linux). The shipped binary = a copy of `pixelforge-player` for that platform with the project's `.pforge` cart appended; the runtime detects + reads the appended cart at startup. No external assets, no engine install needed on A2's machine.
- R3. Build → WASM produces one self-contained `.html` file. The shipped `.html` = `pixelforge-player.wasm` + base64-encoded `.pforge` cart + the click-to-start splash + favicon, all inline. `file://` and offline play both work; no external script / style / asset references.
- R4. Cross-OS native builds remain rejected at orchestrator preflight with `ErrCrossCompileNotSupported` (no regression from plan-008 U4).

**Runtime subsystems (filling plan-008's log-stubs)**

- R5. The player binary implements concrete sinks for: scene control (change / restart / wait); entity motion (move-with-intent, jump, apply-gravity, apply-thrust, rotate-entity, screen-wrap, place-on-grid, ladder-climb, barrel-roll, move-pattern, bounce, teleport-to); tile collision (solid-collide); spawn / destroy (entity spawn, destroy-self, destroy-other); visual state (hide, show, flash, swap-sprite); damage (die, hurt-player, take-damage, explode-radius); BGM streaming (play-music, stop-music); dialogue rendering (open-dialogue, close-dialogue); snapshot save / load (save-now captures every entity's component state automatically; load-slot restores it).
- R6. The player's physics is a single configurable subsystem with per-cart parameters (gravity vector, drag, collision-response mode, movement constraint, ladder-aware flag, screen-wrap-enabled flag). The four reference games configure this subsystem differently; the player does not contain four separate physics implementations.
- R7. Save snapshots capture every entity's component map automatically; the user does not author per-game save logic. Snapshots round-trip byte-equal across save → load on both native and WASM.

**Validation (CI as proof, not manual demos)**

- R8. The repo carries `.pforge` fixtures for all four named games (`asteroids_proof.pforge`, `mario_proof.pforge`, `bomberman_proof.pforge`, `donkey_kong_proof.pforge`) plus a recorded input trace per game (~90s) that exercises each game's core mechanics. Concretely: Asteroids destroys all rocks; Mario clears 1-1; Bomberman places a bomb + survives the blast + clears the screen; DK climbs ladders + dodges barrels + reaches the top.
- R9. CI runs the long-tag test suite per commit; each fixture's trace replays through the player runtime; pixel-hash and `verbs.bus` event-sequence parity must both match the baseline. Divergence fails the build.
- R12. The editor's preview renderer and the shipped player's renderer share one function `RenderTickAt(.pforge, tick, inputs) → framebuffer`. CI replays through this function; in-studio Play invokes this function; the shipped player binary invokes this function. No parallel rendering implementations.
- R13. The capability matrix doc (`docs/reference-games-capability-matrix.md`) is regenerated from CI pass / fail per fixture per recipe, not human-maintained markdown. Regenerated on every CI run.

**Content + asset library**

- R14. The studio binary embeds a small generic CC0 starter asset set (a handful of placeholder sprites + a couple of SFX/BGM samples) via `embed.FS`. First-launch is non-empty offline.
- R15. The plan-008 asset-library manifest + downloader substrate fetches from a working URL. The `manifest.json` at the configured URL is published as part of v2; downloads work; the Library workspace renders curated packs after bootstrap completes.
- R16. The four named games' assets ship as part of the curated library packs (downloadable on demand by F2's reference-example flow). They are not embedded in the studio binary; the studio binary stays game-agnostic.
- R17. The ingest dispatcher's `SetSpriteRunner` / `SetSFXRunner` / `SetBGMRunner` are wired in `main.go` to the editor's existing PNG / WAV import handlers; new `OGGImportRunner` and `MP3ImportRunner` interfaces are declared and wired for BGM. Drag-drop a `.png` / `.wav` / `.ogg` / `.mp3` onto the studio window: the file flows into the active project without further action.

**Authoring UX**

- R10. The studio's `File → New` menu offers generic genre starter templates: "Blank Platformer," "Blank Arcade Shooter," "Blank Grid Game," "Blank Ladder Platformer." Selecting one yields a runnable but content-empty project with the right physics preset, input bindings, save mode, and one placeholder sprite + scene layout pre-wired. The four named games are NOT presented as project-creation defaults.
- R11. The studio's `File → Open Example` menu offers the four named reference games as opt-in study material the user can open, run, edit, or fork. Reference examples are fetched on demand from a GitHub Release and cached locally; not embedded in the studio binary.
- R18. The verb-recipe catalog is the single source of truth driving: Inspector dropdown options, codegen template inputs, auto-generated reference docs (`docs/verb-catalog.md`), `.pforge` lint, and capability matrix rows. Adding one verb is one struct edit; all consumers update from the catalog.
- R19. The verb catalog covers all four named games' canonical event sheets — empirically determined by what the four games' fixture traces require to pass. Target size approximately 50–80 verbs. Verbs unused by any reference-game fixture trace are flagged for review (the gating signal against catalog sprawl).
- R20. The input-binding workspace supports "press key to bind" capture in addition to the existing dropdown. Binding "Space → jump" is one keypress, not six dropdown clicks.

**Ship-loop distribution**

- R21. The WASM bundler reports the produced HTML size in the success toast (raw + gzip). Build → WASM warns when the raw HTML exceeds 15MB and errors when it exceeds 30MB unless the user passes a `--force` flag.
- R22. The WASM bundler writes both `mygame.html` and `mygame.html.gz` (gzip alongside raw). The `wasm-opt` invocation runs when the binary is present on the build host; absence is silently skipped, not an error.

---

## Acceptance Examples

- AE1. **Covers R1, R2.** Given a project created from the `Blank Platformer` template, when A1 clicks `Build → Host` on macOS, the studio produces one `.app` bundle. When A2 runs the `.app` on a different macOS machine with no Pixelforge installation, the game launches and the player sprite jumps in response to Space.
- AE2. **Covers R3.** Given the same project, when A1 clicks `Build → WASM`, the studio produces one `mygame.html`. When A2 opens `mygame.html` via `file://` in a browser with no network, the click-to-start splash appears; on click the game plays.
- AE3. **Covers R5, R6.** Given the Mario reference example loaded into the in-studio Play, when A1 presses Space, the player sprite jumps with the gravity preset's curve and lands on the solid-tile platform below. No `log.Printf` motion stub fires.
- AE4. **Covers R7.** Given a player binary running the Donkey Kong reference cart, when A1 fires `save_now`, presses Reset, then fires `load_slot`, the entity state (player position, ladder-climb state, barrel positions, lives count) restores byte-equal.
- AE5. **Covers R9, R12.** Given a one-pixel-off regression in the player's render path, when CI runs the four-game fixture replay, the affected game's pixel-hash comparison fails and the build is marked red.
- AE6. **Covers R14, R15.** Given a fresh studio install with no network, when A1 opens it, the asset-library workspace renders the embedded starter assets (non-empty). When A1 later reconnects and the bootstrap completes, the workspace renders the downloaded curated packs alongside the starter set.
- AE7. **Covers R17.** Given a `.png` file dragged onto the studio window, when the dispatcher classifies it as a sprite, the editor's import handler is invoked and the file appears in the project's sprite picker without further action.
- AE8. **Covers R20.** Given A1 opens the input-binding workspace and clicks "Capture" next to the `jump` intent, when A1 presses Space, the binding updates to `Space` and the workspace shows the new mapping.
- AE9. **Covers R21.** Given a build produces a `mygame.html` of 18MB raw, the success toast displays `mygame.html (18.0MB, gzip 6.4MB)` and a warning that the size exceeds the 15MB threshold.
- AE10. **Covers R10, R16.** Given A1 clicks `File → New → Blank Platformer`, the resulting project contains a generic placeholder sprite (not Mario art). The four named games' assets are not present locally until A1 explicitly opens the Mario / Asteroids / Bomberman / DK reference example or downloads the curated library pack.

---

## Success Criteria

- A user with no Go installed can install pixelforge-studio, open it, choose `File → New → Blank Platformer`, press Play, and see a controllable jumping character in under 5 minutes. They produce a runnable game share-able as one binary or one `.html` within their first hour.
- A reviewer of v2's PRs can answer "is Donkey Kong still playable?" by inspecting the latest CI run; no manual demo required.
- A shipped player binary or `.html` runs on any compatible OS / browser with no Pixelforge installation present on the player's machine.
- All eight currently-stubbed `log*Sink` types in `pixelforge_studio/capsuleruntime/subscribers.go` are concrete implementations; no `log*Sink` type remains in the player binary at v2 release.
- `ce-plan` can produce plan-009 without inventing runtime subsystem scope, the universal-player architecture shape, the per-game proof criteria, or the asset-library content surface — those are all settled here.

---

## Scope Boundaries

- The studio binary does not bundle the four named games as built-in content. Reference examples are fetched on demand from a GitHub Release (R11, R16).
- The studio binary does not embed a large curated CC0 mega-pack. Lighter generic starter embed (R14) plus the working online manifest (R15) replaces it.
- Mac code-signing + notarization; Windows code signing; Linux `.desktop` / AppImage / Flatpak packaging — explicitly out of v2.
- Headless browser smoke test for WASM (wasmbrowsertest / chromedp) — not gated by v2.
- Author-by-play (record gameplay → infer recipes); split studio (headless authoring server + thin viewport); content-addressable asset mirror; browser-first studio; editor-embedded-in-every-shipped-game; phone-first authoring; tracker UI for scene authoring; postcard / recipe-card project format change; LLM-generated reference games; cloud build farm; steganographic `.pforge.png` carts — all out of v2 per the prior ideation's rejection summary.
- Reference games beyond the four named (Pac-Man, Frogger, Space Invaders, Tetris) — v3 matrix expansion.
- In-library search, tag-based browsing, community-submission flow, cloud save sync, multiplayer / networking, AI / procedural-generation, Steam / itch.io publishing integrations, iOS / Android targets — all explicitly out.

---

## Key Decisions

- **Universal player + cart-append over per-project codegen.** One runtime to harden, one cart format to evolve, four games proven by one player binary. Per-project codegen had N games × M platforms = NM artifacts to test. Bigger up-front pivot, lower long-term carrying cost.
- **Studio + player are two binaries built from one codebase.** Studio's in-studio Play uses the same code as the shipped player; no parallel implementations; preview-vs-shipped drift is structural-impossible (R12).
- **Engine + studio stay game-agnostic.** The four named games are CI proof + opt-in reference examples, not bundled content. Studio binary stays general-purpose; users build whatever they want.
- **Generic genre starters as project-creation defaults; named reference games as opt-in study material.** Avoids the "studio = the four games" framing while preserving the proof value.
- **Capability matrix is generated from CI, not human-maintained markdown.** Eliminates drift; "is the engine ready?" is answered by inspecting the latest CI run.
- **One configurable physics core, four parameter presets.** Not four bespoke physics implementations. Avoids N×M maintenance.
- **External libraries explicitly OK** (resolv for collision, Kenney CC0 packs for assets, wasm-opt for size, etc.) provided they don't break plan-008 invariants: cycle-break injection pattern, ImGui-only studio chrome, schema additivity discipline, engine `CGO_ENABLED=0` for WASM, no string-concat source generation.
- **Phased delivery inside v2.** Phase 1 = universal player + cart-append + Asteroids end-to-end. Subsequent phases extend the player with each next game's physics + run the new fixture's CI trace. Each phase is independently demoable; the architectural bet pays off at Phase 1, not at Phase 7.

---

## Dependencies / Assumptions

- A working GitHub Release exists at the asset-library URL once v2 ships (the URL plan-008's downloader points at, which 404s today). Publishing the release is part of v2's content workstream.
- A working GitHub Release for reference examples (`asteroids.pforge`, `mario.pforge`, `bomberman.pforge`, `donkey_kong.pforge`) exists once v2 ships, fed by the four CI fixtures with their input traces stripped.
- The Go toolchain is available on the developer's machine for the long-tag build path (vendored SDK fallback handles installer cases — same as plan-008 U3).
- `wasm-opt` (from WebAssembly Binaryen) is optionally installed on the build host for size reduction; absence is acceptable, just skips the optimization.
- A 2D collision library (likely `solarlune/resolv` or equivalent pure-Go AABB lib) covers R5's tile-collision needs; evaluated during planning per the Leverage Doctrine, not pre-committed here.
- Ebitengine 2.9.9's existing API surfaces for input, audio, rendering, and `DroppedFiles()` cover the player runtime's needs; no engine fork required.

---

## Outstanding Questions

### Resolve Before Planning

None — synthesis confirmed by user; v2 scope is settled.

### Deferred to Planning

- [Affects R5][Technical] Should the player runtime use `solarlune/resolv` for 2D collision or build AABB resolution into a new `pixelforge_physics/` package from scratch? Evaluate during planning per the Leverage Doctrine.
- [Affects R8, R12][Technical] Exact format for the recorded input trace files (JSON `{tick, inputs}` per frame, or a tighter binary format). Decide during planning to balance human-readability vs. file size.
- [Affects R10, R11][Technical] Exact mechanism for `File → Open Example` to fetch reference cards — HTTP GET against the asset-library URL, or a dedicated examples manifest endpoint.
- [Affects R3, R22][Needs research] Empirical baseline WASM HTML size for a non-trivial Pixelforge game once R5's runtime sinks are real. Drives whether the 15MB warn / 30MB error thresholds need adjustment.
- [Affects R18][Technical] Whether the auto-generated `docs/verb-catalog.md` is regenerated via `go generate`, a make target, or a CI artifact.
- [Affects R1, R2, R3][Technical] Exact mechanism for the player binary to detect + read its appended `.pforge` cart at startup — magic byte sequence, length-prefixed footer, or platform-specific binary patching (Go's `os.Executable()` + read-self pattern is the likely shape; planning verifies cross-platform behavior).
