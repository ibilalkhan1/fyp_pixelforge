---
date: 2026-05-19
topic: pixelforge-v2-gap-closers
focus: solutions to plan-009's three v2-gap blockers (rendering / binary distribution / gameplay verbs); external libs/repos OK if they don't cause cross-codebase problems
mode: repo-grounded
---

# Ideation: Closing the Three V2-Gap Blockers After Plan-009

## Grounding Context

**Codebase context:** Go 1.24.2 + Ebitengine 2.9.9. Plan-009 (U1–U25 + pre-flight) shipped the universal-player + cart-append architecture, four reference-game proofs as CI fixtures, ImGui studio chrome, runtime sinks for all 11 verb categories, snapshot save/load on native + WASM, and Phase 5 ship-loop UX (ingest dispatcher, starter pack, File menus, WASM size reporting, verb-catalog regen, CI matrix). All 25 units + pre-flight green; 61 packages pass; 5 long-tag integration suites pass (`integration_test`, `asteroids_proof`, `mario_proof`, `bomberman_proof`, `donkey_kong_proof`).

**Notable patterns:** Cycle-break injection (interface in inner package, impl in outer); schema additivity (`omitempty` + sanitize-on-load); `CGO_ENABLED=0` for WASM; ImGui-only studio chrome (no widget-framework fallback); `pixelforge_loop.VerbsBus()` typed pub/sub; build-tag platform split (`//go:build js|!js`, `//go:build long|!long`); cart-envelope (20-byte footer with magic + length + CRC32).

**Three blockers between v2 substrate and v2 user-shippable** (the subject of this ideation):

- **A1 Rendering integration** — `pixelforge_render/rendertick.go` doesn't read `rt.Bodies[entityID]` when blitting sprites; entities render at authored `Scene.Entity.Position` only. `rt.CameraOffsetY` is computed in `updateCameraOffset` but not applied to blits. Result: physics moves bodies, but visuals don't update — pressing Play shows static placeholders.
- **A2 Binary distribution** — `pixelforge_studio/playerbins/bins/` has only `test-fake/pixelforge-player`. `make playerbins` exists (cross-compiles to linux-amd64, darwin-amd64, darwin-arm64, windows-amd64, js-wasm) but hasn't been run + committed. No-Go users can't Build → Host (fallback is `go build` requiring Go).
- **A3 Gameplay verbs** — Body-vs-body AABB collision missing (ship-rock, hero-goomba, barrel-hero); projectile firing missing (Asteroids bullets, Mario fireballs); win/lose conditions missing; scene-transition recipes missing; `motion/move_pattern` still routes to `debugDrop`. Studio user can't express "kill enemies, win level" gameplay.

**External grounding:** `yohamta/donburi` (MIT, ~369 stars, active May 2026, Ebitengine-native ECS with transform+sprite components built-in); `goreleaser/goreleaser` (MIT, ~14k stars, single config → all 5 OS/arch targets); `solarlune/resolv` v0.8.1 (already adopted — `Shape.IntersectionTest` / `CollisionTest` / `Shape.Tags` cover entity-vs-entity); PICO-8 bullet-pool pattern (fixed-size array, alive bool, no GC); GDevelop "all enemies destroyed → next scene" via counter polling; Löve2D cat-fuse pattern; GitHub Release 2GiB asset cap (~25MB × 5 targets = comfortable fit).

**Past learnings (`docs/solutions/`):** `always-on-game-embedding.md` ("one render path" rule); `ring-buffer-snapshot-store.md` (render-time overrides are transient lookups, NOT mutations of `Scene.Entity.Position`); `editor-pforge-schema-shape.md` (single-binary convention via embed); `canvas-vs-native-chrome-split.md` (parallel paths during migration); `scripting-runtime-design.md` (`catalog.RegisterStep/Action` pattern; `pievent.Target` for typed events).

**User constraint (verbatim):** "External libraries / repos OK if they don't cause cross-codebase problems but if it causes more cross-codebase problems then leave it."

## Topic Axes

- **A1. Rendering integration** — wire `rt.Bodies` + `rt.CameraOffsetY` into per-entity sprite draw in `pixelforge_render/rendertick.go`
- **A2. Binary distribution** — populate `pixelforge_studio/playerbins/bins/<os>-<arch>/` so no-Go users can Build → Host
- **A3. Gameplay verbs** — body-vs-body collision, projectile firing, win/lose, scene transitions, patrol

## Ranked Ideas

### 1. Drawable Adapter — body-authoritative entity-draw chokepoint

**Description:** Add `pixelforge_render/drawables.go` exposing `DrawEntities(screen, rt)` that iterates `rt.CurrentScene.Entities`, joins each to `rt.Bodies[e.ID]` and `rt.VisualEffects[e.ID]`, and calls a single `drawOne(screen, e, body, effect, cameraOffsetY)` per entity. Replace the legacy global `pixelforge.Draw()` call inside `rendertick.go`'s `advanceAndRender` with this iteration. When no body exists for an entity, fall back to authored `Scene.Entity.Position`. Camera offset (`rt.CameraOffsetY`) is subtracted at blit time.

**Axis:** A1

**Basis:** `direct:` `pixelforge_render/rendertick.go:228` blits via the global `pixelforge.Draw()` without consulting `rt.Bodies`; the comment at line 230 (`_ = rt`) explicitly admits "mutation happens indirectly via the capsule subscribers." Meanwhile `runtime.go:108–115` documents `Bodies` as "where motion sinks mutate in place." Two systems, no consumer for the live one.

**Rationale:** Single chokepoint, compounding payoff. Once `drawOne` is the only entity-blit path: visual flash (`VisualEffect.FlashColor` — set by U7's visualSink, currently unread), sprite-swap (`VisualEffect.SpriteSwap`), the `Hidden` flag, damage-flash, debug AABB overlays, hitbox heatmaps, replay-scrub timeline rendering, mini-map projections, multi-camera split-screen — all drop into one switch statement instead of being plumbed through `pixelforge.Sprite()` globals. The four `VisualEffect` fields U7's sink already writes but no one reads close their loop on day one. Highest leverage line of code in the v2-gap-closing surface.

**Downsides:** Touches the render hot path — must preserve determinism for U6's cross-CPU pixel-hash CI gate; the existing `screen.ReadPixels` + SHA-256 baseline tests will catch any drift. Replaces global-state draw with explicit iteration — a one-shot architectural change that future hand-rolled draw calls in third-party extensions must respect.

**Confidence:** 90%

**Complexity:** Medium

**Status:** Unexplored

---

### 2. RenderLayer ordered passes — compounding follow-up to Drawable Adapter

**Description:** Replace the single-pass entity-draw with a `RenderPass` interface and a layer list the renderer walks in order. Built-ins: `BackgroundPass`, `TilemapPass`, `EntityPass` (the Drawable Adapter from idea #1), `VisualEffectPass` (post-process overlays), `UIPass`, `DebugPass`. External code calls `pixelforge_render.RegisterPass(layer RenderLayer, fn PassFn)` in an `init()`.

**Axis:** A1

**Basis:** `external:` `yohamta/donburi`'s layered renderer API (`ecs.AddRenderer`) — MIT, active May 2026, Ebitengine-native — proves the pattern delivers on its compounding promise; the layered shape is also used by Godot, Unity, and Phaser. `reasoned:` Pixelforge already has the seams: `pixelforge_visual_effects`, the per-tile `pixelforge_tilemap`, the `pixelforge_menus.MenuStack`, and U7's visualSink-driven overlays — all of which currently merge into one `pixelforge.Draw()` global. Layering makes the merge explicit.

**Rationale:** Particles, weather, lighting, screen-shake, fade-transitions, screen-wipes, pause overlay, CRT scanline filter — each is a `RegisterPass(LayerFX, drawX)` in an init() instead of an edit to shared renderer code. Hackathon contributors can ship visual features without touching `pixelforge_render/rendertick.go`. Pairs with idea #1: `EntityPass` IS the Drawable Adapter; the other layers are mechanical additions.

**Downsides:** Bigger architectural lift than #1 alone; useful only once #1 lands as the foundation. Adding a layer registration system without an immediate consumer is premature; this should ship after idea #1 demonstrably works.

**Confidence:** 75%

**Complexity:** Medium

**Status:** Unexplored

---

### 3. GoReleaser + GitHub Release + tiered player-binary discovery

**Description:** Add `.goreleaser.yaml` that cross-compiles `./cmd/pixelforge-player` for the five `GOOS/GOARCH` targets already enumerated in the Makefile (`Makefile:30-39`), attaches them to a GitHub Release as artifacts, and emits a `manifest.json` listing `(goos, goarch, version, sha256, url)` entries. Studio's player-binary resolution becomes a tiered chain: (1) `BuildRequest.PlayerBinaryPath` explicit; (2) `<userCacheDir>/pixelforge/player-cache/pixelforge-player-<os>-<arch>-<version>` with SHA-256 sidecar verify; (3) `playerbins.PlayerBinaryFor()` (embedded `embed.FS`); (4) HTTPS-fetch from `manifest.json` on miss, verify checksum, cache; (5) `go build` developer fallback. The studio binary never carries 5×25MB of embedded player binaries; the GitHub Release does. Provide a Studio menu item "Update Player Binaries" that re-runs the manifest fetch.

**Axis:** A2

**Basis:** `external:` `goreleaser/goreleaser` (MIT, ~14k stars, mature) cross-compiles Go binaries with one config file. The `~/.cache/pixelforge/player-cache/` directory ALREADY EXISTS in the build pipeline (U3) — download-on-first-run is the direction the cache structure was already chosen for. `direct:` `Makefile:30-39` already encodes the exact 5-target matrix; GoReleaser config is a near-1:1 translation. `direct:` U3's `playerbins.PlayerBinaryFor` already returns `ErrNotEmbedded` as the soft-miss signal for chain continuation.

**Rationale:** Solves no-Go-user Build → Host without bloating the studio binary by 125MB. Decouples player release cadence from studio release cadence — when a player-only bugfix ships, the studio doesn't have to rebuild. Per-OS code-signing/notarization moves to a declarative GoReleaser block instead of being a recurring footgun on every tag. CI-friendly: tag push → release → users get fresh bins automatically. macOS arm64 codesigning (the actual reason `bins/` is empty) becomes a single hook step.

**Downsides:** Requires network access on first Host build (cache makes subsequent builds offline). Adds GitHub Actions secret-management for macOS signing (when v3 enables it). Manifest signing for tamper-protection is a v3 concern (the doc-review surfaced this gap during plan-009).

**Confidence:** 85%

**Complexity:** Medium

**Status:** Unexplored

---

### 4. Collision-as-topic — one resolv pass publishes `on_collide` events

**Description:** Add `collisionSink` to `capsuleruntime` that runs once per tick after motion sinks: queries `rt.Physics` (already a `resolv.Space`) for overlapping body pairs via `resolv.Space.CollisionTest` or `Shape.IntersectionTest`, looks up each entity's archetype, and publishes one `*VerbEvent{Topic: "collision/body_collide", Args: {a_id, b_id, a_archetype, b_archetype}}` per pair on `pixelforge_loop.VerbsBus()`. New catalog recipe `RecipeOnCollide` with trigger constant `TriggerWhenBodyCollide` lets verb sheets subscribe with archetype-pair filters: `Bullet × Enemy → spawn/destroy_other + globals.score += 100`, `Player × Goal → scene/transition victory`, `Enemy × PatrolWall → motion/reverse_velocity`. Damage/spawn/scene-transition sinks already exist (U7/U9) and consume these events unchanged.

**Axis:** A3

**Basis:** `external:` `solarlune/resolv` v0.8.1 already an indirect dep (U6) with native `IntersectionTest` + `Shape.Tags`. `direct:` `pixelforge_physics.World` already owns the resolv `Space` (`runtime.go:106`); `pixelforge_loop.VerbsBus()` is the established sink-publish seam (`verbs_bus.go`); `damage_sink.go` already exists and would consume `damage/take_damage` events emitted by `RecipeOnCollide`. Strong cross-frame convergence — four of six ideation frames independently arrived at this design (the strongest pattern in the candidate set).

**Rationale:** Collision is the atomic primitive under win/lose, projectile-hit, pickup, hazard, scene-transition, and patrol-turnaround. Today each would be a separate ad-hoc check; one `collision/body_collide` topic compresses them all into catalog rows. The four named proof games (Asteroids ship-vs-rock, Mario hero-vs-goomba, Bomberman blast-vs-breakable, DK barrel-vs-hero) are the same verb with different effects. Five A3 gaps collapse to one substrate.

**Downsides:** O(n²) naive pair-enumeration without resolv's broad-phase; resolv's cell-grid Space mitigates but high-entity-count games (200+ bullets) need pairwise filtering by tag. Continuous-collision sweep absent (resolv is positional only) — fast projectiles can tunnel at >cell-size per tick; mitigation: clamp max velocity to `< cellSize` per frame.

**Confidence:** 95%

**Complexity:** Medium

**Status:** Unexplored

---

### 5. EntityPool — generalized fixed-pool allocator for projectiles, particles, pickups

**Description:** Add `pixelforge_pool/entity_pool.go` with `type Pool[T any] struct{ active, free []T }` and `Acquire()`, `Release(id)`, `ForEach(fn)`. Wire a new `combat/fire_projectile` verb to spawn from a per-archetype pool registered at Boot. Pool entries reuse `pixelforge_physics.Body` instances and integrate with idea #4's collision sink — projectile-vs-target is "just another `body_collide` event." Generalizes to particles for explosions, dropped pickups, projectile shells, score-popups, AI-spawned mooks.

**Axis:** A3

**Basis:** `external:` PICO-8 / TIC-80 bullet-pool pattern (fixed-size array indexed by `alive bool`, no allocation per spawn, no GC pressure) is the canonical retro-engine projectile pattern. `direct:` `pixelforge_pool/` package already exists with `pipool.go` — the generic-pool seam is conceptually claimed by the repo's organization. `reasoned:` Determinism contract holds: pool slot selection is deterministic (next-free-index), Body instances are reused rather than created/destroyed, snapshot save (U10) sees a static set of Bodies regardless of in-flight projectile count.

**Rationale:** Once the first pool exists, the next five entity types (particles, drops, shells, popups, mooks) are five-line additions. Without a pool, each gets ad-hoc spawn/despawn with map allocation churn during play (a determinism risk for the cross-CPU pixel-hash CI gate U6 set up). Compounding with #4: every pooled entity gets collision wiring for free. Compounding with #1: every pooled entity gets rendered for free (the Drawable Adapter walks `rt.CurrentScene.Entities`, pool members live there). Three blockers (projectile firing, body-vs-body collision, win/lose-via-collisions) all close via #4 + #5 + #6.

**Downsides:** Adds a generic-pool dependency on Go 1.18+ generics (already met). Fixed pool size means worst-case allocation must be authored per-scene; running out mid-game requires either silent drop-spawn or visible failure. Schema needs an additive `Scene.PoolBudgets map[string]int` to author the budget.

**Confidence:** 80%

**Complexity:** Medium

**Status:** Unexplored

---

### 6. Counter-polling win/lose — `meta/win_when` + `meta/lose_when` against Globals

**Description:** Add two recipes — `RecipeWinWhen` and `RecipeLoseWhen` — that take a `Predicate` reusing the existing `catalog.ConditionBuilder` registry. Predicate examples: `{kind: "counter_geq", path: "globals.score", value: 1000}` / `{kind: "entity_count", archetype: "Enemy", op: "==", value: 0}` / `{kind: "globals_lte", path: "globals.player_hp", value: 0}`. A new `winLoseSink` subscribes to `EventLateUpdate`, evaluates both predicates each tick against `rt.Globals` (already the cross-sink scratchpad per `runtime.go:117-127`), and on first satisfaction publishes a `scene/transition` event (win → `victoryScene`, lose → `loseScene`, both schema-declared per scene). `spawn_sink` increments `globals.enemy_count` on enemy spawn; `damage_sink`'s die cascade decrements. Reuses U7's existing `game_over` menu shape in `pixelforge_menus.registry.go:148`.

**Axis:** A3

**Basis:** `external:` GDevelop's "all enemies destroyed → next scene" via counter polling — the canonical pattern in the no-code-game-engine space; works without callbacks because the game-loop already polls every tick. `direct:` `pixelforge_studio/capsuleruntime/runtime.go:117-127` documents `rt.Globals` as the cross-sink scratchpad with `player_hp` as the precedent for this exact storage shape. `direct:` `pixelforge_menus/registry.go:148` proves the `game_over` menu surface already exists.

**Rationale:** Today a studio user can build a level with enemies but cannot express "the level ends when they're all dead." That's the single biggest "I made a thing but it's not a game" complaint. Counter-polling against `Globals` is ~30 lines of code and finishes the A3 story. Pairs with #4 (collision events increment counters) and the existing damage cascade (die → spawn/destroy_other decrements enemy_count). The polling approach beats event subscription because there's no "who emits 'player_died'?" question — the condition reads world state directly.

**Downsides:** Polling every tick is wasteful at very large entity counts (1000+); fine at arcade scale (~50). Predicate-language sprawl risk — limit to a small set of comparison kinds (`counter_geq`, `counter_lte`, `entity_count_eq`, `globals_eq`) at v2 rather than admit arbitrary expressions. The "first-satisfaction wins" rule means a win + lose condition firing on the same tick is ambiguous; explicit precedence (win > lose) needs a decision.

**Confidence:** 85%

**Complexity:** Low

**Status:** Unexplored

---

## Cross-cutting observation

Ideas **4 + 5 + 6 interlock**: collision-as-topic publishes `on_collide` → entity-pool spawns bullet via `combat/fire_projectile` → bullet hits enemy via `body_collide` → damage cascade fires `die` → spawn-sink decrements `globals.enemy_count` → counter-polling `win_when` predicate matches → scene transition fires. The three together close 5 of the 6 missing A3 verbs (projectile, body-vs-body, win, lose, scene-transition) with three small, single-responsibility additions plus existing catalog recipes for everything else.

Ideas **1 + 2** are sequenced: idea #1 (Drawable Adapter) closes A1 immediately; idea #2 (RenderLayer ordered passes) is its natural compounding follow-up. Ship #1 first; #2 is justified by the next visual feature anyone asks for.

Idea **3** is independent — A2 has no cross-cutting dependency on A1 or A3; can land in parallel.

## Rejection Summary

| # | Idea (source frame) | Reason rejected |
|---|------|-----------------|
| 1 | F2 #1 Kill `Entity.Position` field entirely | High blast radius — breaks damage radius, AI sight-lines, snapshot reflection; the Drawable Adapter (survivor #1) achieves the same correctness without breaking consumers |
| 2 | F2 #2 Position as method (compiler-enforced) | Smaller-blast-radius variant of #1; still touches every reader; survivor #1 wins on surgical change |
| 3 | F2 #3 Auto-sync hook (copy Bodies → Entity.Position pre-draw) | Write-side fix; works but creates two writers (motion sinks + sync hook) and confuses the source-of-truth question; survivor #1 has one writer (motion sinks) and the renderer reads from it directly |
| 4 | F3 #1 Bodies *are* the position (non-physics = mass=0 kinematic) | Conceptually clean but breaks the (already-shipped) entity-without-body case; requires a body for every entity even where physics doesn't apply |
| 5 | F3 #2 Drawables register with the body, not the entity | Architectural inversion; valid but bigger lift; the Drawable Adapter is the surgical version |
| 6 | F3 #3 Lean into duality — Entity.SpawnAt vs Body | Conceptual clarification, not a closer of A1 blocker |
| 7 | F4 #2 RenderLayer ordered passes (as A1 closer) | Promoted to survivor #2 as compounding follow-up to #1; not standalone |
| 8 | F4 #3 Camera as a service (struct, not scalar) | Additive; doesn't close A1; can land later when horizontal-scroll/screen-shake/zoom are needed |
| 9 | F5 #1 Renderer-as-spreadsheet-cell | Analogy is sound but design collapses to "renderer reads body, falls back to authored" — that's survivor #1 |
| 10 | F5 #2 Pneumatic tube — body publishes BodyMoved events | Overcomplicated for the local-tick-coherent case; adds VerbsBus traffic without payoff; renderer-pulls-from-Bodies (survivor #1) is simpler |
| 11 | F5 #7 IK-style baking (pause → bake Body → Entity.Position) | Interesting "pause-snapshots-live-state" UX feature but not an A1 blocker closer; can land later as an editor convenience |
| 12 | F6 #1 Body-as-Transform (collapse Position component into Body) | Duplicate of F3 #1 |
| 13 | F6 #2 RenderSink subscribes to EventLateDraw | Valid architectural alternative; Drawable Adapter (survivor #1) is more direct because it doesn't add an extra event hop |
| 14 | F1 #3 Studio self-bootstrap (`go build` on first launch when bins/ empty) | Only helps Go-having developers; the user explicitly named the no-Go user as the failing case |
| 15 | F2 #4 JIT-compile-on-host (embed Go source instead of binary) | Requires Go toolchain on user machine; contradicts no-Go-user requirement |
| 16 | F2 #5 CI-published bin index via GH Pages + checksum cache | Survivor #3 (GoReleaser + GH Release + manifest) is the canonical Go-ecosystem version of this; GH Pages is a less standard hosting choice |
| 17 | F3 #4 Studio binary IS player binary (cart-append on self) | Clever but plan-009 explicitly chose two-binary architecture; reframing this contradicts the shipped decision and creates ambiguity around chrome stripping |
| 18 | F3 #5 / F6 #3 Cart-as-HTML — WASM-only canonical | Drops the native Build → Host promise plan-009 committed to (R2 in the brainstorm). Could be a v3 simplification if the team decides to retreat from native — out of scope for closing the v2 gap |
| 19 | F4 #4 GoReleaser commits bins to repo (release-bot PR) | Repo bloat; survivor #3 (artifacts in GH Release, not committed) avoids carrying 125MB in git history |
| 20 | F4 #5 Three-tier discovery chain (alone) | Folded into survivor #3 — three-tier (cache → embed → on-demand) IS part of the GoReleaser-tiered solution |
| 21 | F5 #3 Printer-driver-style first-run fetch | Folded into survivor #3 — the download path IS the printer-driver pattern with GoReleaser as the build side |
| 22 | F5 #4 Vaccine cold-chain (signed manifest + content-addressed cache) | Manifest signing is a v3 concern flagged in plan-009's doc-review; v2 ships TLS-only fetch + per-pack SHA-256 |
| 23 | F5 #8 OCI registry multi-arch manifest list | Over-engineered for this scale; GoReleaser + GH Release is simpler with comparable benefits |
| 24 | F6 #4 GoReleaser-emitted manifest + download (alone) | Folded into survivor #3 |
| 25 | F1 #5 Fixed-pool projectile + resolv tags (specific to projectiles) | Generalized in survivor #5 (EntityPool covers projectiles + particles + pickups + drops) |
| 26 | F1 #6 Concrete motion/move_pattern handler | Small additive fix; lands as a single sink case after #4 + #5 + #6 set the substrate; valuable but doesn't warrant its own ideation row |
| 27 | F2 #6 Trace-driven verb codegen | Speculative — traces may not be expressive enough for arbitrary verb shapes; the catalog hand-coded recipes work fine and U24's gendocs already auto-documents them |
| 28 | F3 #6 Catalog-driven verbs from observed gameplay | Duplicate of F2 #6 |
| 29 | F3 #7 ONE verb: `react(condition, action)` | Over-abstraction; the existing 47-recipe catalog model is fine and U24 proves it scales (gendocs auto-emits 493-line docs); collapsing to one verb adds complexity without payoff |
| 30 | F4 #8 Behavior tree via catalog + blackboard | Premature — closed-form patterns (sine, patrol, follow-waypoints) should land first; BT is the eventual destination when the recipe count exceeds ~100 |
| 31 | F5 #5 MIDI step sequencer / decision table UI | Interesting UI concept but adds scope (new authoring surface); the catalog dropdown already exists and covers the same expression |
| 32 | F5 #6 Drools-style rule engine | Over-engineered for arcade-game scale; survivor #4 (collision-as-topic) gives the same "fact-driven cascade" without a generic rule engine |
| 33 | F6 #5 body_vs_body verb-pair compositions | Duplicate of survivor #4 |
| 34 | F6 #6 Recompute-on-load collision (never serialized) | Supporting note for survivor #4 (collisions are pure functions of Bodies × Entities), not a standalone idea |
| 35 | F1 #7 Win/lose meta-sink (specific framing) | Same design as survivor #6; F6 #7 wording chosen as the survivor's source |
