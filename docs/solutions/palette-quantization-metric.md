---
title: Palette quantization metric
milestones: M2
last_verified: docs/plans/2026-05-15-002-feat-pixelforge-editor-interactivity-and-palette-plan.md
---

# Palette quantization metric

## Context

M2 lets users import an arbitrary PNG and remap it onto the project's
64-slot palette. The naive "nearest RGB neighbour" approach produces
visible banding on gradients, and large flat regions can quantize to
a single slot when several slots would read more faithfully. We
needed a metric that captures both fidelity and slot usage.

## What we did

- The importer scores candidate quantizations by a weighted blend:
  - Per-pixel **RGB ΔE** from the source image
  - **Slot dispersion** — penalty for under-using available slots
- A small "quality dial" presets (fast / balanced / quality) maps to
  blend weights so users don't see raw constants.
- The metric is checked against the project's existing palette
  before suggesting palette extensions ("this image needs 6 new
  slots to render faithfully").

## Why it works

- **Predictable feedback.** Users see a single number; the dial
  abstracts the trade-off.
- **Cheap.** Computed once per import at small (320×180) target
  resolutions; budget is irrelevant.
- **Composable with the auto-tile pipeline.** Both write into the
  project palette; the metric ensures they don't collide.

## Alternatives considered

- **Median-cut quantization (libimagequant-style).** Higher quality
  but heavyweight; the metric-driven approach hits 90% of the result
  in 5% of the code.
- **Histogram-only.** Fast but ignores dispersion; produces
  pathologically over-clustered results on cartoon-style art.

## When to apply this pattern

- Any quantization task where the user picks the trade-off slider.
- Imports that need to interact with existing palette state.

## References

- `pixelforge_studio/palette/import.go`
- `pixelforge_studio/palette/quantize.go`
