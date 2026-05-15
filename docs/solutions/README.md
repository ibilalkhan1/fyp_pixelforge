# Solutions Index

Curated institutional learnings from the Pixelforge Studio milestones.
Each entry captures a decision that survived implementation — the
*why* alongside the *what* — so the next milestone can pick up where
the last left off without relearning the same trade-offs.

## Chrome

- [canvas-vs-native-chrome-split.md](canvas-vs-native-chrome-split.md) — Why M3 shipped a hybrid chrome instead of an all-canvas migration.
- [file-picker-design.md](file-picker-design.md) — Modal stack + Esc precedence pattern.
- [focus-manager-design.md](focus-manager-design.md) — Tab traversal and the modal-precedence rules `pgui.FocusManager` enforces.

## Schema & Editor Identity

- [editor-pforge-schema-shape.md](editor-pforge-schema-shape.md) — Why the editor's `editor.pforge` fixture is additive-only.
- [always-on-game-embedding.md](always-on-game-embedding.md) — The Esc toggle, modal precedence, and game-canvas allocation.

## Palette & Tile

- [palette-quantization-metric.md](palette-quantization-metric.md) — M2 quantization metric and threshold heuristics.
- [auto-tile-heuristic.md](auto-tile-heuristic.md) — How auto-tile rules are derived from user-painted neighbour patterns.

## Interaction

- [dirty-state-ux.md](dirty-state-ux.md) — IsDirty / PromptIfDirty and the save-handler contract.

## Capture

- [ring-buffer-snapshot-store.md](ring-buffer-snapshot-store.md) — M4's recorder pattern and its reuse of piscope's primitive.

## Scripting

- [scripting-runtime-design.md](scripting-runtime-design.md) — M5's per-project Engine, Kind catalogs, non-generic `Inspectable` sidecar, and input-log replay synthesis.
