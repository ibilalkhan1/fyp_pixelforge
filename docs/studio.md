# Pixelforge Studio

> **Status: M1 — Foundation rewrite in progress.** The legacy studio
> documented in this file's previous revision has been replaced by a
> ground-up rebuild. This document will return with screenshots and a
> full user guide when the M3 milestone (Editor-as-Pixelforge-Cart)
> lands. Until then, see the plan and schema reference linked below.

## Where things stand (M0 + M1 complete)

The new studio is structured as eight milestones; M0 (teardown + new
shell) and M1 (`.pforge` schema + component registry) have shipped.
Concretely, today's binary:

- Opens a 1280×800 window with title bar, asset-browser placeholder,
  canvas placeholder, inspector placeholder, and status bar.
- Loads and persists user settings (window size, theme, recent
  projects) under your platform's user config directory.
- Carries a complete `.pforge` JSON schema covering screen size,
  palette + 4 ColorTables + presets + animations, sprites, audio,
  scenes/entities/components, behavior graphs, and event subscriptions.
- Provides a reflection-driven component registry (`pfcomponent`) that
  auto-emits inspector widgets from `pf:"..."` struct tags.
- Exports a self-contained Go project that builds and runs on any
  machine — the hardcoded `/home/tux/...` `replace` directive that
  blocked v1 export is gone, replaced by auto-detected vendor /
  dev-replace / published-version strategies.
- Renders an auto-generated inspector for any selected entity using
  widgets derived from registered component tags.

## What's not yet implemented

Palette editing UI, asset import pipeline, audio editor, visual
scripting, continuous capture, and procedural level graphs all live
behind upcoming milestones M2–M7. The plan at
[`docs/plans/2026-05-15-001-feat-pixelforge-no-code-editor-plan.md`](plans/2026-05-15-001-feat-pixelforge-no-code-editor-plan.md)
documents milestone scope and dependencies.

## Running today

```bash
go run ./pixelforge_studio
```

You'll see the chrome with placeholder labels. Project load/save is in
place via the API; UI for file → open/save lands with M2.

## Project file format

See [`pforge-schema.md`](pforge-schema.md) for the v1 wire format.
