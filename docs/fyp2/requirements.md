# FYP-I requirement traceability

Reviewed 24 November 2026. Source: supplied FYP-I report, printed pages 17-20 (PDF pages 31-34). Status applies to current evidence, not a claim of full acceptance.

| ID | Requirement | Status | Evidence / limit |
|---|---|---|---|
| FR-1 | Pixel rendering | Verified core + browser | Root engine tests passed. Browser draws a 320x180 indexed-colour canvas using PixelForge functions. |
| FR-2 | Shapes | Verified core | Root shape tests passed. Wall and crystal drawing use RectFill, Line and CircFill. |
| FR-3 | Sprites | Core verified; integration partial | Root sprite tests passed. Browser 8x8 sprite editing/drawing works. Native preview asset resolution remains open. |
| FR-4 | Palette | Verified core | Root palette tests passed. Engine has 64 entries; browser painting exposes the first 16. |
| FR-5 | Render targets | Verified core | Canvas/Surface and SetDrawTarget exist; root surface tests passed. Full native editor composition not accepted. |
| FR-6 | Keyboard | Implemented; browser verified | Native key subsystem exists. Browser adapter collects arrow/WASD state and Go applies movement. Native player input bridge remains open. |
| FR-7 | Mouse | Implemented; partial verification | Native mouse package exists. Browser pointer editing was exposed; full native mouse behaviour not retested. |
| FR-8 | Gamepad | Implemented; unverified devices | Gamepad/pad code exists. No controller-device acceptance run. Browser demo has keyboard/touch controls, no gamepad claim. |
| FR-9 | 4-channel audio | Package tests passed | Core audio package passed. Audible native output and multi-channel playback still need rehearsal. |
| FR-10 | Audio scheduling | Source evidence | Scheduling implementation exists. Current checks do not establish sample-accurate output on real audio hardware. |
| FR-11 | Events | Verified | Event package passed. Publish microbenchmark: 3.112 ns/op, zero allocations on this Apple M4 run. |
| FR-12 | Lifecycle | Implemented; integration partial | Loop events exist. Shared player render path still needs authored behaviour, rendering and housekeeping acceptance. |
| FR-13 | Coroutines | Verified package | Routine and scripting runtime package tests passed. Complete Studio-exported behaviour execution remains open. |
| FR-14 | Bitmap fonts | Source evidence | Bitmap/font packages exist. Not separately exercised in the current browser demo. |
| FR-15 | In-game GUI | Source evidence | GUI package exists. Do not equate this with a complete native Studio workflow. |
| FR-16 | Frame debugger | Source evidence | Scope/debug/capture implementations exist. Live stepping and history navigation not visually accepted here. |
| FR-17 | Performance monitor | Source evidence | Metrics/stat modules exist. CPU/RAM overlay values not verified in a visible native session. |
| FR-18 | Screenshots | Browser path available | Browser screenshot download implemented. Native F12 capture remains to be checked in the presentation environment. |
| FR-19 | Backend integration | Build verified | Native Studio compiles with Ebitengine and Dear ImGui dependencies. Universal player compiles to WASM. Native UI launch not verified. |
| FR-20 | Object pooling | Source evidence | Pool package exists. No new standalone pool benchmark or allocation audit performed in this delivery. |
| NFR-1 | Single-goroutine contract | Partial | Core documentation requires single-goroutine use. Blanket per-API goroutine panic enforcement is not established. Old slide claims of work-stealing contradict this contract. |
| NFR-2 | Minimal allocations | Partial | Event publish measured zero allocations. Entity rendering and browser output allocate. Do not generalize one benchmark to every hot path. |
| NFR-3 | Modularity | Implemented | Separate engine packages exist. Studio/runtime cross-package dependencies remain substantial. |
| NFR-4 | Bounded surfaces | Implemented | SetScreenSize enforces positive dimensions and a 131072-pixel limit. The limit is a total-pixel cap, not one fixed resolution. |
| NFR-5 | Cross-platform | Partial | Native Mac and WASM builds pass. Public browser demo works. Windows/Linux launches and broader browser acceptance remain unverified. |
| NFR-6 | Readable documentation | Partial | README and delivery documentation now describe setup and limits. Older report/source comments still contain stale or contradictory claims. |
| NFR-7 | Tests | Partial | 17 targeted packages pass. Full native test suite is not green/accepted; player tests stalled in macOS window initialization. |
