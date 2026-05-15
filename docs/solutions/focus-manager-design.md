---
title: Focus manager design
milestones: M3
last_verified: docs/plans/2026-05-15-003-feat-pixelforge-editor-as-cart-and-gui-growth-plan.md
---

# Focus manager design

## Context

Multiple canvas-resident input fields (TextInput, Dropdown, soon Slider
edits) share the same canvas. Tab-traversal between them needs a
deterministic rule. Modals further complicate the picture: an open
modal must consume keyboard input even when the user clicked outside
its body.

## What we did

- `pixelforge_gui.FocusManager` keeps a singly-linked focus chain of
  input widgets registered with it. Tab moves forward, Shift+Tab moves
  back, Esc dismisses the topmost focusable.
- Modals push themselves onto a `ModalStack`. When the stack is
  non-empty, Esc dismisses the top modal *before* the chrome-visibility
  toggle considers itself; this is the "modal precedence" rule.
- Input widgets check `FocusManager.HasFocus(self)` before consuming
  keyboard events. The cofont-rendered carets are driven by the same
  predicate so visual focus matches logical focus.

## Why it works

- **Deterministic and testable.** A linked-list focus chain is small
  enough to verify in unit tests without driving Ebitengine input.
- **Composable.** Workspaces register their fields with the same
  manager the chrome uses, so Tab works seamlessly across regions.
- **Precedence is one rule.** Editing modal-precedence in one place
  fixes it everywhere instead of debugging "why didn't my Esc fire?"
  across N components.

## Alternatives considered

- **Per-widget focus flags.** Worked at M0 with one input field;
  scales badly once N>3.
- **External event-bus.** `pievent`-driven focus would have made the
  contract harder to reason about for one-shot inputs.

## When to apply this pattern

- Any UI subsystem with two or more keyboard-receiving widgets.
- When a system has modal contexts that override the parent surface's
  default input handling.

## References

- `pixelforge_gui/focus.go`
- `pixelforge_gui/widgets/modal.go` — modal-stack integration
- `pixelforge_studio/editor/cart.go` — focus manager wiring per workspace
