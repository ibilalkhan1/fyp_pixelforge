---
title: Canvas-vs-native chrome split
milestones: M3, M3.1
last_verified: docs/plans/2026-05-15-004-feat-pixelforge-capture-spine-and-m3-cleanup-plan.md
---

# Canvas-vs-native chrome split

## Context

M3 set out to migrate the entire editor chrome onto a Pixelforge cart —
"the editor is itself a Pixelforge program" (R1). In practice we
shipped a **hybrid**: workspace area, asset browser, and inspector
chrome moved onto the canvas; menu bar, status bar, confirm modal, and
file picker stayed on the native overlay path. M3.1 (U43-U46) is what
finishes that migration.

## What we did

- Two parallel widget banks: `pixelforge_studio/editor/widgets/` (the
  *native* bank — `ebitenutil.DebugPrintAt`-backed) and
  `pixelforge_gui/widgets/` (the *canvas* bank — engine primitives).
- The editor cart's `DrawCanvas` path dispatches to the canvas bank
  via `CanvasWorkspace`; the legacy `Workspace.Draw` path continues
  to drive the native bank for chrome that hasn't migrated yet.
- A "retire in M3.3" comment block tops every native-bank file that
  the canvas equivalent supersedes; deletion happens only after
  parity tests verify the canvas dropdowns reach feature equivalence
  with their native counterparts.

## Why it works

- **Atomic flips are risky.** A single-PR rewrite of the chrome
  blocks the rest of the editor on getting *every* canvas widget
  pixel-perfect. The hybrid lets workspace work continue while the
  chrome migration lands incrementally.
- **The seams already existed.** `editor.CanvasWorkspace`,
  `RegisterWorkspace`'s name idempotency, and the cart's split
  `Draw` / `DrawCanvas` calls were sized for this exact migration.
- **Parity tests catch regressions.** The canvas menu bar and status
  bar were tested independently before they replaced the native
  versions in the cart's render path.

## Alternatives considered

- **All-canvas day-one.** Would have pushed M3 out by months; the
  native bank still drove every dropdown in the inspector.
- **Two long-lived banks.** Tempting but doubles the surface area
  contributors have to think about. The "retire in M3.3" markers
  keep the duplication intentional and time-boxed.

## When to apply this pattern

- Migrations that span >5 files in the rendering layer.
- Cases where the destination implementation is a brand-new
  abstraction that hasn't yet proven itself on small features.
- Cases where rollback is cheap because the legacy path remains
  compiled.

## References

- [`docs/plans/2026-05-15-003-…M3 plan`](../plans/2026-05-15-003-feat-pixelforge-editor-as-cart-and-gui-growth-plan.md)
- [`docs/plans/2026-05-15-004-…M4/M3.1 plan`](../plans/2026-05-15-004-feat-pixelforge-capture-spine-and-m3-cleanup-plan.md)
- `pixelforge_studio/editor/cart.go` — the dispatch seam.
- `pixelforge_studio/editor/workspaces.go` — `RegisterWorkspace` idempotency.
