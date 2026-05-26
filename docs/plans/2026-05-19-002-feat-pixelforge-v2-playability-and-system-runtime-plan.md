---
date: 2026-05-19
type: feat
topic: pixelforge-v2-playability-and-system-runtime
status: active
origin: docs/ideation/2026-05-19-pixelforge-v2-gap-closers-ideation.md
predecessor: docs/plans/2026-05-19-001-feat-arcade-shipping-v2-plan.md
upstream-requirements: docs/brainstorms/2026-05-18-arcade-shipping-v2-requirements.md
depth: deep
---

# feat: PixelForge v2 — Playability Gap Closers + System-Runtime Distribution Pivot

## Summary

Make the four reference games (Asteroids, Mario, Bomberman, Donkey Kong) **actually playable end-to-end** through PixelForge Studio's GUI with zero code-writing by the user, and pivot the default shipping model from "engine bundled in every game binary" to "system-installed runtime + tiny `.pforge` cart" so shipped games behave like classic console ROMs — a small data file the installed runtime opens. Browser target becomes a standard four-file triplet (`index.html` + `wasm_exec.js` + `pixelforge-runtime.wasm` + `game.pforge`) hostable anywhere. Close the three v2-gap blockers the predecessor plan left open: body-authoritative rendering (A1), system-runtime distribution (A2), and the gameplay-verb substrate that turns Studio from substrate into a thing-that-makes-games (A3 — body-vs-body collision, projectile firing, counter-polling win/lose, `motion/move_pattern`).

---

## Problem Frame

Plan-009 (predecessor: `docs/plans/2026-05-19-001-feat-arcade-shipping-v2-plan.md`) shipped all 25 implementation units green: universal-player + cart-append envelope, four proof-cart fixtures, ImGui studio chrome, snapshot save/load on native+WASM, ingest dispatcher wiring, starter pack, File menu, WASM size reporting, verb-catalog regen, CI matrix workflow. All 61 packages pass; 5 long-tag suites pass.

But three blockers survive — together they mean a Studio user who clicks Play sees nothing recognizable as the game they authored, and a no-Go end user who receives a shipped artifact cannot install or run it as a standalone tiny binary the way classic console games shipped. The ideation (`docs/ideation/2026-05-19-pixelforge-v2-gap-closers-ideation.md`) catalogues them:

- **A1 — Rendering integration is missing the read of `rt.Bodies`.** `pixelforge_render/rendertick.go:219` calls the global `pixelforge.Draw()` which blits sprites at their authored `Scene.Entity.Position`. The comment at `rendertick.go:230` (`_ = rt`) explicitly admits this: motion sinks mutate `rt.Bodies[id].Position` deterministically every tick, but the renderer never consults that map. `updateCameraOffset` at `rendertick.go:304` correctly reads `rt.Bodies` for camera follow — proof the substrate is right there, just unconsumed by the blit path. `rt.CameraOffsetY` is computed but never applied to entity blits. Result: physics moves bodies invisibly; the screen shows static placeholders.

- **A2 — Distribution shape needs to invert.** `pixelforge_studio/playerbins/bins/` contains only `test-fake/pixelforge-player`. The `Makefile` already enumerates the 5-target `playerbins` matrix but it hasn't been run + committed. Even when it is, plan-009's shipping model bakes the ~20 MB engine into every game binary via cart-append. The user has now redirected: ship a single system-installed `pixelforge-runtime` binary per OS; games ship as `.pforge` cart files (few KB) that the runtime opens via OS file association — like SNES + ROM. Browser becomes a standard triplet (`index.html` + `wasm_exec.js` + `pixelforge-runtime.wasm` + `game.pforge`), the runtime hosted once, carts loaded by `fetch()`. The old cart-append remains available as an opt-in "Self-Contained Executable" export for users who want one-file shipping.

- **A3 — Gameplay verbs missing the primitives every arcade game needs.** Body-vs-body AABB collision is missing (ship-vs-rock, hero-vs-goomba, barrel-vs-hero); projectile firing is missing (Asteroids bullets, Mario fireballs); win/lose conditions are missing (all enemies destroyed, hero reached flag, hp ≤ 0); scene-transition recipes are missing; `motion/move_pattern` still routes to `debugDrop` at `motion_sink.go:68`. The four proof `.pforge` fixtures exist and load, but the verb sheets cannot express "kill enemies, win level" — so they don't.

The cost shape, restated: Studio passes its CI gates, but a user authoring a game through the GUI cannot make a thing the user can win, cannot kill enemies in it, and cannot ship it as anything smaller than a 20 MB executable. The promise plan-009 made ("Studio is a tool, not a tutorial") is one rendering chokepoint, one collision sink, one win-condition recipe, and one distribution-model flip away from being true.

---

## Origin Document

This plan is sourced from `docs/ideation/2026-05-19-pixelforge-v2-gap-closers-ideation.md` (the six surviving ideas, ranked by confidence) plus the user's seven explicit architectural decisions captured at `/ce-work` invocation and confirmed via `AskUserQuestion` (system-runtime distribution model; standard WASM triplet; plan-first execution).

Upstream context comes from `docs/brainstorms/2026-05-18-arcade-shipping-v2-requirements.md` (R1–R22, A1–A3, F1–F3, AE1–AE10), which fed predecessor plan-009. This plan **supersedes** the distribution-model portions of plan-009 — R2 and R3 are reinterpreted (see Requirements Traceability below). All other plan-009 invariants are preserved: cycle-break injection (interface in inner package, impl in outer); schema additivity (`omitempty` + sanitize-on-load); `CGO_ENABLED=0` for WASM; ImGui-only studio chrome; determinism (U6 cross-CPU pixel-hash CI gate must stay green); no string-concat source generation.

**Hard constraint preserved verbatim from durable memory:** *do not use git at all during this work.* This plan does not enumerate commit boundaries.

---

## System-Wide Impact

| Surface | Touched by |
|---|---|
| `pixelforge_render/rendertick.go` (A1 chokepoint) | U1 — `pixelforge.Draw()` call replaced by `DrawEntities` chokepoint |
| `pixelforge_render/` (new file) | U1 — new `drawables.go` for per-entity blit |
| `pixelforge_studio/capsuleruntime/motion_sink.go` (A3) | U2 — `motion/move_pattern` concrete handler |
| `pixelforge_studio/capsuleruntime/` (new collision sink, new win/lose sink, new projectile spawn) | U3, U6, U5 |
| `pixelforge_studio/scripting/catalog/` (A3 — new recipes + triggers) | U4, U5, U6 — `RecipeOnCollide`, `RecipeWinWhen`, `RecipeLoseWhen`, `RecipeFireProjectile`, `TriggerWhenBodyCollide` |
| `pixelforge_pool/` (extend existing pkg) | U5 — generic `Pool[T]` extending today's `pipool.go` |
| `pixelforge_project/` (schema additive: `Scene.PoolBudgets`, `Project.PhysicsPreset` extension if needed) | U5, U6 |
| `cmd/pixelforge-runtime/` (NEW — system-installed runtime) | U7 |
| `pixelforge_studio/installer/` (NEW — file-association manifests) | U8 |
| `.goreleaser.yaml` (NEW — root-level config) | U8 |
| `pixelforge_studio/build/workspace.go` (UX — new menu labels) | U9, U10 |
| `pixelforge_studio/buildpipeline/builders_long.go` (new cart-only + web-triplet builders alongside existing hostLong/wasmLong) | U9, U10 |
| `pixelforge_studio/exportweb/` (NEW pkg — triplet emission) | U10 |
| `pixelforge_studio/buildpipeline/wasm_template.html` (new triplet template replaces single-file inline) | U10 |
| `pixelforge_studio/editor/` (Inspector verb-sheet UI for new recipes) | U14 |
| `pixelforge_studio/integration_test/fixtures/asteroids_proof.pforge` + 3 sibling carts (verb sheets extended with new recipes) | U12 |
| `pixelforge_studio/integration_test/*_proof_test.go` (baselines refreshed) | U13 |
| `Makefile` (rename `playerbins` → `runtimebins` + add `goreleaser` target) | U8, U15 |
| `pixelforge_studio/playerbins/bins/<os>-<arch>/` (NEW — populate via runtimebins) | U15 |
| `.github/workflows/long.yml` (extend matrix for runtime smoke + collision/projectile/win-lose tests) | U16 |

**Stakeholders:** A1 (end-user game authors) gain a Studio that produces playable games entirely via GUI; A2 (game consumers) install `pixelforge-runtime` once and double-click `.pforge` files to play; A3 (PixelForge engineers) gain CI gates for collision/projectile/win-lose semantics + per-OS runtime smoke. All plan-008 + plan-009 invariants preserved.

---

## Output Structure

New directories and files this plan introduces or substantially restructures (per-unit `Files:` blocks remain authoritative):

```
pixelforge-go/
├── cmd/
│   ├── pixelforge-player/         # PRESERVED (cart-append mode, opt-in)
│   └── pixelforge-runtime/        # NEW (U7) — system-installed runtime
│       ├── main.go                # cart-by-path / file-assoc / stdin
│       └── main_test.go
├── pixelforge_render/
│   ├── rendertick.go              # MODIFIED (U1) — Draw() call → DrawEntities()
│   └── drawables.go               # NEW (U1) — per-entity blit chokepoint
├── pixelforge_pool/
│   ├── pipool.go                  # PRESERVED
│   └── entity_pool.go             # NEW (U5) — generic Pool[T] + ForEach
├── pixelforge_studio/
│   ├── capsuleruntime/
│   │   ├── motion_sink.go         # MODIFIED (U2) — move_pattern concrete
│   │   ├── collision_sink.go      # NEW (U3) — body-vs-body topic publisher
│   │   ├── projectile_sink.go     # NEW (U5) — combat/fire_projectile
│   │   ├── win_lose_sink.go       # NEW (U6) — predicate evaluator
│   │   └── (existing sinks unchanged structurally)
│   ├── scripting/catalog/
│   │   ├── verb_recipes.go        # MODIFIED — new trigger + recipe constants
│   │   └── builtin_arcade.go      # MODIFIED — new arcade recipes
│   ├── exportweb/                 # NEW (U10) — standard triplet emission
│   │   ├── doc.go
│   │   ├── triplet.go             # emit index.html + wasm_exec.js + .wasm + .pforge
│   │   ├── triplet_test.go
│   │   └── index_template.html    # new template (fetch-based loader)
│   ├── buildpipeline/
│   │   ├── builders_long.go       # MODIFIED (U9, U10) — cartOnlyBuilder + webTripletBuilder added; hostLong/wasmLong preserved as opt-in
│   │   └── wasm_template.html     # PRESERVED for opt-in inline-cart path
│   ├── build/
│   │   └── workspace.go           # MODIFIED (U9, U10) — new menu labels
│   ├── editor/
│   │   └── inspector_collision.go # NEW (U14) — collision-pair + win/lose + pool-budget UI
│   ├── installer/                 # NEW (U8)
│   │   ├── linux/                 # .desktop + MIME registration
│   │   ├── darwin/                # LaunchServices Info.plist fragment
│   │   └── windows/               # WiX file-association fragment
│   └── integration_test/fixtures/
│       ├── asteroids_proof.pforge # MODIFIED (U12) — new verb-sheet entries
│       ├── mario_proof.pforge     # MODIFIED (U12)
│       ├── bomberman_proof.pforge # MODIFIED (U12)
│       └── donkey_kong_proof.pforge # MODIFIED (U12)
├── .goreleaser.yaml               # NEW (U8) — root-level goreleaser config
├── Makefile                       # MODIFIED (U8, U15) — runtimebins + goreleaser targets
└── .github/workflows/
    └── long.yml                   # MODIFIED (U16) — runtime smoke + collision matrix
```

The tree is a scope declaration. Implementers may adjust layout where implementation reveals a better one; per-unit `Files:` blocks remain authoritative.

---

## High-Level Technical Design

*This section illustrates the intended approach and is directional guidance for review, not implementation specification. The implementing agent should treat it as context, not code to reproduce.*

### Distribution shape: from "engine in every binary" to "system runtime + cart"

```
                        ┌──────────────────────────────────────┐
                        │      single Go codebase              │
                        │  pixelforge_* engine packages        │
                        │  + pixelforge_cart (preserved)       │
                        │  + pixelforge_pool (extended)        │
                        └────────────┬─────────────────────────┘
                                     │ go build with build tags
                ┌────────────────────┼────────────────────────┐
                ▼                    ▼                        ▼
       pixelforge-studio    pixelforge-runtime      pixelforge-runtime.wasm
       (editor + chrome     (system-installed       (web runtime;
        + embedded          per OS; opens .pforge   hosted once;
        player runtime)     by path / file-assoc)   loads .pforge via fetch)
                │                    │                        │
                │ Build → Cart       │ install once via       │ Studio Export to Web
                │  produces          │   .deb / .pkg / .msi   │  produces folder:
                │  game.pforge       │   (registers .pforge   │   index.html
                │  (few KB)          │    file association)   │   wasm_exec.js
                │                    │                        │   pixelforge-runtime.wasm
                ▼                    ▼                        │   game.pforge
        game.pforge          double-click .pforge             ▼
        (~5–50 KB)           → opens in installed runtime    hostable anywhere
                             → plays                          (file:// or any web server)

       OPT-IN PATH (preserved from plan-009):
       Studio → "Build → Self-Contained Executable"
                ▼
       game.exe / game.app / game.bin  (~20 MB; player + appended cart;
                                        no install needed on consumer machine)
```

### A1: Drawable Adapter — body-authoritative blit chokepoint

```
                           pixelforge_render.advanceAndRender
                                              │
                                              ▼
                              pixelforge_render.DrawEntities(screen, rt)
                                              │
                                              ▼
                              for each e in rt.CurrentScene.Entities:
                                              │
                       ┌──────────────────────┼──────────────────────┐
                       ▼                      ▼                      ▼
              body := rt.Bodies[e.ID]   effect := rt.VisualEffects   pos := body.Position
              (nil for non-physics       [e.ID]                       OR e.Position (fallback)
               entities — OK)                                         pos.Y -= rt.CameraOffsetY
                       │                      │                      │
                       └──────────────────────┼──────────────────────┘
                                              ▼
                              drawOne(screen, e, effect, pos)
                              (honors Hidden, FlashColor, SpriteSwap)
```

`pixelforge.Draw()` is no longer called from the render hot path. Existing global is left in place for other consumers (legacy demos under `snake/`, `pacman/`, `piano/`, etc.) but the capsule runtime path goes through the adapter.

### A3: Collision-as-topic + EntityPool + Win/Lose — the three interlock

```
                        pixelforge_loop.EventLateUpdate (each tick)
                                              │
                  ┌───────────────────────────┼──────────────────────────────┐
                  ▼                           ▼                              ▼
          collisionSink                projectileSink (per-pool)       winLoseSink
          (resolv.Space.CollisionTest)  (Acquire/Release entities)     (predicate eval)
                  │                           │                              │
                  ▼                           ▼                              ▼
         publishes per pair:          on combat/fire_projectile:    if predicate satisfied:
         *VerbEvent{                  acquire body from pool;        publish scene/transition
           Topic:"collision/           wire into rt.Bodies +         (win → winScene;
            body_collide",             rt.CurrentScene.Entities;     lose → loseScene)
           Args:{a_id,b_id,            collision sink picks it up
            a_archetype,                next tick via normal pair
            b_archetype}}              enumeration
                  │
                  ▼
         catalog recipe match (RecipeOnCollide):
           if archetype-pair filter matches → fire bound actions
           (damage/take_damage, spawn/destroy_other, scene/transition, etc.)
```

The cascade chain — `fire_projectile` → `body_collide` → `take_damage` → `die` → `spawn/destroy_other` → counter decrements → `win_when` predicate fires → `scene/transition` — runs through existing sinks (damage, spawn, scene) with no new infrastructure beyond the three new sinks + four new catalog recipes. This is the key compounding observation from the ideation: 5 of the 6 missing A3 verbs (projectile, body-vs-body, win, lose, scene-transition) collapse to three additions.

### Studio "Build" menu (post-pivot)

| Menu item | Output | Default? | Engine bundled? | Install required? |
|---|---|---|---|---|
| **Build → Cart** | `game.pforge` (few KB) | Yes (host) | No | Yes (one-time `pixelforge-runtime` install) |
| **Build → Web** | `index.html` + `wasm_exec.js` + `pixelforge-runtime.wasm` + `game.pforge` | Yes (web) | Runtime served once; carts share it | No (host the triplet anywhere) |
| **Build → Self-Contained Executable** | `game.exe` / `.app` / `.bin` (~20 MB) | No (opt-in) | Yes (appended) | No (truly standalone) |

---

## Key Technical Decisions

- **Distribution flip: system runtime + cart is the new default; cart-append survives as opt-in.** Plan-009 R2/R3 are reinterpreted (see Requirements Traceability). The user's explicit framing: "the game once shipped is small standalone binary like those retro games used to be." Truly 30 KB retro size is not reachable in Go+Ebitengine, but a 5–50 KB `.pforge` cart against an installed system runtime is the same shape as SNES+ROM and matches the user's intent exactly. Cart-append is preserved as a labeled opt-in menu item ("Build → Self-Contained Executable") so users who want one-file shipping still have it — predecessor plan-009's machinery isn't deleted, it's demoted from default to advanced.

- **`pixelforge-runtime` is a distinct binary from `pixelforge-player`.** Different names because the binaries have different roles: `pixelforge-runtime` reads cart by path argument or via OS file association (system-installed, user-visible); `pixelforge-player` reads cart from appended footer in its own executable (one-shot, opaque to user). Same Ebitengine + capsuleruntime pipeline; both files in `cmd/`. Sharing the binary name "pixelforge-player" for both modes would confuse OS file-association registration (the installer registers `.pforge → /usr/bin/pixelforge-runtime`, not the bundled-cart binary which doesn't exist on the user's system). *Why:* clarity of install footprint outweighs minor code duplication; the two `main.go` files share ~30 lines of cart-loading logic factored into `pixelforge_cart.LoadFromPath`.

- **GoReleaser is the canonical Go-ecosystem tool for cross-OS releases.** MIT, ~14k stars, mature. One `.goreleaser.yaml` cross-compiles `pixelforge-runtime` for the 4 native targets (linux-amd64, darwin-amd64, darwin-arm64, windows-amd64; WASM goes through `make runtimebins` separately because installer packaging doesn't apply), wraps each in the appropriate package format with file-association manifests, attaches all to a GitHub Release. Per-OS code-signing (when v3 enables it) drops in as a GoReleaser hook. *Why:* ideation idea #3 explicit; ideation already enumerated alternatives (manual matrix, GH Pages index, OCI registry) and GoReleaser won.

- **Adopt the ideation's three-interlock for A3: collision + pool + counter-polling.** Ideas #4, #5, #6 from the ideation. Confidence 95% / 80% / 85% respectively. Cross-frame convergence in the ideation was strongest on collision-as-topic (4 of 6 frames). *Why:* compresses 5 of 6 missing A3 verbs into 3 small additions; ideation's analysis is thorough and the substrate (`rt.Globals`, `rt.Physics`, `verbs.bus`, `damage_sink`, `spawn_sink`) is already in place to receive these events.

- **`motion/move_pattern` ships as a concrete handler in this plan.** Ideation row #26 (the "rejected as too-small" follow-up to ideas #4–#6) is promoted to U2. Without it, sine/patrol/waypoint-driven entities (a basic Mario Goomba patrol, a Bomberman pickup hover, an Asteroids medium-asteroid orbit) stay log-only — the four proof carts cannot demonstrate full gameplay. *Why:* it's a 30-line addition that completes the motion-sink story; deferring it would leave one verb log-only after the rest of A3 lands.

- **Standard WASM triplet over inline single-HTML.** User explicit. Inline single-HTML works but rebuilds the entire bundle every cart change (cart bytes get base64-encoded inside the HTML alongside the wasm). Triplet shares the wasm + wasm_exec.js across carts (a user hosting 5 games at `/games/{a,b,c,d,e}/` ships one ~12 MB wasm + 5 tiny carts, not 5 × 20 MB inlined HTMLs). *Why:* matches the runtime+cart shape on web; preserves disk + bandwidth across multiple shipped games; standard Go-WASM layout makes the triplet familiar to anyone who has shipped a Go WASM app before.

- **The existing `wasm_template.html` (inline single-file) is preserved for opt-in self-contained export.** When the user picks "Build → Self-Contained Executable" with target=WASM, they get one .html (predecessor plan-009 U3 behavior). When they pick "Build → Web" (the new default), they get the triplet. *Why:* preserves the existing path; lets the user choose disk-shape per ship.

- **EntityPool is a generic `Pool[T any]`, not projectile-specific.** Ideation idea #5. Once one pool exists, the next five entity types (particles, drops, shells, popups, spawned mooks) are five-line additions. Fixed-size pre-allocated arrays indexed by `alive bool`; no per-spawn allocation; deterministic slot selection (next-free-index) preserves snapshot determinism. Per-archetype budget declared via new additive schema field `Scene.PoolBudgets map[string]int`. *Why:* compounding payoff is the whole reason it's a separate package extension (`pixelforge_pool/entity_pool.go`) rather than a projectile-local helper.

- **Win/lose precedence: lose > win on same-tick satisfaction.** When both `RecipeWinWhen` and `RecipeLoseWhen` predicates first satisfy on the same tick, lose takes precedence. *Why:* matches arcade tradition (a player who depletes HP mid-victory dies — the death is the final state, not the kill that triggered it). Documented in `win_lose_sink.go` and tested explicitly. User decision at U6 design clarification.

- **Collision broad-phase relies on resolv's existing cell-grid Space.** `solarlune/resolv` (already a dep via plan-009 U6) ships `Space.CollisionTest` and `Shape.IntersectionTest` with cell-grid broad-phase. For arcade-scale games (50–200 entities), this is sufficient. Tag-based pair filtering at recipe-match time culls before per-pair archetype lookup. *Why:* avoids reimplementing broad-phase; resolv's adoption is settled; mitigates the ideation's downside note about O(n²) naive enumeration.

- **Continuous-collision sweep deliberately out of scope.** Resolv is positional, not swept. Fast projectiles (>cell-size velocity per tick) can tunnel through targets. Mitigation: documented max-velocity-per-frame constraint in `pixelforge_physics/doc.go` and `combat/fire_projectile` recipe clamps muzzle velocity. *Why:* swept collision is a deep rabbit-hole; the ideation flagged this downside; the mitigation is one constraint that fits arcade scale (Asteroids bullets travel ~5 px/tick, well under typical 16-px cell size).

- **File association on host OS is installer-time, not runtime-time.** Each installer (`.deb` / `.pkg` / `.msi`) registers the `.pforge` MIME type and command association during install. The runtime binary itself does NOT mutate user file associations at first launch — that surprises users and gets flagged by AV/Gatekeeper. *Why:* standard install hygiene; avoids the runtime needing root/admin for first launch.

- **No git operations during execution.** Carried verbatim from user durable memory and predecessor plan. This plan does not enumerate commit boundaries; progress in `ce-work` is tracked by the task list, not by commits.

---

## Requirements Traceability

This plan **supersedes** the distribution-model portions of plan-009. The brainstorm's R-IDs map to new units as follows; reinterpreted R-IDs are flagged explicitly.

| R-ID | Origin Requirement | Status | Implementation Units |
|---|---|---|---|
| R1 | Two binaries from one codebase (studio + player) | **Extended** — now three binaries: studio, player (legacy cart-append), runtime (NEW system-installed) | U7 |
| R2 | Build → Host = self-contained binary with appended cart | **Reinterpreted** — DEFAULT is now Build → Cart (`.pforge` file for system runtime). Cart-append survives as opt-in "Build → Self-Contained Executable." | U9 |
| R3 | Build → WASM = one self-contained .html | **Reinterpreted** — DEFAULT is now Build → Web (triplet: index.html + wasm_exec.js + runtime.wasm + cart). Inline single-HTML survives as opt-in. | U10 |
| R4 | Cross-OS native build rejected | **Preserved** — applies to opt-in cart-append path | (no new units; preserved from plan-009 U3) |
| R5 | Concrete sinks for all 11 verb categories | **Extended** — adds collision/body_collide, combat/fire_projectile, meta/win_when, meta/lose_when, motion/move_pattern | U2, U3, U5, U6 |
| R6 | One configurable physics core | Preserved | (no new units) |
| R7 | Snapshot save/load round-trip | Preserved | (no new units; new pool state must serialize — covered in U5 test scenarios) |
| R8 | Four `.pforge` fixtures + traces | **Extended** — fixtures updated with collision/win/lose verb sheets; new baselines | U12, U13 |
| R9 | CI per-commit fixture replay | **Extended** — adds collision/projectile/win-lose assertions to baselines | U13, U16 |
| R10 | File → New genre starter templates | Preserved | (no new units; existing templates from plan-009 U22 unchanged) |
| R11 | File → Open Example | Preserved | (no new units) |
| R12 | Single shared RenderTickAt function | **Strengthened** — the chokepoint A1 closes IS the proof of R12; preview + runtime + replay all share the body-authoritative draw | U1 |
| R13 | Capability matrix regenerated from CI | Preserved | (no new units; matrix gains collision/win-lose rows automatically via U16) |
| R14, R15, R16 | Asset library + starter pack + examples | Preserved | (no new units) |
| R17 | Ingest dispatcher wiring | Preserved | (no new units) |
| R18, R19 | Verb catalog as source of truth | **Extended** — new recipes auto-update catalog regen (plan-009 U24) | U4, U5, U6 |
| R20 | Press-key-to-bind | Preserved | (no new units) |
| R21, R22 | WASM size reporting + gzip + wasm-opt | **Reinterpreted** — applies to runtime.wasm only (the cart is data, no size threshold) | U10 |
| **R23 (NEW)** | **System-installed `pixelforge-runtime` binary per OS with `.pforge` file association** | New | U7, U8 |
| **R24 (NEW)** | **Studio Build → Cart produces `.pforge` file as the default host artifact** | New | U9 |
| **R25 (NEW)** | **Studio Build → Web produces the standard four-file triplet** | New | U10 |
| **R26 (NEW)** | **Studio Inspector exposes collision-pair, projectile, and win/lose authoring via GUI dropdowns; zero code-writing required to make any of the four proof games end-to-end playable** | New | U14 |
| **R27 (NEW)** | **A1 closure — `pixelforge_render` reads `rt.Bodies` and applies `rt.CameraOffsetY` at per-entity blit time** | New | U1 |

Acceptance examples AE1–AE10 (from brainstorm) are reinterpreted: AE1 (Build → Host on macOS) now means "Build → Cart produces .pforge; double-clicking on a macOS machine with `pixelforge-runtime` installed launches the game." AE2 (Build → WASM) means "Build → Web produces the triplet; opening `index.html` shows splash → click-to-start → game plays." The behavior outcome stays identical; the artifact shape changes. AE3–AE10 unchanged.

---

## Implementation Units

Five phases. Phase A closes A1 with one chokepoint. Phase B closes A3 with the three-interlock + `move_pattern`. Phase C closes A2 with the runtime binary + installer flow. Phase D wires the four proof games through Studio's Play loop end-to-end. Phase E ships the release-process polish. Each phase is independently demoable; phase boundaries are dependency edges, not artificial milestones.

---

### Phase A — A1: Body-authoritative rendering

### U0. Sprite-source pipeline wired in capsuleruntime (prerequisite for visible pixels)

**Goal:** Wire actual sprite-source resolution from `Project.Sprites[*].RelativePath` (PNG bytes in the cart's `assets` FS) into the capsuleruntime, exposed as a per-name `pixelforge.Canvas` lookup on `rt`. Today `pixelforge_entity.RenderAll` calls `DrawSpriteFn(s, dx, dy)` with a zero-`Source` Sprite (see `pixelforge_entity/render.go:141-146`), tolerated only by the test recorder; production rendering produces invisible pixels. U1's verification ("loading a proof cart in Studio + pressing Play shows the player sprite moving") requires this pipeline.

**Requirements:** R27 (precondition), R12 (precondition for the parity claim — without sprite data, both preview and player render blank pixels identically, which is parity in a useless sense).

**Dependencies:** none — depends only on existing `capsuleruntime.Boot(p, assets fs.FS, opts)` accepting an asset FS.

**Files:**
- `pixelforge_studio/capsuleruntime/sprite_cache.go` (new) — `type SpriteCache struct{ canvases map[string]*pixelforge.Canvas }`; `Resolve(name string) (*pixelforge.Canvas, bool)`; `LoadAll(p *pixelforge_project.Project, assets fs.FS) error`
- `pixelforge_studio/capsuleruntime/sprite_cache_test.go` (new)
- `pixelforge_studio/capsuleruntime/runtime.go` (modify) — `Runtime` gains `Sprites *SpriteCache` field; `Boot` calls `LoadAll` during init
- `pixelforge_render/sprite_resolver.go` (new — narrow, no capsuleruntime dep) — `type SpriteResolver func(name string) (*pixelforge.Canvas, bool)`; helper that closes over `rt.Sprites.Resolve` for the Drawable Adapter (U1) to consume

**Approach:**
- `LoadAll` walks `Project.Sprites`. For each `SpriteAsset`, opens `assets.Open(RelativePath)`, PNG-decodes into a `pixelforge.Canvas` via the existing palette-mapping path (`pixelforge.NewCanvas` + per-pixel index assignment using the project's palette).
- Missing assets (file not in FS): log a warning + skip; resolution returns `(nil, false)` for that name. The renderer treats this as "draw nothing for this entity" (graceful — matches existing `RenderAll` behavior for unknown sprite names).
- Decoded canvases are cached; subsequent `Resolve` calls are O(1) map lookups. Cache lives for the runtime's lifetime (no eviction at v2 scale).
- WASM compatibility: PNG decode via stdlib `image/png` is pure-Go + WASM-safe.
- Determinism: PNG decode is bit-identical; palette mapping is deterministic; cache is keyed by sprite name (stable). Same project + same asset FS → same canvases.

**Patterns to follow:**
- Existing `pixelforge.Canvas` + `palette.go` for palette-aware canvas construction.
- Existing `pixelforge_studio/codegen/` patterns for asset-FS access.
- `pixelforge_entity/render.go:137-162` `buildSprite` — extend to accept a resolver so it can fill in `Source`.

**Test scenarios:**
- *PNG decode round-trip:* Load a 16×16 test PNG; assert canvas dimensions + at least one non-transparent pixel matches expected color index.
- *Missing asset graceful:* Project references "ghost.png" not in asset FS; assert `LoadAll` returns nil error + `Resolve("ghost")` returns `(nil, false)`.
- *Cache hit O(1):* Resolve called twice; both return same `*pixelforge.Canvas` pointer (cached).
- *WASM compile-test:* `GOOS=js GOARCH=wasm go build ./pixelforge_studio/capsuleruntime/...` succeeds.
- *Snapshot does not serialize cache:* SpriteCache is regenerated from asset FS at Boot; never round-trips through snapshot save/load.
- *Multi-sprite project:* Project with 10 sprites; all load; all resolvable; load time logged as a baseline.

**Verification:** `go test ./pixelforge_studio/capsuleruntime/...` passes; `rt.Sprites.Resolve(name)` returns non-nil canvas for every sprite in any of the four proof carts after `Boot`.

---

### U1. Drawable Adapter — `DrawEntities` chokepoint reads `rt.Bodies` + applies `rt.CameraOffsetY`

**Goal:** Replace the global `pixelforge.Draw()` call inside `pixelforge_render/rendertick.go`'s `advanceAndRender` with `DrawEntities(screen, rt)` — a new per-entity blit chokepoint that iterates `rt.CurrentScene.Entities`, joins each to `rt.Bodies[e.ID]` and `rt.VisualEffects[e.ID]`, applies `rt.CameraOffsetY` at blit time, and falls back to authored `Scene.Entity.Position` when no body exists. Honors `VisualEffect.Hidden`, `FlashColor`, and `SpriteSwap`.

**Requirements:** R27, R12 (strengthens).

**Dependencies:** U0 (without sprite-source resolution, the chokepoint produces invisible pixels — verification can't confirm "sprite moves visibly").

**Files:**
- `pixelforge_render/rendertick.go` (modify) — replace `pixelforge.Draw()` at line 219 with `DrawEntities(screen, rt)`; remove the `_ = rt` comment at line 230
- `pixelforge_render/drawables.go` (new) — `DrawEntities(screen *ebiten.Image, rt *capsuleruntime.Runtime)`; `drawOne(screen, e, body, effect, cameraOffsetY)`
- `pixelforge_render/drawables_test.go` (new)
- `pixelforge_render/rendertick_test.go` (modify) — add scenarios where body position diverges from authored position
- `pixelforge_studio/integration_test/render_bodies_test.go` (new) — `//go:build long`; end-to-end test that motion sink mutations affect rendered pixels

**Approach:**
- `DrawEntities` iterates the current scene's entities in stable order (preserves z-ordering). For each entity:
  - `body, hasBody := rt.Bodies[e.ID]`. Position source: `hasBody ? body.Position : e.Position` (the fallback preserves entities that don't carry a body — UI elements, decorative tiles, etc.).
  - `effect := rt.VisualEffects[e.ID]`. If `effect.Hidden`, skip blit entirely. If `effect.SpriteSwap != ""`, use that sprite name instead of `e.Sprite`. If `effect.FlashColor` is non-zero AND the effect's TTL > 0, tint the blit (use Ebitengine's `DrawImageOptions.ColorScale` or a palette override).
  - Camera offset: `blitY := positionY - rt.CameraOffsetY`. Camera is vertical-only in v2 (DK's tall scenes); X stays 0 (existing behavior).
  - Blit via existing `pixelforge_ebiten.DrawSpriteAt(screen, sprite, blitX, blitY, options)` (or equivalent — the precise helper exists in `pixelforge_ebiten`; implementer picks the right one at implementation time).
- The global `pixelforge.Draw()` is NOT removed — legacy demos under `snake/`, `pacman/`, `piano/`, etc. still call it via their own entry points. The capsuleruntime render path simply stops invoking it.
- Determinism: `DrawEntities` is pure-ish in the same sense as `RenderTickAt` — no goroutines, no `time.Now()`, no unseeded `math/rand`. The U6 cross-CPU pixel-hash CI gate is the regression detector.

**Execution note:** Test-first. Write `TestDrawEntities_BodyOverridesPosition` first (motion sink mutates `rt.Bodies[hero].Position.X = 100`; rendered pixel at x=100 contains the hero sprite, not at the authored x=0) BEFORE modifying `rendertick.go`. The test will fail until the chokepoint is wired in.

**Patterns to follow:**
- `pixelforge_render/rendertick.go:304` `updateCameraOffset` is the existing pattern for "iterate entities + read `rt.Bodies`." Mirror its shape.
- `docs/solutions/always-on-game-embedding.md` ("one render path" rule).
- `docs/solutions/ring-buffer-snapshot-store.md` ("render-time overrides are transient lookups, NOT mutations of `Scene.Entity.Position`") — the adapter respects this: it reads `Bodies` but does not write back to `Scene.Entity.Position`.

**Test scenarios:**
- *Body overrides authored position:* Set `e.Position = {0, 0}` and `rt.Bodies[e.ID].Position = {100, 50}`; render; assert sprite blits at (100, 50), not (0, 0).
- *No body: fallback to authored:* Entity with `e.Position = {32, 32}` and `rt.Bodies[e.ID] == nil`; render; assert blit at (32, 32).
- *Camera offset applied:* `rt.CameraOffsetY = 64`; entity with body at y=100; assert blit at y=36 (100 - 64).
- *Hidden effect skips blit:* `rt.VisualEffects[e.ID].Hidden = true`; render; assert sprite NOT in framebuffer at that position (transparent or background pixel).
- *Flash color tints blit:* `effect.FlashColor = red`, `effect.FlashTTL = 10`; render; assert blit color has red dominant channel.
- *Sprite swap uses override:* `effect.SpriteSwap = "hero_jump"`; render; assert hero_jump pixels blit, not the default hero pixels.
- *Z-order preserved:* Two entities at same position; assert entity earlier in `Scene.Entities` slice draws underneath the later one (or the documented order — pick one and assert).
- *Determinism (intra-CPU):* Same runtime + same tick → byte-equal `*image.RGBA` across 100 invocations.
- *Determinism (cross-CPU):* Long-tag test; same runtime + same tick on `ubuntu-latest` + `macos-latest`; assert byte-equal per plan-009 U6's determinism contract (or whichever fallback that probe landed on).
- *Covers R27, R12.* `Covers AE3.` (Mario jump test — hero sprite actually moves visibly when motion sink fires).

**Verification:** `go test ./pixelforge_render/...` passes; `go test -tags=long ./pixelforge_studio/integration_test -run TestRenderBodies` passes; manually loading any of the four proof carts in Studio + pressing Play shows the player sprite moving in response to input.

---

### Phase B — A3: Gameplay verb substrate

### U2. `motion/move_pattern` concrete handler

**Goal:** Replace the `debugDrop` fallback for `motion/move_pattern` in `pixelforge_studio/capsuleruntime/motion_sink.go:68` with a concrete handler covering three patterns: `sine` (Y oscillation), `patrol` (back-and-forth between bounds, reverse on wall hit), `waypoint` (linear interpolation through a list of points).

**Requirements:** R5 (extended).

**Dependencies:** none — purely local to motion_sink.

**Files:**
- `pixelforge_studio/capsuleruntime/motion_sink.go` (modify) — replace `debugDrop` case for `motion/move_pattern` with concrete dispatch; new private functions `applySinePattern`, `applyPatrolPattern`, `applyWaypointPattern`
- `pixelforge_studio/capsuleruntime/motion_sink_movepattern_test.go` (new)
- `pixelforge_studio/scripting/catalog/builtin_arcade.go` (modify, if needed) — add `pattern` arg enum: `"sine" | "patrol" | "waypoint"`

**Approach:**
- `motion/move_pattern Args{entity, pattern, ...}`: dispatch on `pattern`.
- Sine: `Args{entity, axis: "y", amplitude: 16, period_ticks: 60, phase: 0}`. `body.Position.Y = origin.Y + amplitude * SinDeg((tick*360/period + phase) mod 360)`. Uses `pixelforge_physics.SinDeg` (LUT, deterministic).
- Patrol: `Args{entity, axis: "x", min: 32, max: 96, speed: 1}`. Reverses velocity on bound hit. Maintains direction in entity's existing scratchpad (use a new `MovePattern` field on the runtime side, or store in `rt.Globals["pattern:<entityID>:dir"]` — implementer picks at U2 based on snapshot-round-trip cost).
- Waypoint: `Args{entity, waypoints: [[x,y], ...], speed: 1, loop: true}`. Linear interpolation tick-by-tick toward next waypoint; advances index on arrival within tolerance.
- All three patterns mutate `rt.Bodies[entity].Position` (NOT `Scene.Entity.Position` — per the `ring-buffer-snapshot-store.md` invariant). U1's Drawable Adapter picks them up automatically.

**Patterns to follow:**
- Existing motion sink handlers (apply_thrust, screen_wrap, ladder_climb) — same shape: lookup body, mutate Position, optionally publish notification event.
- `pixelforge_physics/trig.go` — `SinDeg` / `CosDeg` LUT-based deterministic trig.

**Test scenarios:**
- *Sine Y oscillation:* Entity at y=50, amplitude=10, period=60; advance 15 ticks; assert y ≈ 60 (quarter-period peak).
- *Sine X oscillation with axis arg:* Pattern with `axis:"x"`; advance ticks; Y unchanged, X oscillates.
- *Patrol bounds reverse:* Entity at x=32, min=32, max=64, speed=2; advance until x reaches 64; assert direction reverses; advance more; x returns toward 32.
- *Waypoint linear interp:* Waypoints `[[0,0], [10,0]]`, speed=1; advance 10 ticks; assert x progresses from 0 to 10 linearly (one pixel per tick).
- *Waypoint loop:* `loop: true`; after reaching last waypoint, returns to index 0.
- *Waypoint no-loop:* `loop: false`; after reaching last waypoint, body stops at final point.
- *Missing entity:* `Args{entity: "ghost"}`; assert `debugDrop` + no mutation (existing pattern).
- *Unknown pattern:* `Args{pattern: "spiral"}`; assert `debugDrop` + no mutation.
- *Snapshot round-trip:* Save mid-patrol (direction = -1); load; assert direction preserved (the U5 + U10 plan-009 snapshot contract).
- *Determinism:* Same args + same tick sequence on two runs → byte-equal positions.

**Verification:** `go test ./pixelforge_studio/capsuleruntime/...` passes; `motion/move_pattern` no longer appears in any `grep debugDrop pixelforge_studio/capsuleruntime/motion_sink.go` output.

---

### U3. Collision-as-topic sink

**Goal:** Add `collisionSink` to capsuleruntime that runs once per tick post-motion. Queries `rt.Physics` (already a `resolv.Space` per plan-009 U6) for overlapping body pairs via `resolv.Space.CollisionTest` or per-shape `Shape.IntersectionTest`. For each pair, looks up archetypes, publishes one `*VerbEvent{Topic: "collision/body_collide", Args:{a_id, b_id, a_archetype, b_archetype}}` on `pixelforge_loop.VerbsBus()`. New topic constant `EventTopicCollisionBodyCollide`.

**Requirements:** R5 (extended), R26.

**Dependencies:** U1 (so visible effects of collisions render correctly), but executable independently.

**Files:**
- `pixelforge_studio/capsuleruntime/collision_sink.go` (new) — `type collisionSink struct{ rt *Runtime }`; `OnLateUpdate()` runs the pairwise scan
- `pixelforge_studio/capsuleruntime/collision_sink_test.go` (new)
- `pixelforge_studio/capsuleruntime/registry.go` (modify) — register `collisionSink` to `EventLateUpdate` after motion sinks during Boot
- `pixelforge_studio/capsuleruntime/runtime.go` (modify if needed) — expose `IterBodyPairs(fn func(a, b *Body))` helper if it doesn't exist yet
- `pixelforge_studio/scripting/catalog/verb_recipes.go` (modify) — add `EventTopicCollisionBodyCollide = "collision/body_collide"` constant
- `pixelforge_loop/verbs_bus.go` (no change — stable contract)

**Approach:**
- `OnLateUpdate`: walk every pair of bodies via resolv's cell-grid Space broad-phase. For each pair `(a, b)`:
  - Lookup `a.Entity.Archetype` and `b.Entity.Archetype` (entity ID is stored on the body; the inverse lookup goes through `rt.CurrentScene.Entities`).
  - Skip self-pairs and dead entities (via `rt.VisualEffects[id].Hidden` OR a new `rt.DeadEntities` set populated by `damage_sink.die`).
  - Publish `*VerbEvent{Topic: EventTopicCollisionBodyCollide, Args: map[string]any{"a_id": a_id, "b_id": b_id, "a_archetype": a_arch, "b_archetype": b_arch}}`.
- Pair de-duplication: emit each pair exactly once per tick (a×b and b×a are the same event). Use a `pair-set` of `(min_id, max_id)` per tick, reset between ticks.
- Performance: with O(n²) naive scan, 100 entities = 4950 pairs per tick. Resolv's cell-grid Space cuts this to typical O(n × neighbors_per_cell). For arcade scale (50–200 entities), within budget. Profile in U13 if pixel-hash test runtime balloons.
- Determinism: pair enumeration order must be stable (sort by `(min_id, max_id)` before emission). This guarantees the bus event sequence is deterministic — same world state → same event order. Load-bearing for plan-009's bus-event-parity CI gate.
- Tunneling note: fast projectiles (vel > cell_size per tick) can pass through targets between ticks. Mitigated by velocity cap in U5's `fire_projectile` recipe; documented in `collision_sink.go` doc comment.

**Patterns to follow:**
- `pixelforge_loop.VerbsBus().Publish` — the established sink-publish seam (`verbs_bus.go`).
- `damage_sink.go` — established sink shape for `OnLateUpdate`-style subscribers.
- `docs/solutions/scripting-runtime-design.md` — `catalog.RegisterStep/Action` pattern; `pievent.Target` for typed events.

**Test scenarios:**
- *Two overlapping bodies emit one event:* Place ship at (0,0) size 16, rock at (8,8) size 16; tick; assert exactly one `collision/body_collide` event on the bus with `a_id, b_id` = (ship, rock).
- *Non-overlapping bodies emit nothing:* Place ship at (0,0), rock at (1000, 1000); tick; assert zero collision events.
- *De-duplication:* Three overlapping bodies (A, B, C all mutually overlapping); tick; assert exactly 3 events (A×B, A×C, B×C), not 6.
- *Hidden entity skipped:* Two overlapping bodies, one marked Hidden via VisualEffects; tick; assert zero events.
- *Archetype labels correct:* Place ship + rock; tick; assert event Args contain `a_archetype = "Ship", b_archetype = "Rock"` (or sorted alphabetically by archetype — pick one and document).
- *Deterministic event order across ticks:* Same world state + same tick → same event sequence across 100 runs.
- *Stress (50 bodies):* 50 entities with random positions; assert tick completes in < 5 ms on a typical dev machine (budget check; not a hard gate, but flag if it balloons).
- *Snapshot round-trip:* Collisions are NOT serialized (they're recomputed each tick); save mid-collision; load; tick; assert collision event re-emitted on the new tick (confirms the "collision is a pure function of bodies" invariant from ideation rejection row 34).
- *Bus event parity (CI):* Long-tag test that records collision events from a synthetic scene; baseline-compares to a checked-in expected sequence.

**Verification:** `go test ./pixelforge_studio/capsuleruntime/...` passes; sink registered in Boot pipeline; bus events visible in `pixelforge_replay.Recorder` output.

---

### U4. `RecipeOnCollide` + `TriggerWhenBodyCollide` catalog wiring

**Goal:** Add the `RecipeOnCollide` recipe + `TriggerWhenBodyCollide` trigger to the verb catalog so verb sheets can subscribe to body-collide events with archetype-pair filters. The recipe binds (archetype-A, archetype-B) → action chain (e.g., `damage/take_damage` + `spawn/destroy_other` + `globals.score += 100`).

**Requirements:** R5 (extended), R18, R19, R26.

**Dependencies:** U3 (the topic must exist before the recipe references it).

**Files:**
- `pixelforge_studio/scripting/catalog/verb_recipes.go` (modify) — add `TriggerWhenBodyCollide = "when_body_collide"` constant
- `pixelforge_studio/scripting/catalog/builtin_arcade.go` (modify) — add `RecipeOnCollide` recipe with `archetype_a` + `archetype_b` filter params, `actions` action chain
- `pixelforge_studio/scripting/catalog/verb_recipes_test.go` (modify) — tests for filter matching
- `pixelforge_studio/capsuleruntime/dispatch.go` (or equivalent recipe-dispatch seam — modify) — recognize `TriggerWhenBodyCollide` events and match them to subscribed recipes by archetype pair
- `docs/verb-catalog.md` (regenerated by plan-009 U24 mechanism — no manual edit)

**Approach:**
- Recipe shape: `{kind: "on_collide", archetype_a: "Bullet", archetype_b: "Rock", actions: [{kind: "damage/take_damage", target: "b_id", amount: 1}, {kind: "spawn/destroy_other", target: "a_id"}, {kind: "globals/inc", path: "score", by: 100}]}`.
- Archetype-pair match: the dispatch layer receives a `collision/body_collide` event with `a_archetype, b_archetype`. Iterates registered `RecipeOnCollide` entries; matches pair regardless of order (`a × b` matches recipes with `(a, b)` OR `(b, a)`); if matched, executes actions with `a_id, b_id` substituted as named references (`target: "a_id"` resolves to the event's `a_id`).
- "Any" wildcard: archetype filter `"*"` matches any archetype. Useful for "Player × any → take damage" patterns.
- Self-collide guard: same-archetype recipes (`archetype_a == archetype_b`) match only if a_id ≠ b_id.
- Action chain is the existing catalog action-list machinery from plan-009 U7 — no new dispatch infrastructure.

**Patterns to follow:**
- Existing recipe definitions in `builtin_arcade.go` (e.g., `RecipeWhenInputJump`).
- Existing trigger constants in `verb_recipes.go:122-139`.
- `docs/solutions/scripting-runtime-design.md`.

**Test scenarios:**
- *Pair match (ordered):* Recipe `(Bullet, Rock)` + event with `(a=Bullet, b=Rock)`; assert actions fire with `a_id=Bullet.id, b_id=Rock.id`.
- *Pair match (reverse-ordered):* Recipe `(Bullet, Rock)` + event with `(a=Rock, b=Bullet)`; assert actions fire with `a_id=Rock.id` (the event's actual a_id, not the recipe's archetype_a).
- *Wildcard match:* Recipe `(Player, "*")` + event `(Player, Goomba)`; assert match.
- *No match:* Recipe `(Bullet, Rock)` + event `(Player, Goomba)`; assert no actions fire.
- *Self-collide guard:* Recipe `(Asteroid, Asteroid)` + event with same entity ID twice; assert NO match (self-pair excluded).
- *Multiple recipes one event:* Two `RecipeOnCollide` entries both matching the pair; both fire (action-chain composition).
- *Catalog regen:* Run `make verb-catalog` (plan-009 U24); assert `docs/verb-catalog.md` contains the new `RecipeOnCollide` entry + trigger.

**Verification:** `go test ./pixelforge_studio/scripting/catalog/...` + `go test ./pixelforge_studio/capsuleruntime/...` passes; `go generate ./pixelforge_studio/scripting/catalog/...` produces updated `docs/verb-catalog.md` (CI gate per plan-009 U24).

---

### U5a. Generic `Pool[T]` allocator in `pixelforge_pool`

**Goal:** Extend `pixelforge_pool/` with a generic `Pool[T any]` type providing `Acquire / Release / ForEach / Active`. Fixed-size pre-allocated array indexed by `alive bool`; deterministic next-free-index slot selection (LIFO). No dependencies on capsuleruntime, project, or physics — pure data structure.

**Requirements:** R5 (extended substrate).

**Dependencies:** none.

**Files:**
- `pixelforge_pool/entity_pool.go` (new) — `type Pool[T any] struct{ slots []T; alive []bool; freeStack []int; max int }`; `NewPool[T](size int) *Pool[T]`; `(*Pool[T]).Acquire() (*T, int, bool)`; `(*Pool[T]).Release(idx int)`; `(*Pool[T]).ForEach(fn func(idx int, item *T))`; `(*Pool[T]).Active() int`
- `pixelforge_pool/entity_pool_test.go` (new)

**Approach:**
- Pre-allocate `slots` + `alive` + `freeStack` arrays at construction. No `append` during use.
- Acquire: pop from `freeStack`; mark alive; return pointer + index.
- Release: mark dead; push index to `freeStack`.
- ForEach: iterate `slots` in index order; call `fn` only when `alive[i]`.
- Determinism: same Acquire/Release sequence → same slot indices across runs.

**Test scenarios:**
- *Acquire returns expected indices:* `NewPool[int](5)`; acquire 3; assert indices = [0,1,2] (or [4,3,2] for LIFO from initial fill — pick one + document).
- *Release returns slot:* Acquire all 5; release index 2; acquire; assert returns index 2.
- *Pool exhaustion:* Acquire all 5; sixth returns `(nil, -1, false)`.
- *ForEach iterates only alive:* Acquire 3 slots; release one; ForEach visits exactly 2.
- *Multiple acquire/release cycles deterministic:* Same sequence on two pool instances → same final state.

**Verification:** `go test ./pixelforge_pool/...` passes.

---

### U5b. `Scene.PoolBudgets` schema field + per-archetype pool registration at Boot

**Goal:** Add `PoolBudgets map[string]int` field to `pixelforge_project.Scene` (schema-additive with `omitempty`). At capsuleruntime Boot, for each `(archetype, budget)` in the scene's PoolBudgets, create `rt.Pools[archetype] = pixelforge_pool.NewPool[pixelforge_physics.Body](budget)` and pre-register bodies with `rt.Physics` as inactive shapes (activated on Acquire).

**Requirements:** R5 (extended), schema additivity invariant.

**Dependencies:** U5a.

**Files:**
- `pixelforge_project/scene.go` (modify) — add `PoolBudgets map[string]int \`json:"pool_budgets,omitempty"\`` field
- `pixelforge_project/scene_test.go` (modify) — round-trip a scene with and without PoolBudgets
- `pixelforge_studio/capsuleruntime/runtime.go` (modify) — `Runtime` gains `Pools map[string]*pixelforge_pool.Pool[pixelforge_physics.Body]`; `Boot` walks `CurrentScene.PoolBudgets` + creates pools
- `pixelforge_studio/capsuleruntime/runtime_pool_test.go` (new)

**Approach:**
- Schema-additive: existing carts without PoolBudgets load with empty map; no validation error.
- Boot registers physics bodies in inactive state (use existing resolv.Space mechanism for inactive shapes, OR just don't add to Space until Acquire).
- Activation/deactivation: on Acquire, add body to Space; on Release, remove from Space.

**Test scenarios:**
- *Scene with PoolBudgets loads:* `pixelforge_project.LoadReader` against fixture with `{"pool_budgets":{"Bullet":8}}`; assert field populated.
- *Scene without PoolBudgets loads (back-compat):* Pre-edit cart loads cleanly; `PoolBudgets == nil` or empty map.
- *Boot creates pools:* Scene with `PoolBudgets["Bullet"] = 8`; after Boot, `len(rt.Pools["Bullet"].slots) == 8`.
- *Empty PoolBudgets:* Boot succeeds with no pools registered.
- *Multiple archetypes:* PoolBudgets with 3 entries; all 3 pools created.

**Verification:** `go test ./pixelforge_project/...` + `go test ./pixelforge_studio/capsuleruntime/...` passes.

---

### U5c. `combat/fire_projectile` verb + `RecipeFireProjectile` + projectileSink

**Goal:** Add the `combat/fire_projectile` event topic + `RecipeFireProjectile` catalog entry + `projectile_sink.go` that handles the event by acquiring a body from the per-archetype pool, setting position/velocity/TTL, wiring into `rt.Bodies` and `rt.CurrentScene.Entities`. U3's collision sink picks up the new body next tick via normal pair enumeration.

**Requirements:** R5 (extended), R18, R26.

**Dependencies:** U3, U5b.

**Files:**
- `pixelforge_studio/capsuleruntime/projectile_sink.go` (new)
- `pixelforge_studio/capsuleruntime/projectile_sink_test.go` (new)
- `pixelforge_studio/scripting/catalog/verb_recipes.go` (modify) — add `EventTopicCombatFireProjectile = "combat/fire_projectile"` constant
- `pixelforge_studio/scripting/catalog/builtin_arcade.go` (modify) — add `RecipeFireProjectile` (spawner_archetype, projectile_archetype, direction_arg, speed, ttl_ticks)
- `pixelforge_studio/capsuleruntime/registry.go` (modify) — register projectileSink

**Approach:**
- `combat/fire_projectile Args{spawner, projectile_archetype, direction, speed, ttl_ticks}`: lookup spawner body; pool := rt.Pools[projectile_archetype]; body, idx, ok := pool.Acquire(); if !ok, publish `combat/projectile_dropped` notification (deterministic) and return; else set body.Position/Velocity using SinDeg LUT for direction; register TTL via existing bomb_timer.go mechanism.
- Velocity cap: clamp speed < cell_size (documented in collision_sink.go from U3).
- Position pixelforge.Vec2 from spawner's Position; offset by 1 unit in firing direction (avoid self-collide).

**Test scenarios:**
- *Fire projectile spawns body:* Setup hero at (10,10), pool budget 8; fire direction=0 speed=4; assert pool.Active() == 1; body.Position ≈ (10,10); body.Velocity = (4,0).
- *Direction 90° produces y-velocity:* direction=90 speed=4; assert body.Velocity ≈ (0,4) via LUT trig.
- *Pool exhaustion drops + notifies:* Fire 9 times against pool of 8; ninth publishes `combat/projectile_dropped` event; assert no panic.
- *TTL releases pool slot:* Fire with ttl=10; advance 10 ticks; assert pool.Active() == 0.
- *Projectile collides with target:* Fire toward Rock; advance until collision tick; assert `collision/body_collide(Bullet, Rock)` event fires.

**Verification:** `go test ./pixelforge_studio/capsuleruntime/...` passes.

---

### U5d. Snapshot extension for pool state round-trip

**Goal:** Extend snapshot encoder/decoder (plan-009 U10's `snapshot.go`) to serialize pool state: active slots per-archetype with per-slot position/velocity/ttl; free slot count. Load reconstructs by Acquire'ing N times in deterministic order and restoring per-slot state. Preserves the plan-008 byte-equal-round-trip invariant for any cart with live pooled entities.

**Requirements:** R7 (snapshot round-trip), R5 (extended).

**Dependencies:** U5a, U5b, U5c.

**Files:**
- `pixelforge_studio/capsuleruntime/snapshot.go` (modify) — serialize `Pools` state alongside `Entities`
- `pixelforge_studio/capsuleruntime/snapshot_pool_test.go` (new)

**Approach:**
- Snapshot schema-additive field `pools: {archetype: {active: [{slot, pos, vel, ttl}], free_count: N}}`.
- Determinism: per-archetype iteration order is `sort.Strings(keys)`; per-slot iteration order is index ascending. Same world state → same bytes.
- Old snapshots without pools field load cleanly (treated as no pools).

**Test scenarios:**
- *Fire 3 projectiles, snapshot, load, assert byte-equal:* Save mid-firing-sequence; reload into fresh runtime; encode again; assert byte-equal.
- *Per-slot TTL preserved:* Snapshot with bullet at ttl=42; load; assert ttl == 42.
- *Old snapshot loads (back-compat):* v2 snapshot without pools field loads with empty pools.
- *Multiple archetypes preserved:* Pools for Bullet + Particle both serialize + deserialize.

**Verification:** `go test ./pixelforge_studio/capsuleruntime/...` passes; AE4-style round-trip extended to assert pool determinism.

**Dependencies:** U3 (pool members participate in collision pairs).

**Files:**
- `pixelforge_pool/entity_pool.go` (new) — `type Pool[T any] struct{ active, free []T; max int }`; `Acquire() (*T, bool)`, `Release(idx int)`, `ForEach(fn func(*T))`, `Active() int`
- `pixelforge_pool/entity_pool_test.go` (new)
- `pixelforge_project/scene.go` (modify) — add `PoolBudgets map[string]int` field with `omitempty` (schema-additive per plan-008 invariant)
- `pixelforge_studio/capsuleruntime/runtime.go` (modify) — `Runtime` now holds `Pools map[string]*pixelforge_pool.Pool[pixelforge_physics.Body]` keyed by archetype; populated at Boot from `Scene.PoolBudgets`
- `pixelforge_studio/capsuleruntime/projectile_sink.go` (new) — handles `combat/fire_projectile`: acquires body from pool, sets initial velocity + position + archetype, wires into `rt.Bodies` and `rt.CurrentScene.Entities`
- `pixelforge_studio/capsuleruntime/projectile_sink_test.go` (new)
- `pixelforge_studio/scripting/catalog/builtin_arcade.go` (modify) — `RecipeFireProjectile { spawner: "Player", projectile_archetype: "Bullet", direction_arg: "facing", speed: 4, ttl_ticks: 120 }`
- `pixelforge_studio/scripting/catalog/verb_recipes.go` (modify) — add `EventTopicCombatFireProjectile = "combat/fire_projectile"` constant
- `pixelforge_studio/capsuleruntime/snapshot.go` (modify) — extend snapshot encoder/decoder to walk pool state (active bodies serialize per-entity; free slots serialize as count)

**Approach:**
- `Pool[T]` is fixed-size pre-allocated (no `append` during gameplay). Indexed by `alive bool` (or by `active []int` + `free []int` queues). `Acquire` pops from `free`, marks alive, returns `*T` + slot index. `Release(idx)` marks dead, returns slot to free queue. `ForEach` iterates only alive slots.
- Pool registration: at Boot, for each `(archetype, budget)` in `Scene.PoolBudgets`, create `rt.Pools[archetype] = NewPool[pixelforge_physics.Body](budget)`. Pre-allocate all bodies + register them with `rt.Physics` (as inactive shapes initially — toggle active on Acquire).
- `combat/fire_projectile Args{spawner: "hero", projectile_archetype: "Bullet", direction: 90, speed: 4, ttl_ticks: 120}`: lookup spawner body for muzzle position; `pool := rt.Pools["Bullet"]`; `body, ok := pool.Acquire()`; if `!ok` (pool exhausted), drop silently (deterministic) + emit `combat/projectile_dropped` notification event for replay debugging; if ok, set `body.Position = spawner.Position`, `body.Velocity = direction × speed` (using SinDeg/CosDeg LUT), register a TTL timer (existing `bomb_timer.go` mechanism from plan-009 U15) to call `pool.Release(idx)` on expiry.
- Velocity cap: clamp `speed` to `< cell_size` to prevent tunneling per the collision-sink doc.
- Recipe shape: `RecipeFireProjectile` exposes `spawner_archetype, projectile_archetype, direction_arg, speed, ttl_ticks` as Inspector-visible params.
- Snapshot determinism: pool active slots serialize per-entity (id + position + velocity + ttl_remaining). Free slot count serializes as a single integer. Load reconstructs by Acquire'ing N times and restoring per-slot state.
- Pool slot selection: deterministic (next-free-index). Same world state → same Acquire result → same body ID across runs.

**Patterns to follow:**
- Existing `pipool.go` shape (for the package convention).
- `pixelforge_studio/capsuleruntime/bomb_timer.go` (plan-009 U15) — TTL pattern.
- `docs/solutions/ring-buffer-snapshot-store.md` — pool state must snapshot byte-equal.

**Test scenarios:**
- *Acquire returns next-free slot deterministically:* `NewPool(5)`; acquire 3 times; assert slot indices `[0, 1, 2]`. Release slot 1; acquire; assert slot 1 returned (LIFO or FIFO — pick one and document).
- *Pool exhaustion:* `NewPool(2)`; acquire 3 times; third returns `(nil, false)`; assert no panic.
- *Release returns slot to free queue:* Acquire all; release one; acquire; assert success.
- *ForEach iterates only alive:* `NewPool(5)`; acquire 3; release slot 1; ForEach; assert callback fires for 2 entries (slots 0, 2).
- *Fire projectile spawns body:* `Scene.PoolBudgets["Bullet"] = 10`; fire from hero at direction 0, speed 4; assert `rt.Bodies` count incremented by 1; body Position matches hero Position; Velocity matches (4, 0).
- *Projectile TTL release:* Fire projectile with TTL=10; advance 10 ticks; assert pool slot released back to free queue; `rt.Bodies` count decremented.
- *Projectile collides with target:* Fire projectile toward rock at speed 4; advance enough ticks; assert `collision/body_collide` event fires with `(Bullet, Rock)`; verify the catalog `RecipeOnCollide(Bullet, Rock)` actions run (damage/take_damage + spawn/destroy_other).
- *Pool exhaustion drops silently:* Fire projectile when pool exhausted; assert no panic; assert `combat/projectile_dropped` notification on the bus (replay debugging signal).
- *Snapshot round-trip:* Fire 3 projectiles; snapshot; load; assert pool state matches (3 active, N-3 free, per-projectile position+velocity+ttl restored byte-equal).
- *Determinism:* Same scene + same input trace → same projectile IDs + positions across 100 runs.
- *Schema additive:* `Scene` without `PoolBudgets` field loads with empty map (no error).

**Verification:** `go test ./pixelforge_pool/...` + `go test ./pixelforge_studio/capsuleruntime/...` passes; snapshot round-trip test asserts byte-equal mid-firing-sequence; `RecipeFireProjectile` visible in regenerated `docs/verb-catalog.md`.

---

### U6. `RecipeWinWhen` + `RecipeLoseWhen` counter-polling

**Goal:** Add `RecipeWinWhen` + `RecipeLoseWhen` recipes with a `Predicate` reusing the existing `catalog.ConditionBuilder` registry. Predicate kinds: `counter_geq`, `counter_lte`, `entity_count_eq`, `globals_lte`. A new `winLoseSink` subscribes to `EventLateUpdate`, evaluates both predicates against `rt.Globals` and `rt.CurrentScene.Entities` each tick, and on first satisfaction publishes a `scene/transition` event (win → winScene, lose → loseScene; both declared on the recipe). Win takes precedence on same-tick satisfaction. `spawn_sink` increments/decrements `globals.entity_count:<archetype>` on spawn/destroy; `damage_sink`'s die cascade also triggers decrements via existing flow.

**Requirements:** R5 (extended), R18, R26.

**Dependencies:** U3 (collision is the trigger that drives counter changes for most games).

**Files:**
- `pixelforge_studio/capsuleruntime/win_lose_sink.go` (new) — `type winLoseSink struct{ rt *Runtime; fired bool }`; `OnLateUpdate()` evaluates predicates
- `pixelforge_studio/capsuleruntime/win_lose_sink_test.go` (new)
- `pixelforge_studio/scripting/catalog/builtin_arcade.go` (modify) — `RecipeWinWhen { predicate: ..., transition_to: "victoryScene" }`; `RecipeLoseWhen { predicate: ..., transition_to: "gameOverScene" }`
- `pixelforge_studio/scripting/catalog/verb_recipes.go` (modify) — predicate kind constants
- `pixelforge_studio/capsuleruntime/spawn_sink.go` (modify) — increment `globals.entity_count:<archetype>` on entity_spawn; decrement on destroy_self/destroy_other
- `pixelforge_studio/capsuleruntime/registry.go` (modify) — register `winLoseSink` to `EventLateUpdate` AFTER collisionSink (so this tick's collisions can affect this tick's win check)
- `pixelforge_menus/` (no change — reuses existing `game_over` menu shape per plan-009 referencing `pixelforge_menus/registry.go:148`)

**Approach:**
- Predicate evaluation against `rt.Globals` and `rt.CurrentScene.Entities`:
  - `counter_geq Args{path: "score", value: 1000}`: `rt.Globals["score"] >= 1000`
  - `counter_lte Args{path: "player_hp", value: 0}`: `rt.Globals["player_hp"] <= 0`
  - `entity_count_eq Args{archetype: "Enemy", value: 0}`: count entities in current scene with that archetype == 0
  - `globals_lte Args{path: "lives", value: 0}`: same shape as counter_lte (alias)
- First-satisfaction wins: `winLoseSink` carries a `fired bool` flag; set on first satisfaction; resets only on `scene/transition` to a fresh game scene. Prevents repeated fire on subsequent ticks where the predicate stays satisfied.
- Lose > win precedence: if both fire on same tick, publish lose event only. Implementation: evaluate lose recipes first; if any match, set `fired` and publish lose; else evaluate win.
- Multiple win recipes / multiple lose recipes per scene: allowed; first match wins.
- Schema: recipes live in `Project.WinLose []WinLoseRecipe` or per-`Scene.WinLose` — pick per-Scene (more flexible; v3 can add Project-level if needed). Schema-additive: scene without WinLose stays valid (no auto win/lose).
- `spawn_sink` extension: when an entity spawns/dies, the sink writes `rt.Globals["entity_count:Enemy"] += 1` (or -1). This bumps a counter the predicate reads — no per-tick scene iteration for `entity_count_eq` (O(1) lookup instead of O(n) entity walk).

**Patterns to follow:**
- Existing `damage_sink.go` (plan-009 U9) reads/writes `rt.Globals` — established pattern.
- Existing `scene_sink.go` `change` handler (plan-009 U7) is the receiver of the `scene/transition` event.
- `pixelforge_menus/registry.go:148` `game_over` menu — reuse for the loseScene target.

**Test scenarios:**
- *Counter-geq win:* `globals.score = 999`; advance tick; assert no win. Set `globals.score = 1000`; advance tick; assert `scene/transition victoryScene` event published.
- *Counter-lte lose:* `globals.player_hp = 1`; advance tick; no lose. Set to 0; advance; lose event published.
- *Entity_count_eq win:* Scene with 3 Enemy entities; predicate `entity_count_eq Enemy 0`; advance; no win. Destroy 3 enemies; advance; win event published.
- *Lose > win precedence:* Scene where both predicates satisfy on same tick (e.g., last enemy killed by an attack that also drops player HP to 0); assert lose event only; win event NOT published.
- *First-satisfaction only:* Counter-geq fires at tick 100; advance to tick 200; assert event NOT re-published despite predicate still satisfied.
- *Reset on scene transition:* After win → victoryScene → restart to game scene; predicate re-evaluable.
- *Spawn increments counter:* Boot scene with 0 Enemy; fire `spawn/entity Args{prefab: "goomba"}`; assert `globals.entity_count:Enemy == 1`.
- *Destroy decrements counter:* Same scene with 3 Enemy; fire `damage/die Args{entity: "goomba_1"}` (cascade through die → destroy); assert `globals.entity_count:Enemy == 2`.
- *Schema additive — scene without WinLose loads:* Scene with no `WinLose` field loads cleanly; sink does nothing.
- *Snapshot round-trip:* Mid-game with `fired=false`; snapshot; load; advance; predicate evaluates correctly on new instance.
- *Determinism:* Same trace → same win/lose firing tick across 100 runs.

**Verification:** `go test ./pixelforge_studio/capsuleruntime/...` + `go test ./pixelforge_studio/scripting/catalog/...` passes; new recipes visible in regenerated `docs/verb-catalog.md`.

---

### Phase C — A2: System-runtime distribution

### U7. `cmd/pixelforge-runtime` binary

**Goal:** Create the system-installed runtime binary `cmd/pixelforge-runtime/main.go`. Reads `.pforge` cart from: (1) explicit CLI arg `pixelforge-runtime path/to/game.pforge`; (2) stdin pipe when no arg given and stdin is not a TTY; (3) OS file association (path comes in via `os.Args[1]`). Boots the runtime + runs Ebitengine game loop. Shares cart-loading logic with `cmd/pixelforge-player` via `pixelforge_cart.LoadFromPath`.

**Requirements:** R23, R1 (extended).

**Dependencies:** none — relies only on existing capsuleruntime + pixelforge_ebiten + pixelforge_cart packages.

**Files:**
- `cmd/pixelforge-runtime/main.go` (new)
- `cmd/pixelforge-runtime/main_test.go` (new) — smoke test loading a fixture cart
- `pixelforge_cart/loadfrompath.go` (new) — `LoadFromPath(path string) ([]byte, error)` extracted from existing `ReadSelf` logic
- `pixelforge_cart/loadfrompath_test.go` (new)

**Approach:**
- `main()` resolution order:
  1. If `len(os.Args) >= 2`, treat `os.Args[1]` as path → `pixelforge_cart.LoadFromPath(path)`.
  2. Else if stdin is not a TTY (`!term.IsTerminal(syscall.Stdin)`), `io.ReadAll(os.Stdin)`.
  3. Else: print stderr help (`Usage: pixelforge-runtime <path/to/game.pforge>`) + exit 2.
- After cart bytes loaded: `pixelforge_project.LoadReader(bytes.NewReader(cart))` → `capsuleruntime.Boot(p, embeddedAssets, capsuleruntime.Options{})` → `pixelforge_ebiten.Run()`. Same shape as `cmd/pixelforge-player/main.go` post-cart-load.
- WASM build: same `main.go` works under `GOOS=js GOARCH=wasm` — the entry just reads cart from `js.Global().Get("__pixelforgeCart")` instead (build-tagged via `pixelforge_cart.LoadFromJS()` mirror). For the triplet path (U10), the JS bootloader fetches `game.pforge` and stashes it on the window object before invoking `go.run`.
- Window title: derived from `Project.Title` if present, else the cart filename's basename.
- `--version` flag prints build version + supported `.pforge` schema versions.
- `--smoke-tick N` flag (debug, same as plan-009 U2): runs N ticks then exits with code 0. Used by U16 CI smoke.

**Patterns to follow:**
- `cmd/pixelforge-player/main.go` (plan-009 U2) — same Ebitengine + Boot pipeline.
- `pixelforge_cart.ReadSelf` (plan-009 U1) — extract the cart-load + validate logic into `LoadFromPath` so both binaries share it.
- Existing stdin-vs-TTY detection: `golang.org/x/term.IsTerminal(int(os.Stdin.Fd()))` — pure-Go, cross-platform.

**Test scenarios:**
- *Path arg loads cart:* Pre-built test cart at `/tmp/test.pforge`; `pixelforge-runtime /tmp/test.pforge --smoke-tick 60`; exit 0.
- *Stdin pipe loads cart:* `cat /tmp/test.pforge | pixelforge-runtime --smoke-tick 60`; exit 0.
- *No arg + TTY:* `pixelforge-runtime` (with TTY stdin); exits 2 with help message on stderr.
- *Missing file:* `pixelforge-runtime /nonexistent.pforge`; exits non-zero with `file not found` diagnostic.
- *Corrupt cart:* Path to a binary file that's not a valid .pforge; exits non-zero with `parse error` diagnostic.
- *`--version`:* Prints version + schema versions; exit 0.
- *`--smoke-tick 60`:* Runs 60 ticks via the existing replay-style harness path; exit 0.
- *WASM smoke:* `GOOS=js GOARCH=wasm go build ./cmd/pixelforge-runtime` compiles cleanly; resulting `.wasm` is non-empty.
- *Cart load shared with player:* `LoadFromPath` test asserts byte-equal output for a cart path that's also a footer-appended player binary (the same cart bytes, two different load paths).

**Verification:** `go test ./cmd/pixelforge-runtime/...` + `go test ./pixelforge_cart/...` passes; the binary boots one of the four proof carts when handed a path.

---

### U8a. `.goreleaser.yaml` skeleton + cross-compile matrix (no installers)

**Goal:** Add `.goreleaser.yaml` at repo root with just the `builds` + `archives` blocks: cross-compile `pixelforge-runtime` for linux-amd64, darwin-amd64, darwin-arm64, windows-amd64 into tar.gz/zip archives. No installer-packaging yet — that's U8b/U8c/U8d per OS. `make goreleaser-snapshot` target invokes `goreleaser release --snapshot --clean`.

**Requirements:** R23 (substrate).

**Dependencies:** U7.

**Files:**
- `.goreleaser.yaml` (new) — builds + archives sections only
- `Makefile` (modify) — add `goreleaser-snapshot` target
- `docs/install-linux.md` (modify) — add the snapshot-build instructions for developers

**Approach:**
- One `builds` entry for `pixelforge-runtime`; standard goos/goarch matrix.
- Archive naming: `pixelforge-runtime_<version>_<goos>_<goarch>.tar.gz` (.zip for windows).
- No hooks, no signing, no packaging in this unit.

**Test scenarios:**
- *make goreleaser-snapshot succeeds:* On a developer machine with `goreleaser` installed; outputs under `dist/`.
- *Archives contain runtime binary:* Untar a snapshot archive; assert `pixelforge-runtime` binary present, executable.
- *YAML parses:* `goreleaser check` exits 0.

**Verification:** `make goreleaser-snapshot` produces 4 archives.

---

### U8b. Linux installer — `.deb` + `.rpm` + `.AppImage` with `.pforge` MIME registration

**Goal:** Extend `.goreleaser.yaml` with `nfpms` block emitting `.deb` + `.rpm`. Add `.AppImage` via the `appimage` post-build hook (or `linuxdeploy` invocation). Include the `.desktop` file + shared-mime-info XML registering `application/x-pixelforge` for `.pforge` extension. Post-install scripts run `update-mime-database` + `update-desktop-database`.

**Requirements:** R23.

**Dependencies:** U8a.

**Files:**
- `pixelforge_studio/installer/linux/pixelforge-runtime.desktop` (new) — XDG .desktop with `MimeType=application/x-pixelforge`
- `pixelforge_studio/installer/linux/application-x-pixelforge.xml` (new) — shared-mime-info XML
- `pixelforge_studio/installer/linux/postinstall.sh` (new) — `update-mime-database` + `update-desktop-database`
- `pixelforge_studio/installer/linux/postremove.sh` (new) — cleanup
- `.goreleaser.yaml` (modify) — add nfpms entry + appimage hook
- `pixelforge_studio/installer/README.md` (new) — linux section first

**Test scenarios:**
- *Snapshot produces .deb + .rpm + .AppImage:* `make goreleaser-snapshot`; verify all three under `dist/`.
- *.deb install registers MIME (in container):* `dpkg -i` in ubuntu container; `xdg-mime query default application/x-pixelforge` returns `pixelforge-runtime.desktop`.
- *.AppImage runs:* Snapshot AppImage launches with `--version` arg, exit 0.

**Verification:** Three Linux package formats produced; MIME registration verified in container.

---

### U8c. macOS installer — `.pkg` with ad-hoc codesign + UTI registration

**Goal:** Extend `.goreleaser.yaml` with macOS `.pkg` packaging (via `productbuild` hook). Bundle `pixelforge-runtime.app` into the pkg with `Info.plist` declaring `CFBundleDocumentTypes` for `.pforge` and `UTExportedTypeDeclarations` for `com.pixelforge.cart` UTI. Post-build hook ad-hoc codesigns via `codesign --force --sign -` (mirrors plan-009 U1's pattern).

**Requirements:** R23.

**Dependencies:** U8a.

**Files:**
- `pixelforge_studio/installer/darwin/Info.plist` (new) — bundle metadata + document types
- `pixelforge_studio/installer/darwin/build-pkg.sh` (new) — productbuild + codesign sequence
- `.goreleaser.yaml` (modify) — add darwin packaging hook
- `pixelforge_studio/installer/README.md` (modify) — add darwin section
- `docs/install-macos.md` (modify) — add .pkg install instructions

**Test scenarios:**
- *Snapshot on darwin produces .pkg:* `make goreleaser-snapshot` on a darwin host; .pkg under `dist/`.
- *codesign --verify passes:* Built binary inside .pkg passes verify with ad-hoc identity.
- *Launching on darwin/arm64:* Installed runtime launches without `killed: 9`.
- *Double-click `.pforge` opens runtime (manual smoke):* After install, double-click a .pforge file; runtime launches.

**Verification:** macOS .pkg installs cleanly; double-click flow works on a real Mac.

---

### U8d. Windows installer — `.msi` with WiX file-association registry entries

**Goal:** Extend `.goreleaser.yaml` to invoke WiX (`candle` + `light`) building a `.msi` from `pixelforge_studio/installer/windows/pixelforge-runtime.wxs`. The WiX source declares registry entries `HKCR\.pforge` → `PixelForge.Cart` + `HKCR\PixelForge.Cart\shell\open\command` → `"...\pixelforge-runtime.exe" "%1"`.

**Requirements:** R23.

**Dependencies:** U8a.

**Files:**
- `pixelforge_studio/installer/windows/pixelforge-runtime.wxs` (new) — WiX source
- `pixelforge_studio/installer/windows/build-msi.bat` (new) — candle + light invocation
- `.goreleaser.yaml` (modify) — windows packaging hook
- `pixelforge_studio/installer/README.md` (modify) — add windows section

**Test scenarios:**
- *Snapshot produces .msi:* On windows host (or via wine in CI); .msi under `dist/`.
- *WiX compiles cleanly:* `candle pixelforge-runtime.wxs` exits 0.
- *.msi install registers file association (manual on windows):* After install, regedit shows `HKCR\.pforge` entry.

**Verification:** .msi produced; manual install on a Windows machine registers `.pforge` correctly.

**Dependencies:** U7.

**Files:**
- `.goreleaser.yaml` (new) — root-level config
- `pixelforge_studio/installer/linux/pixelforge-runtime.desktop` (new) — XDG .desktop entry registering `.pforge` MIME
- `pixelforge_studio/installer/linux/application-x-pixelforge.xml` (new) — shared-mime-info XML defining `application/x-pixelforge` MIME for `.pforge` extension
- `pixelforge_studio/installer/darwin/Info.plist.fragment` (new) — CFBundleDocumentTypes entry for `.pforge`
- `pixelforge_studio/installer/windows/pixelforge-runtime.wxs` (new) — WiX file-association registry entries
- `pixelforge_studio/installer/README.md` (new) — explains the manifest layout + GoReleaser hook order
- `Makefile` (modify) — add `goreleaser-snapshot` target invoking `goreleaser release --snapshot --clean`

**Approach:**
- `.goreleaser.yaml`:
  - `builds`: one entry for `pixelforge-runtime`; `goos: [linux, darwin, windows]`; `goarch: [amd64, arm64]` (filter ignores windows-arm64 + linux-arm64 if not in scope yet — pick at implementation).
  - `archives`: `tar.gz` for linux/darwin, `zip` for windows.
  - `nfpms`: emit `.deb` + `.rpm` for linux, including the `.desktop` + MIME XML files at the correct paths (`/usr/share/applications/`, `/usr/share/mime/packages/`). Post-install scripts run `update-mime-database` and `update-desktop-database`.
  - `dmg` or `app_bundle`: macOS `.pkg` includes the runtime + Info.plist fragment; ad-hoc codesigned via hook.
  - `msi`: Windows WiX template emits the file-association registry entries.
  - Hooks: pre-build `make runtimebins` (renamed from `make playerbins` to reflect the new role; see U15); post-build adhoc-codesign on darwin.
- Linux MIME registration: standard freedesktop pattern; `.desktop` declares `MimeType=application/x-pixelforge`; the XML defines the MIME glob `*.pforge`. Installer scripts call `update-mime-database /usr/share/mime` post-install.
- macOS Info.plist fragment: `CFBundleDocumentTypes` with `LSItemContentTypes` referencing a UTI `com.pixelforge.cart` (also declared via `UTExportedTypeDeclarations`). Installer drops a `pixelforge-runtime.app` bundle into `/Applications/`.
- Windows WiX: registry entries `HKCR\.pforge` → `PixelForge.Cart`; `HKCR\PixelForge.Cart\shell\open\command` → `"C:\Program Files\PixelForge\pixelforge-runtime.exe" "%1"`.
- GoReleaser does NOT publish the GitHub release in this plan (publication is a release-process step in U15); `--snapshot` mode for CI + local builds.
- macOS notarization is OUT of scope (deferred per plan-009 Scope Boundaries). Ad-hoc signature only; users will see "unidentified developer" on first launch (acceptable for v2 hobby-project posture per plan-009 trademark posture decision).

**Patterns to follow:**
- GoReleaser docs (`https://goreleaser.com/customization/`) — official patterns.
- Plan-009 U1's `codesign_darwin.go` — ad-hoc signature pattern.
- nFPM docs for `.deb` + `.rpm` (`https://nfpm.goreleaser.com/`).

**Test scenarios:**
- *Snapshot build succeeds (linux):* `make goreleaser-snapshot` produces `.deb` + `.rpm` + `.AppImage` artifacts under `dist/`.
- *Snapshot build succeeds (darwin):* same target produces `.pkg` artifact (test on darwin host).
- *Snapshot build succeeds (windows):* same target produces `.msi` (test in CI windows leg; locally on linux requires wine for wix tools — note this gracefully).
- *`.deb` registers MIME:* Install `.deb` in a clean Ubuntu container; assert `xdg-mime query default application/x-pixelforge` returns `pixelforge-runtime.desktop`.
- *.desktop file present:* After install, `/usr/share/applications/pixelforge-runtime.desktop` exists with the expected MimeType line.
- *Double-click smoke (manual / CI):* After install on each OS, double-click a `.pforge` file; runtime launches with that cart loaded. (Headless CI: invoke via `xdg-open game.pforge` on Linux + assert child process is `pixelforge-runtime`.)
- *Cross-platform manifest absence:* If `pixelforge_studio/installer/<os>/` is empty, `make goreleaser-snapshot` fails fast with a clear error (not a silent skip).
- *Ad-hoc codesign on darwin:* macOS `.pkg` install contains a runtime binary that passes `codesign --verify`; running it on darwin/arm64 does not produce `killed: 9`.

**Verification:** `make goreleaser-snapshot` produces artifacts for all target OSes; manual install of the Linux `.deb` registers the MIME type; darwin `.pkg` install passes codesign verify.

---

### U9. Studio "Build → Cart" (default) + "Build → Self-Contained Executable" (opt-in)

**Goal:** Replace the default Studio "Build → Host" button behavior with "Build → Cart" — produces just the `.pforge` cart bytes via the existing `codegen.EncodeCart`, writes to `<outputDir>/cart/<name>.pforge`, and shows a success toast with "Open" and "Reveal in Finder" actions. Add a new menu item "Build → Self-Contained Executable" that runs the plan-009 cart-append flow (preserved unchanged) for users who want one-file shipping.

**Requirements:** R24, R2 (reinterpreted).

**Dependencies:** none — the substrate is already in place (codegen.EncodeCart from plan-009 U3).

**Files:**
- `pixelforge_studio/buildpipeline/builders_long.go` (modify) — add `cartOnlyBuilder` alongside existing `hostLongBuilder` + `wasmLongBuilder`; register new target `TargetCart`
- `pixelforge_studio/buildpipeline/builders_long_test.go` (modify) — tests for cartOnlyBuilder
- `pixelforge_studio/build/workspace.go` (modify) — `BuildCart() <-chan struct{}` method; existing `BuildHost()` preserved (now wired to `TargetSelfContained` or renamed `BuildSelfContained()` — pick one and surface that label in the menu)
- `pixelforge_studio/build/workspace_test.go` (modify)
- `pixelforge_studio/editor/menu.go` (modify) — File → Build → Cart (default) + Build → Web + Build → Self-Contained Executable (advanced submenu)
- `pixelforge_studio/integration_test/build_pipeline_cart_test.go` (new) — `//go:build long`; end-to-end Studio → cart-only → runtime

**Approach:**
- `cartOnlyBuilder.Build`: `codegen.EncodeCart(req.Project)` → `os.WriteFile("<outDir>/cart/<name>.pforge", cartBytes, 0644)` → `emit(PhaseDone{ArtifactPath: <path>})`. No player-binary discovery, no append, no per-OS post-processing. Fast.
- Success toast: shows file size (`game.pforge (12.4 KB)`); buttons "Open" (`exec.Command("pixelforge-runtime", path)`) and "Reveal" (`exec.Command("xdg-open", filepath.Dir(path))` on linux, `open -R` on macOS, `explorer /select` on windows).
- "Open" button is the inner test of the system-runtime install: if the user has `pixelforge-runtime` on PATH, the cart opens; if not, a clean error toast points to the install instructions URL.
- Self-contained executable opt-in: same hostLongBuilder flow from plan-009 U3, just demoted from default. Menu label clarifies the size cost: "Build → Self-Contained Executable (~20 MB; no install required for recipients)."
- Existing hostLongBuilder + wasmLongBuilder are NOT deleted — only their menu-default status changes.

**Patterns to follow:**
- Plan-009 U12 (`build/workspace.go` BuildHost / BuildWASM funnel pattern) — the same channel-return shape.
- Plan-009 U3 `codegen.EncodeCart` — the cart-bytes producer is already extracted.

**Test scenarios:**
- *Cart-only build produces .pforge:* Build asteroids_proof.pforge via cartOnlyBuilder; assert output file at `<outDir>/cart/asteroids_proof.pforge` exists; size < 50 KB.
- *Cart byte-equal across builders:* `cartOnlyBuilder` output bytes equal the cart appended by `hostLongBuilder` (proves single source of truth on the cart format).
- *Cart loads in runtime:* Built cart → `pixelforge_cart.LoadFromPath` → `pixelforge_project.LoadReader` → assert project structure matches the source.
- *Cart opens via runtime smoke:* Build cart; invoke `pixelforge-runtime <path> --smoke-tick 60`; assert exit 0.
- *Self-contained menu item still works:* Same fixture; "Build → Self-Contained Executable"; assert .exe/.app/.bin produced via plan-009 U3 path (unchanged).
- *Toast actions:* Open action invokes runtime; Reveal action opens file manager — mock the exec layer in tests to assert correct command construction.
- *Covers AE1 (reinterpreted):* macOS user clicks "Build → Cart" → game.pforge produced → consumer on different macOS machine with `pixelforge-runtime` installed double-clicks the .pforge → game launches.

**Verification:** `go test -tags=long ./pixelforge_studio/buildpipeline -run TestCartOnlyBuilder` passes; manual Studio test produces a cart that opens in the installed runtime.

---

### U10a. `pixelforge_studio/exportweb/` package + `EmitTriplet` function

**Goal:** Create the new package emitting the four-file triplet to a target directory. Pure function: takes (cartBytes, wasmBytes, wasmExecJS, gameTitle, outDir) → writes index.html + wasm_exec.js + pixelforge-runtime.wasm + game.pforge. Includes new `index_template.html` that loads cart via `fetch()`. No builder integration in this unit.

**Requirements:** R25 (substrate).

**Dependencies:** U7 (runtime.wasm exists via U15's runtimebins).

**Files:**
- `pixelforge_studio/exportweb/doc.go` (new)
- `pixelforge_studio/exportweb/triplet.go` (new) — `EmitTriplet(opts TripletOptions) error`
- `pixelforge_studio/exportweb/triplet_test.go` (new)
- `pixelforge_studio/exportweb/index_template.html` (new) — fetch-based loader

**Test scenarios:**
- *EmitTriplet writes 4 files at expected paths.*
- *index.html references siblings via `<script src="wasm_exec.js">` + `fetch("game.pforge")` + `fetch("pixelforge-runtime.wasm")`.*
- *Byte-equal cart + wasm preserved through write.*
- *Optional zip:* When `Zip: true`, `<gameName>-web.zip` contains all 4 files.

**Verification:** `go test ./pixelforge_studio/exportweb/...` passes.

---

### U10b. `webTripletBuilder` integration + Studio menu wiring

**Goal:** Add `webTripletBuilder` to `pixelforge_studio/buildpipeline/builders_long.go` alongside the preserved `wasmLongBuilder`. Wire Studio's "Build → Web" menu item to the triplet path; demote inline-single-HTML to "Build → Self-Contained Executable → Web Single-File" submenu. WASM size reporting applies to runtime.wasm only.

**Requirements:** R25, R3 (reinterpreted), R21 + R22 (apply to runtime.wasm only).

**Dependencies:** U7 (so the runtime.wasm exists).

**Files:**
- `pixelforge_studio/exportweb/doc.go` (new)
- `pixelforge_studio/exportweb/triplet.go` (new) — `EmitTriplet(opts TripletOptions) error` that writes the four files (plus optional zip)
- `pixelforge_studio/exportweb/triplet_test.go` (new)
- `pixelforge_studio/exportweb/index_template.html` (new) — fetch-based loader
- `pixelforge_studio/buildpipeline/builders_long.go` (modify) — add `webTripletBuilder`; existing `wasmLongBuilder` preserved as inline-single-HTML opt-in
- `pixelforge_studio/buildpipeline/builders_long_test.go` (modify)
- `pixelforge_studio/build/workspace.go` (modify) — `BuildWeb()` method; existing `BuildWASM()` renamed to `BuildWebSingleFile()` or similar
- `pixelforge_studio/editor/menu.go` (modify) — File → Build → Web (default); Build → Self-Contained Executable → Web Single-File (opt-in)

**Approach:**
- `index_template.html` (new) outline (sketch, not implementation):
  - `<head>` with title + favicon (data: URL preserved from plan-009 icon_logo.go).
  - `<body>` with `<canvas id="screen">` + `<button id="start">Click to Start</button>`.
  - `<script src="wasm_exec.js"></script>` (external reference, not inlined).
  - Inline `<script>` that on button click: `const cart = await (await fetch("game.pforge")).arrayBuffer(); window.__pixelforgeCart = new Uint8Array(cart); const wasm = await WebAssembly.instantiateStreaming(fetch("pixelforge-runtime.wasm"), go.importObject); go.run(wasm.instance);`. Uses `instantiateStreaming` for smaller memory footprint vs base64-decode.
- `EmitTriplet`: writes the four files in dependency order (wasm first, cart second, wasm_exec.js third, index.html last). Validates wasm_exec.js Go-version match (the file must come from the same Go version that compiled the wasm — read from a sidecar in `playerbins/bins/js-wasm/`).
- Optional zip: `<gameName>-web.zip` includes all four files at root.
- Cart loaded via `fetch()` is async — index.html's click-to-start gate already handles browser autoplay policy; fetch fires after the click so audio context init aligns with cart availability.
- WASM size reporting: applies to `pixelforge-runtime.wasm` only (the cart is data, not engine — no threshold). Reuses plan-009 U23's `wasm_size_report.go` for the wasm file; toast shows triplet total size + per-file breakdown.
- `wasm-opt` invocation: same as plan-009 U23; applies to runtime.wasm.
- Preserves plan-009 U23's gzip output: writes `pixelforge-runtime.wasm.gz` alongside for hosts that serve pre-compressed assets.

**Patterns to follow:**
- Plan-009 U3 wasmLongBuilder — same phase-emission cadence; just different output shape.
- Plan-009 U23 wasm_size_report — reuse for the runtime.wasm portion.
- Standard Go-WASM layout: `https://github.com/golang/go/wiki/WebAssembly`.

**Test scenarios:**
- *Triplet emit produces 4 files:* `EmitTriplet`; assert all 4 files exist at expected paths; assert wasm + cart byte-equal to source inputs.
- *index.html references siblings:* Parse the HTML; assert `<script src="wasm_exec.js">` present + `fetch("game.pforge")` + `fetch("pixelforge-runtime.wasm")` strings present.
- *Triplet opens via file://:* In headless browser (chromedp — note this is plan-009 deferred, so test asserts the static structure only; full headless verification stays a follow-up): assert index.html parses + script tags resolve to sibling files.
- *Zip option:* `EmitTriplet{Zip: true}`; assert `<gameName>-web.zip` exists; unzips to the 4 files.
- *wasm_exec.js version match:* Stub a runtime.wasm built with Go 1.X; ensure `playerbins/bins/js-wasm/wasm_exec.js` (or sidecar) is the same version; assert error if mismatched.
- *Size reporting:* Build triplet from a representative fixture; toast contains `runtime.wasm: X.X MB (gzip Y.Y MB)`, `cart: Z KB`, `total: ...`.
- *wasm-opt invoked when present:* Stub `exec.LookPath`; verify Optimize called against runtime.wasm.
- *Self-contained single-file opt-in still works:* Same fixture via "Build → Self-Contained Executable → Web Single-File"; plan-009 U3 wasmLongBuilder path produces the inline-HTML; assert non-empty + base64 patterns present.
- *Covers AE2 (reinterpreted):* Consumer opens `index.html` via file:// with no network; sees splash → clicks → game plays. (Static parts asserted; runtime behavior covered by U16 headless smoke.)

**Verification:** `go test -tags=long ./pixelforge_studio/exportweb/...` + `go test -tags=long ./pixelforge_studio/buildpipeline -run TestWebTripletBuilder` passes; manual Studio build produces a folder a user can host on any static web server.

---

### Phase D — Wire four proof games to playability through Studio

### U11. Studio Play loop end-to-end against each of the four proof carts

**Goal:** Verify the in-Studio Play button — driving through `pixelforge_render.RenderTickAt` (now body-authoritative post-U1) — runs each of the four proof carts to a recognizable end-state (victory or game-over screen). Author win/lose recipes in each cart's verb sheets via the Studio Inspector (U14) — zero code-writing. Document this as the "manual smoke" gate that complements automated CI (U13).

**Requirements:** R12 (strengthened), R26.

**Dependencies:** U0, U1, U3, U4, U5a, U5b, U5c, U5d, U6, U12, U14.

**Files:**
- `pixelforge_studio/integration_test/play_loop_smoke_test.go` (new) — `//go:build long`; for each proof cart, boot via studio's preview path + advance N ticks + assert reached a win or lose end-state (NOT a hang; not a panic; the win/lose sink fires within the cart's authored verb-sheet logic)
- `docs/studio.md` (modify or new section) — "Play loop verification checklist" documenting the four-cart manual smoke

**Approach:**
- For each cart: load via `pixelforge_project.LoadFromPath`; Boot; run ticks from the existing `.trace.jsonl` (drives input); after N ticks (`trace.Meta.DurationTicks`), assert: (a) win OR lose scene transition fired; (b) no panic; (c) bus event sequence includes at least one `collision/body_collide` event (proves U3 is wired) and one `meta/win_when` OR `meta/lose_when` predicate satisfaction (proves U6).
- "Manual smoke" doc: a short checklist the maintainer runs after big render or sink changes — open Studio, File → Open Example → each game → press Play → confirm reaches win/lose end-state. The automated test is the CI gate; the doc is the human gate.

**Patterns to follow:**
- Existing `*_proof_test.go` shape from plan-009 (Asteroids/Mario/Bomberman/DK).
- `docs/solutions/scripting-runtime-design.md`.

**Test scenarios:**
- *Asteroids reaches victory:* Load asteroids_proof.pforge; play trace; assert `scene/transition` to victoryScene fired (all asteroids destroyed).
- *Mario reaches victory:* Load mario_proof.pforge; play trace; assert victory transition (hero past flag x-coordinate, or all goombas destroyed — depends on cart authoring).
- *Bomberman reaches victory:* Load bomberman_proof.pforge; play trace; assert victory transition (all breakable walls cleared).
- *Donkey Kong reaches victory:* Load donkey_kong_proof.pforge; play trace; assert victory transition (hero reached top platform).
- *No panic:* All four carts run their trace to completion without panic.
- *Each cart fires collision events:* Bus event log contains `collision/body_collide` events.
- *Each cart fires win or lose:* Bus event log contains exactly one `meta/win` or `meta/lose` event (the first-satisfaction gate from U6).
- *Snapshot mid-game preserves:* For DK, save at tick 1800 → load → continue trace → assert end-state still reached (AE4 covered by U10 plan-009 + extended here for the new collision/win-lose state).

**Verification:** `go test -tags=long ./pixelforge_studio/integration_test -run TestPlayLoopSmoke` passes for all four games.

---

### U12. Refresh four proof carts' verb sheets with collision + win/lose recipes

**Goal:** Update each of the four `*_proof.pforge` fixtures with `RecipeOnCollide`, `RecipeFireProjectile` (where applicable), `RecipeWinWhen`, `RecipeLoseWhen` entries authored via the new catalog recipes. Update each cart's `assets/` and `Scene.PoolBudgets` if needed.

**Requirements:** R8, R18.

**Dependencies:** U4, U5a, U5b, U5c, U6.

**Files:**
- `pixelforge_studio/integration_test/fixtures/asteroids_proof.pforge` (modify)
- `pixelforge_studio/integration_test/fixtures/mario_proof.pforge` (modify)
- `pixelforge_studio/integration_test/fixtures/bomberman_proof.pforge` (modify)
- `pixelforge_studio/integration_test/fixtures/donkey_kong_proof.pforge` (modify)

**Approach:**
- Per cart, add to verb sheets:
  - **Asteroids:** `RecipeOnCollide(Ship, Rock) → damage/die(Ship)`; `RecipeOnCollide(Bullet, Rock) → damage/take_damage(Rock, 1) + spawn/destroy_other(Bullet) + globals.score += 100`; `RecipeFireProjectile(spawner: Ship, archetype: Bullet, direction: facing, speed: 4, ttl: 120)`; `RecipeWinWhen(entity_count_eq Rock 0 → victoryScene)`; `RecipeLoseWhen(counter_lte player_hp 0 → gameOverScene)`. `Scene.PoolBudgets["Bullet"] = 8`.
  - **Mario:** `RecipeOnCollide(Hero, Goomba) → checks vertical velocity (or "hit from above" predicate — pick one): kill goomba OR damage hero`; `RecipeWinWhen(globals.hero_past_flag = true → victoryScene)` (flag entity sets globals on contact via existing event flow); `RecipeLoseWhen(counter_lte hero_hp 0 → gameOverScene)`.
  - **Bomberman:** `RecipeOnCollide(Blast, Breakable) → damage/take_damage(Breakable, 999) + globals.score += 50`; `RecipeOnCollide(Blast, Hero) → damage/die(Hero)`; `RecipeWinWhen(entity_count_eq Breakable 0 → victoryScene)`; `RecipeLoseWhen(counter_lte hero_hp 0 → gameOverScene)`.
  - **Donkey Kong:** `RecipeOnCollide(Hero, Barrel) → damage/die(Hero)`; `RecipeOnCollide(Hero, Goal) → globals.reached_goal = true`; `RecipeWinWhen(globals_eq reached_goal true → victoryScene)`; `RecipeLoseWhen(counter_lte hero_hp 0 → gameOverScene)`.
- Add `victoryScene` and `gameOverScene` as additional scenes in each cart's `Project.Scenes` (simple flat-color background + title text — reuse the existing menu-style rendering from `pixelforge_menus`).
- Validate schema additivity: `pixelforge_project.LoadReader` against each updated cart parses cleanly; loaders sanitize-on-load handle any deprecated fields silently.
- Carts edited by hand for this plan (no codegen). Future carts authored entirely through U14's Inspector UI.

**Patterns to follow:**
- Existing per-game verb sheets in the .pforge JSON — extend additively.
- `pixelforge_menus/registry.go:148` `game_over` menu — reference for the gameOverScene shape.

**Test scenarios:**
- *Each cart loads cleanly post-edit:* `pixelforge_project.LoadFromPath` returns valid project with no schema errors.
- *Each cart contains the expected new recipes:* Iterate `Project.Scenes[*].VerbSheets`; assert RecipeOnCollide + RecipeWinWhen + RecipeLoseWhen entries present.
- *Cart back-compat:* A pre-edit cart (e.g., legacy mario_strip_scene.pforge) loads without sanitize errors (proves schema additivity).
- *PoolBudgets respected:* Asteroids cart loads → `Scene.PoolBudgets["Bullet"] = 8` → Boot creates pool of size 8 → 9th fire returns pool-exhausted (dropped silently).
- *VictoryScene/gameOverScene exist:* Each updated cart has both scenes in `Project.Scenes` list.

**Verification:** `go test -tags=long ./pixelforge_studio/integration_test -run TestProofCartLoad` passes for all four; manual Studio Open Example confirms each loads cleanly.

---

### U13. Refresh `*_proof_test.go` baselines with new render path + new bus events

**Goal:** Re-record the four proof carts' `*_proof.baseline.json` files against the new render path (body-authoritative post-U1) and new bus event sequence (with collision + projectile + win/lose events from U3 + U5 + U6). Update `*_proof_test.go` files to assert against the refreshed baselines. Preserves plan-009's CI gating + cross-CPU determinism contract.

**Requirements:** R8, R9, R12.

**Dependencies:** U0, U1, U3, U5a, U5b, U5c, U5d, U6, U12.

**Files:**
- `pixelforge_studio/integration_test/fixtures/asteroids_proof.baseline.json` (regenerate)
- `pixelforge_studio/integration_test/fixtures/mario_proof.baseline.json` (regenerate)
- `pixelforge_studio/integration_test/fixtures/bomberman_proof.baseline.json` (regenerate)
- `pixelforge_studio/integration_test/fixtures/donkey_kong_proof.baseline.json` (regenerate)
- `pixelforge_studio/integration_test/asteroids_proof_test.go` (modify) — `//go:build long`; add assertions on new collision/win events
- `pixelforge_studio/integration_test/mario_proof_test.go` (modify)
- `pixelforge_studio/integration_test/bomberman_proof_test.go` (modify)
- `pixelforge_studio/integration_test/donkey_kong_proof_test.go` (modify)

**Approach:**
- Regenerate via `pixelforge_replay.Recorder` + `Replayer.Run`: load cart + trace → run trace → capture per-checkpoint pixel hashes + full bus event sequence → write baseline JSON.
- Update test assertions to check the new event types (`collision/body_collide`, `combat/fire_projectile`, `meta/win`, `meta/lose`) appear at expected ticks.
- Cross-CPU determinism: regenerate on the CI matrix's amd64 leg; verify byte-equal on arm64 per plan-009 U6's determinism strategy outcome (or fall back to tolerance per that decision).

**Patterns to follow:**
- Plan-009 U11 + U14 + U16 + U18 — same regeneration pattern.

**Test scenarios:**
- *Each baseline regenerates deterministically:* Run `Replayer.Run` twice; output baselines byte-equal.
- *Cross-CPU determinism preserved:* CI matrix amd64 + arm64 produce same baselines (or within plan-009 U6's tolerance strategy).
- *New events appear in sequence:* Baseline event sequences include `collision/body_collide`, `combat/fire_projectile` (where applicable), `meta/win` or `meta/lose`.
- *Regression detection still works:* Introduce a known one-pixel bug in render path; rerun; assert FAIL at expected checkpoint (AE5 negative test).

**Verification:** `go test -tags=long ./pixelforge_studio/integration_test -run '/_proof$'` passes for all four games.

---

### U14. Studio Inspector UI for collision + projectile + win/lose authoring

**Goal:** Add Inspector panel sections in the Studio verb-sheet authoring surface that expose the new recipes (RecipeOnCollide, RecipeFireProjectile, RecipeWinWhen, RecipeLoseWhen) entirely through GUI dropdowns + numeric inputs + archetype pickers. Zero code-writing — every parameter that affects gameplay is a clicky widget. Also exposes `Scene.PoolBudgets` as a per-archetype budget editor.

**Requirements:** R26.

**Dependencies:** U4, U5a, U5b, U5c, U6.

**Files:**
- `pixelforge_studio/editor/inspector_collision.go` (new) — ImGui panel for RecipeOnCollide; archetype-pair dropdowns + action-chain editor
- `pixelforge_studio/editor/inspector_projectile.go` (new) — RecipeFireProjectile panel: spawner archetype, projectile archetype, direction source, speed, TTL
- `pixelforge_studio/editor/inspector_winlose.go` (new) — RecipeWinWhen + RecipeLoseWhen panels: predicate kind dropdown + value field + transition_to scene picker
- `pixelforge_studio/editor/inspector_pool_budgets.go` (new) — Scene.PoolBudgets editor (per-archetype int input)
- `pixelforge_studio/editor/inspector_collision_test.go` (new) — model-level tests for state mutations (UI test smoke is best-effort, given ImGui)
- `pixelforge_studio/editor/inspector_test.go` (modify) — integration assertions that new panels appear in the Inspector when the right entity / scene is selected

**Approach:**
- Inspector is ImGui-only (preserves the plan-009 invariant).
- Collision panel: when a Scene is selected, show its `OnCollide` recipe list. Each row: archetype-A dropdown (populated from `Project.Archetypes` + `"*"` wildcard) + archetype-B dropdown + collapsible action-chain editor (action picker dropdown — `damage/take_damage` + `spawn/destroy_other` + `globals/inc` etc. — with action-specific param widgets).
- Projectile panel: when a Scene is selected, show its `FireProjectile` recipes. Spawner archetype + projectile archetype + direction (dropdown: "facing" | "constant" + numeric input) + speed numeric + TTL numeric.
- Win/Lose panel: when a Scene is selected, show its WinLose recipes. Predicate kind dropdown + param widgets (varies by kind: counter_geq shows path + value; entity_count_eq shows archetype dropdown + value; globals_lte shows path + value) + transition_to scene picker (dropdown of `Project.Scenes` names).
- Pool budgets panel: simple key-value editor mapping archetype names to integer budgets. Validates budget ≥ 1 + warns when sum exceeds a heuristic threshold (~1000) suggesting redesign.
- All UI mutations write through the existing `pixelforge_project` schema editing seam (no string-concat; structural mutations only — preserves the plan-008 invariant).
- "Validate" button at the bottom of each panel: runs the cart-load sanitize path against the in-memory project; surfaces errors inline.
- Reuse existing archetype picker + scene picker components from plan-009's Inspector (which already exposes Spawn / Damage recipes).

**Patterns to follow:**
- Existing Inspector panels in `pixelforge_studio/editor/` (Spawn, Damage, Motion — plan-009 U7 + U22).
- `docs/canvas-vs-native-chrome-split.md` — ImGui-only chrome rule.
- `pixelforge_studio/scripting/` view-as-go.tmpl shape for the action-chain editor (mirror its structure, but it's edit-mode here, not read-mode).

**Test scenarios:**
- *Collision panel shows for selected scene:* Open Inspector on a Scene; assert OnCollide section visible.
- *Add OnCollide recipe via UI:* Click "Add OnCollide"; pick archetype-A=Ship, archetype-B=Rock, action=damage/die; assert `Scene.OnCollide` slice has the new recipe with correct fields.
- *Edit existing recipe:* Modify archetype-A dropdown; assert mutation reflected in project model.
- *Delete recipe:* Click trash icon; assert recipe removed from project model.
- *Projectile panel:* Add RecipeFireProjectile via UI; assert all params bind correctly.
- *Win panel — counter_geq:* Predicate kind = counter_geq; path = "score"; value = 1000; transition_to = "victoryScene" (picked from dropdown of Project.Scenes); assert mutation recorded.
- *Win panel — entity_count_eq:* Kind switch reconfigures form to archetype + value fields; assert correct schema serialization.
- *Pool budgets editor:* Add "Bullet" → 8; assert `Scene.PoolBudgets["Bullet"] = 8`.
- *Validate surfaces errors:* Introduce invalid archetype name; click Validate; assert error displayed.
- *No-code authoring smoke:* From scratch, author a "ship destroys rocks → wins" recipe entirely through the UI without typing any code; save; load; play; assert behavior matches.

**Verification:** `go test ./pixelforge_studio/editor/...` passes (model-level mutations + serialization); manual Studio walk-through authors a full Asteroids verb-sheet without writing code — and the resulting cart plays end-to-end via U11's Play loop.

---

### Phase E — Release-process polish

### U15. `make runtimebins` + bins committed (or release-attached)

**Goal:** Rename plan-009's `make playerbins` target to `make runtimebins` (it now produces `pixelforge-runtime` binaries, not `pixelforge-player`). Run it locally + commit results under `pixelforge_studio/playerbins/bins/<os>-<arch>/pixelforge-runtime[.exe|.wasm]` so the studio installer ships with runtime bins on day one. Decide commit vs release-artifact path per plan-009 U3's deferred decision (LFS or release-attached); document in `pixelforge_studio/playerbins/README.md`.

**Requirements:** R23 (provisioning side), R3 (web triplet needs runtime.wasm).

**Dependencies:** U7.

**Files:**
- `Makefile` (modify) — rename target + update binary names; preserve `make playerbins` as an alias for backward compat
- `pixelforge_studio/playerbins/bins/linux-amd64/pixelforge-runtime` (binary; not in source — produced by make)
- `pixelforge_studio/playerbins/bins/darwin-amd64/pixelforge-runtime` (binary)
- `pixelforge_studio/playerbins/bins/darwin-arm64/pixelforge-runtime` (binary)
- `pixelforge_studio/playerbins/bins/windows-amd64/pixelforge-runtime.exe` (binary)
- `pixelforge_studio/playerbins/bins/js-wasm/pixelforge-runtime.wasm` (binary)
- `pixelforge_studio/playerbins/embed.go` (modify) — `PlayerBinaryFor` renamed to `RuntimeBinaryFor`; keeps the old name as a deprecated alias
- `pixelforge_studio/playerbins/README.md` (new) — explains the bins layout + LFS-vs-release-artifact decision rationale

**Approach:**
- `make runtimebins` cross-compiles via the existing matrix; outputs go to existing layout under `playerbins/bins/`.
- LFS vs release-attached: plan-009 U3's decision was deferred; implementer at U15 picks based on team's git-LFS posture. Default recommendation: LFS for any `bins/` file > 5 MB (the embedded path requires the bytes to be on disk; LFS keeps git history small while preserving the file).
- Rename: `PlayerBinaryFor → RuntimeBinaryFor`; old name kept as a deprecated alias for one release cycle then removed in v3.
- WASM: `pixelforge-runtime.wasm` produced via `GOOS=js GOARCH=wasm go build` (NOT through GoReleaser — WASM doesn't fit GoReleaser's installer-packaging model).

**Patterns to follow:**
- Plan-009 U3 cache + integrity check pattern.
- Plan-009 U3 Makefile playerbins target — preserve the existing 5-target enumeration.

**Test scenarios:**
- *make runtimebins produces 5 binaries:* All five files exist under expected paths post-make.
- *Each binary is non-empty:* size > 1 MB (sanity).
- *Each native binary boots:* For host OS, `./pixelforge_studio/playerbins/bins/<host>-<arch>/pixelforge-runtime --version` exits 0.
- *WASM compiles cleanly:* `pixelforge-runtime.wasm` is a valid WebAssembly module (parse via wasm-tools or similar).
- *Embed extraction works:* `playerbins.RuntimeBinaryFor("linux", "amd64")` returns non-empty bytes matching the on-disk file.

**Verification:** `make runtimebins` succeeds; `playerbins/bins/<os>-<arch>/` populated for all 5 targets; `RuntimeBinaryFor` extracts each binary to a temp path that boots.

---

### U16. CI extends long-tag workflow for runtime smoke + collision/win-lose tests

**Goal:** Extend `.github/workflows/long.yml` (plan-009 U25) to: (a) per-OS runtime smoke (build runtime, smoke-tick against a fixture cart, assert exit 0); (b) include the new collision/projectile/win-lose long-tag tests (U3, U5a–U5d, U6); (c) include the new play-loop smoke (U11) and triplet-emit (U10a) tests. CI gates on all of these.

**Requirements:** R9, R13.

**Dependencies:** U7, U8a, U8b, U8c, U8d, U10a, U10b, U11, U13.

**Files:**
- `.github/workflows/long.yml` (modify) — add jobs: `runtime-smoke-{linux,darwin,windows}`, `web-triplet-build`, expand `long-tests` matrix
- `.github/workflows/long.yml` ensures `make runtimebins` runs in the smoke jobs (so the embed has fresh bins)
- `.github/workflows/release.yml` (new — optional) — tag-triggered GoReleaser publish (the actual GitHub Release publish is out of v2 automation; this workflow is the seam if/when the team enables it)

**Approach:**
- New CI jobs:
  - `runtime-smoke-linux`: `make runtimebins` → run `./pixelforge_studio/playerbins/bins/linux-amd64/pixelforge-runtime asteroids_proof.pforge --smoke-tick 60`; assert exit 0.
  - `runtime-smoke-darwin` + `runtime-smoke-windows`: same per OS.
  - `web-triplet-build`: run the U10 triplet emitter against asteroids_proof.pforge; assert 4 files emitted + structure validates.
  - Long-tests matrix gains the four new `_proof_test.go` post-U13 refresh + `TestPlayLoopSmoke` + `TestRenderBodies`.
- Determinism gate preserved: cross-CPU pixel-hash + bus-event-parity per plan-009 U6's strategy.
- Optional `release.yml`: triggered on tag push; runs `goreleaser release` (no `--snapshot`). Marked as opt-in (requires GitHub release secrets that aren't part of v2 scope).

**Patterns to follow:**
- Plan-009 U25 `long.yml` shape.
- GoReleaser GitHub Actions integration (`https://goreleaser.com/ci/actions/`) for the optional release.yml.

**Test scenarios:**
- *Workflow YAML parses:* `act --dry-run` or equivalent passes.
- *Runtime smoke job exit 0 (per OS):* On a CI run, each runtime-smoke-X job succeeds.
- *Web-triplet job exit 0:* Job produces 4 files + asserts structure.
- *Long-tests matrix covers new scenarios:* Job list includes the four refreshed `*_proof_test.go` + `TestPlayLoopSmoke` + collision/projectile/win-lose unit suites.
- *Pixel-hash regression detection (AE5):* Introduce one-pixel-off bug; CI fails at the affected game's checkpoint.
- *Optional release.yml dry-run:* `goreleaser release --snapshot` succeeds without trying to push.

**Verification:** Workflow runs cleanly on a synthetic PR; matrix accurately reflects test outcomes; per-OS runtime smoke jobs go green.

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `pixelforge.Draw()` removal breaks legacy demos (snake, pacman, piano) | Low (legacy demos use their own entry points, not capsuleruntime) | Medium | U1 keeps the global available — only the capsuleruntime render path stops calling it. Legacy demos unaffected. |
| Drawable Adapter introduces visual drift vs the existing global Draw path | Medium | High (would invalidate plan-009 baselines) | U13 regenerates baselines explicitly; the drift IS the point — old path drew at authored positions, new path draws at body positions. Cross-CPU determinism gate stays the regression detector. |
| Cross-CPU determinism breaks under the new collision sink (resolv internal float64) | Medium | High (CI strategy collapses if byte-equal cross-CPU fails) | Plan-009 U6's determinism probe already characterized this; we inherit whichever strategy that landed on (byte-equal or tolerance). The new collision sink uses resolv's same SAT path; risk is no worse than what plan-009 already accepted. |
| Pool exhaustion drops projectiles silently — player can't tell why their bullet didn't fire | Low | Medium | `combat/projectile_dropped` notification event on the bus; U14's Inspector exposes pool budgets editor with sum-warning at >1000; documented "what to do when projectiles disappear" in `docs/studio.md`. |
| Fast projectile tunneling (velocity > cell_size) | Medium | Medium | Velocity cap in `combat/fire_projectile` recipe; documented in collision_sink.go. Continuous-collision sweep deliberately out of scope (resolv limitation). |
| File-association registration fails silently on some Linux desktops | Medium | Medium | `update-mime-database` post-install script + clear log output; user-doc fallback ("if double-click doesn't work, run `pixelforge-runtime path/to/game.pforge`"). |
| macOS Gatekeeper blocks ad-hoc-signed `.pkg` on first launch | High (expected behavior for unsigned-but-adhoc) | Medium (UX papercut, not a blocker) | Install instructions document right-click → Open → Open Anyway; notarization deferred to v3 (plan-009 scope decision). |
| Win/lose recipe ambiguity (which scene-transition wins when multiple recipes fire) | Low | Low | Hard-coded precedence: lose > win; first-satisfaction; documented + tested explicitly in U6. |
| Counter-polling tick cost at large entity counts (1000+) | Low (arcade scale is 50-200) | Low | spawn_sink writes counters to `rt.Globals` directly (O(1) lookup at predicate eval, not O(n) scene iteration). Profiling step in U13 confirms tick budget. |
| Predicate language sprawl (users want `score % 100 == 0` etc.) | Medium | Low | v2 ships only 4 predicate kinds (counter_geq, counter_lte, entity_count_eq, globals_lte). Arbitrary expressions explicitly deferred to v3. |
| Studio Inspector UI for new recipes is too discoverable / clutters the panel | Medium | Low | Collapse panels by default; only expand when the relevant recipe type exists on the current scene; "Add" button at top of each section. |
| Inline-single-file WASM path bit-rots without active use | Medium | Low | Preserved-only-as-opt-in stays in the codebase; if it bit-rots, demote to "legacy" doc warning rather than maintain unused complexity. |
| GoReleaser config drift across OS-specific quirks | High (typical GoReleaser pain) | Medium | Snapshot job in CI (U16); manual macOS test on a real Mac before any release. Documented troubleshooting in `pixelforge_studio/installer/README.md`. |
| Plan-009's `pixelforge-player` binary becomes dead code once "Build → Cart" is default | Low (still consumed by opt-in self-contained path) | Low | Keep it. The opt-in self-contained mode is a real user-visible feature. |
| Determinism breaks under EntityPool snapshot round-trip | Medium | High | U5 explicit test: snapshot mid-firing → load → assert per-slot state byte-equal. The pool's deterministic slot selection (next-free-index, sorted active iteration) is load-bearing for this. |
| Cart filename collision in OS file association (user has another `.pforge` extension app) | Low | Low | Standard MIME-handling behavior — OS picks one, user can change default. Document. |

---

## Test Strategy

- **Unit tests:** every new package (`pixelforge_pool` extension, `pixelforge_studio/exportweb`, `pixelforge_studio/capsuleruntime` new sinks, `pixelforge_studio/editor` new inspector panels) gets a `_test.go` covering its public API. Run under `go test ./...` (no tag).
- **Long-tag integration tests:** the four refreshed `*_proof_test.go` files + new `play_loop_smoke_test.go` + new `render_bodies_test.go` + new `build_pipeline_cart_test.go` + new `web_triplet_test.go` gate via `//go:build long`. They Run via `make ci-long` or equivalent.
- **Cross-OS runtime smoke:** new CI jobs (U16) per OS smoke-tick the runtime against a fixture cart.
- **Cross-CPU determinism:** preserved from plan-009 U6; the long-tag matrix runs on `ubuntu-latest` + `macos-latest`.
- **WASM build test:** triplet emission test (U10) + the optional opt-in single-HTML test (preserved from plan-009 U23).
- **Race tests:** `go test -race ./...` clean for every modified package.
- **Snapshot round-trip tests:** explicitly exercise pool state + win/lose `fired` flag + collision pair-cache (the last should be ephemeral and not serialize — explicit test).
- **Schema additivity tests:** load a pre-edit `.pforge` (legacy) against the post-edit loader; assert sanitize-on-load handles missing PoolBudgets / OnCollide / WinLose gracefully.
- **Manual smoke checklist:** documented in `docs/studio.md` per U11 — open Studio, File → Open Example → each game → press Play → verify reaches end-state. Run before any release.

---

## Dependencies & Assumptions

- **External libraries:**
  - `github.com/goreleaser/goreleaser` (MIT, dev-tool, not a runtime dep) — release config tool; invoked via Makefile.
  - `github.com/goreleaser/nfpm/v2` (MIT, dev-tool via goreleaser config) — `.deb` + `.rpm` packagers.
  - `golang.org/x/term` (BSD, std-adjacent) — TTY detection in `pixelforge-runtime` main.
  - All previously-adopted libs from plan-009 (`solarlune/resolv`, `jfreymuth/oggvorbis`, `hajimehoshi/go-mp3`) preserved.
- **Tooling on release host:**
  - GoReleaser binary installed (single binary; `https://goreleaser.com/install/`).
  - WiX toolchain on Windows release leg (or via cross-platform `wixtools` Docker image) for `.msi` packaging.
  - `wasm-opt` (optional, plan-009 U23) for the triplet's runtime.wasm.
- **Pre-built runtime binaries via `make runtimebins` populate `playerbins/bins/`** so day-one users with no Go toolchain installed can use Studio's Build → Cart + Build → Web. Studio embeds these via `//go:embed all:bins` (plan-009 U3 mechanism).
- **Go 1.24.2 toolchain** required for studio developers only (the long-tag build path + `make runtimebins`). End users authoring games via Studio do not need Go installed; end users consuming games install `pixelforge-runtime` via the per-OS installer (no Go needed there either).
- **Ebitengine 2.9.9** stays the engine version; no upgrade in this plan.
- **A working GitHub Release URL** for the asset-library manifest (per plan-009 U20) is preserved; this plan does not change manifest publication.
- **macOS notarization** OUT of scope; ad-hoc codesign only (plan-009 decision preserved).
- **Windows code-signing (Authenticode)** OUT of scope.
- **Linux: tested against Ubuntu 22.04 / Debian 12 / Fedora 39 (representative MIME-handling matrix)**. Arch / Alpine / NixOS not gated by CI; documented as "best-effort" in `pixelforge_studio/installer/README.md`.

---

## Scope Boundaries

### Deferred for later

- Cryptographic cart signing + manifest signing (still v3 per plan-009)
- Notarization on macOS
- Authenticode signing on Windows
- Linux AppImage code-signing
- Headless browser smoke test for the WASM triplet (chromedp / playwright)
- Linux file-association testing on Arch / Alpine / NixOS
- Continuous-collision sweep for fast projectiles (resolv-level work)
- Predicate sprawl beyond the 4 kinds (no arbitrary expressions in v2)
- Additional reference games beyond the existing four (Pac-Man, Frogger, Space Invaders, Tetris — still v3)
- Variable-jump-height (plan-009 deferred; still deferred here)

### Outside this product's identity

- Cloud build farm / hosted Studio
- Author-by-play (record gameplay → infer recipes)
- LLM-generated cart content
- Browser-first Studio
- Editor-embedded-in-every-shipped-game (Studio stays separate from runtime)
- Phone / tablet / VR targets
- Multiplayer / networking
- Cloud save sync
- Steam / itch.io publishing integrations
- iOS / Android targets

### Deferred to Follow-Up Work

- Tag-triggered GoReleaser publish workflow (`.github/workflows/release.yml`) — scaffold lands in U16 but secrets + actual publish gated to a follow-up PR after manual macOS bring-up.
- Brotli compression of triplet runtime.wasm (gzip-only in U10; brotli a follow-up).
- Inspector "Validate" button surfacing exhaustive predicate-evaluation dry-runs (basic validate in U14; deep validation a follow-up).
- Empirical re-tune of runtime.wasm size threshold after the first real triplet build measurement.
- Demote/remove inline-single-HTML WASM path if it bit-rots after a release cycle of disuse.

---

## Deferred to Implementation

- **Pool slot reuse policy** (LIFO vs FIFO) in U5 — both are deterministic; pick one and document in the package doc comment. Snapshot round-trip cost is the same either way.
- **Move-pattern direction-state storage** in U2 — store on `rt.Globals` (cross-cutting) or on a new `Runtime.MovePatternState` map (clean separation). Both round-trip in snapshots; pick at implementation time.
- **Collision pair sort key** in U3 — sort by `(min(a_id, b_id), max(a_id, b_id))` or by entity creation order. The first guarantees stable order across reorderings; pick + document.
- **Win/lose recipe location** in U6 — `Project.WinLose []WinLoseRecipe` or `Scene.WinLose []WinLoseRecipe`. The plan recommends per-Scene for flexibility; implementer confirms at U6.
- **GoReleaser nFPM scripts** for `.deb` / `.rpm` post-install — exact `update-mime-database` invocation path varies by distro; implementer measures on Ubuntu 22.04 + Fedora 39 at U8.
- **WiX template specifics** for `.msi` — implementer can use a minimal heat-generated WXS or a hand-written one; both are valid.
- **Triplet zip structure** — files at zip root, or under `<gameName>/` subdir? Pick at U10 based on user-test (some users want to unzip-in-place; others want a self-contained subdir).
- **Inspector panel collapse state persistence** in U14 — store in user prefs or always-expand-when-recipe-exists. Either is fine; pick based on Inspector-load-time cost.

---

## Success Metrics

- A user with no Go installed, no PixelForge installation, and no coding experience can install `pixelforge-runtime` via their OS package manager, then double-click a `.pforge` file authored by someone else and play the game.
- A user with PixelForge Studio installed can author Asteroids (or Mario / Bomberman / DK equivalent) entirely through the Studio GUI — zero text/code editing — and produce a `.pforge` file that plays end-to-end via the installed runtime.
- The four proof carts, post-U12 verb-sheet refresh, run to a recognizable end-state (win or lose scene transition) in `TestPlayLoopSmoke` for each game.
- Studio's "Build → Cart" produces a `.pforge` file under 100 KB for each of the four proof carts.
- Studio's "Build → Web" produces a folder where `index.html` opens via `file://` with no network and shows splash → click → game.
- The opt-in "Build → Self-Contained Executable" still produces a working ~20 MB single binary (plan-009 behavior preserved).
- `pixelforge_render/rendertick.go` no longer calls `pixelforge.Draw()`; `grep "pixelforge\.Draw()" pixelforge_render/` returns zero matches.
- `motion/move_pattern` no longer appears in any `grep debugDrop pixelforge_studio/capsuleruntime/motion_sink.go` output.
- All long-tag CI tests stay green; pixel-hash cross-CPU baselines stay equal (or within the plan-009 U6 tolerance).
- `make runtimebins` completes successfully on each of linux/darwin/windows hosts; `playerbins/bins/<os>-<arch>/` populated.
- `make goreleaser-snapshot` produces `.deb` + `.rpm` + `.AppImage` + `.pkg` + `.msi` artifacts under `dist/`.

---

## Phased Delivery Summary

| Phase | Units | Demoable outcome |
|---|---|---|
| Phase A (rendering integration) | U0, U1 | Sprite-source pipeline wired in capsuleruntime; Studio Play renders body-driven motion with visible sprites; the static-placeholder bug is gone |
| Phase B (gameplay verbs) | U2, U3, U4, U5a, U5b, U5c, U5d, U6 | Collision events fire; projectiles spawn from a pool (with snapshot round-trip); win/lose recipes evaluate (lose>win precedence); `move_pattern` patrols/sines/waypoints |
| Phase C (system runtime) | U7, U8a, U8b, U8c, U8d, U9, U10a, U10b | `pixelforge-runtime` binary installs per OS via `.deb`/`.pkg`/`.msi`; Studio's default Build → Cart produces tiny `.pforge`; Build → Web produces standard triplet |
| Phase D (four-game playability) | U11, U12, U13, U14 | Each of the four proof carts has collision + win/lose verb sheets authored via Studio Inspector; pressing Play runs each to a recognizable end-state; baselines refreshed |
| Phase E (release polish) | U15, U16 | `make runtimebins` populates the embed; CI gates runtime smoke + collision/win-lose tests per-OS; `make goreleaser-snapshot` produces installable artifacts |

Total: 24 implementation units across 5 phases (post-tightening edit; original plan had 16, expanded for atomicity at U0/U5/U8/U10 after Phase 1 ce-work investigation surfaced wider surface than initially scoped).

Phase A and Phase B can mostly land in parallel — they share no files. Phase C is independent of A+B (different surface area). Phase D depends on A + B + C. Phase E depends on D.

Within Phase A: U0 must land before U1 (U1's verification depends on visible pixels). Within Phase B: U3, U5a, U5b, U6, U2 can land in parallel (different sinks, different files); U4 depends on U3; U5c depends on U5a + U5b + U3; U5d depends on U5a + U5b + U5c. Within Phase C: U7 must land first; U8a before U8b/U8c/U8d (all three per-OS installers depend on the GoReleaser skeleton); U10a before U10b.

---

## Documentation Plan

- `pixelforge_render/drawables.go` doc comment — describes the chokepoint contract + body-vs-authored fallback rules + camera-offset application.
- `pixelforge_studio/capsuleruntime/collision_sink.go` doc comment — describes the pair-enumeration + de-dup contract + tunneling note.
- `pixelforge_pool/entity_pool.go` doc comment — describes the deterministic slot-selection contract + snapshot round-trip invariant.
- `pixelforge_studio/capsuleruntime/win_lose_sink.go` doc comment — describes the first-satisfaction + lose>win precedence + reset-on-scene-transition rules.
- `cmd/pixelforge-runtime/main.go` doc comment + a `pixelforge-runtime --help` output that explains the path / stdin / file-association load order.
- `pixelforge_studio/installer/README.md` (new) — per-OS file-association manifest layout + GoReleaser hook order + troubleshooting.
- `pixelforge_studio/playerbins/README.md` (new) — bins layout + LFS vs release-artifact rationale + `make runtimebins` invocation.
- `docs/studio.md` — extend with "Play loop verification checklist" + "Build menu items explained (Cart vs Web vs Self-Contained Executable)".
- `docs/verb-catalog.md` — regenerated by plan-009 U24 mechanism; new recipes appear automatically.
- `docs/reference-games-capability-matrix.md` — regenerated by plan-009 U25 mechanism; new tests appear automatically.
- `docs/install-linux.md` + `docs/install-macos.md` — extend with `pixelforge-runtime` install instructions and `.pforge` double-click flow.
- Each new sink file gets a leading doc comment naming the verb topics it owns + the test scenarios that cover it.

---

## Operational / Rollout Notes

- **Backward compatibility:** Existing `.pforge` carts (from plan-009 and earlier) load unchanged via the sanitize-on-load path. Plan-009 cart-append behavior is preserved as the opt-in "Build → Self-Contained Executable" menu item. Plan-009 inline-single-HTML WASM behavior is preserved as the opt-in "Web Single-File" menu item.
- **Migration for end users:** Users with the old cart-append `.exe` games keep their working binaries. New games shipped post-v2 are `.pforge` files that need `pixelforge-runtime` installed once.
- **Migration for game authors:** Existing projects open unchanged. New collision / projectile / win-lose recipes are additive — no project needs updating to keep working; projects that want the new gameplay surface use the Inspector panels in U14.
- **Rollback:** If the system-runtime pivot goes sideways, the default Studio Build menu can flip back to "Build → Self-Contained Executable" as the primary option — the plan-009 path is preserved, not deleted. One line change in `pixelforge_studio/editor/menu.go`.
- **CI cadence:** `.github/workflows/long.yml` runs per push to main + every PR; long-tag suite must stay green or merge is blocked. Per-OS runtime smoke jobs gate independently.
- **Installer publication:** `make goreleaser-snapshot` runs locally for testing; tagged release triggers `goreleaser release` (this plan delivers the config + snapshot path; actual publication is a maintainer-side operation per `pixelforge_studio/installer/README.md`).
- **`.pforge` file-association on user machines** is set at installer time; uninstalling `pixelforge-runtime` should de-register (handled by nFPM / pkg / WiX uninstall hooks).
- **No git operations** during execution (per user durable memory + plan-009 invariant).

---

## Future Considerations

- v3: Cart signing (Ed25519 + manifest signing) closes the supply-chain gap plan-009 documented; `pixelforge-runtime` verifies signatures before booting cart content.
- v3: Notarization on darwin + Authenticode on windows for the runtime installer.
- v3: Pac-Man / Frogger / Space Invaders / Tetris as additional proof games — the substrate (collision, pool, win/lose) added here supports them with zero new verbs.
- v3: Behavior trees as a successor to the verb-recipe catalog once recipe count exceeds ~100 (per ideation rejection row 30).
- v3: Continuous-collision sweep for fast projectiles (resolv-level contribution).
- v3: Browser-side Studio (or at minimum, browser-side cart authoring viewer) once the triplet stabilizes.
- v3: `pixelforge-runtime --headless` mode for AI-driven cart playtesting (replay traces server-side at high speed).
- The shared `RenderTickAt` seam stays the integration chokepoint — future mobile, GPU-direct, or HDR rendering paths drop in here without breaking the preview-vs-shipped invariant.
- The verb-recipe catalog continues as single source of truth — the Inspector dropdowns auto-update from `verb_recipes.go` registration; no parallel data structure.
