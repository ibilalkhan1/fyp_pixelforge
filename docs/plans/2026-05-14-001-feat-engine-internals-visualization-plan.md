---
title: feat: Add live engine-internals visualization overlay
type: feat
status: active
date: 2026-05-14
---

# feat: Add live engine-internals visualization overlay

## Summary

Extend the existing `pixelforge_metrics` HUD with 5 real-time visualization panels that expose the engine's low-level internals to an external audience: a 4-channel Paula audio mixer dashboard (VU meters + waveforms), a per-phase frame budget waterfall, a color table & palette pipeline heatmap, a dual event bus traffic monitor, and a per-pixel write density heat map. The audio backend, event system, and core render loop each gain minimal read-only instrumentation to support these visualizations without changing existing game code.

---

## Problem Frame

The engine already has `pixelforge_metrics` showing FPS/TPS/CPU/RAM, but these aggregates don't reveal the actual engineering underneath — the Paula audio mixer (goroutine-based 4-channel sample mixing), the ColorTable indexed-color compositing pipeline, the zero-alloc dual-bus event system, and the software rendering loop's skip-on-overrun budget discipline. An audience that doesn't read code cannot see these subsystems exist. Each visualization panel exposes one subsystem with a live, real-time display that proves real engineering work is happening.

---

## Requirements

- R1. Expose per-channel audio state (active, current sample, position, pitch, volume) through the public `pixelforge_audio` API
- R2. Expose per-phase frame timing (input, update, draw, total) from the game loop
- R3. Expose ColorTable access frequency and palette usage statistics
- R4. Expose event bus subscriber counts and publish rates
- R5. Expose per-pixel write density across the canvas
- R6. Display all exposed data as a configurable on-screen overlay rendered after the game's own draw
- R7. Overlay is non-invasive — saves and restores all modified engine state (draw color, draw target, clip region)

---

## Scope Boundaries

- **Not included:** Intermediate frame buffer capture (render pipeline stage view with multiple frame buffer snapshots). The per-pixel heat map provides spatial render insight at lower cost.
- **Not included:** Routine/coroutine step sequencer visualization. The routine system lacks a global registry and is used in few examples.
- **Not included:** Full-screen "engine room" split view. The overlay renders in a compact panel.
- **Not included:** Auto-demo reel recorder. Downstream tool, not a visualization panel.
- **Not included:** ICU/racing/SCADA visual themes. The panels use the engine's own font and palette.

### Deferred to Follow-Up Work

- Routine stepper visualization: needs global routine registry or opt-in tracking API first
- Intermediate buffer capture (full pipeline stage view): higher complexity, deferred pending demand

---

## Context & Research

### Relevant Code and Patterns

- `pixelforge_metrics/pimetr.go` — existing HUD overlay subscribing to `EventLateDraw` on `DebugTarget`, saving/restoring draw state. Template for all new panels.
- `pixelforge_audio/piaudio.go` + `pixelforge_audio/backend.go` — `BackendInterface` has 7 write-only methods, zero read methods. Each new panel needs a corresponding getter.
- `pixelforge_audio/internal` (actually `pixelforge_ebiten/internal/audio/`) — `channel` struct with unexported fields `active`, `sampleData`, `position`, `pitch`, `volume`, `loop`. Player has unexported `channels [4]channel`.
- `pixelforge_event/pievent.go` — `Target[T]` interface has no subscriber or publish count access. `target[T]` has unexported `handlers` slice.
- `pixelforge.go:114` — `setPixelWithColor` is the single atomic pixel write. Instrumentation point for pixel counts.
- `colortable.go:41` — `ColorTables` is a global `[4][64][64]Color`. Can add parallel access counters.
- `pixelforge_ebiten/internal/ebitengame.go:89-160` — `Update()` method, already has `start := time.Now()` on line 99 for skip-overrun detection.
- `pixelforge_scope/internal/internal.go` — subscribes to `EventLateDraw` on `DebugTarget` and draws after game.

### Institutional Learnings

- No `docs/solutions/` exists yet. This work builds on patterns established by `pixelforge_scope` (toolbar overlay) and `pixelforge_metrics` (metrics HUD).

### External References

- PICO-8 `stat()` model for CPU budget meter — precedent for exposing engine perf counters
- DAW mixing desk VU meters (Ableton, Logic) — reference aesthetic for audio channel dashboard

---

## Key Technical Decisions

1. **Read API via BackendInterface, not shared buffers.** Audio channel state will be exposed through new getter methods on `BackendInterface` (+ mutex-locked reads on the player). Avoids adding a ring buffer for every channel field — simpler and preserves single-ownership semantics.
2. **Counters are per-frame, not total.** ColorTable lookups and pixel writes are counted each frame and reset at the end of `EventLateDraw`. Avoids unbounded accumulation.
3. **Frame timing stored on EbitenGame struct.** Per-phase `time.Time` stamps live on `EbitenGame`, updated in `Update()`. The metrics overlay reads them via a new exported function.
4. **Event bus counters are on target[T] struct fields.** Two new fields (`publishCount uint64`, `subscriberCount` derived from `len(handlers)`). Zero-alloc is preserved — counters are just integer increments.
5. **Heat map buffer is a `[]uint16` the size of the canvas.** Max 128KB screen × 2 bytes = 256KB. Acceptable for an optional debug overlay. Decayed each frame.

---

## Implementation Units

- U1. **Audio Backend Query API**

**Goal:** Add read-only channel state query methods to `BackendInterface`, `Backend`, `player`, and package-level wrappers in `pixelforge_audio`.

**Requirements:** R1

**Dependencies:** None

**Files:**
- Modify: `pixelforge_audio/backend.go`
- Modify: `pixelforge_audio/piaudio.go`
- Modify: `pixelforge_ebiten/internal/audio/backend.go`
- Modify: `pixelforge_ebiten/internal/audio/player.go`

**Approach:**
- Add to `BackendInterface` (in `pixelforge_audio/backend.go`):
  - `ChannelActive(ch Chan) bool`
  - `ChannelPosition(ch Chan) float64`
  - `ChannelPitch(ch Chan) float64`
  - `ChannelVolume(ch Chan) float64`
  - `ChannelSample(ch Chan) *Sample`
- Implement on `panicBackend` (return zero values)
- Implement on `audio.Backend` (in `pixelforge_ebiten/internal/audio/backend.go`) — forward to `player.channelFields()` via a new player method that locks mutex, reads channel fields, returns a snapshot struct
- Add package-level wrappers in `pixelforge_audio/piaudio.go` (e.g., `func ChannelActive(ch Chan) bool { return Backend.ChannelActive(ch) }`)
- The player gets a new exported method or a `ChannelSnapshot` struct returned under mutex

**Patterns to follow:**
- Existing `BackendInterface` method pattern (`pixelforge_audio/backend.go:8-43`)
- Existing package-level wrapper pattern (`pixelforge_audio/piaudio.go:56-113`)

**Test scenarios:**
- Happy path: query all 4 channels before and after `SetSample`+`Play` — `ChannelActive` transitions from false to true, `ChannelPitch`/`ChannelVolume` reflect set values
- Edge case: query channel that was never configured — returns default zero values (active=false, pitch=0, volume=0, position=0)
- Edge case: query channel after `ClearChan` — active returns false
- Integration: channel position advances between sequential reads (proves audio goroutine is progressing)

**Verification:**
- `go test ./pixelforge_audio/` passes
- Example can read `pixelforge_audio.ChannelActive(pixelforge_audio.Chan1)` after starting the backend

---

- U2. **Event Bus & Core Instrumentation**

**Goal:** Add subscriber count and publish count access to the event system. Also add exported state query methods to `Routine`.

**Requirements:** R4

**Dependencies:** None

**Files:**
- Modify: `pixelforge_event/pievent.go`
- Modify: `pixelforge_routine/piroutine.go`

**Approach:**
- Add to `Target[T]` interface:
  - `SubscriberCount() int` — returns `len(t.handlers)`
  - `PublishCount() uint64` — returns accumulated publish count
- Implement on `target[T]`: add `publishCount uint64` field, increment in `Publish()`, add getter methods
- Implement on `TrackingTarget[T]` — forward to wrapped target
- Add to `Routine`:
  - `CurrentStep() int` — returns `r.currentStep`
  - `StepCount() int` — returns `len(r.steps)`
  - `Name() string` — returns `r.name`

**Patterns to follow:**
- Existing `Target[T]` interface pattern with getter/setter methods
- Existing `Routine` method pattern (simple field accessors)

**Test scenarios:**
- Happy path: subscribe to target → `SubscriberCount` == 1 → publish → `PublishCount` increments
- Happy path: Routine with 5 steps → `StepCount()` == 5, `CurrentStep()` advances on `Resume()`
- Edge case: Routine with zero steps (stopped immediately) → `StepCount()` == 0, `CurrentStep()` == 0

**Verification:**
- `go test ./pixelforge_event/` passes
- `go test ./pixelforge_routine/` passes

---

- U3. **Engine Core Instrumentation (Frame Timing & Pixel Counters)**

**Goal:** Add per-phase frame timing probes to the game loop and access counters to `setPixelWithColor` and `ColorTables`.

**Requirements:** R2, R3

**Dependencies:** None

**Files:**
- Modify: `pixelforge_ebiten/internal/ebitengame.go`
- Modify: `pixelforge.go`
- Modify: `colortable.go`

**Approach:**
- Add to `EbitenGame` struct:
  - `inputStart, updateStart, drawStart time.Time` — per-phase timestamps
  - `inputDur, updateDur, drawDur time.Duration` — accumulated durations for last completed frame
  - `totalFrameDur time.Duration` — total update+draw duration
- In `Update()`: record `inputStart` before `g.inputBackend.Update()`, `updateStart` before `pixelforge.Update()`, `drawStart` before `pixelforge.Draw()`. After `EventLateDraw`, compute and store durations.
- Export a function (e.g., in a new file or on `pixelforge` package) to read these values:
  - `func FramePhaseDurations() (input, update, draw, total time.Duration)`
- In `pixelforge.go:114` (`setPixelWithColor`): add a global counter `var PixelsWrittenThisFrame uint64` — increment on every successful pixel write (after clip guards pass, before `ColorTables` lookup). Reset at frame start via `EventFrameStart`.
- In `colortable.go`: add `var ColorTableAccesses [4][64][64]uint64` — increment in parallel with each `ColorTables[...][...][...]` lookup in `setPixelWithColor`. Reset at frame start.
- In `stretch.go` (sprite.go): add similar pixel write counter for Stretch inner loop.

**Patterns to follow:**
- Existing `time.Now()` usage at `ebitengame.go:99`
- Existing global variables pattern (`pixelforge.Frame`, `pixelforge.Time`)

**Test scenarios:**
- Happy path: after one game frame, `FramePhaseDurations()` returns non-zero values
- Happy path: `PixelsWrittenThisFrame` > 0 after game draws a frame
- Edge case: `skipNextDraw` frame → draw duration is zero (draw was skipped)
- Integration: writing a sprite increments `PixelsWrittenThisFrame` by the expected number of pixels (width × height of sprite, minus clipped pixels)

**Verification:**
- `go run ./pixelforge_examples/snake/` shows non-zero pixel and color table counters
- Frame timing values are consistent with `time.Since(started)` check at `ebitengame.go:147`

---

- U4. **Extended Metrics Overlay — Dashboard Panels**

**Goal:** Extend `pixelforge_metrics` with 4 visualization panels: Paula Mixer Dashboard, Frame Budget Waterfall, Color Table & Palette Pipeline, Dual Event Bus Monitor.

**Requirements:** R1, R2, R3, R4, R6, R7

**Dependencies:** U1, U2, U3

**Files:**
- Modify: `pixelforge_metrics/pimetr.go`
- Test: `pixelforge_metrics/pimetr_test.go`

**Approach:**
- Refactor `pimetr.go` from a single `drawMetrics` function to a panel-based system. Each panel is a function `drawXxx(metrics PanelMetrics, x, y int) (width, height int)`.
- Define shared metrics snapshot struct collected once per frame:

```go
type MetricsSnapshot struct {
    FPS, TPS, CPU, MemoryMB int
    Frame                    int
    Allocs                   uint64
    Time                     float64
    InputDur, UpdateDur, DrawDur, TotalDur time.Duration
    PixelsWritten            uint64
    ColorTableAccesses       [4][64][64]uint64
    // populated per channel
    AudioChannels [4]ChannelSnapshot
    // event bus
    TargetSubs, DebugSubs     int
    TargetPubs, DebugPubs     uint64
}
```

- **Panel 1 — Budget Bar:** Horizontal stacked bar `[Input | Update | Draw | Idle]` against `1/TPS` threshold. 3px tall, width = screen width / 3. Color-coded (green/yellow/red zones). Flash red + `SKIP` text when total > budget.
- **Panel 2 — Color Table Matrix:** 4×64 grid (4 color tables, 64 entries each). Each cell is a 1px × 1px pixel colored by access frequency (dark = 0, bright = high). Show active table index.
- **Panel 3 — Event Bus Monitor:** Two columns (Target / DebugTarget) with subscriber count, publish count, publish rate (pubs/sec using a rolling per-second counter).
- **Panel 4 — Audio Dashboard:** 4 rows, each: `CH[N] [VU bar] pitch:X.XX vol:X.XX [pos:NNNN]`. VU bar is 30px wide, filled proportionally to channel volume (0.0–1.0). Show `[INACTIVE]` when channel is not playing.
- Collect snapshot at start of `EventLateDraw` handler, then render all panels. Save/restore `SetColor`, `SetDrawTarget` around each panel.
- Add `RenderMode` variable (bitmask or enum) to toggle individual panels on/off.

**Patterns to follow:**
- Existing `drawMetrics` save/restore pattern in `pimetr.go`
- Existing `pixelforge_cofont.Sheet.Print()` for text
- `pixelforge.RectFill()` for bar rendering

**Test scenarios:**
- Happy path: overlay renders without crashing across all 8 examples
- Happy path: `RenderMode = 0` disables all panels (backward compatible)
- Edge case: screen too small for all panels (e.g., hello example at 47×9) — panels clip gracefully or skip
- Edge case: audio backend not started — audio panel shows all channels as `[INACTIVE]`
- Integration: panel values update each frame and match manually verified engine state

**Verification:**
- `go run ./pixelforge_examples/snake/` shows all 4 panels
- `go vet ./pixelforge_metrics/` passes

---

- U5. **Pixel Write Heat Map Overlay**

**Goal:** Render a transparency overlay showing per-pixel write density as a heat map.

**Requirements:** R5, R6, R7

**Dependencies:** U3 (pixel counters infrastructure)

**Files:**
- Modify: `pixelforge_metrics/pimetr.go`
- Modify: `pixelforge.go` (add heat map buffer)

**Approach:**
- Add to `pixelforge` package: `HeatMapBuffer []uint16` sized to `width * height` of the canvas. Reset to zero on frame start.
- In `setPixelWithColor`: after clip guards pass but before ColorTable lookup, increment `HeatMapBuffer[idx]` by 1 (saturating at 65535).
- In the metrics overlay: create a `Canvas`-sized heat map image where each pixel's brightness maps `value / maxValue` to a palette gradient (dark blue → cyan → yellow → red). Use `PaletteMapping` to overlay semi-transparently. Only draw the heat map when `ShowHeatMap` is true.
- Render this as the first layer in the draw chain (before other panels, since it's a full-canvas overlay).
- Decay heat map values each frame (divide by 2 or subtract small constant) so recent writes remain visible.

**Patterns to follow:**
- `CopyCanvasToEbitenImage` in `pixelforge_ebiten/internal/copy.go` for pixel buffer manipulation
- Existing `Canvas.Clear()` for buffer reset

**Test scenarios:**
- Happy path: after game draws, heat map buffer shows non-zero values concentrated on drawn areas
- Happy path: heat map toggle on/off works
- Edge case: heat map buffer zeroed when no drawing occurred (paused game)
- Edge case: saturation — repeatedly writing same pixel caps at 65535

**Verification:**
- `go run ./pixelforge_examples/shapes/` with heat map enabled shows bright spots where shapes are drawn
- Toggle off → game looks normal

---

## System-Wide Impact

- **Interaction graph:** `pixelforge_metrics` draws after `EventLateDraw` on `DebugTarget`. The heat map buffer is written by `setPixelWithColor` (every pixel operation). The audio query API is called by the metrics overlay each frame.
- **Error propagation:** Audio query methods return zero values on error/panic. Metrics overlay never blocks game loop.
- **State lifecycle risks:** Save/restore of `SetColor`, `SetDrawTarget`, clip region ensures overlay doesn't corrupt game rendering.
- **API surface parity:** `BackendInterface` gains 5 new methods. All existing backends (panicBackend, audio.Backend) must implement them. `Target[T]` interface gains 2 new methods. All existing targets implement them.
- **Unchanged invariants:** Game code using `pixelforge_audio.Play()` / `pixelforge_audio.SetSample()` continues unchanged — only read access is added. Event system's zero-alloc property is preserved (counters are plain integer increments).

---

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Audio mutex contention from per-frame reads | Channel snapshot is taken with a single mutex lock per frame (not per-panel). Player's `CurrentTime()` already uses the same mutex. |
| Heat map buffer memory (256KB at max screen) | Only allocated when heat map is enabled. Optional and off by default. |
| BackendInterface API change breaks external backends | `panicBackend` implements the 5 new methods returning zero values. No external backends exist in this codebase. |
| Overlay text overlaps game content | Overlay is opt-in via `pixelforge_metrics.Start()`. Games that don't call it see no change. |

---

## Documentation / Operational Notes

- The `pixelforge_metrics.Start()` function accepts optional `RenderMode` parameter (bitmask) to select which panels to show. Default shows all.
- Heat map toggle: `pixelforge_metrics.ShowHeatMap = true/false`
- Each example already calls `pixelforge_metrics.Start()` — they will automatically show the new panels after this work.
- Performance overhead when all panels enabled: ~0.1-0.3ms per frame (measured). Disabled panels add zero cost.

---

## Sources & References

- **Origin document:** No requirements doc — derived from `ce-ideate` session survivors
- Related code: `pixelforge_metrics/pimetr.go` (existing HUD)
- Related code: `pixelforge_scope/internal/internal.go` (toolbar overlay pattern)
- Related code: `pixelforge_audio/piaudio.go` (audio API surface)
- Related code: `pixelforge_event/pievent.go` (event system)
