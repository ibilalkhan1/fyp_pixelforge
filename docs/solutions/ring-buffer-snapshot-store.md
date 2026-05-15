---
title: Ring-buffer snapshot store
milestones: M4
last_verified: docs/plans/2026-05-15-004-feat-pixelforge-capture-spine-and-m3-cleanup-plan.md
---

# Ring-buffer snapshot store

## Context

M4's continuous-capture spine records every game frame so the user
can rewind, clip, regression-test, and share a bug repro. The
working set must stay bounded — at 30 TPS with 320×180 frames, a
naïve unbounded slice would consume gigabytes in minutes.

## What we did

- The recorder wraps `internal/pixelforge_ring.Buffer[Frame]` — the
  same primitive `pixelforge_scope` already uses for its piscope
  rewind buffer. Reuses pre-allocated frame storage so the steady
  state allocates nothing on the capture hot path.
- Each `Frame` carries: a paletted `pixelforge.Canvas` clone,
  palette + mapping, frame/tick numbers, and per-tick event + input
  logs.
- Default budget = 300 frames × 320×180 × 1 byte/pixel ≈ 17 MB.
- The recorder subscribes to `pixelforge_loop.Target()` for the
  `EventLateDraw` capture trigger and uses `SubscribeAll` on every
  input target (mouse, key, pad) to populate per-tick logs without
  knowing each target's payload shape.

## Why it works

- **Pattern reuse.** Lifting from `pixelforge_scope/internal/recorder.go`
  meant no new primitive had to be designed or tested.
- **Predictable memory.** Bounded budget + reusable canvas storage
  means the recorder's working set is provable.
- **Cheap scrub.** `pixelforge.Screen().SetData(frame.Canvas.Data())`
  is the cheapest possible "show frame N" path — it's what piscope
  rewind already uses.
- **Decoupled event taps.** `SubscribeAll` lets the recorder follow
  any pievent target without compile-time coupling to its payload.

## Alternatives considered

- **Encoded PNGs per frame.** Smaller in memory but slow to scrub;
  scrub latency dominates UX for the capture workspace.
- **External capture process.** Crosses the OS boundary and breaks
  the "single binary" promise.

## When to apply this pattern

- Any subsystem that needs bounded, cheap rewind of game state.
- Cases where the capture frequency is predictable (TPS-driven).
- When the work the consumer does per frame is light (memcpy +
  palette swap, no per-frame encoding).

## References

- `pixelforge_studio/capture/recorder.go`
- `pixelforge_scope/internal/recorder.go` — the canonical pattern.
- `internal/pixelforge_ring/piring.go` — the underlying primitive.
