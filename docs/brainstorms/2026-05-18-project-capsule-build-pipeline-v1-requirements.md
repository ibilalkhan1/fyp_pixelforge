---
date: 2026-05-18
topic: project-capsule-build-pipeline-v1
origin: docs/ideation/2026-05-18-pixelforge-nes-class-no-code-ideation.md (idea #7)
satisfies_dependencies:
  - docs/brainstorms/2026-05-18-screenroom-mario-strip-v1-requirements.md (idea #1 R12)
  - docs/brainstorms/2026-05-18-audio-library-picker-v1-requirements.md (idea #4 R8)
  - docs/brainstorms/2026-05-18-rpg-class-systems-v1-requirements.md (idea #6 R2)
---

# Project Capsule + Build Pipeline — v1

## Summary

v1 ships the complete ship loop: codegen refactored to emit a Project Capsule (typed wrapper that loads any `.pforge` with all assets embedded via `go:embed`), a dockable Build pane with five target checkboxes (Windows `.exe` / macOS `.app` / Linux binary / WASM single-HTML / Source), per-platform builds invoked through a Go toolchain vendored inside the studio (~150-300 MB installer, no separate Go install), WASM bundled as a single `.html` (inline `wasm_exec.js` + base64-encoded `.wasm` + click-to-start splash), auto-icon generated from a designer-marked or auto-picked project sprite (`.ico` / `.icns` / favicon), and background build-on-save producing an always-fresh host-platform binary in `<project-dir>/exports/<host>/`.

---

## Problem Frame

Pixelforge's codegen today emits a Go source tree (`main.go` + `go.mod` + `project.pforge` + `assets/`) with a thin shim and a vendored engine. The shipped artifact is **source code**. A designer in the target audience — friends, classmates, not pre-trained on Go — cannot ship that. They don't have `go` on their PATH; they don't know what `go build` means; they certainly can't hand a directory of Go files to a classmate and expect a playable game.

Six earlier brainstorms (idea #1 ScreenRoom, #4 Audio Library, #6 RPG-class) all explicitly depend on a binary existing. Without a build pipeline, every authoring feature is wasted:
- A Mario-strip level can be painted, previewed, played in the editor — but never handed to a friend.
- An audio library can be browsed and bound — but binding stops at preview, no shippable artifact carries the audio.
- A save system can serialize the blackboard at design time — but there's no game-shaped binary writing to the user-config dir.

The user's stated requirements (from the first brainstorm session) were unambiguous: **single-file double-clickable binary per platform**, **WASM in the browser offline like the Chrome Dino game**, **custom icon from a project sprite**, **"just like retro ROMs."** Three of those four — single-file, WASM, custom icon — are completely absent from the current codegen pipeline.

This brainstorm scopes the ship pipeline that makes every prior brainstorm useful end-to-end. Without it, the Pixelforge "no code, ship a complete game" promise collapses at the last step.

---

## Actors

- **A1. Designer.** Authors the game in the studio, clicks Build, drags binaries to Discord. Doesn't install Go, doesn't run `go build`, doesn't open a terminal. Maybe marks one sprite as the game's icon.
- **A2. End-player (classmate).** Receives a single file from the designer (`.exe`, `.app.zip`, Linux binary, or `.html`). Double-clicks. Plays the game. Saves progress to their local machine.
- **A3. Pixelforge Studio.** Hosts the Build pane; manages the vendored Go toolchain; invokes per-platform builds in parallel; runs background build-on-save; generates icons from project sprites.
- **A4. Vendored Go toolchain.** Shipped inside the studio installer (~150-300 MB). Used for all build invocations across all targets (Go's cross-compile is single-toolchain).
- **A5. Project Capsule.** Generated `capsule.go` + embedded `.pforge` + embedded assets. The runtime contract every build target imports.

---

## Key Flows

- **F1. Designer ships a build of their game to a classmate**
  - **Trigger:** Designer finishes authoring, wants their friend on Windows to play
  - **Actors:** A1, A3, A4, A2
  - **Steps:** (1) Designer opens the Build pane; (2) checks "Windows .exe"; (3) clicks Build; (4) per-target status pill shows "Building..." then "Done"; (5) "Open output folder" button drops them in `<project-dir>/exports/windows/`; (6) designer drags `<game-name>.exe` into a Discord chat with their friend; (7) friend on Windows downloads, double-clicks, plays
  - **Outcome:** A binary native to the target OS exists, plays standalone, requires no Go install on the end-player's machine.
  - **Covered by:** R6, R7, R8, R11, R12, R28, R30

- **F2. Designer ships a WASM/browser version**
  - **Trigger:** Designer wants anyone with a browser to play their game without downloading anything
  - **Actors:** A1, A3, A4, A2
  - **Steps:** (1) Designer checks "WASM" in the Build pane; (2) clicks Build; (3) studio compiles to `game.wasm` then assembles inline-HTML; (4) output is `<project-dir>/exports/wasm/<game-name>.html`; (5) designer uploads the single `.html` to itch.io / their own webpage / Discord; (6) friend opens the URL in any modern browser; (7) browser loads the `.html`, presents a "Click to start" splash; (8) friend clicks, game runs in browser, plays offline (no network calls)
  - **Outcome:** A single `.html` file is the entire game; works in any browser, online or offline, on any platform with a browser (desktop / mobile / tablet).
  - **Covered by:** R6, R8, R16, R17, R18, R19, R28

- **F3. Designer iterates with always-fresh artifact**
  - **Trigger:** Designer is authoring + iterating + wants to test the actual built binary, not just the studio preview
  - **Actors:** A1, A3
  - **Steps:** (1) Designer saves a change; (2) studio's status pill changes to "Building..."; (3) after a couple seconds, status pill shows "Build ready · 2s ago"; (4) designer can drag the file from `<project-dir>/exports/<host>/<game-name>` to a chat at any moment, or run it directly to test outside the studio
  - **Outcome:** The shippable artifact is always within seconds of the latest save; designer never has to remember to "build before sharing."
  - **Covered by:** R24, R25, R26, R27

- **F4. Designer customizes the game icon from a project sprite**
  - **Trigger:** Designer's game has a distinctive player sprite they want as the icon
  - **Actors:** A1, A3
  - **Steps:** (1) Designer opens the player sprite's inspector; (2) toggles "Use as game icon"; (3) on next build, the binary on all platforms shows that sprite as its icon (taskbar on Windows, dock on macOS, browser tab on WASM)
  - **Outcome:** Shipped games look like real shipped products from launch; designer never opens an icon-generation tool.
  - **Covered by:** R20, R21, R22, R23

---

## Requirements

**Project Capsule architecture**

- R1. Codegen emits two meaningful files per build: `project.pforge` (game data, unchanged from current) and `capsule.go` (a generated, gofmt'd, hand-readable Go file providing a `func Run(opts CapsuleOpts) error` entry point + typed accessors for every component / scene / item / dialogue / menu / verb-binding). `main.go` shrinks to ~6 lines: `func main() { capsule.Run(capsule.Defaults()) }`.
- R2. The Capsule **loads any `.pforge`** that conforms to the project schema — so the **same Capsule runtime** drives the editor's live preview, the shipped binary, regression tests, and the bundled Forgequest tutorial cart if/when that lands. One contract, all call sites.
- R3. The Capsule **embeds the project's assets** (`.pforge` + audio-assets/ + sprite-assets/ + any other asset directories) via `go:embed` at build time. The shipped binary has **zero filesystem dependencies** for game content. (Save files written at runtime go to the user-config dir per idea #6, not the binary.)
- R4. The Capsule pins a **specific vendored version of the Pixelforge engine + Ebitengine** per build. Projects built today continue to work with that engine version after the engine evolves; future engine breakage doesn't retroactively brick existing builds.
- R5. The Source target output is a **complete Go project tree** the designer can `go build` themselves with their own toolchain — useful for designers learning Go, for code review, for ports.

**Build pane UI**

- R6. A new **dockable Build workspace** appears alongside Scene / Inspector / Assets / Capture / Behavior / Palette / Audio / Dialogue / Items / Menus. Same dockable-panel + ImGui pattern.
- R7. The Build pane lists **five target checkboxes**: Windows `.exe`, macOS `.app`, Linux binary, WASM single-HTML, Source. Designer checks any subset.
- R8. The **Build button** kicks off builds for checked targets **in parallel**. Each target shows its own status indicator: queued / building (with spinner) / done (with timestamp) / failed (with click-for-error-details).
- R9. **"Open output folder"** button appears next to each "done" target; clicking opens the OS file browser at `<project-dir>/exports/<target>/`.
- R10. **Per-target error display**: a failed build expands to show the full error output (compiler errors, linker errors, asset embedding errors) in a scrollable text view. Designer can copy/paste for help.

**Native build targets**

- R11. **Windows target**: `GOOS=windows GOARCH=amd64`. Icon + version metadata embedded via a `goversioninfo`-style `.syso` file linked into the build. Output: `<game-name>.exe`. Single-file double-clickable on Windows 10+.
- R12. **macOS target**: `GOOS=darwin GOARCH=arm64` (Apple Silicon only in v1; Intel users emulate via Rosetta). Output is an `.app` bundle directory (`Contents/MacOS/<binary>` + `Contents/Resources/AppIcon.icns` + `Contents/Info.plist`), zipped for distribution. Single-zip distribution.
- R13. **Linux target**: `GOOS=linux GOARCH=amd64`. Output: single binary file, executable bit set. No `.desktop` entry or XDG integration in v1 (Linux user runs the binary directly).
- R14. **Game name** comes from `Project.Name`; **version** stamped from current ISO date (e.g., `2026-05-18`) or a designer-set `Project.Version` field. Both surface in binary metadata (Windows `.exe` properties, macOS `Info.plist`).
- R15. Native targets ship **unsigned**. Windows SmartScreen + macOS Gatekeeper will warn end-players on first launch ("unrecognized publisher" / "from an unidentified developer"). v1 documents this; code signing deferred.

**WASM single-HTML target**

- R16. **WASM build**: `GOOS=js GOARCH=wasm go build -o game.wasm`. Studio matches the right `wasm_exec.js` shipped with the same Go version the toolchain uses.
- R17. Studio **assembles one `<game-name>.html`** containing: inline `<script>` of `wasm_exec.js`, base64-encoded `<script>` of `game.wasm`, and a thin HTML bootstrap that instantiates the WASM module. Single file, double-clickable into a browser, no auxiliary files.
- R18. The HTML includes a **"Click to start" splash** to satisfy browser autoplay policy (audio requires user gesture). On click, the splash disappears and the engine initializes. Splash is plain, branded with the project's game name.
- R19. WASM games **work offline** — the `.html` carries everything; no network requests at runtime. Hosting on a static URL works the same as opening the file locally via `file://`.

**Auto-icon generation**

- R20. Designer can **mark one sprite as the game icon** via a toggle in the sprite inspector ("Use as game icon" checkbox). Only one sprite per project can be marked.
- R21. If no sprite is marked, the studio **auto-picks the most-used 16×16 sprite** (heuristic: highest reference count across scenes + entity instances). Tie-breaker: first sprite alphabetically.
- R22. If no usable sprite exists (project has no sprites), the studio falls back to a **default Pixelforge icon** shipped inside the studio installer.
- R23. At build time, the studio generates **per-platform icon formats** from the selected sprite:
  - Windows `.ico` with 16 / 32 / 48 / 256 px sizes
  - macOS `.icns` with corresponding scales
  - WASM `.ico` favicon embedded in the HTML
  - Linux: skipped in v1 (no XDG integration per R13)

**Build-on-save (always-fresh artifact)**

- R24. On every meaningful project save, the studio kicks off a **background build for the host platform only** (the OS the studio is running on). Cross-platform + WASM targets only build on explicit Build-pane invocation.
- R25. Background builds are **debounced 2-3 seconds** after the last save — rapid edits coalesce into one build.
- R26. A **status pill** in the studio's status bar shows current build state: `Build ready · 2s ago` / `Building...` / `Build failed (click for details)`. Click expands details inline.
- R27. Designer can **suspend background builds** via a Settings → Build toggle (for low-CPU machines or focus mode). Explicit Build-pane invocations still work when suspended.

**Output paths**

- R28. Builds output to `<project-dir>/exports/<target>/`:
  - `<project-dir>/exports/windows/<game-name>.exe`
  - `<project-dir>/exports/macos/<game-name>.app/` (directory; also `<game-name>.app.zip` for distribution)
  - `<project-dir>/exports/linux/<game-name>` (single binary, executable)
  - `<project-dir>/exports/wasm/<game-name>.html` (single file)
  - `<project-dir>/exports/source/` (Go project tree)
- R29. Outputs **overwrite previous builds** silently — no per-build versioning in the file path. Designer who wants to keep old versions copies them out manually.

**Toolchain dependency**

- R30. The studio installer **vendors a Go toolchain for the studio's host platform** (~150-300 MB installer). Designer never installs Go separately. Cross-compilation uses Go's built-in single-toolchain cross-compile (`GOOS=X GOARCH=Y go build`).
- R31. The studio installer also bundles **the matching `wasm_exec.js`** for the vendored Go version so WASM builds always have the right runtime shim.

---

## Acceptance Examples

- **AE1. Covers R7, R8, R11.** Given a designer opens the Build pane in a project named "MyMarioClone" on a Linux machine and checks Windows .exe + Linux binary, when they click Build, both targets build in parallel. After both complete, `<project-dir>/exports/windows/MyMarioClone.exe` and `<project-dir>/exports/linux/MyMarioClone` exist; both are runnable on their respective OSs.
- **AE2. Covers R3, R28.** Given a designer ships a Linux binary from a project with 12 sprites + 4 audio patches + 1 dialogue tree + 5 items, when the end-player runs the binary on a Linux machine with no project directory or assets folder present, the game runs identically to the editor preview. No "file not found" errors; no missing audio; no missing sprites.
- **AE3. Covers R16, R17, R18, R19.** Given a designer ships a WASM build, when they upload `<game-name>.html` to itch.io and open it on a phone browser with the device in airplane mode (no network), the page loads from the browser cache, shows a "Click to start" splash, and on click runs the game with audio.
- **AE4. Covers R20, R21, R22, R23.** Given a project where the designer has not marked any sprite as the icon and the most-used 16×16 sprite is "hero_idle", when the designer ships a Windows .exe, the file's icon in Explorer shows the hero_idle sprite. The .exe properties show the game name and version. Right-click → Properties confirms.
- **AE5. Covers R24, R25, R26.** Given the designer makes 5 rapid edits within 1 second, when 2-3 seconds pass after the last edit, the studio's status pill changes from "Building..." to "Build ready · just now". Only one build was triggered (not five), and the host-platform binary at `<project-dir>/exports/<host>/<game-name>` reflects all 5 edits.
- **AE6. Covers R27.** Given the designer suspends background builds via Settings, when they make 10 changes over a minute, no build runs in the background. Clicking the Build pane's explicit Build button still produces a fresh binary on demand.
- **AE7. Covers R5.** Given a designer enables the Source target in the Build pane, when the build completes, `<project-dir>/exports/source/` contains a complete Go project tree (`main.go`, `go.mod`, `go.sum`, `capsule.go`, `project.pforge`, vendored engine in `vendor/`, embedded assets in `assets/`). Running `go build` from inside this directory produces a binary identical to the studio's native-target output.
- **AE8. Covers R15.** Given an end-player on Windows 10 downloads an unsigned `.exe` from Discord and double-clicks it, when SmartScreen shows the "Windows protected your PC" warning, the player clicks "More info" → "Run anyway" and the game launches. Same flow on macOS Gatekeeper.

---

## Success Criteria

- **Designer outcome (single platform):** A designer who has authored a game in the studio clicks one checkbox in the Build pane + one Build button, and within seconds has a single file (the host-OS binary) ready to drag into Discord. Their classmate on the same OS double-clicks the file and plays the game with no install steps, no Go runtime, no auxiliary files.
- **Designer outcome (cross-platform):** A designer on Linux ships builds for Windows + macOS + Linux + WASM by checking four boxes and clicking Build once. Each binary works on its respective OS or in any browser. They never install Go, never type a command, never see a compile error unless their game has one.
- **Designer outcome (iteration):** A designer iterating on a game sees the binary at `<project-dir>/exports/<host>/<game-name>` always within ~3 seconds of their latest save. Shipping is "drag the file" — never "rebuild then drag."
- **Player outcome:** An end-player who receives the shipped artifact opens it without seeing Pixelforge anywhere — no studio install, no "powered by" splash, no first-run download prompts. The game appears as a polished standalone artifact with its custom icon visible in the OS chrome / browser tab.
- **Studio install outcome:** A first-time designer downloads the Pixelforge installer (~150-300 MB), runs it, opens the studio, and within their first session has built a Windows binary of their game without ever encountering the words "Go" or "compiler."
- **Downstream handoff outcome:** Planning consumes this doc and does not need to invent Capsule API shape, Build pane UX, per-platform build invocations, WASM bundling mechanism, icon generation pipeline, or build-on-save semantics. Only implementation specifics (exact `go-version-info` library choice, exact ImGui Build pane layout, exact background-build CPU throttling) are open.

---

## Scope Boundaries

- **Code signing.** Windows Authenticode certificates, macOS notarization, Gatekeeper bypass — all out of v1. Designers ship unsigned binaries; OS warnings on first launch are accepted as the cost. Documentation explains.
- **Auto-updater for shipped games.** Designer ships a new build, end-players replace the file. No update-in-place mechanism in v1 or v2.
- **Steam / itch.io / app store upload integration.** Designer uploads to distribution platforms manually. v1 stops at producing the artifact.
- **Mobile targets (iOS, Android).** Ebitengine supports both, but cross-compilation + app-store packaging is its own multi-month surface. Out of v1; likely out of product identity in the medium term.
- **Distributable installers** (.msi for Windows, .pkg / `.dmg` for macOS, .deb / .rpm for Linux). Raw binaries only in v1.
- **Universal macOS binaries** (Intel + Apple Silicon in one `.app`). v1 ships Apple Silicon (arm64) only; Intel Mac users emulate via Rosetta. Universal-binary support deferred.
- **TinyGo for smaller WASM output.** Standard Go's WASM works; TinyGo has Ebitengine compatibility issues. v1 picks robustness over binary size.
- **Browser hot-reload during development.** Studio preview is the iteration loop; WASM build is for shipping. No Vite-style WASM hot-swap.
- **Asset compression / minification beyond Go's defaults.** No texture atlasing pass, no audio bitrate reduction, no PNG re-encoding. Designer's assets ship as-is.
- **Telemetry / analytics in shipped games.** Games don't phone home. Out of product identity.
- **DRM / per-cart licensing / copy protection.** Games are free to share verbatim.
- **Multi-language game export.** Tied to idea #6's localization deferral; out.
- **Per-build versioning in output filenames** (e.g., `<game-name>-v2.exe`). Designer who wants version-tagged outputs renames manually.
- **Source-export ZIP packaging.** Source target emits a directory; designer zips it themselves to ship.
- **Cloud / remote build service.** All builds run locally on the designer's machine via the vendored toolchain.
- **`.desktop` file generation for Linux** (XDG icon registration). Linux ships raw binary; XDG entry is a polish for later.
- **Custom build configuration** beyond target selection + icon sprite + game name + version. No debug-vs-release flags, no custom linker flags, no build tags. One-knob build pipeline.
- **Pre-build / post-build hooks** (designer-supplied scripts that run before or after compilation). Out of v1.
- **Build reproducibility** (bit-for-bit identical binaries across machines / time). Not a goal for v1; Go's build determinism is good but the studio doesn't enforce it.

---

## Key Decisions

- **Full ship vision in v1, not phased.** Earlier brainstorms (#1, #4, #6) explicitly depend on this. The user's stated requirements (WASM, single-file, custom icon) cover everything in the cut. Phasing would leave the entire ship loop incomplete for months.
- **Vendor Go inside the studio installer.** The audience (designer + classmates, not pre-trained on Go) cannot be asked to install Go separately. ~150-300 MB installer is the right cost. Matches Pico-8 / Construct 3 install-size precedent.
- **Project Capsule = thin wrapper, not generated source.** Codegen emits one typed wrapper (`capsule.go`) + the data (`.pforge`). Game logic lives in the vendored engine; capsule.go is the only generated code per project. Matches the "thin shim + .pforge" decision from the earlier ImGui migration plan.
- **Single Capsule contract for editor preview + shipped binary + tests.** One runtime path. Reduces "works in editor but not in shipped game" bugs to zero by construction.
- **WASM via inline-HTML, not three-file deployment.** Standard Go WASM ships `.wasm + wasm_exec.js + .html` separately; designer would have to deploy three files. Inline-HTML produces one file, matching the "single artifact, drag-to-share" UX of native targets. Pico-8's `.html` export sets the bar.
- **Click-to-start splash for WASM.** Browser autoplay policy requires user gesture before audio plays. Splash is the correct UX answer — designers don't have to know browsers have this rule.
- **Background build-on-save, host platform only.** Always-fresh artifact for the most common case (designer iterating on their own machine). Cross-platform + WASM are explicit because they're slower and only matter at ship time.
- **Auto-icon defaults to most-used sprite.** Sensible default for designers who don't think about icons. Marking a specific sprite as "use as icon" is the override.
- **macOS Apple Silicon only in v1.** Universal binaries double build time + binary size. Intel users can use Rosetta. Will revisit when Apple drops Rosetta.
- **No code signing in v1.** Cost (certs, Apple developer account) + setup (per-designer signing keys) is significant friction. Accept the OS warnings; document.
- **No installers, no app stores.** Designer ships raw artifacts; distribution platform is the designer's choice (Discord, itch.io, personal site). v1 stops at producing the artifact.
- **Outputs overwrite silently.** Save/version management is the designer's responsibility (or git's). Studio doesn't accumulate old builds.
- **Source target ships alongside binary targets.** Designers curious about Go, code reviewers, and tinkerers can use the Source export. Doesn't cost much to maintain since it's the same template, just without the build step.

---

## Dependencies / Assumptions

- **Depends on the existing codegen template** in `pixelforge_studio/codegen/generator.go` as the starting point for the Capsule refactor. Current `main.go` + `go.mod` + `project.pforge` + `assets/` becomes `main.go` (6 lines) + `capsule.go` (generated wrapper) + `project.pforge` + assets embedded via `go:embed`.
- **Depends on Ebitengine WebAssembly support** (`GOOS=js GOARCH=wasm`). Verified to exist; planning confirms current Ebitengine version compatibility.
- **Depends on `go-version-info`-equivalent library** for Windows `.exe` icon + metadata embedding. Such libraries exist in the Go ecosystem (`goversioninfo` is the canonical one). Planning picks the specific dependency.
- **Depends on the studio's existing dockable workspace pattern** (post-ImGui migration). The Build pane registers via the same `editor.Workspace` interface as Scene / Inspector / etc.
- **Assumes the vendored Go toolchain** can be redistributed inside the studio installer per Go's license terms (BSD-style, allows redistribution). Confirmed.
- **Assumes `goversioninfo`** (or equivalent) and the icon-generation pipeline (PNG → multi-resolution .ico, .icns) can run cross-platform on the studio's host (e.g., generating Windows .ico from a Linux machine). Most tooling supports this; planning verifies.
- **Assumes WASM single-HTML bundle size** stays under browser limits (a few MB to a few dozen MB). For a typical NES-class game with ~30 SFX + 4 BGM loops + 50 sprites, expect 10-30 MB. Browsers handle this fine. Planning measures real builds to verify.
- **Assumes browser autoplay policy** is handled by the click-to-start splash. Chrome / Firefox / Safari / Edge all behave the same way (require user gesture before audio); the splash satisfies all of them.
- **Assumes save files (idea #6)** work correctly across all build targets. Native builds use Go's `os.UserConfigDir`. WASM uses `localStorage` (per idea #6's deferred question). Planning verifies WASM save paths.
- **Save file paths cross-platform**: native targets use Go's standard cross-platform user-config-dir resolution; WASM uses browser `localStorage`. Game state cannot transfer between platforms in v1 (Windows save ≠ macOS save ≠ WASM save).

---

## Outstanding Questions

### Resolve Before Planning

- *(none — scope is fully resolved at the brainstorm level)*

### Deferred to Planning

- **[Affects R1, R2] [Technical]** Exact `CapsuleOpts` shape — what configuration knobs the runtime needs (window size, fullscreen flag, debug overlay toggle, etc.). Planning enumerates from the existing engine's options and from what shipped games may need.
- **[Affects R6] [Technical]** Build pane layout in the dockable workspace — single column of target checkboxes vs grouped (native / web / source) vs expandable per-target settings. Planning picks based on visual fit.
- **[Affects R11] [Technical]** Exact `goversioninfo` (or equivalent) library choice for Windows icon + version metadata. Candidates: `goversioninfo`, `rsrc`, `akavel/rsrc`. Planning picks based on maintenance status + license.
- **[Affects R12] [Technical]** Exact `.icns` generation tool / library — macOS `iconutil` is native; cross-platform requires a pure-Go alternative or an embedded native tool. Planning resolves.
- **[Affects R17, R18] [Technical]** Exact HTML bootstrap template + WASM size limits per browser. Planning measures real builds; if size exceeds a browser limit, planning adds a "WASM streaming" workaround.
- **[Affects R20, R21] [Technical]** Exact "most-used sprite" heuristic — count references in scenes? in entity instances? in tile palettes? Planning specifies. Tie-breaker rules also need pinning.
- **[Affects R23] [Technical]** Exact resolution-set per platform icon format (Windows .ico typical sizes: 16/32/48/256 — but some include 64/128 too; macOS .icns: 16/32/64/128/256/512/1024). Planning picks based on common practice.
- **[Affects R25] [Technical]** Exact background-build debounce timer + CPU throttling. 2-3 seconds is the target feel; planning measures perceived responsiveness and picks the right value.
- **[Affects R30] [Technical]** Exact Go toolchain version vendored. Pin to a specific version (e.g., Go 1.24.x at studio release time) and updated with studio releases. Planning specifies release-coupling discipline.
- **[Affects R30] [Technical]** How to distribute the vendored Go toolchain inside the studio installer. Candidates: extract on first run; carry as a sibling directory; embed via Go's own `go:embed` (recursive — Go embeds Go). Planning picks.
- **[Affects R15] [Needs research]** End-player documentation for OS warnings on unsigned binaries. v1 ships unsigned; designers will hear "my friend says they can't open it." Planning surfaces what onboarding text the studio should show to help designers explain to their classmates.
- **[Affects R13] [Needs research]** Whether Linux binaries should ship as static binaries (CGO_ENABLED=0) or dynamically linked. Static is more portable across distros but larger; Ebitengine may require CGO for audio/input on some Linux configs. Planning verifies.
- **[Affects R17] [Needs research]** Whether WASM save files (via browser `localStorage`) tolerate the size of full RPG-class game state. `localStorage` is typically 5-10 MB per origin; saves should be small but planning measures.
