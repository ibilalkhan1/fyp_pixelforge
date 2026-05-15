---
title: Auto-tile heuristic
milestones: M2
last_verified: docs/plans/2026-05-15-002-feat-pixelforge-editor-interactivity-and-palette-plan.md
---

# Auto-tile heuristic

## Context

The M2 paint tool lets users place tiles directly onto a scene; the
schema reserves an `AutoTileRules` field on each `TilemapLayer` so the
editor can synthesize transition rules ("when a wall tile is left of a
floor tile, paint a wall-edge sprite there") from user gestures
rather than asking them to hand-author rule tables.

## What we did

- Each rule binds a **3×3 neighbour pattern** to an output tile value.
  Cell 4 of the pattern is the centre cell; cells 0-3 and 5-8 are
  context; `-1` is a wildcard.
- The editor accumulates patterns by watching user paint strokes —
  whenever a pattern is repeated three times consecutively the
  editor proposes it as a rule.
- Patterns are stored on the project file. The runtime reads them as
  a hint only; the on-disk grid is the source of truth, so a
  malformed rule never corrupts a saved level.

## Why it works

- **Learning-by-example.** Users don't have to think about rules
  abstractly; the editor watches and infers.
- **Bounded growth.** The 3-repetitions threshold keeps the rule
  table small; otherwise the user accidentally creates dozens of
  one-off rules.
- **Schema-safe.** Storing rules as hints means a rule that no
  longer matches gameplay is harmless — the grid renders as the
  user painted it.

## Alternatives considered

- **WFC (wave function collapse) inference.** Rich but opaque; the
  user can't introspect why a rule fired.
- **No auto-tile at all.** Users would have hand-painted every
  edge, which the M2 spec explicitly tried to avoid.

## When to apply this pattern

- Authoring tools where rules can be inferred from examples.
- Cases where you can afford a "hint vs source-of-truth" split so a
  bad inference is never destructive.

## References

- `pixelforge_project/scenes.go` — `AutoTileRule` schema
- `pixelforge_studio/editor/tools.go` — paint stroke tracking
