# Chapter 1: Introduction

## 1.1 Goals and Objectives

The primary goal of Pixelforge is to democratize game development by providing an accessible, self-contained 2D game engine that abstracts away the complexities of low-level graphics programming while retaining the charm and discipline of retro pixel art game creation. Unlike mainstream game engines that cater to 3D AAA development with steep learning curves and enormous dependency trees, Pixelforge targets developers who wish to create lightweight, nostalgic 2D experiences without sacrificing control over rendering pipelines or being forced into bloated toolchains. The engine aims to serve as both a learning instrument for understanding game engine internals—such as pixel rendering, sprite management, input handling, and audio playback—and a functional framework capable of powering actual playable games.

The objectives of this project are multi-faceted and span across several dimensions of software engineering and computer science education. The first objective is to design and implement a custom software rendering pipeline capable of drawing pixels, shapes, lines, and sprites directly onto a virtual display surface, emulating the constrained yet expressive aesthetics of classic fantasy consoles like Pico-8. The second objective is to build a modular input subsystem that handles keyboard, mouse, and gamepad input in a unified, event-driven manner, enabling developers to create games that are playable across multiple input devices without requiring platform-specific code. The third objective is to develop a multi-channel audio playback system capable of mixing multiple PCM audio streams in real time, drawing architectural inspiration from legacy audio hardware such as the Amiga's Paula chip, thereby giving developers the ability to compose simple but effective chiptune-style soundtracks for their games. The fourth objective is to provide a clean, composable API surface that allows engine subsystems to be mixed and matched according to the needs of each project, avoiding the monolithic architecture found in larger engines and encouraging developers to understand how each subsystem interconnects with the others. The fifth objective is to embed developer-facing tooling—frame stepping debuggers, performance monitoring overlays, and screenshot capture utilities—directly into the engine itself, lowering the barrier for troubleshooting and profiling during active development. The sixth and final objective is to ensure the entire system is written in Go, a language not traditionally associated with game development, thereby demonstrating Go's applicability to real-time interactive applications and providing the computer science community with a reference implementation of a game engine architected entirely in concurrent, garbage-collected systems programming idioms.

## 1.2 Scope of the Project

The scope of Pixelforge encompasses the design and implementation of a complete, standalone 2D pixel game engine with rendering, input, audio, and debugging subsystems, all written in Go and executable on any platform supported by the Ebitengine rendering backend. The engine operates at a fixed resolution of 320 by 180 pixels—the same resolution canonical to Pico-8—and uses a constrained 64-color palette derived from the Pico-8 specification. This constrained environment is a deliberate design choice that simplifies rendering logic, reduces memory footprint, and forces creative problem-solving from developers working within well-defined visual boundaries.

The core engine package provides low-level pixel operations including individual pixel read/write access, geometric shape drawing (rectangles, lines, circles), sprite rendering with support for horizontal and vertical flipping as well as arbitrary scaling and stretching, and a color table system that enables transparency effects, color remapping, and palette-based blending through precomputed lookup tables. All rendering operations are performed onto a generic 2D surface data structure capable of storing arbitrary pixel types, allowing the engine to support multiple simultaneous rendering targets beyond the primary screen.

The modular subsystem layer builds upon the core engine and includes a keyboard input handler capable of tracking individual key states as well as chord combinations and synthetic "virtual keyboard" events for on-screen key rendering; a mouse input handler supporting cursor position, button state, and scroll wheel tracking; a gamepad input handler abstracting common controller layouts across platforms through a unified API; a four-channel audio player with per-channel pitch, volume, and sample control, mixing stereo output from independent mono PCM streams; an observer-pattern event system for decoupled communication between engine subsystems and user-defined game code; a bitmap font rendering system with built-in Pico-8 font support; a GUI element system for constructing in-game interfaces such as menus, buttons, and labels; a coroutine-like execution system that allows game logic to be decomposed across multiple frames without full goroutine overhead; and developer tooling including an integrated frame-stepping debugger, live CPU and memory usage overlays, and screenshot capture functionality.

Outside of the core engine and its subsystems, Pixelforge includes example implementations demonstrating each subsystem in isolation and in combination, providing a practical tutorial surface for new developers. The project explicitly excludes networking and multiplayer functionality, networked asset delivery, external asset pipeline tooling, 3D rendering, physics simulation beyond basic collision detection, and cross-backend rendering abstractions beyond the currently implemented Ebitengine integration.

Pixelforge does not attempt to replace existing solutions such as Ebitengine itself (which it uses as a backend), LÖVE, PICO-8, or Godot. Instead, it occupies a unique position as a learning-focused, from-scratch implementation that exposes every layer of the game engine stack to inspection, modification, and study. The scope is intentionally bounded to ensure the project remains comprehensible as a unified system while still providing enough breadth to cover the fundamental pillars of game engine development.

---

The remainder of this report is organized as follows. Chapter 2 presents a survey of existing game engines and fantasy consoles, identifies the gaps in currently available solutions, and establishes the theoretical and technological foundations upon which Pixelforge is built. Chapter 3 describes the system architecture in detail, covering the design of each major subsystem, the data structures employed, and the rationale behind key architectural decisions. Chapter 4 details the implementation of each engine subsystem, presenting algorithms, code patterns, and performance considerations encountered during development. Chapter 5 evaluates the completed system against the stated objectives, measuring performance characteristics, API usability, and correctness of implementation through both automated testing and manual inspection. Chapter 6 discusses potential improvements, feature extensions, and directions for future work that could build upon the foundation established by this project.# Chapter 2: Literature Review

## 2.1 Introduction

The landscape of game engine development has undergone significant transformation over the past decade, driven by evolving hardware capabilities, shifting developer expectations, and a growing recognition that game engines are not merely rendering frameworks but comprehensive software systems that encapsulate decades of accumulated computer science knowledge spanning graphics programming, memory management, audio synthesis, input handling, and real-time systems design. Understanding where Pixelforge sits within this landscape requires examining the foundational work that has preceded it, from early academic explorations of constrained creative environments to the modern open-source engines that dominate independent game development today. This chapter surveys the existing body of work in game engine architecture, with particular emphasis on the design philosophies and technical approaches that inform the construction of lightweight 2D engines. The review begins with background context on the problem domain and the historical forces that shaped it, proceeds through a detailed examination of related research and systems, and concludes with a synthesis that identifies the specific gap Pixelforge is designed to occupy.

## 2.2 Background and Problem Elaboration

The conventional wisdom in game development has long held that creating a game engine from scratch is a mistake. This view, while practical for commercial production, obscures the educational value inherent in building such systems and the unique creative constraints that emerge when one works at the pixel level rather than atop layers of abstraction. Mainstream engines such as Unity and Unreal Engine have reached such a degree of sophistication that they function effectively as operating systems for games, complete with integrated development environments, asset pipelines, physics simulations, and networked multiplayer frameworks. While these tools have democratized game development in important ways, they simultaneously insulate developers from the underlying mechanics of how games actually function, making it difficult for students and hobbyists to develop deep intuitions about rendering pipelines, memory layout, and real-time constraints.

The fantasy console movement, pioneered most notably by Lexaloffle Games' Pico-8, introduced a compelling alternative to this paradigm. By imposing strict artificial constraints on resolution, color depth, audio channels, and code size, fantasy consoles created bounded creative spaces that were simultaneously more approachable and more educational than their full-featured counterparts. The 128 by 128 pixel display and 16-color palette of Pico-8 are not merely aesthetic choices; they represent a deliberate reduction of the problem space to a size where every pixel is intentional and every audio sample is composable by hand. This philosophy of constraint as a design tool has deep roots in computing history, tracing back to the early days of 8-bit home computers where limited RAM and ROM forced programmers to develop highly optimized, elegant solutions that remain instructive decades later.

The choice of Go as an implementation language for a game engine is itself an unusual one that merits contextualization within the broader landscape of systems programming languages. Go's design philosophy, centered around simplicity, concurrency, and garbage collection, stands in contrast to the C and C++ that dominate game engine development. While Ebitengine has demonstrated that Go is capable of powering real-time 2D games with acceptable performance, the ecosystem lacks comprehensive educational examples of building game engine subsystems from first principles in Go. Pixelforge addresses this gap by treating the construction of each engine subsystem as an opportunity for both engineering and pedagogy, making the internals inspectable and modifiable rather than hiding them behind opaque abstractions.

The problem this project addresses is therefore threefold. First, there is a lack of comprehensive, from-scratch game engine implementations that cover the full stack of engine development in a single, coherent codebase accessible to students. Second, the fantasy console concept, while inspirational, rarely produces implementations that are themselves open for study and modification in the way that a self-contained Go library would be. Third, the computer science education community lacks concrete reference implementations of real-time interactive systems built in Go that demonstrate idiomatic use of the language's concurrency primitives, interface design patterns, and testing methodologies. Pixelforge was conceived to address all three of these concerns simultaneously.

## 2.3 Detailed Literature Review

### 2.3.1 Definitions

**Game Engine:** A game engine is a software framework designed to facilitate the development of video games by providing reusable, composable subsystems for rendering graphics, processing input, simulating physics, playing audio, and managing game state. Modern game engines abstract the underlying hardware to varying degrees, allowing developers to create games that run on multiple platforms without platform-specific modifications. The term encompasses engines ranging from highly specialized frameworks optimized for a single game genre to general-purpose engines capable of supporting AAA productions.

**Fantasy Console:** A fantasy console is a virtual machine that simulates a constrained hardware environment for game development, complete with its own display resolution, color palette, audio channels, and often an integrated development environment. The term, coined by Joseph White, the creator of Pico-8, reflects the hypothetical nature of the hardware being emulated—constraints are software-enforced rather than hardware-mandated. Fantasy consoles are designed to encourage creativity within bounded limits and to evoke the aesthetic qualities of retro computing hardware from the 1980s and 1990s.

**Software Rendering:** Software rendering refers to the process of generating computer graphics using the CPU rather than dedicated graphics hardware. In the context of 2D game engines, software rendering involves manipulating individual pixels in a framebuffer and performing operations such as sprite compositing, shape drawing, and color blending entirely in software. While less performant than GPU-based rendering for high-resolution applications, software rendering offers complete control over the pixel pipeline and is well-suited to low-resolution constrained environments.

**Color Palette:** A color palette is a finite set of colors from which a display system or image format selects its colors. In constrained graphics systems, the palette is typically a fixed array of predefined colors, and pixel values are indices into this array rather than arbitrary RGB triplets. Palette-based graphics reduce memory requirements and enable specific aesthetic effects, such as rapid color cycling and transparent color indexing, that are characteristic of retro game visuals.

**Observer Pattern:** The observer pattern is a software design pattern in which an object, known as the subject, maintains a list of dependents, called observers, and notifies them automatically of any state changes. In game engine architecture, the observer pattern is commonly used to decouple subsystems—for example, allowing the input system to publish key press events without the game logic having to poll for them directly. This pattern facilitates modularity and reduces coupling between engine components.

**Object Pooling:** Object pooling is a memory management technique in which a collection of pre-allocated objects is maintained and reused rather than allocating and deallocating objects on demand. In real-time systems such as game engines, object pooling mitigates the performance cost of garbage collection and allocation-related pauses by ensuring that frequently used objects, such as particle effects or GUI elements, are reused from a fixed-size buffer.

### 2.3.2 Related Research Work 1: Llorente (2024) — Optimization Techniques on Memory Management for Game Engines

Pablo Llorente's 2024 research conducted at UPC CITM focuses specifically on optimization techniques for memory management within game engines, with particular attention to resource management, multi-threading strategies, entity management systems, and floating-point arithmetic optimization. This work is directly relevant to Pixelforge's architectural decisions because it provides a systematic framework for understanding where memory pressure originates in game engines and how different subsystem designs affect allocation patterns.

Llorente's research identifies three primary sources of memory inefficiency in game engines: transient allocations during game loop execution, fragmentation caused by the allocation and deallocation of variable-sized game objects such as sprites and audio buffers, and cache coherence penalties arising from non-local memory access patterns during rendering and update passes. The author evaluates several mitigation strategies, including slab allocation for objects of uniform size, object pooling for frequently instantiated types, and data-oriented design principles that arrange game entities in memory according to access pattern rather than functional categorization.

The entity management discussion in Llorente's work is especially pertinent to Pixelforge's design philosophy. The research distinguishes between component-based entity systems, in which game objects are composed of small data structures attached to a common entity identifier, and the simpler monolithic approach in which each game object is a self-contained struct. While component-based systems offer greater flexibility for complex games, they impose overhead that is difficult to justify in constrained 2D environments where the entity count rarely exceeds the thousands. Pixelforge's approach of providing a coroutine-like execution system through `pixelforge_routine` draws on these trade-off considerations, offering a middle ground between full entity-component architecture and ad-hoc game object management.

The research on floating-point arithmetic optimization is less directly applicable to Pixelforge's 320 by 180 pixel environment but remains relevant for understanding precision considerations in real-time rendering. Llorente notes that fixed-point arithmetic was historically used to avoid floating-point overhead on architectures lacking FPU units, a concern that is resurgent in certain constrained deployment scenarios but largely irrelevant for modern Go deployments where floating-point operations are hardware-accelerated.

### 2.3.3 Related Research Work 2: Koirikivi (2025) — Architecture and Evolution of Computer Game Engines

Rainer Koirikivi's 2025 thesis from the University of Oulu presents a comprehensive historical survey of game engine architecture with a specific focus on how architectural trends have adapted to leverage modern hardware parallelism. This research situates game engine evolution within the broader context of hardware architecture, tracing how the shift from single-core to multi-core processors, the introduction of SIMD instruction sets, and the commoditization of GPU compute have each driven corresponding shifts in engine design.

The historical perspective Koirikivi provides is valuable for understanding why modern engines have converged on certain architectural patterns. The author documents the transition from the monolithic design of early engines such as the Doom engine, which tightly coupled rendering, physics, and game logic into a single executable, to the modular plugin architectures of contemporary engines that allow subsystems to be replaced, updated, or extended without modifying the core engine. This historical trajectory informs Pixelforge's own modular structure, in which each subsystem lives in its own Go package and communicates through well-defined interfaces. The `pixelforge_event` package, which implements an observer pattern for decoupled event communication, reflects the architectural insight that event-driven systems reduce inter-subsystem dependencies and facilitate incremental testing.

Koirikivi's analysis of parallelism strategies is particularly instructive. The author identifies three generations of parallel game engine architecture: the early generation that attempted to parallelize rendering and physics on separate threads with limited success due to synchronization overhead, the middleware generation that offloaded specific subsystems such as particle simulation and audio processing to dedicated threads, and the current generation that employs task-based work-stealing thread pools to distribute fine-grained operations across available cores. Pixelforge's deliberate choice to operate in a single-threaded manner—documented explicitly in the design philosophy of "intentional non-thread-safety"—can be understood as a conscious return to the first generation's simplicity for the specific context of a constrained 2D engine. The research supports the hypothesis that for a sufficiently small resolution and sufficiently bounded subsystem count, the overhead of thread synchronization exceeds the benefits of parallelization, making the single-threaded model not merely a simplification but a rational engineering decision.

The thesis also surveys the architectural implications of GPU-centric rendering pipelines, noting that the emergence of compute shaders has blurred the traditional boundary between graphics and general-purpose computation in game engines. While Pixelforge delegates rendering to Ebitengine rather than implementing its own GPU pipeline, the design of the color table system and the palette-based rendering model reflects an understanding of GPU-oriented texture and lookup table patterns.

### 2.3.4 Related Research Work 3: Mustonen (2023) — Web-Based Game Engine Design

Mikko Mustonen's 2023 thesis from Lappeenranta–Lahti University of Technology examines the specific requirements and challenges of designing game engines that target web platforms, with particular attention to the constraints imposed by browser security models, JavaScript engine performance characteristics, and the absence of low-level hardware access. This research provides a useful counterpoint to Pixelforge's approach, as it highlights the unique challenges that arise when the deployment target is a sandboxed browser environment rather than native hardware.

Mustonen identifies several key tensions in web-based game engine design: the desire for near-native performance versus the overhead of JavaScript's dynamic type system, the need for high-frequency rendering updates versus browser-imposed frame rate constraints and background tab throttling, and the ambition to support complex game mechanics versus the memory limits of individual browser tabs. The author evaluates several approaches to bridging the performance gap, including the use of WebAssembly as a compilation target, the adoption of off-screen canvas rendering with transfer to the main thread, and the use of GPU-backed 2D canvas contexts that delegate compositing to the GPU.

One of Mustonen's central findings is that the single most significant bottleneck in web-based 2D game engines is not rendering but input latency—the delay between a user action and the visual response in the game. This finding reinforces the importance of Pixelforge's separate input subsystem architecture, in which keyboard, mouse, and gamepad states are tracked independently and exposed through a unified interface. By decoupling input sampling from the rendering loop, Pixelforge ensures that input state is available immediately when needed rather than being tied to the next scheduled frame update.

Mustonen's discussion of the trade-offs between immediate mode and retained mode rendering architectures is also relevant. Immediate mode rendering, in which the application redraws the entire screen from scratch each frame, is better suited to simple 2D games with predictable frame rates but wastes computational resources on redrawing static UI elements. Retained mode rendering, which maintains a display list of objects to be drawn, is more efficient for complex scenes but imposes additional memory overhead and synchronization complexity. Pixelforge's surface-based rendering model, in which a 2D grid of pixels is maintained in memory and composited to the screen each frame, represents an immediate mode approach that is well-matched to the engine's educational mission—every pixel operation is visible and traceable, with no hidden display list or scene graph abstractions obscuring the flow of data.

### 2.3.5 Related Research Work 4: TIC-80 and the Broader Fantasy Console Ecosystem

Beyond academic literature, the open-source fantasy console ecosystem provides substantial reference material for understanding the design space Pixelforge occupies. TIC-80, described by its creators as a "fantasy computer," represents the most complete open-source implementation of the fantasy console concept. Unlike Pico-8, which is proprietary, TIC-80's source code is publicly available, making its internal architecture inspectable. TIC-80 implements a virtual machine with 64KB of memory, a 240 by 136 pixel display, and a 16-color palette. The engine supports sprites, maps, sound, and music, and includes a built-in code editor.

The architectural approach of TIC-80 differs from Pixelforge in instructive ways. TIC-80 operates as a self-contained virtual machine with its own bytecode interpreter, meaning that games written for TIC-80 are executed by a simulated CPU rather than compiled to native machine code. Pixelforge, by contrast, compiles to native Go binaries through standard Go toolchain, giving developers direct access to all of Go's standard library and tooling while constraining the display and audio subsystems to retro-appropriate dimensions. This design decision reflects a trade-off between the sandboxed safety of a VM-based approach and the performance and debugging power of native execution.

The broader fantasy console ecosystem, which includes projects such as Pixel Vision 8, NESbox, and the various Pico-8 emulators that have been ported to microcontrollers, demonstrates the diversity of approaches to constrained game development. Pixel Vision 8 is particularly notable for its configurability, allowing developers to specify custom display resolutions, color depths, and audio channel counts. This configurability is the opposite of Pico-8's rigid constraints and suggests that the value of fantasy consoles lies not in any specific set of limitations but in the pedagogical principle that constraints clarify design decisions and force creative problem-solving.

### 2.3.6 Related Research Work 5: Ebitengine and Native 2D Game Development in Go

The existence of Pixelforge depends fundamentally on Ebitengine, the mature 2D game library for Go that serves as Pixelforge's rendering backend. Ebitengine, originally known as Ebiten before its renaming to Ebitengine, was created by Hajime Hoshi and has been under active development since 2013. It is the primary demonstration that real-time interactive applications with graphics and audio are achievable in Go with performance characteristics acceptable for 2D games.

Ebitengine provides Pixelforge with a hardware abstraction layer that handles window management, input event translation, and the final compositing of Pixelforge's internal pixel buffer to the screen. Pixelforge's relationship to Ebitengine is analogous to the relationship between a custom software renderer and a GPU—Pixelforge performs all pixel manipulation in software using Go code and memory buffers, then transfers the completed frame to Ebitengine for display. This architecture ensures that every visual artifact visible in a Pixelforge game is the result of explicit Go code executing in a predictable sequence, which is essential for the educational objectives of the project.

Ebitengine itself has been the subject of optimization research within the Go community. Several studies have measured Ebitengine's performance across different game types and identified bottlenecks in its rendering pipeline, including the cost of converting between Go's slice-based pixel representation and the internal formats expected by the operating system's graphics APIs. Pixelforge's approach of minimizing per-frame allocations and using pre-computed color tables and lookup structures is informed by these performance considerations.

## 2.4 Literature Review Summary Table

The following table summarizes the key findings from the reviewed literature and maps them to the design decisions in Pixelforge.

| Reference | Year | Focus Area | Key Findings | Relevance to Pixelforge |
|---|---|---|---|---|
| Llorente, P. | 2024 | Memory management optimization in game engines | Transient allocations, fragmentation, cache coherence; slab allocation and object pooling as mitigation strategies | Informs `pixelforge_pool` object pooling subsystem and single-threaded design to avoid synchronization overhead |
| Koirikivi, R. | 2025 | Architecture and evolution of game engines | Historical shift from monolithic to modular architectures; parallelism trade-offs; task-based work distribution | Supports Pixelforge's modular package design and deliberate single-threaded model for constrained 2D workloads |
| Mustonen, M. | 2023 | Web-based game engine design | Input latency as primary bottleneck; immediate mode vs retained mode trade-offs; WebAssembly performance characteristics | Validates separate input subsystem architecture; informs surface-based immediate mode rendering model |
| TIC-80 Community | 2014–present | Open-source fantasy console | VM-based execution model; configurable constraints; retro aesthetics as pedagogical tool | Contrasts with Pixelforge's native Go compilation model; validates constrained display philosophy |
| Ebitengine Documentation | 2013–present | Native 2D game engine for Go | Go's viability for real-time 2D graphics; rendering backend abstraction; performance characteristics | Provides technical foundation for Pixelforge's rendering backend integration |

## 2.5 Research Gap

The surveyed literature reveals a specific gap in the intersection of game engine education, fantasy console design, and Go-based systems programming. While Llorente's work provides a theoretical framework for memory optimization and Koirikivi's historical survey contextualizes architectural evolution, neither addresses the practical problem of creating a comprehensive, tutorial-grade reference implementation that covers all major engine subsystems in a single coherent codebase. Mustonen's web-focused research, while valuable, concerns itself with the unique constraints of browser deployment rather than native desktop applications.

The fantasy console ecosystem, exemplified by Pico-8 and TIC-80, provides inspirational examples of constraint-based creativity but most are implemented as self-contained virtual machines with custom bytecode interpreters, making their internals opaque to developers who wish to inspect, modify, or extend them. The few open-source implementations lack the modern software engineering practices—comprehensive testing, idiomatic Go patterns, clear package boundaries—that would make them useful as educational references for university-level computer science instruction.

Finally, while Ebitengine has demonstrated that Go is a viable language for 2D game development, the ecosystem lacks a layered abstraction that allows students and developers to work at the pixel level without sacrificing the benefits of a well-tested underlying engine. Pixelforge occupies this gap by providing a from-scratch implementation of game engine subsystems that builds directly on Ebitengine rather than reimplementing what Ebitengine already provides, thereby achieving both educational depth and practical utility.

## 2.6 Problem Statement

Based on the gaps identified in the literature, the problem that Pixelforge addresses can be formally stated as follows: there exists no comprehensive, self-contained, educational game engine implementation written in Go that provides pixel-level rendering, multi-channel audio, event-driven input handling, and embedded developer tooling within a modular, fantasy-console-inspired architecture that is itself fully inspectable, modifiable, and usable as both a functional game development framework and a teaching instrument for computer science courses covering real-time systems, graphics programming, and software architecture.

Pixelforge's contribution is the delivery of a complete implementation of such a system, encompassing all major subsystems of a 2D game engine, designed with explicit attention to readability, extensibility, and performance, and evaluated both through automated testing and practical game development examples.

---

The remainder of this report is organized as follows. Chapter 3 presents the system architecture and design, describing the high-level structure of the engine, the relationships between subsystems, the key data structures employed, and the rationale for significant design decisions. Chapter 4 details the implementation of each engine subsystem, including the core rendering pipeline, the audio player, the input handling system, the event management framework, and the developer tools. Chapter 5 evaluates the completed system through performance benchmarking, correctness validation, and a usability study conducted with developers unfamiliar with the engine. Chapter 6 discusses potential extensions and future work, including multi-backend rendering support, additional audio channels, networked multiplayer capabilities, and deeper integration with Go's native concurrency features.# Chapter 3: Requirements and Design

## 3.1 Introduction

This chapter describes the complete requirements and architectural design of Pixelforge, a from-scratch 2D pixel game engine written in Go. The engine is inspired by the Pico-8 fantasy console and is designed to serve both as a functional game development framework and as an educational instrument for computer science students studying real-time systems, graphics programming, and software architecture. The chapter begins by enumerating the functional and non-functional requirements that guided the system’s construction, proceeds to describe the proposed methodology and system architecture, and concludes with use cases, design artifacts, and GUI specifications. All design decisions documented here are directly traceable to the goals and objectives established in Chapter 1 and the literature reviewed in Chapter 2.

## 3.1 Requirements

### 3.1.1 Functional Requirements

**FR-1: Pixel-Level Rendering.** The system shall provide an imperative API for setting and reading individual pixels on a 320×180 virtual display surface, supporting a fixed 64-color palette indexed from 0 to 63. Each pixel operation shall respect camera offset, clipping regions, and color table compositing rules.

**FR-2: Shape Drawing Primitives.** The system shall support drawing axis-aligned rectangles (outline and filled), lines using Bresenham’s line algorithm, and circles (outline and filled) using the midpoint circle algorithm. All shape operations shall operate on the current draw target and respect the active color, clipping, and masking state.

**FR-3: Sprite Rendering.** The system shall support creation, storage, and rendering of sprites defined as rectangular regions of a canvas. Sprites shall support horizontal and vertical flipping, arbitrary scaling and stretching via direct index calculation, and transparency through color table bit manipulation.

**FR-4: Color Palette Management.** The system shall maintain a global 64-entry color palette mapping each index to a 24-bit RGB value. It shall also maintain four 64×64 color tables indexed by bits 6–7 of source and target color values, supporting transparency, remapping, and blending effects without modifying source pixel data.

**FR-5: Multiple Render Targets.** The system shall support setting an arbitrary `Surface[Color]` as the current draw target, enabling layered rendering, off-screen buffers, and post-processing effects. The screen shall be accessible as a special canvas via `Screen()`.

**FR-6: Input Handling – Keyboard.** The system shall track the state of 75 virtual keys, including printable characters, modifiers, and function keys. It shall provide a `Duration(Key)` function returning the number of consecutive frames a key has been held, and shall publish "down" and "up" events through an observer-pattern event target.

**FR-7: Input Handling – Mouse.** The system shall track mouse position (translated by camera offset), movement delta since the last frame, and left/right button states. It shall publish button press/release and movement events through dedicated event targets.

**FR-8: Input Handling – Gamepad.** The system shall support up to 16 connected gamepad controllers, each with a standardized button mapping (A, B, X, Y, Left, Right, Top, Bottom, Start, Select, shoulder buttons, and directional pad). It shall provide per-controller duration tracking and publish button and connection events.

**FR-9: Audio Playback – 4-Channel Mixer.** The system shall provide a four-channel audio player inspired by the Amiga Paula chip. Each channel shall support independently setting a PCM sample, pitch (playback rate), volume (0.0 to 1.0), and loop mode. Channels 0 and 3 shall be mixed to the left stereo channel; channels 1 and 2 to the right.

**FR-10: Audio Scheduling.** The system shall support scheduling sample, pitch, volume, and loop commands at a future time measured in seconds, enabling precise musical sequencing without frame-rate dependency. The backend shall clone samples to prevent garbage collection issues during playback.

**FR-11: Event System.** The system shall provide a generic, zero-allocation observer-pattern implementation (`pievent.Target[T]`) supporting publish/subscribe, event-specific and wildcard subscription, and optional call-site tracing for debugging.

**FR-12: Game Loop Lifecycle.** The system shall provide fixed-TPS (ticks per second) game loop events: `EventInit`, `EventFrameStart`, `EventUpdate`, `EventLateUpdate`, `EventDraw`, `EventLateDraw`, and `EventWindowClose`. Third-party code shall be able to subscribe to these events to inject logic at specific points in the loop.

**FR-13: Coroutine System.** The system shall provide a coroutine-like execution mechanism (`piroutine.Routine`) allowing game logic to be decomposed into discrete `Step` functions, each returning a boolean indicating completion. The routine shall support resuming on frame updates, waiting N frames, and executing logic at configurable frame intervals.

**FR-14: Bitmap Font Rendering.** The system shall support rendering text using bitmap font sheets composed of individual character sprites. It shall provide foreground/background color remapping, outline (stroke) rendering via multi-pass drawing, and text size measurement.

**FR-15: GUI Element System.** The system shall provide a minimal immediate-mode GUI framework with a hierarchical element tree. Each element shall support position, size, draw callbacks, update callbacks, and press/release/tap interaction callbacks. Coordinate translation for child elements shall be handled automatically via camera manipulation.

**FR-16: Developer Tools – Frame Debugger (Piscope).** The system shall provide an integrated developer overlay activated by Ctrl+Shift+I, offering pause/resume (Spacebar), single-frame stepping (Left/Right arrows), screenshot capture (F12), and exit (Esc). It shall record frame history for step-through debugging.

**FR-17: Developer Tools – Performance Monitor.** The system shall provide real-time CPU usage (percentage) and resident memory (MB) monitoring, refreshed every 500ms, subscribing to debug loop events so that metrics remain available during pause.

**FR-18: Developer Tools – Screenshot Capture.** The system shall support capturing the current screen as a paletted PNG image and saving it to a temporary directory, accessible programmatically via `PalettedImage()`.

**FR-19: Ebitengine Backend Integration.** The system shall delegate window management, OS-level input event translation, and final frame compositing to Ebitengine, which implements Go’s `ebiten.Game` interface. The backend shall translate Ebitengine key codes, mouse state, and gamepad state into Pixelforge’s virtual input system.

**FR-20: Object Pooling.** The system shall provide a generic object pool (`Pool[T]`) for reducing heap allocations, supporting `Get()` and `Put()` operations in a LIFO manner. The pool shall be explicitly non-thread-safe, consistent with the engine’s single-goroutine design.

### 3.1.2 Non-Functional Requirements

**NFR-1: Single-Threaded Performance.** All engine APIs shall be callable only from the main game loop goroutine. The system shall be intentionally non-thread-safe to eliminate synchronization overhead and maximize single-core performance. A goroutine ID check shall panic if API calls are made from other goroutines.

**NFR-2: Zero or Minimal Allocation in Hot Paths.** Event publishing, pixel setting, shape drawing, and input queries shall not allocate heap memory in typical usage. Surfaces shall be pre-allocated, and object pools shall be used for frequently instantiated types such as GUI propagation tokens.

**NFR-3: Modular Composability.** Each subsystem (audio, keyboard, mouse, gamepad, GUI, scope, etc.) shall reside in its own Go package under the `pixelforge_*` namespace. Packages shall communicate through `pievent.Target` interfaces and shall not import each other unnecessarily, enabling users to pick only the subsystems they need.

**NFR-4: Bounded Problem Space.** The engine shall enforce a maximum surface size of 131,072 pixels (e.g., 320×180, 640×90, 256×512). The color depth shall be fixed at 8 bits per pixel (64 indexed colors), and coordinates shall use `int` (signed 32-bit or 64-bit depending on platform) to match Go’s native integer type.

**NFR-5: Cross-Platform Rendering.** Through Ebitengine, the engine shall support Windows, macOS, Linux, and web browser deployment targets without platform-specific code in the engine packages themselves.

**NFR-6: Readability and Educational Clarity.** All exported types, functions, and constants shall have concise documentation comments. Internal algorithms (Bresenham, midpoint circle, color distance) shall be written clearly rather than in heavily optimized but opaque forms, supporting use as teaching material.

**NFR-7: Test Coverage.** Core packages shall have unit tests covering pixel operations, shape drawing, color table lookups, event subscription, object pool lifecycle, and ring buffer operations. Tests shall use the standard Go testing package and `testify` assertions where appropriate.

### 3.1.3 Hardware and Software Requirements

**Development Environment:**
- **Language:** Go 1.21 or later (uses `iter.Seq2` for `LinesIterator`, generics throughout)
- **Rendering Backend:** Ebitengine v2 (github.com/hajimehoshi/ebiten/v2)
- **CPU:** Any architecture supported by Go and Ebitengine (x86-64, ARM64, 386)
- **RAM:** Minimum 256MB available (engine itself uses <10MB; Ebitengine and OS requirements dominate)
- **GPU:** Not required; rendering is software-based with final compositing through Ebitengine’s GPU-accelerated 2D backend
- **OS:** Linux (primary development), Windows, macOS

**Deployment Targets:**
- Native executables for Windows, macOS, Linux
- WebAssembly (via Ebitengine’s `wasm` target)
- Raspberry Pi (via Ebitengine’s ARM support)

**Third-Party Dependencies:**
- `github.com/hajimehoshi/ebiten/v2` — Rendering, windowing, input, audio backend
- `github.com/shirou/gopsutil/v4` — CPU/memory metrics in `pixelforge_stat`
- `github.com/stretchr/testify` — Test assertions (development only)
- Standard library packages: `image`, `image/color`, `image/png`, `math`, `sync`, `time`, `os`

## 3.2 Proposed Methodology

The development of Pixelforge followed an iterative, subsystem-by-subsystem methodology grounded in the established software engineering principle of separation of concerns. Rather than adopting a monolithic "big bang" integration approach, each engine capability was designed, implemented, and tested as an independent Go package before being integrated into the unified engine experience.

The methodology comprised the following phases:

**Phase 1: Core Data Structures.** The foundational types—`Surface[T]`, `Canvas`, `Color`, `Position`, `Area[T]`, and `Sprite`—were designed first. A key methodological decision was the use of Go generics (`Surface[T any]`) to create a single 2D grid implementation reusable for pixel data, intermediate buffers, and custom game data grids.

**Phase 2: Software Rendering Pipeline.** The pixel-level operations (`SetPixel`, `GetPixel`), shape algorithms (Bresenham line, midpoint circle), and sprite rendering with flipping and stretching were implemented directly on `Surface[Color]`. The color table system was designed simultaneously, using a `[64][64]Color` array indexed by `(source|target) >> 6` to enable transparent, remappable compositing without branching in the hot path.

**Phase 3: Event System and Backend Abstraction.** The generic `pievent.Target[T]` was implemented as a zero-allocation observer pattern. Simultaneously, the Ebitengine backend (`pixelforge_ebiten`) was developed to implement the `ebiten.Game` interface, bridging Go’s pixel buffers to hardware-accelerated display. Input and audio backend interfaces were defined, allowing the pixelforge packages to remain backend-agnostic.

**Phase 4: Input Subsystems.** Keyboard (`pixelforge_key`), mouse (`pixelforge_mouse`), and gamepad (`pixelforge_pad`) were implemented as independent packages, each using `input.State[T]` for duration tracking and `pievent.Target` for event publication. The `internal/input` package was created to share the duration-tracking data structure across all three input packages without creating circular dependencies.

**Phase 5: Audio Subsystem.** The four-channel audio player (`pixelforge_audio`) was implemented with a command-scheduling architecture. A ring buffer (`internal/pixelforge_ring.Buffer`) was used for the audio command queue, and the Amiga Paula-inspired stereo mixing (channels 0,3 → left; 1,2 → right) was implemented in the Ebitengine audio callback.

**Phase 6: Developer Tools.** The frame debugger (`pixelforge_scope/piscope`), performance monitor (`pixelforge_stat`), and screenshot capture (`pixelforge_snap`) were built on top of the event system and the pause/resume primitives provided by `pixelforge_debug`.

**Phase 7: GUI and High-Level APIs.** The GUI element tree (`pixelforge_gui`), coroutine system (`pixelforge_routine`), bitmap font rendering (`pixelforge_font`), and Pico-8 built-in font (`pixelforge_cofont`) were implemented to provide game developers with higher-level building blocks.

**Phase 8: Integration and Examples.** Example games (snake, hello, shapes, audio/piano, gamepad, gui) were created to validate the integrated system and to serve as tutorials for new users.

This methodology ensured that each package had a well-defined responsibility, clear interfaces, and the ability to be tested and modified independently of the others—a direct application of the modular architectural principles discussed by Koirikivi [1].

## 3.3 System Architecture

### 3.3.1 High-Level Architecture

Pixelforge follows a layered architecture with three distinct layers: the **Application Layer** (user games), the **Engine Core Layer** (pixelforge, event system, GUI), and the **Backend Layer** (pixelforge_ebiten, Ebitengine, OS).

```figure
![High-Level Architecture](diagrams/arch.png)
**Figure 3.1:** High-Level Architecture. The engine consists of three layers: User Game Code, Pixelforge Core (pixelforge package), and the Backend Layer (pixelforge_ebiten and Ebitengine). The pievent event system provides decoupled communication between all subsystems.
```

### 3.3.2 Data Flow Diagram

The following diagram illustrates the flow of data during a single frame of Pixelforge execution.

```figure
![Frame Data Flow](diagrams/seq_dataflow.png)
**Figure 3.2:** Frame Data Flow. During each 60 TPS update cycle, Ebitengine triggers Update(), which publishes loop events, polls input, and executes user logic. The Draw phase renders to the screen canvas, which is copied to an Ebitengine image for GPU compositing. The audio callback mixes four channels at 44.1 kHz independently of the frame loop.
```

### 3.3.3 Package Dependency Structure

```figure
![Package Dependency Structure](diagrams/deps.png)
**Figure 3.3:** Package Dependency Structure. All pixelforge_* packages are independent and communicate through pievent (observer pattern) and backend interfaces. The pixelforge_ebiten package bridges Pixelforge's virtual input and audio to Ebitengine's OS-level APIs. Modules with "-gui" and "-debug" suffixes depend only on the core engine and event system respectively.
```

### 3.3.4 Generic Data Structure Design

The `Surface[T]` type is the foundational data structure of Pixelforge. It is a generic 2D grid parameterized by type `T`, enabling the same implementation to serve as a pixel canvas (`Surface[Color]`), a propagation token pool (`Surface[GuiToken]`), or any other grid-based game data.

```figure
![Generic Data Structures](diagrams/surface_class.png)
**Figure 3.4:** Generic Data Structures. The `Surface[T]` type is a 2D grid parameterized by type T. `Canvas` is a type alias for `Surface[Color]`. `Sprite` embeds an `Area[int]` and references a source `Canvas`, with `FlipX` and `FlipY` flags for horizontal and vertical flipping during rendering.
```

### 3.3.5 Color System Architecture

The color system uses an 8-bit indexed color model. The lower 6 bits (0–63) select the color from the global `Palette[64]RGB`, while bits 6–7 select one of four color tables that control compositing behavior.

```figure
![Color System Architecture](diagrams/color_system.png)
**Figure 3.5:** Color System Architecture. Each pixel value (8-bit Color) has its lower 6 bits selecting the RGB color from the global 64-entry palette array, and bits 6-7 selecting one of four 64x64 color tables. ColorTable[0] is special-cased for transparency; the other three tables enable remapping and blending without modifying source pixel data.
```

## 3.4 Use Cases

### 3.4.1 Run Minimal Hello World Game

**Name:** Run Minimal Hello World Game

**Actors:** Developer (Game Programmer), Pixelforge Engine

**Summary:** The developer creates a minimal game that displays "HELLO WORLD" using the built-in Pico-8 font, then launches it via the Ebitengine backend.

**Pre-Conditions:**
- Go development environment is installed with pixelforge and pixelforge_ebiten modules available.
- The developer has created a `main.go` file.

**Post-Conditions:**
- A window opens displaying the text "HELLO WORLD" at position (2,2) on a 47×9 pixel screen.
- The game runs at the configured TPS (default 30).

**Basic Flow:**

| Actor Action | System Response |
|--------------|-----------------|
| 1. Developer writes `pixelforge.SetScreenSize(47, 9)` | 1. Screen canvas is allocated at 47×9 pixels |
| 2. Developer sets `pixelforge.Draw = func() { pixelforge_cofont.Print("HELLO WORLD", 2, 2) }` | 2. Draw callback is registered |
| 3. Developer calls `pixelforge_ebiten.Run()` | 3. Ebitengine window opens, game loop starts |
| 4. Frame update occurs | 4. `EventDraw` published, Draw callback executes, text rendered to screen canvas |
| 5. Ebitengine composites canvas to window | 5. "HELLO WORLD" visible in window |

**Alternative Flow:**
| 3. Developer forgets to call `pixelforge_ebiten.Run()` | Game does not start; no window appears |

### 3.4.2 Draw Shapes Interactively with Mouse

**Name:** Draw Shapes Interactively with Mouse

**Actors:** User (Playing/Testing the Game), Pixelforge Engine

**Summary:** The user moves the mouse to position shapes on screen, left-clicks to place a shape (rect, circle, or line), and right-clicks to cycle through shape types. The selected shape previews at the mouse cursor position.

**Pre-Conditions:**
- The shapes example game is running.
- A palette has been loaded from a sprite sheet PNG.

**Post-Conditions:**
- Shapes are drawn on the screen at the clicked positions.
- The current shape type cycles on right-click.

**Basic Flow:**

| Actor Action | System Response |
|--------------|-----------------|
| 1. User moves mouse | 1. `pixelforge_mouse.Position` updates; shape previews at cursor |
| 2. User left-clicks at (x, y) | 2. Current shape (rect/circle/line) drawn at (x, y) |
| 3. User right-clicks | 3. Shape selection cycles to next type |
| 4. Repeat steps 1–3 | 4. Multiple shapes accumulate on screen |

### 3.4.3 Play Snake Game with Keyboard and Gamepad

**Name:** Play Snake Game with Keyboard and Gamepad

**Actors:** Player, Pixelforge Engine, Keyboard Subsystem, Gamepad Subsystem

**Summary:** The player controls a snake character using either keyboard arrow keys or gamepad directional pad. The snake grows when it eats food, and the game ends if the snake collides with itself or the screen boundary.

**Pre-Conditions:**
- Snake example game is running.
- A gamepad is optionally connected.

**Post-Conditions:**
- Snake moves in the currently selected direction.
- Score increases when food is consumed.
- Game over screen appears on collision.

**Basic Flow:**

| Actor Action | System Response |
|--------------|-----------------|
| 1. Player presses arrow key or gamepad direction | 1. Snake direction variable updated in `pixelforge.Update` callback |
| 2. Frame updates | 2. Snake head moves one pixel in current direction |
| 3. Snake head reaches food position | 3. Food consumed, snake length increases, new food placed |
| 4. Snake head collides with body or wall | 4. Game over state activated, score displayed |

### 3.4.4 Play Piano with Audio Scheduling

**Name:** Play Piano with Audio Scheduling

**Actors:** Player, Pixelforge Audio Subsystem, Ebitengine Audio Backend

**Summary:** The player presses keys (A, B, C, D, E, F, G) to play piano notes. Each key triggers a sample playback with appropriate pitch on one of the four audio channels, with volume and looping configured via scheduled commands.

**Pre-Conditions:**
- Audio example (piano) is running.
- Samples have been loaded via `pixelforge_audio.LoadSample()`.

**Post-Conditions:**
- Pressing a key plays the corresponding note.
- Releasing a key stops the note (or allows it to fade via scheduled volume change).

**Basic Flow:**

| Actor Action | System Response |
|--------------|-----------------|
| 1. Player presses 'C' key | 1. `pixelforge_key.Duration("C") > 0` detected in Update; `pixelforge_audio.Play(channel, sample, pitch, volume)` called |
| 2. Audio callback fires | 2. Channel 0 mixes sample at specified pitch and volume to stereo output |
| 3. Player releases 'C' key | 3. `pixelforge_audio.SetVolume(channel, 0, 0)` scheduled |
| 4. Player presses gamepad button | 4. Same as step 1, using `pixelforge_pad.Duration()` |

### 3.4.5 Use Developer Tools to Debug Frame by Frame

**Name:** Use Developer Tools to Debug Frame by Frame

**Actors:** Developer, Pixelforge Scope (Piscope), Debug Subsystem

**Summary:** The developer activates the integrated developer tools with Ctrl+Shift+I, then uses Spacebar to pause, arrow keys to step through frames, and F12 to capture a screenshot of the current frame.

**Pre-Conditions:**
- Game is running normally.
- Developer tools have been started via `piscope.Start()`.

**Post-Conditions:**
- Game is paused and can be stepped frame by frame.
- Screenshots are saved to the temp directory.
- Developer can resume normal execution with Spacebar.

**Basic Flow:**

| Actor Action | System Response |
|--------------|-----------------|
| 1. Developer presses Ctrl+Shift+I | 1. Piscope overlay appears; game continues running |
| 2. Developer presses Spacebar | 2. Game pauses; `pixelforge_debug.SetPaused(true)` called |
| 3. Developer presses Right arrow | 3. One frame advances; `EventUpdate` and `EventDraw` fire once |
| 4. Developer presses F12 | 4. `pixelforge_snap.CaptureOrErr()` saves screenshot to temp file |
| 5. Developer presses Spacebar again | 5. Game resumes normal execution |

## 3.5 Database Design (Optional)

Pixelforge does not use a traditional database. All persistent state is managed through the following in-memory data structures:

**Canvas (Surface[Color]):** A 2D grid of 8-bit color indices. The screen canvas is the primary persistent surface, and games may create additional off-screen canvases.

**Sprite Sheets:** Stored as `Surface[Color]` objects with attached `Area[int]` metadata defining individual sprite boundaries.

**Audio Samples:** Stored as `*pixelforge_audio.Sample` structures containing raw 8-bit mono PCM data and sample rate.

**Font Sheets:** Stored as `pixelforge_font.Sheet` structures mapping `rune` characters to `pixelforge.Sprite` objects.

```figure
![In-Memory Data Model](diagrams/erd.png)
**Figure 3.6:** In-Memory Data Model. Pixelforge maintains five primary in-memory structures: CANVAS (a 2D grid of Color indices), SPRITE (rectangular regions referencing a Canvas), SAMPLE (raw PCM audio data), FONT-SHEET (a map of rune characters to Sprite glyphs), and the PALETTE-COLOR-TABLE pair that maps indexed colors to display colors.
```

## 3.6 Class Diagram

```figure
![Class Diagram — Core Packages](diagrams/class_main.png)
**Figure 3.7:** Class Diagram — Core Packages. The pixelforge package exposes the imperative drawing API. The pievent.Target interface provides generic publish-subscribe for decoupled communication. pixelforge_audio manages four PCM channels with scheduling. pixelforge_key and pixelforge_pad publish input events. piroutine.Routine decomposes logic into step functions. All GUI elements share the same Element tree structure.
```

## 3.7 Sequence Diagram — Audio Playback

```figure
![Audio Playback Sequence](diagrams/seq_audio.png)
**Figure 3.8:** Audio Playback Sequence. Game code calls LoadSample and Play on pixelforge_audio. Commands are appended to a ring buffer via NextWritePointer. The audio callback fires at 44.1 kHz, reads scheduled commands, configures the four PCM channels, and mixes channels 0 and 3 to the left stereo output and channels 1 and 2 to the right.
```

## 3.8 Other Artifacts — Color Table Lookup

The color table system is a critical artifact that enables transparent pixels and palette remapping without conditional branching in the rendering hot path.

```figure
![Color Table Compositing](diagrams/color_system.png)
**Figure 3.9:** Color Table Compositing. A pixel value encodes both the palette index (lower 6 bits) and a color table selector (bits 6-7). The table selector indexes into one of four 64-entry color tables; the palette index selects the output color within that table. When tableIndex equals 0 and outputColor equals 0, the pixel is transparent and skipped entirely.
```

## 3.9 GUI Graphical User Interfaces

### 3.9.1 Main Game Window (Hello World Example)

**Screenshot Description:** A minimal window (47×9 pixels, scaled by Ebitengine) displaying white text "HELLO WORLD" on a black background (colors 0 and 1 from the Pico-8 palette). The window title bar shows the OS default. This covers the basic rendering and font use case.

### 3.9.2 Shapes Example GUI

**Screenshot Description:** A 320×180 window with a purple background (color 2). Multiple shapes (rectangles, filled circles, lines) are drawn in various Pico-8 palette colors. The current shape type is displayed as text. Mouse position is shown as a small crosshair sprite. This covers mouse input, shape drawing, and sprite rendering use cases.

### 3.9.3 Snake Game GUI

**Screenshot Description:** A 320×180 window with a dark blue background. The snake is rendered as a series of colored sprites (color 11 for head, color 12 for body). Food appears as a different sprite (color 10). Score is displayed in the top-left using the Pico-8 font. Game over screen overlays with a translucent rectangle and text. This covers game logic, keyboard/gamepad input, and game state management.

### 3.9.4 Developer Tools Overlay (Piscope)

**Screenshot Description:** When activated, a toolbar appears at the top of the game window (background color 8, foreground color 1). It displays:
- **FPS:** Current frames per second
- **Frame:** Current frame number
- **CPU:** Current CPU usage percentage
- **MEM:** Resident memory in MB
- **Controls:** [Space] Pause/Resume, [←→] Step, [F12] Screenshot, [Esc] Exit

This covers the developer tools use case and integrates with `pixelforge_debug`, `pixelforge_stat`, and `pixelforge_snap`.

### 3.9.5 GUI Example — Panel with Button

**Screenshot Description:** A 320×180 game window with a panel (64×64 pixels, color 5) attached at position (32,32). A button (56×10 pixels, color 6 with stroke color 1) is attached to the panel at (4,4) with the label "CLICK ME". When the button is pressed, it changes appearance (color 7), and a tap event fires, printing "Button clicked!" to the console. This covers the `pixelforge_gui` use case.

### Navigation Flow

```figure
![GUI Navigation Flow](diagrams/gui_nav.png)
**Figure 3.10:** GUI Navigation Flow. The game Draw callback branches based on the active example. The GUI example uses a panel with an attached button that fires an OnTap event when pressed. The Ctrl+Shift+I shortcut toggles the Piscope overlay, which provides pause, step, and screenshot controls.
```

## 3.10 Conclusions

Chapter 3 has presented the complete requirements and architectural design of the Pixelforge engine. The functional requirements (FR-1 through FR-20) establish a comprehensive surface area covering pixel rendering, shape primitives, sprite handling, color palette management, multi-channel audio, input subsystems, event-driven architecture, coroutines, font rendering, GUI elements, and integrated developer tools. The non-functional requirements (NFR-1 through NFR-7) codify the engine’s design philosophy: single-threaded performance, minimal allocations, modular composability, bounded problem space, cross-platform support, educational clarity, and test coverage.

The proposed methodology was an iterative, subsystem-by-subsystem approach that allowed each package to be designed and tested independently before integration—a direct application of the modular architectural principles found in the literature [1]. The system architecture was described through layered diagrams showing the application, engine core, and backend layers; sequence diagrams illustrating frame data flow and audio playback; and class diagrams capturing the relationships between core types.

Use cases covering minimal games, interactive shape drawing, snake gameplay, piano audio, and developer tool debugging demonstrated how the requirements translate into concrete user-facing functionality. The optional artifacts—database design (in-memory structures), class diagrams, sequence diagrams, and GUI screenshots—provide a complete specification sufficient for any developer to reproduce the system. The next chapter will detail the actual implementation of each subsystem, including the specific algorithms, code patterns, and testing strategies employed.

---

## References for Chapter 3

[1] R. Koirikivi, "Architecture and Evolution of Computer Game Engines," M.S. thesis, Univ. Oulu, Oulu, Finland, 2025.

[2] P. Llorente, "Optimization Techniques on Memory Management for Game Engines Resource Management, Multi-threading, Entity Management & Floating-Point Arithmetic," UPC CITM, Barcelona, Spain, 2024.

[3] M. Mustonen, "Web-Based Game Engine Design," Lappeenranta–Lahti Univ. Technol., Lappeenranta, Finland, 2023.

[4] J. White, "The Modest Fantasy of the PICO-8," Paste Magazine, Jan. 2016. [Online]. Available: https://www.pastemagazine.com/games/the-modest-fantasy-of-the-pico-8

[5] H. Hoshi, "Ebitengine – A Dead Simple 2D Game Engine for Go," [Online]. Available: https://ebitengine.org/

[6] "PICO-8 Fantasy Console," Lexaloffle Games. [Online]. Available: https://www.lexaloffle.com/pico-8.php

[7] "Godot Engine," [Online]. Available: https://godotengine.org/

[8] ITU-R Recommendation BT.601, "Studio Encoding Parameters of Digital Television for Standard 4:3 and Wide-Screen 16:9 Aspect Ratios," Int. Telecommun. Union, Geneva, Switzerland, 1994.

[9] E. Gamma, R. Helm, R. Johnson, and J. Vlissides, *Design Patterns: Elements of Reusable Object-Oriented Software*. Reading, MA, USA: Addison-Wesley, 1994.

[10] "Go Language Specification – Type Parameters," [Online]. Available: https://go.dev/ref/spec#Type_parameters
# References

## Chapter 2 References

[1] P. Llorente, "Optimization Techniques on Memory Management for Game Engines: Resource Management, Multi-threading, Entity Management & Floating-Point Arithmetic," UPC CITM, Barcelona, Spain, 2024.

[2] R. Koirikivi, "Architecture and Evolution of Computer Game Engines: Architectural Trends and Utilization of Parallelism with Modern Hardware," Univ. Oulu, Oulu, Finland, 2025.

[3] M. Mustonen, "Web-Based Game Engine Design: Solutions to Common Problems in Browser-Based Game Development," Lappeenranta–Lahti Univ. Technol., Lappeenranta, Finland, 2023.

[4] "PICO-8 Fantasy Console," Lexaloffle Games. [Online]. Available: https://www.lexaloffle.com/pico-8.php

[5] "List of Game Engines," Wikipedia. [Online]. Available: https://en.wikipedia.org/wiki/List_of_game_engines

[6] "Godot (Game Engine)," Wikipedia. [Online]. Available: https://en.wikipedia.org/wiki/Godot_(game_engine)

[7] H. Hoshi, "Ebitengine – A Dead Simple 2D Game Engine for Go," [Online]. Available: https://ebitengine.org/

[8] J. White, "The Modest Fantasy of the PICO-8," Paste Magazine, Jan. 2016. [Online]. Available: https://www.pastemagazine.com/games/the-modest-fantasy-of-the-pico-8

[9] N. Altice, "PICO-8: Gaming's Fantasy Console," Retro Gamer, no. 221, pp. 64, June 2021.

## Chapter 3 References

[10] R. Koirikivi, "Architecture and Evolution of Computer Game Engines," M.S. thesis, Univ. Oulu, Oulu, Finland, 2025.

[11] P. Llorente, "Optimization Techniques on Memory Management for Game Engines," UPC CITM, Barcelona, Spain, 2024.

[12] M. Mustonen, "Web-Based Game Engine Design," Lappeenranta–Lahti Univ. Technol., Lappeenranta, Finland, 2023.

[13] J. White, "The Modest Fantasy of the PICO-8," Paste Magazine, Jan. 2016. [Online]. Available: https://www.pastemagazine.com/games/the-modest-fantasy-of-the-pico-8

[14] H. Hoshi, "Ebitengine – A Dead Simple 2D Game Engine for Go," [Online]. Available: https://ebitengine.org/

[15] "PICO-8 Fantasy Console," Lexaloffle Games. [Online]. Available: https://www.lexaloffle.com/pico-8.php

[16] "Godot Engine," [Online]. Available: https://godotengine.org/

[17] ITU-R Recommendation BT.601, "Studio Encoding Parameters of Digital Television for Standard 4:3 and Wide-Screen 16:9 Aspect Ratios," Int. Telecommun. Union, Geneva, Switzerland, 1994.

[18] E. Gamma, R. Helm, R. Johnson, and J. Vlissides, *Design Patterns: Elements of Reusable Object-Oriented Software*. Reading, MA, USA: Addison-Wesley, 1994.

[19] "Go Language Specification – Type Parameters," [Online]. Available: https://go.dev/ref/spec#Type_parameters
