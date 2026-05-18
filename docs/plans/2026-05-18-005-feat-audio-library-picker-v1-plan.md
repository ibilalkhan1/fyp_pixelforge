---
title: "feat: Audio Library Picker + Bindings Table + Paula allocator (idea #4 v1)"
type: feat
status: active
date: 2026-05-18
depth: deep
origin: docs/brainstorms/2026-05-18-audio-library-picker-v1-requirements.md
ideation: docs/ideation/2026-05-18-pixelforge-nes-class-no-code-ideation.md (idea #4)
ships_with:
  - docs/plans/2026-05-18-002-feat-screenroom-mario-strip-v1-plan.md (idea #1)
  - docs/plans/2026-05-18-003-feat-tileatlas-emergent-rules-v1-plan.md (idea #2)
  - docs/plans/2026-05-18-004-feat-nes-palette-art-director-v1-plan.md (idea #3)
related_plans:
  - docs/plans/2026-05-18-001-feat-entity-verb-sheet-v1-plan.md (idea #5 — pievent registry + Bind triggers from verb-sheet)
---

# feat: Audio Library Picker + Bindings Table v1 (idea #4)

## Summary

v1 ships four coupled deliverables: (1) a **bundled audio library** of ~30 SFX + 4-5 BGM 8-bit-mono-PCM WAVs (Paula-compatible per `DecodeWavOrErr` constraints), embedded via `go:embed` directory pattern; (2) a new **dockable Audio workspace** registered via the existing `Workspace` interface (mirrors `pixelforge_studio/palette/workspace.go:187-192`), with two panels — library picker (left, categorized by role) + bindings table (right); (3) a new **`pixelforge_audio.Allocator`** that reads `AudioSample.SuggestedChannelPriority` and routes playback (BGM → Chan1/Chan2 lock; SFX → Chan3/Chan4 round-robin via `ChannelActive` query) — this is **new code** because the brainstorm's "engine's existing auto-allocator" claim is aspirational, and without it the runtime can't route bound samples; (4) a new **`audio.Import` pipeline** mirroring `palette.Import` (file picker → DecodeWavOrErr format-gate → copy into `<project>.pforge-assets/audio/<basename>.wav` → append AudioSample), used by both user-imported WAVs (via new `File → Import WAV…` menu entry) and library "Bind..." button (which materializes the embedded WAV bytes into a temp file or passes via `[]byte` overload). Plus: editor's missing Paula backend init in `pixelforge_studio/main.go`. **Drag-drop is deferred** per established cimgui-go convention (per ImGui migration plan U7 and idea #5 plan U7); v1 uses per-row "Bind..." button + "Pick sound..." overlay (modal-stack participant per `focus-manager-design.md`). Event-topic dropdown sources from `pievent.EnumerateTargets()` with free-text fallback. Zero external dependencies for code; sound-design source is sfxr/Bfxr/jsfxr presets exported to WAV (CC0 fallback for BGM where sfxr falls short) — implementer can override if a different sourcing path is preferred. **No schema changes.** Ship-loop dependency on idea #7's plan (R8) for the cross-machine demo, but v1 of this plan is testable end-to-end via the editor preview.

---

## Leverage Doctrine (applied)

Per `docs/plans/2026-05-18-001-feat-entity-verb-sheet-v1-plan.md`'s Leverage Doctrine appendix.

**Candidates evaluated:**

| Candidate | Status | Verdict |
|---|---|---|
| Go audio playback libraries (oto, beep, malgo) | Mature | **Skip** — existing `pixelforge_audio` is the project's deliberately Paula-styled mixer; introducing a generic audio library bypasses the "NES-class" identity and the existing `Backend` abstraction (`pixelforge_audio/backend.go:5`) is already structured for runtime swap. The studio just needs to call the same backend the runtime calls. |
| sfxr / Bfxr / jsfxr Go ports for runtime synthesis | A few exist (`jpswamy/sfxr`, etc.) | **Use as asset-production tool, NOT a runtime dependency.** Per brainstorm decision: patches are pre-rendered WAVs, no runtime synthesis. Use sfxr in the asset-curation pipeline to PRODUCE the bundled library, then ship just the WAVs. Zero runtime dependency. |
| ImGui drag-drop libraries / wrappers | None for cimgui-go | **Defer drag-drop** entirely per established convention. Button-as-primary. |
| Audio decode / WAV parse libraries | `audio/wav` from x/exp, etc. | **Skip** — existing `pixelforge_audio.DecodeWavOrErr` (`pixelforge_audio/decode.go:34-110`) is the strict 8-bit-mono-PCM gate the library WAVs must pass through. Existing tests at `pixelforge_audio/decode_test.go` already validate format constraints. Wrapping a generic decoder would require re-deriving the strict gate. |

Total custom: ~50 LOC allocator + ~80 LOC import pipeline + ~120 LOC workspace + ~80 LOC bindings table + ~60 LOC sound-picker overlay + ~30 LOC audition + supporting tests. Well below wrap costs.

---

## Problem Frame

Pixelforge's audio story has the bones in place and zero user-facing surface:

- **Engine has a working Paula mixer** (`pixelforge_audio/piaudio.go:Play(ch, sample, pitch, vol)`). Channels Chan1/Chan2/Chan3/Chan4 are constants. `LoadSample` / `UnloadSample` / `ChannelActive` / `ChannelPosition` are all there.
- **Engine has schema reservations** for routing: `AudioSample.SuggestedChannelPriority string` ("bgm"|"sfx"|"voice"|"ambient") + `AudioBinding.ForceChannel int` (1-4; 0 = auto). The brainstorm cites an "existing Paula auto-allocator using SuggestedChannelPriority to route playback." **The allocator does not exist.** Grep across the repo: the field is referenced only in schema definitions and doc comments. Without an allocator, every `Play` call requires a hard-coded channel choice — the runtime has no way to honor the schema's routing intent.
- **Studio has no audio surface.** Asset browser (`asset_browser.go:36-49`) renders `p.Audio` entries as read-only rows. No audition button. No `audio.Import` function (compare `palette.Import` at `import_pipeline.go:40-107`, which is the established pattern). The File menu (`file_menu.go:166-204`) has neither `Import PNG…` (per idea #3's plan) nor `Import WAV…`. The empty-state hint says "File → Import to add" but the menu entry doesn't exist.
- **Studio doesn't even init the audio backend.** `pixelforge_studio/main.go:31-62` never sets `piaudio.Backend`; default is `panicBackend{}` (`pixelforge_audio/backend.go:5`). Calling `Play` from the studio today **crashes** with "audio backend not initialized." Only `pixelforge_ebiten/internal/ebitengame.go:22-24` sets the backend — for the shipped game.
- **No bundled library exists.** Designers either bring their own WAVs (and there's no import flow, so they must hand-edit the .pforge JSON) or ship games with no sound.
- **No bindings UI.** `AudioBinding` (`pixelforge_project/audio.go:32-51`) carries `Topic`, `SampleName`, `SceneID`, `TriggerCondition`, `ForceChannel`. Designers who want a sound to play when an event fires must edit `.pforge` JSON by hand.

The brainstorm's bet: a designer opens the Audio workspace, types "BGM" into the filter, auditions 4 BGM loops by clicking play, drags (or "Bind"-buttons) the chosen loop onto the `SceneEntered:title` topic, saves, and the title screen has music. Three-minute task, no learning curve. The blocker today isn't audio quality (Paula mixer works); it's that every step of that flow is missing.

---

## Carried Forward from Origin

All 10 requirements, 6 acceptance examples, 4 flows, 3 actors from origin are in scope.

| Origin | Scope summary | Plan unit(s) |
|---|---|---|
| R1 | Bundled library of ~30 SFX + 4-5 BGM WAVs, 8-bit mono PCM, NES-authentic | U3 |
| R2 | Role-based categories (jump/shoot/hit/pickup/coin/menu-confirm/win-jingle/lose-stinger/damage/death/ambient + BGM: town/dungeon/boss/title/victory) | U3, U6 |
| R3 | Dockable Audio workspace; same pattern as Scene/Inspector/etc. | U6 |
| R4 | Library picker + bindings table; no mixer-lane | U6, U7 |
| R5 | Audition through Paula mixer; SFX play once, BGM loops; click-to-stop | U1 (backend init), U5 (audition helper), U6 (picker integration) |
| R6 | Bindings table; drag-or-button bind + "Pick sound..." overlay | U7 |
| R7 | Library WAV copied into project's `<assets>/audio/` on first bind | U4 (import), U6 (Bind button calls import) |
| R8 | Ship-loop integration (depends on idea #7) | Documented dependency only |
| R9 | User-imported WAVs equal-footing with library patches | U4, U7, U8 |
| R10 | Paula auto-allocator (using SuggestedChannelPriority) | **U2 (new allocator — not "existing" per research)** |
| AE1-AE6, F1-F4 | All six acceptance examples + four flows | U9 (integration tests) |
| A1-A3 | Designer, Studio, Pixelforge engine audio pipeline | All units |

Origin's "Deferred to Planning" section: all 6 technical/design questions resolved in Phase 2 (see Key Technical Decisions). One discovered question that the research surfaced: the allocator doesn't exist — resolved by adding it as U2 (this was implicit in R10; making it explicit).

---

## High-Level Technical Design

How the pieces fit together:

```
                  BUNDLED LIBRARY (go:embed)                     U3
                  ─────────────────────────────────────────
   ┌─────────────────────────────────────────────────────┐
   │ pixelforge_studio/audiolib/                         │
   │ ├── cart_assets/library/                            │
   │ │   ├── sfx/                                        │
   │ │   │   ├── jump/spring.wav                         │
   │ │   │   ├── jump/heavy.wav                          │
   │ │   │   ├── shoot/laser_small.wav                   │
   │ │   │   ├── ... (≈30 SFX organized by category)    │
   │ │   └── bgm/                                        │
   │ │       ├── town/peaceful_loop.wav                  │
   │ │       ├── dungeon/dark_loop.wav                   │
   │ │       ├── boss/intense_loop.wav                   │
   │ │       ├── title/intro_loop.wav                    │
   │ │       └── victory/triumphant_jingle.wav           │
   │ ├── library.go      (//go:embed cart_assets/...)    │
   │ └── catalog.json    (NEW: name+category+loop meta)  │
   └────────────────┬────────────────────────────────────┘
                    │
                    │ at studio startup:
                    ▼
           ┌────────────────────────┐
           │ audiolib.LoadCatalog() │  parses catalog.json
           │  → []LibraryPatch{     │  validates each WAV via
           │     Name, Category,    │  DecodeWavOrErr (skip on
           │     IsBGM, FSPath,     │  bad WAV; log + continue
           │     Duration }         │  per editor-pforge-schema-
           └─────────┬──────────────┘  shape.md discipline)
                     │
   ┌─────────────────┼────────────────────────────────────┐
   ▼                 ▼                                    ▼
┌────────────┐  ┌──────────────────┐               ┌──────────────────┐
│ AUDIO      │  │ AUDIO            │               │ BINDINGS TABLE   │
│ WORKSPACE  │  │ WORKSPACE        │               │ PANEL (right)    │
│ (chrome)   │  │ LIBRARY PICKER   │               │                  │
│            │  │ (left panel)     │               │ Walks            │
│ ImGui      │  │                  │               │ Project.Bindings │
│ docked     │  │ ┌──────────────┐ │               │ as rows:         │
│ panel —    │  │ │ Filter:[BGM] │ │               │                  │
│ Workspace  │  │ ├──────────────┤ │               │ Topic   │ Sample │
│ interface  │  │ │ ▶ jump/spring│ │               │ ─────── │ ────── │
│            │  │ │   Bind...    │ │               │ PlrJump │ jump_… │
│ Render(e)  │  │ │ ▶ jump/heavy │ │   ── Bind ─→  │ +Add    │        │
│            │  │ │   Bind...    │ │               │                  │
│            │  │ │ ▶ shoot/...  │ │               │ Each row:        │
│            │  │ │ ...          │ │               │ - Topic dropdown │
│            │  │ └──────────────┘ │               │   (from pievent. │
│            │  │                  │               │   EnumerateTar-  │
│            │  │ Audition via     │               │   gets() + free  │
│            │  │ Play button →    │               │   text fallback) │
│            │  │ audiolib.Audi-   │               │ - SampleName     │
│            │  │ tion(sample)     │               │   "Pick sound..."│
│            │  │                  │               │   button → over- │
│            │  │ Bind button →    │               │   lay (next col) │
│            │  │ Material+Import  │               │ - SceneID input  │
│            │  │ then add row     │               │ - TriggerCond    │
│            │  │ to bindings      │               │   input          │
└────────────┘  └─────────┬────────┘               │ - [Delete] btn   │
                          │                        └────────┬─────────┘
                          │                                 │ "Pick sound..."
                          ▼                                 ▼
              ┌───────────────────────────┐    ┌────────────────────────┐
              │ audio.Import(libPath OR   │    │ SOUND PICKER OVERLAY   │
              │   []bytes, projectPath,   │    │ (modal; modal-stack    │
              │   sampleName)             │    │  participant)          │
              │                           │    │                        │
              │ - DecodeWavOrErr (8-bit   │    │ Lists all              │
              │   mono PCM strict gate)   │    │ Project.Audio entries  │
              │ - Copy bytes → <proj-     │    │ (library-sourced +     │
              │   assets>/audio/<name>.wav│    │  user-imported, equal  │
              │ - Append AudioSample{     │    │  footing per R9).      │
              │   Name, RelativePath,     │    │                        │
              │   SuggestedChannel-       │    │ Click → updates row's  │
              │   Priority, Loop,         │    │ SampleName + close.    │
              │   SampleRateHz} to        │    │                        │
              │   p.Audio                 │    │ Cancel button: closes. │
              │ - MarkDirty               │    │                        │
              └─────────┬─────────────────┘    └────────────────────────┘
                        │
                        │
                        ▼
              ┌─────────────────────────────────────────┐
              │ NEW pixelforge_audio.Allocator          │   U2
              │ (used by audition AND runtime           │
              │ scripting catalog's Play step)          │
              │                                         │
              │ Allocator.Pick(sample *AudioSample,     │
              │   force int) Chan:                      │
              │  if force > 0 → return force            │
              │  switch sample.SuggestedChannelPriority │
              │  case "bgm":                            │
              │    if !ChannelActive(Chan1) → Chan1     │
              │    if !ChannelActive(Chan2) → Chan2     │
              │    return Chan1  // steal oldest        │
              │  case "sfx","voice","ambient":          │
              │    if !ChannelActive(Chan3) → Chan3     │
              │    if !ChannelActive(Chan4) → Chan4     │
              │    return roundRobin(Chan3, Chan4)      │
              └─────────────────────────────────────────┘
                        │
                        │ used by audition (U5) AND
                        │ by runtime scripting catalog's
                        │ existing Play step
                        ▼
              ┌──────────────────────────────┐
              │ pixelforge_audio.Play(ch,    │
              │   sample, pitch, vol)        │  (existing engine API)
              │                              │
              │ For BGM: caller follows with │
              │ SetLoop(ch, 0, sample.Len(), │
              │ LoopForward, 0)              │
              └──────────────────────────────┘
```

*This illustrates the intended approach and is directional guidance for review, not implementation specification.*

The structural insight: **the studio audition path AND the runtime Play step share the same allocator**. Whatever the designer hears in the studio is what the shipped game produces — no parallel preview engine, no audition-vs-runtime divergence. The library WAV → AudioSample copy on Bind means the shipped binary has zero dependency on the studio's bundle.

---

## Output Structure

```
pixelforge_audio/
├── allocator.go                                (NEW, U2) — Allocator{Pick(sample, force) Chan}
├── allocator_test.go                           (NEW, U2)
├── piaudio.go                                  (no change)
├── decode.go                                   (no change)
└── ... (existing files)

pixelforge_studio/audiolib/                     (NEW package, idiomatic per palette/, capture/, scripting/ siblings)
├── cart_assets/                                (NEW dir)
│   ├── library/                                (NEW — embedded WAV tree)
│   │   ├── sfx/<category>/<patch>.wav          (≈30 files)
│   │   └── bgm/<category>/<patch>.wav          (4-5 files)
│   └── catalog.json                            (NEW — patch metadata: name+category+isBGM+suggestedPriority)
├── library.go                                  (NEW, U3) — //go:embed + LoadCatalog
├── library_test.go                             (NEW, U3)
├── import.go                                   (NEW, U4) — audio.Import() function
├── import_test.go                              (NEW, U4)
├── audition.go                                 (NEW, U5) — Audition(sampleName, isLoop) using Allocator
├── audition_test.go                            (NEW, U5)
├── workspace.go                                (NEW, U6) — Workspace interface impl + RegisterWith(e)
├── workspace_test.go                           (NEW, U6)
├── picker_panel.go                             (NEW, U6) — library picker UI (left panel)
├── picker_panel_test.go                        (NEW, U6)
├── bindings_panel.go                           (NEW, U7) — bindings table UI (right panel)
├── bindings_panel_test.go                      (NEW, U7)
├── sound_picker_overlay.go                     (NEW, U7) — "Pick sound..." modal-stack participant
└── sound_picker_overlay_test.go                (NEW, U7)

pixelforge_studio/
├── main.go                                     (MODIFY, U1) — init piaudio.Backend before ebiten.RunGame
└── editor/
    ├── file_menu.go                            (MODIFY, U8) — File → Import WAV…; View → Audio entry
    └── audio_import_handler.go                 (NEW, U8) — file picker → audio.Import orchestration

pixelforge_ebiten/internal/audio/               (MODIFY, U1)
└── start.go                                    (EXISTING; may need to re-export StartAudioBackend if package is currently internal/)

pixelforge_studio/integration_test/
├── audio_e2e_test.go                           (NEW, U9)
└── fixtures/
    ├── jump_test.wav                           (NEW — 8-bit mono synthetic for tests)
    ├── bgm_loop_test.wav                       (NEW — 8-bit mono synthetic loop)
    └── audio_project_with_bindings.pforge      (NEW — fixture with 3 bindings)
```

Implementer may consolidate or split files; per-unit `**Files:**` sections remain authoritative.

---

## Implementation Units

Dependency-ordered. Foundation (U1, U2, U3) → import + audition (U4, U5) → UI (U6, U7) → integration (U8) → tests (U9).

### U1. Editor Paula backend initialization

**Goal:** Initialize `piaudio.Backend` at studio startup so audition (and any other studio-side audio playback) works. Without this, any `Play` call panics via the default `panicBackend{}`. Re-export `StartAudioBackend` from `pixelforge_ebiten/internal/audio` (or inline ~3 lines) so the studio can mirror the runtime's init sequence.

**Requirements:** R5 (audition through Paula mixer — foundation: backend must exist).

**Dependencies:** none (purely infrastructure).

**Files:**
- `pixelforge_studio/main.go` (MODIFY — add backend init before `ebiten.RunGame(e)`)
- `pixelforge_ebiten/internal/audio/start.go` (MODIFY — if necessary, move helper out of `internal/` so `pixelforge_studio/main.go` can import; OR copy the 3-line init into a new package like `pixelforge_audio/initbackend/`)
- `pixelforge_studio/main_test.go` (NEW — assert backend is non-panic after init)

**Approach:**
- Studio main currently: `editor.NewEbitenImguiBackend(...)` → `editor.NewWithSettings(...)` → `ebiten.RunGame(e)`. Adds: between editor creation and `RunGame`, init Paula backend.
- Existing pattern at `pixelforge_ebiten/internal/ebitengame.go:22-24`:
  ```
  ctx := ebitenaudio.NewContext(audio.CtxSampleRate)
  piaudio.Backend = audio.StartAudioBackend(ctx)
  ```
- The `internal/audio` package is import-restricted. **Decision:** rename `pixelforge_ebiten/internal/audio` → `pixelforge_ebiten/audio` OR create a new exported `pixelforge_audio/ebitenbackend` package that wraps the same 3 lines. The cleanest move: extract `StartAudioBackend` to a non-internal package so both the shipped game runtime AND the studio call it identically.
- After init, verify with `piaudio.Backend != nil && _, isPanic := piaudio.Backend.(panicBackend); !isPanic` (or equivalent type assertion).

**Patterns to follow:** existing `pixelforge_ebiten/internal/ebitengame.go:22-24` init; existing `pixelforge_studio/main.go:31-62` startup flow.

**Test scenarios:**
- `TestStudioStartup_AudioBackendInitialized`: launch studio (test harness); confirm `piaudio.Backend` is not the panic backend.
- `TestStudioStartup_PlayDoesNotPanic`: after init, calling `piaudio.Play(Chan1, sampleStub, 1.0, 1.0)` does not panic.
- `TestStudioStartup_AudioBackendIdleByDefault`: after init, no channels are active (`ChannelActive` returns false for Chan1..Chan4).
- `TestPackageReexport_StartAudioBackendCallable`: from `pixelforge_studio/main.go`'s import perspective, `StartAudioBackend` (or its replacement) is importable.

**Verification:** `go test ./pixelforge_studio/...` passes; manual smoke: launch studio, observe no panic on startup; (later, after U5) audition works.

---

### U2. Paula channel allocator

**Goal:** Implement `pixelforge_audio.Allocator` that reads `AudioSample.SuggestedChannelPriority` and selects an output Chan. Used by both audition (U5) and the runtime scripting catalog's `Play` step (which currently must hard-code channel choice). Handles `AudioBinding.ForceChannel` override (when non-zero, returns that channel directly).

**Requirements:** R10 (Paula auto-allocator using SuggestedChannelPriority; BGM locks Chan1/Chan2; SFX round-robin Chan3/Chan4).

**Dependencies:** none (engine-side, parallel with U1).

**Files:**
- `pixelforge_audio/allocator.go` (NEW)
- `pixelforge_audio/allocator_test.go` (NEW)
- `pixelforge_studio/scripting/catalog/builtin_actions.go` (MODIFY — the existing `Play` action in the scripting catalog should now consult Allocator instead of hard-coding a channel; OR if Play doesn't currently pick a channel, route through Allocator)

**Approach:**
- `Allocator` is stateful (tracks round-robin index for SFX channels). Singleton or instance-per-project decision: **singleton** in v1 (one allocator per process; matches the singleton Paula mixer).
- API:
  - `NewAllocator() *Allocator`
  - `a.Pick(sample *AudioSample, forceChannel int) Chan` — returns the chosen channel.
  - `a.Reset()` — clears round-robin state (used at scene transition or for test isolation).
- Algorithm (per the brainstorm's R10):
  1. If `forceChannel` in 1..4, return that channel directly.
  2. Switch on `sample.SuggestedChannelPriority`:
     - `"bgm"`: prefer Chan1 if not active, else Chan2 if not active, else **steal Chan1** (BGM steals BGM; the new BGM "takes over"). Trade-off: brief glitch on swap; alternative is to keep playing old BGM and ignore the new — defer that complexity.
     - `"sfx"`, `"voice"`, `"ambient"`, or empty (default to SFX): round-robin between Chan3 and Chan4. Track last-picked index. If both active, steal the **oldest** (the one whose round-robin slot came up last).
  3. (Future: `ChannelPosition` could give us "oldest" more precisely. v1 uses round-robin slot ordering; good enough.)
- `Allocator.Pick` consults `piaudio.ChannelActive(ch)` for activity check — confirmed available per research.
- **Critical: the allocator does not call `Play` itself.** It returns the Chan; the caller (audition or runtime) calls `Play`.
- **Runtime integration**: the existing `Play` step in `pixelforge_studio/scripting/catalog/builtin_actions.go` (need to confirm exact location and existing behavior) must consult `Allocator.Pick(sample, binding.ForceChannel)` before calling `piaudio.Play`. If `Play` step doesn't currently exist (audio bindings have never been wired to a runtime trigger), this unit OR a follow-up unit must wire it through `pievent`. **Implementer verifies at execution time.**

**Patterns to follow:** existing `pixelforge_audio.Play(ch, sample, ...)` signature; existing `piaudio.ChannelActive(ch) bool`; the brainstorm's R10 policy.

**Test scenarios:**
- `TestAllocator_ForceChannelOverridesPriority`: sample with `SuggestedChannelPriority="bgm"`, `forceChannel=4`; returns Chan4.
- `TestAllocator_BGMPicksChan1WhenIdle`: SuggestedChannelPriority="bgm", Chan1 idle; returns Chan1.
- `TestAllocator_BGMPicksChan2WhenChan1Busy`: BGM, Chan1 active (mock `ChannelActive`), Chan2 idle; returns Chan2.
- `TestAllocator_BGMStealsChan1WhenBothBusy`: BGM, both Chan1+Chan2 active; returns Chan1 (steal-oldest policy).
- `TestAllocator_SFXRoundRobinAcrossChan3Chan4`: SFX, both idle; first call returns Chan3, second returns Chan4, third returns Chan3.
- `TestAllocator_SFXSkipsBusyChannel`: SFX, Chan3 active, Chan4 idle; returns Chan4 (doesn't round-robin-pick the busy one).
- `TestAllocator_VoiceAndAmbientBehaveLikeSFX`: SuggestedChannelPriority="voice"; returns Chan3 (defaults to SFX path).
- `TestAllocator_EmptyPriorityDefaultsToSFX`: SuggestedChannelPriority=""; returns Chan3 or Chan4 round-robin.
- `TestAllocator_ResetClearsRoundRobin`: pick SFX 3 times; reset; first SFX pick after reset returns Chan3 (start of cycle).
- `TestAllocator_InvalidForceChannelTreatedAsAuto`: forceChannel=99; falls through to priority-based pick.
- `TestAllocator_Pick_DoesNotCallPlay`: confirm Pick is pure — no side effects on channel state (no Load/Play).

**Verification:** `go test ./pixelforge_audio/...` passes; existing `Play` callers route through Allocator without behavior regression.

---

### U3. Bundled audio library + catalog (sound-design + go:embed)

**Goal:** Produce the bundled library — ~30 SFX + 4-5 BGM WAVs in 8-bit mono PCM (Paula-compatible per `DecodeWavOrErr`). Organize under `pixelforge_studio/audiolib/cart_assets/library/{sfx,bgm}/<category>/`. Add `catalog.json` declaring patch metadata (name, category, isBGM, suggestedPriority). Embed via `go:embed`. Provide `audiolib.LoadCatalog() ([]LibraryPatch, error)` that parses catalog + validates each WAV's format via `DecodeWavOrErr` (malformed WAV → log + skip, never panic per `editor-pforge-schema-shape.md`).

**Requirements:** R1 (~30 SFX + 4-5 BGM, NES-authentic, mono, Paula-compatible), R2 (role-based categories).

**Dependencies:** none (asset work + simple Go).

**Files:**
- `pixelforge_studio/audiolib/cart_assets/library/sfx/<category>/<patch>.wav` (NEW — many files; see sound-design source below)
- `pixelforge_studio/audiolib/cart_assets/library/bgm/<category>/<patch>.wav` (NEW)
- `pixelforge_studio/audiolib/cart_assets/library/catalog.json` (NEW — patch metadata)
- `pixelforge_studio/audiolib/library.go` (NEW — embed directives + LoadCatalog)
- `pixelforge_studio/audiolib/library_test.go` (NEW)

**Approach — sound-design source (recommendation):**
- **Primary path: sfxr / Bfxr / jsfxr-style preset generators.** Run sfxr (Mac/Linux/Windows), use the genre presets (Jump, Laser, Hit, Pickup, Coin, etc.) that match the brainstorm's categories. Tune sliders if a preset is close-but-not-quite. Export each as 8-bit mono PCM WAV. Free, widely-used, recognizably NES-class. ~30 SFX in a focused half-day of work.
- **BGM fallback: CC0 sourcing** (OpenGameArt, FreeSound CC0 entries). Search "NES-style", "chiptune", "8-bit loop". License-check (CC0 only — no attribution required). Convert to 8-bit mono via `ffmpeg -acodec pcm_u8 -ac 1 input.ogg output.wav`. Loop point determined by ear; trim to even-length loop in Audacity.
- **Tertiary fallback: hand-record via FamiTracker / FamiStudio** (NES-style trackers). More time-intensive but produces highest fidelity.
- **What v1 ships:** at minimum, **one valid patch per category** (so the library demonstrates coverage of each NES-class game type) + any extras the curator has time for. The brainstorm's "~30 SFX + 4-5 BGM" is the target; shipping with fewer is acceptable if it lands the bet (designer can ship a game with sound). Document the actual ship count in the plan's verification step.

**Approach — code:**
- `LibraryPatch` struct: `Name, Category string; IsBGM bool; SuggestedPriority string; FSPath string` (path in embedded FS); `Duration time.Duration` (computed from WAV header).
- `catalog.json` schema (manually authored alongside the WAV curation):
  ```
  [
    { "name": "jump/spring", "category": "jump", "isBGM": false, "suggestedPriority": "sfx", "fsPath": "library/sfx/jump/spring.wav" },
    { "name": "shoot/laser_small", "category": "shoot", "isBGM": false, "suggestedPriority": "sfx", "fsPath": "library/sfx/shoot/laser_small.wav" },
    ...
    { "name": "town/peaceful_loop", "category": "town", "isBGM": true, "suggestedPriority": "bgm", "fsPath": "library/bgm/town/peaceful_loop.wav" }
  ]
  ```
- `library.go`:
  ```
  //go:embed cart_assets/library
  var libraryFS embed.FS

  //go:embed cart_assets/library/catalog.json
  var catalogBytes []byte

  func LoadCatalog() ([]LibraryPatch, error) {
      // Parse catalog.json
      // For each patch, read the WAV bytes from libraryFS, call pixelforge_audio.DecodeWavOrErr
      //   on success: append to result list
      //   on failure: log + skip (never panic; matches editor-pforge-schema-shape.md discipline)
      // Return result list + any errors aggregated
  }
  ```
- Catalog format is defensive: missing `fsPath` → skip; missing `name` → skip; unknown `category` → keep (designer might use unknown category for filtering).
- `audiolib.ReadPatchBytes(name string) ([]byte, error)` returns the embedded WAV bytes for a given patch name (used by the import flow in U4/U6).

**Patterns to follow:** existing `go:embed` precedent at `pixelforge_studio/editor/imgui_theme.go:29` (single file) and `pixelforge_audio/decode_test.go:21-32` (multiple WAVs); existing JSON-parse pattern from project loader; defensive-load pattern from `palette-quantization-metric.md`.

**Test scenarios:**
- `TestLoadCatalog_ReturnsAllValidPatches`: load catalog; result length matches the number of valid `catalog.json` entries.
- `TestLoadCatalog_MalformedCatalogJsonReturnsError`: catalog with broken JSON → returns error, no panic.
- `TestLoadCatalog_PatchWithMissingFSPathSkipped`: catalog entry with `"fsPath": ""` → not in result; warning logged.
- `TestLoadCatalog_PatchWithInvalidWAVSkipped`: catalog entry pointing to a stereo WAV (or 16-bit, etc.); DecodeWavOrErr fails; entry skipped, others continue.
- `TestLoadCatalog_ResultHasNameCategoryIsBGM`: each LibraryPatch has the expected fields populated from catalog.json.
- `TestLoadCatalog_DurationComputedFromWAV`: patch.Duration matches the WAV header's sample-count / sample-rate calculation.
- `TestReadPatchBytes_KnownPatchReturnsBytes`: ReadPatchBytes("jump/spring") returns non-empty bytes (assuming patch exists; mark as skip if curation hasn't shipped yet).
- `TestReadPatchBytes_UnknownPatchReturnsError`: ReadPatchBytes("nonexistent") returns error.
- `TestLibrary_AllWAVsAreParseable`: integration — for every catalog entry, ReadPatchBytes + DecodeWavOrErr succeeds. This is the v1 ship gate for the bundle.
- `TestLibrary_CategoryCoverageMatchesBrainstorm`: confirm at least one patch per brainstorm-specified category exists in the catalog (jump, shoot, hit, pickup, coin, menu-confirm, win-jingle, lose-stinger, damage, death, ambient, town, dungeon, boss, title, victory). Failing categories logged as warnings (not test failures — sound-design may ship incrementally).
- Covers AE2 (BGM filter shows 4-5 loops with icons).

**Verification:** `go test ./pixelforge_studio/audiolib/...` passes; manual: open library picker, see categorized patches, confirm count.

---

### U4. `audio.Import` pipeline (mirror palette.Import)

**Goal:** Build the canonical `audiolib.Import(source, projectPath) (ImportResult, error)` function. Source is either a filesystem path (for user-imported WAVs) or `[]byte` (for library WAVs read from the embedded FS). Decodes via `DecodeWavOrErr`, copies into `AssetsDir(projectPath)/audio/<basename>.wav`, appends an `AudioSample` to `p.Audio` with `SampleRateHz`/`Loop`/`SuggestedChannelPriority` populated from source metadata.

**Requirements:** R7 (library WAV copied into project's `<assets>/audio/` on first bind), R9 (user-imported WAVs equal-footing).

**Dependencies:** U3 (LibraryPatch metadata informs the AudioSample fields for library imports).

**Files:**
- `pixelforge_studio/audiolib/import.go` (NEW)
- `pixelforge_studio/audiolib/import_test.go` (NEW)

**Approach:**
- Signature options (implementer decides between two; suggestion below):
  ```
  type ImportSource interface { Bytes() ([]byte, error); Name() string }
  type FileSource struct { Path string }       // for user file picker
  type LibrarySource struct { Patch LibraryPatch } // for library Bind button
  func Import(src ImportSource, p *Project, projectPath string) (ImportResult, error)
  ```
  OR simpler: two separate functions `ImportFromFile(path, p, ppath)` and `ImportFromLibrary(patch, p, ppath)`, both calling a shared `importBytes(bytes, name, p, ppath)` core.
- Core flow:
  1. Get WAV bytes (from path or library FS).
  2. `DecodeWavOrErr(bytes)` — fails fast on bad format.
  3. Choose dest filename: `<name_or_basename>.wav`. Sanitize name: lowercase, replace `/` with `_` (e.g., `jump/spring` → `jump_spring.wav`).
  4. Handle name collisions: if `AssetsDir(ppath)/audio/jump_spring.wav` exists, suffix `_2`, `_3`, etc. until unique.
  5. Write bytes to `AssetsDir(ppath)/audio/<unique_name>.wav` (create dir if needed).
  6. Build `AudioSample{Name, RelativePath: "audio/<unique_name>.wav", SuggestedChannelPriority, Loop, SampleRateHz}`.
     - `SuggestedChannelPriority`: from LibrarySource's patch; for FileSource, default to `"sfx"` (designer can edit in inspector if needed).
     - `Loop`: from LibrarySource's `IsBGM`; for FileSource, default `false` (designer can edit).
     - `SampleRateHz`: from WAV header (`DecodeWavOrErr` exposes it; if not, parse from header bytes).
  7. Append to `p.Audio`.
  8. `e.MarkDirty()` (per `dirty-state-ux.md`).
  9. Return `ImportResult{SampleName, RelativePath, Duration, IsBGM}` for the caller to display.
- Per `editor-pforge-schema-shape.md`: on import failure, return error with helpful message ("WAV format not supported: requires 8-bit mono PCM"); do not crash.
- Per `loader.go:130` validateAssets: the new file MUST resolve under `AssetsDir(ppath)` (the loader will assert this on next load).

**Patterns to follow:** `pixelforge_studio/palette/import_pipeline.go:40-107` end-to-end — the canonical Import shape (decode → validate → copy → append to project → return result).

**Test scenarios:**
- `TestImport_FromFile_ValidWAVAppendsToProject`: import jump_test.wav; `p.Audio` gains an entry; file copied to expected path; ImportResult populated.
- `TestImport_FromFile_InvalidWAVReturnsError`: import a stereo WAV; returns error mentioning format requirement; project unchanged.
- `TestImport_FromLibrary_CopiesEmbeddedBytes`: ImportFromLibrary("jump/spring", p, ppath); bytes from library FS written to `<assets>/audio/jump_spring.wav`; AudioSample.Loop matches LibraryPatch.IsBGM; SuggestedChannelPriority matches LibraryPatch.SuggestedPriority.
- `TestImport_NameCollisionSuffixes`: import the same library patch twice; second copy lands at `jump_spring_2.wav`; both AudioSamples exist with distinct names.
- `TestImport_PathSanitization`: import a library patch with `/` in name; resulting RelativePath uses `_` instead.
- `TestImport_AssetsDirCreatedIfMissing`: import into a project whose `<assets>/audio/` doesn't exist yet; dir created; file written.
- `TestImport_MarkDirtyCalled`: after successful import, e.MarkDirty observed.
- `TestImport_SampleRateHzReadFromWAVHeader`: import a 22050 Hz WAV; resulting AudioSample.SampleRateHz == 22050.
- `TestImport_FileSourceDefaultsToSFXPriority`: import user file with no metadata; SuggestedChannelPriority defaults to "sfx".
- `TestImport_LibrarySourceUsesPatchPriority`: ImportFromLibrary with patch.SuggestedPriority="bgm"; resulting AudioSample.SuggestedChannelPriority == "bgm".
- Covers AE3 (Bind copies library WAV into project's audio-assets/), F4 setup (user imports custom WAV → appears as AudioSample).

**Verification:** `go test ./pixelforge_studio/audiolib/...` passes; manual: import a WAV from disk OR via library Bind; observe file appears in `.pforge-assets/audio/`.

---

### U5. Audition helper

**Goal:** `audiolib.Audition(p *Project, sample *AudioSample, isLoop bool)` plays a sample through the Paula mixer using the Allocator-chosen channel. SFX plays once; BGM loops automatically. Click-to-stop semantics: a second call with the same sample stops it. Tracks currently-auditioning sample/channel state.

**Requirements:** R5 (audition through Paula mixer; SFX plays once, BGM loops; click-again-to-stop).

**Dependencies:** U1 (backend init), U2 (Allocator), U4 (AudioSample exists in project for the library-bound path; for raw-library audition before Bind, the audition reads from library FS directly without project mutation).

**Files:**
- `pixelforge_studio/audiolib/audition.go` (NEW)
- `pixelforge_studio/audiolib/audition_test.go` (NEW)

**Approach:**
- `AuditionState` (singleton on the audiolib package or held on Editor): tracks `{currentSample *Sample, currentChan Chan, isLoop bool}` for click-to-stop detection.
- API:
  - `Start(sample *pixelforge_audio.Sample, suggestedPriority string, isLoop bool) Chan` — picks channel via Allocator, LoadSample if not loaded, Play, then for loops SetLoop. Returns the chosen channel for status display.
  - `Stop()` — ClearChan(currentChan, 0); reset AuditionState.
  - `IsActive(sample *Sample) bool` — true if currently auditioning the given sample.
- The picker's Play button is a 3-state toggle:
  - If no audition active OR auditioning a DIFFERENT sample → Stop() (if needed) + Start(thisSample).
  - If auditioning THIS sample → Stop().
- The audition is just `piaudio.Play(ch, sample, 1.0, 1.0)` followed by `piaudio.SetLoop(ch, 0, sample.Len(), LoopForward, 0)` if `isLoop`.
- **No separate preview engine** per `always-on-game-embedding.md` — the audition uses the same Paula mixer the runtime uses. If the game is playing audio when audition starts, the Allocator handles channel competition (BGM steals BGM, SFX round-robins). This is correct behavior — what the designer hears is what ships.
- For library audition (designer clicks Play on a library patch that hasn't been imported yet), the audition must work without mutating the project. **Implementer decision:** materialize the embedded library WAV bytes to a transient `*pixelforge_audio.Sample` (via `DecodeWavOrErr`) and Load it for the audition. Track in AuditionState; Unload on Stop or on switching to a different sample.
- Per `dirty-state-ux.md`: audition is NOT a mutation; no MarkDirty.

**Patterns to follow:** existing `pixelforge_audio.Play(ch, sample, pitch, vol)` API; existing `LoadSample`/`UnloadSample` lifecycle (mandatory before Play); existing `SetLoop` for BGM looping.

**Test scenarios:**
- `TestAudition_StartPlaysSample`: call Start with a sample; assert `ChannelActive(returnedChan)` becomes true (test must wait one tick or mock); `AuditionState.currentSample == sample`.
- `TestAudition_StartUsesAllocatorPick`: spy on Allocator; Start with bgm sample; Pick called with that sample.
- `TestAudition_BGMLoopsViaSetLoop`: Start with isLoop=true; assert SetLoop was called for the chosen channel with LoopForward.
- `TestAudition_SFXDoesNotSetLoop`: Start with isLoop=false; assert SetLoop was NOT called (default Play behavior: LoopNone).
- `TestAudition_StopClearsChannel`: Start then Stop; assert ClearChan called on the channel; AuditionState reset.
- `TestAudition_StartSameSampleAgainStops`: Start sample A; call Start sample A again; assert Stop semantics fired (ClearChan).
- `TestAudition_StartDifferentSampleStopsCurrent`: Start sample A; Start sample B; assert sample A's channel cleared, sample B's channel started.
- `TestAudition_IsActiveTracksState`: Start sample A; IsActive(A) is true, IsActive(B) is false; Stop; IsActive(A) is false.
- `TestAudition_LibraryBytesPathWithoutProjectMutation`: audition a library patch directly from bytes (no Import); project.Audio is unchanged; audition still plays.
- `TestAudition_PlayBackendNotInitPanics`: edge — if backend wasn't init (regression for U1), audition fails gracefully (logs + returns error, doesn't panic the studio). [Defensive — U1 should prevent this state.]
- Covers AE1 (Play → audition; Play again → stop), AE4 (BGM loops automatically).

**Verification:** `go test ./pixelforge_studio/audiolib/...` passes; manual: click Play on a library SFX, hear it; click again, stops. Click Play on a BGM loop, hear it loop; click another patch, the loop stops and the new one starts.

---

### U6. Audio workspace + library picker panel

**Goal:** New dockable `audiolib.AudioWorkspace` implementing the existing `Workspace` interface. Workspace renders two columns: library picker (left, this unit) + bindings table (right, U7). Library picker shows rows with name, category, duration, loop icon (BGM only), Play button (calls U5 Audition), Bind button (calls U4 Import + appends a new AudioBinding row).

**Requirements:** R3 (dockable Audio workspace), R4 (two panels — library picker + bindings table), R5 (Play button auditions; SFX once, BGM loops), R6 (Bind button drops onto a row in the bindings table).

**Dependencies:** U3 (library catalog), U4 (Import for Bind), U5 (Audition for Play).

**Files:**
- `pixelforge_studio/audiolib/workspace.go` (NEW — Workspace impl + RegisterWith(e))
- `pixelforge_studio/audiolib/workspace_test.go` (NEW)
- `pixelforge_studio/audiolib/picker_panel.go` (NEW — library picker rendering)
- `pixelforge_studio/audiolib/picker_panel_test.go` (NEW)
- `pixelforge_studio/main.go` (MODIFY — add `audiolib.RegisterWith(e)` alongside `palette.RegisterWith(e); capture.RegisterWith(e); scripting.RegisterWith(e)`)

**Approach:**
- `AudioWorkspace`:
  - `Name() string` → `"audio"` (matches keymap action `workspace.audio`).
  - `DisplayName() string` → `"Audio"`.
  - `Render(e *Editor)` → wraps everything in `imgui.Begin("Audio")` + horizontal split: left half library picker, right half bindings table (U7).
- `RegisterWith(e *Editor)`:
  - `w := NewAudioWorkspace(e)`
  - `e.RegisterWorkspace(w)`
  - (Mirror of `palette/workspace.go:187-192`.)
- Library picker:
  - Render scrollable list of patches from `audiolib.LoadCatalog()` (cached at workspace init).
  - Top of panel: filter input (`imgui.InputText("Filter")`) — designer types category name, list filters by category substring match (case-insensitive). Empty filter = show all.
  - Each row: `imgui.PushID(patch.Name)` + `imgui.Text(patch.Name)` + `imgui.SameLine + imgui.TextColored(grey, patch.Category)` + `imgui.SameLine + imgui.Text(duration)` + (if patch.IsBGM) `imgui.SameLine + imgui.Text("⟲")` (loop icon — use unicode or icon font) + `imgui.SameLine + imgui.Button("▶")` (Play — toggles Audition.Start/Stop) + `imgui.SameLine + imgui.Button("Bind...")` (Bind).
  - Play button visual: when this patch is the currently-auditioning sample, button shows `■` (stop icon) instead of `▶`.
  - Bind button click:
    1. Calls `audiolib.Import(LibrarySource{Patch: patch}, p, projectPath)` (U4).
    2. On success, appends a new `AudioBinding` row to `p.Bindings` with `SampleName = result.SampleName`, `Topic = ""` (empty — designer fills in via the bindings table dropdown).
    3. Scrolls bindings table to the new row (signal via editor state).
    4. MarkDirty.
    5. On import error, shows transient error toast.
- Per `focus-manager-design.md`: workspace TextInput (filter) registers with FocusManager — standard widget discipline.

**Patterns to follow:** `pixelforge_studio/palette/workspace.go:187-192` for Workspace registration shape; `pixelforge_studio/main.go:55-57` for call site; existing `imgui.Begin`/`End` window pattern; existing `asset_browser.go` for list-with-selection rendering reference; existing filter input pattern (search the codebase for `InputText` usage with "Filter" label).

**Test scenarios:**
- `TestAudioWorkspace_RegisteredWithEditor`: after `RegisterWith(e)`, `e.GetWorkspace("audio")` returns non-nil with DisplayName "Audio".
- `TestAudioWorkspace_RenderProducesLibraryPickerAndBindingsPanel`: render workspace; output (mock ImGui sink) shows both library picker section and bindings panel section.
- `TestPickerPanel_FilterByCategoryHidesNonMatching`: catalog has patches in jump, shoot, BGM categories; filter "shoot" → only shoot patches visible; filter "BGM" → only BGM patches visible.
- `TestPickerPanel_EmptyFilterShowsAll`: filter "" → all patches visible.
- `TestPickerPanel_PlayButtonInvokesAudition`: click Play on patch X; Audition.Start called with patch X's sample.
- `TestPickerPanel_PlayButtonShowsStopWhenAuditioning`: while patch X is auditioning, its button label = "■"; other rows show "▶".
- `TestPickerPanel_StopButtonCallsAuditionStop`: click "■" (stop) on auditioning patch; Audition.Stop called.
- `TestPickerPanel_BindButtonImportsAndAppendsBinding`: click Bind on patch "jump/spring"; verify: (a) Import called with LibrarySource for that patch; (b) `p.Audio` has new entry; (c) `p.Bindings` has new entry with `SampleName="jump_spring"` and `Topic=""`; (d) MarkDirty called.
- `TestPickerPanel_BindFailureShowsErrorToast`: simulate Import failure; UI shows error message, no project mutation.
- `TestPickerPanel_LoopIconShownOnlyForBGM`: BGM patches render with ⟲ icon; SFX patches don't.
- `TestPickerPanel_DurationShownAsHumanReadable`: 1.5s patch shows "1.5s"; 0.25s patch shows "0.3s" (rounded).
- Covers AE1 (Play button auditions), AE2 (filter shows BGM only), AE3 (Bind drops onto bindings table).

**Verification:** `go test ./pixelforge_studio/audiolib/...` passes; manual: open Audio workspace via View → Audio (Ctrl+4); see library picker on left; click Play on a patch; hear it. Click Bind; observe new row in bindings table.

---

### U7. Bindings table panel + sound picker overlay

**Goal:** Right-panel of Audio workspace. Walks `Project.Bindings []AudioBinding`. Each row: Topic dropdown (sources from `pievent.EnumerateTargets()` + project's defined topics, with free-text fallback), Sample picker ("Pick sound..." button → overlay with all AudioSample entries from `p.Audio`, equal-footing for library + user-imported), SceneID input, TriggerCondition input, Delete button. Add row via library Bind (from U6) or via "Add Binding" button at bottom.

**Requirements:** R6 (bindings table + sound picker overlay), R9 (library + user-imported samples on equal footing in picker).

**Dependencies:** U3 (library exists), U4 (Import populates p.Audio), U6 (workspace renders this as right panel).

**Files:**
- `pixelforge_studio/audiolib/bindings_panel.go` (NEW)
- `pixelforge_studio/audiolib/bindings_panel_test.go` (NEW)
- `pixelforge_studio/audiolib/sound_picker_overlay.go` (NEW — modal-stack participant)
- `pixelforge_studio/audiolib/sound_picker_overlay_test.go` (NEW)

**Approach:**
- Bindings panel layout:
  - Table with columns: Topic | Sound | SceneID | Condition | (Delete)
  - One row per `p.Bindings[i]` entry.
  - Top: count display (`"3 bindings"`). Bottom: `Add Binding` button (appends an empty AudioBinding row + opens topic dropdown immediately).
- **Topic column**:
  - `imgui.BeginCombo(...)` showing current topic.
  - Combo options sourced from `pievent.EnumerateTargets()` (engine-registered targets like `loop.main`, `key.main`, etc.) PLUS distinct topics already present in `p.Bindings` PLUS distinct topics in `p.EventSubscriptions` (per existing `buildWidgetContext.EventTopics` aggregation at `inspector_test.go:50`).
  - Last entry: `"<type custom...>"` → opens a text input for free-text entry (since topics are freeform strings per research finding #3).
  - Empty registry case: combo shows "(no topics defined)" + the custom-text option.
- **Sound column**:
  - Shows current `SampleName` as text.
  - "Pick sound..." button → opens `SoundPickerOverlay` (modal stack participant per `focus-manager-design.md`).
  - On pick → row's `SampleName` updates → MarkDirty.
- **SceneID column**: simple `imgui.InputText` with placeholder "(all scenes)".
- **Condition column**: simple `imgui.InputText` with placeholder "(always)".
- **Delete button**: removes row from `p.Bindings`; MarkDirty.
- **Sound picker overlay**:
  - `imgui.OpenPopupModal("Pick Sound")` per `focus-manager-design.md` modal-stack discipline.
  - Scrollable list of all `p.Audio` entries (library-sourced + user-imported, **no visual distinction per R9** — they're all just AudioSamples).
  - Each row: name + duration + (if isLoop) loop icon. Click selects + closes.
  - Bottom: Cancel button (closes without selection).
  - Imperative test interface per `file-picker-design.md`: `Open()`, `Pick(idx)`, `Cancel()`, `IsOpen()`.
- Per `dirty-state-ux.md`: every row mutation (Topic change, Sound change, SceneID change, Condition change, row add, row delete) calls MarkDirty.

**Patterns to follow:** existing `Project.Bindings` walk (likely in `pixelforge_studio/scripting/` or similar); existing `imgui.BeginCombo` for dropdowns; existing `OpenPopupModal` from idea #3's plan U4 (diff modal); existing topic aggregation at `inspector_test.go:50`; `topic_catalog.go:48-65` for pievent.EnumerateTargets usage.

**Test scenarios:**
- `TestBindingsPanel_RendersBindingsRows`: project with 3 AudioBindings; panel renders 3 rows; row contents match Topic/SampleName/etc.
- `TestBindingsPanel_TopicDropdownSourcesFromEnumerateTargets`: with engine targets `["loop.main", "key.main"]`; dropdown includes both + "<type custom...>".
- `TestBindingsPanel_TopicDropdownIncludesProjectDefinedTopics`: project has subscriptions for "game/PlayerDied"; dropdown includes it alongside engine targets.
- `TestBindingsPanel_TopicCustomTextEntry`: pick "<type custom...>"; text input appears; type "game/NewTopic"; row's Topic updates; MarkDirty.
- `TestBindingsPanel_PickSoundButtonOpensOverlay`: click Pick sound; SoundPickerOverlay.IsOpen() == true.
- `TestSoundPickerOverlay_ListsAllAudioSamples`: project with 5 AudioSamples (2 library + 3 user); overlay shows all 5; no visual distinction.
- `TestSoundPickerOverlay_PickUpdatesRowSampleName`: overlay open for binding row 2; pick sample index 3; row 2's SampleName == p.Audio[3].Name; overlay closes; MarkDirty.
- `TestSoundPickerOverlay_CancelLeavesRowUnchanged`: row 2 has SampleName="X"; open overlay, Cancel; row 2's SampleName still "X"; no MarkDirty.
- `TestBindingsPanel_AddBindingAppendsEmptyRow`: click Add Binding; p.Bindings grows by 1; new row has empty Topic, empty SampleName; MarkDirty.
- `TestBindingsPanel_DeleteRemovesRow`: 3 rows, click Delete on row 1; p.Bindings has 2 rows (rows 0 and 2 from original); MarkDirty.
- `TestBindingsPanel_EmptyProjectNoBindings`: project with no bindings; panel shows "0 bindings" + Add Binding button; no rows.
- `TestSoundPickerOverlay_RegistersWithModalStack`: per focus-manager-design.md, overlay open + Esc → overlay closes (modal-stack precedence).
- Covers AE3 (binding row appears), AE5 (library + user samples equal footing).

**Verification:** `go test ./pixelforge_studio/audiolib/...` passes; manual: open Audio workspace; click Bind on library patch; see row appear; click Topic dropdown, pick a topic; click Pick sound, see overlay listing all audio.

---

### U8. File menu integration — Import WAV + View → Audio entry

**Goal:** Add `File → Import WAV…` menu entry that opens a file picker, invokes `audiolib.Import(FileSource)` (U4), and shows a transient success/error toast. Add `View → Audio` menu entry (the activation logic at `keymap.go:71` and `file_menu.go:251-262` is already wired — just add the visible menu entry).

**Requirements:** R9 (user-imported WAVs via existing import path — this wires the missing import path itself).

**Dependencies:** U4 (Import function), U6 (workspace registered so activation is meaningful).

**Files:**
- `pixelforge_studio/editor/file_menu.go` (MODIFY — add Import WAV under File menu; add Audio entry under View menu)
- `pixelforge_studio/editor/audio_import_handler.go` (NEW — orchestrates file picker → audio.Import → toast)
- `pixelforge_studio/editor/audio_import_handler_test.go` (NEW)

**Approach:**
- File menu addition: under existing File items (currently New/Open/Save/Save As/Export/Quit), add:
  ```
  {Label: "Import WAV…", Shortcut: "Ctrl+Shift+W", OnSelect: func() { handleWAVImport(e) }}
  ```
  (Shortcut optional; pick something non-colliding.)
- View menu addition: under existing Scene/Palette items, add:
  ```
  {Label: "Audio", Shortcut: "Ctrl+4", OnSelect: func() { e.SetActiveWorkspaceByName("audio") }}
  ```
- `handleWAVImport(e)`:
  1. Open existing file picker scoped to `*.wav`.
  2. On confirm with path P:
     - Call `audiolib.Import(FileSource{Path: P}, e.Project, e.ProjectPath)`.
     - On success: show transient toast "Imported `<sampleName>`".
     - On error: show toast with error message.
  3. On cancel: do nothing.
- File picker integration per `file-picker-design.md`: drive imperatively for test predictability.

**Patterns to follow:** existing menu structure at `file_menu.go:166-204`; existing `widgets.MenuItem` with `OnSelect` closure; idea #3's plan U3 for File → Import PNG (analogous shape); existing file picker at `pixelforge_studio/editor/widgets/file_picker.go`.

**Test scenarios:**
- `TestFileMenu_ImportWAVEntryPresent`: after menu build, "Import WAV…" entry exists under File.
- `TestViewMenu_AudioEntryPresent`: after menu build, "Audio" entry exists under View.
- `TestViewMenu_AudioEntryActivatesWorkspace`: click View → Audio; `e.ActiveWorkspaceName() == "audio"`.
- `TestImportHandler_FilePickerConfirmInvokesImport`: simulate file picker confirm with valid WAV path; audiolib.Import called; success toast shown.
- `TestImportHandler_FilePickerCancelDoesNothing`: simulate cancel; no Import call; no toast.
- `TestImportHandler_ImportErrorShowsErrorToast`: simulate Import returning error; error toast shown with message.
- Covers F4 (user imports custom WAV via existing path).

**Verification:** `go test ./pixelforge_studio/editor/...` passes; manual: launch studio, File → Import WAV, pick a WAV, see toast; View → Audio shows workspace.

---

### U9. End-to-end audio acceptance tests

**Goal:** Integration tests covering AE1-AE6 + F1-F4. Loads fixtures, simulates designer actions via public APIs (Audition.Start, BindingsPanel.AddBinding, SoundPickerOverlay.Pick, etc.), verifies acceptance examples.

**Requirements:** R1-R10 covered transitively.

**Dependencies:** U1-U8 all merged.

**Files:**
- `pixelforge_studio/integration_test/audio_e2e_test.go` (NEW)
- `pixelforge_studio/integration_test/fixtures/jump_test.wav` (NEW — 8-bit mono 22050 Hz synthetic; ~0.2s)
- `pixelforge_studio/integration_test/fixtures/bgm_loop_test.wav` (NEW — 8-bit mono 22050 Hz synthetic loop; ~2s)
- `pixelforge_studio/integration_test/fixtures/audio_project_with_bindings.pforge` (NEW — 3 bindings, 2 library + 1 user-imported style AudioSamples)

**Test scenarios:**
- `TestE2E_AE1_PlayButtonAuditionsThenStops`: library has "shoot/laser_small"; Audition.Start(laser); ChannelActive returns true on its channel; Audition.Start(laser) again; ChannelActive returns false (or Stop semantics observable).
- `TestE2E_AE2_BGMFilterShowsBGMPatches`: catalog has 30 SFX + 4 BGM; filter="BGM" or "bgm"; picker rows = 4; all have loop icon.
- `TestE2E_AE3_BindCopiesWAVAndAppendsBinding`: library has "jump/spring"; call BindButton's handler → (a) `<assets>/audio/jump_spring.wav` exists on disk after; (b) p.Audio has entry with RelativePath="audio/jump_spring.wav"; (c) p.Bindings has new entry with SampleName="jump_spring", empty Topic.
- `TestE2E_AE4_BGMLoopsAutomaticallyDuringAudition`: Audition.Start(bgm_loop_test, isLoop=true); verify SetLoop called with LoopForward; assert (after WAV duration elapses, in a test using mock clock) channel is still active (loop continued).
- `TestE2E_AE5_PickSoundOverlayShowsBothOrigins`: project with 2 library-sourced + 2 user-imported AudioSamples; open SoundPickerOverlay for some binding row; overlay's list has 4 entries; no distinction in rendering.
- `TestE2E_AE6_ShippedBinarySelfContained`: **deferred / shape-only check** — confirm `pixelforge_ebiten` does not import `pixelforge_studio/audiolib`. The full ship-loop test depends on idea #7's plan; v1 of this plan only asserts the structural independence.
- `TestE2E_F1_PaintBindRunFlow`: end-to-end — Audition jump/spring (verifies playback works), Bind it onto "game/PlayerJumped", save project, load project, verify binding persists.
- `TestE2E_F2_BGMBindingFlow`: filter BGM, audition town/peaceful_loop, Bind to "scene.enter:title", verify binding persists.
- `TestE2E_F3_BuiltGameAudioMatch`: shape check only — verify the project's `.pforge-assets/audio/` directory exists after Bind and contains the expected WAV files. Cross-machine verification depends on idea #7.
- `TestE2E_F4_UserImportedWAVAppearsEqual`: import a custom WAV via audio_import_handler; user WAV appears in SoundPickerOverlay alongside library-sourced ones with no visual distinction.
- `TestE2E_AllocatorCoexistsWithAuditionAndRuntime`: simulate game playing BGM on Chan1 and SFX on Chan3; designer auditions another SFX; allocator picks Chan4 (round-robin avoids busy channels).
- `TestE2E_LegacyProjectWithoutAudioBackendStillLoads`: load a `.pforge` from before this plan; project loads; Audio workspace shows 0 bindings; no errors.

**Verification:** `go test ./pixelforge_studio/integration_test/...` passes; all 6 AEs green; F1, F2, F4 fully green; F3 partial (depends on idea #7 for the final cross-machine assertion).

---

## Scope Boundaries

### Deferred to Follow-Up Work

- **Mixer-lane panel** (4-channel Paula visualization with flash-on-voice-stealing) — debugging surface; deferred to v2. The allocator from U2 emits the signals; v2 wires the visualization.
- **`AudioBinding.ForceChannel` UI** — schema supports it; v1 doesn't surface in inspector. v2 mixer-lane panel exposes both visualization and override together.
- **Drag-drop from library to bindings** (ImGui's uintptr-payload mechanism) — per established cimgui-go-deferral convention; v1 ships "Bind..." button only. v2 polish.
- **Parametric synthesis** (recipe + sliders that synthesize WAVs at edit-time) — out per origin (patches are static WAVs).
- **Sheet-music notation editor** (4 voices = 4 staves) — out per origin.
- **MIDI / tracker editor** — out per origin.
- **Audio effects chain** (reverb, delay, low-pass, sidechain) — out per origin (Paula didn't have any).
- **Designer-built library patches** ("Save your own SFX as a library patch") — out per origin.
- **Adaptive / layered BGM** — out per origin.
- **Real-time waveform scrub** for loop-point finding — out per origin.
- **Audio quantizer / format converter** (auto-convert stereo / 16-bit to Paula format) — out per origin. Designer's WAVs must already be 8-bit mono PCM; DecodeWavOrErr's strict gate surfaces format errors.
- **Pitch / speed / volume modifiers on bound patches** — out per origin.
- **Per-binding sound preview "live" in the bindings table** (audition a binding's sample directly from its row) — minor polish; v2.

### Outside this product's identity

- Community patch gallery (download patches from the internet at runtime or studio-time).
- Mic-capture authoring (Ebitengine has no mic API).
- Browser-based / mobile audio workspace.
- AI-assisted patch suggestions ("here's a sound that fits your game").

---

## Key Technical Decisions

- **Zero external dependencies for code.** Four candidates evaluated (audio playback libs, sfxr-as-runtime-synth, ImGui drag-drop libs, WAV decode libs); all rejected via leverage doctrine. Total custom ~420 LOC across all units.
- **Sound-design source: sfxr/Bfxr/jsfxr presets → exported WAVs as primary; CC0 sourcing for BGM as fallback.** Recommendation, not mandate — implementer may use a different source if equivalent quality is achieved at lower time cost. Hand-crafted via FamiTracker/FamiStudio is acceptable but more time-intensive.
- **Paula auto-allocator is NEW code** (U2). Brainstorm called it "existing"; research confirmed it doesn't exist. Adding it is in-scope; without it, R10 isn't satisfied and the runtime can't route bound samples. Singleton in v1; per-project instance is a v2 refinement.
- **BGM stealing policy: new BGM steals oldest BGM channel.** Brief glitch on swap; designers will hear this as "the new music starts." Alternative (ignore new BGM if both channels busy) was rejected because it surprises the designer ("why didn't my BGM start?").
- **SFX stealing policy: round-robin Chan3/Chan4, steal oldest when both busy.** Matches NES feel — SFX can interrupt each other; designers expect this.
- **Drag-drop is deferred** (established convention). v1 uses "Bind..." button + "Pick sound..." overlay. Convention citation: ImGui migration plan U7, idea #5 plan U7.
- **`audio.Import` mirrors `palette.Import`** exactly in shape — file path/bytes → decode/validate → copy into `.pforge-assets/audio/` → append to `p.Audio` → MarkDirty. Implementer can take this directly from `pixelforge_studio/palette/import_pipeline.go:40-107`.
- **Library WAV copy semantics on first Bind:** designed for ship-loop correctness — the project file references only files in its own `.pforge-assets/`, so the shipped binary has zero studio dependency. Trade-off: 30 KB per used library patch duplicated into the project. Acceptable per origin Key Decision #6.
- **Topic dropdown sources from `pievent.EnumerateTargets()` + project topics + free-text fallback.** Three-source hybrid. Engine targets are discoverable; project-defined topics are project-specific; free text covers the long tail. Matches `topic_catalog.go:48-65` pattern.
- **Sound picker overlay is a modal-stack participant** per `focus-manager-design.md`. Imperative test interface per `file-picker-design.md`.
- **Editor audio backend init is its own unit** (U1). The studio currently uses `panicBackend{}` by default; without this fix, every Play call crashes. Mirrors `pixelforge_ebiten/internal/ebitengame.go:22-24`; may require re-exporting the helper.
- **No `omitempty` on AudioSample / AudioBinding** — per existing `pixelforge_project/audio.go` discipline (deterministic save). v1 doesn't change schema.
- **Library catalog metadata in JSON, not Go code.** Curators can edit `catalog.json` directly to add/remove patches without recompiling. Defensive loader skips broken entries.
- **Embedded library validates at load time** per `editor-pforge-schema-shape.md` defensiveness — malformed WAV in bundle → log + skip, never panic.
- **Workspace name "audio" matches existing keymap action `workspace.audio`** at `keymap.go:71` — activation is already wired; this plan just adds the visible menu entry.
- **AudioBinding.Topic field name (actual)** — not "EventTopic" as brainstorm wrote. Plan uses real field name.

---

## Dependencies / Assumptions

- **Strict dependency on idea #7's plan** for ship-loop outcome (R8 / AE6). The cross-machine "classmate plays the game" demo requires idea #7's Capsule + Build pipeline. v1 of this plan delivers everything testable in the editor preview; ship-loop test in U9 is shape-only until idea #7 lands.
- **Existing `pixelforge_audio` package** — Paula mixer (Play/LoadSample/SetLoop/ChannelActive/ClearChan); decode (DecodeWavOrErr). Unchanged in v1 except adding the new `allocator.go`.
- **Existing `AudioSample` + `AudioBinding` schemas** — unchanged; this plan adds no schema fields.
- **Existing `pievent.RegisterTarget` + `EnumerateTargets`** registry — bindings table topic dropdown sources from this.
- **Existing `Workspace` interface** + `RegisterWorkspace` pattern from post-ImGui-migration U3 — Audio workspace plugs into this.
- **Existing `palette.Import`** at `pixelforge_studio/palette/import_pipeline.go:40-107` — `audio.Import` mirrors its shape.
- **Existing keymap dispatch** for `workspace.audio` (`keymap.go:71`, `file_menu.go:251-262`) — already wired; this plan just adds the visible View menu entry.
- **Existing `AssetsDir(pforgePath)` convention** (`pixelforge_project/loader.go:120-125`) — library WAV copies land at `<assets>/audio/`.
- **`docs/solutions/`** anchors: `editor-pforge-schema-shape.md` (defensive load), `dirty-state-ux.md` (MarkDirty on every mutation), `focus-manager-design.md` (modal-stack), `always-on-game-embedding.md` (one audio path), `file-picker-design.md` (imperative test interface).
- **`go:embed` directory-pattern works** for the library tree (precedent: imgui_theme.go single file; decode_test.go multiple WAVs; new use case is directory tree). Verified plausible from Go stdlib docs; no project blocker.
- **Sound-design work is in scope** per origin's Dependencies / Assumptions section. Curator (implementer or sound designer) produces the WAVs. Plan ships with minimum-one-per-category target; ~30 SFX + 4-5 BGM is the goal but not a hard gate.
- **`pixelforge_ebiten/internal/audio.StartAudioBackend`** may need to be re-exported (move out of `internal/`) so the studio can call it. Implementer decision per U1.

---

## Risk Analysis & Mitigation

| Risk | Severity | Mitigation |
|---|---|---|
| `internal/audio.StartAudioBackend` can't be cleanly re-exported without breaking the runtime side | Medium | U1 has two paths: (a) move package out of `internal/`; (b) duplicate the 3-line init in a new shared package. Both are low-risk; implementer picks per actual import constraints. |
| Sound-design work slips, library ships near-empty | Medium | Catalog code (U3) supports any number of patches; picker just shows whatever's in catalog.json. v1 ships with whatever curator produced; the wiring is the load-bearing deliverable. Document actual ship count in verification. |
| sfxr / Bfxr presets aren't NES-authentic enough for the brainstorm's vibe | Low | Sfxr's "Mario-jump"-style presets are recognizably NES-class; deviation from "perfect NES sound" is acceptable for v1 since designers don't have a baseline to compare against. If quality is the blocker, swap to CC0 sourcing or FamiTracker for affected patches. |
| Paula allocator's "steal oldest" logic produces audible glitches in real use | Medium | Trade-off acknowledged in Key Technical Decisions. v2 mixer-lane panel surfaces the steal events visually so designers learn to design around channel limits. If glitches are a v1 ship blocker, implement smoother fade-out on steal (~50 LOC). |
| Audition uses the same channels as the running game, causing surprising audio collisions during authoring | Medium | This is *correct* behavior per `always-on-game-embedding.md` (one audio path). Designer can pause the game preview via existing chrome-visibility mechanism. If audition-vs-game collision is a UX blocker, add an "audition pauses game" preference toggle (v2). |
| File picker doesn't support `*.wav` filter | Low | Existing file picker (`pixelforge_studio/editor/widgets/file_picker.go`) supports extension filtering per idea #3 plan U3 usage; verify on first execution. |
| `pievent.EnumerateTargets()` is empty in the studio context | Medium | Engine targets register lazily on package import (per `pixelforge_studio/scripting/topic_catalog.go:48-65`). The studio imports the necessary engine packages on startup; targets should be available. Document fallback: if empty, the dropdown shows only project-defined topics + custom-text option. |
| Drag-drop is a v2 polish but designers expect it from modern editors | Low | Brainstorm explicitly defers; "Bind..." button has acceptable UX. If demand surfaces post-ship, the cimgui-go drag-drop deferral could be revisited as its own targeted plan. |
| Project save with unbound AudioSamples (imported but never used) leaves orphan files in `.pforge-assets/audio/` | Low | Existing `validateAssets` at `loader.go:130-` flags orphans. v2 could add an "Unused audio" cleanup action. Not a v1 blocker. |
| `MarkDirty` on file-copy side-effect creates ambiguous undo semantics | Medium | Per `dirty-state-ux.md`: file is on disk after Bind; undoing the binding row does NOT delete the file (one-way commit for file-system side-effects). Document this clearly; designers learn the model quickly. |
| Library catalog.json drifts out of sync with embedded WAV tree (curator adds WAV but forgets catalog entry, or vice versa) | Low | `TestLoadCatalog_PatchWithMissingFSPathSkipped` catches catalog→WAV mismatches. WAV→catalog mismatches are harmless (file just isn't in picker). Curator workflow: edit catalog.json + drop WAV together. |
| The `Play` step in the runtime scripting catalog doesn't currently exist OR doesn't call the allocator | Medium | U2 instructs the implementer to verify and wire. If Play step doesn't exist yet, that's a separate concern (likely part of idea #5's verb-sheet or a future runtime-event work) — flag at execution time. |

---

## System-Wide Impact

**New packages introduced:** `pixelforge_studio/audiolib/` (the new workspace + library + import + audition). Sibling to existing `palette/`, `capture/`, `scripting/`.

**Modified packages:**
- `pixelforge_audio` — new `allocator.go` (engine-side); no API changes to existing files.
- `pixelforge_ebiten/internal/audio` — possibly re-exported (move to `pixelforge_ebiten/audio` or new package).
- `pixelforge_studio` — `main.go` adds backend init + `audiolib.RegisterWith(e)`.
- `pixelforge_studio/editor` — `file_menu.go` adds File → Import WAV + View → Audio; new `audio_import_handler.go`.
- `pixelforge_studio/scripting/catalog` — `Play` step (existing or new) routes through Allocator.

**Affected workflows:**
- **Designer authoring** — primary target. New workflow: open Audio workspace → audition library patches → Bind onto event topics → run game preview to hear bound audio.
- **Engine** — adds Allocator; no behavioral regression on existing audio playback (existing tests cover regression).
- **Shipped runtime** — uses the same Allocator the studio uses; no separate path. Audio routing in shipped game matches studio preview by construction.
- **Codegen / build pipeline (idea #7)** — must include `.pforge-assets/audio/` in shipped binary. Idea #7's plan owns that; this plan's R7/R8 verification depends on it.

**Documentation impact:**
- Post-v1, three `docs/solutions/` entries worth capturing:
  1. Allocator policy (BGM steal-oldest, SFX round-robin) and the trade-offs.
  2. Studio-bundle vs project-asset boundary (library WAV copied on first Bind for ship-loop correctness).
  3. Embedded asset library pattern (curated bundle + catalog.json + defensive load + go:embed directory).
- The brainstorm's R10 misclaim ("existing auto-allocator") is a lesson worth capturing — the solution doc should note that schema reservations require backing implementations.

**Operational / rollout:**
- Standard release. Coupled with idea #1 / idea #2 / idea #3 / (idea #7) in the same milestone.
- No fixture migration: existing projects load with empty `p.Audio` and `p.Bindings` — new fields not introduced.
- Audio workspace appears on first launch after this plan ships; no opt-in required.
- Sound-design work is a separate deliverable; plan can ship with a partial library and patches can land iteratively post-release.

---

## Notes for Implementer

**Coordination with other plans:**
1. This plan is independent of ideas #1, #2, #3 in terms of code surfaces — they don't conflict. Execute in any order relative to those.
2. The ship-loop test (AE6) **requires idea #7's plan** (Capsule + Build pipeline) to verify cross-machine. v1 of this plan delivers everything testable in the editor; the final ship test is a follow-up.
3. The `Play` step in the runtime scripting catalog needs to route through the new Allocator (U2). If `Play` doesn't currently exist, this plan's U2 should NOT also wire it from scratch — that's a runtime-event concern that may belong with idea #5's verb-sheet plan or a future runtime-bindings unit. Implementer verifies and either: (a) extends an existing Play step's channel selection; (b) flags the gap and creates a follow-up unit; (c) defers runtime Play wiring (the studio audition still works, but the shipped game wouldn't play bound audio until the runtime step is implemented).

**Sound-design source decision (recommended):**
- If you have ~half a day for asset work: use sfxr (Mac/Linux/Windows downloadable) with the "Mario Jump" / "Laser Shoot" / "Coin Pickup" presets. Export each as 8-bit mono PCM WAV. Drop into `cart_assets/library/sfx/<category>/`. Update `catalog.json`.
- For BGM loops: search OpenGameArt for "NES-style chiptune CC0"; download 8-bit mono ogg/mp3; convert with `ffmpeg -acodec pcm_u8 -ac 1 -ar 22050 input.ogg output.wav`; trim to even-length loop in Audacity; drop into `cart_assets/library/bgm/<category>/`.
- Library can ship with N < 30 patches in v1; the picker handles whatever count. The bet is that the wiring works end-to-end, not that the patch count is exhaustive.
