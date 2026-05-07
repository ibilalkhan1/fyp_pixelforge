# Chapter 3: Requirements and Design

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

### Chapter 3: Navigation Flow

```figure
![GUI Navigation Flow](diagrams/gui_nav.png)
**Figure 3.10:** GUI Navigation Flow. The game Draw callback branches based on the active example. The GUI example uses a panel with an attached button that fires an OnTap event when pressed. The Ctrl+Shift+I shortcut toggles the Piscope overlay, which provides pause, step, and screenshot controls.
```

## 3.10 Conclusions

Chapter 3 has presented the complete requirements and architectural design of the Pixelforge engine. The functional requirements (FR-1 through FR-20) establish a comprehensive surface area covering pixel rendering, shape primitives, sprite handling, color palette management, multi-channel audio, input subsystems, event-driven architecture, coroutines, font rendering, GUI elements, and integrated developer tools. The non-functional requirements (NFR-1 through NFR-7) codify the engine’s design philosophy: single-threaded performance, minimal allocations, modular composability, bounded problem space, cross-platform support, educational clarity, and test coverage.

The proposed methodology was an iterative, subsystem-by-subsystem approach that allowed each package to be designed and tested independently before integration—a direct application of the modular architectural principles found in the literature [1]. The system architecture was described through layered diagrams showing the application, engine core, and backend layers; sequence diagrams illustrating frame data flow and audio playback; and class diagrams capturing the relationships between core types.

Use cases covering minimal games, interactive shape drawing, snake gameplay, piano audio, and developer tool debugging demonstrated how the requirements translate into concrete user-facing functionality. The optional artifacts—database design (in-memory structures), class diagrams, sequence diagrams, and GUI screenshots—provide a complete specification sufficient for any developer to reproduce the system. The next chapter will detail the actual implementation of each subsystem, including the specific algorithms, code patterns, and testing strategies employed.

---

## Chapter 3: References for Chapter 3

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
