---
date: 2026-05-17
topic: pixelforge-studio-imgui-migration
focus: hassle-free migration of Pixelforge Studio GUI from pigui/widgets to Dear ImGui via cimgui-go + first-party ebiten-backend
mode: repo-grounded
status: ideation (preceding ce-brainstorm)
---

# Ideation: Pixelforge Studio → Dear ImGui (cimgui-go) Migration

## Framing

User chose option 2 from the prior turn: Dear ImGui via cimgui-go with the Ebitengine backend. Two constraints from the user:

1. **"Hassle-free"** — minimize implementation cost and risk, not blast radius. Heavy changes are explicitly authorized.
2. **"Pixelforge itself might need heavy changes — don't shy away from them."** — the engine and its in-tree pigui library are on the table for restructuring if the move is grounded.

This ideation answers: *given what already exists in the codebase and what's committed in the plans, what concrete migration strategies are worth bringing forward to brainstorm/plan?*

## Grounding Context

### Codebase (current state, May 2026)

**Studio shell — `pixelforge_studio/editor/`:**
- `editor.go` — implements `ebiten.Game`. Holds two parallel chrome paths: native (`menuBar *widgets.MenuBar`, `statusMessage string`) and **latent** canvas-resident (`canvasMenuBar *pguiwidgets.MenuBar`, `canvasStatusBar *pguiwidgets.StatusBar`). Only the native path is wired into the active frame loop.
- `chrome.go` (~150 LOC of `ebitenutil.DebugPrintAt` + `vector.DrawFilledRect`) draws title bar / left panel / right panel / status bar from hardcoded rects recomputed every frame from window size.
- `cart.go` + `cart_loader.go` — "editor as cart": 1280×800 logical `pixelforge.Canvas`, owns a `pgui.Element` tree (`root`, `workspaceRoot`), loads theme from an embedded `editor.pforge` fixture. Gated behind a `CanvasWorkspace` interface check — **currently no workspace implements it**, so cart code is plumbed but inert.
- `inspector.go` — reflection-driven, reads `pf:"..."` struct tags via `pfcomponent.Get()`, caches widget instances by `(EntityID, CompIdx, FieldIdx)`. The dispatch table is dictated by `pfcomponent` metadata, not by the widget API.
- `workspaces.go` — `Workspace` interface (`Name`, `Draw`, `Update`) with `CanvasWorkspace` extension. Scene workspace is native-drawn; capture and scripting workspaces are pgui-based.
- 24 `_test.go` files covering chrome, canvas, inspector, workspaces, keymap, settings, cart, etc.

**Widget banks:**
- `pixelforge_gui/widgets/` — 17 widgets, ~3,200 LOC, all with tests. Most retained-mode `Element`-callback shape (button, dropdown, panel, scrollable, tabs, text_input, menu_bar, status_bar, modal, file_picker, node_graph, timeline, step_card, code_block, draggable, rule_row, confirm_modal).
- `pixelforge_studio/editor/widgets/` — a thinner native-overlay widget set used by current menu bar / file picker.

**pigui consumers outside `pixelforge_studio/editor`:**
- `pixelforge_examples/gui/main.go` — demo
- `pixelforge_scope/internal/{internal,gui}.go` — frame debugger toolbar
- `pixelforge_studio/capture/{workspace,timeline}.go` — M4 surface, pgui-based
- `pixelforge_studio/scripting/{workspace,lane_editor}.go` — M5 surface, pgui-based

**Games (snake, pacman, piano, hello)** — no pigui use. Engine is GUI-lib-agnostic.

### Plans (committed, May 15–16 archive)

The May 15 master plan (`feat-pixelforge-no-code-editor-plan.md`) and the May 15 003 plan (`feat-pixelforge-editor-as-cart-and-gui-growth-plan.md`) commit M3 to:

- Workspace area (Scene, Palette, Asset, Inspector) moves to canvas-resident rendering via a grown `pixelforge_gui` widget catalog (R13/R14).
- Hybrid in M3 (canvas widgets + native menu/status/modals), full canvas chrome by M3.1.
- Logical 1280×800 editor canvas, with game canvas (e.g. 320×180) layered underneath in the Scene viewport.

M5 (visual scripting) and M6 (Paula audio) compose new widgets (`StepCard`, `RuleRow`, `NodeGraphView`, `CodeBlock`, `CellGrid`, `MixerLane`) **from the M3 pgui foundation**. M5 *explicitly rejects* the node-graph metaphor for authoring — that means `ImNodes` is not the right tool for M5 either; row-based composition is.

The ideation doc from May 15 articulates the "editor IS a Pixelforge cart" dogfooding bet but does not lock it: *"proves the engine can build apps"* is framed as a credibility story, not an architectural requirement.

### cimgui-go local snapshot

At `/home/red/Desktop/render/cimgui-go-main/`:

- Wraps **Dear ImGui 1.92.8 WIP docking branch** — dockable panels are native, not a fork.
- **First-party Ebiten backend** at `backend/ebiten-backend/` (package `ebitenbackend`). This is the canonical backend and supersedes the older `gabstv/ebiten-imgui` I mentioned earlier.
- `examples/ebiten-game/main.go` shows the integration pattern: keep your `ebiten.Game`, call `cimgui.BeginFrame()` / `EndFrame()` in `Update()`, call `cimgui.Draw(screen)` in `Draw()`, delegate `Layout()`. **No replacement of the Ebitengine driver loop required.**
- `examples/ebiten-game-in-texture/main.go` shows `CreateTextureFromGame(game, w, h)` — embeds an entire `ebiten.Game` as a texture renderable inside an `imgui.Image(...)` panel. This is the integration primitive for "the game preview is a dockable editor panel."
- Pre-compiled `.a` libs in `lib/` for linux-x64, macos-arm64/x64, windows-x64. `go build` works out of the box; no cmake.
- Local glibc check: `2.41` (Debian 13). Well above the Ubuntu CI baseline cimgui-go is built against — pre-compiled libs should link cleanly. If they don't, `make` from cimgui-go's top level rebuilds against local glibc (requires C++ compiler + luajit).
- **Bonus bindings shipped:** ImNodes, ImGuizmo, ImPlot, ImGuiColorTextEdit, ImMarkdown — available for any future use (debug visualizations, plot overlays, syntax-highlighted code panels).

### Topic axes

For this migration, the surface decomposes into:

1. **Backend wiring** — how `cimgui-go/backend/ebiten-backend` plugs into `Editor`'s existing `ebiten.Game` (Update/Draw/Layout).
2. **Chrome replacement scope** — what gets ripped out (native chrome only? + cart? + workspaces? + widget bank?).
3. **Inspector dispatch** — wiring the `pfcomponent` reflection registry into ImGui calls.
4. **Engine canvas placement** — where the 320×180 game preview and the 1280×800 editor canvas live in an ImGui world.
5. **Migration sequencing** — order of operations and what defines "done".

---

## Candidate strategies (raw)

Generated against the five axes above, biased by frames: pain/friction (native chrome is the source of "results are bad"); leverage (one move that unlocks M3/M4/M5/M6 chrome); inversion (what assumption can we drop); cross-domain (what other editors do); constraint-flipping (heavy changes allowed).

1. **Side-by-side flag** — Add ImGui as a `--imgui` parallel chrome. Iterate until parity, then delete native.
2. **Replace native chrome only, keep cart for content** — Cart canvas survives as the game preview surface; ImGui owns menus/panels/inspector/asset browser.
3. **Full nuke** — Delete `cart.go`, `cart_loader.go`, `chrome.go`, `chrome_visibility.go`, latent `canvasMenuBar`/`canvasStatusBar`, and the `pixelforge_studio/editor/widgets/` directory. ImGui becomes the only studio chrome. Keep `pixelforge_gui` untouched for engine-side use (scope, examples, in-game UI).
4. **Editor as separate binary** — Restate the dependency direction: studio depends on engine, engine never depends on studio. (Already true at code level — the change is that studio swaps its in-tree GUI from pigui to ImGui without affecting engine consumers.)
5. **Hybrid: ImGui for chrome, keep pigui in-game** — Acknowledge `pixelforge_examples/gui/` and `pixelforge_scope/` as legitimate pigui users; don't break them.
6. **`EditorUI` interface abstraction** — Define an interface, implement two backends (legacy pgui, new ImGui), cut over under the interface.
7. **Docking-first workspaces** — Replace tab-strip workspaces with ImGui DockSpace. "Workspaces" become saved dock layouts.
8. **Game preview as ImageButton with input routing** — Engine renders to texture via `CreateTextureFromGame`, lives inside a dockable ImGui panel, gets input only when focused.
9. **ImNodes for runtime debugging** — Use `imnodes` bindings to visualize the `pixelforge_event` pub/sub graph or `pixelforge_routine` coroutine flow live (not for authoring — M5 rejected nodes for authoring).
10. **`editor.pforge` fixture survives, ImGui draws it** — Keep the `.pforge`-as-editor-layout file as the persistence/theme source; ImGui renders the layout. Dogfooding spirit preserved at low cost.
11. **Port `pfcomponent` widget kinds to an ImGui dispatch table** — `WidgetKind` enum maps directly to `ImGui::DragFloat`, `ImGui::ColorEdit4`, `ImGui::Combo`, `ImGui::InputText`. Inspector becomes ~50 LOC.
12. **Use `imgui.ini` for layout persistence** — ImGui already serializes window positions / dock layouts to an `.ini` file. Free state restoration; replaces hand-rolled settings serialization for chrome geometry.

### Cross-cutting synthesis

Combinations that are stronger than their parts:

- **(3) + (7) + (8) + (11) + (12)** form one coherent destination: studio chrome is entirely ImGui-docked, inspector is reflection-dispatched into ImGui calls, game preview is a docked panel via `CreateTextureFromGame`, layout state lives in `imgui.ini`. This is the "full kill" destination expanded with concrete tactics for each axis.
- **(5) is an invariant**, not a strategy — any plan must honor it because `pixelforge_examples/gui/` and `pixelforge_scope/` already consume pigui.
- **(10) is a cheap reframing** that defangs the dogfooding objection: the *schema* (`.pforge`) survives even if the *renderer* changes. Drop into any strategy.

---

## Critique and survivors

Adversarial pass against the "hassle-free" constraint.

### Rejected

- **(1) Side-by-side flag.** Doubles the maintenance load during the transition. Every chrome decision has to be made twice. The team has to choose between two UIs for every demo, every screenshot, every bug repro. "Easy to abandon" sounds appealing but in practice it postpones the cutover indefinitely and the legacy path never gets killed. *Reject — adds work, delays decision.*

- **(6) `EditorUI` interface abstraction.** Premature abstraction. Once the cutover lands, the second backend is dead and the interface is overhead. The interface is also hard to design well because pigui's retained-callback model and ImGui's immediate-mode model don't share a meaningful shape — the lowest common denominator would be so thin it offers no leverage. *Reject — YAGNI; the abstraction dies the day after cutover.*

- **(9) ImNodes for runtime debugging.** Strong idea on its own merits but not a migration strategy. Belongs in a post-migration backlog. *Defer — surfaced for the "after migration" pile, not the migration.*

### Survivors (in dependency order)

#### S1. Full kill of studio's pigui chrome — ImGui as the only studio renderer

**Basis:** `direct:` `editor.go` already carries two parallel chrome paths (native `menuBar` + latent `canvasMenuBar`), explicitly mid-migration. Killing both paths in favor of a single new one is *less* code than completing the M3.1 hybrid commitment. The cart layer (`cart.go`, `cart_loader.go`, `chromeVisibility`) was built to support a migration that hasn't happened; it can be retired without leaving a feature gap because no `CanvasWorkspace` is wired today.

**Why it matters:** The user has explicitly authorized heavy changes. The hybrid in-flight state (M0/M1 shipped + M3 not started + canvasMenuBar latent) is the *cheapest moment in the project's lifetime* to swap chrome libraries — every additional milestone that lands on pgui makes the swap more expensive.

**What survives:** `pixelforge_gui` itself stays in the tree, untouched, for `pixelforge_examples/gui/` and `pixelforge_scope/`. Engine consumers and in-game UI keep their library. Only the **studio's** use of pgui dies. (S2 below addresses capture/scripting workspaces that currently use pgui.)

**Meeting test:** This is the strategic call. Every other survivor depends on whether the team takes this step.

#### S2. Sequence — what to migrate first, what to kill last

**Basis:** `reasoned:` Among the studio surfaces, native chrome (menu / title / status / panels) is what the user has described as visually bad. The capture and scripting workspaces are pgui-resident but later milestones (M4, M5) — they haven't shipped. The right order falls out of risk and reward:

1. **Wire `cimgui-go/backend/ebiten-backend` into `Editor.Update`/`Draw`/`Layout`.** ~1 day. Editor still draws everything it does today; ImGui frame is a no-op overlay. Validates build on Parrot/Debian 13 (glibc 2.41) before any rewrite.
2. **Replace `chrome.go` (title/menu/status/panels) with ImGui equivalents.** ~3-5 days. Visible win immediately. Inspector and asset browser still draw natively into the panel rects.
3. **Port the inspector via `pfcomponent` dispatch** (see S5). ~2-3 days. Becomes the smallest, cleanest part of the editor.
4. **Set up ImGui DockSpace** (see S3) and migrate workspaces from `Workspace` interface to dock-window registration. ~3 days.
5. **Embed the game preview as a docked image panel** (see S4). ~1-2 days. Replaces the canvas-as-workspace-viewport plumbing.
6. **Delete `cart.go`, `cart_loader.go`, `chrome.go`, `chrome_visibility.go`, `canvasMenuBar`/`canvasStatusBar` fields, and `pixelforge_studio/editor/widgets/`** after their last consumer is gone. Tests that test pgui-rendered surfaces get rewritten against ImGui's headless `imgui.io.WantCaptureMouse`/test-mode patterns or deleted.
7. **Capture and scripting workspaces** (currently pgui) — rewrite onto ImGui *as part of* M4/M5 work, not as a separate migration. Their plans haven't shipped, so this is "build them on ImGui from day one" rather than "migrate them."

**Why it matters:** Each step is independently revertable. Steps 1 and 2 alone deliver the user-visible improvement; everything else compounds from there.

**Meeting test:** This sequence is the answer to "hassle-free."

#### S3. ImGui DockSpace replaces the workspace tab strip

**Basis:** `external:` Every game tool that ships an editor (Unreal, Unity, Godot, RenderDoc, Tracy, ImHex) uses dockable panels because the editor's surface area outgrows any single hand-laid-out layout. The current `workspaces.go` tab strip is a hardcoded 22px-high strip with 100px-wide tabs — that scales linearly with workspaces and visibly compresses past ~6.

**Why it matters:** `Workspace` becomes "a registered set of ImGui windows" rather than "a panel that owns the canvas." Users can have Scene + Inspector + Asset Browser + Timeline + Console open simultaneously, save the layout, switch presets. This is what makes a tool feel professional vs. home-made. cimgui-go ships the docking branch by default — zero extra work to enable.

**Meeting test:** Probably the single largest UX improvement from this migration.

#### S4. Game preview is a dockable ImGui image panel

**Basis:** `direct:` `cimgui-go/examples/ebiten-game-in-texture/main.go` demonstrates exactly this pattern via `currentBackend.CreateTextureFromGame(&Game{}, w, h)` rendered through `imgui.ImageWithBgV(texture.ID, ...)`.

**Why it matters:** This dissolves the awkward "editor canvas above game canvas, both blitted to the window" composition. The game (Pixelforge `Surface` rendered through Ebitengine) becomes a texture; the texture lives inside an ImGui window that can be docked, resized, undocked into its own OS window (multi-viewport, supported by docking branch), or hidden. Input routing is solved by ImGui's `IsItemFocused()` / `IsItemHovered()` — feed engine input only when the panel is focused.

**Meeting test:** This is what makes the "always-on game underneath" objection disappear.

#### S5. Reflection inspector ports to a 5-function dispatch

**Basis:** `direct:` `inspector.go` already dispatches on `pfcomponent.Get(comp.Type)` metadata for each field. The widget cache (`map[inspectorKey]widgets.Widget`) exists because pgui widgets are stateful. ImGui widgets are stateless — `imgui.DragFloat("Speed", &speed, 0.1, 0, 100)` reads/writes the value directly each frame. **The cache goes away.** The dispatch table shrinks to roughly:

```
WidgetKind        → ImGui call
Slider            → imgui.SliderFloat / SliderInt (with min/max from pf tag)
Numeric           → imgui.InputFloat / InputInt
Text              → imgui.InputText
Checkbox          → imgui.Checkbox
ColorPicker       → imgui.ColorEdit4
Vector2           → imgui.InputFloat2
Enum              → imgui.Combo (items from registry)
SpriteRef         → imgui.Combo (items from project.Sprites)
AudioRef          → imgui.Combo (items from project.Audio)
EventTopic        → imgui.Combo (items from project.EventTopics)
```

**Why it matters:** Total inspector code drops from `inspector.go` (173 LOC) + the cache infrastructure + per-widget render code to a single `~80-line` switch over `WidgetKind`. The `pfcomponent` registry itself does not change — only its consumer.

**Meeting test:** Largest reduction in code, and the easiest part to demo.

#### S6. `editor.pforge` schema survives as the editor's theme + layout fixture

**Basis:** `reasoned:` The dogfooding story (the editor is a Pixelforge cart, opens itself) was a credibility bet, not an architectural commitment. The schema (`.pforge`) is the load-bearing part — it defines the layout, theme, default workspaces, default keymap. ImGui can read this schema and apply it: colors → `imgui.PushStyleColor`, fonts → `imgui.PushFont`, panel positions → dock layout initialization.

**Why it matters:** "The editor still loads `editor.pforge` to know what it looks like" preserves the story at no real cost. It dies as "the editor is *rendered through* Pixelforge primitives," but survives as "the editor is *configured by* a Pixelforge project file." That's still a meaningful dogfooding story when you say it out loud at a demo.

**Meeting test:** Cheap insurance against the "you abandoned dogfooding" objection.

#### S7. Use `imgui.ini` for window/dock state persistence

**Basis:** `direct:` ImGui serializes window positions, sizes, dock layouts, and tab states to an `.ini` file automatically. The studio currently rolls its own settings persistence for `WindowWidth`/`WindowHeight`/recent projects via `settings.go`.

**Why it matters:** Splits persistence cleanly: `imgui.ini` owns chrome geometry, `settings.go` owns project-level state (recent files, theme choice). Both `imgui.ini` and `editor.pforge` can ship with the studio binary as defaults; user overrides live in the user config dir.

**Meeting test:** Trivial to adopt; deletes more code than it adds.

---

## Recommended path forward (synthesis)

S1 is the strategic decision. S2 is its sequencing. S3–S7 are the tactical decisions inside that sequencing. **None of them require touching `pixelforge_gui` or the engine itself.** The "heavy changes" the user authorized are concentrated in `pixelforge_studio/editor/` — that is the right place because that's where the result the user described as "bad" lives.

### Distinction from prior plans

The existing M3 plan commits to growing `pixelforge_gui` into a full editor widget catalog. This ideation argues the opposite move: **don't grow pgui for the editor; freeze pgui at its current shape (engine-side use only) and rebuild the editor on ImGui.** That's the heavy change. The May 15 003 plan effectively gets retired and replaced.

The May 16 plans (M5 visual scripting, M6 audio) survive in intent — `StepCard`, `RuleRow`, `CellGrid`, `MixerLane` are *behaviors and layouts*, not *widget implementations*. They get rebuilt on ImGui primitives (`imgui.BeginChild`, `imgui.Selectable`, `imgui.BeginDragDropSource`) instead of pgui ones. Their plans need a small refresh, not a rewrite.

### What this ideation does not answer

These belong in `ce-brainstorm` next:

- Exact dock layout for the default workspace (which panels, what default sizes).
- Theme mapping — how `editor.pforge` palette slots translate to `ImGuiStyleColor_*` slots.
- Font strategy — keep `pixelforge_cofont` for in-game text, use a system TTF for editor chrome, or atlas a 16px TTF inside ImGui and route in-game text differently.
- Test strategy — what survives of the 24 editor `_test.go` files; ImGui's `TestEngine` integration if needed.
- Build-system gate — confirm pre-compiled cimgui-go `.a` links on glibc 2.41 (likely fine; verify on first wiring).
- Whether the cart canvas concept disappears entirely or survives as an internal Pixelforge `Surface` that the game preview panel consumes.

### Ride-alongs (post-migration)

- ImNodes for live `pixelforge_event` pub/sub graph visualization (debug-only, no authoring).
- ImPlot for `pixelforge_metrics` FPS/budget overlay rendered into a dockable graph window instead of native overlay.
- ImGuiColorTextEdit for the `CodeBlock` widget (M5 code preview) with real syntax highlighting.

---

## Suggested next step

Run `ce-brainstorm` on **S1 + S2** together. The brainstorm should produce a tight requirements document covering: the new file layout in `pixelforge_studio/editor/`, the deletion list, the inspector dispatch table, the dock layout defaults, the build verification gate, and the test rewrite plan. Then `ce-plan` produces the milestone breakdown that replaces `2026-05-15-003-feat-pixelforge-editor-as-cart-and-gui-growth-plan.md`.
