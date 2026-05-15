---
title: File picker design
milestones: M1.5, M3.1
last_verified: docs/plans/2026-05-15-004-feat-pixelforge-capture-spine-and-m3-cleanup-plan.md
---

# File picker design

## Context

The studio needs in-editor file picking for open / save / asset
import flows. Falling back to the OS picker fragments the chrome,
fails on Linux without a GTK file dialog, and breaks the
"editor-as-cart" identity claim. The picker has to be self-contained.

## What we did

- The picker is a modal with a path TextInput (breadcrumb), a
  Scrollable list of entries, and a footer with cancel/confirm
  buttons. In save mode it adds a name TextInput.
- Esc dismisses the topmost modal first; only an empty modal stack
  delegates Esc to the chrome-visibility toggle (see
  [focus-manager-design](focus-manager-design.md)).
- The picker auto-falls-back to the user's home directory when
  `StartPath` is unreadable or empty.
- File-mode confirm clicks fire on entry double-click; save-mode
  confirm requires non-empty name input.
- M3.1 ships a canvas-resident equivalent in
  `pixelforge_gui/widgets/file_picker.go` alongside the native bank;
  the native picker retires once parity tests pass.

## Why it works

- **No OS portability surface.** Works the same on every supported
  desktop platform without GTK / Cocoa shim code.
- **Modal-stack-friendly.** Plays nicely with confirm dialogs that
  fire from inside picker callbacks (e.g. "overwrite this file?").
- **Predictable in tests.** The picker can be driven imperatively
  via `Open`, `Enter(idx)`, `Confirm()`, `Cancel()` without any
  input layer mocking.

## Alternatives considered

- **OS native picker.** Inconsistent on Linux, alien-feeling on
  every platform when wrapped in a pixel-art chrome.
- **Drag-and-drop only.** Doesn't cover save flows.

## When to apply this pattern

- Self-contained editors that should not depend on the OS shell.
- Workflows where the picker callback may itself open another
  modal — the picker's modal-stack-aware Esc handling lets you
  layer.

## References

- `pixelforge_studio/editor/widgets/file_picker.go` — the native
  bank picker, retiring at M3.3.
- `pixelforge_gui/widgets/file_picker.go` — the canvas-resident
  equivalent shipped at U44.
