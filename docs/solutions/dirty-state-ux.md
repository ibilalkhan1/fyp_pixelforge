---
title: Dirty-state UX
milestones: M1, M1.5
last_verified: docs/plans/2026-05-15-001-feat-pixelforge-no-code-editor-plan.md
---

# Dirty-state UX

## Context

The editor mutates the in-memory project on nearly every UI
interaction. Without a "dirty" signal, the user can't tell whether
their changes are saved, and the Quit / Open / New operations risk
silently dropping work.

## What we did

- The editor exposes `IsDirty() bool` / `MarkDirty()` / `ClearDirty()`.
- `PromptIfDirty(title, msg, action)` is the single seam every
  potentially-destructive operation routes through. It runs `action`
  immediately on a clean project and otherwise opens the confirm
  modal.
- Save handlers call `ClearDirty()` on success; load handlers call
  it after `SetProject`.
- The title bar appends a `*` suffix when dirty.

## Why it works

- **Single seam.** Every "are you sure?" prompt lives in one
  function — adding a new destructive action means adding a
  `PromptIfDirty` call, not a new modal flow.
- **Composable with modal precedence.** PromptIfDirty's modal
  participates in the focus-manager modal stack like any other.
- **Cheap to reason about.** The dirty flag is a `bool`; no
  observer pattern or pub-sub overhead.

## Alternatives considered

- **Modification-time tracking.** More precise but the user-visible
  signal is still binary, so the extra bookkeeping bought nothing.
- **Per-component dirty flags.** Useful for "save just this
  workspace" but unnecessary at M1 when the file is monolithic.

## When to apply this pattern

- Editors with monolithic save units (.pforge is one file).
- Cases where you can route every destructive op through a single
  prompt function.

## References

- `pixelforge_studio/editor/editor.go` — `PromptIfDirty`
- `pixelforge_studio/editor/confirm_modal.go`
