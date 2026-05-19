// Package pixelforge_render is the load-bearing parity seam between
// the studio preview, the shipped pixelforge-player binary, and the
// CI replay harness. It exposes a single tick+render function that
// every consumer ends up calling, so the pixels rendered by
// `pixelforge-studio` while authoring are byte-identical (modulo the
// boundaries documented below) to the pixels the shipped player
// produces and to the frames the deterministic replay test asserts
// against.
//
// # Entry points
//
// Two public entry points, one private shared core:
//
//   - RenderTickAtScreen(rt, screen, tick, inputs) — used by both the
//     player binary and the studio preview. Draws directly into the
//     Ebitengine *ebiten.Image the runtime owns; performs no GPU
//     readback. This is the per-frame path; cost is bounded by the
//     existing draw stack, no extra allocation.
//
//   - RenderTickAtRGBA(rt, tick, inputs) — used by CI replay tests and
//     headless pixel-hash assertions. Renders into an off-screen
//     *ebiten.Image, calls (*ebiten.Image).ReadPixels, returns a fresh
//     *image.RGBA. The readback is a measurable cost (a GPU→CPU
//     transfer) so RGBA mode is not used on hot paths — only by tests
//     and tooling that need to hash, save, or compare frames.
//
// Both delegate to a private advanceAndRender(rt, screen, tick,
// inputs) which contains the actual per-tick simulation + draw logic.
// The two public wrappers only differ in (a) which surface
// advanceAndRender writes into and (b) whether a ReadPixels readback +
// RGBA allocation happens after the draw. This is the literal "one
// render path" the arcade-shipping plan promises: implemented as one
// private function with two thin public wrappers.
//
// # Parity contract — honest version
//
// The plan asserts "structural impossibility of preview-vs-shipped
// drift" because of this seam. That claim needs to be qualified
// honestly so future contributors don't get sandbagged by surprise
// drift surfaces this seam does not, by itself, eliminate:
//
//  1. **Rendering-stage drift is eliminated.** Given the same
//     *capsuleruntime.Runtime, the same `tick`, and the same
//     InputFrame, advanceAndRender produces deterministically equal
//     pixels regardless of whether the caller is the player binary,
//     the studio preview, or the replay harness. This is the
//     guarantee this package makes, and it is enforced by the
//     determinism test (TestRenderTickAt_BitIdentical).
//
//  2. **Boot-stage drift is a separate concern.** The studio preview
//     and the shipped player do NOT necessarily Boot from byte-
//     identical inputs today. The studio Boots from an in-memory
//     *pixelforge_project.Project (potentially with unsaved edits);
//     the player Boots by reading the appended .pforge cart with
//     pixelforge_project.LoadReader. These two paths CAN diverge if
//     the in-memory Project has state the JSON loader does not
//     produce. Closing that gap is the job of U10 (snapshot save +
//     entity-component reflection) and later phases — not U4.
//     RenderTickAt is the contract for "given equal runtimes, render
//     equal pixels"; making the runtimes themselves equal is upstream
//     of this package.
//
//  3. **Editor overlays are explicitly excluded.** Gizmos, selection
//     handles, the chrome ribbon, the bounding boxes the studio draws
//     over the live preview — those are studio-side overlays drawn
//     AFTER advanceAndRender returns. They never enter the render
//     path. The contract is: the studio preview's RenderTickAt output
//     is byte-equal to the player's, and the studio then adds editor
//     chrome on top via a separate draw call. So a viewer comparing
//     the two visually will see different things (studio = game +
//     chrome, player = game only), but the underlying game pixels are
//     identical — which is the actually-load-bearing invariant for
//     replay-based correctness CI.
//
// # Determinism guarantees + boundaries
//
// Inside advanceAndRender:
//
//   - No time.Now() reads.
//   - No math/rand without a seeded source the runtime owns.
//   - No goroutines started by this function.
//   - No filesystem reads (all assets are already in the Runtime).
//   - Input is taken from the InputFrame argument, not from
//     ebiten.IsKeyPressed or similar live polling — so the same
//     InputFrame produces the same intent dispatch every time.
//
// What this package does NOT promise:
//
//   - **Cross-CPU determinism** (amd64 vs arm64). The plan defers the
//     empirical answer to U6 (the physics determinism probe). If
//     resolv's float64+SAT math diverges across architectures, the
//     pixel-hash CI strategy in U11/U14/U16/U18 will need to be
//     scoped to a single architecture or fall back to tolerance-
//     based comparison. RenderTickAt does not pre-promise something
//     U6 is responsible for measuring.
//
//   - **Audio sample-accurate output.** Audio sinks may run async via
//     the Ebitengine audio context; RenderTickAt only governs the
//     pixel output. Audio determinism is a separate seam.
//
//   - **GPU driver pixel rounding.** Ebitengine WritePixels round-
//     trips bytes 1:1 through the screen surface, but a driver bug
//     that mis-converts a palette entry would surface as drift this
//     package cannot guard against. The CI test pipeline hashes the
//     output of RenderTickAtRGBA, which goes through ReadPixels — so
//     if a driver mis-renders, the test fails loudly.
//
// # The InputFrame contract
//
// InputFrame is the explicit, reproducible representation of one
// tick's input — the slice of currently-held ebiten.Key values plus
// an optional GamepadState. Recorders capture it; replayers replay
// it; tests construct it inline. It deliberately does not carry
// mouse state, which is a non-deterministic input source (cursor
// position is sub-pixel float in some drivers) and is not part of
// the v1 replayable surface.
//
// Empty InputFrame{} (no keys, no pad) is valid: advanceAndRender
// must handle it without panic and still render a frame. This is
// tested by TestRenderTickAt_EmptyInput.
package pixelforge_render
