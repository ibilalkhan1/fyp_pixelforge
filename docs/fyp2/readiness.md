# PixelForge FYP-II delivery status

Reviewed 24 November 2026. Baseline: `692f63aff4932a54ef782d2c8feaaa2457ba4247`.

## What is available

- A focused browser demonstrator in `web-demo/`, backed by the actual Go PixelForge renderer compiled to WebAssembly.
- Tile editing, 8x8 sprite painting, keyboard movement, AABB wall collision, collectible completion, scene JSON import/export, browser save/restore, screenshots and standalone playable HTML export.
- The native Studio still builds. It is a separate application, not the browser demo.
- Normal Studio builds now select real exporters instead of placeholder writers. macOS exports keep the executable unchanged, store the cart in app Resources, and sign after metadata and icons are written.
- Universal carts can package referenced sprites and WAV audio with project JSON. Legacy JSON-only carts remain readable. Missing files and unsafe paths fail clearly. MP3/OGG export is rejected until decode support exists.
- Smoke startup retains assets and uses the same loaders as normal startup. Old cached players are invalidated by a cache-version bump.

## Build and demonstrate

Requires the repository's Go toolchain version (go.mod: 1.24.2 or newer).

```sh
./web-demo/build.sh
python3 -m http.server 8765 --bind 127.0.0.1 --directory web-demo
```

Open `http://127.0.0.1:8765`. For a temporary faculty link, in a second terminal:

```sh
ngrok http http://127.0.0.1:8765
```

Share the HTTPS URL ngrok prints. Visitors may see ngrok's first-visit page and need to choose Visit Site. Keep the Mac awake, online, and both processes running. Restarting or changing ngrok configuration can change the URL. This is a temporary demo, not production hosting.

Use Export playable HTML before the presentation and keep the downloaded file locally. The export contains the engine and the selected scene. Its download and script syntax were verified; local file-protocol browser execution still needs a manual rehearsal on the presentation machine. The browser demonstrator uses its own versioned JSON schema, not Studio's `.pforge` format. Audio and full native Studio tools are outside this demo.

## Verified checks

17 targeted Go packages passed, including root engine, physics, project, events, routines, cart, saves, capsule runtime, code generation, build pipeline, scripting runtime/catalogue, camera, tilemap, entities, audio and the browser demo. Default-tag native Studio compilation and universal-player WASM compilation passed on this Mac. The extended real-exporter buildpipeline tests also passed, including the native developer build fallback. Public browser load, tile and sprite editing, keyboard movement, wall collision, collectible pickup, browser save/restore and HTML download were checked.

A later GitHub run at `50be43a2` passed the full default Linux `go test ./...` suite under Xvfb, plus the browser checks and no-new-lint gate. This is separate from local macOS native acceptance. The inherited lint backlog remains; see [lint-baseline.md](lint-baseline.md). Long integration test failures now fail their parent job directly while still uploading results for the capability report. Follow the latest [PR checks](https://github.com/ibilalkhan1/fyp_pixelforge/pull/1/checks) for the current revision.

The event-publish microbenchmark measured 3.112 ns/op, 0 B/op, 0 allocs/op on Apple M4 with 100ms benchtime. It is not a complete-game FPS measurement or a comparison with another engine.

## Outstanding acceptance work

1. Native Studio's complete edit/play/export loop is not yet accepted. Its preview asset resolver, authored-behaviour startup, player input bridge, drawing integration and per-tick runtime housekeeping still need joint end-to-end verification and fixes.
2. MP3/OGG music decoding remains incomplete. Current runtime loading supports WAV samples. Do not claim working MIDI support without an actual decoded/playback demonstration.
3. Mario, Asteroids, Bomberman and Donkey Kong fixtures are limited scripted proofs, not complete reference games. Some regression-detector tests explicitly skip.
4. Full native player tests could not finish in this environment because native macOS window-service initialization stalled. The process timed out. The passing Linux CI default suite does not establish local macOS native acceptance.
5. Windows/Linux executable launches, browser-device compatibility, gamepad controls and audible audio playback need separate acceptance runs.
6. Rebuild every shipped player binary when distributing the new asset-bearing cart format. Old embedded player binaries cannot read ZIP carts.

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
