---
title: editor.pforge schema shape
milestones: M0-M3
last_verified: docs/plans/2026-05-15-003-feat-pixelforge-editor-as-cart-and-gui-growth-plan.md
---

# `editor.pforge` schema shape

## Context

The editor dogfoods Pixelforge by loading itself from an embedded
`editor.pforge` fixture. The fixture supplies chrome theme slots,
panel widths, and reserved hooks for future content. Schema decisions
made at M0 propagate every milestone — additions are easy, removals
are not.

## What we did

- The fixture is *additive*: every milestone adds fields to the
  `Theme` struct (FontName at M3, etc.) but never removes them.
- Default field values match the existing `DefaultEditorTheme` so
  older fixtures load cleanly without migration. `sanitize()` in
  `pixelforge_studio/editor/settings.go` repairs out-of-range fields
  rather than rejecting them.
- The fixture lives under `pixelforge_studio/editor/cart_assets/` and
  is embedded via `go:embed`. Failure to parse is logged but never
  fatal — the cart falls back to defaults so a broken fixture never
  crashes the studio.

## Why it works

- **No migration debt.** Additive evolution + sanitize on load means
  the schema can ship a new field every milestone without breaking
  existing project files.
- **Embedding survives binary distribution.** The fixture rides in
  the binary, so the user never needs to ship a separate asset.
- **Single source of truth.** Every chrome decision routes through
  the theme; there's never a "hardcoded vs themed" split.

## Alternatives considered

- **Schema versioning per field.** Overkill for a single fixture
  with a known consumer.
- **External JSON config.** Adds a deployment failure mode the
  embedded approach doesn't.

## When to apply this pattern

- Configuration data that ships with the binary.
- Schemas with one canonical consumer (you control both producer
  and consumer).
- Cases where the consumer can gracefully degrade on missing fields.

## References

- `pixelforge_studio/editor/cart_loader.go`
- `pixelforge_project/theme.go`
- `pixelforge_studio/editor/cart_assets/editor.pforge`
