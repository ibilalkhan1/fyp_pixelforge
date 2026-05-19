---
title: "feat: Arcade-shipping v1 — ship-loop wiring, recipe gap-fill, asset library (post-007 follow-up)"
type: feat
status: completed
date: 2026-05-18
depth: deep
origin: solo (no upstream brainstorm; follow-up to plan-007 deferred items + new asset-library scope)
satisfies_dependencies:
  - docs/plans/2026-05-18-007-feat-project-capsule-build-pipeline-v1-plan.md (closes deferred Capsule runtime loaders, real go build, per-platform icon, WASM save backend)
  - docs/plans/2026-05-18-001-feat-entity-verb-sheet-v1-plan.md (wires verb-recipe topic surface into actual subsystems)
ships_with:
  - all previous milestone plans (closes the end-to-end no-code ship loop)
---

# feat: Arcade-shipping v1

## Summary

Close the end-to-end no-code ship loop so a user can author Asteroids, Bomberman, Mario, or Donkey Kong fully in the studio GUI (every level, sprite, sound, BGM, menu) and click **Build → Host** or **Build → WASM** to get a runnable single-file executable. Three tracks land in one batch: (A) finish the runtime + build wiring deferred by plan-007 — Capsule loaders into engine registries, `verbs.bus` event target + subscribers for the 32 published recipe topics, real `go build` behind `//go:build long`, two-button Host/WASM build UI, per-platform icon emission rasterized from `docs/logo.svg`, and the WASM save backend; (B) audit the verb-recipe catalog against the four reference games and add the small set of missing arcade primitives (thrust, rotate, screen-wrap, jump, gravity, solid-collide, place-on-grid, explode-radius, ladder-climb, barrel-roll, fixed-tick-loop); (C) ship a curated CC0 + CC-BY asset library that the studio downloads to `os.UserCacheDir()` on first launch, with watched-folder + drag-drop custom-asset ingest, an attribution screen in the studio, an auto-injected credits page in exported games, and a Library workspace with per-game tabs.

---

## Problem Frame

Plan 007 landed the Capsule + build pipeline substrate, but multiple critical pieces were deferred:

- `CapsuleRun` loads the embedded project then drops it on the floor — the `_ = capsuleAssetsFS` discard line in `pixelforge_studio/codegen/templates.go` is literally where the loader hooks belong but never landed.
- Per-target builders write placeholder marker files instead of invoking `go build` (the `//go:build long` shim file does not exist; `pixelforge_studio/buildpipeline/builders.go` registers a single `scaffoldBuilder` for all five targets).
- Per-platform icon generation returns `errIconUnsupported` from `GenerateWindowsIcoStub` / `GenerateMacIcnsStub`; `IconResult.FaviconBase64` is populated only with a deterministic hash-color fallback.
- The verb-recipe catalog publishes to a `Target[string]` bus, but `pixelforge_loop/piloop.go` registers `loop.main` as `Target[Event]` (typed enum) — every `publish_event` action would fail its type assertion at runtime today, and no subsystem subscribes regardless.
- `pixelforge_save/backend_native.go` is gated `//go:build !js` and no `backend_js.go` sibling exists; every `save_now` recipe call on a WASM build would crash.

Meanwhile, the verb catalog covers RPG primitives well but lacks arcade essentials — no `screen_wrap`, no `jump`/`apply_gravity`, no `place_on_grid` / `explode_radius`, no `ladder_climb`, no `barrel_roll`. The four named reference games are unbuildable today even if the ship loop worked end-to-end.

And the studio ships zero curated art/audio: every project starts empty, contradicting the "no-code complete game" promise the four reference games depend on. The user must manually source every sprite + sound for every project.

This plan closes all three gaps in one batch so the loop is provably end-to-end: open studio → pick the Bomberman starter pack → drag entities onto a level → bind verb sheets → click Build → ship a runnable `.exe` (or self-contained `.html`).

---

## Output Structure

(New files only — modified files listed per-unit.)

```
docs/
  reference-games-capability-matrix.md         (U7)

pixelforge_audio/
  registry.go                                  (U1)

pixelforge_dialogue/
  registry.go                                  (U1)

pixelforge_save/
  backend_js.go                                (U6)

pixelforge_studio/
  assetlibrary/
    manifest.go                                (U9)
    downloader.go                              (U9)
    pack.go                                    (U9)
    library.go                                 (U12)
    workspace.go                               (U12)
    preview.go                                 (U12)
    credits.go                                 (U10)
  buildpipeline/
    builders_long.go                           (U3)
    icon_logo.go                               (U5)
  capsuleruntime/
    loaders.go                                 (U1)
    runtime.go                                 (U1, U2)
    registry.go                                (U1)
    subscribers.go                             (U2)
  ingest/
    watcher.go                                 (U11)
    dragdrop.go                                (U11)
    classifier.go                              (U11)
  scripting/catalog/
    builtin_arcade.go                          (U8)
    builtin_arcade_test.go                     (U8)
```

---

## Key Technical Decisions

1. **`pixelforge_studio/capsuleruntime/` sub-package owns Capsule wiring.** The codegen capsule template stays small — `CapsuleRun` calls `capsuleruntime.Boot(project, assets, opts)` which loads everything and installs subscribers, then returns. Lets us test runtime wiring without going through `go generate`. The existing `_ = capsuleAssetsFS` discard line in `templates.go` is the literal seam.

2. **Dedicated string event-bus target `verbs.bus` for recipes, not retarget `loop.main`.** Research surfaced that `loop.main` is `pievent.Target[Event]` (typed enum) but `buildPublishEvent` does a `Target[string]` assertion that would fail. Registering a new `Target[string]` under `"verbs.bus"` and changing the catalog's `EventBusTarget` constant from `"loop.main"` → `"verbs.bus"` is the lowest-risk fix — keeps the typed-enum loop bus pristine; preserves every existing `loop.main` consumer.

3. **Subscribers live capsule-side, not engine-side.** Engine packages stay unaware of the verb-recipe topic surface. Same cycle-break shape as `PNGImportRunner` / `WAVImportRunner` from plan-005 — engine exposes registries, capsule wires subscribers that read payloads and call into those registries.

4. **Real `go build` lands at `pixelforge_studio/buildpipeline/builders_long.go` behind `//go:build long`.** Precedent: `pixelforge_studio/codegen/generator_long_test.go`. The shim calls `NewBuildCommand(ctx, target, "build", "-o", outPath, ".")` in the generated capsule's outDir. The current scaffold builder gets a `//go:build !long` sibling so the long-tag build cleanly supersedes it. Studio's Build button auto-applies `-tags=long` so users don't need to know about the tag.

5. **Build UI exposes Host + WASM only.** Two large buttons in `pixelforge_studio/build/workspace.go` replace the 5-checkbox matrix. Orchestrator preflight rejects any non-host native target via a new `buildpipeline.CanBuildOnHost(t Target) bool` helper that codifies `t == HostTarget() || t == TargetWASM || t == TargetSource`. The `Source` target stays callable via API for tests but is not surfaced in the UI.

6. **Icon source: rasterize `docs/logo.svg` at build time, one icon for every exported game.** Pure-Go SVG raster via `oksvg` + `rasterx` (no cgo, no external tools). Pack into `.ico` for Windows, `.icns` for macOS, base64 32×32 PNG for WASM favicon. Windows additionally emits a `.syso` via `goversioninfo` so the Windows linker picks up the icon automatically. Three new deps justified — pure-Go, one-time cost. No per-project icon plumbing.

7. **Asset library: fetch-on-first-launch from a GitHub Release; user packs in a watched sibling folder.** Manifest hosted at `https://github.com/ibilalkhan1/fyp_pixelforge/releases/download/asset-library-v1/manifest.json` (overridable via `PIXELFORGE_ASSET_LIBRARY_URL` env var for local dev). Downloader writes packs to `os.UserCacheDir()/pixelforge/library/<pack-id>/` with SHA-256 verification + atomic writes. User-library sibling at `os.UserCacheDir()/pixelforge/user-library/` watched via `fsnotify`. Schema stays additive per `docs/solutions/editor-pforge-schema-shape.md` (no fields removed; sanitize on load).

8. **License posture: CC0 + CC-BY with attribution screen + auto-injected credits page.** Manifest entry has `license` / `author` / `source_url`. Studio shows attribution in the Library workspace below each asset. Exported games auto-inject a "Credits" menu entry — for native it's an engine scene that the menu system opens; for WASM it's an additional `<div>` accessible from the click-to-start splash. CC0 assets are omitted from credits (no attribution required) to keep the page focused.

9. **One curated content artifact (the asset packs themselves) is deferred to a follow-up content workstream.** This plan ships the manifest schema, downloader, library workspace, ingest, credits UX, and a per-game-tabbed picker — the substrate. Sourcing/verifying CC0+CC-BY assets and publishing them as a GitHub Release is content work, not code; tests use synthetic manifests pointing at fixture URLs.

---

## Implementation Units

### U1. Capsule runtime loaders

**Goal:** Make `CapsuleRun` actually wire the loaded project into the engine. Sprites populate a name-keyed registry the entity renderer reads from. Audio samples decode from `capsuleAssetsFS` and register by name. Dialogue scripts parse and register by name. Menu templates already register via `pixelforge_menus`. Items + scenes go into capsule-side registries the subscriber layer (U2) reads.

**Requirements:** Enables every other unit. Closes the central gap from plan-007 ("Capsule runtime loaders for sprites/audio/dialogue not landed").

**Dependencies:** None.

**Files:**
- Create: `pixelforge_studio/capsuleruntime/loaders.go`
- Create: `pixelforge_studio/capsuleruntime/runtime.go`
- Create: `pixelforge_studio/capsuleruntime/registry.go`
- Create: `pixelforge_audio/registry.go` (new public `RegisterSample(name, *Sample)` + `LookupSample(name) *Sample` + `ResetForTest`)
- Create: `pixelforge_dialogue/registry.go` (new public `RegisterScript(name, *Tree)` + `LookupScript(name) *Tree` + `ResetForTest`)
- Create: `pixelforge_studio/capsuleruntime/loaders_test.go`
- Modify: `pixelforge_studio/codegen/templates.go` — capsule template imports `pixelforge_studio/capsuleruntime`; `CapsuleRun` calls `capsuleruntime.Boot(p, capsuleAssetsFS, opts)` between project-load and `pixelforge_ebiten.Run()`
- Modify: `pixelforge_studio/codegen/generator.go` — import path threaded into the rendered template

**Approach:** `Boot(project, assets, opts) error` walks the project. For each `SpriteAsset` it reads `assets/sprites/<rel>.png`, decodes via the existing engine image loader, stores in the sprite-name registry. For each `AudioSample` it reads `assets/audio/<rel>.wav`, calls `pixelforge_audio.DecodeWav`, registers under the sample's name. For each `DialogueScript` it parses and registers. For items + scenes, populates capsule-side registries used by subscribers (U2). All registries follow the `sync.RWMutex + map + Register/Lookup/All + ResetForTest` pattern from `pixelforge_menus/registry.go`. Generated capsule's `package main` keeps the `//go:embed all:assets` directive — `embed.FS` passed to `Boot` directly. First decode error is returned wrapped with the offending asset's path so failures are diagnosable.

**Patterns to follow:** `pixelforge_menus/registry.go` (registry shape). `docs/solutions/editor-pforge-schema-shape.md` (additive schema discipline). `docs/solutions/scripting-runtime-design.md` (registries are pure data, no engine spin-up to test).

**Test scenarios:**
- *Happy path:* Boot decodes 3 sprite assets from a synthetic `embed.FS`; sprite registry's `All()` returns all 3 by name.
- *Happy path:* Boot decodes 2 WAV samples; `pixelforge_audio.LookupSample("blast")` returns a non-nil `*Sample`.
- *Happy path:* Boot parses a valid dialogue script and `pixelforge_dialogue.LookupScript("intro")` returns the parsed Tree.
- *Edge:* Empty project (zero sprites/audio/dialogue) — Boot returns nil without error.
- *Error path:* Sprite PNG bytes are malformed — Boot returns error wrapping the asset's `RelativePath`.
- *Error path:* Audio WAV bytes are not RIFF — Boot returns error naming the asset.
- *Edge:* `Boot` is idempotent — calling twice on the same project + assets produces equivalent registry state (each `Register*` call replaces the prior entry).
- *Integration:* Generated capsule integration test — running `Generate` on a minimal project produces a `capsule.go` that compiles AND its `CapsuleRun` (run with a new `--smoke` opt) reports "boot complete" without crashing.

**Verification:** A generated capsule's `CapsuleRun` populates all registries before reaching `pixelforge_ebiten.Run()`. Registry sizes match project asset counts.

---

### U2. Event-bus wiring — `verbs.bus` target + recipe subscribers

**Goal:** Make verb recipes actually drive subsystems. Today `publish_event` publishes to a target nothing subscribes to AND the type assertion fails at runtime. Fix both: register a dedicated `pievent.Target[string]` under `"verbs.bus"`, change the catalog's `EventBusTarget` constant to point at it, install subscribers for every published topic.

**Requirements:** Recipes become functional. Without this, every "play sound", "open dialogue", "give item", "change scene", "save now" verb is silently dropped.

**Dependencies:** U1 (subscribers need populated registries).

**Files:**
- Create: `pixelforge_studio/capsuleruntime/subscribers.go`
- Create: `pixelforge_studio/capsuleruntime/subscribers_test.go`
- Modify: `pixelforge_loop/piloop.go` — `init()` registers a new `pievent.Target[string]` under name `"verbs.bus"`; the existing `loop.main` typed-enum target stays untouched
- Modify: `pixelforge_studio/scripting/catalog/verb_recipes.go` — `EventBusTarget = "verbs.bus"` (was `"loop.main"`)
- Modify: `pixelforge_studio/scripting/catalog/builtin_actions.go` — verify the `pievent.Target[string]` assertion path now succeeds; no logic change expected

**Approach:** `InstallSubscribers(rt *Runtime)` in `subscribers.go` looks up `verbs.bus` via `pievent.LookupTarget`, type-asserts to `Target[string]`, then `SubscribeAll`s. Each handler is a small function that decodes the payload and calls into its subsystem's public API:

- `audio/play_sound` → `pixelforge_audio.LookupSample(name).Play(channel)`
- `audio/play_music`, `audio/stop_music` → BGM manager (capsule-side)
- `scene/change` → capsule scene controller `Activate(scene_id)`
- `scene/restart` → scene controller `Restart()`
- `scene/wait` → tick-counter coroutine
- `ui/open_dialogue` → `pixelforge_dialogue.LookupScript(name)` → `TextBoxRenderer.Open`
- `ui/close_dialogue` → renderer's `Close()`
- `ui/open_menu` / `ui/close_menu` → `pixelforge_menus` stack push/pop
- `inventory/give_item` / `take_item` / `set_item_count` → capsule inventory
- `damage/die` / `hurt_player` / `take_damage` → entity health system
- `motion/move_pattern` / `bounce` / `teleport_to` / `move_with_intent` → entity motion
- `spawn/entity` / `destroy_self` / `destroy_other` → entity manager
- `visual/hide` / `show` / `flash` / `swap_sprite` → entity render state
- `save/now` / `save/load_slot` / `save/delete_slot` → `pixelforge_save.Service`

Payload is the recipe-action arg map (string→any). Each handler tolerates missing or wrong-type fields by no-oping with a debug log — never crashes a shipped game. Test each handler by publishing the topic against a fake subsystem injected into the Runtime constructor.

**Patterns to follow:** `docs/solutions/ring-buffer-snapshot-store.md` (`SubscribeAll` convention). `docs/solutions/scripting-runtime-design.md` (pure-function handlers tested without engine spin-up). `docs/solutions/dirty-state-ux.md` (single-seam discipline — one `InstallSubscribers` call from `Boot`).

**Test scenarios:**
- *Happy path:* Publishing `audio/play_sound` with payload `{name: "blast"}` calls `Play()` on a fake audio backend with that sample.
- *Happy path:* Publishing `scene/change` with `{scene_id: "level_2"}` calls `Activate("level_2")` on a fake scene controller.
- *Happy path:* Publishing `inventory/give_item` with `{item_id: "key", count: 1}` adds to a fake inventory.
- *Happy path:* Publishing `ui/open_dialogue` with `{script_id: "intro"}` opens via a fake dialogue manager.
- *Happy path:* Publishing `save/now` with `{slot_id: "slot1"}` writes via a fake save backend.
- *Edge:* `inventory/give_item` with no `count` defaults to 1.
- *Error path:* Publishing `audio/play_sound` with no `name` field — handler no-ops, debug log emitted, no panic.
- *Error path:* Publishing with a wrong-type payload field (e.g. `count: "five"`) — handler no-ops, no panic.
- *Edge:* `verbs.bus` target is registered exactly once even if `Boot` is called twice (idempotency).
- *Catalog:* `verb_recipes.go::EventBusTarget == "verbs.bus"`.
- *Integration:* `loop.main` is still `pievent.Target[Event]` (no regression to existing typed-enum target).

**Verification:** Every topic in `pixelforge_studio/scripting/topic_catalog.go` has a registered handler. Running an authored event-sheet through a smoke capsule produces real subsystem effects, not published-and-dropped events.

---

### U3. Real `go build` for Host + WASM targets behind `//go:build long`

**Goal:** Replace the scaffold-placeholder builders for Host (Windows/macOS/Linux) and WASM with real `go build -o <out> .` invocations in the generated capsule's outDir. WASM bundler already produces a real `.html` (per plan-007 AE3); this unit adds the missing `go build js/wasm` step that precedes the bundler.

**Requirements:** Without this, every Build click produces a placeholder file. The whole loop is theatre.

**Dependencies:** U1, U2 (generated capsule must actually run something meaningful).

**Files:**
- Create: `pixelforge_studio/buildpipeline/builders_long.go` (`//go:build long`; registers `hostLongBuilder` + `wasmLongBuilder` replacing scaffolds)
- Modify: `pixelforge_studio/buildpipeline/builders.go` — add `//go:build !long` so scaffold steps aside under long
- Modify: `pixelforge_studio/buildpipeline/toolchain.go` — `NewBuildCommand` grows a per-target CGO selection (Host targets: `CGO_ENABLED=1`; WASM: `CGO_ENABLED=0`). Alternatively add `NewNativeBuildCommand` + `NewWASMBuildCommand` variants — pick during impl based on which keeps the call sites cleaner.
- Modify: `pixelforge_studio/build/workspace.go` — Build buttons invoke `Build()` which internally adds `-tags=long` so users don't see the flag
- Create: `pixelforge_studio/integration_test/build_pipeline_long_test.go` (`//go:build long`)

**Approach:** `hostLongBuilder.Build(ctx, req, emit)`:
1. emit `PhaseQueued`
2. run codegen.Generate into a temp build dir
3. emit `PhaseGenerating`
4. (Windows host only) generate `.syso` via U5 helper, write next to `main.go`
5. call `NewBuildCommand(ctx, host, "build", "-o", outPath, ".")` with `Cmd.Dir = tempBuildDir` and `CGO_ENABLED=1`
6. emit `PhaseCompiling`
7. on success copy `outPath` to `<outDir>/host/<gameName><ext>` (and `<gameName>.app/Contents/MacOS/<gameName>` for macOS bundle layout)
8. emit `PhaseDone{OutputPath}`
9. on failure emit `PhaseFailed{Err}` with captured stderr

`wasmLongBuilder` runs `GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -o game.wasm .` then calls existing `wasm_bundler.BundleWASM` to wrap into a self-contained `.html`. Cancellation kills the underlying `*exec.Cmd` via context.

**Patterns to follow:** `pixelforge_studio/codegen/generator_long_test.go` for `//go:build long` precedent. `pixelforge_studio/buildpipeline/orchestrator.go` for `Builder` interface + phase-emit shape.

**Test scenarios:**
- *Happy path (long):* A minimal "empty project" capsule builds to a real Host executable that exits 0 when invoked with `--smoke`.
- *Happy path (long):* A project with one sprite, one sound, one scene builds Host AND WASM successfully.
- *Error path (long):* Forcing a broken template (mocked) surfaces as `PhaseFailed{Err: contains "compile"}`.
- *Edge (long):* Cancelling context mid-build kills the `go build` subprocess; `PhaseFailed{Err: ErrBuildCancelled}` emitted; no stranded process.
- *Default-tag regression:* Existing `build_pipeline_e2e_test.go` tests still pass — scaffold builder is the registered builder when `long` is absent.
- *Toolchain:* `NewBuildCommand` for Host target sets `CGO_ENABLED=1`; for WASM sets `CGO_ENABLED=0`.
- *Integration:* WASM long-tag test produces `.html` that, when opened in headless Chrome (skip if `chromedp` unavailable), reaches the click-to-start splash without console errors.

**Verification:** `go test -tags=long ./pixelforge_studio/integration_test/...` produces real binaries that execute. Studio Build button shells out with `-tags=long` automatically.

**Execution note:** Test-first for the new long-tag builders — write the `//go:build long` integration test asserting "real executable produced" first, watch it fail with "scaffold output is a marker file", then implement.

---

### U4. Host-target gating + two-button Build UI

**Goal:** Replace the five-checkbox Build workspace with two large buttons: **Host** and **WASM**. Orchestrator preflight rejects any cross-OS native target with a clear, host-naming error message. `Source` target stays callable via API for tests, not surfaced in UI.

**Requirements:** Aligns Build UI with the user-confirmed shape.

**Dependencies:** None (independent of U3 — UI and gating land cleanly with the scaffold builders still in place).

**Files:**
- Modify: `pixelforge_studio/build/workspace.go` — Render path becomes Host button + WASM button + per-target status pill; remove `Checked`/`SetChecked`/`CheckedTargets`; replace with direct `BuildHost()` + `BuildWASM()` methods
- Modify: `pixelforge_studio/buildpipeline/build_on_save.go` — add `CanBuildOnHost(t Target) bool` next to existing `HostTarget()`
- Modify: `pixelforge_studio/buildpipeline/orchestrator.go` — `Build()` preflight rejects targets where `!CanBuildOnHost(t)` with new `ErrCrossCompileNotSupported` sentinel that names the host's actual OS
- Modify: `pixelforge_studio/build/workspace_test.go` — assert button-click → builder dispatch + cross-OS rejection

**Approach:** `Workspace.BuildHost()` → `Build(ctx, req, []Target{HostTarget()})`. `Workspace.BuildWASM()` → `Build(ctx, req, []Target{TargetWASM})`. Per-target status pill reads from the workspace's existing status map. Source target stays in `Target` enum + `Builder` registry, just not rendered. Preflight rejection happens in `Build()` before goroutine dispatch — invalid targets emit `PhaseFailed{Err: ErrCrossCompileNotSupported(naming the host OS)}` immediately, then close the channel.

**Patterns to follow:** Existing `Workspace.StartBuild()` `done`-channel shape (preserved for tests). `docs/solutions/dirty-state-ux.md` (single-seam — Build/Cancel as the only entry points).

**Test scenarios:**
- *Happy path:* Clicking Host invokes `Build` with `[HostTarget()]`.
- *Happy path:* Clicking WASM invokes `Build` with `[TargetWASM]`.
- *Edge:* `CanBuildOnHost(TargetWASM)` returns true on every platform.
- *Edge:* `CanBuildOnHost(TargetSource)` returns true on every platform.
- *Edge:* On a Linux test host, `CanBuildOnHost(TargetWindows)` returns false; `CanBuildOnHost(TargetMacOS)` returns false.
- *Error path:* Programmatic `Build(ctx, req, []Target{TargetWindows})` on a non-Windows host emits `PhaseFailed{Err: ErrCrossCompileNotSupported}` immediately; no goroutine for that target spawned.
- *Error path:* `ErrCrossCompileNotSupported.Error()` mentions both the requested target's OS AND the host's actual OS (so error message is diagnosable).
- *UI:* Status pill updates show current phase (queued → generating → compiling → done) as the orchestrator emits.

**Verification:** Studio shows only Host + WASM buttons. The five-checkbox UI is gone. Cross-OS attempts via API fail fast with a clear message.

---

### U5. Icon emission from `docs/logo.svg`

**Goal:** Replace the deterministic hash-color fallback icon with a real raster of `docs/logo.svg` at every output. One source SVG → favicon (WASM), `.ico` + Windows `.syso` (Windows .exe), `.icns` (macOS .app). No per-project icon plumbing — same Pixelforge logo for every game.

**Requirements:** Closes the `errIconUnsupported` stub gap from plan-007.

**Dependencies:** None.

**Files:**
- Create: `pixelforge_studio/buildpipeline/icon_logo.go` — `//go:embed docs/logo.svg`; pure-Go raster via `oksvg` + `rasterx`; PNG/ICO/ICNS/SYSO output helpers
- Create: `pixelforge_studio/buildpipeline/icon_logo_test.go`
- Modify: `pixelforge_studio/buildpipeline/icon.go` — `GenerateFavicon(sprite)` retained but renamed to private `generateLegacyFavicon`; new public `GenerateFavicon()` (no args) returns the rasterized logo PNG base64; `GenerateWindowsIcoStub` + `GenerateMacIcnsStub` call into logo helpers and return real bytes; `errIconUnsupported` deprecated (kept one release for callers)
- Modify: `pixelforge_studio/buildpipeline/wasm_bundler.go` — `BundleWASM` callers updated to use the new no-arg `GenerateFavicon()` (callsite simplification)
- Modify: `pixelforge_studio/buildpipeline/builders_long.go` (U3) — Windows host builder writes the `.syso` next to generated `main.go` before `go build`; macOS host builder packages `.icns` into the `.app` bundle's `Contents/Resources/`
- Modify: `go.mod` — add `github.com/srwiley/oksvg`, `github.com/srwiley/rasterx`, `github.com/biessek/golang-ico`, `github.com/jackmordaunt/icns/v2`, `github.com/josephspurrier/goversioninfo`

**Approach:** Embed `docs/logo.svg` via `//go:embed`. `RasterLogoPNG(size int) ([]byte, error)` parses SVG via `oksvg.ReadIconStream`, rasterizes via `rasterx.NewDasher` at the requested size, encodes to PNG. `BuildLogoICO()` packs PNGs at 16/32/48/256 into a Windows ICO via `golang-ico`. `BuildLogoICNS()` packs the same sizes into a macOS ICNS via `icns/v2`. `BuildWindowsSyso(buildDir, gameName)` uses `goversioninfo` to emit `rsrc_windows_amd64.syso` so the Windows linker picks up the icon automatically. WASM favicon = base64 of the 32×32 PNG, fed into the existing `BundleWASM` template. Linux Host build skips icon embedding (no native binary icon convention) — accepted scope reduction.

**Patterns to follow:** `IconResult` shape preserved (`Sprite` field becomes optional/nil; `Note` text updates to "logo.svg"). No project schema changes (icon is global).

**Test scenarios:**
- *Happy path:* `RasterLogoPNG(32)` produces a non-empty PNG decodable to 32×32.
- *Happy path:* `BuildLogoICO()` returns bytes parseable as a valid Windows ICO containing four sub-images at 16/32/48/256.
- *Happy path:* `BuildLogoICNS()` returns bytes parseable as a valid macOS ICNS.
- *Happy path:* `BuildWindowsSyso(tempDir, "MyGame")` writes a file matching `rsrc_windows_amd64.syso`.
- *Happy path:* `GenerateFavicon()` returns base64 PNG; decoded image dominant non-transparent color matches the logo's `#a248b6` purple within tolerance.
- *Edge:* `IsIconUnsupported(err)` returns false on every new path (sentinel deprecated for real-output paths).
- *Edge:* `RasterLogoPNG(0)` returns an error (invalid size).
- *Regression:* Existing `IconResult.FaviconBase64` is non-empty (preserves AE5 invariant).
- *Integration (host-conditional):* macOS Host build (skipped on non-darwin hosts) packages `AppIcon.icns` and updates `Info.plist`'s `CFBundleIconFile`.

**Verification:** A Windows .exe built via Host shows the Pixelforge logo as its file icon in Explorer. A macOS .app shows it in Finder. WASM HTML's favicon shows it in the browser tab.

---

### U6. WASM save backend

**Goal:** `pixelforge_save/backend_js.go` adapter on top of browser `localStorage` so games built for WASM can save/load. Today only `backend_native.go` exists (`//go:build !js`), so WASM games would crash on first `save_now` recipe call.

**Requirements:** Recipes route saves through `Service`; without a JS backend, the verb is broken on WASM.

**Dependencies:** None.

**Files:**
- Create: `pixelforge_save/backend_js.go` (`//go:build js`)
- Create: `pixelforge_save/backend_js_test.go` (`//go:build js` — runs only under `wasmbrowsertest` / `go test -exec=wasmbrowsertest`)
- Modify: `pixelforge_save/doc.go` — remove the "BackendWASM (browser localStorage, build-tagged js) — pending" note
- Modify: `pixelforge_studio/capsuleruntime/runtime.go` (U1) — Boot's save-service construction picks the correct backend via build-tagged files (no GOOS branching in shared code)

**Approach:** `BackendJS` implements `pixelforge_save.Backend` using `syscall/js` to call `window.localStorage.getItem`/`setItem`/`removeItem`/`key`. Keys namespaced as `pixelforge.save.<gameTitle>.<slot>` so two different games don't collide in a shared origin. Encoded value is base64 of the JSON snapshot (localStorage stores strings only). `List()` iterates `localStorage.length`/`key(i)` filtering by prefix. `Delete()` calls `removeItem`. Quota errors (localStorage is 5–10 MB depending on browser) bubble up as a typed `ErrQuotaExceeded` the engine can surface to the user.

**Patterns to follow:** `pixelforge_save/backend_native.go` for interface conformance shape and method signatures. `pixelforge_save/backend.go` for the `Backend` interface contract.

**Test scenarios:**
- *Happy path:* `Write(slot, data)` then `Read(slot)` round-trips byte-equal.
- *Happy path:* `Write` to two slots; `List()` returns both, in any order.
- *Happy path:* `Delete(slot)` removes the entry; subsequent `Read` returns `ErrNotFound`.
- *Error path:* `Write` of an oversize blob (mock-set localStorage to throw QuotaExceededError) returns `ErrQuotaExceeded`.
- *Edge:* Two `BackendJS` instances with different `gameTitle` are isolated — writing slot "a" to game X does not appear in game Y's `List()`.
- *Edge:* Reading a slot that was never written returns `ErrNotFound`.
- *Regression:* Existing `backend_native.go` tests still pass (no cross-tag breakage).

**Verification:** A WASM build with a `save_now` recipe persists state across browser refresh — verified manually in a real browser if `wasmbrowsertest` is unavailable; manual-verify note explicit in the test file.

---

### U7. Recipe capability matrix doc

**Goal:** A single Markdown table at `docs/reference-games-capability-matrix.md` mapping the four reference games to recipes — green/yellow/red per (game, mechanic) pair. Green = directly buildable today; yellow = buildable with a wrapper combo; red = needs a new primitive (U8 covers these).

**Requirements:** Concrete "what's possible today" deliverable. Honest assessment supplants vague promises.

**Dependencies:** None (audit can start from existing catalog).

**Files:**
- Create: `docs/reference-games-capability-matrix.md`

**Approach:** Table rows = (game, mechanic). Columns = (status, recipes used, gap). Roughly 24 rows total — six mechanics each for Asteroids / Bomberman / Mario / Donkey Kong. Each RED row maps directly to a U8 entry. Matrix is iterated once after U8 ships so RED rows flip to GREEN with the new recipes listed.

Sketch of the expected shape:

| Game | Mechanic | Status | Recipes / Gap |
|------|----------|--------|---------------|
| Asteroids | thrust + rotate ship | RED | needs `apply_thrust`, `rotate_entity` (U8) |
| Asteroids | shoot bullet | GREEN | `spawn_entity` + `motion/move_pattern` |
| Asteroids | screen wrap | RED | needs `screen_wrap` (U8) |
| Asteroids | break asteroid into smaller pieces | YELLOW | `destroy_self` + `spawn_entity` ×N; no built-in size param |
| Bomberman | place bomb on grid cell | RED | needs `place_on_grid` + `spawn_entity` (U8) |
| Bomberman | bomb fuse → explosion + radius damage | RED | needs `explode_radius` (U8) |
| Mario | jump + gravity | RED | needs `jump`, `apply_gravity` (U8) |
| Mario | platform collision | RED | needs `solid_collide` (U8) |
| Donkey Kong | ladder climb | RED | needs `ladder_climb` (U8) |
| Donkey Kong | barrel roll physics | RED | needs `barrel_roll` (U8) |

(continued for all four games)

**Patterns to follow:** `docs/pforge-schema.md` for table-driven documentation tone. Other docs in `docs/` for header/section style.

**Test scenarios:** None (documentation deliverable).

**Test expectation:** none — pure documentation artifact. Verified by review covering all four games and matching every U8 entry.

**Verification:** Matrix is reviewable; every U8 recipe is justified by at least one matrix row; every RED row has a corresponding U8 entry.

---

### U8. Arcade primitive recipes

**Goal:** Add the missing verb recipes the reference games need, identified by U7. Each is a `Recipe` registered via the catalog's existing `Register*` init pattern.

**Requirements:** Without these, Asteroids / Bomberman / Mario / Donkey Kong can't be authored in GUI.

**Dependencies:** U2 (event-bus wiring works; new recipes publish via `verbs.bus`), U7 (matrix identifies the set).

**Files:**
- Create: `pixelforge_studio/scripting/catalog/builtin_arcade.go`
- Create: `pixelforge_studio/scripting/catalog/builtin_arcade_test.go`
- Modify: `pixelforge_studio/scripting/catalog/catalog.go` — `init()` registers the new builtin set alongside existing builtins
- Modify: `pixelforge_studio/capsuleruntime/subscribers.go` (U2) — install handlers for the new topics (`motion/apply_thrust`, `motion/rotate`, `motion/screen_wrap`, `motion/jump`, `motion/apply_gravity`, `collision/solid_collide`, `motion/place_on_grid`, `damage/explode_radius`, `motion/ladder_climb`, `motion/barrel_roll`, `motion/fixed_tick_loop`)

**Approach:** Each recipe is small. Initial set (refined by U7):

| Recipe | Purpose | Default args |
|---|---|---|
| `apply_thrust` | Adds vector to entity velocity (Asteroids ship) | `{angle_deg: 0, magnitude: 0.2}` |
| `rotate_entity` | Adjusts entity rotation by degrees (Asteroids ship) | `{degrees: 5}` |
| `screen_wrap` | Repositions entity to opposite edge when crossing screen edge (Asteroids) | (no args) |
| `jump` | Sets vertical velocity to negative value (Mario / DK player) | `{strength: 5}` |
| `apply_gravity` | Adds positive vertical velocity per tick (Mario / DK) | `{strength: 0.3}` |
| `solid_collide` | Entity-vs-tile collision against tile-atlas solid-flag layer; resolves penetration (Mario / DK) | (no args) |
| `place_on_grid` | Snaps entity position to nearest grid cell (Bomberman) | `{cell_size: 16}` |
| `explode_radius` | After fuse ticks, destroys self + damages entities within radius (Bomberman bomb) | `{fuse_ticks: 120, radius: 32, damage: 1}` |
| `ladder_climb` | Overrides gravity while entity overlaps ladder-flagged tile (DK) | `{speed: 1.5}` |
| `barrel_roll` | Applies per-tile slope-flag direction to entity velocity (DK barrel) | `{base_speed: 1.0}` |
| `fixed_tick_loop` | Step that groups a per-tick condition+action sequence (used by all four) | `{ticks_per_step: 1}` |

Each recipe defines `ActionKind`, default args, and a one-line description. Subscribers in U2 route each new topic to the entity manager / physics layer in the capsule runtime.

**Patterns to follow:** `pixelforge_studio/scripting/catalog/builtin_actions.go` for `Recipe` shape. `docs/solutions/scripting-runtime-design.md` (pure-function builders + `Register*` init). `pixelforge_studio/scripting/catalog/verb_recipes_test.go` for test structure (note: existing `TestVerbRecipes_AllBuiltinRecipesValid` already handles `IsCondition()` skip-path; new condition-style recipes must conform).

**Test scenarios:**
- *Happy path:* Each new recipe is registered; `LookupRecipe(id)` returns it.
- *Happy path:* Each recipe's default args populate without error.
- *Happy path:* Each recipe's published topic appears in `topic_catalog.go` enumeration (no orphaned topics).
- *Integration:* Build a synthetic "Asteroids-shaped" event sheet (rotate-left/right + thrust + shoot + screen-wrap) using only the new + existing recipes; parses cleanly and executes against fake subscribers without error.
- *Integration:* Build a synthetic "Mario-shaped" event sheet (jump + apply_gravity + solid_collide); same.
- *Integration:* Build a synthetic "Bomberman-shaped" event sheet (place_on_grid + spawn_entity + explode_radius); same.
- *Integration:* Build a synthetic "DK-shaped" event sheet (ladder_climb + barrel_roll); same.
- *Regression:* `TestVerbRecipes_AllBuiltinRecipesValid` still passes with the expanded set.

**Verification:** U7 matrix's RED rows for the four games flip to GREEN. Each game has an integration test that authors the core mechanic with only registered recipes.

---

### U9. Asset library catalog schema + first-launch downloader

**Goal:** Studio ships a manifest of curated CC0 + CC-BY asset packs (sprites / SFX / BGM) for the four reference games. On first launch the studio downloads packs to `os.UserCacheDir()/pixelforge/library/` with SHA-256 verification + atomic writes. Repo stays binary-free.

**Requirements:** Every project starts non-empty. "No-code complete game" promise is only real if the GUI offers ready-to-use assets.

**Dependencies:** None.

**Files:**
- Create: `pixelforge_studio/assetlibrary/manifest.go` — manifest struct + parse + schema-version check
- Create: `pixelforge_studio/assetlibrary/downloader.go` — HTTP fetch + checksum verify + atomic write
- Create: `pixelforge_studio/assetlibrary/pack.go` — pack discovery + on-disk layout helpers
- Create: `pixelforge_studio/assetlibrary/library.go` — in-memory index of installed packs
- Create: `pixelforge_studio/assetlibrary/manifest_test.go`, `downloader_test.go`, `pack_test.go`, `library_test.go`
- Modify: `pixelforge_studio/main.go` — call `assetlibrary.EnsureBootstrap(ctx, cacheDir)` on studio startup, non-blocking with a progress toast hook

**Approach:** Manifest is JSON, hosted at a stable GitHub Release URL on this repo (`https://github.com/ibilalkhan1/fyp_pixelforge/releases/download/asset-library-v1/manifest.json`), overridable via `PIXELFORGE_ASSET_LIBRARY_URL` env var for local dev/testing. Schema (additive forever):

```json
{
  "schema_version": "1",
  "packs": [
    {
      "id": "asteroids-starter",
      "version": "1.0.0",
      "title": "Asteroids Starter Pack",
      "game": "asteroids",
      "url": "https://github.com/.../releases/download/asset-library-v1/asteroids-starter-1.0.0.tar.gz",
      "sha256": "abc123…",
      "size_bytes": 1234567,
      "assets": [
        {"path": "sprites/ship.png", "kind": "sprite", "license": "CC0", "author": "Kenney", "source_url": "https://kenney.nl/..."},
        {"path": "audio/blast.wav",  "kind": "sfx",    "license": "CC-BY-4.0", "author": "...", "source_url": "https://freesound.org/..."}
      ]
    }
  ]
}
```

Downloader: HEAD-check cached manifest, fetch new on version change, download each pack tarball, SHA-256 verify before unpacking. Unpack to `<userCache>/pixelforge/library/<pack-id>/`. Atomic writes (tmp file → rename) so partial failures leave nothing on disk. Progress reported via callback for UI toast updates. `EnsureBootstrap` is idempotent — if packs already present + manifest version matches, no-op.

`os.UserCacheDir()` is the cross-platform answer: `~/.cache/pixelforge/` on Linux, `%LOCALAPPDATA%\pixelforge\` on Windows, `~/Library/Caches/pixelforge/` on macOS. `assetlibrary.New(cacheDir string)` injects the dir so tests use `t.TempDir()`; only `main.go` calls the real `os.UserCacheDir()`.

**Patterns to follow:** `docs/solutions/editor-pforge-schema-shape.md` (additive schema; sanitize on load; no field removal across versions). `docs/solutions/dirty-state-ux.md` (single-seam — `EnsureBootstrap` is the only entry point; no scattered fetch calls).

**Test scenarios:**
- *Happy path:* Manifest with two packs parses; assets list populates with license/author.
- *Happy path:* Download with matching SHA-256 succeeds; pack unpacks under `<cacheDir>/library/<pack-id>/`.
- *Error path:* Download with mismatched SHA-256 fails with `ErrChecksumMismatch`; no partial pack on disk.
- *Edge:* Download interrupted mid-flight (mocked) — tmp file removed; no partial pack visible to `library.Installed()`.
- *Edge:* Second `EnsureBootstrap` call with same manifest version no-ops (no re-download); third call after version bump re-downloads only affected packs.
- *Error path:* HTTP 404 on a pack URL surfaces as `ErrPackUnavailable` with the pack id named in the error.
- *Error path:* Manifest with `schema_version: "2"` on a v1-only studio returns `ErrUnsupportedSchema` instead of partial parse.
- *Edge:* Concurrent `EnsureBootstrap` calls coalesce (only one HTTP fetch) — verify with `sync.Once` or file lock.
- *Edge:* `PIXELFORGE_ASSET_LIBRARY_URL` env var overrides the default URL for both manifest and pack fetches.

**Verification:** Fresh studio install with no `<userCache>/pixelforge/library/`: studio launches, downloader runs in background, the four starter packs land in the cache dir, library workspace (U12) shows their assets.

---

### U10. License / credits UX

**Goal:** Display attribution where required. In the studio: license + author shown next to each asset in the Library workspace (U12). In exported games: auto-injected "Credits" scene listing every CC-BY asset used. CC0 assets omitted from credits (no attribution required).

**Requirements:** Legal hygiene for CC-BY use. Without this we ship unattributed CC-BY content.

**Dependencies:** U9 (need manifest license/author fields).

**Files:**
- Create: `pixelforge_studio/assetlibrary/credits.go` — credits assembler walks project, finds asset names, looks up license/author in installed packs, emits a credits dataset
- Create: `pixelforge_studio/assetlibrary/credits_test.go`
- Create: `pixelforge_studio/integration_test/credits_e2e_test.go`
- Modify: `pixelforge_studio/codegen/templates.go` — capsule template includes a `CapsuleCredits()` accessor returning the embedded credits dataset; menu auto-detects non-empty credits and adds a "Credits" entry to the title menu
- Modify: `pixelforge_studio/codegen/generator.go` — credits assembled at generate-time, embedded as a string-constant slice in `capsule.go`
- Modify: `pixelforge_studio/buildpipeline/wasm_bundler.go` + `wasm_template.html` — credits page accessible from the click-to-start splash as a "View Credits" overlay populated from the same dataset

**Approach:** `AssembleCredits(project, library) []CreditEntry` walks every asset reference in the project (sprites, audio bindings, BGM), looks each up in the installed library packs by name, returns `[]CreditEntry{Name, License, Author, SourceURL}` for CC-BY entries. CC0 entries excluded (no attribution duty). Codegen embeds the slice as a `var capsuleCredits = []runtime.CreditEntry{…}` literal in `capsule.go`. Engine menu auto-detects non-empty credits and adds a "Credits" entry to the title screen menu template. WASM bundler's HTML adds a hidden `<div id="credits">` populated from the same data, exposed via a "View Credits" button on the click-to-start splash.

**Patterns to follow:** `pixelforge_menus/registry.go` — Credits is a new built-in menu template kind. `pixelforge_studio/codegen/templates.go` string-constant embedding pattern (`SceneID*` constants are the precedent).

**Test scenarios:**
- *Happy path:* Project uses one CC-BY asset → credits list has one entry with correct Author + SourceURL.
- *Happy path:* Project uses only CC0 assets → credits list is empty; menu omits Credits entry.
- *Edge:* Project references an asset whose manifest entry is unknown (orphaned/user-imported) → entry recorded as `Author: "Unknown"` with a warning logged at generate-time.
- *Integration:* Generated capsule contains the credits literal and parses as valid Go.
- *Integration:* WASM bundler HTML includes the `<div id="credits">` populated from the credits data.
- *UI:* Library workspace renders license + author for every displayed asset (verified via headless workspace state test).

**Verification:** Build a Host game using a CC-BY asset; launch it; the title menu has a Credits entry; opening it shows author + source URL. CC0-only project has no Credits entry. WASM build shows the same data via the "View Credits" splash button.

---

### U11. Custom asset ingest — watched folder + drag-drop

**Goal:** Two complementary ways for users to add their own (or internet-sourced) assets to the studio. (a) Watched folder: `<userCache>/pixelforge/user-library/` auto-syncs into the library on file changes via `fsnotify`. (b) Drag-drop: dropping `.png`/`.wav`/`.ogg`/`.mp3` files on the studio window classifies by extension and routes through the existing import handlers.

**Requirements:** Lets users add custom assets without going through the one-at-a-time File→Import flow.

**Dependencies:** U9 (Library knows about user-library tab), existing PNG/WAV import runners (plan-005).

**Files:**
- Create: `pixelforge_studio/ingest/watcher.go` — fsnotify-based folder watcher with debounce
- Create: `pixelforge_studio/ingest/dragdrop.go` — Ebitengine drag-drop hook (Ebitengine 2.9.9 in go.mod supports `ebiten.AppendDroppedFiles`, so no version bump needed)
- Create: `pixelforge_studio/ingest/classifier.go` — extension → asset-kind dispatcher
- Create: `pixelforge_studio/ingest/watcher_test.go`, `dragdrop_test.go`, `classifier_test.go`
- Modify: `pixelforge_studio/main.go` — start watcher, register drag-drop callback
- Modify: `pixelforge_studio/editor/import_handler.go` — extend to accept `.ogg`/`.mp3` for BGM (today only handles `.png` + `.wav`); declare new `OGGImportRunner` + `MP3ImportRunner` interfaces injected via `RegisterWith` (cycle-break pattern)
- Modify: `go.mod` — add `github.com/fsnotify/fsnotify`

**Approach:** Watcher: `fsnotify` on `<userCache>/pixelforge/user-library/`; debounces 500ms (tunable); on stable, classifier reads extension, routes through `PNGImportRunner` (sprite), `WAVImportRunner` (sfx), new `OGGImportRunner` / `MP3ImportRunner` (BGM). Library workspace (U12) shows user-library assets in a separate "Custom" tab from curated packs.

Drag-drop: each frame the editor reads `ebiten.AppendDroppedFiles(nil)` and drains into the classifier. Visual feedback: drop overlay drawn on the window while files are over it (Ebitengine doesn't expose hover state cleanly — the overlay shows on first dropped file, fades out after 1s of no drops).

**Patterns to follow:** `PNGImportRunner` / `WAVImportRunner` cycle-break injection (registered via `RegisterWith` from outer package). `docs/solutions/dirty-state-ux.md` (single-seam ingest). `docs/solutions/file-picker-design.md` (imperative test API — no input mocking; expose `Ingest(path string)` directly).

**Test scenarios:**
- *Happy path:* A `.png` appearing in the watched folder triggers the PNG import runner with that path.
- *Happy path:* A `.wav` in the watched folder triggers WAV import.
- *Happy path:* A `.ogg` triggers OGG import (BGM kind).
- *Edge:* A `.txt` is ignored (unrecognized extension), no error.
- *Edge:* Watcher debounce coalesces 5 rapid writes to the same file into a single ingest.
- *Edge:* Classifier extension dispatch is case-insensitive (`.PNG` works).
- *Happy path:* Drag-drop dropping a single .png triggers PNG import.
- *Happy path:* Drag-drop dropping a mix of .png + .wav + .ogg triggers all three import runners.
- *Edge:* Drag-drop of an unrecognized extension is ignored with a debug log; UI shows no error toast.
- *Error path:* A malformed `.png` (e.g. text file with .png extension) surfaces the import-runner's error to the user as a toast; ingest pipeline does not crash.
- *UI:* Library workspace's Custom tab lists user-library assets (verified via headless state test).

**Verification:** Drop a `.png` onto the studio: appears in the Custom tab and is selectable in the asset picker. Save a `.wav` into the watched folder: same.

---

### U12. Asset Library Studio workspace

**Goal:** New "Library" workspace in the studio that browses installed packs + user assets. Per-game tabs (All / Asteroids / Bomberman / Mario / Donkey Kong / Custom). Asset preview (sprite thumbnail, audio audition button). Pick-to-project flow that copies an asset from the library into the current project's `assets/` dir.

**Requirements:** Without a UI, the downloaded library is just files in a cache dir.

**Dependencies:** U9 (downloader populates library), U10 (license display), U11 (Custom tab shows user-library assets).

**Files:**
- Create: `pixelforge_studio/assetlibrary/workspace.go` — Library workspace; imgui browser with per-game tabs
- Create: `pixelforge_studio/assetlibrary/preview.go` — thumbnail rendering, audio audition wrapper
- Create: `pixelforge_studio/assetlibrary/workspace_test.go`
- Modify: `pixelforge_studio/main.go` — `assetlibrary.RegisterWith(e)` after the other workspaces register
- Modify: `pixelforge_studio/editor/file_menu.go` — View menu adds "Library" entry; Ctrl+9 shortcut (continuing the Ctrl+5/6/7/8 series from plan-007)

**Approach:** Workspace shows tabs: All / Asteroids / Bomberman / Mario / Donkey Kong / Custom. Each tab lists assets from the matching pack (or all user-library assets for Custom). Asset row: thumbnail/icon + name + license badge + author + "Add to project" button. "Add to project" copies the file into `<projectAssetsDir>/<kind>/` and creates the matching `SpriteAsset` / `AudioSample` record in the project. Audio rows have an audition button using the existing `pixelforge_studio/audiolib/audition.go`. State separated from imgui calls per the file-picker test pattern — `Workspace.AddToProject(assetID)` is callable from tests without an imgui frame.

**Patterns to follow:** `pixelforge_studio/audiolib/workspace.go` (picker pattern + audition wiring). `docs/solutions/file-picker-design.md` (Render decoupling for testability).

**Test scenarios:**
- *Happy path:* Workspace loads installed packs from the library and renders them in tabs (verified via state inspection).
- *Happy path:* Per-game tab filters to that pack's assets only.
- *Happy path:* `AddToProject(assetID)` copies the file into `<projectAssetsDir>/sprites/` and records the `SpriteAsset` in the project struct.
- *Edge:* Adding the same asset twice is idempotent — no duplicate `SpriteAsset` record.
- *UI:* License badge text matches the manifest entry (CC0 / CC-BY-4.0).
- *Integration:* Audio audition triggers the existing `audiolib` audition path.
- *Edge:* Empty library (no packs downloaded) renders an "Install starter packs" prompt that triggers `EnsureBootstrap` on click.
- *Integration:* Custom tab shows user-library assets (depends on U11 watcher populating the library index).

**Verification:** Open studio → Ctrl+9 → Library renders the four reference-game tabs + Custom; clicking "Add to project" on a Bomberman wall sprite copies it into the project and the sprite appears in the sprite picker immediately.

---

## Scope Boundaries

**In scope (per confirmed synthesis):**
- Capsule runtime loaders for sprites / audio / dialogue / menus / items / scenes
- Real `go build` Host + WASM builders behind `//go:build long`
- Two-button Build UI (Host + WASM); cross-OS native builds rejected with clear error
- Per-platform icon emission from `docs/logo.svg` (Windows `.syso` + `.ico`, macOS `.icns`, WASM favicon)
- WASM save backend on browser `localStorage`
- New `verbs.bus` event target + handlers for all 32 published verb-recipe topics
- Recipe capability matrix for the four reference games
- ~11 arcade primitive recipes (thrust, rotate, screen-wrap, jump, gravity, solid-collide, place-on-grid, explode-radius, ladder-climb, barrel-roll, fixed-tick-loop)
- Asset library manifest schema + first-launch downloader with SHA-256 verification
- License / credits UX in studio + auto-injected credits in exported games (CC-BY only)
- Custom asset ingest: watched folder + drag-drop
- Library workspace with per-game tabs + Custom tab

### Deferred to Follow-Up Work
- **Curated asset pack assembly** (content workstream): sourcing/verifying CC0+CC-BY assets, packaging into tarballs, publishing the GitHub Release. This plan ships the manifest schema + downloader + workspace; the actual pack contents are a separate workstream. Tests use synthetic manifests pointing at fixture URLs.
- **macOS `.app` packaging polish**: codesign + notarization + DMG. Host builder produces a working unsigned `.app`.
- **Windows code signing**: `.exe` is unsigned; SmartScreen "unverified publisher" warning is expected.
- **Linux desktop integration**: `.desktop` file + AppImage / Flatpak packaging. Host builder produces an ELF; packaging conventions left to the user.
- **More reference games**: Pac-Man, Frogger, Space Invaders, Tetris, etc. Extending the matrix + recipe set is a v2 plan.
- **Asset library v2 features**: in-library search, tag-based browsing, community-submission flow, in-engine asset preview.
- **Headless browser smoke test for WASM** (`wasmbrowsertest` or `chromedp`): U3 + U6 + U10 tests cover the build pipeline + save backend + credits page, but a full end-to-end "click splash → game loads → save persists across refresh" test is deferred.
- **`os.UserCacheDir()` vs `os.UserConfigDir()` choice tuning**: starting with cache (downloadable, reproducible); user data dir is a v2 consideration if packs grow into authoring state.

### Outside this product's identity
- Multiplayer / networking primitives.
- Cloud / Steam / itch.io publishing integrations.
- iOS / Android targets (no Ebitengine mobile path in this codebase).
- AI / procedural-generation features.
- Cloud save / cross-device save sync.

---

## Risk Analysis & Mitigation

- **`oksvg` raster output differs across Go versions or library updates** → golden fixture comparison uses dimensions + dominant-color match within tolerance, not byte-equality.
- **`//go:build long` real builds depend on user having Go installed** → `NewBuildCommand` already handles vendored SDK fallback; long-tag CI job runs only on PRs touching `buildpipeline/`; default suite stays Go-free.
- **WASM cgo conflict** — Ebitengine native uses cgo (`CGO_ENABLED=1`); WASM target requires `=0` → per-target env override in `NewBuildCommand` (Host: `=1`; WASM: `=0`); audited by U3 tests.
- **Asset pack download bandwidth on first launch** (tens of MB) → async/background download with progress toast; studio fully functional in meantime; `--no-asset-library` flag for fully-offline operation.
- **License compliance hole** — a CC-BY asset's `author` field has a typo, breaking attribution → manifest validation at studio startup (asserts every CC-BY entry has non-empty `author` + `source_url`); CI lints the manifest as part of the asset-pack release workflow.
- **Event-bus type assertion still wrong somewhere** — `loop.main` consumers expect `Target[Event]`; new `verbs.bus` consumers expect `Target[string]` → U2 test asserts both registered with correct types; lookup in `builtin_actions.go` uses `EventBusTarget` constant exclusively.
- **`os.UserCacheDir()` differs by platform** → inject `cacheDir` into `assetlibrary.New(cacheDir)`; tests use `t.TempDir()`; only `main.go` calls real `os.UserCacheDir()`.

---

## Dependencies / Prerequisites

- Go 1.24+ (already required by codebase).
- Ebitengine 2.9.9 (already in `go.mod`) — `AppendDroppedFiles` available, no version bump for U11.
- New module dependencies (introduced by U5, U9, U11):
  - `github.com/srwiley/oksvg` (SVG raster) — U5
  - `github.com/srwiley/rasterx` (SVG raster) — U5
  - `github.com/biessek/golang-ico` (Windows `.ico`) — U5
  - `github.com/jackmordaunt/icns/v2` (macOS `.icns`) — U5
  - `github.com/josephspurrier/goversioninfo` (Windows `.syso`) — U5
  - `github.com/fsnotify/fsnotify` (watched folder) — U11
- GitHub Release hosting the asset packs (one-time setup; content workstream — see Deferred).

---

## Phased Delivery

Suggested groupings for git-history readability. Each phase is shippable independently if the user wants to pause.

- **Phase 1 — Runtime wiring (Capsule actually runs authored games):** U1, U2, U6.
- **Phase 2 — Real build pipeline + ship-able output:** U3, U4, U5.
- **Phase 3 — Recipe audit + arcade primitives:** U7, U8.
- **Phase 4 — Asset library + custom ingest:** U9, U10, U11, U12.

After Phase 2 ships, the loop is provably end-to-end for an empty-but-buildable capsule; after Phase 3, the four reference games are authorable; after Phase 4, users have curated content + a custom-ingest path.

---

## Implementation-Time Unknowns (Deferred to ce-work)

- Exact `fsnotify` debounce window in U11 (start at 500ms; tune based on real OS event volume).
- Whether `//go:build !long` on the existing scaffold is cleaner than `ResetBuildersForTest()` in `builders_long.go` init — pick during U3 impl based on which keeps tests + production code simpler.
- Whether icon generation runs once at studio startup (cached) or per-build (re-rasterized) — measure rasterization cost during U5; if >50 ms cache to studio startup.
- Exact pack format (`.tar.gz` vs `.zip`; per-pack vs single bundle) — defer to the content workstream; downloader's pack handler is pluggable via manifest's `url` extension.
- Whether `pixelforge_audio.Register/LookupSample` is package-global or capsule-injected — start with package-global (mirrors `pixelforge_menus`); refactor if tests show isolation issues.
- macOS `Info.plist` field details (`CFBundleVersion` / `CFBundleShortVersionString` format) — surface during U5 macOS path.
- Whether the Build button automatically streams build progress into a status pane vs. just updating the per-target pill — visual decision deferred to U4 impl.
