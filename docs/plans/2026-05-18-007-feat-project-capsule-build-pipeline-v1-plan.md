---
title: "feat: Project Capsule + Build Pipeline — codegen refactor, 5-target builds, WASM bundler, auto-icon, build-on-save (idea #7 v1)"
type: feat
status: active
date: 2026-05-18
depth: deep
origin: docs/brainstorms/2026-05-18-project-capsule-build-pipeline-v1-requirements.md
ideation: docs/ideation/2026-05-18-pixelforge-nes-class-no-code-ideation.md (idea #7)
satisfies_dependencies:
  - docs/plans/2026-05-18-002-feat-screenroom-mario-strip-v1-plan.md (idea #1 R12 — ship loop)
  - docs/plans/2026-05-18-005-feat-audio-library-picker-v1-plan.md (idea #4 R8 — ship loop)
  - docs/plans/2026-05-18-006-feat-rpg-class-systems-v1-plan.md (idea #6 — save file IO in shipped runtime)
ships_with:
  - all previous milestone plans
---

# feat: Project Capsule + Build Pipeline v1 (idea #7)

## Summary

v1 ships the complete ship loop. **Codegen refactored** to emit a Project Capsule — typed wrapper (`capsule.go`) + minimal `main.go` (~6 lines) + project data + assets embedded via `//go:embed all:assets/*` — that **for the first time actually wires the engine to the project data** (current codegen only emits `SetScreenWidth/SetTPS` and runs an empty game). One Capsule contract drives editor preview + shipped binary + tests + future Forgequest tutorials. **Build pane** dockable workspace with 5 target checkboxes (Windows `.exe`, macOS `.app`, Linux binary, WASM single-HTML, Source) + parallel build orchestration via `exec.CommandContext("go", "build", ...)` invoking a **vendored Go SDK** shipped alongside the studio installer (`<studio-install-dir>/go-sdk/bin/go`) with PATH fallback for source builds. **WASM target** bundles `game.wasm` (base64-encoded) + `wasm_exec.js` (inline) + click-to-start splash into one self-contained `<game-name>.html` (~50 LOC hand-rolled bundler — no mature Go library exists). **Auto-icon** generated from designer-marked sprite (new `Project.IconSpriteName` field) or auto-picked (most-referenced sprite + alphabetical tiebreak), produced via `josephspurrier/goversioninfo` (.syso for Windows) + `jackmordaunt/icns` (macOS) + `Kodeworks/golang-image-ico` (favicon). **Build-on-save** debounced 1.5s (between VS Code's 1s and brainstorm's 2-3s), host-platform only, runs at OS-priority-class `BELOW_NORMAL` / `nice 10` (VS 2022 precedent), suspendable via Settings → Build toggle. Output: `<project-dir>/exports/<target>/`. All native targets ship `CGO_ENABLED=0` (engine has zero CGO, confirmed via grep) — Linux ships as fully-portable static binary; macOS arm64 only in v1 (Intel via Rosetta). All builds unsigned in v1 (code-signing deferred). Total: **11 units**, foundational for the entire 7-idea milestone. Without this plan, every prior plan's ship-loop deliverables are unverifiable end-to-end.

---

## Leverage Doctrine (applied)

Per `docs/plans/2026-05-18-001-feat-entity-verb-sheet-v1-plan.md`'s Leverage Doctrine appendix.

**Candidates evaluated** (per external research, with maintenance status):

| Candidate | Status | Verdict |
|---|---|---|
| `josephspurrier/goversioninfo` (Windows .syso) | v1.7.0 May 2026; 907 stars; active | **Use** — canonical Windows resource embedder; cross-platform (runs on Linux/macOS); pure Go |
| `akavel/rsrc` (Windows .syso, lower-level) | Older | **Reject** — goversioninfo vendors most of rsrc's code internally + adds JSON config |
| `jackmordaunt/icns` (macOS .icns) | v2.2.7 Nov 2023; 92 stars; format-stable | **Use** — pure Go, cross-platform, auto-generates retina OSTypes from one PNG |
| `Kodeworks/golang-image-ico` (Windows .ico) | Dormant but functional; 27 stars; multi-size via loop | **Use** — small leaf dep, functional; alternative (biessek/golang-ico) is also dormant |
| WASM single-HTML inliner libraries | None mature for Go | **Build native** — `text/template` + `encoding/base64` is ~50 LOC; no library worth wrapping |
| Vendored Go SDK distribution libraries | None | **Build native** — extract + invoke by absolute path pattern (Wails/Fyne precedent); no library |
| Go build orchestration / parallel build libraries | None mature; `exec.CommandContext` is enough | **Build native** — wrap existing `generator_long_test.go:34` pattern with parallel + cancellation |
| `fsnotify` for project-save watching | Available | **Skip** — the studio is the only writer of the .pforge; hook on `SaveTo` directly per `file_menu.go:73-84` |
| Service-task / job-queue libraries (`tylertreat/BoomFilters`, `gocraft/work`) | Heavy | **Skip** — 4 concurrent builds need only goroutines + channel + context.Cancel; ~80 LOC |

**Three new direct dependencies** (all pure Go, all add to studio's `go.mod`):
- `github.com/josephspurrier/goversioninfo` (Windows resource embedder)
- `github.com/jackmordaunt/icns/v2` (macOS .icns)
- `github.com/Kodeworks/golang-image-ico` (Windows .ico; favicon)

Plus stdlib: `os/exec`, `runtime`, `encoding/base64`, `text/template`, `golang.org/x/sys/windows` (for `SetPriorityClass` on Windows builds).

Total custom: ~150 LOC Capsule generator + ~300 LOC build orchestrator + ~100 LOC WASM bundler + ~120 LOC icon pipeline + ~150 LOC Build workspace + ~80 LOC build-on-save + ~100 LOC toolchain detection + supporting tests. Well below wrap costs.

---

## Problem Frame

Pixelforge's codegen today (`pixelforge_studio/codegen/generator.go`) emits a Go source tree — `main.go` + `go.mod` + `project.pforge` + `assets/`. The thin shim's `main.go` calls `SetScreenWidth/SetTPS` then `pixelforge_ebiten.Run()` and runs an **empty game**. Sprites, scenes, audio, behaviors are not applied to the engine (admitted by template comment at generator.go:40-50). Three critical defects compound:

- **The shipped artifact is source code, not a binary.** Designers in the target audience (friends/classmates, not pre-trained on Go) cannot ship Go source. They don't have `go` on PATH. They don't know what `go build` means. They can't hand a directory of Go files to a classmate and expect a playable game.
- **The current shim doesn't even run the game's content.** Even if a designer learned `go build`, the binary would launch an empty 320×180 window — sprites unloaded, scenes empty, no audio. The runtime contract for "Capsule loads the project's content" was deferred and never implemented.
- **Three prior brainstorms explicitly depend on this.** Idea #1's Mario-strip (R12 — ship the Mario level to a classmate), idea #4's audio library (R8 — the cross-machine "classmate plays with sound" demo), idea #6's RPG systems (R2 — save files in user-config-dir from a shipped binary). All assume "the build pipeline exists" and "the binary actually runs the project." Neither is true today.

The user's first-session requirements were unambiguous: **single-file double-clickable binary per platform**, **WASM in the browser offline like Chrome Dino**, **custom icon from a project sprite**, **just like retro ROMs**. None of those four ship today. The "no-code, ship a complete game" promise collapses at the last step.

This plan is **foundational infrastructure** for the entire 7-idea milestone. Without it, every prior plan's ship-loop deliverables remain unverified end-to-end.

---

## Carried Forward from Origin

All 31 requirements, 8 acceptance examples, 4 flows, 5 actors from origin are in scope.

| Origin | Scope summary | Plan unit(s) |
|---|---|---|
| R1, R2, R3, R4, R5 | Project Capsule architecture (typed wrapper + load any .pforge + embedded assets + pinned engine + Source target) | U1, U4 (asset embedding) |
| R6 | Dockable Build workspace | U7 |
| R7, R8, R9, R10 | 5 target checkboxes, parallel builds, per-target status, Open output folder, error display | U4, U7 |
| R11 | Windows .exe (GOOS=windows GOARCH=amd64) + icon via goversioninfo | U4 (Windows builder), U5 (icon) |
| R12 | macOS .app bundle (Apple Silicon arm64 only) + .icns icon + zipped for distribution | U4 (macOS builder), U5 |
| R13 | Linux static binary (GOOS=linux GOARCH=amd64 CGO_ENABLED=0) | U4 (Linux builder) |
| R14 | Game name from Project.Name; version from new Project.Version field (or ISO date fallback) | U2 (schema) |
| R15 | Unsigned binaries; OS warnings documented | Documented in scope boundaries + plan |
| R16, R17, R18, R19 | WASM build → single .html with inline wasm_exec.js + base64 .wasm + click-to-start splash + offline | U6 (WASM bundler) |
| R20, R21, R22, R23 | Auto-icon: designer-marked sprite OR most-used 16×16 sprite OR fallback default; per-platform formats | U2 (IconSpriteName field), U5 |
| R24, R25, R26, R27 | Build-on-save: debounced 1.5s, host-only, status pill in chrome, suspendable via settings | U8 (build-on-save), U10 (settings) |
| R28, R29 | Output paths under `<project-dir>/exports/<target>/`; overwrite silently | U4 |
| R30, R31 | Vendored Go toolchain in studio installer (~150-300 MB); matching wasm_exec.js | U3 (toolchain), U9 (installer packaging) |
| AE1-AE8, F1-F4 | All eight acceptance examples + four flows | U11 (integration tests) |
| A1-A5 | Designer, end-player, Studio, vendored Go toolchain, Project Capsule | All units |

Origin's "Deferred to Planning" section: all 13 technical/research questions resolved in Phase 2 (see Key Technical Decisions). Two discovered questions from research:
1. **Current codegen emits an empty game** — Capsule refactor is where the runtime wiring finally lands. Documented in U1's approach as the most important scope.
2. **No installer exists today** — installer packaging (U9) is mostly scripts/release-tooling; doesn't change studio runtime behavior.

---

## High-Level Technical Design

How the pieces fit together:

```
                  CODEGEN (refactor, U1)
                  ════════════════════════════════════════════════════════
   ┌──────────────────────────────────────────────────────────────────────┐
   │ Generate(p *Project, outDir string, opts) →                          │
   │                                                                      │
   │ outDir/                                                              │
   │ ├── main.go         (NEW SHAPE: ~6 lines — capsule.Run(Defaults())) │
   │ ├── capsule.go      (NEW — generated typed wrapper; FIRST           │
   │ │                    runtime wiring of sprites/scenes/audio/        │
   │ │                    behaviors to the engine)                       │
   │ ├── go.mod                                                           │
   │ ├── project.pforge   (existing — game data)                          │
   │ ├── assets/                                                          │
   │ │   ├── sprites/                                                     │
   │ │   ├── audio/                                                       │
   │ │   └── (other subdirs as they exist in <name>.pforge-assets/)      │
   │ └── vendor/          (existing — Pixelforge engine + Ebitengine     │
   │                       pinned; excludes pixelforge_stat for WASM     │
   │                       compatibility)                                 │
   │                                                                      │
   │ capsule.go uses //go:embed:                                          │
   │   //go:embed project.pforge                                          │
   │   var projectData []byte                                             │
   │   //go:embed all:assets                                              │
   │   var assetsFS embed.FS                                              │
   └──────────────────────────────────────────────────────────────────────┘

                  BUILD ORCHESTRATOR (U4)
                  ════════════════════════════════════════════════════════
   ┌──────────────────────────────────────────────────────────────────────┐
   │ pixelforge_studio/buildpipeline/orchestrator.go                      │
   │                                                                      │
   │ Build(req BuildRequest, targets []Target) <-chan BuildStatus         │
   │                                                                      │
   │ For each Target in parallel goroutines:                              │
   │   1. Codegen.Generate(p, tempDir, opts)  (U1)                        │
   │   2. Icon.Generate(...)  (U5)                                        │
   │   3. exec.CommandContext("<vendoredGo>", "build", ...)               │
   │      env: GOOS, GOARCH, CGO_ENABLED=0, GOROOT=<vendoredGoRoot>       │
   │      proc priority: BELOW_NORMAL / nice 10                            │
   │   4. Per-target post-process:                                         │
   │      - Windows: include .syso (already in tempDir from U5)            │
   │      - macOS: assemble .app bundle dir + zip; place .icns; write     │
   │        Info.plist                                                    │
   │      - Linux: chmod +x; static binary as-is                          │
   │      - WASM: U6 bundler → single .html                               │
   │      - Source: copy tempDir verbatim to exports/source/              │
   │   5. Move output to <project-dir>/exports/<target>/<game-name>.{ext} │
   │   6. Emit BuildStatus{Target, Phase, BuiltAt, Err} on channel        │
   └──────────────────────────────────────────────────────────────────────┘

                  TOOLCHAIN (U3)
                  ════════════════════════════════════════════════════════
   ┌──────────────────────────────────────────────────────────────────────┐
   │ pixelforge_studio/buildpipeline/toolchain.go                         │
   │                                                                      │
   │ ResolveGoBinary() (path string, err error)                           │
   │   1. Look for <executableDir>/go-sdk/bin/go (vendored — installer)   │
   │   2. Fallback to exec.LookPath("go") (PATH — source build mode)      │
   │   3. Return error if neither found                                   │
   │                                                                      │
   │ ResolveGoRoot() string                                               │
   │   1. <executableDir>/go-sdk (vendored)                                │
   │   2. Output of `go env GOROOT` (PATH fallback)                       │
   │                                                                      │
   │ ResolveWasmExecJS() (path string, err error)                         │
   │   1. <GOROOT>/lib/wasm/wasm_exec.js (Go 1.24+)                       │
   │   2. <GOROOT>/misc/wasm/wasm_exec.js (Go pre-1.24)                   │
   └──────────────────────────────────────────────────────────────────────┘

                  WASM BUNDLER (U6)
                  ════════════════════════════════════════════════════════
   ┌──────────────────────────────────────────────────────────────────────┐
   │ pixelforge_studio/buildpipeline/wasm_bundler.go                      │
   │                                                                      │
   │ BundleWASM(wasmPath, wasmExecJSPath, gameName,                       │
   │            faviconBase64 string, outPath string) error                │
   │   1. Read wasm bytes; base64-encode                                  │
   │   2. Read wasm_exec.js verbatim                                      │
   │   3. text/template execute with {GameName, WasmExecJS, WasmBase64,   │
   │      FaviconBase64} → single HTML                                    │
   │   4. Write to outPath                                                │
   │                                                                      │
   │ HTML template structure:                                             │
   │   <html><head>                                                       │
   │     <title>{game_name}</title>                                       │
   │     <link rel=icon href="data:image/png;base64,{favicon}">           │
   │     <style>body{margin:0; background:#000; ... splash overlay ...}   │
   │            #splash{...}#splash button{...}</style>                   │
   │   </head><body>                                                       │
   │     <div id=splash><h1>{game_name}</h1><button>Click to start</button>│
   │     <canvas hidden></canvas>                                          │
   │     <script>{wasm_exec.js inline}</script>                            │
   │     <script>                                                          │
   │       const wasmBase64 = "{wasm_base64}";                            │
   │       document.getElementById("splash").onclick = async () => {      │
   │         document.getElementById("splash").hidden = true;             │
   │         const bytes = Uint8Array.from(atob(wasmBase64),              │
   │           c=>c.charCodeAt(0));                                       │
   │         const go = new Go();                                          │
   │         const result = await WebAssembly.instantiate(bytes,          │
   │           go.importObject);                                          │
   │         go.run(result.instance);                                     │
   │       };                                                              │
   │     </script>                                                         │
   │   </body></html>                                                      │
   └──────────────────────────────────────────────────────────────────────┘

                  ICON PIPELINE (U5)
                  ════════════════════════════════════════════════════════
   ┌──────────────────────────────────────────────────────────────────────┐
   │ pixelforge_studio/buildpipeline/icon.go                              │
   │                                                                      │
   │ ResolveIconSprite(p *Project) *SpriteAsset                           │
   │   1. If p.IconSpriteName non-empty → lookup that sprite              │
   │   2. Else: scan all sprites; rank by reference count in:             │
   │        - scenes' entity instances (Components.Values)                │
   │        - bindings, dialogue, items                                   │
   │      tie-break: alphabetical by Name                                 │
   │      filter: prefer 16×16 (but not strict)                           │
   │   3. Fallback: embedded default Pixelforge icon (shipped in studio)  │
   │                                                                      │
   │ GenerateIcons(sprite *SpriteAsset, outDir string)                    │
   │   - Load sprite PNG bytes                                            │
   │   - Resize to standard sizes (image.Image scale)                     │
   │   - Windows .ico via Kodeworks/golang-image-ico — 16/24/32/48/      │
   │     64/128/256 (256 PNG-compressed)                                  │
   │   - macOS .icns via jackmordaunt/icns — auto OSTypes                 │
   │   - WASM favicon (32×32 PNG base64-encoded)                          │
   │                                                                      │
   │ GenerateSyso(iconPath string, p *Project, outDir string)             │
   │   - goversioninfo with version_info JSON + .ico → resource_*.syso   │
   │   - filename suffix matches GOARCH (resource_amd64.syso etc.)        │
   └──────────────────────────────────────────────────────────────────────┘

                  STUDIO UI (U7, U8)
                  ════════════════════════════════════════════════════════
   ┌──────────────────────────────────────────────────────────────────────┐
   │ Build Workspace (U7)                                                 │
   │  ┌────────────────────────────────────────────────────────────┐      │
   │  │ [ ] Windows .exe       Status: ─                           │      │
   │  │ [ ] macOS .app          Status: ─                          │      │
   │  │ [ ] Linux binary        Status: Done · 12s ago [Open]      │      │
   │  │ [ ] WASM single-HTML    Status: Building... ⟳              │      │
   │  │ [ ] Source              Status: ─                          │      │
   │  │                                                            │      │
   │  │ [Build] [Open output folder]                               │      │
   │  └────────────────────────────────────────────────────────────┘      │
   │                                                                      │
   │ Chrome status pill (U8)                                              │
   │  Status bar: "Build ready · 2s ago" / "Building..." /                │
   │              "Build failed (click for details)"                      │
   └──────────────────────────────────────────────────────────────────────┘

                  RUNTIME COMPOSITION (the actual ship loop)
                  ════════════════════════════════════════════════════════

  Capsule's main.go:
    package main
    import "<project>/capsule"
    func main() { capsule.Run(capsule.Defaults()) }

  Capsule.Run(opts):
    1. Load embedded project.pforge → *pixelforge_project.Project
    2. For each Sprite in p.Sprites: load PNG bytes from embedded assets/
    3. For each AudioSample in p.Audio: load WAV bytes from embedded assets/
    4. Apply ScreenWidth/Height/TPS to pixelforge engine globals
    5. Register all pievent targets
    6. Initialize Capsule runtime services from prior plans:
       - pixelforge_blackboard (idea #5)
       - pixelforge_input (idea #5)
       - pixelforge_dialogue (idea #6)
       - pixelforge_menus (idea #6)
       - pixelforge_save (idea #6)
       - pixelforge_audio.Allocator (idea #4)
    7. Set up sceneGame loop: tilemap.Render + entity.RenderAll +
       dialogueRenderer.Update/Draw + menuStack.Update/Draw +
       camera.Update (idea #1)
    8. Call pixelforge_ebiten.Run() (existing)
```

*This illustrates the intended approach and is directional guidance for review, not implementation specification.*

The structural insight: **the Capsule is the seam where every prior plan's runtime contract converges**. Idea #1's renderers, idea #4's allocator, idea #5's blackboard/input/verb-catalog, idea #6's dialogue/menus/save — all plug into the Capsule's startup sequence. Before this plan, none of them are actually running in a shipped binary. With it, everything ships.

---

## Output Structure

```
pixelforge_studio/codegen/                       (MODIFY heavily, U1)
├── generator.go                                  — refactor Generate to emit Capsule
├── generator_test.go
├── templates.go                                  — replace main.go template; ADD capsule.go template
├── capsule_template.go                           (NEW) — capsule.go generation logic
└── capsule_template_test.go                      (NEW)

pixelforge_studio/buildpipeline/                  (NEW package, U3-U8)
├── orchestrator.go                               (U4) — parallel build coordination
├── orchestrator_test.go
├── toolchain.go                                  (U3) — vendored Go SDK discovery
├── toolchain_test.go
├── builders/
│   ├── windows.go                                (U4) — Windows .exe builder + goversioninfo wiring
│   ├── windows_test.go
│   ├── macos.go                                  (U4) — macOS .app bundle assembler + Info.plist
│   ├── macos_test.go
│   ├── linux.go                                  (U4) — Linux static binary
│   ├── linux_test.go
│   ├── wasm.go                                   (U4) — WASM build invocation (delegates bundling to U6)
│   ├── wasm_test.go
│   ├── source.go                                 (U4) — Source target (copy gen output)
│   └── source_test.go
├── wasm_bundler.go                               (U6) — inline-HTML bundler
├── wasm_bundler_test.go
├── icon.go                                       (U5) — auto-pick + generate icons
├── icon_test.go
├── priority/
│   ├── priority_unix.go                          (U4) — nice 10 on linux/darwin (build tags)
│   └── priority_windows.go                       (U4) — SetPriorityClass on windows
├── build_on_save.go                              (U8) — debouncer + host-only background build
├── build_on_save_test.go
└── doc.go

pixelforge_studio/buildpipeline/cart_assets/      (NEW)
└── default_icon.png                              — Pixelforge fallback icon (16×16)

pixelforge_studio/build/                          (NEW package — Studio workspace)
├── workspace.go                                  (U7) — Build pane UI (target checkboxes + Build button + status pills)
├── workspace_test.go
└── status_view.go                                (U7) — per-target status rendering

pixelforge_studio/editor/
├── imgui_chrome.go                               (MODIFY, U8) — extend status bar with BuildStatusPill
├── status_pill.go                                (NEW, U8)   — rich status pill widget
├── settings.go                                   (MODIFY, U10) — add BuildOnSaveDisabled bool
├── file_menu.go                                  (MODIFY, U10) — View → Build entry + Ctrl+8 binding
├── file_menu.go                                  (MODIFY, U8)  — SaveTo hook to trigger build-on-save
└── keymap.go                                     (MODIFY, U10) — workspace.build = Ctrl+8

pixelforge_studio/main.go                         (MODIFY, U7) — register build.Workspace + start build-on-save daemon

pixelforge_project/
├── project.go                                    (MODIFY, U2) — add Version + IconSpriteName fields
├── project_test.go                               (MODIFY)
└── rpg_defaults.go                               (extend from idea #6 — default Version to current ISO date if empty)

pixelforge_studio/modulepath/
└── detect.go                                     (MODIFY, U1) — exclude pixelforge_stat from WASM vendor copy

go.mod                                            (MODIFY, U5) — add goversioninfo, icns/v2, golang-image-ico

scripts/                                          (NEW, U9)
├── package-installer-linux.sh                    — assemble Linux .tar.gz bundle (studio + go-sdk)
├── package-installer-macos.sh                    — assemble macOS .dmg or .zip bundle
├── package-installer-windows.ps1                 — assemble Windows .zip bundle
└── README.md                                     — installer packaging instructions

pixelforge_studio/integration_test/
├── build_pipeline_e2e_test.go                    (NEW, U11)
└── fixtures/
    ├── tiny_capsule_project.pforge               — minimal valid project for fast build tests
    ├── mario_strip_full.pforge                   — full project exercising all systems (idea #1+#2+#3+#4+#5+#6)
    └── icon_sprite.png                           — synthetic 16×16 PNG for icon tests
```

Implementer may consolidate or split files; per-unit `**Files:**` sections remain authoritative.

---

## Implementation Units

Dependency-ordered. Foundation (U1-U3) → orchestration (U4-U6) → polish (U5, U7-U8) → release (U9-U10) → tests (U11).

### U1. Project Capsule architecture — codegen refactor

**Goal:** Refactor `pixelforge_studio/codegen/generator.go` to emit a Project Capsule: minimal `main.go` (~6 lines) + generated `capsule.go` (typed wrapper that ACTUALLY wires the engine to the project data) + `project.pforge` + `assets/` (embedded via `//go:embed all:assets`) + `vendor/`. This is **the most important unit** because the current codegen emits a binary that runs an empty game; the Capsule is where every prior plan's runtime contract finally converges.

**Requirements:** R1 (capsule.go + main.go shrink), R2 (one Capsule for editor preview + shipped binary + tests), R3 (assets embedded), R4 (pinned engine), R5 (Source target produces complete project tree).

**Dependencies:** Strict on schema additions from idea #5 (Archetype + Blackboard packages must exist), idea #1 (TileAtlas + engine renderers), idea #6 (Dialogue + Menus + Save packages) — Capsule imports them all. **Practical execution order:** Capsule template includes hooks for all prior subsystems; if a subsystem hasn't landed yet, its hook is a no-op call. Capsule ships v1 as the umbrella that wires everything that exists.

**Files:**
- `pixelforge_studio/codegen/generator.go` (MODIFY — extend Generate to emit capsule.go alongside main.go; keep existing vendor/asset-copy flow)
- `pixelforge_studio/codegen/templates.go` (MODIFY — new minimal main.go template; new capsule.go template constants)
- `pixelforge_studio/codegen/capsule_template.go` (NEW — generation logic for capsule.go, including type-safe accessors)
- `pixelforge_studio/codegen/capsule_template_test.go` (NEW)
- `pixelforge_studio/codegen/generator_test.go` (MODIFY — new tests for Capsule output)
- `pixelforge_studio/modulepath/detect.go` (MODIFY — `shouldVendorDir` excludes `pixelforge_stat` when target is WASM)

**Approach:**
- **New `main.go` template** (~6 lines):
  ```
  package main
  import "<modulepath>/capsule"
  func main() {
      if err := capsule.Run(capsule.Defaults()); err != nil {
          panic(err)
      }
  }
  ```
- **New `capsule.go` generated content** (more substantial; ~300-400 lines depending on project content):
  - Package declaration: `package capsule`
  - `//go:embed project.pforge` → `var projectData []byte`
  - `//go:embed all:assets` → `var assetsFS embed.FS`
  - `type CapsuleOpts struct { WindowWidth, WindowHeight int; Fullscreen bool; DebugOverlay bool; DataOverride *pixelforge_project.Project }` (DataOverride lets editor preview pass its in-memory project instead of loading from embed)
  - `func Defaults() CapsuleOpts` — sensible defaults
  - `func Run(opts CapsuleOpts) error` — the foundational runtime sequence:
    1. Load project: `p := opts.DataOverride; if p == nil: p = pixelforge_project.LoadReader(projectData)`
    2. Apply engine globals: `pixelforge.SetScreenSize(p.ScreenWidth, p.ScreenHeight); pixelforge.SetTPS(p.TPS)`
    3. Init audio backend: `piaudio.Backend = audio.StartAudioBackend(...)` (idea #4 U1)
    4. Load all sprite PNGs from assetsFS into `pixelforge` sprite registry
    5. Load all audio samples from assetsFS into `pixelforge_audio` sample registry
    6. Init blackboard from defaults: `bb := pixelforge_blackboard.New()` (idea #5)
    7. Init input intent layer (idea #5)
    8. Init verb catalog + register all bindings from `p.Bindings`
    9. Init dialogue runtime: `dialogue := pixelforge_dialogue.NewRuntime(p.Dialogues, bb)` (idea #6)
    10. Init menu stack: `menus := pixelforge_menus.NewStack()` (idea #6)
    11. Init save service: `saver := pixelforge_save.NewService(p.Name)` (idea #6)
    12. Init Paula allocator (idea #4)
    13. Init scene state: load p.Scenes[0] (or save-restore if a save exists)
    14. Set up `Game` struct implementing ebiten.Game interface:
       - `Update()`: dispatch input intents, run scripting engine tick (unless paused via menus.IsPaused), update dialogue + menus
       - `Draw(screen)`: tilemap.Render → entity.RenderAll → dialogue.Draw → menus.Draw
    15. Call `pixelforge_ebiten.Run(game)`
- **Generated `capsule.go` is gofmt'd, hand-readable** — designers learning Go can read it. Comments explain the sequence.
- **Typed accessors** (per R1): the generator inspects `p.Scenes`, `p.Sprites`, `p.Items`, `p.Dialogues`, `p.Menus` and emits typed constants:
  ```
  // Generated constants:
  const SceneIDTitle  = "title"
  const SceneIDLevel1 = "level_1"
  const ItemIDPotion  = "potion"
  // etc.
  ```
  Designers/programmers can reference these (e.g., for unit tests of game logic).
- **The DataOverride field** lets the editor preview pass its in-memory project state (with unsaved edits) to the same Capsule code path that the shipped binary uses. **One Capsule contract, multiple call sites** per R2.
- **Excludes `pixelforge_stat`** from vendor for WASM target via `modulepath/detect.go:235-252` extension — `pixelforge_stat` uses gopsutil which doesn't compile under `GOOS=js`. Add a target-aware `shouldVendorDir(dir, target Target)` parameter; existing call sites pass current host target; new call sites for cross-compile + WASM pass the explicit target.

**Patterns to follow:** existing codegen at `pixelforge_studio/codegen/generator.go:73-332`; existing template at `templates.go:12-66`; existing vendor flow at `modulepath/detect.go:170-252`; existing test pattern at `generator_long_test.go:34` for `go build` invocation; idea #4's plan U1 for backend init order; idea #5's plan for blackboard/input/catalog init; idea #6's plan for dialogue/menus/save init.

**Test scenarios:**
- `TestGenerate_EmitsMainAndCapsule`: Generate writes both main.go (~6 lines) and capsule.go (substantial) to outDir.
- `TestGenerate_CapsuleEmbedsProjectAndAssets`: capsule.go contains both `//go:embed project.pforge` and `//go:embed all:assets` directives.
- `TestGenerate_CapsuleHasRunFunction`: capsule.go exports `func Run(opts CapsuleOpts) error`.
- `TestGenerate_CapsuleHasTypedAccessors`: project with scenes "title" and "level_1"; capsule.go contains `const SceneIDTitle = "title"` and `const SceneIDLevel1 = "level_1"`.
- `TestGenerate_CapsuleHasDefaultsFunction`: capsule.go exports `func Defaults() CapsuleOpts`.
- `TestGenerate_VendorExcludesPixelforgeStatForWASM`: Generate with target=WASM; resulting vendor/ does NOT contain pixelforge_stat directory.
- `TestGenerate_VendorIncludesPixelforgeStatForNative`: Generate with target=Linux; vendor includes pixelforge_stat (current behavior preserved).
- `TestGenerate_AssetsCopiedToAssetsDir`: project with sprites + audio; outDir/assets/sprites/*.png and outDir/assets/audio/*.wav exist.
- `TestGenerate_NoAssetsDoesNotFail`: project with empty p.Sprites and p.Audio; Generate succeeds; assets/ directory exists but empty.
- `TestGenerate_GofmtRoundTrip`: generated capsule.go passes `gofmt -l` (no diff); generated main.go same.
- `TestGenerate_GeneratedFilesCompileViaGoBuild` (`//go:build long`): Generate + `go build` in outDir; binary exists.
- `TestGenerate_BinaryRunsAndExitsCleanly` (`//go:build long`): Generate + go build + exec binary with `--exit-immediately` flag (or 1-frame run); exit code 0.
- `TestCapsuleRun_LoadOverrideUsesInMemoryProject`: invoke `capsule.Run(CapsuleOpts{DataOverride: testProject})`; capsule uses testProject not embedded data. (Editor preview's mechanism.)
- `TestCapsuleRun_LoadFromEmbedWhenNoOverride`: invoke `capsule.Run(Defaults())`; capsule loads project from embedded bytes.
- `TestCapsuleRun_SetsScreenSizeFromProject`: project ScreenWidth=320 Height=180; after Run, `pixelforge.GetScreenSize() == (320, 180)`.
- `TestCapsuleRun_LoadsSpritesFromEmbed`: project with 3 sprites; after Run, all 3 are loaded in the sprite registry.
- `TestCapsuleRun_FailsCleanlyOnMalformedProjectData`: corrupted projectData bytes; Run returns error (not panic).
- `TestModulePathDetect_ShouldVendorDirRespectsTarget`: shouldVendorDir("pixelforge_stat", TargetNative) → true; ("pixelforge_stat", TargetWASM) → false.
- Covers AE2 (binary runs with all assets — depends on this unit producing functional capsule), AE7 (Source target produces complete tree).

**Verification:** `go test ./pixelforge_studio/codegen/...` passes including `-tags long`; manual smoke: generate a project; `go build`; run binary; observe sprites + scenes + audio actually playing (not empty game).

---

### U2. Schema additions — Project.Version + Project.IconSpriteName

**Goal:** Add two new `omitempty` fields to `Project`: `Version string` (game version stamped into binary metadata; defaults to current ISO date if empty) and `IconSpriteName string` (designer-marked sprite to use as game icon; if empty, auto-pick heuristic fires).

**Requirements:** R14 (game version field), R20 (designer-marked icon sprite).

**Dependencies:** none (foundational; parallel with U1, U3).

**Files:**
- `pixelforge_project/project.go` (MODIFY — add 2 fields)
- `pixelforge_project/project_test.go` (MODIFY)
- `pixelforge_project/rpg_defaults.go` (MODIFY from idea #6 — extend applyDefaults to set Version to current ISO date if empty)

**Approach:**
- `Project.Version string \`json:"version,omitempty"\`` — designer-set; empty → applyDefaults stamps `time.Now().Format("2006-01-02")` (today's ISO date).
- `Project.IconSpriteName string \`json:"icon_sprite_name,omitempty"\`` — designer-set via SpriteAsset inspector toggle (U7 hooks it via custom widget OR a designated checkbox in sprite inspector).
- `applyDefaults` extension:
  - If `p.Version == ""`: set to `time.Now().Format("2006-01-02")`.
  - If `p.IconSpriteName != ""`: validate the sprite exists in `p.Sprites`; if not, clear (sanitize-clamp per `editor-pforge-schema-shape.md`); log warning.
- **Designer UI for IconSpriteName**: a checkbox in the SpriteAsset inspector ("Use as game icon"). When toggled on for sprite X, `p.IconSpriteName = "X"`; when X's checkbox is toggled off OR another sprite Y is toggled on, X's checkbox goes off (singleton enforcement at the UI layer, not the schema layer). Inspector logic lives in U7 (custom widget) or as inline UI in existing inspector.

**Patterns to follow:** existing additive-omitempty discipline; existing `applyDefaults` extensions per project.go:107-113; existing JSON-tag conventions; `editor-pforge-schema-shape.md` sanitize discipline.

**Test scenarios:**
- `TestProject_VersionEmptyDefaultsToISODate`: load project without `version`; after applyDefaults, Version matches `YYYY-MM-DD` format.
- `TestProject_VersionExplicitPreserved`: load project with `version: "1.2.3"`; after applyDefaults, Version unchanged.
- `TestProject_IconSpriteNamePersisted`: project with IconSpriteName="hero"; round-trip; preserved.
- `TestProject_IconSpriteNameValidatedExisting`: load project with IconSpriteName="nonexistent"; after applyDefaults + sanitize, IconSpriteName="" + warning logged.
- `TestProject_IconSpriteNameOmitEmpty`: project with IconSpriteName=""; marshaled JSON omits the key.
- `TestProject_VersionOmitEmptyWhenStamped`: NOT — date-stamped value is always non-empty, so omitempty doesn't trigger.

**Verification:** `go test ./pixelforge_project/...` passes; existing fixture (`editor.pforge`) round-trips with Version auto-stamped to today's date on load.

---

### U3. Vendored Go toolchain discovery + invocation helper

**Goal:** Build `pixelforge_studio/buildpipeline/toolchain.go` that locates the vendored Go SDK at `<executableDir>/go-sdk/bin/go` (shipped by the installer per U9), falls back to `exec.LookPath("go")` if vendored SDK is absent (source-build mode), resolves `GOROOT` + `wasm_exec.js` paths consistently, and exposes a single `exec.CommandContext`-wrapping helper for build invocations with priority-class / CGO settings.

**Requirements:** R30 (vendored Go toolchain), R31 (matching wasm_exec.js).

**Dependencies:** none (foundational; parallel with U1, U2).

**Files:**
- `pixelforge_studio/buildpipeline/toolchain.go` (NEW)
- `pixelforge_studio/buildpipeline/toolchain_test.go` (NEW)
- `pixelforge_studio/buildpipeline/priority/priority_unix.go` (NEW — build tag `//go:build !windows`)
- `pixelforge_studio/buildpipeline/priority/priority_windows.go` (NEW — build tag `//go:build windows`)

**Approach:**
- `ResolveGoBinary() (path string, err error)`:
  1. Get executable dir via `os.Executable()` + `filepath.Dir()`.
  2. Check `<execDir>/go-sdk/bin/go` (or `go.exe` on Windows). If exists, return that path.
  3. Fallback to `exec.LookPath("go")`. If found, return.
  4. Otherwise return error: "Go toolchain not found. Install Go or use the bundled studio installer."
- `ResolveGoRoot() string`:
  1. If vendored: `<execDir>/go-sdk`.
  2. Else: invoke `<goBinary> env GOROOT`.
- `ResolveWasmExecJS() (path string, err error)`:
  1. Try `<GOROOT>/lib/wasm/wasm_exec.js` (Go 1.24+).
  2. Try `<GOROOT>/misc/wasm/wasm_exec.js` (Go pre-1.24).
  3. Else error.
- `NewBuildCommand(ctx context.Context, target Target, args ...string) *exec.Cmd`:
  - Resolves Go binary.
  - Sets `Env`: `GOROOT=<resolved>`, `GOOS=<target.GOOS>`, `GOARCH=<target.GOARCH>`, `CGO_ENABLED=0`, plus os.Environ() (filtered to drop user `GOPATH`/`GOTOOLCHAIN` to avoid surprises).
  - Returns `exec.CommandContext` (cancellable via ctx).
- `SetBuildPriority(cmd *exec.Cmd)` (cross-platform via build tags):
  - Unix: `cmd.SysProcAttr = &syscall.SysProcAttr{...}` — uses `syscall.Setpriority` post-Start, OR sets via `nice` wrapper. (Implementation detail; first approach is cleaner.)
  - Windows: uses `golang.org/x/sys/windows.SetPriorityClass(handle, BELOW_NORMAL_PRIORITY_CLASS)` post-Start.
- **Toolchain discovery is cached** in package-level sync.Once to avoid repeated `os.Stat`/`exec.LookPath` calls.

**Patterns to follow:** existing `runGoModTidy` at `generator.go:158-166` for `exec.Command` shape; `capture/export.go:183` for sync.Once-cached path resolution; standard Go cross-compile env-var convention.

**Test scenarios:**
- `TestResolveGoBinary_PrefersVendored`: temp dir with `go-sdk/bin/go` shim; SetExecutableDir(tempDir); ResolveGoBinary returns the shim path.
- `TestResolveGoBinary_FallsBackToPATH`: no vendored; exec.LookPath returns system Go; ResolveGoBinary returns that.
- `TestResolveGoBinary_ErrorIfNeitherFound`: no vendored AND PATH doesn't have go (manipulate PATH for test); ResolveGoBinary returns error.
- `TestResolveGoRoot_VendoredReturnsAbsolute`: vendored SDK at `<execDir>/go-sdk`; ResolveGoRoot returns that.
- `TestResolveGoRoot_PATHFallbackInvokesGoEnv`: ResolveGoRoot invokes `go env GOROOT` when no vendored SDK.
- `TestResolveWasmExecJS_Go124Path`: GOROOT has lib/wasm/wasm_exec.js; returns it.
- `TestResolveWasmExecJS_LegacyMiscPath`: GOROOT has only misc/wasm/wasm_exec.js; returns it.
- `TestResolveWasmExecJS_ErrorIfMissing`: GOROOT has neither; returns error.
- `TestNewBuildCommand_SetsGOOSGOARCH`: NewBuildCommand(target=Windows); cmd.Env contains "GOOS=windows" and "GOARCH=amd64".
- `TestNewBuildCommand_SetsCGOEnabledZero`: cmd.Env contains "CGO_ENABLED=0".
- `TestNewBuildCommand_DropsUserGOTOOLCHAIN`: parent env has GOTOOLCHAIN=local; cmd.Env does NOT contain GOTOOLCHAIN.
- `TestSetBuildPriority_UnixSetsNice`: on Linux/macOS; spawn dummy process; SetBuildPriority; check process priority via /proc or ps (test may need to be `//go:build long`).
- `TestSetBuildPriority_WindowsSetsBelowNormal`: on Windows; test priority via Windows API call.

**Verification:** `go test ./pixelforge_studio/buildpipeline/...` passes; manual smoke: launch studio with vendored SDK; ResolveGoBinary returns vendored path; launch studio with no vendored SDK; ResolveGoBinary returns PATH go.

---

### U4. Build orchestrator + per-target builders (Windows/macOS/Linux/WASM/Source)

**Goal:** New `pixelforge_studio/buildpipeline/orchestrator.go` exposes `Build(req BuildRequest, targets []Target) <-chan BuildStatus`. Parallel goroutines invoke per-target builders (separate files per target). Each builder: generates Capsule (U1) into temp dir, generates icon resources (U5) if needed, invokes `go build` via toolchain helper (U3) with proper GOOS/GOARCH/CGO settings, post-processes (e.g., assembles .app bundle for macOS), moves output to `<project-dir>/exports/<target>/<game-name>.{ext}`. Status streaming via channel: Queued → Building → Done/Failed.

**Requirements:** R7 (5 target checkboxes), R8 (parallel build with per-target status), R9 (output paths), R10 (per-target error display), R11 (Windows), R12 (macOS .app bundle + arm64), R13 (Linux static binary, CGO_ENABLED=0), R28 (output paths), R29 (overwrite silently).

**Dependencies:** U1 (Capsule codegen), U2 (Project.Version), U3 (toolchain helper), U5 (icon generation), U6 (WASM bundler — only needed for WASM target).

**Files:**
- `pixelforge_studio/buildpipeline/orchestrator.go` (NEW)
- `pixelforge_studio/buildpipeline/orchestrator_test.go` (NEW)
- `pixelforge_studio/buildpipeline/builders/windows.go` (NEW)
- `pixelforge_studio/buildpipeline/builders/windows_test.go` (NEW)
- `pixelforge_studio/buildpipeline/builders/macos.go` (NEW)
- `pixelforge_studio/buildpipeline/builders/macos_test.go` (NEW)
- `pixelforge_studio/buildpipeline/builders/linux.go` (NEW)
- `pixelforge_studio/buildpipeline/builders/linux_test.go` (NEW)
- `pixelforge_studio/buildpipeline/builders/wasm.go` (NEW)
- `pixelforge_studio/buildpipeline/builders/wasm_test.go` (NEW)
- `pixelforge_studio/buildpipeline/builders/source.go` (NEW)
- `pixelforge_studio/buildpipeline/builders/source_test.go` (NEW)

**Approach:**
- **Types:**
  ```
  type Target int
  const (
      TargetWindows Target = iota; TargetMacOS; TargetLinux; TargetWASM; TargetSource
  )
  type BuildRequest struct { Project *Project; ProjectPath string; OutputDir string }
  type BuildStatus struct { Target Target; Phase Phase; Output string; Err error; BuiltAt time.Time }
  type Phase int
  const (
      PhaseQueued Phase = iota; PhaseGenerating; PhaseCompiling; PhasePackaging; PhaseDone; PhaseFailed
  )
  ```
- **`Build(req, targets) <-chan BuildStatus`**:
  - Spawns one goroutine per target.
  - Each goroutine: creates a temp dir, calls `Capsule.Generate(req.Project, tempDir, codegenOpts)`, then dispatches to per-target builder, which handles compile + package.
  - Status events flow to a single channel; caller reads until channel closes.
  - On cancellation (caller cancels ctx), all in-flight builds terminate.
- **Windows builder** (`builders/windows.go`):
  - Generate icon `.syso` via U5 (placed in tempDir alongside main.go).
  - `NewBuildCommand(ctx, TargetWindows, "build", "-o", "<game-name>.exe", "-ldflags", "-H windowsgui", ".")` (the -H windowsgui suppresses the cmd console window).
  - Set priority via U3 helper.
  - Run; on success, move `<game-name>.exe` to `<outputDir>/windows/<game-name>.exe`.
- **macOS builder** (`builders/macos.go`):
  - Generate `.icns` via U5.
  - Build binary via `NewBuildCommand(ctx, TargetMacOS, "build", "-o", "binary", ".")`.
  - **Assemble .app bundle:**
    - `<game-name>.app/Contents/MacOS/<binary>` ← compiled binary.
    - `<game-name>.app/Contents/Resources/AppIcon.icns` ← generated icns.
    - `<game-name>.app/Contents/Info.plist` ← templated XML with `CFBundleName=<game-name>`, `CFBundleIdentifier=com.pixelforge.<sanitized-name>`, `CFBundleVersion=<project.Version>`, `CFBundleIconFile=AppIcon`, etc.
  - Zip the `.app` bundle for distribution (`<game-name>.app.zip`).
  - Move both `<game-name>.app/` directory AND `<game-name>.app.zip` to `<outputDir>/macos/`.
  - **GOARCH=arm64 only in v1** (Apple Silicon; Intel users use Rosetta per origin scope).
- **Linux builder** (`builders/linux.go`):
  - `NewBuildCommand(ctx, TargetLinux, "build", "-ldflags", "-s -w", "-o", "<game-name>", ".")` (`-s -w` strips debug info for smaller binary).
  - CGO_ENABLED=0 ensures static binary.
  - `os.Chmod(0o755)` on resulting binary.
  - Move to `<outputDir>/linux/<game-name>`.
- **WASM builder** (`builders/wasm.go`):
  - `NewBuildCommand(ctx, TargetWASM, "build", "-ldflags", "-s -w", "-o", "game.wasm", ".")` with GOOS=js GOARCH=wasm.
  - Then call `wasm_bundler.BundleWASM(...)` from U6 to produce single .html.
  - Move `<game-name>.html` to `<outputDir>/wasm/`.
- **Source builder** (`builders/source.go`):
  - Just calls Capsule.Generate(req.Project, `<outputDir>/source/`, codegenOpts).
  - No compilation; no GO build; no priority setting.
  - Designer can `go build` from the source dir themselves.
- **Output overwrites silently** per R29 — each builder calls `os.RemoveAll(<outputDir>/<target>/<game-name>...)` before writing new output.
- **Open output folder helper** (`OpenOutputFolder(target)`): platform-specific shell-out to file manager (`xdg-open`/`open`/`explorer`).
- All builders take a `context.Context` so the orchestrator can cancel mid-build (e.g., when build-on-save triggers a new build before the previous finishes).

**Patterns to follow:** existing `exec.Command` pattern at `generator.go:158-166`; existing test pattern at `generator_long_test.go:34` for verified-builds tests; standard Go cross-compile env conventions.

**Test scenarios:**
- **Orchestrator:**
  - `TestBuild_QueuesAllTargets`: Build with 3 targets; receives 3 Queued events on channel.
  - `TestBuild_TargetsRunInParallel`: 3 targets; total time < sum of individual times (each builder simulated as 1s sleep).
  - `TestBuild_CancellationStopsInFlight`: start build; cancel ctx after 100ms; builders see context done; no output files written.
  - `TestBuild_FailedTargetDoesNotBlockOthers`: 2 targets, one fails; other still completes; channel emits both Done and Failed.
  - `TestBuild_OutputDirCreated`: build to nonexistent `<projectDir>/exports/linux/`; dir created.
- **Linux builder** (`//go:build long` for actual go build):
  - `TestLinuxBuilder_ProducesExecutable`: build tiny project; output file at expected path; has executable bit.
  - `TestLinuxBuilder_CGOEnabledZero`: inspect resulting binary's `ldd` (or `file`) output; confirm no dynamic deps.
  - `TestLinuxBuilder_OverwritesExisting`: existing file at output path; rebuild; file replaced.
- **Windows builder** (`//go:build long`):
  - `TestWindowsBuilder_ProducesExe`: build tiny project with target=Windows from Linux host; `<game-name>.exe` exists; verify via file magic bytes (MZ header).
  - `TestWindowsBuilder_IncludesSyso`: tempDir during build contains generated resource_amd64.syso.
  - `TestWindowsBuilder_LdflagsWindowsGui`: invocation includes `-ldflags "-H windowsgui"`.
- **macOS builder** (`//go:build long`):
  - `TestMacOSBuilder_ProducesAppBundle`: build; `<game-name>.app/Contents/MacOS/<binary>` exists; `Info.plist` exists; `Resources/AppIcon.icns` exists.
  - `TestMacOSBuilder_PlistHasCorrectFields`: parse generated Info.plist; CFBundleName matches project.Name.
  - `TestMacOSBuilder_AppZipExists`: `<game-name>.app.zip` exists alongside the directory.
  - `TestMacOSBuilder_GOARCHIsARM64`: invocation env has GOARCH=arm64.
- **WASM builder** (`//go:build long`):
  - `TestWASMBuilder_ProducesHTML`: build; `<game-name>.html` exists at output path; nothing else (no .wasm sidecar, no .js sidecar).
  - `TestWASMBuilder_HTMLContainsWasmBase64`: HTML file contains "wasmBase64" variable.
- **Source builder:**
  - `TestSourceBuilder_ProducesProjectTree`: output contains main.go, capsule.go, go.mod, project.pforge, assets/, vendor/.
  - `TestSourceBuilder_NoExecCalls`: spy on exec.Command — no calls made (Source target doesn't compile).
- Covers AE1 (Build pane → both Windows and Linux build in parallel), AE2 (binary runs identically to editor preview), AE7 (Source target produces complete tree).

**Verification:** `go test ./pixelforge_studio/buildpipeline/...` passes (including `-tags long` for real builds); manual smoke: build Linux from Linux host → run binary → game launches.

---

### U5. Icon generation pipeline — auto-pick + per-platform formats

**Goal:** `pixelforge_studio/buildpipeline/icon.go` resolves the icon sprite (designer-marked via Project.IconSpriteName OR auto-picked by reference count), then generates per-platform formats: Windows .ico (via `Kodeworks/golang-image-ico`), Windows .syso (via `josephspurrier/goversioninfo`), macOS .icns (via `jackmordaunt/icns/v2`), WASM favicon (32×32 PNG base64). Fallback to a default Pixelforge icon if project has no sprites.

**Requirements:** R20 (designer-marked), R21 (auto-pick heuristic), R22 (default fallback), R23 (per-platform formats).

**Dependencies:** U2 (Project.IconSpriteName field), U4 (orchestrator invokes this per-target).

**Files:**
- `pixelforge_studio/buildpipeline/icon.go` (NEW)
- `pixelforge_studio/buildpipeline/icon_test.go` (NEW)
- `pixelforge_studio/buildpipeline/cart_assets/default_icon.png` (NEW — embedded fallback)
- `go.mod` (MODIFY — add 3 deps: goversioninfo, icns/v2, golang-image-ico)

**Approach:**
- **`ResolveIconSprite(p *Project) (*pixelforge_project.SpriteAsset, error)`**:
  1. If `p.IconSpriteName != ""`: look up in `p.Sprites`. If found, return.
  2. Else: auto-pick:
     - Build reference-count map by walking `p.Scenes[].Entities[].Components.Values` for sprite-typed values; counting occurrences in `p.Bindings`, `p.Dialogues` (stage directions referencing sprites), `p.Items[].IconSpriteRef`.
     - Filter to sprites that exist in `p.Sprites`.
     - Sort by ref count (desc) then by name (asc) for tiebreak.
     - If non-empty list, return first.
     - **Prefer 16×16** sprites: if any sprite with ref count > 0 has FrameW=16 + FrameH=16, prefer it over larger sprites with same count.
  3. Else: return the default Pixelforge icon (from `//go:embed cart_assets/default_icon.png`).
- **`GenerateIcons(sprite *SpriteAsset, projectPath string, outDir string)`**:
  - Load sprite's PNG bytes (from project assets or embedded default).
  - Decode to `image.Image`.
  - Resize to standard sizes via `image.Image` scaling (use `image/draw` or `golang.org/x/image/draw` for higher-quality resize — actually `x/image/draw` is better for icons).
  - **Windows .ico**: write multi-size ICO containing 16/24/32/48/64/128/256 PNG entries. Use `Kodeworks/golang-image-ico`'s loop pattern (encode each size + concatenate). 256×256 entry MUST be PNG-compressed (Vista+).
  - **Windows .syso**: build via `josephspurrier/goversioninfo` with:
    - `versioninfo.json` content: ProductName=<game-name>, FileVersion=<project.Version>, CompanyName="Pixelforge", FileDescription=<game-name>, OriginalFilename=<game-name>.exe.
    - Icon path: the .ico generated above.
    - Output: `resource_amd64.syso` (filename suffix matches GOARCH; Go linker picks up `.syso` files automatically when building).
  - **macOS .icns**: use `jackmordaunt/icns/v2` to write multi-size ICNS from a 1024×1024 input (resize sprite up). Library auto-generates OSTypes (ic04/ic05/ic07/ic08/ic09/ic10).
  - **WASM favicon**: encode 32×32 sprite as PNG, base64; return string for embedding in HTML.
  - **Linux**: skipped per R23 (no XDG integration in v1).
- **`ReferenceCount(p *Project, spriteName string) int`**: counts mentions of `spriteName` in:
  - `Scene.Entities[].Components.Values` for any field whose schema kind is WidgetSpriteRef.
  - `Items[].IconSpriteRef`.
  - Future: `Dialogues` stage-direction references.
- Cache decoded sprite icons per-build to avoid double-decode.

**Patterns to follow:** existing PNG decode at `pixelforge_studio/palette/import_pipeline.go`; existing `//go:embed` precedent at `imgui_theme.go:29`; library docs for each of the 3 new deps.

**Test scenarios:**
- `TestResolveIconSprite_DesignerMarkedReturnsThat`: project with IconSpriteName="hero" and sprite "hero"; returns hero.
- `TestResolveIconSprite_AutoPickByReferenceCount`: project with 3 sprites; "hero" referenced 5 times, "enemy" 3 times, "rock" 1 time; auto-pick returns "hero".
- `TestResolveIconSprite_TiebreakByAlphabetical`: 2 sprites tied at 5 refs; returns the alphabetically-first.
- `TestResolveIconSprite_Prefers16x16OverLarger`: "hero" (16×16, 3 refs) vs "boss" (32×32, 3 refs); returns "hero".
- `TestResolveIconSprite_FallbackToDefaultWhenNoSprites`: project with empty p.Sprites; returns embedded default.
- `TestResolveIconSprite_FallbackWhenMarkedNotFound`: IconSpriteName="ghost" but no such sprite; auto-pick fires (logged warning).
- `TestReferenceCount_CountsEntityComponentRefs`: scene has 3 entities all referencing "hero" via Sprite component; count returns 3.
- `TestReferenceCount_CountsItemIconRefs`: item with IconSpriteRef="potion"; count for "potion" includes this.
- `TestGenerateIcons_WindowsIcoContainsAllSizes`: generate; open .ico; entries present for 16/24/32/48/64/128/256.
- `TestGenerateIcons_Windows256IsPNGCompressed`: 256×256 entry in .ico has PNG magic bytes.
- `TestGenerateIcons_SysoContainsVersionInfo`: parse resource.syso; contains ProductName=<game-name>, FileVersion=<project.Version>.
- `TestGenerateIcons_IcnsCreated`: generate; .icns file exists; verifiable via macOS `iconutil` (or magic bytes 'icns').
- `TestGenerateIcons_FaviconBase64Returns32x32PNG`: returned string decoded → PNG with 32×32 dimensions.
- `TestGenerateIcons_CrossPlatform`: run all generators from Linux host; all 4 output formats produced (Windows + macOS + favicon, Linux skipped).
- Covers AE4 (icon visible in Windows Explorer; .exe properties show game name + version).

**Verification:** `go test ./pixelforge_studio/buildpipeline/...` passes; manual smoke: build Windows .exe with hero sprite as icon; copy to Windows machine; verify Explorer shows hero icon + Properties shows game version.

---

### U6. WASM single-HTML bundler

**Goal:** Hand-rolled bundler that combines `game.wasm` (base64-encoded), `wasm_exec.js` (inline), favicon (base64), and a click-to-start splash into one self-contained `<game-name>.html`. Single file, double-clickable into a browser, works offline.

**Requirements:** R16 (WASM build), R17 (single .html with inline contents), R18 (click-to-start splash satisfies browser autoplay), R19 (offline — no network requests).

**Dependencies:** U3 (toolchain.ResolveWasmExecJS), U5 (favicon for WASM target).

**Files:**
- `pixelforge_studio/buildpipeline/wasm_bundler.go` (NEW)
- `pixelforge_studio/buildpipeline/wasm_bundler_test.go` (NEW)
- `pixelforge_studio/buildpipeline/wasm_template.html` (NEW — `//go:embed` source for the template)

**Approach:**
- `BundleWASM(wasmPath, wasmExecJSPath, gameName, faviconBase64, outPath string) error`:
  1. Read wasm bytes from wasmPath.
  2. Base64-encode wasm bytes (33% size inflation).
  3. Read wasm_exec.js verbatim.
  4. Execute `text/template` with:
     - `.GameName` — project name (escaped for HTML).
     - `.WasmExecJS` — inline `<script>` body.
     - `.WasmBase64` — base64-encoded wasm.
     - `.FaviconBase64` — favicon for browser tab.
  5. Write to outPath.
- **HTML template structure** (`wasm_template.html`):
  ```
  <!DOCTYPE html>
  <html><head>
    <meta charset=utf-8>
    <title>{{.GameName}}</title>
    <link rel=icon type=image/png href="data:image/png;base64,{{.FaviconBase64}}">
    <style>
      body { margin:0; background:#000; font-family:sans-serif; color:#fff; }
      #splash {
        position:fixed; top:0; left:0; right:0; bottom:0;
        display:flex; flex-direction:column;
        align-items:center; justify-content:center;
        background:#000; cursor:pointer;
      }
      #splash h1 { font-size:48px; }
      #splash p { font-size:18px; opacity:0.7; }
      canvas { display:block; }
    </style>
  </head><body>
    <div id=splash>
      <h1>{{.GameName}}</h1>
      <p>Click to start</p>
    </div>
    <canvas id=screen hidden></canvas>
    <script>{{.WasmExecJS}}</script>
    <script>
      const wasmBase64 = "{{.WasmBase64}}";
      document.getElementById("splash").addEventListener("click", async () => {
        document.getElementById("splash").style.display = "none";
        document.getElementById("screen").hidden = false;
        const bytes = Uint8Array.from(atob(wasmBase64), c => c.charCodeAt(0));
        const go = new Go();
        const result = await WebAssembly.instantiate(bytes, go.importObject);
        go.run(result.instance);
      });
    </script>
  </body></html>
  ```
- **Size**: 33% inflation from base64 + wasm_exec.js (~30KB) + template overhead (~1KB). A 15MB wasm → ~20MB HTML. Browsers handle this; under the practical ~30MB threshold for UX.
- **Offline guarantee** (R19): no `<script src=>`, no `<link href=>` to external URLs; no fetch() calls in user code. Everything inline.
- **Click-to-start splash** (R18): satisfies all major browsers' autoplay policy. Chrome/Firefox/Safari all require user gesture before AudioContext starts.

**Patterns to follow:** standard `text/template`; standard `encoding/base64`; existing `//go:embed` pattern; standard HTML structure.

**Test scenarios:**
- `TestBundleWASM_OutputIsSingleHTMLFile`: bundle; outPath exists; no other files in outDir.
- `TestBundleWASM_HTMLContainsGameName`: html contains `<title>MyGame</title>`.
- `TestBundleWASM_HTMLContainsWasmExecJSInline`: html contains "function Go()" (a token from wasm_exec.js).
- `TestBundleWASM_HTMLContainsBase64Wasm`: html contains "wasmBase64" variable with non-empty content.
- `TestBundleWASM_HTMLContainsFavicon`: html contains `data:image/png;base64,` favicon URL.
- `TestBundleWASM_HTMLContainsSplash`: html contains `<div id=splash>` + click-to-start text.
- `TestBundleWASM_NoExternalReferences`: parse html; no `<script src=>` with http/https; no `<link href=>` with external URL.
- `TestBundleWASM_SizeRoughly33PercentLargerThanWasm`: 1MB wasm input; output ~1.33MB + overhead.
- `TestBundleWASM_GameNameHTMLEscaped`: gameName="<script>alert(1)</script>"; html does NOT contain unescaped script tag (proper HTML escape).
- `TestBundleWASM_EmptyWasmBytesReturnsError`: wasmPath points to empty file; returns error.
- `TestBundleWASM_MissingWasmExecJSReturnsError`: wasmExecJSPath doesn't exist; returns error.
- Covers AE3 (WASM .html works offline on phone with no network).

**Verification:** `go test ./pixelforge_studio/buildpipeline/...` passes; manual smoke: build WASM target; open `<game-name>.html` in browser; click splash; game runs; turn off network; refresh; game still loads + runs.

---

### U7. Build workspace + Build pane UI

**Goal:** New dockable `BuildWorkspace` (Name="build", DisplayName="Build"). UI: 5 target checkboxes with per-target status pills, a Build button that kicks off orchestrator with selected targets, an "Open output folder" button per target after success, error display per target on failure.

**Requirements:** R6 (dockable Build workspace), R7 (5 target checkboxes), R8 (parallel builds with status), R9 (Open output folder), R10 (per-target error display).

**Dependencies:** U4 (orchestrator), U5 (icon — invoked by orchestrator), U6 (WASM bundler — invoked by orchestrator).

**Files:**
- `pixelforge_studio/build/workspace.go` (NEW — Workspace impl + RegisterWith)
- `pixelforge_studio/build/workspace_test.go` (NEW)
- `pixelforge_studio/build/status_view.go` (NEW — per-target status pill widget)
- `pixelforge_studio/build/status_view_test.go` (NEW)

**Approach:**
- Standard `Workspace` interface (Name="build", DisplayName="Build", Render(e)).
- **UI layout** (per cimgui-go):
  ```
  imgui.Begin("Build")
    imgui.Text("Targets")
    imgui.Separator()
    for each target in Targets:
      imgui.Checkbox(target.DisplayName, &state.checked[target])
      imgui.SameLine()
      renderStatusView(state.statuses[target])
      if state.statuses[target].Phase == PhaseDone:
        imgui.SameLine(); if imgui.Button("Open"): openOutputFolder(target)
      if state.statuses[target].Phase == PhaseFailed:
        if imgui.Button("Show error"): state.errorExpanded[target] = !state.errorExpanded[target]
        if state.errorExpanded[target]:
          imgui.InputTextMultiline("##err", state.statuses[target].Err.String(), readOnly)
    imgui.Separator()
    if imgui.Button("Build"):
      selectedTargets := filterChecked(state.checked)
      go runBuild(selectedTargets) // starts orchestrator
  imgui.End()
  ```
- **State**: workspace holds `checked map[Target]bool`, `statuses map[Target]BuildStatus`, `errorExpanded map[Target]bool`.
- **runBuild()**:
  1. Gathers selected targets.
  2. Calls orchestrator.Build(req, targets) returning `<-chan BuildStatus`.
  3. Goroutine reads channel; on each event, updates `state.statuses` (under mutex); triggers next-frame re-render.
- **OpenOutputFolder**: shells out to OS-native file manager:
  - Linux: `exec.Command("xdg-open", dirPath)`.
  - macOS: `exec.Command("open", dirPath)`.
  - Windows: `exec.Command("explorer", dirPath)`.
- **Status view widget**:
  - `PhaseQueued`: "—" grey text.
  - `PhaseBuilding`: animated spinner + "Building..." (use imgui.SpinnerDots or text-based).
  - `PhaseDone`: green check + "Done · 12s ago" (timestamp relative to now).
  - `PhaseFailed`: red x + "Failed".
- Multiple concurrent builds permitted; UI doesn't block on Build button click.

**Patterns to follow:** existing Workspace interface from `palette/workspace.go:187-192`; idea #4's plan U6 + U7 for left-right panel pattern; existing imgui.Begin/End structure.

**Test scenarios:**
- `TestBuildWorkspace_RegisteredWithEditor`: after RegisterWith, `e.GetWorkspace("build")` returns non-nil with DisplayName="Build".
- `TestBuildWorkspace_RendersAllFiveTargets`: render workspace; output (stub ImGui sink) contains 5 checkboxes for the 5 targets.
- `TestBuildWorkspace_CheckboxTogglesState`: click Windows checkbox; state.checked[TargetWindows]==true.
- `TestBuildWorkspace_BuildButtonInvokesOrchestrator`: check Windows + Linux; click Build; orchestrator.Build called with both targets.
- `TestBuildWorkspace_StatusUpdatesFromChannel`: simulate orchestrator emitting "Building" then "Done"; state.statuses transitions accordingly.
- `TestBuildWorkspace_OpenButtonAppearsAfterDone`: target status = Done; "Open" button rendered.
- `TestBuildWorkspace_OpenInvokesOSFileManager`: click Open for Linux target; exec.Command called with "xdg-open" + correct path.
- `TestBuildWorkspace_FailedTargetShowsErrorButton`: target status = Failed; "Show error" button rendered.
- `TestBuildWorkspace_ErrorExpandShowsErrorText`: click Show error; expanded text contains error message.
- `TestStatusView_QueuedShowsDash`: phase=Queued; renders "—".
- `TestStatusView_BuildingShowsSpinner`: phase=Building; renders spinner + "Building...".
- `TestStatusView_DoneShowsCheckmarkAndAge`: phase=Done, builtAt 30s ago; renders "Done · 30s ago".
- `TestStatusView_FailedShowsX`: phase=Failed; renders failure indicator.
- Covers AE1 (parallel builds with status), AE7 (Source target ships).

**Verification:** `go test ./pixelforge_studio/build/...` passes; manual smoke: launch studio; View → Build (Ctrl+8); check Linux; click Build; status pill cycles through Building → Done; click Open; file manager opens at `<projectDir>/exports/linux/`.

---

### U8. Build-on-save (debounced, host-only) + chrome status pill

**Goal:** On every project save (`FileMenu.SaveTo`), trigger a background build for the host platform only, debounced 1.5s (so rapid saves coalesce into one build). Status pill in chrome (status bar) shows current state: `Build ready · 2s ago` / `Building...` / `Build failed (click for details)`. Click expands details.

**Requirements:** R24 (background build on save), R25 (debounce 2-3s — using 1.5s per VS Code precedent + brainstorm interval), R26 (status pill in chrome).

**Dependencies:** U4 (orchestrator), U10 (settings.BuildOnSaveDisabled toggle).

**Files:**
- `pixelforge_studio/buildpipeline/build_on_save.go` (NEW — daemon + debouncer)
- `pixelforge_studio/buildpipeline/build_on_save_test.go` (NEW)
- `pixelforge_studio/editor/file_menu.go` (MODIFY — `SaveTo` invokes build-on-save trigger after `ClearDirty()`)
- `pixelforge_studio/editor/imgui_chrome.go` (MODIFY — status pill rendered in status bar)
- `pixelforge_studio/editor/status_pill.go` (NEW — widget rendering)
- `pixelforge_studio/editor/editor.go` (MODIFY — add `BuildState()` accessor)

**Approach:**
- **`BuildOnSaveDaemon` struct**: holds `lastSaveTime time.Time`, `pendingTimer *time.Timer`, `latestStatus BuildStatus`, `mu sync.RWMutex`.
- **`OnSave()`** method called from `file_menu.go` after `ClearDirty()`:
  1. Check `settings.BuildOnSaveDisabled` — if true, no-op.
  2. Cancel previous pendingTimer if set.
  3. Set new timer for 1.5s.
  4. On timer fire: start a background build for host target only (`runtime.GOOS` → Target lookup).
  5. Update `latestStatus` as orchestrator emits events.
- **Cancellation**: if a save fires during a build, the previous build's context is canceled (we want the FRESH source compiled, not a stale snapshot).
- **Status pill rendering** (in chrome's status bar):
  - Idle (no recent build): `Build ready · 12s ago` (green text).
  - Building: `Building...` (yellow text + spinner).
  - Failed: `Build failed (click for details)` (red text + clickable).
  - Click on pill (when failed): opens Build workspace + expands error.
- **Status pill state** lives on `*Editor` (accessor `BuildState() BuildState`); daemon updates it; chrome renders it each frame.
- **CPU priority**: invoked builds use `BELOW_NORMAL` / `nice 10` (per U3) so background builds don't starve UI.
- **Host-only**: build-on-save NEVER cross-compiles (Windows/macOS/WASM builds only fire on explicit Build pane click). Build-on-save target = `runtime.GOOS` mapped to Target enum.

**Patterns to follow:** existing `time.Timer` debounce patterns; existing `*Editor` state access (`editor.go:253-260` for dirty state); existing status-bar render at `imgui_chrome.go:179`.

**Test scenarios:**
- `TestBuildOnSave_DebouncesRapidSaves`: invoke OnSave 5 times in 100ms; only 1 build triggered (after 1.5s of silence).
- `TestBuildOnSave_RespectsDisabledSetting`: settings.BuildOnSaveDisabled=true; OnSave does nothing.
- `TestBuildOnSave_CancelsPreviousBuildOnNewSave`: trigger build; before complete, trigger another save; first build's context is canceled.
- `TestBuildOnSave_HostOnlyTarget`: simulate host=Linux; OnSave triggers build for TargetLinux only (not Windows/macOS/WASM).
- `TestBuildOnSave_OutputPathIsHostExportsDir`: triggered build outputs to `<projectDir>/exports/linux/` (if host=Linux).
- `TestBuildOnSave_UpdatesEditorBuildState`: trigger; after build completes, editor.BuildState().Phase == PhaseDone with BuiltAt timestamp.
- `TestStatusPill_IdleShowsBuildReadyWithRelativeAge`: build completed 5s ago; pill shows "Build ready · 5s ago".
- `TestStatusPill_BuildingShowsSpinner`: phase=Building; pill shows "Building..." + spinner.
- `TestStatusPill_FailedShowsClickPrompt`: phase=Failed; pill shows "Build failed (click for details)".
- `TestStatusPill_FailedClickActivatesBuildWorkspace`: click failed pill; e.ActiveWorkspaceName() becomes "build".
- `TestFileMenu_SaveToInvokesOnSave`: SaveTo successful; OnSave called on daemon.
- Covers AE5 (5 rapid edits coalesce to 1 build), AE6 (suspend toggle prevents background builds).

**Verification:** `go test ./pixelforge_studio/...` passes; manual smoke: edit project; save; status pill shows Building → Done within ~3s; verify `<projectDir>/exports/<host>/<game-name>` is fresh.

---

### U9. Installer packaging scripts

**Goal:** Build script(s) per host platform that assemble a release bundle containing: (1) studio binary, (2) extracted Go SDK at `go-sdk/`, (3) `cart_assets/`, (4) optionally `wasm_exec.js` standalone for documentation. Output: `pixelforge-studio-<host>-<version>.tar.gz` (Linux/macOS) or `.zip` (Windows). Total size ~150-300 MB.

**Requirements:** R30 (vendored Go toolchain in installer), R31 (matching wasm_exec.js).

**Dependencies:** U3 (toolchain discovery expects vendored SDK at `<execDir>/go-sdk/`).

**Files:**
- `scripts/package-installer-linux.sh` (NEW)
- `scripts/package-installer-macos.sh` (NEW)
- `scripts/package-installer-windows.ps1` (NEW)
- `scripts/README.md` (NEW — packaging instructions)
- `Makefile` (MODIFY — add `make package` target invoking the appropriate script for current host)

**Approach:**
- **Linux script** (`package-installer-linux.sh`):
  1. Build studio binary: `go build -o release/pixelforge-studio ./pixelforge_studio`.
  2. Download Go SDK: `wget https://go.dev/dl/go1.24.X.linux-amd64.tar.gz` (version pinned).
  3. Extract: `tar -xzf go1.24.X.linux-amd64.tar.gz -C release/go-sdk`.
  4. Copy cart_assets: `cp -r pixelforge_studio/editor/cart_assets release/`.
  5. Bundle: `tar -czf pixelforge-studio-linux-<version>.tar.gz release/`.
  6. Cleanup release/.
- **macOS script**: similar with darwin-arm64 SDK + zip output.
- **Windows script**: PowerShell equivalent with windows-amd64 SDK + zip output.
- **Pinned Go version**: each script reads the version from `scripts/GO_VERSION` (text file with `1.24.1` or similar); release-coupling discipline is "update GO_VERSION on each studio release."
- **README** documents:
  - How to build a release for the current platform.
  - Cross-platform packaging caveats (Linux can't build macOS bundle; need a Mac runner — out of v1; designers build for their own platform).
  - The expected output directory structure at install time:
    ```
    pixelforge-studio/
    ├── pixelforge-studio          (or pixelforge-studio.exe)
    ├── go-sdk/
    │   ├── bin/go
    │   ├── pkg/
    │   ├── src/
    │   └── lib/wasm/wasm_exec.js
    └── cart_assets/
        └── editor.pforge
    ```

**Patterns to follow:** standard shell-script packaging; Wails/Fyne installer patterns.

**Test scenarios:**
- `TestPackagingLinux_ProducesTarGz`: run package-installer-linux.sh; tarball exists with correct name; contents include pixelforge-studio + go-sdk/bin/go + cart_assets/.
- `TestPackagingLinux_GoSDKVersionPinned`: tarball's go-sdk/bin/go --version matches scripts/GO_VERSION.
- `TestPackagingLinux_WasmExecJSPresent`: tarball contains go-sdk/lib/wasm/wasm_exec.js.
- (macOS/Windows tests are platform-conditional; CI may not run them.)
- Manual verification:
  - Extract tarball to `/tmp/test-pixelforge/`.
  - Run `/tmp/test-pixelforge/pixelforge-studio`.
  - In studio, build Linux target.
  - Verify build succeeds using vendored SDK (not system Go).

**Verification:** `make package` produces a valid bundle; extracting + running the studio + building a small project all work.

---

### U10. View menu integration + settings + keymap

**Goal:** Add View → Build menu entry (Ctrl+8), register Build workspace in main.go, add `BuildOnSaveDisabled bool` to settings with toggle in Settings UI (or just inline at file menu for v1 simplicity).

**Requirements:** R6 (View menu entry for Build), R27 (suspend toggle).

**Dependencies:** U7 (build workspace registered), U8 (build-on-save daemon).

**Files:**
- `pixelforge_studio/editor/file_menu.go` (MODIFY — View menu Build entry; handleShortcuts dispatch)
- `pixelforge_studio/editor/keymap.go` (MODIFY — workspace.build = Ctrl+8)
- `pixelforge_studio/editor/settings.go` (MODIFY — add BuildOnSaveDisabled field; sanitize default false)
- `pixelforge_studio/main.go` (MODIFY — `build.RegisterWith(e)`; start build-on-save daemon)

**Approach:**
- File menu View additions:
  ```
  {Label: "Build", Shortcut: "Ctrl+8", OnSelect: func() { e.SetActiveWorkspaceByName("build") }}
  ```
- Keymap.go: add `Ctrl+8 → "workspace.build"`.
- Handle shortcuts: dispatch `workspace.build` → SetActiveWorkspaceByName("build").
- Settings.BuildOnSaveDisabled (bool, default false — i.e., build-on-save ON by default per R26 "always-fresh artifact").
- Settings UI: a simple checkbox in a Settings dialog under "Build" section (or a Settings → Build sub-menu in the menubar for v1 simplicity).
- main.go: add `build.RegisterWith(e)` alongside other RegisterWith calls (pixelforge_studio/main.go:55-57). Also start the daemon: `daemon := buildpipeline.NewBuildOnSaveDaemon(e); e.SetSaveHook(daemon.OnSave)`.

**Patterns to follow:** existing entries from `file_menu.go:191-196`; existing keymap entries; existing settings field additions.

**Test scenarios:**
- `TestFileMenu_BuildEntryPresent`: View menu contains "Build" entry with Ctrl+8 shortcut.
- `TestFileMenu_BuildActivatesWorkspace`: click View → Build; e.ActiveWorkspaceName() == "build".
- `TestKeymap_Ctrl8MapsToBuildWorkspace`: keymap has Ctrl+8 → workspace.build.
- `TestSettings_BuildOnSaveDisabledDefaultsFalse`: load settings.json without the key; after sanitize, BuildOnSaveDisabled == false (build-on-save enabled).
- `TestSettings_BuildOnSaveDisabledPersisted`: set to true; save; reload; persisted as true.
- `TestMain_BuildWorkspaceRegistered`: after main initialization, e.GetWorkspace("build") non-nil.
- `TestMain_BuildOnSaveDaemonStarted`: daemon's OnSave hook is attached to editor's save flow.

**Verification:** `go test ./pixelforge_studio/...` passes; manual smoke: View menu shows Build; Ctrl+8 opens it; toggle settings to disable; verify no background builds fire on save.

---

### U11. End-to-end build pipeline acceptance tests

**Goal:** Integration tests covering AE1-AE8 + F1-F4. Loads fixtures, invokes orchestrator (or build_on_save), verifies output binaries are produced and runnable where possible.

**Requirements:** R1-R31 covered transitively.

**Dependencies:** U1-U10 all merged.

**Files:**
- `pixelforge_studio/integration_test/build_pipeline_e2e_test.go` (NEW)
- `pixelforge_studio/integration_test/fixtures/tiny_capsule_project.pforge` (NEW — minimal project for fast tests)
- `pixelforge_studio/integration_test/fixtures/mario_strip_full.pforge` (NEW — exercises ideas #1+#2+#3+#4+#5+#6 together)
- `pixelforge_studio/integration_test/fixtures/icon_sprite.png` (NEW)

**Test scenarios:**
- `TestE2E_AE1_BuildBothPlatformsInParallel` (`//go:build long`): orchestrator.Build with [Windows, Linux]; both complete; output files exist at expected paths.
- `TestE2E_AE2_BinaryRunsIdenticallyToEditorPreview` (`//go:build long`): build Linux binary from mario_strip_full fixture; spawn it with `--exit-after 60` flag (or 1-second runtime); verify it exits cleanly + produces same blackboard state as editor would (capture stdout / debug log).
- `TestE2E_AE3_WASMOpensInBrowserOffline`: build WASM; output .html parsed for: contains base64 wasm; contains inline wasm_exec.js; contains click-to-start splash; NO external URLs. (Browser-runtime tests deferred — manual smoke verification.)
- `TestE2E_AE4_WindowsIconReflectsDesignerChoice`: project with IconSpriteName="hero" and a hero sprite; build Windows; inspect .ico inside the .syso (or alongside); contains the hero's PNG bytes at multiple sizes; .syso version info has correct ProductName + FileVersion.
- `TestE2E_AE5_BackgroundBuildDebouncesRapidEdits`: 5 OnSave calls in 1 second; verify only 1 build was actually invoked.
- `TestE2E_AE6_SuspendedBuildOnSaveSkipsBackgroundBuilds`: settings.BuildOnSaveDisabled=true; trigger OnSave; no build fires. Then trigger explicit Build via Build pane; build still runs.
- `TestE2E_AE7_SourceTargetProducesCompleteTree` (`//go:build long`): build Source target; resulting `exports/source/` contains main.go, capsule.go, go.mod, project.pforge, assets/, vendor/; `cd exports/source && go build` succeeds; binary identical (or near-identical) to direct Linux build.
- `TestE2E_AE8_UnsignedWindowsBinaryRequiresSmartScreenBypass`: shape-only — verify .exe lacks signing certificate; document that end-player must click "More info" → "Run anyway". (Manual verification on Windows.)
- `TestE2E_F1_DesignerShipsToClassmate`: end-to-end — build Windows .exe; copy to a different directory (simulating "Discord transfer"); spawn it (if running on Windows) or just verify file integrity + structure.
- `TestE2E_F2_DesignerShipsWASMToBrowser`: build WASM; output .html serves correctly from a local HTTP server (`net/http.FileServer`); browser-test deferred.
- `TestE2E_F3_AlwaysFreshArtifactAfterSave`: edit project; save; wait 2s; verify `<projectDir>/exports/<host>/<game-name>` is fresh (modified time > save time).
- `TestE2E_F4_IconAutoPickedWhenNotMarked`: project with no IconSpriteName; auto-pick returns most-referenced sprite; build Linux (icon doesn't matter for Linux v1) — but verify icon.ResolveIconSprite returned the expected sprite.
- `TestE2E_CapsuleEmbedsAllAssets`: build any target; resulting binary's strings dump contains identifiable bytes from the embedded sprites + audio (use a known marker).
- `TestE2E_VendoredToolchainUsedNotPATH`: spawn studio with mock vendored go binary that writes "VENDORED" to a marker file when invoked; trigger build; marker file written (proves vendored go was called, not system go).
- Covers all 8 AEs + F1-F4.

**Verification:** `go test ./pixelforge_studio/integration_test/... -tags long` passes; manual cross-platform validation:
- On Linux: build Windows + Linux + WASM; verify all 3 outputs.
- On macOS: build all + macOS .app; verify .app bundle structure.
- On Windows: same triad.
- Transfer Windows .exe to a fresh Windows machine; double-click; game runs.

---

## Scope Boundaries

### Deferred to Follow-Up Work

- **Code signing** (Windows Authenticode, macOS notarization, Gatekeeper bypass) — out per origin.
- **Auto-updater for shipped games** — out per origin.
- **Steam / itch.io / app store upload integration** — out per origin.
- **Mobile targets (iOS, Android)** — out per origin.
- **Distributable installers** (.msi, .pkg, .dmg, .deb, .rpm) — out per origin; raw binaries only.
- **Universal macOS binaries** (Intel + Apple Silicon) — out per origin; arm64 only in v1.
- **TinyGo for smaller WASM** — out per origin.
- **Browser hot-reload during dev** — out per origin.
- **Asset compression / minification** — out per origin.
- **Telemetry / analytics in shipped games** — out per origin.
- **DRM / per-cart licensing** — out per origin.
- **Multi-language game export** — out per origin (tied to idea #6 localization).
- **Per-build versioning in output filenames** — out per origin.
- **Source-export ZIP packaging** — out per origin; designer zips manually.
- **Cloud / remote build service** — out per origin.
- **`.desktop` file generation for Linux XDG integration** — out per origin.
- **Custom build configuration** beyond target selection + icon + name + version — out per origin.
- **Pre-/post-build hooks** — out per origin.
- **Build reproducibility** (bit-for-bit identical) — out per origin.
- **Installer for non-host platforms** (Linux build → macOS installer, etc.) — v1 packages installer for current host only; cross-platform installer assembly is a CI/release concern, not the studio's.
- **Hot-reload of vendored Go SDK** — installer ships pinned version; updates require re-downloading studio installer.

### Outside this product's identity

- Telemetry / analytics in shipped games (origin Scope Boundaries).
- DRM / copy protection.
- Cloud / remote build service.

---

## Key Technical Decisions

- **Three new direct dependencies** (per leverage doctrine evaluation): `josephspurrier/goversioninfo` (Windows .syso), `jackmordaunt/icns/v2` (macOS .icns), `Kodeworks/golang-image-ico` (favicon). All pure Go; all cross-platform; total dep weight is small.
- **WASM bundler hand-rolled** (~50 LOC); no mature Go library exists for inline-HTML WASM.
- **Vendored Go SDK strategy**: ship extracted at `<execDir>/go-sdk/` in installer; toolchain helper discovers + uses by absolute path. Fallback to PATH `go` for source-build mode (when no installer was used).
- **Capsule refactor wires the engine for the first time.** Current codegen emits empty games. v1 is when sprites + scenes + audio + behaviors actually run in shipped binaries.
- **Capsule.Run takes optional DataOverride** — editor preview passes in-memory project; shipped binary loads from embedded data. **One contract, both call sites** (R2).
- **Build orchestrator uses goroutines + channel for parallel builds** + `context.Cancel` for cancellation (per `exec.CommandContext`).
- **`CGO_ENABLED=0` for all native targets**. Engine has zero CGO (verified via grep). Linux ships as fully-portable static binary. Resolves brainstorm's outstanding R13 question definitively.
- **`pixelforge_stat` excluded from WASM vendor** — uses gopsutil which requires CGO + has `//go:build !js`. Studio uses it; runtime doesn't. The Capsule's vendor allowlist drops it when target=WASM.
- **macOS arm64 only in v1.** Universal binaries doubles build time + size. Intel users use Rosetta.
- **Output overwrites silently.** Per R29; designer manages version retention via git or copies.
- **Build debounce = 1.5s.** Between VS Code's 1s and brainstorm's 2-3s. Configurable in settings if v2 needs.
- **Background builds run at `nice 10` / `BELOW_NORMAL_PRIORITY_CLASS`.** VS 2022 precedent. UI remains responsive during builds.
- **WASM bundle uses `WebAssembly.instantiate` (not `instantiateStreaming`)** because we're feeding raw bytes, not a fetch Response.
- **Icon auto-pick prefers 16×16** sprites with reasonable ref count over larger sprites. Most NES-class games have a player sprite at this size; good default.
- **Project-level `IconSpriteName` field**, not per-sprite "is_icon" flag — more natural for "one project, one icon"; matches how Bindings reference sprites by name.
- **Source target ships alongside binary targets** — almost free (just calls Capsule.Generate without invoking go build); valuable for designers learning Go.
- **`-ldflags "-H windowsgui"`** for Windows builds to suppress the cmd console window when double-clicking the .exe.
- **`-ldflags "-s -w"`** for native targets to strip debug info (smaller binaries).
- **Toolchain version pinned in `scripts/GO_VERSION`**; studio releases update it explicitly.
- **No installer-packaging unification** (each host platform's installer is a separate script). Cross-platform installer assembly is a CI concern; v1 designers package for their own platform.
- **Capsule's typed accessors are generated as Go constants** for scene IDs, item IDs, etc. Lets programmers using the Source target reference them type-safely.
- **`Project.Version` defaults to current ISO date** if empty (R14). Stamped into binary metadata.
- **No build-config knobs in v1** (debug/release, custom linker flags, build tags). Single-knob build pipeline per origin.

---

## Dependencies / Assumptions

- **Strict dependencies on all prior plans** for the Capsule's runtime wiring (U1):
  - Idea #1's plan (engine renderers `pixelforge_tilemap`, `pixelforge_entity`, `pixelforge_camera`).
  - Idea #4's plan (`pixelforge_audio.Allocator` + `audiolib.LoadCatalog` + audio backend init).
  - Idea #5's plan (`pixelforge_blackboard`, `pixelforge_input`, verb catalog).
  - Idea #6's plan (`pixelforge_dialogue`, `pixelforge_menus`, `pixelforge_save`).
  - If any of those are absent at execution time, Capsule's `Run()` skips the missing system (no-op call) but still produces a runnable binary that runs whatever IS present.
- **Existing codegen infrastructure** (`pixelforge_studio/codegen/generator.go`, `pixelforge_studio/modulepath/`) continues to work; this plan extends rather than replaces.
- **Existing `Workspace` interface + `RegisterWorkspace` pattern** continues to work.
- **Existing settings infrastructure** (`pixelforge_studio/editor/settings.go`) — new field plugs in via established `sanitize` discipline.
- **`os.Executable()` + `filepath.Dir`** for vendored SDK discovery — standard Go stdlib.
- **Ebitengine works under `GOOS=js GOARCH=wasm`** — verified in repo (existing audio backend has `buffer_js.go`).
- **`josephspurrier/goversioninfo`, `jackmordaunt/icns/v2`, `Kodeworks/golang-image-ico`** are all pure-Go, no CGO, cross-platform — usable from any host to produce any target's resources.
- **WASM single-HTML bundle size stays under ~30 MB practical browser threshold** for typical NES-class games (~10-20 MB inlined). Larger games may surface as a follow-up.
- **Browser autoplay policy** is handled by click-to-start splash (R18). Verified by external research — Chrome/Firefox/Safari/Edge all behave the same.
- **Go SDK can be redistributed in the studio installer per BSD license terms** — confirmed.
- **The studio's release lifecycle pins a specific Go version** (e.g., Go 1.24.X) per release. Updates require re-downloading the installer.
- **Linux save files (idea #6) work with `os.UserConfigDir`** in shipped binaries — confirmed (idea #6's plan U1).

---

## Risk Analysis & Mitigation

| Risk | Severity | Mitigation |
|---|---|---|
| Capsule's runtime wiring is incorrect — binary launches but game doesn't run as expected | **High** | Extensive integration tests at U11 (AE2 verifies binary plays identically to editor preview). Manual smoke testing of each subsystem (sprites, audio, save, dialogue) post-build. |
| Vendored Go SDK in installer is the wrong version for cross-compile target | Low | Single Go toolchain cross-compiles all GOOS/GOARCH combos. Pinned version applies to all targets simultaneously. |
| `go-version-info` (goversioninfo) library breaks on Linux producing Windows .syso | Low | External research confirmed cross-platform compatibility; goversioninfo is the canonical tool used by Wails/Fyne for exactly this case. Test scenario `TestGenerateIcons_CrossPlatform` covers. |
| macOS .icns generation breaks on Linux | Low | jackmordaunt/icns/v2 is pure Go, cross-platform. Same test coverage as Windows .syso. |
| WASM bundle exceeds practical browser limits | Medium | Document the ~30 MB practical threshold. v1 ships as-is; v2 could add asset compression if real games hit the limit. |
| Build orchestrator's parallel goroutines deadlock or leak channels | Medium | Use context.Cancel + WaitGroup pattern; tests cover cancellation + concurrent runs. Channel is buffered to avoid blocking on consumer. |
| Build-on-save's debounce timer leaks goroutines on rapid edits | Low | Each timer fires once; timer.Stop() before new timer. Standard pattern. |
| Vendored Go SDK absent in some installer variants; PATH fallback used; designer surprised by "Go not found" error | Medium | U3's ResolveGoBinary returns helpful error message: "Install Go from go.dev OR re-download Pixelforge with bundled toolchain." |
| Studio's existing codegen tests break after refactor | Medium | U1's test suite extends existing tests rather than replacing. Backward-compat for the empty-game smoke test (it should now produce a non-empty game; update assertion). |
| Windows .exe icon doesn't render in Explorer due to .syso format issue | Medium | goversioninfo is well-tested in the Go ecosystem; if issue arises, fallback is `akavel/rsrc` (alternative). Manual cross-platform smoke is essential. |
| Cross-compiling for macOS from Linux/Windows requires Mach-O linker — Go has it built-in (linker is part of toolchain) | Low | Verified — Go's cross-compile is single-toolchain; no external linkers needed. |
| Auto-icon picks the wrong sprite — designer's game ships with "enemy_skeleton" as icon instead of hero | Medium | Auto-pick prefers 16×16 + most-referenced; usually correct. If wrong, designer marks correct sprite via Project.IconSpriteName. Documented in onboarding. |
| Background-build's CPU throttling fails on a platform where priority APIs aren't supported | Low | Platform-specific via build tags; tests cover platform-specific behavior; failure is graceful (build runs at normal priority, no crash). |
| Build-on-save fires while a previous build is in-flight; conflicting outputs to same path | Medium | Build orchestrator's per-target context cancellation ensures only one build per target at a time; second build cancels first. |
| Generated `capsule.go` exceeds Go's source-file size limits for huge projects | Low | Even very large projects shouldn't produce capsule.go beyond ~10K lines; Go's limit is much higher. If hit, split into multiple generated files. |
| WASM build's wasm_exec.js version mismatch with Go SDK breaks runtime | Low | U3's ResolveWasmExecJS uses the SDK's own wasm_exec.js; same SDK → same version → no mismatch. |
| Designer ships unsigned macOS .app to friend; friend can't open it | Medium | Documented onboarding text for designers explaining the Right-click → Open bypass. v2 considers Apple Developer ID. |
| Sourcing wasm_exec.js from `lib/wasm/` on Go 1.24+ but `misc/wasm/` on older versions causes confusion | Low | U3's ResolveWasmExecJS tries both paths in order. |
| Studio binary itself is built with CGO (cimgui-go); the vendored Go SDK doesn't have to know | Low | Studio runs as a normal Go binary; CGO is the studio's concern. Vendored Go just compiles target binaries (which don't need CGO). |
| Coordination with idea #1/#2/#3/#4/#5/#6 plans — Capsule imports their packages; if any package has breaking changes, Capsule breaks | High | Pin engine version per Capsule build (R4). Existing Capsules continue to work even if engine evolves. Re-build to pick up new version. |

---

## System-Wide Impact

**New packages introduced:**
- `pixelforge_studio/buildpipeline/` (orchestrator + toolchain + builders + WASM bundler + icon pipeline + build-on-save).
- `pixelforge_studio/build/` (Build workspace UI).

**Modified packages:**
- `pixelforge_studio/codegen` — generator + templates refactored for Capsule.
- `pixelforge_studio/modulepath` — target-aware shouldVendorDir.
- `pixelforge_project` — adds Version + IconSpriteName fields.
- `pixelforge_studio/editor` — chrome status pill, file_menu Build entry, keymap, settings field, SaveTo hook for build-on-save daemon.
- `pixelforge_studio/main.go` — RegisterWith for build workspace; daemon startup.

**Affected workflows:**
- **Designer authoring** — primary target. New: Build pane in View menu (Ctrl+8); 5 target checkboxes; click Build → see status pills; click Open → file manager opens at output. Plus: passive background build-on-save with status pill in chrome.
- **Engine runtime** — Capsule.Run wires every prior plan's subsystems for the first time. This is the moment the engine actually delivers on the "runs the project" promise.
- **Shipped binary** — produced by Capsule + builders; runs identically to editor preview by construction (one Capsule contract).
- **Designer distribution** — drag the file from `<projectDir>/exports/<target>/` to Discord/itch.io/etc.

**Documentation impact:**
- Post-v1, capture as `docs/solutions/` entries:
  1. Capsule pattern: typed wrapper + DataOverride for one-contract-multiple-call-sites.
  2. Vendored Go toolchain discovery + invocation.
  3. WASM single-HTML bundling discipline.
  4. Cross-platform icon generation pipeline (.ico, .icns, favicon).
  5. Build-on-save debouncer with priority throttling.
- Update existing solution docs:
  - `editor-pforge-schema-shape.md` — note that the Capsule is the canonical load path.
  - The Empty-Capsule legacy main.go template can be archived (cite this plan as the change).
- **Designer onboarding doc**: explain unsigned-binary OS warnings, drag-to-Discord workflow, vendored Go SDK existence (so designers don't try to install Go separately).

**Operational / rollout:**
- **Largest single milestone landing** in the 7-idea release.
- Studio installer with bundled Go SDK is a new release artifact — first time Pixelforge has had one. Establishes the release-engineering surface.
- Designers downloading the new installer get the entire 7-plan capability + ship loop.
- Pre-v1 generated projects' main.go is the empty-game shim; re-generate via Build pane to get the Capsule version.
- Save files (idea #6) work in shipped binaries on all native targets via `os.UserConfigDir`; WASM uses localStorage.

---

## Notes for Implementer

**Coordination with all prior plans:**
1. This plan is the **integration layer** for ideas #1, #2, #3, #4, #5, #6. The Capsule's `Run()` calls each subsystem's init. Coordination order:
   - Execute ideas #1, #2, #3, #4, #5, #6 plans first (they ship their engine + studio packages).
   - Execute this plan's U1 (Capsule refactor) — wires everything together.
   - Execute this plan's U2-U10 — build pipeline + UI + installer.
   - Execute this plan's U11 (E2E tests) — verifies the whole stack.
2. If any prior plan slips, this plan's Capsule still ships — the missing subsystem's hook becomes a no-op call. The Capsule degrades gracefully.
3. **The most subtle bug class**: Capsule importing a prior plan's package that doesn't yet exist. Use `if subsystem-loaded` checks in capsule.go to remain compileable.

**Installer release engineering:**
- v1 ships installer packaging scripts (U9) but the actual installer-build CI workflow is operational, not in this plan. Establish on first release.
- Pinning Go version in `scripts/GO_VERSION` is the release-coupling discipline. Update on each studio release.

**Cross-platform development:**
- This plan adds the FIRST need to test on Windows + macOS during development. Existing tests run on Linux; this plan's Windows/macOS-specific code (priority APIs, .syso embedding, .icns generation) needs platform-conditional CI.
- Recommendation: GitHub Actions matrix build (linux/macos/windows runners) for `-tags long` tests.

**Sourcing the bundled Pixelforge icon (U5):**
- Create a simple 16×16 PNG with Pixelforge branding (a stylized pixel-art "P" or game-controller motif). Embed via `//go:embed cart_assets/default_icon.png`. Manual asset task; ~30 min of pixel art.
- This is the fallback when project has no sprites OR when designer hasn't marked one.
