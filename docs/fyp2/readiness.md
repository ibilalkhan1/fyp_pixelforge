# PixelForge FYP-II delivery status

Reviewed 24 November 2026. Baseline: `692f63aff4932a54ef782d2c8feaaa2457ba4247`.

## What is available

- Permanent faculty creator: https://ibilalkhan1.github.io/fyp_pixelforge/ . Free GitHub Pages static hosting keeps the demo available without a laptop or ngrok session.
- A focused browser game creator backed by the actual Go renderer through WebAssembly: two editable examples plus blank projects, tile and 8x8 character painting, collection/escape rules, hazards, movement/collision, one local save slot, JSON exchange and self-contained playable HTML export.
- Two separate playable Go examples, Snake and Pac-Man, with Start, Restart, sound controls and held touch direction controls. These are not editable creator templates.
- Native Studio F5 Play now starts a detached project copy with embedded sprite assets, authored keyboard behaviours and shared runtime housekeeping. F6 Stop restores the authored scene. Basic movement and restoration were observed on the Mac.
- Normal builds use real exporters. A generated standalone native game compiled, launched and responded to arrow input on Mac. CI also compiles a generated native game on Linux and Mac.
- Universal carts package sprites and supported audio. Real WAV, MP3 and OGG/Vorbis decoders replace header-only placeholders. The mixer uses signed 8-bit mono data, with bounded in-memory decoding. This is not streaming audio or MIDI support.

## Build and demonstrate

Requires the repository Go toolchain (go.mod: 1.24.2 or newer) and native platform build dependencies where applicable.

```sh
./web-demo/build.sh
python3 -m http.server 8765 --bind 127.0.0.1 --directory web-demo
```

Open `http://127.0.0.1:8765` for a local backup, or use the permanent faculty link. The static site contains HTML, CSS, JavaScript and WebAssembly. It does not host a server-side native Studio or publish visitors' exported games automatically.

Download editable JSON and playable HTML before the presentation. Browser JSON and Studio `.pforge` remain different formats. Audio, authored scripts and multiple levels are outside the focused browser creator. Manually rehearse local-file HTML on the presentation laptop. Use the local HTTP bundle if browser file restrictions prevent it.

## Verified checks

Implementation revision `8e5626139ebef98a4a4a2ac7eb5a8842966dccef` passed [full CI](https://github.com/ibilalkhan1/fyp_pixelforge/actions/runs/45462200846): the default Linux suite, both long proof suites, generated native game builds on Linux and Mac, new-finding lint, verb catalogue and capability report. [Browser/core CI](https://github.com/ibilalkhan1/fyp_pixelforge/actions/runs/45462200912) also passed. Each long suite recorded 85 passing test/subtest events and six explicit skips. Skips are not passes. Current [PR checks](https://github.com/ibilalkhan1/fyp_pixelforge/pull/1/checks) show later documentation revisions.

The full default Mac suite passed across 64 test packages in a working desktop session. The shared rendering regression verifies sprite pixels, key-driven movement, release and one housekeeping advance per tick. A host regression guards against duplicate lifecycle events. Real encoded-tone fixtures verify WAV, MP3 and Vorbis decode output. The generated native source regression compiles a game instead of checking syntax alone.

The public creator loaded, saved a named project, moved the player and downloaded playable HTML. Prior authoring checks covered blank projects, painting, collision, both win rules, hazards, saves and JSON. Public Snake scored, restarted and moved. Public Pac-Man reached score 150 with three lives, then Restart reset it to zero. Browser error logs were empty in these checks. Native Studio F5 Play, arrows, Space audio trigger and F6 Stop restoration were observed. The independent exported Mac app moved the same authored character.

The new-finding lint gate compares against the supplied-source baseline. Inherited lint debt remains. See [lint-baseline.md](lint-baseline.md) and [native/browser acceptance](native-browser-acceptance.md).

The isolated event-publish benchmark measured 3.112 ns/op, 0 B/op and 0 allocs/op on Apple M4 with 100ms benchtime. It is not a full-game FPS measurement or cross-engine comparison.

## Outstanding acceptance work

1. Complete authored reference games. Mario, Asteroids, Bomberman and Donkey Kong fixtures are limited subsystem proofs, not finished games. Six explicit skips remain in each long suite.
2. Broader native authoring: complex scene switching, behaviour combinations, controller input, effects/dialogue overlays, mouse-based workspace editing and long sessions require acceptance beyond the verified basic preview.
3. Windows/Linux interactive executable launches. CI builds do not replace launching and playing the game on those systems.
4. Audible output and volume on the presentation speakers. Decoder tests and a triggered audio command do not prove what the room hears.
5. Manual offline HTML rehearsal on the actual presentation laptop and public-link testing on another device/network.
6. Rebuild shipped players when changing the cart format or runtime lifecycle. Old cached players are invalidated by the current cache version.

Do not claim 100% completion of the original FYP scope from this evidence.

## Claims to correct from FYP-I

- Core software rasterization is custom Go code. Native display/input integration uses Ebitengine, Studio chrome uses Dear ImGui/C bindings, and physics uses resolv. The project has third-party dependencies.
- The core API explicitly follows a single-game-goroutine contract. A work-stealing thread-pool claim requires independent evidence.
- The report states 320x180 as PixelForge's default. PICO-8's official specifications state 128x128 and 16 colours. NES inspiration does not mean full NES PPU emulation.
- Do not compare memory, FPS, minimum hardware or zero-allocation behaviour across all subsystems using one isolated benchmark.
- Correct AABB inequalities: AxMin < BxMax, AxMax > BxMin, AyMin < ByMax, AyMax > ByMin. Discrete integration of motion is not the brachistochrone problem.

## Team and provenance

FYP-I names: Muhammad Bilal (45775), Hammad Ahmad (46391), Basheer Shah (47141). Supervisor: Sir Tajamul Shahzad. Names and documented responsibilities come from the supplied FYP-I presentation and should be confirmed by the team.

The baseline's reachable Git history contains 220 commits: Bilal Khan 98, Basheer identity variants 63, Hammad identity variants 59. These are recorded author counts, not proof of individual effort. Current AI-assisted updates must retain accurate authorship. Do not backdate commits, impersonate teammates or create empty commits for activity graphs.

Primary reference checks: https://www.lexaloffle.com/pico-8.php and https://ebitengine.org/.
