---
date: 2026-05-19
type: feat
topic: arcade-shipping-v2
status: active
origin: docs/brainstorms/2026-05-18-arcade-shipping-v2-requirements.md
depth: deep
---

# feat: Arcade Shipping v2 — Universal Player + Cart-Append + Reference-Game Proof

## Summary

Land the universal-player + cart-append architecture so any user-authored game ships as one self-contained executable (host system or browser); fill the eight log-stub runtime sinks left by plan-008 with concrete implementations driven by a configurable physics core; and prove the architecture by getting four reference games (Asteroids, Mario, Bomberman, Donkey Kong) playable end-to-end as CI fixtures with recorded input traces, pixel-hash parity, and `verbs.bus` event-sequence parity. The engine and studio stay general-purpose; the four games are CI proof + opt-in study material, not bundled product.

---

## Problem Frame

Plan-008 shipped the scaffolding for arcade shipping: `capsuleruntime` loaders, `verbs.bus` typed pub/sub, real `go build -tags=long` host + WASM builders, icon rasterization, WASM save backend, asset-library manifest substrate, custom-ingest watcher, Library workspace, and 11 arcade verb recipes. None of that landed a *playable* user-authored game.

Today the loop is honest about its substrate-ness: a user authors a game, clicks Build, gets a runnable `.exe`, presses Space — and the binary logs `[capsuleruntime] motion/jump map[strength:5]` to stderr without moving the sprite. Eight of eleven verb-recipe sinks are `log*Sink` stubs at `pixelforge_studio/capsuleruntime/subscribers.go:503–537` (`logMusicSink`, `logSceneController`, `logDamageSink`, `logMotionSink`, `logSpawnSink`, `logVisualSink`, plus `dialogueController` log-only and `saveServiceSink` which writes literal empty `pisave.Snapshot{}` at line 651); the asset-library manifest URL 404s; the ingest dispatcher's `SetSpriteRunner`/`SetSFXRunner`/`SetBGMRunner` are wired only in tests, not in `pixelforge_studio/main.go`; three of four named games have no `.pforge` fixture (only `mario_strip_scene.pforge` and `goomba_scene.pforge` exist under `pixelforge_studio/integration_test/fixtures/`); and the current build pipeline regenerates per-cart `main.go` + `capsule.go` + invokes `go build` per project — not the universal-player architecture this plan lands.

The cost shape is dishonesty: every "Build" produces an artifact, the artifact runs, the runtime does nothing recognizable as gameplay. A user encountering this would correctly conclude the engine is theatre — and we would have shipped substrate without proving the substrate connects to anything.

---

## Origin Document

This plan is sourced from `docs/brainstorms/2026-05-18-arcade-shipping-v2-requirements.md` (see origin). All requirements R1–R22, actors A1–A3, key flows F1–F3, acceptance examples AE1–AE10, scope boundaries, key decisions, and dependencies carry forward. The user explicitly authorized external libraries + downloaded code provided plan-008 invariants stay intact (cycle-break injection pattern, ImGui-only studio chrome, schema additivity, `CGO_ENABLED=0` for WASM, no string-concat source generation).

**User constraint preserved verbatim from durable memory:** *do not use git at all during this work.*

---

## System-Wide Impact

| Surface | Touched by |
|---|---|
| `pixelforge_studio/capsuleruntime/` (sinks) | U7, U8, U9, U10, U11 — every `log*Sink` replaced |
| `pixelforge_studio/buildpipeline/` | U3, U23 — pipeline pivot + WASM bundle + size reporting |
| `pixelforge_studio/codegen/` | U3 — codegen simplifies dramatically (no per-cart `main.go`) |
| `pixelforge_studio/main.go` | U19 — ingest runners wired |
| `pixelforge_studio/editor/` | U21, U22 — File menu templates + Open Example |
| `pixelforge_studio/assetlibrary/` | U20 — manifest publication + Examples field |
| `pixelforge_studio/integration_test/fixtures/` | U11, U14, U16, U18 — four proof carts |
| `pixelforge_loop/verbs_bus.go` | unchanged (stable contract) |
| `pixelforge_project/` | U10 — snapshot serialization touches Project|Scene|Entity reflection |
| New: `pixelforge_cart/` | U1 — magic-footer envelope, read-self, append |
| New: `pixelforge_physics/` | U6 — resolv-backed substrate + tilemap AABB + gravity |
| New: `pixelforge_render/` (or `capsuleruntime/render.go`) | U4 — shared `RenderTickAt` seam |
| New: `cmd/pixelforge-player/` | U2 — stripped universal-player binary |
| New: `pixelforge_studio/playerbins/` | U3 — per-OS pre-built player binaries via `embed.FS` (no-Go user path) |
| New: `pixelforge_replay/` | U5 — `.trace.jsonl` format + replay harness |
| New: `.github/workflows/long.yml` | U25 — long-tag CI matrix + capability matrix regen |

Stakeholders: A1 (end users) get a working ship loop; A2 (game consumers) get binaries that run on their machine without Pixelforge installed; A3 (Pixelforge engineers) get CI gates on every commit instead of manual demos. Plan-008 invariants are preserved.

---

## Output Structure

New directories and files this plan introduces or substantially restructures (file paths repo-relative; per-unit `Files:` sections remain authoritative for exact create/modify lists):

```
pixelforge-go/
├── cmd/
│   └── pixelforge-player/
│       ├── main.go                       # universal player entry (U2)
│       └── main_test.go
├── pixelforge_cart/                      # NEW (U1)
│   ├── doc.go
│   ├── envelope.go                       # magic+length+CRC32 footer
│   ├── envelope_test.go
│   ├── append.go                         # write-side: copy player + append cart
│   ├── append_test.go
│   ├── selfread_native.go                # //go:build !js  (os.Executable + ReadAt)
│   ├── selfread_js.go                    # //go:build js   (read window.__pixelforgeCart)
│   └── selfread_test.go
├── pixelforge_physics/                   # NEW (U6)
│   ├── doc.go
│   ├── world.go                          # solarlune/resolv wrapper
│   ├── world_test.go
│   ├── tilemap.go                        # 16x16 tile-grid AABB
│   ├── tilemap_test.go
│   ├── gravity.go                        # integrator + jump impulse
│   ├── gravity_test.go
│   ├── oneway.go                         # one-way platform DIY (MTV.Y check)
│   ├── ladder.go                         # ladder-climb mechanic
│   ├── trig.go                           # LUT-based sin/cos for determinism
│   └── trig_test.go
├── pixelforge_render/                    # NEW (U4)
│   ├── doc.go
│   ├── rendertick.go                     # RenderTickAt(p, tick, inputs) → *image.RGBA
│   └── rendertick_test.go
├── pixelforge_replay/                    # NEW (U5)
│   ├── doc.go
│   ├── trace.go                          # .trace.jsonl read/write
│   ├── trace_test.go
│   ├── recorder.go                       # capture during studio preview
│   ├── recorder_test.go
│   ├── replayer.go                       # deterministic replay through RenderTickAt
│   └── replayer_test.go
├── pixelforge_studio/
│   ├── integration_test/
│   │   └── fixtures/
│   │       ├── asteroids_proof.pforge         # NEW (U11)
│   │       ├── asteroids_proof.trace.jsonl    # NEW (U11)
│   │       ├── asteroids_proof.baseline.json  # NEW (U11) — pixel-hashes + bus events
│   │       ├── mario_proof.pforge             # NEW (U14)
│   │       ├── mario_proof.trace.jsonl        # NEW (U14)
│   │       ├── mario_proof.baseline.json      # NEW (U14)
│   │       ├── bomberman_proof.pforge         # NEW (U16)
│   │       ├── bomberman_proof.trace.jsonl    # NEW (U16)
│   │       ├── bomberman_proof.baseline.json  # NEW (U16)
│   │       ├── donkey_kong_proof.pforge       # NEW (U18)
│   │       ├── donkey_kong_proof.trace.jsonl  # NEW (U18)
│   │       └── donkey_kong_proof.baseline.json # NEW (U18)
│   ├── starterpack/                      # NEW (U20)
│   │   ├── assets/                       # CC0 placeholder sprites + SFX/BGM
│   │   └── embed.go                      # //go:embed all:assets
│   └── scripting/catalog/
│       └── cmd/gendocs/                  # NEW (U24)
│           └── main.go                   # go-generate target for verb-catalog.md
├── docs/
│   ├── reference-games-capability-matrix.md  # regenerated by CI (U25)
│   └── verb-catalog.md                       # regenerated by go generate (U24)
└── .github/
    └── workflows/
        └── long.yml                          # NEW (U25) — long-tag CI matrix
```

This tree is a scope declaration. Per-unit `Files:` blocks remain authoritative; implementers may adjust layout if implementation reveals a better one.

---

## High-Level Technical Design

*This section illustrates the intended approach and is directional guidance for review, not implementation specification. The implementing agent should treat it as context, not code to reproduce.*

### Architecture: universal player + cart-append

```
                                ┌──────────────────────────────────────┐
                                │       single Go codebase             │
                                │  pixelforge_* engine packages        │
                                │  + pixelforge_cart + pixelforge_     │
                                │    physics + pixelforge_render       │
                                └────────────┬─────────────────────────┘
                                             │ go build with build tags
                          ┌──────────────────┼──────────────────┐
                          ▼                  ▼                  ▼
                  pixelforge-studio   pixelforge-player   pixelforge-player.wasm
                  (editor + chrome    (stripped runtime;  (stripped runtime;
                   + embedded         host binary per     web build)
                   player runtime)    OS: Win/Mac/Linux)
                          │                  │                  │
                          │ Build → Host     │ shipped artifact │ shipped artifact
                          │ produces cart    │ = copy of        │ = wasm + base64
                          │ + appends it     │   player +       │   cart in one .html
                          │ to a copy of     │   appended       │
                          │   player         │   .pforge        │
                          ▼                  ▼                  ▼
                  pixelforge-studio    mygame.exe / .app  mygame.html
                  reads in-process     (single binary)    (single file)
                  .pforge for Play
```

**Cart envelope (16-byte footer, written by `pixelforge_cart.Append`, read by `pixelforge_cart.ReadSelf`):**

```
[ ... executable bytes (PE / Mach-O / ELF) ... ]
[ ... appended cart bytes (raw .pforge JSON, gzip optional) ... ]
[ 4 bytes: CRC32 of cart payload                                ]
[ 8 bytes: cart payload length (uint64, little-endian)          ]
[ 8 bytes: magic "PFORGEv2" (ASCII)                             ]
```

At startup, the universal-player binary calls `os.Executable()` → `os.Open(path)` → `f.ReadAt(footer, size-20)`. Validate magic; read length; seek `size - 20 - length` → read cart bytes; verify CRC32. Failure modes (no cart / bad CRC / wrong magic) abort cleanly with a stderr message ("no .pforge cart appended — this player binary is unshipped"). For WASM, the cart is fed via `js.Global().Get("__pixelforgeCart")` instead (build-tagged file).

**Shared `RenderTickAt`:**

```
package pixelforge_render

func RenderTickAt(
    rt *capsuleruntime.Runtime,  // already-Boot'd runtime
    tick uint64,                  // logical tick number
    inputs InputFrame,            // sampled-once-per-tick input state
) (*image.RGBA, error)            // 320x180 paletted frame, alpha-premultiplied
```

This function is called by:
- `pixelforge_studio` preview path (the in-studio Play window draws via this exact function),
- `cmd/pixelforge-player/main.go`'s Ebitengine `Draw()` (the shipped binary calls this function from its `ebiten.Game.Draw` impl),
- `pixelforge_replay.Replayer` (CI fixture replay calls this function deterministically tick by tick).

One implementation; structural impossibility of preview-vs-shipped drift; one place to optimize; one place to pixel-hash for CI.

### Cart-append flow at Build → Host

```
[orchestrator.Build]
        │
        ▼
[hostLongBuilder.Build]      (replaces today's "codegen + go build per cart" path)
        │
        ├─ codegen.EncodeCart(req.Project) → []byte    (writes JSON .pforge bytes)
        ├─ pixelforge_cart.Append(
        │     playerBinaryPath(req.Target),            (pre-built player from CI artifact or cached)
        │     cartBytes,
        │     filepath.Join(req.OutputDir, "host", gameName + artifactExt(req.Target)),
        │   )
        ├─ if windows: BuildWindowsSyso → embed game icon BEFORE append
        ├─ if macos:   write Info.plist + AppIcon.icns into .app bundle BEFORE append
        └─ emit PhaseDone{ArtifactPath: ...}
```

The "pre-built player" comes from a CI artifact cache or a developer-machine `go build ./cmd/pixelforge-player` for the host target. Cross-OS native builds remain rejected via `ErrCrossCompileNotSupported` (no regression from plan-008 U4).

### WASM single-file bundle

The existing `wasm_bundler.go` already produces one `.html`. v2 extends it to inline the cart:

```
mygame.html
├─ <head>
│   ├─ <link rel="icon" href="data:image/png;base64,..."/>      (from icon_logo.go)
│   └─ <title>Game Title</title>
├─ <body>
│   ├─ <canvas id="screen">
│   ├─ <button id="start">Click to Start</button>               (audio autoplay gate)
│   ├─ <script>...entire wasm_exec.js verbatim...</script>      (Go-version-matched)
│   └─ <script>
│        const WASM_B64 = "AGFzbQ...";                          (base64 of pixelforge-player.wasm)
│        const CART_B64 = "...";                                (base64 of .pforge bytes)
│        const startBtn = document.getElementById("start");
│        startBtn.onclick = async () => {
│          window.__pixelforgeCart = Uint8Array.from(atob(CART_B64), c => c.charCodeAt(0));
│          const wasmBytes = Uint8Array.from(atob(WASM_B64), c => c.charCodeAt(0));
│          const go = new Go();
│          const {instance} = await WebAssembly.instantiate(wasmBytes, go.importObject);
│          startBtn.remove();
│          go.run(instance);
│        };
│      </script>
```

The Go side reads `__pixelforgeCart` via `js.Global().Get(...)` + `js.CopyBytesToGo`. Click-to-start is the universal pattern for satisfying browser autoplay policy before `AudioContext` initialization.

### Input-trace format (`.trace.jsonl`)

JSON-lines, one tick per line:

```
{"v":1,"meta":{"game":"asteroids_proof","seed":42,"width":320,"height":180,"tps":60,"duration_ticks":5400}}
{"tick":0,"keys":[],"pad":null}
{"tick":1,"keys":[],"pad":null}
...
{"tick":47,"keys":["Space"],"pad":null}
{"tick":48,"keys":["Space","ArrowLeft"],"pad":null}
...
```

First line is metadata; subsequent lines are per-tick input frames. Empty `keys: []` lines may be coalesced via run-length compression (`"keys":["Space"],"hold":12`) — implementation detail, the canonical wire format stays explicit per-tick. Format chosen for: git-diffability (FCEUX FM2 precedent), splice-ability for trace authoring, human readability for debugging CI failures. At 90s × 60fps × ~50 bytes/line = ~270KB per game.

### Snapshot save: entity-component reflection

```
pisave.Snapshot{
    SchemaVersion: 2,                  // bumped from 1; v1 was empty
    Tick:          12345,
    SceneID:       "level_1",
    Entities: []EntitySnapshot{
        {ID: "player_1", Position: Vec3{x,y,z}, Components: map[string]json.RawMessage{
            "Sprite":    {...},
            "Physics":   {...},
            "Inventory": {...},
        }},
        ...
    },
    Globals: map[string]json.RawMessage{
        "music_track":      "...",
        "lives":            "3",
        "score":            "42",
    },
}
```

Serialization walks the active `pixelforge_project.Project.Scenes[currentScene].Entities`, iterates each entity's registered component types, marshals each via `json.Marshal` into a `RawMessage`. Load does the reverse: replaces the current scene's entities with the snapshot's, deserializes each component back to its registered type. No game-specific code; works for all four reference games + any user-authored game. Schema-additive per plan-008 invariant: new component types append to the map; missing-on-load components stay nil.

---

## Key Technical Decisions

- **Cart-append envelope is DIY, not `gonutz/payload`.** A 20-byte footer (`[CRC32 4B][length 8B][magic "PFORGEv2" 8B]`) costs ~80 LOC across `pixelforge_cart/envelope.go` + `selfread_*.go`. Zero external deps preserves plan-008's minimal-dependency discipline; CRC32 catches corruption that `gonutz/payload`'s footer alone doesn't; custom magic catches cart/runtime version mismatches at startup. *Why:* small enough to own; protects the cart integrity gap external libs leave; aligns with the "no string-concat source generation" + minimal-dependency invariants the user reaffirmed.

- **Adopt `github.com/solarlune/resolv` v0.8.1 for collision; build `pixelforge_physics` thin layer on top.** MIT, pure-Go, no CGO, WASM-safe. Covers AABB + SAT + spatial partitions out of the box. *Lacks:* one-way platforms (DIY via MTV.Y sign check) and slopes (DIY via `ConvexPolygon` triangle). Tilemap AABB is hand-written ~150 lines (specific to 16×16 grid; resolv's SAT is overkill there). *Why:* Leverage Doctrine + user authorization of external libs; avoids reinventing SAT; keeps our domain code thin and focused on arcade-specific mechanics (one-way platforms, ladders, screen-wrap, grid placement) where resolv would force awkward shapes.

- **`.trace.jsonl` text format over binary.** Plain JSON-lines mirrors FCEUX FM2's design choice — text-by-default for splicing, git-diffability, and source-control hygiene. Trace files live in `pixelforge_studio/integration_test/fixtures/`; reviewers can read them. Size cost (~270KB per game × 4 games ≈ 1.1MB total) is acceptable. *Why:* CI fixture trace files are *committed artifacts*; debugging a flaky pixel-hash test requires reading the trace.

- **Reference-example fetch reuses the asset-library manifest.** Extend `pixelforge_studio/assetlibrary/manifest.go`'s `Manifest` schema additively with an `Examples []Example` field alongside `Packs`. One `EnsureBootstrap` call covers both library packs and reference examples; one URL, one cache root, one SHA-256-verified downloader. *Why:* avoids two parallel fetchers; preserves the plan-008 asset-library substrate intact; schema additivity respected.

- **Verb catalog regenerated via `go generate ./...`.** A `//go:generate go run ./pixelforge_studio/scripting/catalog/cmd/gendocs` directive on a sentinel file in the catalog package. CI runs `go generate ./... && git diff --exit-code docs/verb-catalog.md` (or the equivalent path-check) to verify the file is up-to-date. *Why:* idiomatic Go; works headlessly in CI; one source of truth for Inspector dropdowns + codegen + docs + lint + capability-matrix.

- **Determinism strategy is empirically probed in U6, not assumed.** `pixelforge_physics` uses LUT-based trig (`SinDeg/CosDeg` over a 65536-entry table) + Fixed32 16.16 arithmetic for our own velocity integrators (IEEE-754 `+ - * /` are bit-identical across amd64/arm64 via SSE2/NEON 64-bit). HOWEVER: `solarlune/resolv` uses `float64` and `math.Sqrt` internally for SAT projections — these are NOT guaranteed bit-identical cross-CPU, and Go's arm64 codegen may emit FMA where amd64 does not. The cross-CPU byte-equal pixel-hash assumption is therefore unverified at plan-time. U6's early-determinism probe (see Execution note) measures empirically before any baselines land; the outcome determines whether (a) we proceed with byte-equal pixel-hash CI across amd64+arm64, (b) we downgrade to tolerance buckets + perceptual hashing, or (c) we restrict pixel-hash CI to single-CPU (`ubuntu-latest amd64`). The strategy chosen is documented in `pixelforge_physics/doc.go` and gates U11/U14/U16/U18 baseline-recording approach. *Why:* committing to byte-equal cross-CPU before measuring is overconfident; the probe is cheap and gates four games' worth of baseline investment.

- **WASM size thresholds (R21) stay at 15MB warn / 30MB error initially; tune empirically after U23.** Framework research projects Pixelforge-scale at ~18-25MB raw uncompressed for the full runtime + Ebitengine + cofont + dialogue + GUI. Initial builds will trip the warn threshold; that's the *intended signal* — the threshold makes the cost visible. U23 measures the real baseline post-implementation and proposes adjusted thresholds in a follow-up.

- **Pre-built player binaries cached in `CI_ARTIFACTS_DIR` or built on demand from `cmd/pixelforge-player/`.** The build pipeline at host-build time needs a copy of `pixelforge-player[.exe|.app]` for the target OS to append the cart to. Two sources: (a) CI publishes per-OS player binaries as workflow artifacts on every commit to `main`, cached for local dev; (b) `buildpipeline/builders_long.go` falls back to `go build -tags=long -o $TEMP/pixelforge-player ./cmd/pixelforge-player` when no artifact is cached. *Why:* zero-network-dep developer experience (fallback path always works), with the option to skip the per-cart compile cost when a cached player binary exists.

- **Engine + studio stay general-purpose; named games are CI proof + opt-in study material, never bundled.** Carried verbatim from origin. The four `*_proof.pforge` fixtures live in `pixelforge_studio/integration_test/fixtures/` (CI-internal); the same `.pforge` files are also published as reference examples on a GitHub Release for users to fetch via `File → Open Example`. The studio binary never embeds them via `go:embed`.

- **Trademark posture: literal names.** User explicitly chose to keep "Mario," "Bomberman," "Donkey Kong," "Asteroids" as the reference-game names everywhere — fixture filenames, internal code, menu labels, GitHub Release asset names. Hobby/research project posture; trademark risk accepted. Plan does NOT adopt SuperTux-style genre-descriptor rename.

---

## Requirements Traceability

| R-ID | Origin Requirement | Implementation Units |
|---|---|---|
| R1 | Two binaries from one codebase (studio + player) | U2 |
| R2 | Build → Host = player + appended cart | U1, U3 |
| R3 | Build → WASM = one self-contained .html with inline base64 cart | U3 (WASM bundler owns inlining) |
| R4 | Cross-OS native build rejected | preserved by U3 (no regression) |
| R5 | Concrete sinks for all 11 verb categories | U7 (scene/visual/spawn/dialogue/music) + U8 (motion/collision) + U9 (damage) + U13 (jump/gravity/solid_collide) + U15 (place_on_grid/ladder/grid-explode) + U17 (barrel_roll/multi-screen) |
| R6 | One configurable physics core with per-cart params | U6 |
| R7 | Save snapshot captures every entity's component state automatically | U10 |
| R8 | .pforge fixtures + input traces for all four named games | U11, U14, U16, U18 |
| R9 | CI per-commit fixture replay; pixel-hash + bus event parity must match | U5, U25 |
| R10 | File → New genre-starter templates | U22 |
| R11 | File → Open Example fetches from GitHub Release | U21 |
| R12 | Single shared RenderTickAt function | U4 |
| R13 | Capability matrix regenerated from CI | U25 |
| R14 | Embed small CC0 starter asset set | U20 |
| R15 | Asset-library manifest URL works | U20 |
| R16 | Four named games' assets ship as curated library packs (opt-in) | U21 |
| R17 | Ingest dispatcher wired in main.go for PNG/WAV/OGG/MP3 | U19 |
| R18 | Verb catalog single source of truth | U24 |
| R19 | Catalog covers all four games; ~50-80 verbs target | U24 |
| R20 | Press-key-to-bind capture in input-binding workspace | U22 (combined with template wiring) |
| R21 | WASM size warn 15MB / error 30MB; size in success toast | U23 |
| R22 | mygame.html + mygame.html.gz both written; wasm-opt optional | U23 |

Acceptance examples AE1–AE10 are covered by test scenarios in the matching units (see per-unit `Test scenarios`).

---

## Implementation Units

Five phases. Phase 1 lands the universal-player architecture proven end-to-end by one game (Asteroids). Phases 2–4 each extend the player with the next game's physics + ship its CI fixture. Phase 5 polishes the ship-loop UX + CI. Each phase is independently demoable; the architectural bet pays off at Phase 1, not at Phase 5.

### Phase 1: Universal-Player Architecture + Asteroid-Shooter Proof

### U1. pixelforge_cart envelope + read-self

**Goal:** Create the `pixelforge_cart` package implementing the 20-byte append/read-self envelope used by every shipped binary.

**Requirements:** R2.

**Dependencies:** none (greenfield package).

**Files:**
- `pixelforge_cart/doc.go` (new)
- `pixelforge_cart/envelope.go` (new) — `MagicV2 = "PFORGEv2"`, `FooterSize = 20`, `EncodeFooter(payload []byte) []byte`, `DecodeFooter(f []byte) (length uint64, crc uint32, err error)`
- `pixelforge_cart/envelope_test.go` (new)
- `pixelforge_cart/append.go` (new) — `Append(playerBinaryPath, cart []byte, outputPath string) error`
- `pixelforge_cart/append_test.go` (new)
- `pixelforge_cart/selfread_native.go` (new) — `//go:build !js`; `ReadSelf() ([]byte, error)` via `os.Executable()` + `os.Open` + `f.ReadAt`
- `pixelforge_cart/selfread_js.go` (new) — `//go:build js`; `ReadSelf()` via `js.Global().Get("__pixelforgeCart")` + `js.CopyBytesToGo`
- `pixelforge_cart/selfread_test.go` (new) — `//go:build !js` exercising the native path via a synthesized test executable
- `pixelforge_cart/codesign_darwin.go` (new) — `//go:build darwin`; `adhocSign(path string) error` invokes `codesign --force --sign - <path>`
- `pixelforge_cart/codesign_other.go` (new) — `//go:build !darwin`; `adhocSign(path string) error` is a no-op returning nil

**Approach:**
- Footer layout: `[CRC32 4B][length uint64 LE 8B][magic 8B]`. Stored at end-of-file; reader seeks `fileSize - 20`.
- `Append` opens the destination file once, holds a single `*os.File` for the entire write sequence (open-once not open-per-step) to eliminate TOCTOU windows. Copies `playerBinaryPath` → `outputPath` (atomic via `<tmp>` + `os.Rename`), then appends cart bytes + footer. Preserves executable bit on Unix.
- `ReadSelf` opens own executable via `os.Executable()` → `filepath.EvalSymlinks` → `os.Open(resolved)` ONCE; all subsequent seek/read operations use that single file descriptor (TOCTOU-safe — the FD is inode-bound after open, immune to path-level substitution mid-read). On Linux, prefer `/proc/self/exe` over `EvalSymlinks` to skip the symlink-race entirely. Seeks to footer position, validates magic, reads length, seeks back, reads payload, validates CRC32. Returns `ErrNoCart` if magic missing (clean "unshipped player" signal).
- Hard limit on payload size: refuse anything >256MB to avoid OOM on corrupted binary.
- macOS `.app` bundle case: `Append` writes the binary into `<name>.app/Contents/MacOS/<name>` (not the `.app` directory itself).
- **macOS ad-hoc signature (load-bearing on arm64):** on `darwin` hosts, `Append` runs `codesign --force --sign - <output-binary-or-.app>` as the final step after appending cart bytes. Apple Silicon **requires** a valid signature even for unsigned binaries; appending bytes invalidates any prior signature and produces `killed: 9` at launch with no diagnostic if not re-signed. The `-` sentinel is the ad-hoc identity (no Apple Developer account, no entitlements, no notarization). On `darwin/amd64` the signature isn't strictly required, but the codesign step runs unconditionally on darwin for consistency. Failure modes: `codesign` not found in PATH → `ErrCodesignUnavailable` (skip on developer machines without Xcode CLT, surface in build status); codesign exit non-zero → `ErrCodesignFailed` with stderr captured.
- Windows note: append BEFORE Authenticode sign — documented in `doc.go` (code-signing is explicitly out of v2 scope but invariant matters for v3).
- **Cart integrity threat model is corruption-only, not tampering.** CRC32 catches accidental file-system corruption + transmission errors; it does NOT defend against a malicious actor who can write to the binary (they recompute a valid CRC trivially). Cryptographic signing of the cart payload is explicitly deferred to v3. `doc.go` documents this distinction so downstream readers understand the trust boundary; the player binary treats any well-formed CRC-valid cart as trusted input.

**Patterns to follow:**
- Mirror `pixelforge_save/backend_native.go` + `backend_js.go` build-tag split for the read-self platform variants.
- Match `pixelforge_studio/assetlibrary/downloader.go`'s atomic-write semantics (write to `.tmp`, `os.Rename` on success).

**Test scenarios:**
- *Happy path:* Encode a known payload, decode the footer, recover length + CRC, verify magic. Round-trip Append → ReadSelf returns byte-equal payload.
- *CRC mismatch:* Encode footer with intentionally-wrong CRC32; ReadSelf returns `ErrCorruptCart`.
- *Magic mismatch:* Read a player binary with no appended cart; ReadSelf returns `ErrNoCart` cleanly (not a panic).
- *Wrong magic ("PFORGEv1"):* ReadSelf returns `ErrVersionMismatch` with explicit version string in error message.
- *Truncated payload:* File smaller than the declared length; ReadSelf returns `ErrCorruptCart`.
- *Oversize payload:* Footer claims length=300MB; ReadSelf returns `ErrPayloadTooLarge` without attempting allocation.
- *Symlinked executable:* Create a symlink to the test binary, invoke ReadSelf via the symlink — must EvalSymlinks and read the real file.
- *macOS .app:* Append into `<name>.app/Contents/MacOS/<name>`; verify Append picks the right inner path when given the `.app` outer path.
- *macOS ad-hoc codesign (darwin only):* After Append, run `codesign --verify --verbose <output>` and assert exit code 0; `spctl --assess --type execute <output>` may still warn (Gatekeeper expects notarization) but the binary must be LAUNCHABLE — assert `<output>` exits cleanly when run.
- *macOS arm64 launch smoke (darwin/arm64 only):* Run the appended binary on Apple Silicon; assert it does NOT die with `killed: 9` (the Apple Silicon-specific failure mode for unsigned-after-modification binaries).
- *codesign-not-found:* Stub PATH to omit `codesign`; Append on darwin returns `ErrCodesignUnavailable`; the binary is still written (caller can decide whether to ship unsigned).

**Verification:** `go test ./pixelforge_cart/...` passes including the symlink + macOS bundle scenarios; `go test ./pixelforge_cart -race` clean; appended binaries launch on darwin/arm64 without `killed: 9`.

---

### U2. cmd/pixelforge-player entry point

**Goal:** Create the universal-player binary `cmd/pixelforge-player/main.go` that reads its appended cart at startup, Boots the runtime, and runs the Ebitengine game loop.

**Requirements:** R1, R2.

**Dependencies:** U1.

**Files:**
- `cmd/pixelforge-player/main.go` (new)
- `cmd/pixelforge-player/main_test.go` (new) — `//go:build !js` smoke test that the binary boots a synthetic appended cart and exits cleanly on a fake input frame

**Approach:**
- `func main()`: call `pixelforge_cart.ReadSelf()`; on `ErrNoCart`, print stderr help message + exit non-zero ("this is the universal player; it expects a .pforge cart to be appended via `pixelforge-studio Build → Host`"); on `ErrCorruptCart`, exit with diagnostic; on success, `pixelforge_project.LoadReader(bytes.NewReader(cartPayload))` → `capsuleruntime.Boot(p, embeddedAssets, capsuleruntime.Options{})` → `pixelforge_ebiten.Run()`.
- Embedded asset FS: `//go:embed all:assets`-style entry pointing at an empty `assets/.keep` directory IF the cart payload doesn't carry its own assets (the cart is expected to embed assets via the same mechanism the codegen path used). Final implementation: assets ride INSIDE the cart bytes (the `.pforge` is itself a JSON file pointing at base64-encoded sprite/audio blobs OR carries a separate inner `assets/` subdirectory — this is a U3 codegen concern, U2 just hands cart bytes to `pixelforge_project.LoadReader`).
- WASM build: same entry; `//go:build js` variant of `os.Args[0]` / `os.Executable` not needed since `ReadSelf` is build-tagged.

**Patterns to follow:**
- Existing `pixelforge_studio/codegen/templates.go:17-30` is the inspiration for the thin shell; U2's `main.go` is the real shell.
- Existing build-tag split in `pixelforge_save/backend_*.go` for native vs JS.

**Test scenarios:**
- *No cart appended:* Build the binary, run it, verify stderr message + non-zero exit code.
- *Smoke test with synthetic cart:* Append a minimal `.pforge` (one scene, one entity, one sprite) via U1's `Append`; run the binary in a goroutine with a 100ms cancel context; verify it Boots without panic.
- *Corrupt cart:* Append a payload with bad CRC; verify graceful exit with diagnostic.
- *WASM build smoke:* `GOOS=js GOARCH=wasm go build ./cmd/pixelforge-player` compiles without error; resulting `.wasm` is non-empty.

**Verification:** Binary at `cmd/pixelforge-player/` builds via `go build -tags=long`; appended-cart smoke test asserts boot-without-panic; WASM smoke test compiles cleanly.

---

### U3. Build-pipeline pivot to copy-player + append-cart (host + WASM)

**Goal:** Rewire BOTH `hostLongBuilder` AND `wasmLongBuilder` to stop generating per-cart `main.go` and instead: produce the cart bytes, then either (host) locate the pre-built player binary for the target OS + call `pixelforge_cart.Append`, or (WASM) compile the player to WASM once + inline the cart as base64 in the HTML template via `BundleWASM`. Codegen simplifies to "encode the project and any sidecar artifacts." This unit owns the R3 inline-cart WASM bundler logic in addition to R2's host-append.

**Requirements:** R2, R3, R4 (preserved).

**Dependencies:** U1, U2.

**Files:**
- `pixelforge_studio/buildpipeline/builders_long.go` (modify) — both `hostLongBuilder.Build` and `wasmLongBuilder.Build` rewritten
- `pixelforge_studio/buildpipeline/wasm_bundler.go` (modify) — `BundleWASM` accepts `cartBytes []byte`; base64-encodes alongside the wasm + inlines both into the HTML template
- `pixelforge_studio/buildpipeline/wasm_template.html` (modify) — add `CART_B64` placeholder + click-to-start splash that decodes `__pixelforgeCart` before calling `go.run(instance)`
- `pixelforge_studio/buildpipeline/builders_long_test.go` (modify) — assertions updated for both host and WASM paths
- `pixelforge_studio/buildpipeline/orchestrator.go` (modify) — add `PlayerBinaryPath string` + `WasmBinaryPath string` to `BuildRequest`; if empty, fall back to building from `cmd/pixelforge-player`
- `pixelforge_studio/playerbins/embed.go` (new) — `//go:embed all:bins`; `PlayerBinaryFor(goos, goarch string) ([]byte, error)`; extracts the embedded per-OS player binary to a temp path on demand
- `pixelforge_studio/playerbins/bins/.gitkeep` (new) — placeholder directory; release process pre-builds player binaries here (commit via Git LFS or release-time `make playerbins`)
- `pixelforge_studio/playerbins/embed_test.go` (new)
- `pixelforge_studio/codegen/generator.go` (modify) — extract `EncodeCart(p *pixelforge_project.Project) ([]byte, error)` that produces the `.pforge`-as-zip-or-jsonblob the player will read; deprecate the per-cart `main.go`/`capsule.go`/`go.mod` emission for the universal-player path while preserving the source-target builder for users who want the editable Go source
- `pixelforge_studio/codegen/templates.go` (modify) — keep the templates for `sourceLongBuilder`; the host + wasm builders no longer use them
- `pixelforge_studio/codegen/generator_test.go` (modify)
- `Makefile` (modify) — add `playerbins` target invoking `go build` for each supported host OS + WASM target, dropping results into `pixelforge_studio/playerbins/bins/<os>-<arch>/pixelforge-player[.exe|.wasm]`

**Approach:**
- `EncodeCart` returns a single `[]byte` containing the project JSON + assets (the existing project encoder + a small wrapping format that bundles `assets/` into the same byte stream). Two viable wire formats:
  - **(a) ZIP-wrapped** — `assets/` files + `project.pforge` in a real ZIP; player code unzips on `ReadSelf` return.
  - **(b) JSON-with-inline-base64-assets** — extend the `.pforge` JSON schema additively with a `bundled_assets` array carrying base64-encoded asset bytes per file.
- Decision deferred to implementation: (a) is more efficient (no base64 inflation); (b) is human-readable + matches schema additivity discipline. Implementer picks based on size/perf tradeoff at U10 (snapshot save uses similar serialization decisions).
- **Player binary discovery chain (no-Go user path is load-bearing):** at build-time, look up the player binary in this order — (1) `BuildRequest.PlayerBinaryPath` if explicitly set; (2) `<userCacheDir>/pixelforge/player-cache/pixelforge-player-<os>-<arch>-<commit>[.exe]` with SHA-256 integrity verified against a sidecar `.sha256` file; (3) `pixelforge_studio/playerbins.PlayerBinaryFor(GOOS, GOARCH)` — embedded in the studio binary via `//go:embed all:bins`, extracted to a temp path at first use **(this is the path a no-Go user takes — no toolchain required)**; (4) `go build -tags=long -o $TEMP/pixelforge-player ./cmd/pixelforge-player` (developer fallback only). If (3) succeeds, the extracted binary is copied to (2)'s cache location with its `.sha256` sidecar for next-time speed.
- **Player-binary cache integrity check:** every cache hit re-computes the SHA-256 of the cached binary against the sidecar `.sha256` value; mismatch invalidates the cache entry and falls through to the next source. Prevents poisoned cache from trojaning shipped artifacts.
- **Release process owns playerbins/bins/ content.** Pure-Go player binary cross-compiles cleanly via `GOOS=<os> GOARCH=<arch> go build -tags=long ./cmd/pixelforge-player`. Release pipeline runs `make playerbins` before tagging; binaries committed via Git LFS or attached as release artifacts (decided at release-time — implementer defers). Studio installer ships these embedded so day-one users without Go can Build → Host out of the box.
- `hostLongBuilder.Build`: replace the `codegen.Generate` + `NewBuildCommand(go build)` + per-OS post-processing with: `codegen.EncodeCart` → resolve player binary via the discovery chain above → for Windows, run `BuildWindowsSyso` on the player binary copy BEFORE append → for macOS, set up `.app` bundle skeleton → `pixelforge_cart.Append(playerPath, cartBytes, outputPath)` (which now includes the darwin ad-hoc codesign step from U1).
- `wasmLongBuilder.Build`: `codegen.EncodeCart` → resolve WASM player binary via the same chain (`PlayerBinaryFor("js", "wasm")` reads `playerbins/bins/js-wasm/pixelforge-player.wasm`) → `BundleWASM(wasmBytes, cartBytes, gameName, faviconBase64, outPath, credits)` writes the single `.html` with both `WASM_B64` and `CART_B64` inlined + click-to-start splash that wires `window.__pixelforgeCart` before `go.run(instance)`. Output: `req.OutputDir/wasm/<gameName>.html`.
- `ErrCrossCompileNotSupported` preflight stays unchanged for HOST targets; the WASM target IS by definition a cross-compile (GOOS=js GOARCH=wasm) and is permitted from every host OS — this preserves plan-008 U4's invariant exactly.
- The `Compiling` phase emission stays for UX continuity: now it covers the player-binary-extraction/cache-hit step (fast) or the developer-fallback `go build` (slow); either way the user sees the phase progress.

**Patterns to follow:**
- Existing `hostLongBuilder.Build` phase emission (`PhaseQueued` → `PhaseGenerating` → `PhaseCompiling` → `PhasePackaging` → `PhaseDone`); even though we're no longer compiling per-cart, keep the phase emission for UX continuity (the "Compiling" phase is now the player-build fallback or a skip-when-cached fast path).
- `pixelforge_studio/buildpipeline/icon_logo.go` for the Windows `.syso` flow — unchanged.
- `builders_shared.go:artifactExt` unchanged.

**Test scenarios:**
- *Happy host build (Linux):* Submit a minimal project; assert output file at `<outputDir>/host/<name>` exists, is executable, and `pixelforge_cart.ReadSelf` against the binary returns the original cart payload.
- *Happy host build (Windows .exe):* Same as above with `<name>.exe`; assert `.syso`-derived icon is embedded; `ReadSelf` returns cart bytes.
- *Happy host build (macOS .app):* Output is `<name>.app/Contents/MacOS/<name>`; `Info.plist` exists; `AppIcon.icns` exists; `ReadSelf` on the inner Mach-O returns cart bytes; binary launches on darwin/arm64 (codesign verified).
- *Happy WASM build:* Submit a minimal project; assert output file at `<outputDir>/wasm/<name>.html` exists, is non-empty, contains both `WASM_B64` and `CART_B64` placeholders filled (regex match on base64 patterns), and contains the click-to-start splash + `__pixelforgeCart` assignment.
- *Cross-compile rejection (host):* `BuildRequest{Target: TargetWindows}` on Linux host → `ErrCrossCompileNotSupported` (preserved from plan-008 U4).
- *WASM cross-platform allowed:* `BuildRequest{Target: TargetWASM}` on Linux host → succeeds (WASM is the universal cross-compile target).
- *Player-binary path 1 (explicit):* Set `PlayerBinaryPath`; verify no cache lookup, no embed extraction, no `go build`.
- *Player-binary path 2 (cache hit with valid checksum):* Pre-populate cache + `.sha256` sidecar; verify cache used without falling through to embed.
- *Player-binary path 2 (cache hit with WRONG checksum):* Pre-populate cache + sidecar with intentionally-wrong `.sha256`; verify cache invalidated + fall through to embed (poisoned-cache protection).
- *Player-binary path 3 (embed extraction):* No explicit path, no cache; verify `playerbins.PlayerBinaryFor` is consulted + extracted to temp + used. This is the no-Go user happy path.
- *Player-binary path 4 (developer fallback):* No explicit path, no cache, no embedded binary (test stub returns ErrNotEmbedded); verify the fallback `go build -tags=long ./cmd/pixelforge-player` runs.
- *Empty project:* Project with no scenes, no sprites; `EncodeCart` returns a valid (small) byte stream; Append succeeds; resulting binary boots and exits gracefully.
- *AE1 + AE2 coverage:* Build → Host on macOS produces a launchable .app; A2 runs the .app on a separate macOS machine with no Pixelforge installation → game launches. Build → WASM produces mygame.html; A2 opens via file:// with no network → splash appears, click starts the game.

**Verification:** `go test -tags=long ./pixelforge_studio/buildpipeline/...` passes; produced binaries from happy-path scenarios are executable + round-trip via `pixelforge_cart.ReadSelf`; the no-Go user path (embedded player binary discovery) runs without `go build`.

---

### U4. Shared RenderTickAt seam

**Goal:** Carve out the single rendering function `pixelforge_render.RenderTickAt(rt, tick, inputs) → *image.RGBA` that the studio preview, the shipped player, and CI replay all call. Eliminates any possibility of preview-vs-shipped pixel drift.

**Requirements:** R12.

**Dependencies:** none (greenfield), but consumed by U2, U5.

**Files:**
- `pixelforge_render/doc.go` (new)
- `pixelforge_render/rendertick.go` (new) — `RenderTickAt(rt *capsuleruntime.Runtime, tick uint64, inputs InputFrame) (*image.RGBA, error)`; `type InputFrame struct{ Keys []string; Pad *GamepadState }`
- `pixelforge_render/rendertick_test.go` (new)
- `pixelforge_studio/editor/preview/preview.go` (modify if exists, else new) — preview window's `Draw` calls `pixelforge_render.RenderTickAt`
- `cmd/pixelforge-player/main.go` (modify, from U2) — Ebitengine `Draw` calls `pixelforge_render.RenderTickAt`

**Approach:**
- `RenderTickAt` is pure-ish: given the runtime + tick + inputs, advance the simulation by one tick, then render to an in-memory `*image.RGBA`. Determinism: no goroutines started by this function, no `time.Now()` reads, no `math/rand` without explicit seeded source.
- Inside: applies `inputs` to the runtime's input layer; invokes the existing `pixelforge_ebiten.Update`-equivalent path; calls the existing draw stack; copies the resulting framebuffer to a fresh `*image.RGBA` via `screen.ReadPixels` (Ebitengine 2.6+).
- For non-test paths where Ebitengine owns the screen surface, `RenderTickAt` returns a reference to that surface (not a copy) for performance. A `RenderTickAtCopy` variant returns a fresh allocation for hashing.
- Cycle-break: `pixelforge_render` imports `capsuleruntime`; `capsuleruntime` does NOT import `pixelforge_render` (no cycle). Preview package imports both; player binary imports both.

**Execution note:** Test-first for the determinism guarantee — write `TestRenderTickAt_BitIdentical` first (same runtime + same inputs → same `*image.RGBA` bytes) before wiring into preview/player.

**Patterns to follow:**
- `docs/solutions/always-on-game-embedding.md` ("one render path" rule) — apply at this new seam.
- Existing `pixelforge_ebiten/run.go` for the Ebitengine integration shape.

**Test scenarios:**
- *Bit-identical replay:* Same `rt + tick + inputs` returns byte-equal `*image.RGBA` pixels across 100 invocations.
- *Tick advances state:* `RenderTickAt(rt, 0, ...)` vs `RenderTickAt(rt, 1, ...)` produces different frames (state advanced).
- *Empty input:* `InputFrame{Keys: nil}` is valid (no panic); frame still renders.
- *Cross-platform smoke:* Compile-test under `GOOS=linux/amd64`, `GOOS=darwin/arm64`, `GOOS=js/wasm` (no runtime test under WASM CI yet — that's a v3 concern).

**Verification:** `go test ./pixelforge_render/...` passes; preview window pixels match player binary pixels for the same `.pforge` + same tick + same inputs (golden-image test fixture).

---

### U5. Input-trace format + replay harness

**Goal:** Create `pixelforge_replay` package with `.trace.jsonl` reader/writer + a `Replayer` that deterministically runs a trace through `pixelforge_render.RenderTickAt` and produces a sequence of `*image.RGBA` frames + recorded `verbs.bus` events for assertion.

**Requirements:** R8, R9.

**Dependencies:** U4.

**Files:**
- `pixelforge_replay/doc.go` (new)
- `pixelforge_replay/trace.go` (new) — `type Trace struct{ Meta TraceMeta; Frames []InputFrame }`; `LoadTrace(r io.Reader) (*Trace, error)`; `(*Trace) Encode(w io.Writer) error`
- `pixelforge_replay/trace_test.go` (new)
- `pixelforge_replay/recorder.go` (new) — `type Recorder struct{ ... }`; records per-tick inputs as the studio preview runs; flushes to `.trace.jsonl`
- `pixelforge_replay/recorder_test.go` (new)
- `pixelforge_replay/replayer.go` (new) — `type Replayer struct{ ... }`; `Run(rt *capsuleruntime.Runtime, trace *Trace) (frames []*image.RGBA, events []piloop.VerbEvent, err error)`
- `pixelforge_replay/replayer_test.go` (new)

**Approach:**
- JSON-lines schema (canonical): first line is `{"v":1,"meta":{...}}`; subsequent lines are `{"tick":N,"keys":[...],"pad":null|{...}}`. Comments not supported (strict JSON-lines).
- Empty-input compression: `{"tick":47,"keys":[],"hold":12}` means "tick 47–58 all have empty input." Optional optimization, decoder accepts both forms.
- `Recorder.Tick(inputs)` appends a frame; `Recorder.Flush()` writes the .jsonl file.
- `Replayer.Run` instantiates a fresh `pixelforge_loop.ResetVerbsBusForTest`-equivalent test bus, subscribes a recording subscriber, iterates the trace, calls `pixelforge_render.RenderTickAt` per tick, captures the bus events emitted during that tick, accumulates frames. Returns the full sequence.
- Determinism guard: `Replayer` does not start any goroutines; audio sinks are stubbed via `Sinks` override (per the existing `capsuleruntime.Options.SinksOverride` seam).

**Patterns to follow:**
- `docs/solutions/scripting-runtime-design.md`'s `SynthesiseFromInputLog` — the precedent.
- `docs/solutions/ring-buffer-snapshot-store.md`'s Frame schema — paletted canvas + tick + per-tick events.
- `pixelforge_loop.ResetVerbsBusForTest` — reuse for the isolated test bus.

**Test scenarios:**
- *Round-trip encode/decode:* A `Trace` with 5400 frames + mixed inputs encodes to .jsonl and decodes byte-equal.
- *Empty-input compression decode:* Decode a .jsonl with `"hold":12` runs; resulting frames match the expanded form tick-by-tick.
- *Replayer determinism:* Same trace + same runtime → identical frame sequence across 10 runs.
- *Bus-event capture:* A trace that triggers `motion/jump` produces a recorded `*VerbEvent{Topic: "motion/jump", ...}` in the captured events slice at the expected tick.
- *Malformed line:* A .jsonl with a non-JSON line returns `ErrMalformedTrace` with the line number.
- *Version mismatch:* `"v":2` returns `ErrTraceVersion` cleanly (forward compat hook).

**Verification:** `go test ./pixelforge_replay/...` passes; an end-to-end smoke test in `pixelforge_studio/integration_test/` records 60 ticks against a synthetic project and replays back to identical frames.

---

### U6. pixelforge_physics substrate (with early cross-CPU determinism probe)

**Goal:** Create the `pixelforge_physics` package: adopt `solarlune/resolv` v0.8.1 as the substrate; build a thin layer for tile-grid AABB, gravity integrator, one-way platforms, ladder mechanics, screen-wrap, and LUT-based trig. **Critically, validate the cross-CPU determinism assumption BEFORE any reference-game baselines are recorded** — the entire pixel-hash CI strategy depends on it.

**Execution note:** Run the cross-CPU determinism probe FIRST, before tilemap/gravity/oneway/ladder feature work. Write a synthetic-scene test that exercises the integrator + resolv collision over N=1000 ticks; run it on `ubuntu-latest amd64` AND `macos-latest arm64` CI legs; compare body positions byte-equal. **If positions diverge**, decide the determinism strategy NOW (before U11 records any baselines): (a) downgrade pixel-hash CI to single-CPU only — record on `ubuntu-latest amd64`, run only there; (b) move to tolerance-based comparison (per-checkpoint position tolerance + perceptual frame hash instead of SHA-256); (c) replace resolv's internal float64+Sqrt SAT with hand-rolled integer-only AABB (large scope, defer to U6b). The probe outcome gates U11/U14/U16/U18 CI strategy; document the decision in `pixelforge_physics/doc.go` and the Key Technical Decisions section.

**Requirements:** R5, R6.

**Dependencies:** none (greenfield).

**Files:**
- `pixelforge_physics/doc.go` (new)
- `pixelforge_physics/world.go` (new) — `type World struct{ *resolv.Space; cfg PhysicsConfig }`; `NewWorld(cfg) *World`
- `pixelforge_physics/world_test.go` (new)
- `pixelforge_physics/tilemap.go` (new) — `BuildTileGridCollider(atlas *pixelforge_project.TileAtlas, solidIDs []int) []*resolv.ConvexPolygon`; hand-written 16×16 AABB-vs-tilemap sweeper alongside resolv
- `pixelforge_physics/tilemap_test.go` (new)
- `pixelforge_physics/gravity.go` (new) — `Integrate(body *Body, dt Fixed32)`; jump impulse helper
- `pixelforge_physics/gravity_test.go` (new)
- `pixelforge_physics/oneway.go` (new) — `IsOneWayHit(mtv resolv.Vector, vel resolv.Vector) bool`
- `pixelforge_physics/ladder.go` (new) — climbing state machine
- `pixelforge_physics/trig.go` (new) — `var sinLUT = [65536]int32{...}`; `SinDeg(uint16) Fixed32`; CosDeg; AtanDeg
- `pixelforge_physics/trig_test.go` (new)
- `go.mod` (modify) — add `github.com/solarlune/resolv v0.8.1`

**Approach:**
- `type PhysicsConfig struct{ Gravity Vec2; Drag float32; ScreenWrap bool; LadderAware bool; CollisionMode string; ... }` — per-cart presets configure these. Asteroids = `{Gravity: 0, ScreenWrap: true}`; Mario = `{Gravity: (0, 980), LadderAware: false}`; Bomberman = `{Gravity: 0, CollisionMode: "grid-aabb"}`; DK = `{Gravity: (0, 980), LadderAware: true}`.
- LUT generation: a small `go:generate` directive in `trig.go` runs a one-shot helper that writes a binary blob alongside (or computes at `init()` time on first import — measure cost).
- One-way platforms: `IsOneWayHit` returns true iff `mtv.Y < 0` AND `vel.Y >= 0` (player moving down, MTV pushing up out of the platform).
- Tilemap collider: scans `Scene.TileAtlases[*].Grid` for solid tile IDs, emits `*resolv.ConvexPolygon` rectangles at the right pixel coordinates, adds them all to `World.Space`.
- Fixed-point arithmetic: define `type Fixed32 int32` representing 16.16 fixed-point; ops `+`, `-`, `*` (with shift), `/`. Use for ALL physics math to guarantee determinism across CPUs.

**Patterns to follow:**
- `solarlune/resolv` examples in `pixelforge_studio/integration_test/`-style — wrap, don't expose.
- `docs/solutions/ring-buffer-snapshot-store.md`'s Frame schema — the World state must be snapshot-serializable.

**Test scenarios:**
- *AABB-vs-tile resolve:* Player rectangle at `(8, 8)` moving down 4px; bottom row of tile (16-pixel tile starts at y=16) is solid; player snaps to y=8 (no penetration); ground state = grounded.
- *One-way platform:* Player moving down through one-way platform → resolves to land on top. Player moving up through same platform → passes through.
- *Slope tile (right-triangle):* Player on a /-slope at x=8 sees y resolve to slope height; player on .-slope passes through.
- *Gravity integrator:* `Integrate` for 60 ticks on a body with `vel.Y = 0` under gravity `980 px/s²` at 60 TPS produces `vel.Y ≈ 980` after 1 second of simulation.
- *Screen-wrap:* Body at `x = 321` on a 320-wide screen with `ScreenWrap: true` → wraps to `x = 1`.
- *LUT trig accuracy:* `SinDeg(0)` = 0; `SinDeg(90*256)` = 1.0 (in Fixed32); error vs `math.Sin` is < 0.0001.
- *Cross-CPU determinism probe (LOAD-BEARING):* Same World state + same input sequence → body positions after 1000 ticks across `ubuntu-latest amd64` + `macos-latest arm64`. Test asserts position equality at a tolerance level chosen empirically:
  - **First run:** byte-equal (preferred outcome).
  - **If byte-equal fails:** measure observed drift; commit to tolerance buckets (e.g., ±1 sub-pixel position; perceptual hash for frames) OR single-CPU CI; document outcome.
  - The test name carries the strategy: `TestDeterminism_ByteEqual_AcrossCPU` (preferred), or fallback `TestDeterminism_ToleranceBucket_AcrossCPU`. Whichever variant passes is committed; the failing one is `t.Skip`'d with a reason.
- *Same-CPU determinism (always required):* Same World + same inputs → byte-equal positions across 10 invocations on the SAME runner. This guarantees the within-CI-leg invariant for U11+ pixel-hash baselines.

**Verification:** `go test ./pixelforge_physics/...` passes; cross-CPU determinism probe runs on the long-tag CI matrix; the outcome (byte-equal / tolerance / single-CPU) is documented in `pixelforge_physics/doc.go` and Key Technical Decisions before U11 lands.

---

### U7. Engine-level sinks concrete: scene + visual + spawn + dialogue + music

**Goal:** Replace `logSceneController`, `logVisualSink`, `logSpawnSink`, `dialogueController` (log-only path), and `logMusicSink` in `capsuleruntime/subscribers.go` with concrete implementations that mutate runtime state.

**Requirements:** R5.

**Dependencies:** none (independent of physics).

**Files:**
- `pixelforge_studio/capsuleruntime/subscribers.go` (modify) — replace stub implementations
- `pixelforge_studio/capsuleruntime/scene_sink.go` (new) — `type sceneSink struct{ rt *Runtime }`; implements `change(name)`, `restart()`, `wait(ms)`
- `pixelforge_studio/capsuleruntime/visual_sink.go` (new) — `type visualSink struct{ rt *Runtime }`; implements `hide(entity)`, `show(entity)`, `flash(entity, color, ms)`, `swap_sprite(entity, name)`
- `pixelforge_studio/capsuleruntime/spawn_sink.go` (new) — `entitySpawn(prefab, x, y)`, `destroySelf(entity)`, `destroyOther(entity)`
- `pixelforge_studio/capsuleruntime/dialogue_sink.go` (new) — `openDialogue(scriptID)`, `closeDialogue()` rendering into the active scene's UI layer
- `pixelforge_studio/capsuleruntime/music_sink.go` (new) — wraps `pixelforge_audio.PlayBGM(name)` / `StopBGM()`; reuses existing `realAudioSink` infrastructure
- `pixelforge_studio/capsuleruntime/subscribers_test.go` (modify) — add scenarios for new concrete sinks

**Approach:**
- Each new sink type implements the matching interface defined in `subscribers.go` (`SceneController`, `VisualSink`, `SpawnSink`, `DialogueSink`, `MusicSink`).
- `sceneSink.change(name)` looks up the scene in `rt.Project.Scenes`, swaps `rt.CurrentScene`, fires a `scene/changed` event on `verbs.bus` for any chained handlers.
- `visualSink.flash(entity, color, ms)` queues a per-tick palette override in the runtime's render state; expires automatically after `ms` ticks.
- `spawnSink.entitySpawn` looks up a prefab by name (extend `Project` with a `Prefabs` registry — schema-additive), creates a new entity in the current scene with the prefab's component template.
- `dialogueSink.openDialogue(scriptID)` pushes the dialogue UI onto `rt.ModalStack` (per `docs/solutions/focus-manager-design.md`).
- `musicSink.playMusic(name)` resolves to a registered BGM track from `pixelforge_audio.Lookup(name)` and starts playback; `stopMusic` halts the channel cleanly.

**Patterns to follow:**
- Existing `realAudioSink` (subscribers.go:490) — the only fully concrete sink today. Mirror its shape.
- `docs/solutions/dirty-state-ux.md` — route any "potentially-destructive" sink operations (scene change without save, etc.) through `PromptIfDirty` where the studio preview path is concerned (player path bypasses since user has no save-buffer to lose).
- `pixelforge_audio.Play` for the music sink's audio mechanic.

**Test scenarios:**
- *Scene change:* Fire `scene/change Args{name: "level_2"}` against a project with two scenes; assert `rt.CurrentScene.ID == "level_2"` post-dispatch.
- *Scene restart:* Fire `scene/restart`; assert entities reset to their `Scene.Entities` initial state.
- *Scene wait:* Fire `scene/wait Args{ms: 500}`; advance 30 ticks (at 60 TPS); assert no further dispatch happens during wait.
- *Visual flash:* Fire `visual/flash Args{entity:"hero", color:"#ff0000", ms:100}`; assert the per-entity render override is set; advance 6 ticks; assert it's still set; advance 7 more; assert it's cleared.
- *Spawn entity:* Fire `spawn/entity Args{prefab:"goomba", x:64, y:128}`; assert the current scene's entity count incremented by 1 and the new entity's position matches.
- *Destroy self:* Fire `spawn/destroy_self Args{entity:"goomba_3"}`; assert the entity is removed from the scene.
- *Dialogue open/close:* Fire `ui/open_dialogue Args{script:"intro"}`; assert `rt.ModalStack.Top()` is a dialogue; fire `ui/close_dialogue`; assert the stack popped.
- *Music play/stop:* Fire `audio/play_music Args{name:"level_theme"}`; assert `pixelforge_audio.CurrentBGM() == "level_theme"`; fire `audio/stop_music`; assert nil.
- *Unknown scene name:* Fire `scene/change Args{name:"nonexistent"}`; assert log warning + no scene mutation (graceful).
- *Covers AE3 (jump fires concrete sink, not log).* The scene/visual sinks are part of the broader "no `log.Printf` motion stub fires" assertion in AE3 — the motion sink (U8) is the specific one for jump.

**Verification:** `go test ./pixelforge_studio/capsuleruntime/...` passes; `grep -c "log.Printf" pixelforge_studio/capsuleruntime/subscribers.go` ≤ 1 (only intentional debug logs remain).

---

### U8. Motion sink concrete: apply_thrust + rotate_entity + screen_wrap

**Goal:** Replace `logMotionSink`'s `apply_thrust`, `rotate_entity`, and `screen_wrap` topic dispatch with concrete implementations backed by `pixelforge_physics`. These are the three motion primitives Asteroids needs.

**Requirements:** R5, R6.

**Dependencies:** U6.

**Files:**
- `pixelforge_studio/capsuleruntime/subscribers.go` (modify) — `logMotionSink.Apply` switch grows concrete handlers for these three topics; falls back to log for the others (jump/gravity land in U13)
- `pixelforge_studio/capsuleruntime/motion_sink.go` (new) — `type motionSink struct{ rt *Runtime; phys *pixelforge_physics.World }`; handlers for the three topics
- `pixelforge_studio/capsuleruntime/motion_sink_test.go` (new)
- `pixelforge_studio/capsuleruntime/runtime.go` (modify) — `Runtime` now owns `*pixelforge_physics.World`; constructed at Boot per `Project.PhysicsConfig`

**Approach:**
- `motion/apply_thrust Args{entity:"ship", direction:90, force:5}`: looks up entity → applies impulse along the angle (using `pixelforge_physics.SinDeg/CosDeg`) to the entity's velocity vector. Velocity then drives position via the World's integrator.
- `motion/rotate_entity Args{entity:"ship", delta:5}`: increments the entity's rotation by `delta` degrees (modulo 360).
- `motion/screen_wrap Args{entity:"asteroid"}`: if the entity's position exceeds the screen bounds, wraps to the opposite edge. World config flag `ScreenWrap` is the gate.
- `Project.PhysicsConfig` extends the schema additively (new optional field; defaults to empty struct which maps to "no-gravity, no-wrap, AABB-only").

**Patterns to follow:**
- `pixelforge_physics/gravity.go`'s `Integrate` is the pattern for any velocity-affecting handler.
- `pixelforge_loop.VerbsBus().Publish` for any downstream events emitted (e.g., `motion/wrapped` notification for trace replay).

**Test scenarios:**
- *Apply thrust:* Fire `motion/apply_thrust Args{entity:"ship", direction:0, force:10}`; advance one tick via `pixelforge_physics.World.Step`; assert `entity.Velocity.X ≈ 10` (with thrust along x-axis).
- *Thrust at 90°:* `direction:90` produces `Velocity.Y` change only (using LUT trig).
- *Rotate:* Fire `motion/rotate_entity Args{entity:"ship", delta:90}` three times; assert rotation = 270.
- *Screen-wrap (right edge):* Place entity at `x:319` on a 320-wide screen; fire `motion/screen_wrap`; advance position to `x:321`; assert wrap to `x:1`.
- *Screen-wrap disabled:* Same setup with `PhysicsConfig{ScreenWrap: false}`; entity stays at `x:321` (no wrap).
- *Unknown entity:* Fire any motion topic with `entity:"ghost"`; assert log warning + no mutation.
- *Covers AE3.* Jump-specific assertion lands in U13; this unit covers the rotational + thrust + wrap mechanics for Asteroids.

**Verification:** `go test ./pixelforge_studio/capsuleruntime/...` passes; motion handlers are no longer log-only for these three topics.

---

### U9. Damage sink concrete + explode_radius (point-blast variant)

**Goal:** Replace `logDamageSink` with concrete implementations of `die`, `hurt_player`, `take_damage`, and `explode_radius` (point-blast: an entity explodes with a radial blast that damages all entities within radius, with no special grid alignment).

**Requirements:** R5.

**Dependencies:** U7 (for visual/spawn coordination on explode).

**Files:**
- `pixelforge_studio/capsuleruntime/subscribers.go` (modify) — replace `logDamageSink` references
- `pixelforge_studio/capsuleruntime/damage_sink.go` (new) — concrete handlers
- `pixelforge_studio/capsuleruntime/damage_sink_test.go` (new)

**Approach:**
- `damage/die Args{entity:"hero"}`: marks the entity dead; triggers the entity's `on_death` verb binding if present; eventually removed from scene next tick (via `spawnSink.destroyOther`).
- `damage/hurt_player Args{amount:1}`: decrements `rt.Globals["player_hp"]`; if ≤ 0, fires `damage/die`.
- `damage/take_damage Args{entity:"goomba", amount:1}`: decrements the entity's HP component.
- `damage/explode_radius Args{center_x:128, center_y:96, radius:32, damage:5}`: iterates all entities within `radius` of `(center_x, center_y)`, fires `damage/take_damage` on each; spawns a `visual/flash` on the blast center; plays a `audio/play_sound` SFX.

**Patterns to follow:**
- Damage chain: `take_damage` → if HP ≤ 0 → `die`. Implement as cascade-dispatch through `verbs.bus`, not direct function calls (preserves the observable event sequence for CI parity).

**Test scenarios:**
- *Die:* Fire `damage/die Args{entity:"goomba"}`; assert entity flagged dead; advance one tick; assert removed from scene.
- *Hurt player:* `rt.Globals["player_hp"] = 3`; fire `damage/hurt_player Args{amount:1}`; assert HP=2.
- *HP-to-zero:* Repeat hurt until HP=0; assert subsequent fire triggers `damage/die`.
- *Take damage:* Entity with HP=5; fire `damage/take_damage Args{amount:2}`; assert HP=3.
- *Explode radius hits all within range:* Place 3 entities within radius and 2 outside; fire `damage/explode_radius`; assert only the 3 within-radius take damage.
- *Explode radius spawns visual flash:* Verify `visual/flash` published to verbs.bus during dispatch (recordable for CI parity test).
- *Explode radius plays SFX:* Verify `audio/play_sound` published.
- *Covers AE5 indirectly* (pixel-hash divergence test depends on damage sink correctness; AE5's specific assertion lands in U25's CI gating).

**Verification:** `go test ./pixelforge_studio/capsuleruntime/...` passes; damage cascade emits the expected sequence of events on the bus (recordable + verifiable).

---

### U10. Snapshot save concrete: entity-component reflection serialization

**Goal:** Replace `saveServiceSink`'s `pisave.Snapshot{}` literal-empty serialization with a concrete implementation that walks the current scene's entities, marshals each entity's components map, persists via the existing `pisave.Backend` (native filesystem or WASM localStorage), and round-trips byte-equal across save → load on both backends.

**Requirements:** R7.

**Dependencies:** U7 (for scene access).

**Files:**
- `pixelforge_studio/capsuleruntime/subscribers.go` (modify) — replace `saveServiceSink` line 640+
- `pixelforge_studio/capsuleruntime/snapshot.go` (new) — `EncodeSnapshot(rt *Runtime) ([]byte, error)`; `DecodeSnapshot(rt *Runtime, data []byte) error`
- `pixelforge_studio/capsuleruntime/snapshot_test.go` (new)
- `pixelforge_save/backend.go` (modify, if needed) — already exists; ensure `SaveToSlot(snapshot []byte, slot int)` byte-faithful
- `pixelforge_save/backend_native.go` (modify, if needed)
- `pixelforge_save/backend_js.go` (modify, if needed)

**Approach:**
- Snapshot schema: `{"v":2, "tick":N, "scene":"id", "entities":[{"id":"...","pos":[x,y,z],"components":{...}}], "globals":{...}}`
- Entity component serialization: each entity owns a `map[string]Component` keyed by component type name. Each component implements `json.Marshaler` via the existing type registry. Save = iterate map + marshal each. Load = parse map + reconstruct typed components via registry lookup.
- Stable entity IDs: every entity has a string ID; references between components use IDs, never pointers (per Unity Entities convention surfaced by best-practices research).
- Versioning: schema_version=2 (v1 = the empty plan-008 placeholder); load handles v1 by treating it as empty + warning.
- Round-trip determinism: marshal output is sorted by key (use `json.Marshal` with stable map iteration via slice intermediate); same state → same bytes.
- WASM backend: serialized snapshot stored in localStorage via existing `pisave.BackendJS`; `JSKeyPrefix(gameTitle)` per plan-008 U6.

**Patterns to follow:**
- `docs/solutions/ring-buffer-snapshot-store.md`'s Frame schema — the snapshot extends Frame's component-set into a save-slot context.
- `pisave.NewBackendNative` + `NewBackendJS` from plan-008.
- Entity component reflection: `pixelforge_entity/` package's existing component registry.

**Test scenarios:**
- *Round-trip on native backend:* Save snapshot S; load to fresh runtime; encode again; assert byte-equal across save→load→encode.
- *Round-trip on WASM backend:* Same as above with `BackendJS` (test under `//go:build js` GOOS=js GOARCH=wasm via headless harness OR mock-localStorage).
- *Schema v1 forward-compat:* Decode a stored v1 (empty) snapshot; assert no panic + warning logged.
- *Entity reference preservation:* Save → load → assert entity references between components (e.g., `Inventory.holder = "player_1"`) resolve to the right entity.
- *Position byte-equal:* Save with `Position{1.5, 2.5, 0}` → load → assert position field is bit-identical `1.5, 2.5, 0` (no float drift).
- *Lives count byte-equal:* `Globals["lives"] = 3` → save → load → 3.
- *DK ladder-climb state byte-equal:* Save mid-climb (entity in "climbing" sub-state) → load → climbing state restored.
- *Covers AE4.* AE4's full assertion (save → reset → load → byte-equal) is the canonical scenario.

**Verification:** `go test ./pixelforge_studio/capsuleruntime/...` passes; AE4 expressed as a test case asserting full DK-style state round-trip.

---

### U11. Asteroids proof fixture + trace + CI replay test

**Goal:** Create the Asteroids proof: `.pforge` fixture, ~90s recorded input trace, baseline pixel-hashes + bus event sequence, and a CI test that loads the fixture, replays the trace through `pixelforge_render.RenderTickAt`, and asserts parity. Asteroids is Phase 1's proof game because it uses only thrust + rotate + screen_wrap + spawn/destroy + explode_radius (point-blast) — no platformer physics needed.

**Requirements:** R8, R9, R12.

**Dependencies:** U1–U10.

**Files:**
- `pixelforge_studio/integration_test/fixtures/asteroids_proof.pforge` (new)
- `pixelforge_studio/integration_test/fixtures/asteroids_proof.trace.jsonl` (new)
- `pixelforge_studio/integration_test/fixtures/asteroids_proof.baseline.json` (new) — pixel hashes (one per checkpoint frame at ticks 0, 60, 180, 360, 1800, 3600, 5400) + verbs.bus event sequence reference
- `pixelforge_studio/integration_test/asteroids_proof_test.go` (new) — `//go:build long`
- `pixelforge_studio/integration_test/fixtures/assets/asteroids/` (new) — CC0 placeholder sprites: ship, asteroid (large/medium/small), bullet; one BGM, one explosion SFX

**Approach:**
- Author Asteroids in the studio (manually for the first proof — this is the smoke test for the studio + runtime as a whole): ship that rotates + thrusts + wraps the screen; 4 large asteroids that split into 2 medium when shot, mediums split into 2 small, smalls vanish on hit; player dies on collision; "all asteroids destroyed" wins. Use `apply_thrust + rotate_entity + screen_wrap + spawn/destroy + explode_radius`.
- Record a 90s trace using `pixelforge_replay.Recorder` during studio preview: the trace must end with all asteroids destroyed.
- Capture baseline: run `pixelforge_replay.Replayer` once; SHA-256 the framebuffer at 7 checkpoint ticks; capture the full verbs.bus event sequence; write to `asteroids_proof.baseline.json`.
- CI test: load fixture → load trace → load baseline → `Replayer.Run` → assert checkpoint frame hashes match → assert bus event sequence matches.

**Patterns to follow:**
- Existing `mario_strip_scene.pforge` fixture for `.pforge` schema; this is a richer scene with more entities + a verb-sheet-driven game loop.
- `pixelforge_studio/integration_test/build_pipeline_long_test.go` for the `//go:build long` test shape.

**Test scenarios:**
- *Fixture loads:* `pixelforge_project.LoadReader` against the .pforge returns a valid project with the expected entity count.
- *Trace loads:* `pixelforge_replay.LoadTrace` returns a `*Trace` with 5400 frames (90s × 60TPS).
- *Replay produces baseline:* `Replayer.Run` produces frames matching baseline checkpoint hashes.
- *Bus events match baseline:* The recorded event sequence is byte-equal to baseline.
- *Game completion:* At end of trace, scene state shows all asteroids destroyed (entity count of asteroid archetype == 0).
- *Pixel-hash regression detector:* Manually introduce a one-pixel-off bug in U6 physics (off-by-one in screen-wrap); rerun this test; assert it FAILS at the affected checkpoint. (This is the AE5 negative test — verifies CI catches the regression. Restore the bug after asserting failure.)
- *Covers AE3, AE5.*

**Execution note:** This unit is the first time the full Phase 1 stack runs end-to-end. Expect physics tuning iterations. Once stable, the baseline is checked in and the test gates every commit.

**Verification:** `go test -tags=long ./pixelforge_studio/integration_test -run TestAsteroidsProof` passes; introducing a deliberate physics regression makes it fail at the right checkpoint.

---

### U12. Build-host studio integration smoke test

**Goal:** Wire the new universal-player Build → Host path into the studio UX (the existing `buildpipeline.BuildHost()` / `BuildWASM()` two-button UI). Smoke-test that pressing Build in the studio against the Asteroids proof produces a working `.exe`/`.app`/`.bin` that runs Asteroids end-to-end.

**Requirements:** R1, R2, R3, AE1, AE2.

**Dependencies:** U3, U4, U11.

**Files:**
- `pixelforge_studio/build/workspace.go` (modify) — wire Build buttons to `pixelforge_cart`-backed builders; success toast + output path
- `pixelforge_studio/build/workspace_test.go` (modify or new) — smoke test
- `pixelforge_studio/integration_test/build_pipeline_universal_test.go` (new) — `//go:build long`; end-to-end studio→cart→run

**Approach:**
- Build → Host on the Asteroids proof: codegen.EncodeCart → cart bytes → resolve pre-built player or build-from-source → Append → output `.exe`/etc.
- Studio toast: "Built mygame.app (12.3MB)" with a click-to-reveal-in-finder action.
- Test harness: run the built binary in a subprocess with `--smoke-tick=60` flag (a debug flag added to `cmd/pixelforge-player/main.go` in U2 that runs 60 ticks then exits), verify exit code 0 + a stdout marker.

**Test scenarios:**
- *Build → Host on Linux:* Build Asteroids proof on Linux host → resulting binary at `<outDir>/host/asteroids[.bin]` → run it with smoke flag → exit 0.
- *Build → WASM:* Build Asteroids proof → `<outDir>/wasm/asteroids.html` → file is non-empty + contains the inlined base64 wasm + cart.
- *Build → Host (macOS):* Same as Linux, `.app` bundle structure validated (`Contents/MacOS/asteroids`, `Contents/Info.plist`, `Contents/Resources/AppIcon.icns`).
- *Build → Host (Windows):* Cross-compile rejected with `ErrCrossCompileNotSupported` on non-Windows host.
- *No Pixelforge installation needed:* Copy the binary to a clean temp dir with no Go toolchain visible (PATH cleaned); execute; verify runs (proves no dynamic-link dependency on the studio).
- *Covers AE1, AE2.*

**Verification:** `go test -tags=long ./pixelforge_studio/integration_test -run TestBuildPipelineUniversal` passes; manual run from the studio UI produces working artifacts.

---

### Phase 2: Plumber-style platformer proof

### U13. Platformer motion sinks: jump + apply_gravity + solid_collide

**Goal:** Add concrete `jump`, `apply_gravity`, and `solid_collide` handlers to `motionSink`, backed by `pixelforge_physics.gravity` + `pixelforge_physics.tilemap`. These are the platformer essentials Mario needs.

**Requirements:** R5, R6, AE3.

**Dependencies:** U6, U8, U10.

**Files:**
- `pixelforge_studio/capsuleruntime/motion_sink.go` (modify) — new handlers
- `pixelforge_studio/capsuleruntime/motion_sink_test.go` (modify)
- `pixelforge_physics/gravity.go` (modify if needed) — refine integrator for the Mario curve

**Approach:**
- `motion/jump Args{entity:"hero", strength:5}`: if entity is grounded (`physics.IsGrounded(entity)`), applies upward impulse (`vel.Y -= strength`); transitions to "airborne." Otherwise no-op.
- `motion/apply_gravity Args{entity:"hero"}`: `physics.Integrate(entity.Body, dt)` — adds `gravity * dt` to `vel.Y`; updates position; resolves tile collisions via `physics.World.Step`. Grounded state set if downward MTV resolves.
- `collision/solid_collide Args{entity:"hero"}`: catch-all explicit collision check fire-and-forget; mostly fires via the integrator's own collision resolution but available as a verb for entity-vs-entity edge cases.

**Patterns to follow:**
- The Asteroids motion sink shape from U8.
- `pixelforge_physics`'s World.Step pattern.

**Test scenarios:**
- *Jump from grounded:* Entity grounded; fire `motion/jump Args{strength:5}`; assert `vel.Y = -5` post-dispatch; grounded → airborne.
- *Jump while airborne:* Entity airborne; fire jump; assert no velocity change (no double-jump).
- *Gravity accumulation:* Place entity at `y:50`, vel.Y=0, gravity 980/60²; advance 60 ticks via apply_gravity; assert `vel.Y ≈ 16.3` (980/60).
- *Solid collide stops fall:* Entity falling toward a solid tile; advance until collision; assert position snaps to tile-top + grounded=true.
- *Standing on platform:* Entity grounded on a tile; fire apply_gravity; assert vel stays at 0 (gravity zeroed-out by ground resolution).
- *Variable jump (hold-button):* `Args{strength:5}` at tick 0, second jump at tick 5 while still airborne → ignored. (Variable-jump-height would require a separate mechanic; out of scope here.)
- *Covers AE3.* "Player sprite jumps with the gravity preset's curve and lands on the solid-tile platform below. No `log.Printf` motion stub fires."

**Verification:** `go test ./pixelforge_studio/capsuleruntime/...` passes; AE3 expressed as a test case driving the full jump → fall → land sequence.

---

### U14. Mario proof fixture + trace + CI replay test

**Goal:** Mirror U11 for Mario: `.pforge` fixture, ~90s trace clearing level 1-1, baseline checkpoint hashes + bus events, CI test under `//go:build long`.

**Requirements:** R8, R9, R12.

**Dependencies:** U13.

**Files:**
- `pixelforge_studio/integration_test/fixtures/mario_proof.pforge` (new)
- `pixelforge_studio/integration_test/fixtures/mario_proof.trace.jsonl` (new)
- `pixelforge_studio/integration_test/fixtures/mario_proof.baseline.json` (new)
- `pixelforge_studio/integration_test/mario_proof_test.go` (new) — `//go:build long`
- `pixelforge_studio/integration_test/fixtures/assets/mario/` (new) — CC0 sprites: hero, goomba, ground/brick tiles, pipe, coin; level music + jump SFX

**Approach:**
- Author Mario 1-1 in the studio (mirrors NES level layout): goombas patrol left/right; hero jumps + lands on goombas + lands on pipes; clears level by reaching right-edge flag.
- Trace covers full level clear including 2-3 goomba stomps + 1 pipe jump.
- Baseline checkpoints at ticks {0, 60, 600, 1800, 3600, 5400}.

**Test scenarios:**
- *Fixture loads, trace loads.*
- *Replay produces baseline:* All checkpoint frame hashes match.
- *Bus events match baseline.*
- *Level clear:* End of trace, hero position past the flag entity.
- *Goomba interactions:* Bus events show `damage/die Args{entity:"goomba_N"}` at expected ticks (hero lands on goomba).
- *Pixel-regression detector:* Introduce a one-pixel off-by-one in gravity integrator; rerun; assert FAILS at checkpoint 600 (mid-level platforming).

**Verification:** `go test -tags=long ./pixelforge_studio/integration_test -run TestMarioProof` passes.

---

### Phase 3: Bomberman proof

### U15. Grid-bomber sinks: place_on_grid + grid-explode_radius + bomb timer

**Goal:** Add concrete `place_on_grid` motion handler + grid-aware `explode_radius` variant + bomb-timer mechanism. Bomberman needs entities snapped to a grid + bomb explosions that radiate in cardinal directions along grid lines (not the point-blast variant from U9).

**Requirements:** R5.

**Dependencies:** U8, U9.

**Files:**
- `pixelforge_studio/capsuleruntime/motion_sink.go` (modify) — add `place_on_grid` handler
- `pixelforge_studio/capsuleruntime/damage_sink.go` (modify) — overload `explode_radius` for grid variant (or add `damage/grid_explode`)
- `pixelforge_studio/capsuleruntime/bomb_timer.go` (new) — entity timer system: fire `damage/grid_explode` when countdown hits zero
- `pixelforge_studio/capsuleruntime/bomb_timer_test.go` (new)

**Approach:**
- `motion/place_on_grid Args{entity:"bomb", grid_x:5, grid_y:7}`: snap entity position to `(grid_x * tileW, grid_y * tileH)`; mark entity as grid-locked.
- `damage/grid_explode Args{origin:{gx:5,gy:7}, range:3}`: emits blast in 4 cardinal directions; stops at solid tile or after `range` tiles. Each affected tile fires `damage/take_damage` against any entity standing on it.
- Bomb timer: a generic timer component on entities (`{type:"BombTimer", ticks_remaining:120}`); decrements each Update tick; on zero, fires `damage/grid_explode` from the entity's grid position.

**Test scenarios:**
- *Place on grid:* Fire `motion/place_on_grid Args{entity:"bomb", grid_x:3, grid_y:5}` on a 16×16 grid; assert position = (48, 80).
- *Grid explode 4-way:* Bomb at grid (5,5), range 3, no walls; assert 12 tiles affected (3 in each direction).
- *Grid explode blocked by wall:* Place a solid tile at (5,6); explode bomb at (5,5); assert blast cut at (5,6); 6 tiles still affected (3 east, 3 west, blocked north/south at 1).
- *Bomb timer countdown:* Bomb with timer=120; advance 120 ticks; assert `damage/grid_explode` fires at tick 120 (relative).
- *Bomb timer cancel:* Bomb timer=120; player picks up bomb (some other verb) at tick 50; advance 70 more ticks; assert explode does NOT fire (timer cancelled).

**Verification:** `go test ./pixelforge_studio/capsuleruntime/...` passes.

---

### U16. Bomberman proof fixture + trace + CI replay test

**Goal:** Mirror U11 for Bomberman.

**Requirements:** R8, R9, R12.

**Dependencies:** U15.

**Files:**
- `pixelforge_studio/integration_test/fixtures/bomberman_proof.pforge` (new)
- `pixelforge_studio/integration_test/fixtures/bomberman_proof.trace.jsonl` (new)
- `pixelforge_studio/integration_test/fixtures/bomberman_proof.baseline.json` (new)
- `pixelforge_studio/integration_test/bomberman_proof_test.go` (new) — `//go:build long`
- `pixelforge_studio/integration_test/fixtures/assets/bomberman/` (new) — bomber sprite, bomb, wall tile, breakable tile, explosion sprite; BGM + bomb SFX

**Approach:**
- Bomberman level: 11×11 grid (NES classic); hero starts top-left; mix of solid + breakable walls; goal = survive a placed bomb's blast (move out of range) + clear all breakable walls.
- Trace: place bomb, move 2 tiles away, wait for blast, survive; repeat to clear the screen.
- Baseline checkpoints around bomb-placement + blast tick.

**Test scenarios:**
- *Fixture + trace load.*
- *Replay matches baseline.*
- *Survive blast:* Hero alive at end of trace (entity not in dead state).
- *Bomb explode events:* Bus events sequence includes `damage/grid_explode` at expected ticks.

**Verification:** `go test -tags=long ./pixelforge_studio/integration_test -run TestBombermanProof` passes.

---

### Phase 4: Donkey Kong proof

### U17. Donkey Kong motion sinks: ladder_climb + barrel_roll + multi-screen scroll

**Goal:** Add concrete `ladder_climb` (climb up + jump off + fall when reaching ground) and `barrel_roll` (downward-rolling barrel pattern) handlers. Plus multi-screen vertical scroll for DK's tall levels.

**Requirements:** R5.

**Dependencies:** U13.

**Files:**
- `pixelforge_studio/capsuleruntime/motion_sink.go` (modify) — `ladder_climb`, `barrel_roll` handlers
- `pixelforge_physics/ladder.go` (modify) — refine climbing state machine
- `pixelforge_physics/barrel.go` (new) — barrel rolling pattern (gravity + horizontal sweep + bounce on platforms)
- `pixelforge_render/rendertick.go` (modify if needed) — multi-screen vertical scroll camera

**Approach:**
- `motion/ladder_climb Args{entity:"hero", direction:"up"}`: if entity is touching a ladder tile, transitions to "climbing" state; vertical movement now controlled by ladder_climb intent (gravity disabled); horizontal locked. "down" descends; "off" releases.
- `motion/barrel_roll Args{entity:"barrel_3"}`: applies horizontal velocity + gravity per tick; barrel bounces off platform edges (changes direction on collision); falls off end of platform.
- Multi-screen scroll: `Project.Scenes[*].GridHeightScreens > 1` means the scene exceeds one screen-height; camera follows hero vertically with offset.

**Test scenarios:**
- *Climb up ladder:* Hero touching ladder; fire `motion/ladder_climb direction:"up"`; advance ticks; assert hero ascends + gravity disabled.
- *Climb off ladder:* Hero climbing past ladder top; assert grounded on platform above + state=normal.
- *Fall off ladder:* Hero releases ladder mid-air; gravity re-enabled; falls.
- *Barrel rolls along platform:* Place barrel on platform; fire `motion/barrel_roll`; advance; assert barrel moves horizontally + falls off edge.
- *Barrel kills hero:* Hero collides with barrel; assert `damage/die` fires.
- *Multi-screen scroll:* Scene with GridHeightScreens=4; hero at y=1000 (off bottom of screen 1); camera offsets so hero is visible mid-screen.

**Verification:** `go test ./pixelforge_studio/capsuleruntime/...` + `go test ./pixelforge_physics/...` passes.

---

### U18. Donkey Kong proof fixture + trace + CI replay test

**Goal:** Mirror U11 for Donkey Kong.

**Requirements:** R8, R9, R12, AE4.

**Dependencies:** U17, U10.

**Files:**
- `pixelforge_studio/integration_test/fixtures/donkey_kong_proof.pforge` (new)
- `pixelforge_studio/integration_test/fixtures/donkey_kong_proof.trace.jsonl` (new)
- `pixelforge_studio/integration_test/fixtures/donkey_kong_proof.baseline.json` (new)
- `pixelforge_studio/integration_test/donkey_kong_proof_test.go` (new) — `//go:build long`; ALSO exercises AE4 (save-load round-trip mid-game)
- `pixelforge_studio/integration_test/fixtures/assets/donkey_kong/` (new) — climber sprite, kong sprite, barrel, ladder tile, platform tile

**Approach:**
- DK level: 4-screen-tall vertical climb; hero climbs ladders + dodges rolling barrels + reaches top platform.
- Trace covers full climb to top.
- Extra test scenario: at tick 1800 (mid-climb), call `save_now`; reset runtime; `load_slot`; verify scene state byte-equal (AE4 assertion).

**Test scenarios:**
- *Fixture + trace load.*
- *Replay matches baseline.*
- *Climb completion:* Hero at top of level at end of trace.
- *Save-load round-trip mid-game (AE4):* Save at tick 1800 → reset → load → entity positions + ladder-climb state + barrel positions + lives count all byte-equal.

**Verification:** `go test -tags=long ./pixelforge_studio/integration_test -run TestDonkeyKongProof` passes.

---

### Phase 5: Ship-loop completion + UX polish

### U19. Wire ingest dispatcher Sprite/SFX/BGM/OGG/MP3 runners

**Goal:** Wire `ingestDispatcher.SetSpriteRunner`, `SetSFXRunner`, `SetBGMRunner` in `pixelforge_studio/main.go` (currently only tests call them). Declare `OGGImportRunner` / `MP3ImportRunner` mirroring the existing `PNGImportRunner` / `WAVImportRunner` cycle-break pattern.

**Requirements:** R17, AE7.

**Dependencies:** none (independent).

**Files:**
- `pixelforge_studio/main.go` (modify) — call `SetSpriteRunner` / `SetSFXRunner` / `SetBGMRunner` after `palette.RegisterWith` and `audiolib.RegisterWith`
- `pixelforge_studio/editor/audio_import_handler.go` (modify) — extend `WAVImportRunner` with OGG/MP3 variants OR declare separate `OGGImportRunner` / `MP3ImportRunner` interfaces
- `pixelforge_studio/audiolib/workspace.go` (modify) — register OGG/MP3 handlers analogous to WAV
- `pixelforge_studio/ingest/dispatcher_test.go` (modify or new tests for OGG/MP3 classification)

**Approach:**
- Mirror plan-005's cycle-break pattern: interfaces declared in inner package (`editor`); implementations in outer (`audiolib`); main.go does the wiring.
- BGM has no editor-side handler today — extend `audiolib` to register an OGG decoder using `github.com/jfreymuth/oggvorbis` (pure Go, MIT, WASM-safe) and MP3 via `github.com/hajimehoshi/go-mp3` (already in the ecosystem alongside Ebitengine).
- `dispatcher.Ingest(path)` already classifies by extension; verify .ogg / .mp3 dispatch to BGM runner; .wav stays SFX.

**Test scenarios:**
- *Drop PNG → sprite runner fires:* Drop `cat.png`; assert PNGImportRunner.ImportPath invoked; sprite appears in project.
- *Drop WAV → SFX runner fires.*
- *Drop OGG → BGM runner fires + decodes:* OGG decode succeeds; track appears in audio library.
- *Drop MP3 → BGM runner fires + decodes.*
- *Drop unknown extension:* `mystery.xyz` → no runner invoked + log warning.
- *Covers AE7.*

**Verification:** `go test ./pixelforge_studio/ingest/...` + `go test ./pixelforge_studio/audiolib/...` pass; manual drag-drop on the studio window adds the file to the project.

---

### U20. Embedded CC0 starter pack + asset-library manifest publication

**Goal:** Add a small CC0 starter asset set (handful of sprites + 2 SFX + 1 BGM) embedded via `embed.FS` in `pixelforge_studio/starterpack/`. Publish the asset-library manifest at the URL plan-008's downloader points at. Curated library packs (including the four named-game asset sets for R16) live in the same manifest under separate pack entries.

**Requirements:** R14, R15, R16.

**Dependencies:** none.

**Files:**
- `pixelforge_studio/starterpack/embed.go` (new) — `//go:embed all:assets`; `StarterFS() fs.FS`
- `pixelforge_studio/starterpack/assets/sprites/` (new) — 8-12 CC0 placeholder sprites (Kenney pack)
- `pixelforge_studio/starterpack/assets/sfx/` (new) — 2 CC0 SFX
- `pixelforge_studio/starterpack/assets/bgm/` (new) — 1 CC0 BGM track
- `pixelforge_studio/assetlibrary/library.go` (modify) — at bootstrap, register `starterpack.StarterFS()` as the always-available "Starter" pack
- `pixelforge_studio/assetlibrary/manifest.go` (modify) — add `Examples []Example` field alongside `Packs` (for U21's reference-example fetch path)
- `manifest.json` (published artifact, not in repo) — produced by a one-shot CLI tool; documented in `docs/asset-library-manifest.md` (new)
- `docs/asset-library-manifest.md` (new) — instructions for publishing + content guidelines

**Approach:**
- Starter pack is a small handful of Kenney CC0 art (CC0 1.0 license, freely usable). 8-12 sprites covering generic player/enemy/coin/platform tiles; 2 SFX; 1 BGM. Total ~500KB embedded.
- The `Examples` field on the manifest carries `[{id, version, url, sha256, size_bytes, label, description}]` entries — one per reference example (`asteroids_proof.pforge`, etc.).
- Publication: a GitHub Release is created with the manifest + per-pack zip artifacts; release URL goes into `pixelforge_studio/assetlibrary/startup.go`'s default.

**Patterns to follow:**
- Plan-008 U10+U11: `assetlibrary.Manifest` schema; `assetlibrary.Downloader` SHA-256 verification.
- Schema additive: new `Examples` field; old manifests without it parse cleanly via `omitempty`-style defaults.

**Test scenarios:**
- *Embedded starter loads:* On studio first-launch with no network, `Library.Packs()` includes "Starter" with the expected sprite count.
- *Manifest parses with Examples field:* New `manifest.json` parses; `Examples` field populated.
- *Manifest parses without Examples field:* Old `manifest.json` (no `Examples` key) parses cleanly; `Examples` is empty.
- *Downloader fetches + verifies:* Pack download with correct SHA-256 succeeds; with wrong SHA-256 fails with `ErrChecksum`.
- *Covers AE6.*

**Verification:** `go test ./pixelforge_studio/assetlibrary/...` + `go test ./pixelforge_studio/starterpack/...` passes; the published manifest URL returns a valid JSON document.

---

### U21. File → Open Example menu fetching reference games

**Goal:** Add `File → Open Example` to the studio menu. Selecting an example fetches the `.pforge` from the asset-library manifest's `Examples` field, caches it locally under `<cacheDir>/examples/`, and opens it in the editor (the user can then run, edit, or fork into a new project).

**Requirements:** R11, R16.

**Dependencies:** U20.

**Files:**
- `pixelforge_studio/editor/menu.go` (modify) — `File → Open Example` submenu listing manifest.Examples
- `pixelforge_studio/editor/menu_test.go` (modify)
- `pixelforge_studio/assetlibrary/library.go` (modify) — `FetchExample(id) (string, error)` returning cached local path

**Approach:**
- Menu populates from `manifest.Examples`; each entry shows label + size.
- On click: if cached, open immediately; else show "Downloading…" toast + fetch + verify SHA-256 + open.
- "Fork into new project" action: copy the `.pforge` to `<projectsDir>/<example_id>_fork/` + open the copy. The forked project opens with `dirty=false` (the file is already on disk at the fork path); title bar shows the fork path.

**Test scenarios:**
- *Menu populates from manifest.*
- *Cache hit → instant open.*
- *Cache miss → download + open:* No cached file; click "Open Example: Mario"; download progresses; opens on completion.
- *Network failure:* Download fails; toast shows error; menu re-enables for retry.
- *Bad checksum:* Downloaded file has wrong SHA-256; opens NOT performed; toast shows error.
- *Fork → new project:* Fork action creates a new `_fork` directory + opens it.

**Verification:** `go test ./pixelforge_studio/assetlibrary/...` + `go test ./pixelforge_studio/editor/...` passes; manual menu walk produces the expected behavior.

---

### U22. File → New genre starters + input-binding press-to-capture

**Goal:** Add `File → New → Blank Platformer` / `Blank Arcade Shooter` / `Blank Grid Game` / `Blank Ladder Platformer` to the New menu. Each spawns a project pre-wired with the right physics preset + WASD+Space input bindings + auto-snapshot save mode + one placeholder sprite. Also add "press-key-to-bind" capture mode in the input-binding workspace (R20).

**Requirements:** R10, R20, AE8, AE10.

**Dependencies:** U20 (starter sprite from starterpack).

**Files:**
- `pixelforge_studio/editor/menu.go` (modify) — `File → New` submenu with four genre starters
- `pixelforge_studio/editor/templates/blank_platformer.go` (new) — returns a starter `*pixelforge_project.Project` with platformer physics preset
- `pixelforge_studio/editor/templates/blank_arcade_shooter.go` (new) — Asteroids-style physics preset
- `pixelforge_studio/editor/templates/blank_grid_game.go` (new) — Bomberman-style grid physics preset
- `pixelforge_studio/editor/templates/blank_ladder_platformer.go` (new) — DK-style platformer + ladder preset
- `pixelforge_studio/inputbinding/workspace.go` (modify) — add "Capture" button next to each intent
- `pixelforge_studio/inputbinding/workspace_test.go` (modify)

**Approach:**
- Each template returns a hand-coded `*Project` with: one scene, one entity carrying the placeholder sprite + a physics component matching the genre, input bindings pre-set, save config = `AutoSnapshot`, one tile atlas if grid-based.
- Capture mode: clicking "Capture" sets a flag in the workspace; next keypress is recorded as the binding; the dropdown updates.

**Test scenarios:**
- *Blank Platformer creates a runnable project:* `templates.BlankPlatformer()` returns a project; `pixelforge_render.RenderTickAt` against it produces a non-empty frame; entity exists.
- *Capture binding (AE8):* Open workspace; click Capture next to "jump"; emit synthetic Space keypress; assert binding updates to Space.
- *Genre presets are distinct:* Compare physics configs across templates; Platformer has gravity, ArcadeShooter has screen-wrap, GridGame has no gravity + grid mode, LadderPlatformer has ladder-aware.
- *No named-game template (AE10):* `templates.AllNames()` returns exactly the 4 generic genre starters; "Mario", "Asteroids" etc. NOT in the list.

**Verification:** `go test ./pixelforge_studio/editor/templates/...` + `go test ./pixelforge_studio/inputbinding/...` passes.

---

### U23. WASM size reporting + gzip + wasm-opt invocation

**Goal:** Wire the WASM bundler to report HTML size (raw + gzip) in the success toast; warn at 15MB raw / error at 30MB raw (with `--force` override); write `mygame.html.gz` alongside `mygame.html`; invoke `wasm-opt -Oz` when present on the build host (silent skip if absent).

**Requirements:** R21, R22, AE9.

**Dependencies:** U3, U4.

**Files:**
- `pixelforge_studio/buildpipeline/wasm_bundler.go` (modify) — size measurement + warning logic + gzip output
- `pixelforge_studio/buildpipeline/wasm_optimizer.go` (new) — `wasm-opt` invocation; `IsAvailable() bool` + `Optimize(path) error`
- `pixelforge_studio/buildpipeline/wasm_bundler_test.go` (modify)
- `pixelforge_studio/buildpipeline/wasm_optimizer_test.go` (new)
- `pixelforge_studio/build/workspace.go` (modify) — toast message includes size + warning

**Approach:**
- After `BundleWASM` produces `mygame.html`: stat raw size; gzip-compress to `mygame.html.gz`; report both sizes.
- If raw size > 15MB: log warning to build status; toast shows yellow indicator.
- If raw size > 30MB: log error; toast shows red indicator; unless `BuildRequest.ForceLargeWASM == true`.
- `wasm-opt`: check `exec.LookPath("wasm-opt")`; if found, run `wasm-opt -Oz --enable-bulk-memory game.wasm -o game.opt.wasm` BEFORE bundling into HTML.

**Test scenarios:**
- *Small build:* Raw 8MB; no warn; toast shows `mygame.html (8.0MB, gzip 2.5MB)`.
- *Warn threshold (AE9):* Raw 18MB; warn fires; toast shows `mygame.html (18.0MB, gzip 6.4MB)` + warning.
- *Error threshold:* Raw 35MB; error fires; build fails without `--force`.
- *Force override:* Same with `ForceLargeWASM=true`; build proceeds; error→warning.
- *wasm-opt present:* Stub `exec.LookPath` to return a fake wasm-opt path; verify Optimize called.
- *wasm-opt absent:* Stub LookPath to return error; verify Optimize skipped + no build failure.
- *gzip output exists:* Build produces `mygame.html.gz` alongside; gunzip + diff against original = identical.

**Verification:** `go test -tags=long ./pixelforge_studio/buildpipeline/...` passes; AE9 expressed as a test case.

---

### U24. Verb catalog regen + completeness gate

**Goal:** Add `//go:generate` directive to regenerate `docs/verb-catalog.md` from the verb-recipe catalog. CI verifies the doc is up-to-date. Add a completeness gate that flags catalog entries not used by any of the four reference-game traces.

**Requirements:** R18, R19.

**Dependencies:** U11, U14, U16, U18 (all four traces must exist to gate).

**Files:**
- `pixelforge_studio/scripting/catalog/cmd/gendocs/main.go` (new) — reads catalog registry; writes Markdown
- `pixelforge_studio/scripting/catalog/verb_recipes.go` (modify) — add `//go:generate go run ./cmd/gendocs` directive
- `pixelforge_studio/scripting/catalog/cmd/coverage/main.go` (new) — reads all four `*_proof.trace.jsonl`; cross-references verb usage; prints unused verbs
- `docs/verb-catalog.md` (new, generated)
- `.github/workflows/long.yml` (modify, depends on U25) — `go generate ./... && git diff --exit-code` step

**Approach:**
- gendocs walks the catalog's `RegisterStep`/`RegisterAction`/`RegisterCondition` registry; emits per-kind sections with each verb's name, params, default args.
- coverage tool: iterate fixture traces' bus event sequences; mark each event topic; print verbs in catalog never observed.
- Completeness gate: not strict (don't fail CI for unused verbs); informational warning. The brainstorm R19 says "flagged for review" not "removed."

**Test scenarios:**
- *Regen produces stable output:* Run twice; second output byte-equal to first.
- *Regen produces output covering all registered verbs:* Catalog has N verbs; output has N sections.
- *Coverage detects unused verb:* Add a synthetic unused verb; run coverage; assert it appears in unused list.
- *Coverage finds all used verbs:* All Phase 1-4 verbs (jump, gravity, thrust, etc.) appear in the used list.

**Verification:** `go generate ./pixelforge_studio/scripting/catalog/... && git diff --exit-code docs/verb-catalog.md` passes (file is current); coverage tool reports plausible used/unused breakdown.

---

### U25. Capability matrix CI regeneration + long-tag workflow

**Goal:** Create `.github/workflows/long.yml` running the long-tag test suite per commit. Regenerate `docs/reference-games-capability-matrix.md` from CI pass/fail per fixture per recipe. Wire the four `*_proof_test.go` files into the matrix.

**Requirements:** R9, R13, AE5.

**Dependencies:** U11, U14, U16, U18, U24.

**Files:**
- `.github/workflows/long.yml` (new)
- `pixelforge_studio/scripting/catalog/cmd/matrix/main.go` (new) — reads test output JSON; emits Markdown matrix
- `docs/reference-games-capability-matrix.md` (modify, regenerated)

**Approach:**
- Workflow runs on push to main + PRs; sets up Go 1.24.2; runs `go test -tags=long -json ./pixelforge_studio/integration_test/... | tee test-results.json`; runs matrix generator on test-results.json; commits the regenerated matrix back (or uploads as artifact).
- Matrix rows are recipe×game; cells are PASS/FAIL/SKIP based on test sub-test pass/fail.
- CI fails if any `*_proof_test.go` fails (pixel-hash divergence, bus event divergence).

**Test scenarios:**
- *Matrix regen produces stable output:* Run twice on same test-results.json; output byte-equal.
- *Matrix shows pass for all four games:* All cells PASS or N/A; no FAILs.
- *Pixel-hash regression detection (AE5):* Introduce one-pixel-off bug in render path; workflow run fails at the affected game's checkpoint; matrix shows FAIL cell.

**Verification:** Workflow runs cleanly on a synthetic PR; matrix accurately reflects test outcomes.

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Float-determinism breaks cross-CPU replay | High | High | LUT-based trig + fixed-point math; CI matrix on amd64+arm64; test in U6 specifically asserts byte-equal positions across CPUs |
| WASM size exceeds 30MB hard limit | Medium | Medium | Empirical baseline measurement in U23; wasm-opt -Oz; brotli over the wire; click-to-start splash sets correct expectations |
| Cart-append breaks Windows Authenticode | Low (signing out of scope) | Future v3 risk | Document the "append before sign" invariant in `pixelforge_cart/doc.go`; deferred |
| **macOS arm64 binary fails to launch with `killed: 9` after cart-append** | **High** (Apple Silicon is the majority of new Macs) | **Critical** (v2 ships broken on arm64 Macs without mitigation) | U1's `Append` runs `codesign --force --sign - <output>` unconditionally on darwin. Ad-hoc signature is free (no Apple Developer account). Smoke-tested in U1 + U12 on darwin/arm64 CI leg. |
| Cross-CPU pixel-hash baselines diverge (resolv float64, FMA codegen) | Medium-High | High (CI strategy collapses if byte-equal cross-CPU fails) | U6's early-determinism probe runs BEFORE U11 baselines; outcome chooses byte-equal vs tolerance vs single-CPU strategy; documented in `pixelforge_physics/doc.go` + KTD section |
| No-Go A1 user cannot Build → Host (Go toolchain absent) | High | Critical (origin's headline success metric depends on this) | Studio ships per-OS player binaries via `pixelforge_studio/playerbins/` embedded `embed.FS`; U3's player-discovery chain consults the embed before falling back to `go build`. Release process runs `make playerbins` pre-tag to populate `bins/<os>-<arch>/`. |
| Poisoned player-binary cache trojans shipped artifacts | Low (requires local write access) | High (silent supply-chain attack on A2) | SHA-256 sidecar verified on every cache hit; mismatch invalidates entry + falls through; documented in U3. |
| Asset-library manifest tampering (no signature on manifest itself) | Medium (MITM or compromised GitHub Release) | High (downloaded .pforge executes as RCE) | Deferred to v3 with a documented threat model in U20; v2 ships TLS-only fetch + acknowledges the gap in the security section of `docs/asset-library-manifest.md` |
| resolv lacks one-way platforms + slopes | High (known up-front) | Medium | DIY layer in `pixelforge_physics/oneway.go` + ConvexPolygon triangles; covered in U6 |
| Reference-game pixel-hash tests flaky | Medium | High | Use `FilterNearest` everywhere (no GPU rounding); stub audio in tests; checkpoint only at integer-tick boundaries |
| GitHub Release URL changes after publication | Low | Medium | Manifest URL is configurable; default in code; ENV override for forks |
| Boot time regresses from cart parsing | Low | Low | Streaming JSON decode; avoid full-in-memory base64 for large assets (use ZIP variant if needed) |
| Plan-008's existing tests break | Medium | Medium | Each phase preserves backward compatibility; old codegen path stays available via `sourceLongBuilder` |
| Implementing all four games over-runs schedule | High | Low (architectural bet pays at Phase 1) | Phase 1 alone proves the architecture; Phases 2-4 ship asynchronously; CI gates each |
| Studio in-process Play diverges from shipped player | Low (structurally impossible by U4 design) | Critical | U4's single `RenderTickAt` is the only render path; verified by golden-image cross-test |
| Cart payload corrupts (transmission, signing, etc.) | Low | Medium | CRC32 in footer + version magic catch most cases; documented `ErrCorruptCart`/`ErrVersionMismatch` exit codes |

---

## Test Strategy

- **Unit tests:** every new package (`pixelforge_cart`, `pixelforge_physics`, `pixelforge_render`, `pixelforge_replay`) gets a `_test.go` covering its public API. Run under `go test ./...` (no tag).
- **Long-tag integration tests:** the four `*_proof_test.go` files live in `pixelforge_studio/integration_test/` and gate via `//go:build long`. They load fixture + trace + baseline, replay through `RenderTickAt`, assert pixel-hash + bus event parity.
- **Build-pipeline tests:** `build_pipeline_long_test.go` extended to exercise the new copy-player + append-cart path. Existing `build_pipeline_e2e_test.go` (non-long, scaffold) preserved.
- **Cross-CPU determinism CI:** workflow matrix runs the long-tag suite on `ubuntu-latest` (amd64) + `macos-latest` (arm64). Pixel-hash assertions must match across both.
- **WASM build test:** existing wasm build smoke test extended; new test asserts the inlined HTML produces a valid (parseable) document.
- **Race tests:** `go test -race` clean for every modified package.
- **Save round-trip tests:** specifically exercise `pisave.BackendJS` under `GOOS=js GOARCH=wasm` via the existing build-tagged test path.
- **Drag-drop tests:** ingest dispatcher tests already exist; extended for OGG/MP3 classifiers.

---

## Dependencies & Assumptions

- **External libraries to add:**
  - `github.com/solarlune/resolv` v0.8.1 (MIT, pure-Go, WASM-safe) — collision substrate
  - `github.com/jfreymuth/oggvorbis` (MIT, pure-Go) — OGG decode
  - `github.com/hajimehoshi/go-mp3` (Apache-2.0, pure-Go, already in Ebitengine ecosystem) — MP3 decode
- **CC0 assets** (Kenney pack or equivalent): 8-12 sprites + 2 SFX + 1 BGM for the starter pack. License attestation in `docs/credits.md`.
- **GitHub Release** for asset-library manifest + reference-example .pforge files. Publication is a one-shot operation outside this plan's automated work; tracked in `docs/asset-library-manifest.md`.
- **wasm-opt** (from WebAssembly Binaryen) is OPTIONAL on developer machines; silent-skip if absent.
- **Pre-built player binaries shipped with the studio installer.** Per-OS player binaries live in `pixelforge_studio/playerbins/bins/<os>-<arch>/pixelforge-player[.exe|.wasm]` and embed into the studio binary via `//go:embed all:bins`. The release process runs `make playerbins` (cross-compiles via `GOOS=<os> GOARCH=<arch> go build -tags=long ./cmd/pixelforge-player` — pure-Go cross-compile is reliable) BEFORE tagging a studio release. This satisfies the origin's no-Go A1 promise: a user with no Go toolchain can Build → Host on day one because the player binaries are already inside the studio they installed. Cache at `<userCacheDir>/pixelforge/player-cache/` accelerates repeat builds with SHA-256 integrity verification on every hit.
- **Go 1.24.2 toolchain** required ONLY for studio developers (the long-tag build path + the `make playerbins` release step). End users authoring games via the studio do NOT need Go installed; the embedded player binaries handle Build → Host without a local toolchain.
- **Ebitengine 2.9.9** stays the engine version; no upgrade in this plan.

---

## Scope Boundaries

### Deferred for later

- Mac code-signing + notarization
- Windows code-signing (Authenticode)
- Linux `.desktop` / AppImage / Flatpak packaging
- Headless browser smoke test for WASM (wasmbrowsertest / chromedp)
- Reference games beyond the four named (Pac-Man, Frogger, Space Invaders, Tetris) — v3 matrix expansion
- Variable-jump-height (hold-to-rise) — adjacent to U13's jump but explicitly out of v2
- Multiplayer / networking
- Cloud save sync
- iOS / Android targets
- Steam / itch.io publishing integrations

### Outside this product's identity

- Author-by-play (record gameplay → infer recipes)
- Split studio (headless authoring server + thin viewport)
- Content-addressable asset mirror
- Browser-first studio
- Editor-embedded-in-every-shipped-game (we ship player-only)
- Phone-first authoring
- Tracker UI for scene authoring
- Postcard / recipe-card project format change
- LLM-generated reference games
- Cloud build farm
- Steganographic `.pforge.png` carts
- Studio bundling the four named games as built-in content

### Deferred to Follow-Up Work

- Empirical WASM-size threshold retuning post-U23 baseline measurement (separate PR after v2 ships)
- Brotli compression alongside gzip (R22 specifies gzip; brotli is a small follow-up)

---

## Deferred to Implementation

- **Exact cart payload encoding** (ZIP-wrapped vs JSON-with-inline-base64-assets) — implementer chooses at U3 based on size/perf measurement. Both round-trip cleanly through `pixelforge_cart.Append`/`ReadSelf`.
- **Fixed-point precision** in `pixelforge_physics` (16.16 vs 24.8) — implementer measures arithmetic precision needs at U6 and picks. Either is deterministic; tradeoff is value range vs sub-pixel precision.
- **Exact JSON-lines compression scheme** in `.trace.jsonl` (run-length-encoded vs explicit per-tick) — both decode supported; encoder default decided at U5 based on file size measurement.
- **Snapshot wire-format details** in U10 (which component field types map to which JSON shapes) — derived from existing component registry at implementation time.
- **Pre-built player binary cache key** (commit SHA vs Go module hash vs git describe) — implementer picks at U3 based on what's available in the build context.

---

## Success Metrics

- All eight currently-stubbed `log*Sink` types in `pixelforge_studio/capsuleruntime/subscribers.go` are concrete implementations; no `log*Sink` type remains in the player binary at v2 release.
- A user with no Go installed can install pixelforge-studio, open it, choose `File → New → Blank Platformer`, press Play, and see a controllable jumping character in under 5 minutes.
- All four `*_proof_test.go` tests pass in CI under `go test -tags=long ./pixelforge_studio/integration_test/...`.
- `docs/reference-games-capability-matrix.md` is regenerated from CI on every push; reviewers can answer "is Donkey Kong still playable?" by inspecting the latest run.
- A shipped player binary or `.html` runs on a clean machine (no Pixelforge, no Go toolchain) and plays the embedded cart.
- `ce-plan` can produce future arcade-shipping plans without inventing runtime subsystem scope.

---

## Phased Delivery Summary

| Phase | Units | Demoable outcome |
|---|---|---|
| Phase 1 (architectural bet) | U1–U12 | Asteroids playable end-to-end as shipped binary; universal-player + cart-append proven |
| Phase 2 (platformer physics) | U13–U14 | Mario 1-1 clearable; jump + gravity + solid_collide concrete |
| Phase 3 (grid-bomber) | U15–U16 | Bomberman screen clearable; grid placement + grid-explode concrete |
| Phase 4 (DK climbing) | U17–U18 | Donkey Kong climbable; ladder_climb + barrel_roll + multi-screen scroll concrete; AE4 save-load round-trip green |
| Phase 5 (ship-loop completion) | U19–U25 | Drag-drop wired; starter pack + manifest live; menu templates + Open Example present; WASM size reporting; verb catalog regen; CI matrix regen |

Phase 1 is the highest-priority bet — it validates the entire architecture before further phases compound on it. Phases 2-4 each follow the same pattern: extend the player's physics with one game's mechanics, ship the fixture + trace, gate via CI. Phase 5 lands the user-facing UX completeness items that aren't physics-dependent.

---

## Documentation Plan

- `pixelforge_cart/doc.go` — package overview + footer format + read-self semantics + Authenticode invariant note
- `pixelforge_physics/doc.go` — substrate overview + resolv adoption rationale + one-way platform recipe + slope recipe + determinism guarantees
- `pixelforge_render/doc.go` — `RenderTickAt` contract + cycle constraints + determinism guarantees
- `pixelforge_replay/doc.go` — `.trace.jsonl` schema + recorder/replayer usage + determinism boundaries
- `docs/asset-library-manifest.md` — manifest schema + publishing checklist + reference-example update procedure
- `docs/verb-catalog.md` — generated by `go generate`; do-not-edit-by-hand notice at top
- `docs/reference-games-capability-matrix.md` — generated by CI; same notice
- `docs/credits.md` — CC0 asset attestations for starter pack + reference-game assets
- Each new sink file (`scene_sink.go`, `motion_sink.go`, etc.) gets a leading doc comment naming the verb topics it owns + the test scenarios that cover it

---

## Operational / Rollout Notes

- **Backward compatibility:** plan-008's per-cart codegen path (`sourceLongBuilder`) stays available for users who want the editable Go source. Build → Host + Build → WASM switch to the new cart-append path; Build → Source remains the same. No breaking change for existing projects.
- **Migration:** existing `.pforge` files load directly into the new player; no migration needed. The `pisave.Snapshot` v1 → v2 migration is handled at load by U10 (warn + treat as empty).
- **Rollback:** if a v2 release goes sideways, revert by switching the build-pipeline's default target from cart-append back to codegen-per-cart; the codegen path is preserved.
- **CI cadence:** `.github/workflows/long.yml` runs on every push to main + every PR; the long-tag suite must stay green or the merge is blocked.
- **Asset-library manifest update procedure:** documented in `docs/asset-library-manifest.md`; manifest changes are committed to a separate repo (or a `.assets` branch) + published via GitHub Release. Studio's downloader URL is configurable so forks/dev environments can point at staging manifests.

---

## Future Considerations

- v3 will likely add Pac-Man + Frogger + Space Invaders + Tetris to the matrix (per the brainstorm's scope-boundaries deferral).
- Headless browser smoke test (wasmbrowsertest / chromedp) is the natural follow-up after WASM ships in v2.
- Authenticode signing is a clean v3 addition once cart-append-before-sign is documented (which v2 does).
- The shared `RenderTickAt` seam invites future optimizations (GPU-side path, mobile target) without breaking the preview-vs-shipped invariant.
- The verb catalog regen mechanism could grow to also drive the visual scripting GUI's verb palette + the Inspector dropdown — already a single source of truth, just unused there today.
