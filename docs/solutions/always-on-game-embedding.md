---
title: Always-on game embedding
milestones: M3
last_verified: docs/plans/2026-05-15-003-feat-pixelforge-editor-as-cart-and-gui-growth-plan.md
---

# Always-on game embedding

## Context

The studio needs to render the user's game *while* the editor chrome
is also visible. Two competing requirements:

1. The game should advance at TPS so animation, input, and behaviour
   feel live during editing.
2. The chrome must remain crisp and responsive to editor-side input.

## What we did

- `chromeVisibility` owns a project-screen-sized canvas the studio
  paints the game into every tick.
- Esc toggles chrome visibility: when chrome is hidden, the game
  canvas fills the entire window; when visible, it sits inside the
  workspace region.
- Modal precedence (see [focus-manager-design](focus-manager-design.md))
  is enforced before the Esc toggle: open modals dismiss first.
- The game canvas dimensions track the project's `ScreenWidth × ScreenHeight`;
  swapping projects reallocates the canvas to match.

## Why it works

- **One render path.** Game + chrome share the same draw cycle;
  no separate "preview" mode.
- **Cheap toggle.** Hiding chrome doesn't pause the game — input
  keeps flowing, just routed to the game first.
- **Modal precedence is composable.** Adding a new modal type
  doesn't require adding code to the chrome-visibility path; the
  modal stack handles ordering.

## Alternatives considered

- **Separate Run / Edit modes.** Like Unity. Adds context-switch
  friction; the always-on path keeps the live-edit pitch true.
- **Game-as-workspace.** Treating the game as its own workspace
  swaps the canvas off when the user switches to Palette. Killed
  the "always-on" promise.

## When to apply this pattern

- Tools that hold a "live preview" of authored content.
- Cases where toggle responsiveness matters more than running the
  preview in isolation.

## References

- `pixelforge_studio/editor/chrome_visibility.go`
- `pixelforge_studio/editor/editor.go` — `handleEscape` modal precedence
