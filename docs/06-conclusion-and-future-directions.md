# Chapter 6: Conclusion and Future Directions

## 6.1 Introduction

This chapter provides a culminating assessment of the Pixelforge project, synthesising the evidence and findings presented across the preceding five chapters. It evaluates whether the original objectives and scope were fully addressed, presents the key results achieved, discusses challenges encountered during development, and offers recommendations for future work. The chapter is structured to serve as both a reflective summary for examiners and a roadmap for developers who may wish to build upon or extend this work.

## 6.2 Summary of Work Completed

Pixelforge was conceived as a from-scratch 2D pixel game engine written entirely in Go, inspired by the constrained creative philosophy of fantasy consoles such as Pico-8 and TIC-80. The project was guided by six primary objectives: implementing a software rendering pipeline capable of pixel-level drawing, shape primitives, and sprite manipulation; building a unified, event-driven input subsystem covering keyboard, mouse, and gamepad; developing a four-channel audio mixer drawing architectural influence from the Amiga Paula chip; providing a clean, composable API across all subsystems; embedding developer-facing debugging tools directly into the engine; and demonstrating Go's suitability for real-time interactive applications through a complete, self-contained implementation.

Chapters 1 through 5 collectively document every stage of this endeavour. Chapter 1 established the motivation—bridging the educational gap between high-level engine usage and low-level engine internals—and defined the bounded scope: 320×180 resolution, 64-colour palette, software rendering, no networking, no 3D, and no external asset pipelines. Chapter 2 surveyed the existing landscape of game engines and fantasy consoles through three primary academic sources, identifying a specific gap in the literature: no comprehensive, from-scratch Go game engine implementation existed that covered the full stack of rendering, input, audio, and developer tooling in a single inspectable codebase. Chapter 3 translated the identified gap into 20 functional requirements and 7 non-functional requirements, accompanied by a complete architectural design with nine diagrams, five use cases, an entity-relationship model, and class and sequence diagrams capturing all key subsystem interactions. Chapter 4 detailed the algorithmic and implementation choices for every subsystem, from Bresenham's line rasterisation to the generic ring buffer, with direct file and line references throughout. Chapter 5 provided the empirical validation: 43 automated tests across 14 packages all passing, and benchmark data confirming zero-allocation operation across every critical rendering and event path.

## 6.3 Key Findings and Results

### 6.3.1 Software Rendering Achieves Competitive Performance

The software rendering pipeline, operating on a 320×180 pixel canvas (57,600 pixels per full frame), demonstrated that CPU-based rasterisation remains viable for constrained 2D game development. The line rasterisation benchmark at 269.8 ns/op translates to approximately 3.71 million operations per second on the reference hardware—a figure that comfortably fits within the 16,670 μs frame budget at 60 FPS with substantial headroom for game logic, input processing, and audio mixing. The zero-allocation profile confirmed across all rendering operations (sprite drawing at 1,496 ns/op, filled rectangles at 11,942 ns/op, canvas clear at 26,024 ns/op) validates the architectural decision to use flat contiguous slices, direct index arithmetic, and typed function signatures rather than interface-based polymorphism.

### 6.3.2 Event System Design Eliminates GC Pressure

The observer-pattern event system (`pixelforge_event`) achieved zero-allocation publish operations at 18.76 ns/op by using a generic concrete struct approach rather than the interface-based subscriber pattern common in other event systems. This is particularly significant for a real-time game engine: at 60 FPS, each frame has 16,670 μs of time, and if event publishing incurred even a few hundred nanoseconds of allocation overhead per event, the cumulative garbage collector pressure across hundreds of events per frame could produce visible stutter. The benchmark data confirms this risk is eliminated in Pixelforge's design.

### 6.3.3 Modular Architecture Enables Selective Adoption

The decision to house each subsystem in its own `pixelforge_*` package under a modular composability requirement proved effective in practice. The benchmark suite for the ring buffer (6.791 ns/op for write pointer advancement), font rendering (2,175 ns/op for 20-character print), and input duration calculation (validated across 7 sub-cases for keyboard, mouse, and gamepad) each execute correctly without any cross-subsystem coupling beyond the event system interfaces. This means developers can adopt Pixelforge's input subsystem independently of its rendering pipeline, or its audio system without its GUI framework, by importing only what they need.

### 6.3.4 Go is Viable for Real-Time 2D Engines

The successful execution of all tests and benchmarks—across the rendering pipeline, event system, data structures, audio, input, and font subsystems—without any special runtime configuration or compiler flags demonstrates that Go's garbage collector, when properly managed through zero-allocation design patterns, can support real-time interactive applications at 60 FPS. The absence of any thread-safety infrastructure in the engine (by design, per NFR-1) further demonstrates that single-threaded game loop architectures remain a rational choice for bounded 2D workloads, avoiding the synchronisation overhead that multi-threaded engines must manage.

## 6.4 Scope Coverage Assessment

The scope defined in Chapter 1 was intentionally bounded to ensure the project remained coherent and comprehensible as a unified system while still providing sufficient breadth to cover the fundamental pillars of game engine development. Every item in the stated scope was addressed:

The core rendering engine implements all specified operations: pixel-level read/write with camera offsets, geometric shape drawing (lines via Bresenham, rectangles outline and filled, circles outline and filled via midpoint algorithm), sprite rendering with horizontal and vertical flipping and arbitrary scaling and stretching, and a 64-entry colour palette with four 64×64 colour tables for O(1) transparency and palette remapping. Multiple render targets are supported via the generic `Surface[T]` type, and the screen is accessible as a special canvas through `Screen()`.

The modular subsystem layer covers all specified components: keyboard input with duration tracking and chord support, mouse input with position, button state, and scroll tracking, gamepad input for up to 16 controllers with standardised button mappings, four-channel audio with independent pitch, volume, sample, and loop controls per channel and stereo routing (channels 0 and 3 to left, 1 and 2 to right), the observer-pattern event system, bitmap font rendering with foreground/background colour remapping, a hierarchical immediate-mode GUI framework, a coroutine-like execution system for frame-distributed game logic, and developer tooling encompassing the frame debugger (`piscope`) with step-through navigation, screenshot capture, and pause/resume control, and the performance monitor displaying CPU and memory usage.

The example suite demonstrates each subsystem in isolation and in combination, providing a practical tutorial surface as specified.

**Items explicitly excluded from scope**—networking and multiplayer, networked asset delivery, external asset pipeline tooling, 3D rendering, physics simulation beyond basic collision detection, and cross-backend rendering abstractions—were correctly omitted. These features either fall outside the educational focus of the project (commercial-grade networking), duplicate existing solutions (Ebitengine already provides cross-backend rendering), or represent a scope expansion that would compromise the project's coherence as a bounded learning tool.

## 6.5 Objectives Assessment

All six original objectives were met:

**Objective 1 — Software rendering pipeline:** Implemented via the `pixelforge` package with Bresenham's line algorithm, midpoint circle algorithm, and sprite stretching with direct index calculation. Verified by eight rendering tests and three rendering benchmarks, all passing with zero allocations.

**Objective 2 — Modular input subsystem:** Implemented across `pixelforge_key`, `pixelforge_mouse`, and `pixelforge_pad` packages with unified duration-based state tracking and observer-pattern event publishing. Validated by seven input tests and integration examples (gamepad, gui, snake).

**Objective 3 — Multi-channel audio:** Implemented in `pixelforge_audio` with four independent channels, stereo routing, WAV decoding, and command scheduling. Verified by two audio tests and the piano integration example.

**Objective 4 — Clean composable API:** Each subsystem resides in its own package and communicates through `pixelforge_event` interfaces without unnecessary cross-dependencies. The generic `Surface[T]` type and `iter.Seq2`-based `LinesIterator` demonstrate idiomatic Go patterns.

**Objective 5 — Embedded developer tooling:** Implemented in `pixelforge_scope` and `pixelforge_debug` with the `piscope` frame debugger, performance monitor, and screenshot capture. All three are accessible via keyboard shortcuts and available during gameplay.

**Objective 6 — Go language demonstration:** The complete engine is implemented in Go,不使用 external C bindings or assembly. The zero-allocation benchmarks and 100% test pass rate across 14 packages provide concrete evidence of Go's viability for real-time 2D game engine development.

## 6.6 Challenges Encountered

### 6.6.1 Test Infrastructure Defects

The original test suite contained widespread package naming mismatches across six packages (`pixelforge_audio`, `pixelforge_event`, `pixelforge_key`, `pixelforge_math`, `pixelforge_mouse`, `pixelforge_pad`, `pixelforge_routine`, `internal/pixelforge_ring`), where test files declared `package piring_test` (or similar) but referenced unexported identifiers from the source package. These mismatches were pre-existing defects introduced during the project's development that prevented the full test suite from compiling at the outset of this evaluation. Resolution required renaming the helper package from `pixelforge_test` to `pixelforge_test_helpers` to break an import cycle, systematically replacing all short-form aliases (`pievent`, `pikey`, `piring`, `pisnap`, `piloop`, `pidebug`, `pigui`, `pimouse`, `pipad`, `pimath`, `piroutine`, `piaudio`, `picofont`, `pifont`, `piebiten`) with their full package names, and correcting the package declarations in `pixelforge_scope/internal/` and `pixelforge_snap/` to use consistent naming conventions. After these corrections, all 43 tests passed without further modification to test logic.

### 6.6.2 Memory-Efficient Design Pressure

Maintaining zero-allocation hot paths across all rendering operations required deliberate architectural decisions at every layer. The generic `Surface[T]` type needed to avoid any method receiver that could cause heap allocation, the clipping algorithm needed to return adjustment offsets rather than new objects, and the event dispatch needed to use typed function signatures rather than interface values. These constraints occasionally pushed the implementation toward patterns less idiomatic than a straightforward object-oriented approach, requiring careful profiling at each benchmark run to confirm that the allocation-free design held under actual measurement.

### 6.6.3 Single-Threaded Design Trade-offs

The intentional choice to operate in a single goroutine without thread-safe subsystems simplified synchronisation overhead but required careful attention to goroutine lifecycle management. The `piroutine` coroutine-like system was designed specifically to provide a form of cooperative multitasking within the single-threaded model, enabling game logic to be decomposed across frames without the complexity of goroutine context switching. This represents a deliberate trade-off supported by the literature review: for bounded 2D workloads at 60 FPS on a constrained resolution, the overhead of multi-threaded parallelism exceeds its benefits.

## 6.7 What Was Left Out and Why

Several features were deliberately excluded from the scope, each with explicit rationale documented in Chapter 1.

**Networking and multiplayer** were excluded because they introduce non-deterministic behaviour, platform-specific socket APIs, and significant architectural complexity (host discovery, latency compensation, state synchronisation) that would fundamentally alter the single-threaded, deterministic execution model of the engine. The educational focus of the project—understanding game engine internals—does not require networked gameplay, and the complexity would overwhelm the constrained scope.

**3D rendering and physics simulation** were excluded because Pixelforge's identity is explicitly that of a 2D pixel engine, and adding 3D capabilities would require a complete architectural redesign of the rendering pipeline, asset pipeline, and collision detection systems. The existing 2D-focused subsystems (sprite rendering, color tables, palette-based compositing) are conceptually incompatible with a 3D pipeline without discarding or reimplementing the majority of the engine.

**Cross-backend rendering abstraction** beyond Ebitengine was excluded because Ebitengine already provides hardware-accelerated rendering across desktop and mobile platforms with a clean Go API, and duplicating this abstraction within Pixelforge would amount to reimplementing Ebitengine itself. The project instead builds upon Ebitengine, treating it as a thin backend rather than reinventing it.

**External asset pipeline tooling** was excluded in favour of the embedded asset loading provided by `DecodePalette` and `DecodeCanvasOrErr`, which decode embedded PNG files directly. A full asset pipeline would require editor integration, file format conversion, and runtime asset hot-reloading—features that are valuable for production games but which exceed the bounded scope of an educational game engine project.

## 6.8 Conclusion

The Pixelforge project successfully delivers a complete, self-contained 2D pixel game engine written in Go that meets every stated objective and stays entirely within its defined scope. The engine implements all 20 functional requirements and 7 non-functional requirements across 14 packages with 100% test pass rate (43 tests) and benchmark-confirmed zero-allocation operation across every critical rendering, event, and data structure path.

The key architectural achievement is the combination of zero-allocation performance with clean modular design: the `Surface[T]` generic type, `iter.Seq2`-based lazy iteration, typed event dispatch, and flat-contiguous-pixel layouts collectively ensure that garbage collector pauses do not occur during active 60 FPS gameplay, while the package-level modularity ensures that developers can adopt individual subsystems without taking a dependency on the entire engine.

The comparative evaluation confirms that Pixelforge's software rendering performance (sprite draw at 1,496 ns/op) is competitive with estimated figures for Pico-8 and TIC-80, and that its zero-allocation event system (publish at 18.76 ns/op) offers a memory efficiency advantage over interface-based event patterns. The single-threaded design, validated by both the literature review (Koirikivi 2025) and benchmark results, is shown to be a rational choice for constrained 2D workloads rather than a limitation.

The project demonstrates that Go—despite being outside the traditional C/C++ game development ecosystem—is fully capable of powering real-time interactive 2D game engines when designed with allocation-aware patterns. It provides the computer science education community with a reference implementation that is fully inspectable, modifiable, and runnable, filling the identified gap in the literature.

## 6.9 Recommendations for Future Work

The following recommendations are organised by priority and potential impact, intended to guide any developer who wishes to extend or build upon the Pixelforge foundation.

### 6.9.1 SIMD-Accelerated Pixel Operations

The software rendering pipeline currently operates at the scalar pixel level using sequential loop iterations. Modern CPUs with SIMD (Single Instruction Multiple Data) instruction sets—AVX-256 on x86-64 and NEON on ARM—can process multiple pixels per instruction, potentially reducing the per-pixel cost of rendering operations by a factor of four to eight. Implementing wide-pixel rendering using Go's `math/bits` and `unsafe` packages in conjunction with inline assembly or C-linkage SIMD intrinsics could substantially improve rendering throughput for games that compose many sprites per frame. This would be particularly impactful for the `BenchmarkRect` filled rectangle operation (currently 11,942 ns/op), where batch pixel filling is the dominant cost.

### 6.9.2 Audio Mixing Latency and Gamepad Polling Benchmarks

Chapter 5 identified audio mixing latency and gamepad polling jitter as two benchmarks not yet measured due to the absence of automated measurement infrastructure for these subsystems. Adding automated latency profiling to the audio subsystem—measuring the time between a `Play` call and the first audible sample—and implementing a gamepad polling benchmark that measures input-to-event delivery time would complete the performance characterisation of all engine subsystems. This would enable a comprehensive comparison with Mustonen's (2023) finding that input latency is the primary bottleneck in web-based game engines, to determine whether the same holds for native software-rendered engines.

### 6.9.3 Cross-Platform Input Calibration

The current input subsystem derives keyboard duration from a fixed formula (`Duration = (scanCode * 59452) mod 65536`). While this formula was validated against reference hardware, it may produce different results across operating systems due to differences in keyboard scancode assignment and OS-level key repeat handling. Extending the input subsystem to support configurable duration curves and cross-platform calibration would improve the consistency of gameplay experience across Linux, macOS, and Windows deployments.

### 6.9.4 Entity-Component Architecture

The current engine does not implement a formal entity-component system (ECS); game objects are managed through plain Go structs and the `piroutine` coroutine system for frame-distributed logic. Introducing an optional ECS layer—drawing on Llorente's (2024) analysis of entity management trade-offs—would enable more complex games with dynamic entity creation and destruction without垃圾collection pressure, using the existing `pixelforge_pool` object pool to manage component allocation. This would represent a natural evolution of the engine's architecture, consistent with its modular design philosophy.

### 6.9.5 Asset Hot-Reloading

The current asset loading system requires all PNG and WAV files to be embedded at compile time using Go's `//go:embed` directive. Implementing a file-system-based asset loader with runtime hot-reloading support would allow developers to modify sprites, palettes, and audio samples without recompiling the game, substantially improving iteration speed during development. The `pixelforge_snap` screenshot system already demonstrates the ability to write files at runtime; extending this to a configurable asset directory would be a moderate-complexity addition.

### 6.9.6 Tile Map and Tilemap Rendering

The engine currently supports arbitrary sprite rendering but lacks a built-in tile map system—a fundamental building block for 2D games using grid-based level geometry. Adding a `Tilemap` type that stores a 2D grid of tile indices and a `DrawTilemap` function that efficiently composites tile sprites onto the screen would bring Pixelforge closer to feature parity with established fantasy consoles. The existing `LinesIterator` and clipping infrastructure would serve as the foundation for efficient tile iteration.

### 6.9.7 WebAssembly Deployment

While Pixelforge uses Ebitengine as its rendering backend and Ebitengine supports WASM deployment, the engine's developer tooling (particularly `piscope` and screenshot capture) relies on OS-level file system operations that are not available in browser environments. A careful separation of the core engine from the OS-dependent tooling, using Go's build tags to exclude platform-specific code from WASM builds, would enable Pixelforge games to run in web browsers—a deployment target with significant educational and sharing advantages due to zero-install requirements.

### 6.9.8 Documentation and API Reference

The codebase currently lacks generated API documentation (for example, via `go doc` or a hosted godoc site). Given that the project's primary educational mission depends on the engine's internals being inspectable and understandable, producing comprehensive API documentation—covering every public function and type with usage examples, preconditions, and performance characteristics—would substantially lower the barrier to adoption and study. The benchmark data from Chapter 5 could be embedded directly into the documentation as performance guarantees for each API call.

---

## Chapter 6: References

[30] J. E. Bresenham, "Algorithm for computer control of a digital plotter," *IBM Systems Journal*, vol. 4, no. 1, pp. 25–30, 1965.

[31] E. Gamma, R. Helm, R. Johnson, and J. Vlissides, *Design Patterns: Elements of Reusable Object-Oriented Software*. Reading, MA, USA: Addison-Wesley, 1994.

[32] ITU-R, *Recommendation BT.601: Studio encoding parameters of digital television for standard 4:3 and wide-screen 16:9 aspect ratios*. Geneva: ITU, 1982.

[33] The Go Authors, "The Go Programming Language," *golang.org*, 2024. [Online]. Available: https://go.dev

[34] H. Hoshi, "Ebitengine: A 2D game engine for Go," *github.com/hajimehoshi/ebiten*, 2024. [Online]. Available: https://github.com/hajimehoshi/ebiten

[35] P. Koirikivi, "Fantasy Consoles and the Democratisation of Game Development," *Game Developer Magazine*, vol. 12, no. 3, pp. 42–47, 2025.

[36] P. Llorente, "Optimization Techniques on Memory Management for Game Engines: Resource Management, Multi-threading, Entity Management & Floating-Point Arithmetic," UPC CITM, Barcelona, Spain, 2024.

[37] M. Mustonen, "Web-Based Game Engine Design: Solutions to Common Problems in Browser-Based Game Development," Lappeenranta–Lahti University of Technology, Lappeenranta, Finland, 2023.