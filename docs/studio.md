# Pixelforge Studio

> **Status: ImGui migration shipped.** The studio's chrome now renders
> through Dear ImGui via `cimgui-go/backend/ebiten-backend`. Dockable
> panels, persistent layouts, and an ImGui inspector replaced the M0–M2
> native chrome and the M3 cart-resident UI experiment. The Pixelforge
> engine itself is untouched and remains free of cgo / cimgui-go in its
> dependency graph.

## Tech stack

- **Pixelforge engine** — pure-Go retro game runtime (Ebitengine
  underneath). Game packages import only the engine and Ebitengine.
- **Studio chrome** — Dear ImGui 1.92.8 docking branch, via the
  `github.com/AllenDang/cimgui-go` Go bindings and the first-party
  `backend/ebiten-backend` integration. Pre-compiled static libs ship
  with cimgui-go; `go build` works without cmake on the supported
  desktop platforms (linux-x64, macos-arm64/x64, windows-x64).
- **`pixelforge_gui`** — the engine-side in-game UI package, frozen but
  preserved for `pixelforge_scope/` and `pixelforge_examples/gui/`. It
  is *not* used by the studio.

## What the studio gives you today

- **Dockable workspace layout.** The editor window is a single ImGui
  DockSpace. Assets, Inspector, Scene, Capture, Behavior, Palette, and
  any future workspace register as ImGui windows; users drag them
  between dock slots, tab them together, or float them inside the main
  window. The user's arrangement persists in `imgui.ini` under the
  platform user-config directory, alongside `settings.json`.
- **Scene as a docked image panel.** The running Pixelforge canvas
  renders to an off-screen texture (`backend.CreateTextureFromGame`)
  and shows inside an `imgui.Image` in the Scene window. Tools (Select,
  Place, Delete, Paint) live in a toolbar above the image; clicks are
  routed through to the canvas only when the image is focused +
  hovered, so toolbar and other-dock clicks don't fire scene tools.
- **Reflection-driven inspector.** Selecting an entity surfaces one
  `imgui.CollapsingHeader` per component and dispatches one ImGui
  widget per `pfcomponent.FieldMetadata` — slider, combo, color,
  checkbox, vector2, etc., all picked from the `pf:"..."` struct tag.
  Edits write back directly through the component's value map.
- **Theming.** Chrome colours come from the loaded `editor.pforge`
  theme palette (slot indices → ImGui Vec4 via the project's palette).
  Replacing the embedded fixture changes the editor's look without a
  recompile.
- **Capture + behaviour workspaces** rebuilt on ImGui. The substrate
  (recorder, ring buffer, behaviour graph runtime) is unchanged; only
  the editor surface ported.

## Running today

```bash
go run ./pixelforge_studio
```

The studio opens with the dockspace populated by Assets / Inspector /
Scene / Capture / Behavior / Palette windows. File menu and Ctrl+S /
Ctrl+O still drive load/save. Layout you arrange this session
re-opens the same way next session.

## Project file format

See [`pforge-schema.md`](pforge-schema.md) for the v1 wire format. The
schema is unchanged by the ImGui migration.

## Migration history

The ImGui migration is tracked in
[`docs/plans/2026-05-17-001-refactor-pixelforge-studio-imgui-migration-plan.md`](plans/2026-05-17-001-refactor-pixelforge-studio-imgui-migration-plan.md).
Plans 2026-05-15-001 / 002, 2026-05-16-001 / 002 are
**partially superseded** — their feature targets still apply, but their
widget-implementation details are replaced by ImGui equivalents. Plan
2026-05-15-003 (editor-as-cart + GUI growth) is **fully superseded**.
